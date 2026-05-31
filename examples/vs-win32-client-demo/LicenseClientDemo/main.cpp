#include <windows.h>
#include <winhttp.h>
#include <bcrypt.h>

#include <ctime>
#include <iomanip>
#include <iostream>
#include <random>
#include <sstream>
#include <stdexcept>
#include <string>
#include <vector>

#pragma comment(lib, "winhttp.lib")
#pragma comment(lib, "bcrypt.lib")

// License SaaS Visual Studio Win32 C++ demo.
// Account mode: register, login, recharge, heartbeat, cloud variables, unbind.
// Card mode: card login, heartbeat, cloud variables, card unbind.
// Dependencies: Windows SDK only. Uses WinHTTP and BCrypt. No OpenSSL/libcurl required.

static const std::wstring kBaseUrl = L"http://127.0.0.1:8080";
static const std::string kAppKey = "demo-app";
static const std::string kClientSecret = "REPLACE_WITH_CLIENT_SECRET";

static const std::string kAccountCardKey = "REPLACE_WITH_ACCOUNT_REGISTER_CARD";
static const std::string kRechargeCardKey = "REPLACE_WITH_RECHARGE_CARD";
static const std::string kStandaloneCardKey = "REPLACE_WITH_STANDALONE_CARD";

static const std::string kUsername = "testuser_cpp";
static const std::string kPassword = "123456";
static const std::string kMachineCode = "WIN32-DEMO-MACHINE-001";
static const std::string kClientVersion = "1.0.0";

struct HttpResult {
    DWORD status;
    std::string body;
};

static std::wstring Utf8ToWide(const std::string& s) {
    if (s.empty()) return L"";
    int len = MultiByteToWideChar(CP_UTF8, 0, s.c_str(), (int)s.size(), NULL, 0);
    std::wstring out((size_t)len, L'\0');
    MultiByteToWideChar(CP_UTF8, 0, s.c_str(), (int)s.size(), &out[0], len);
    return out;
}

static std::string JsonEscape(const std::string& s) {
    std::ostringstream o;
    for (size_t i = 0; i < s.size(); ++i) {
        unsigned char c = (unsigned char)s[i];
        switch (c) {
        case '"': o << "\\\""; break;
        case '\\': o << "\\\\"; break;
        case '\b': o << "\\b"; break;
        case '\f': o << "\\f"; break;
        case '\n': o << "\\n"; break;
        case '\r': o << "\\r"; break;
        case '\t': o << "\\t"; break;
        default:
            if (c < 0x20) {
                o << "\\u" << std::hex << std::setw(4) << std::setfill('0') << (int)c;
            } else {
                o << (char)c;
            }
        }
    }
    return o.str();
}

static std::string JsonObject(const std::vector<std::pair<std::string, std::string> >& fields) {
    std::ostringstream o;
    o << "{";
    for (size_t i = 0; i < fields.size(); ++i) {
        if (i) o << ",";
        o << "\"" << JsonEscape(fields[i].first) << "\":\"" << JsonEscape(fields[i].second) << "\"";
    }
    o << "}";
    return o.str();
}

static std::string UnixSeconds() {
    return std::to_string((long long)std::time(NULL));
}

static std::string RandomNonce(size_t n = 16) {
    static const char* chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<int> dis(0, 61);
    std::string s;
    for (size_t i = 0; i < n; ++i) s.push_back(chars[dis(gen)]);
    return s;
}

static std::string Hex(const unsigned char* data, ULONG len) {
    std::ostringstream oss;
    oss << std::hex << std::setfill('0');
    for (ULONG i = 0; i < len; ++i) oss << std::setw(2) << (int)data[i];
    return oss.str();
}

