#include <curl/curl.h>
#include <openssl/hmac.h>

#include <chrono>
#include <iomanip>
#include <iostream>
#include <random>
#include <sstream>
#include <stdexcept>
#include <string>

// License SaaS C++ 客户端对接示例
// 依赖：libcurl + OpenSSL
// Ubuntu/Debian: sudo apt install build-essential libcurl4-openssl-dev libssl-dev
// 编译：g++ -std=c++17 client_demo.cpp -o client_demo -lcurl -lssl -lcrypto
// 运行：./client_demo http://127.0.0.1:8080 demo-app <client_secret>

struct HttpResponse {
    long status = 0;
    std::string body;
};

static size_t write_callback(char* ptr, size_t size, size_t nmemb, void* userdata) {
    auto* out = static_cast<std::string*>(userdata);
    out->append(ptr, size * nmemb);
    return size * nmemb;
}

std::string hex_encode(const unsigned char* data, unsigned int len) {
    std::ostringstream oss;
    oss << std::hex << std::setfill('0');
    for (unsigned int i = 0; i < len; ++i) {
        oss << std::setw(2) << static_cast<int>(data[i]);
    }
    return oss.str();
}

std::string hmac_sha256_hex(const std::string& secret, const std::string& message) {
    unsigned char out[EVP_MAX_MD_SIZE];
    unsigned int out_len = 0;
    HMAC(EVP_sha256(),
         reinterpret_cast<const unsigned char*>(secret.data()), static_cast<int>(secret.size()),
         reinterpret_cast<const unsigned char*>(message.data()), message.size(),
         out, &out_len);
    return hex_encode(out, out_len);
}

std::string unix_timestamp_seconds() {
    using namespace std::chrono;
    return std::to_string(duration_cast<seconds>(system_clock::now().time_since_epoch()).count());
}

std::string random_nonce(size_t len = 16) {
    static const char alphabet[] = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ";
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<size_t> dist(0, sizeof(alphabet) - 2);
    std::string s;
    s.reserve(len);
    for (size_t i = 0; i < len; ++i) s.push_back(alphabet[dist(gen)]);
    return s;
}

// 后端签名原文：METHOD + "\n" + PATH_WITH_QUERY + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + BODY_JSON
// 注意：path_with_query 只包含 /api/...，不要带 http://host:port。
std::string make_signature(const std::string& client_secret,
                           const std::string& method,
                           const std::string& path_with_query,
                           const std::string& timestamp,
                           const std::string& nonce,
                           const std::string& body_json) {
    std::string canonical = method + "\n" + path_with_query + "\n" + timestamp + "\n" + nonce + "\n" + body_json;
    return hmac_sha256_hex(client_secret, canonical);
}

HttpResponse request_signed(const std::string& base_url,
                            const std::string& app_key,
                            const std::string& client_secret,
                            const std::string& method,
                            const std::string& path_with_query,
                            const std::string& body_json = "",
                            const std::string& client_token = "") {
    CURL* curl = curl_easy_init();
    if (!curl) throw std::runtime_error("curl_easy_init failed");

    std::string timestamp = unix_timestamp_seconds();
    std::string nonce = random_nonce();
    std::string signature = make_signature(client_secret, method, path_with_query, timestamp, nonce, body_json);

    std::string url = base_url + path_with_query;
    std::string response_body;
    struct curl_slist* headers = nullptr;
    headers = curl_slist_append(headers, ("X-App-Key: " + app_key).c_str());
    headers = curl_slist_append(headers, ("X-Timestamp: " + timestamp).c_str());
    headers = curl_slist_append(headers, ("X-Nonce: " + nonce).c_str());
    headers = curl_slist_append(headers, ("X-Signature: " + signature).c_str());
    if (!client_token.empty()) {
        headers = curl_slist_append(headers, (std::string("Authorization: ") + "Bearer " + client_token).c_str());
    }
    if (method == "POST") {
        headers = curl_slist_append(headers, "Content-Type: application/json");
    }

    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, write_callback);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response_body);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);

    if (method == "POST") {
        curl_easy_setopt(curl, CURLOPT_POST, 1L);
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body_json.c_str());
        curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, static_cast<long>(body_json.size()));
    }

    CURLcode rc = curl_easy_perform(curl);
    long status = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &status);

    curl_slist_free_all(headers);
    curl_easy_cleanup(curl);

    if (rc != CURLE_OK) {
        throw std::runtime_error(std::string("curl error: ") + curl_easy_strerror(rc));
    }
    return {status, response_body};
}

void print_response(const std::string& title, const HttpResponse& res) {
    std::cout << "\n==== " << title << " ====\n";
    std::cout << "HTTP " << res.status << "\n";
    std::cout << res.body << "\n";
}

int main(int argc, char** argv) {
    std::string base_url = argc > 1 ? argv[1] : "http://127.0.0.1:8080";
    std::string app_key = argc > 2 ? argv[2] : "demo-app";
    std::string client_secret = argc > 3 ? argv[3] : "929df12601c3cdbf423c145c90f7e351a134f77c1761fef4db241021333e5066";

    // 固定测试向量：用于确认 HMAC 实现是否和后端一致。
    {
        std::string canonical = "GET\n/api/client/app-info?app_key=demo-app\n1779979787\notsbrssveuywakt\n";
        std::string sig = hmac_sha256_hex(client_secret, canonical);
        std::cout << "HMAC test signature: " << sig << "\n";
        std::cout << "Expected if using demo secret above: 0f489948f6fadd77f63b361dab61fc346b2523ade353e20834cd2775de3e7b9a\n";
    }

    curl_global_init(CURL_GLOBAL_DEFAULT);
    try {
        // 1) 获取软件运行配置。GET 没有 body，签名原文最后一行 body 为空。
        std::string app_info_path = "/api/client/app-info?app_key=" + app_key;
        auto app_info = request_signed(base_url, app_key, client_secret, "GET", app_info_path);
        print_response("GET app-info", app_info);

        // 2) 账号登录示例。需要后台已有账号且未过期；body_json 必须和实际发送内容完全一致。
        std::string login_body = std::string("{\"appKey\":\"") + app_key +
            "\",\"username\":\"u1\",\"password\":\"123456\",\"machineCode\":\"PC-001\"}";
        auto login = request_signed(base_url, app_key, client_secret, "POST", "/api/client/login", login_body);
        print_response("POST login example", login);

        // 3) 卡密登录示例。首次登录会绑定 machineCode；这里把卡密替换成真实未使用卡密再测试。
        std::string card_login_body = std::string("{\"appKey\":\"") + app_key +
            "\",\"cardKey\":\"LS-XXXX-XXXX-XXXX\",\"machineCode\":\"PC-001\"}";
        auto card_login = request_signed(base_url, app_key, client_secret, "POST", "/api/client/card-login", card_login_body);
        print_response("POST card-login example", card_login);

        // 4) 心跳示例：需要把登录返回 JSON 里的 client_token 取出来传入。
        // std::string client_token = "登录返回的 client_token";
        // std::string heartbeat_body = "{\"machineCode\":\"PC-001\",\"clientVersion\":\"1.0.0\"}";
        // auto heartbeat = request_signed(base_url, app_key, client_secret, "POST", "/api/client/heartbeat", heartbeat_body, client_token);
        // print_response("POST heartbeat", heartbeat);
    } catch (const std::exception& e) {
        std::cerr << "ERROR: " << e.what() << "\n";
        curl_global_cleanup();
        return 1;
    }
    curl_global_cleanup();
    return 0;
}