static std::string HmacSha256Hex(const std::string& key, const std::string& message) {
    BCRYPT_ALG_HANDLE hAlg = NULL;
    NTSTATUS st = BCryptOpenAlgorithmProvider(&hAlg, BCRYPT_SHA256_ALGORITHM, NULL, BCRYPT_ALG_HANDLE_HMAC_FLAG);
    if (st < 0) throw std::runtime_error("BCryptOpenAlgorithmProvider failed");

    DWORD cbHash = 0;
    DWORD cbData = 0;
    BCryptGetProperty(hAlg, BCRYPT_HASH_LENGTH, (PUCHAR)&cbHash, sizeof(DWORD), &cbData, 0);
    std::vector<unsigned char> hash(cbHash);

    BCRYPT_HASH_HANDLE hHash = NULL;
    st = BCryptCreateHash(hAlg, &hHash, NULL, 0, (PUCHAR)key.data(), (ULONG)key.size(), 0);
    if (st < 0) {
        BCryptCloseAlgorithmProvider(hAlg, 0);
        throw std::runtime_error("BCryptCreateHash failed");
    }

    st = BCryptHashData(hHash, (PUCHAR)message.data(), (ULONG)message.size(), 0);
    if (st >= 0) st = BCryptFinishHash(hHash, &hash[0], cbHash, 0);

    BCryptDestroyHash(hHash);
    BCryptCloseAlgorithmProvider(hAlg, 0);

    if (st < 0) throw std::runtime_error("BCrypt HMAC failed");
    return Hex(&hash[0], cbHash);
}

static std::string MakeSignature(const std::string& method,
                                 const std::string& pathWithQuery,
                                 const std::string& timestamp,
                                 const std::string& nonce,
                                 const std::string& bodyJson) {
    std::string canonical = method + "\n" + pathWithQuery + "\n" + timestamp + "\n" + nonce + "\n" + bodyJson;
    return HmacSha256Hex(kClientSecret, canonical);
}

static std::string ExtractJsonString(const std::string& json, const std::string& key) {
    std::string pat = "\"" + key + "\"";
    size_t p = json.find(pat);
    if (p == std::string::npos) return "";
    p = json.find(':', p);
    if (p == std::string::npos) return "";
    p = json.find('"', p);
    if (p == std::string::npos) return "";
    ++p;
    std::string out;
    bool esc = false;
    for (; p < json.size(); ++p) {
        char c = json[p];
        if (esc) {
            switch (c) {
            case 'n': out.push_back('\n'); break;
            case 'r': out.push_back('\r'); break;
            case 't': out.push_back('\t'); break;
            case '"': out.push_back('"'); break;
            case '\\': out.push_back('\\'); break;
            default: out.push_back(c); break;
            }
            esc = false;
        } else if (c == '\\') {
            esc = true;
        } else if (c == '"') {
            break;
        } else {
            out.push_back(c);
        }
    }
    return out;
}

static HttpResult HttpRequest(const std::string& method,
                              const std::string& pathWithQuery,
                              const std::string& bodyJson,
                              const std::string& bearerToken) {
    std::wstring fullUrl = kBaseUrl + Utf8ToWide(pathWithQuery);

    URL_COMPONENTS uc;
    ZeroMemory(&uc, sizeof(uc));
    uc.dwStructSize = sizeof(uc);
    wchar_t host[256];
    ZeroMemory(host, sizeof(host));
    uc.lpszHostName = host;
    uc.dwHostNameLength = _countof(host);

    if (!WinHttpCrackUrl(fullUrl.c_str(), 0, 0, &uc)) throw std::runtime_error("WinHttpCrackUrl failed");

    std::wstring hostName(host, uc.dwHostNameLength);
    HINTERNET hSession = WinHttpOpen(L"LicenseSaaS-Win32-Demo/1.0", WINHTTP_ACCESS_TYPE_DEFAULT_PROXY,
                                     WINHTTP_NO_PROXY_NAME, WINHTTP_NO_PROXY_BYPASS, 0);
    if (!hSession) throw std::runtime_error("WinHttpOpen failed");

    HINTERNET hConnect = WinHttpConnect(hSession, hostName.c_str(), uc.nPort, 0);
    if (!hConnect) {
        WinHttpCloseHandle(hSession);
        throw std::runtime_error("WinHttpConnect failed");
    }

    DWORD flags = (uc.nScheme == INTERNET_SCHEME_HTTPS) ? WINHTTP_FLAG_SECURE : 0;
    std::wstring objectName = Utf8ToWide(pathWithQuery);
    HINTERNET hRequest = WinHttpOpenRequest(hConnect, Utf8ToWide(method).c_str(), objectName.c_str(),
                                            NULL, WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES, flags);
    if (!hRequest) {
        WinHttpCloseHandle(hConnect);
        WinHttpCloseHandle(hSession);
        throw std::runtime_error("WinHttpOpenRequest failed");
    }

    std::string ts = UnixSeconds();
    std::string nonce = RandomNonce();
    std::string sig = MakeSignature(method, pathWithQuery, ts, nonce, bodyJson);

    std::wstring headers;
    headers += L"Content-Type: application/json\r\n";
    headers += Utf8ToWide("X-App-Key: " + kAppKey + "\r\n");
    headers += Utf8ToWide("X-Timestamp: " + ts + "\r\n");
    headers += Utf8ToWide("X-Nonce: " + nonce + "\r\n");
    headers += Utf8ToWide("X-Signature: " + sig + "\r\n");
    if (!bearerToken.empty()) {
        headers += Utf8ToWide(std::string("Authorization: ") + "Bearer " + bearerToken + "\r\n");
    }

    LPVOID bodyPtr = bodyJson.empty() ? WINHTTP_NO_REQUEST_DATA : (LPVOID)bodyJson.data();
    DWORD bodyLen = (DWORD)bodyJson.size();
    BOOL ok = WinHttpSendRequest(hRequest, headers.c_str(), (DWORD)-1L, bodyPtr, bodyLen, bodyLen, 0);
    if (ok) ok = WinHttpReceiveResponse(hRequest, NULL);
    if (!ok) {
        WinHttpCloseHandle(hRequest);
        WinHttpCloseHandle(hConnect);
        WinHttpCloseHandle(hSession);
        throw std::runtime_error("WinHTTP send/receive failed");
    }

    DWORD status = 0;
    DWORD statusSize = sizeof(status);
    WinHttpQueryHeaders(hRequest, WINHTTP_QUERY_STATUS_CODE | WINHTTP_QUERY_FLAG_NUMBER,
                        WINHTTP_HEADER_NAME_BY_INDEX, &status, &statusSize, WINHTTP_NO_HEADER_INDEX);

    std::string response;
    for (;;) {
        DWORD available = 0;
        if (!WinHttpQueryDataAvailable(hRequest, &available) || available == 0) break;
        std::vector<char> buf(available + 1, 0);
        DWORD read = 0;
        if (!WinHttpReadData(hRequest, &buf[0], available, &read) || read == 0) break;
        response.append(&buf[0], read);
    }

    WinHttpCloseHandle(hRequest);
    WinHttpCloseHandle(hConnect);
    WinHttpCloseHandle(hSession);

    HttpResult result;
    result.status = status;
    result.body = response;
    return result;
}

static HttpResult SignedGet(const std::string& pathWithQuery, const std::string& token = "") {
    return HttpRequest("GET", pathWithQuery, "", token);
}

static HttpResult SignedPost(const std::string& path, const std::string& bodyJson, const std::string& token = "") {
    return HttpRequest("POST", path, bodyJson, token);
}

static void PrintResult(const std::string& title, const HttpResult& r) {
    std::cout << "\n==== " << title << " ====\n";
    std::cout << "HTTP " << r.status << "\n" << r.body << "\n";
}

static void CheckSignatureVector() {
    std::string secret = "929df12601c3cdbf423c145c90f7e351a134f77c1761fef4db241021333e5066";
    std::string canonical = "GET\n/api/client/app-info?app_key=demo-app\n1779979787\notsbrssveuywakt\n";
    std::string got = HmacSha256Hex(secret, canonical);
    std::string want = "0f489948f6fadd77f63b361dab61fc346b2523ade353e20834cd2775de3e7b9a";
    std::cout << "signature test: " << (got == want ? "OK" : "FAIL") << "\n";
    if (got != want) std::cout << "got=" << got << "\nwant=" << want << "\n";
}

static void AccountModeDemo() {
    std::cout << "\n######## Account mode demo ########\n";

    HttpResult appInfo = SignedGet("/api/client/app-info?app_key=" + kAppKey);
    PrintResult("app info", appInfo);

    std::string regBody = JsonObject({
        {"appKey", kAppKey},
        {"username", kUsername},
        {"password", kPassword},
        {"cardKey", kAccountCardKey},
        {"machineCode", kMachineCode}
    });
    HttpResult reg = SignedPost("/api/client/register", regBody);
    PrintResult("account register", reg);

    std::string token = ExtractJsonString(reg.body, "client_token");
    if (token.empty()) {
        std::string loginBody = JsonObject({
            {"appKey", kAppKey},
            {"username", kUsername},
            {"password", kPassword},
            {"machineCode", kMachineCode}
        });
        HttpResult login = SignedPost("/api/client/login", loginBody);
        PrintResult("account login", login);
        token = ExtractJsonString(login.body, "client_token");
    }

    if (!token.empty()) {
        HttpResult cloud = SignedGet("/api/client/cloud-vars?app_key=" + kAppKey, token);
        PrintResult("account cloud vars", cloud);

        std::string hbBody = JsonObject({
            {"machineCode", kMachineCode},
            {"clientVersion", kClientVersion}
        });
        HttpResult hb = SignedPost("/api/client/heartbeat", hbBody, token);
        PrintResult("account heartbeat", hb);

        if (kRechargeCardKey.find("REPLACE_WITH_") == std::string::npos) {
            std::string rechargeBody = JsonObject({
                {"appKey", kAppKey},
                {"username", kUsername},
                {"cardKey", kRechargeCardKey}
            });
            HttpResult recharge = SignedPost("/api/client/recharge", rechargeBody);
            PrintResult("account recharge", recharge);
        }

        std::string unbindBody = JsonObject({
            {"appKey", kAppKey},
            {"machineCode", kMachineCode}
        });
        HttpResult unbind = SignedPost("/api/client/unbind", unbindBody, token);
        PrintResult("account token unbind", unbind);
    }

    std::string accountUnbindBody = JsonObject({
        {"appKey", kAppKey},
        {"username", kUsername},
        {"password", kPassword}
    });
    HttpResult accountUnbind = SignedPost("/api/client/account-unbind", accountUnbindBody);
    PrintResult("account password unbind", accountUnbind);
}

static void CardModeDemo() {
    std::cout << "\n######## Card mode demo ########\n";

    std::string body = JsonObject({
        {"appKey", kAppKey},
        {"cardKey", kStandaloneCardKey},
        {"machineCode", kMachineCode}
    });
    HttpResult login = SignedPost("/api/client/card-login", body);
    PrintResult("card login", login);

    std::string token = ExtractJsonString(login.body, "client_token");
    if (!token.empty()) {
        HttpResult cloud = SignedGet("/api/client/cloud-vars?app_key=" + kAppKey, token);
        PrintResult("card cloud vars", cloud);

        std::string hbBody = JsonObject({
            {"machineCode", kMachineCode},
            {"clientVersion", kClientVersion}
        });
        HttpResult hb = SignedPost("/api/client/heartbeat", hbBody, token);
        PrintResult("card heartbeat", hb);
    }

    std::string unbindBody = JsonObject({
        {"appKey", kAppKey},
        {"cardKey", kStandaloneCardKey}
    });
    HttpResult unbind = SignedPost("/api/client/card-unbind", unbindBody);
    PrintResult("card unbind", unbind);
}

int main() {
    SetConsoleOutputCP(CP_UTF8);
    try {
        CheckSignatureVector();
        if (kClientSecret.find("REPLACE_WITH_") != std::string::npos) {
            std::cout << "Please edit main.cpp first: set kAppKey, kClientSecret and card keys.\n";
            system("pause");
            return 0;
        }
        AccountModeDemo();
        CardModeDemo();
    } catch (const std::exception& e) {
        std::cerr << "Exception: " << e.what() << "\n";
    }
    system("pause");
    return 0;
}
