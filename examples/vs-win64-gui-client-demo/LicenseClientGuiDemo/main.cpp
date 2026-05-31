#include <windows.h>
#include <winhttp.h>
#include <bcrypt.h>
#include <commctrl.h>

#include <ctime>
#include <iomanip>
#include <map>
#include <random>
#include <sstream>
#include <stdexcept>
#include <string>
#include <vector>

#pragma comment(lib, "winhttp.lib")
#pragma comment(lib, "bcrypt.lib")
#pragma comment(lib, "comctl32.lib")

#define IDC_BASE_URL 1001
#define IDC_APP_KEY 1002
#define IDC_CLIENT_SECRET 1003
#define IDC_USERNAME 1004
#define IDC_PASSWORD 1005
#define IDC_MACHINE 1006
#define IDC_CLIENT_VERSION 1007
#define IDC_ACCOUNT_CARD 1008
#define IDC_RECHARGE_CARD 1009
#define IDC_STANDALONE_CARD 1010
#define IDC_ACCOUNT_TOKEN 1011
#define IDC_CARD_TOKEN 1012
#define IDC_LOG 1013
#define IDC_APPINFO 1101
#define IDC_REGISTER 1102
#define IDC_LOGIN 1103
#define IDC_RECHARGE 1104
#define IDC_ACCOUNT_HEARTBEAT 1105
#define IDC_ACCOUNT_CLOUD 1106
#define IDC_TOKEN_UNBIND 1107
#define IDC_ACCOUNT_UNBIND 1108
#define IDC_CARD_LOGIN 1109
#define IDC_CARD_HEARTBEAT 1110
#define IDC_CARD_CLOUD 1111
#define IDC_CARD_UNBIND 1112
#define IDC_CLEAR_LOG 1113

static HWND g_hWnd = NULL;
static HFONT g_font = NULL;
static std::map<int, HWND> g_controls;

static std::wstring Utf8ToWide(const std::string& s) {
    if (s.empty()) return L"";
    int len = MultiByteToWideChar(CP_UTF8, 0, s.c_str(), (int)s.size(), NULL, 0);
    std::wstring out(len, 0);
    MultiByteToWideChar(CP_UTF8, 0, s.c_str(), (int)s.size(), &out[0], len);
    return out;
}

static std::string WideToUtf8(const std::wstring& s) {
    if (s.empty()) return "";
    int len = WideCharToMultiByte(CP_UTF8, 0, s.c_str(), (int)s.size(), NULL, 0, NULL, NULL);
    std::string out(len, 0);
    WideCharToMultiByte(CP_UTF8, 0, s.c_str(), (int)s.size(), &out[0], len, NULL, NULL);
    return out;
}

static std::wstring GetTextW(int id) {
    HWND h = g_controls[id];
    int len = GetWindowTextLengthW(h);
    std::wstring s(len, L'\0');
    GetWindowTextW(h, &s[0], len + 1);
    return s;
}

static std::string GetText(int id) { return WideToUtf8(GetTextW(id)); }

static void SetText(int id, const std::string& text) {
    SetWindowTextW(g_controls[id], Utf8ToWide(text).c_str());
}

static void AppendLog(const std::string& text) {
    HWND log = g_controls[IDC_LOG];
    std::wstring w = Utf8ToWide(text + "\r\n");
    int len = GetWindowTextLengthW(log);
    SendMessageW(log, EM_SETSEL, len, len);
    SendMessageW(log, EM_REPLACESEL, FALSE, (LPARAM)w.c_str());
}

static std::string BytesToHex(const unsigned char* data, DWORD len) {
    static const char* hex = "0123456789abcdef";
    std::string out;
    out.reserve(len * 2);
    for (DWORD i = 0; i < len; ++i) {
        out.push_back(hex[(data[i] >> 4) & 0xF]);
        out.push_back(hex[data[i] & 0xF]);
    }
    return out;
}

static std::string HmacSha256Hex(const std::string& key, const std::string& data) {
    BCRYPT_ALG_HANDLE hAlg = NULL;
    BCRYPT_HASH_HANDLE hHash = NULL;
    DWORD cbData = 0, cbHash = 0, cbHashObject = 0;
    std::vector<unsigned char> hashObject;
    std::vector<unsigned char> hash;
    NTSTATUS st = BCryptOpenAlgorithmProvider(&hAlg, BCRYPT_SHA256_ALGORITHM, NULL, BCRYPT_ALG_HANDLE_HMAC_FLAG);
    if (st < 0) throw std::runtime_error("BCryptOpenAlgorithmProvider failed");
    st = BCryptGetProperty(hAlg, BCRYPT_OBJECT_LENGTH, (PUCHAR)&cbHashObject, sizeof(DWORD), &cbData, 0);
    if (st < 0) { BCryptCloseAlgorithmProvider(hAlg, 0); throw std::runtime_error("BCryptGetProperty object failed"); }
    st = BCryptGetProperty(hAlg, BCRYPT_HASH_LENGTH, (PUCHAR)&cbHash, sizeof(DWORD), &cbData, 0);
    if (st < 0) { BCryptCloseAlgorithmProvider(hAlg, 0); throw std::runtime_error("BCryptGetProperty hash failed"); }
    hashObject.resize(cbHashObject);
    hash.resize(cbHash);
    st = BCryptCreateHash(hAlg, &hHash, hashObject.data(), cbHashObject, (PUCHAR)key.data(), (ULONG)key.size(), 0);
    if (st < 0) { BCryptCloseAlgorithmProvider(hAlg, 0); throw std::runtime_error("BCryptCreateHash failed"); }
    st = BCryptHashData(hHash, (PUCHAR)data.data(), (ULONG)data.size(), 0);
    if (st < 0) { BCryptDestroyHash(hHash); BCryptCloseAlgorithmProvider(hAlg, 0); throw std::runtime_error("BCryptHashData failed"); }
    st = BCryptFinishHash(hHash, hash.data(), cbHash, 0);
    BCryptDestroyHash(hHash);
    BCryptCloseAlgorithmProvider(hAlg, 0);
    if (st < 0) throw std::runtime_error("BCryptFinishHash failed");
    return BytesToHex(hash.data(), cbHash);
}

static std::string JsonEscape(const std::string& s) {
    std::string o;
    for (char c : s) {
        switch (c) {
        case '\\': o += "\\\\"; break;
        case '"': o += "\\\""; break;
        case '\n': o += "\\n"; break;
        case '\r': o += "\\r"; break;
        case '\t': o += "\\t"; break;
        default: o += c; break;
        }
    }
    return o;
}

static std::string JsonObject(const std::vector<std::pair<std::string, std::string>>& fields) {
    std::string s = "{";
    for (size_t i = 0; i < fields.size(); ++i) {
        if (i) s += ",";
        s += "\"" + fields[i].first + "\":\"" + JsonEscape(fields[i].second) + "\"";
    }
    s += "}";
    return s;
}

static std::string RandomNonce() {
    static const char chars[] = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<int> dist(0, (int)sizeof(chars) - 2);
    std::string s;
    for (int i = 0; i < 20; ++i) s.push_back(chars[dist(gen)]);
    return s;
}

struct HttpResult { DWORD status = 0; std::string body; };

static void CrackUrl(const std::wstring& url, std::wstring& host, INTERNET_PORT& port, bool& https) {
    URL_COMPONENTS uc{};
    wchar_t hostBuf[256]{};
    uc.dwStructSize = sizeof(uc);
    uc.lpszHostName = hostBuf;
    uc.dwHostNameLength = 255;
    if (!WinHttpCrackUrl(url.c_str(), 0, 0, &uc)) throw std::runtime_error("WinHttpCrackUrl failed");
    host.assign(uc.lpszHostName, uc.dwHostNameLength);
    port = uc.nPort;
    https = uc.nScheme == INTERNET_SCHEME_HTTPS;
}

static HttpResult HttpRequest(const std::string& method, const std::string& pathWithQuery, const std::string& body, const std::string& bearerToken, const std::map<std::string, std::string>& signedHeaders) {
    std::wstring base = GetTextW(IDC_BASE_URL);
    std::wstring host;
    INTERNET_PORT port = 0;
    bool https = false;
    CrackUrl(base, host, port, https);

    HINTERNET hSession = WinHttpOpen(L"LicenseClientGuiDemo/1.0", WINHTTP_ACCESS_TYPE_DEFAULT_PROXY, WINHTTP_NO_PROXY_NAME, WINHTTP_NO_PROXY_BYPASS, 0);
    if (!hSession) throw std::runtime_error("WinHttpOpen failed");
    HINTERNET hConnect = WinHttpConnect(hSession, host.c_str(), port, 0);
    if (!hConnect) { WinHttpCloseHandle(hSession); throw std::runtime_error("WinHttpConnect failed"); }
    DWORD flags = https ? WINHTTP_FLAG_SECURE : 0;
    HINTERNET hReq = WinHttpOpenRequest(hConnect, Utf8ToWide(method).c_str(), Utf8ToWide(pathWithQuery).c_str(), NULL, WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES, flags);
    if (!hReq) { WinHttpCloseHandle(hConnect); WinHttpCloseHandle(hSession); throw std::runtime_error("WinHttpOpenRequest failed"); }

    std::wstring headers = L"Content-Type: application/json\r\n";
    for (auto& kv : signedHeaders) headers += Utf8ToWide(kv.first + ": " + kv.second + "\r\n");
    if (!bearerToken.empty()) headers += Utf8ToWide(std::string("Authorization: ") + "Bearer " + bearerToken + "\r\n");

    BOOL ok = WinHttpSendRequest(hReq, headers.c_str(), (DWORD)-1L, body.empty() ? WINHTTP_NO_REQUEST_DATA : (LPVOID)body.data(), (DWORD)body.size(), (DWORD)body.size(), 0);
    if (!ok) { WinHttpCloseHandle(hReq); WinHttpCloseHandle(hConnect); WinHttpCloseHandle(hSession); throw std::runtime_error("WinHttpSendRequest failed"); }
    ok = WinHttpReceiveResponse(hReq, NULL);
    if (!ok) { WinHttpCloseHandle(hReq); WinHttpCloseHandle(hConnect); WinHttpCloseHandle(hSession); throw std::runtime_error("WinHttpReceiveResponse failed"); }

    DWORD status = 0, size = sizeof(status);
    WinHttpQueryHeaders(hReq, WINHTTP_QUERY_STATUS_CODE | WINHTTP_QUERY_FLAG_NUMBER, NULL, &status, &size, NULL);
    std::string resp;
    DWORD avail = 0;
    while (WinHttpQueryDataAvailable(hReq, &avail) && avail > 0) {
        std::vector<char> buf(avail + 1);
        DWORD read = 0;
        if (!WinHttpReadData(hReq, buf.data(), avail, &read)) break;
        resp.append(buf.data(), read);
    }
    WinHttpCloseHandle(hReq);
    WinHttpCloseHandle(hConnect);
    WinHttpCloseHandle(hSession);
    return { status, resp };
}

static HttpResult SignedRequest(const std::string& method, const std::string& path, const std::string& body = "", const std::string& token = "") {
    std::string appKey = GetText(IDC_APP_KEY);
    std::string secret = GetText(IDC_CLIENT_SECRET);
    if (appKey.empty() || secret.empty()) throw std::runtime_error("AppKey/client_secret is empty");
    std::string ts = std::to_string((long long)time(NULL));
    std::string nonce = RandomNonce();
    std::string canonical = method + "\n" + path + "\n" + ts + "\n" + nonce + "\n" + body;
    std::string sig = HmacSha256Hex(secret, canonical);
    std::map<std::string, std::string> headers;
    headers["X-App-Key"] = appKey;
    headers["X-Timestamp"] = ts;
    headers["X-Nonce"] = nonce;
    headers["X-Signature"] = sig;
    return HttpRequest(method, path, body, token, headers);
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
        if (esc) { out.push_back(c); esc = false; continue; }
        if (c == '\\') { esc = true; continue; }
        if (c == '"') break;
        out.push_back(c);
    }
    return out;
}

static void ShowResult(const std::string& title, const HttpResult& r) {
    AppendLog("---- " + title + " ----");
    AppendLog("HTTP " + std::to_string(r.status));
    AppendLog(r.body);
}

static void DoAction(int id) {
    try {
        std::string appKey = GetText(IDC_APP_KEY);
        std::string username = GetText(IDC_USERNAME);
        std::string password = GetText(IDC_PASSWORD);
        std::string machine = GetText(IDC_MACHINE);
        std::string clientVersion = GetText(IDC_CLIENT_VERSION);
        std::string accountCard = GetText(IDC_ACCOUNT_CARD);
        std::string rechargeCard = GetText(IDC_RECHARGE_CARD);
        std::string standaloneCard = GetText(IDC_STANDALONE_CARD);
        std::string accountToken = GetText(IDC_ACCOUNT_TOKEN);
        std::string cardToken = GetText(IDC_CARD_TOKEN);
        HttpResult r;
        switch (id) {
        case IDC_APPINFO:
            r = SignedRequest("GET", "/api/client/app-info?app_key=" + appKey);
            ShowResult("app-info", r);
            break;
        case IDC_REGISTER:
            r = SignedRequest("POST", "/api/client/register", JsonObject({ {"appKey", appKey}, {"username", username}, {"password", password}, {"cardKey", accountCard}, {"machineCode", machine} }));
            ShowResult("account register", r);
            SetText(IDC_ACCOUNT_TOKEN, ExtractJsonString(r.body, "client_token"));
            break;
        case IDC_LOGIN:
            r = SignedRequest("POST", "/api/client/login", JsonObject({ {"appKey", appKey}, {"username", username}, {"password", password}, {"machineCode", machine} }));
            ShowResult("account login", r);
            SetText(IDC_ACCOUNT_TOKEN, ExtractJsonString(r.body, "client_token"));
            break;
        case IDC_RECHARGE:
            r = SignedRequest("POST", "/api/client/recharge", JsonObject({ {"appKey", appKey}, {"username", username}, {"cardKey", rechargeCard} }));
            ShowResult("account recharge", r);
            break;
        case IDC_ACCOUNT_HEARTBEAT:
            r = SignedRequest("POST", "/api/client/heartbeat", JsonObject({ {"machineCode", machine}, {"clientVersion", clientVersion} }), accountToken);
            ShowResult("account heartbeat", r);
            break;
        case IDC_ACCOUNT_CLOUD:
            r = SignedRequest("GET", "/api/client/cloud-vars?app_key=" + appKey, "", accountToken);
            ShowResult("account cloud-vars", r);
            break;
        case IDC_TOKEN_UNBIND:
            r = SignedRequest("POST", "/api/client/unbind", JsonObject({ {"appKey", appKey}, {"machineCode", machine} }), accountToken);
            ShowResult("account token unbind", r);
            break;
        case IDC_ACCOUNT_UNBIND:
            r = SignedRequest("POST", "/api/client/account-unbind", JsonObject({ {"appKey", appKey}, {"username", username}, {"password", password} }));
            ShowResult("account password unbind", r);
            break;
        case IDC_CARD_LOGIN:
            r = SignedRequest("POST", "/api/client/card-login", JsonObject({ {"appKey", appKey}, {"cardKey", standaloneCard}, {"machineCode", machine} }));
            ShowResult("card login", r);
            SetText(IDC_CARD_TOKEN, ExtractJsonString(r.body, "client_token"));
            break;
        case IDC_CARD_HEARTBEAT:
            r = SignedRequest("POST", "/api/client/heartbeat", JsonObject({ {"machineCode", machine}, {"clientVersion", clientVersion} }), cardToken);
            ShowResult("card heartbeat", r);
            break;
        case IDC_CARD_CLOUD:
            r = SignedRequest("GET", "/api/client/cloud-vars?app_key=" + appKey, "", cardToken);
            ShowResult("card cloud-vars", r);
            break;
        case IDC_CARD_UNBIND:
            r = SignedRequest("POST", "/api/client/card-unbind", JsonObject({ {"appKey", appKey}, {"cardKey", standaloneCard} }));
            ShowResult("card unbind", r);
            break;
        }
    }
    catch (const std::exception& e) {
        AppendLog(std::string("ERROR: ") + e.what());
    }
}

static HWND AddLabel(HWND parent, int x, int y, int w, const wchar_t* text) {
    HWND h = CreateWindowW(L"STATIC", text, WS_CHILD | WS_VISIBLE, x, y + 4, w, 22, parent, NULL, NULL, NULL);
    SendMessageW(h, WM_SETFONT, (WPARAM)g_font, TRUE);
    return h;
}

static HWND AddEdit(HWND parent, int id, int x, int y, int w, const wchar_t* text, bool password = false) {
    DWORD style = WS_CHILD | WS_VISIBLE | WS_BORDER | ES_AUTOHSCROLL;
    if (password) style |= ES_PASSWORD;
    HWND h = CreateWindowExW(WS_EX_CLIENTEDGE, L"EDIT", text, style, x, y, w, 24, parent, (HMENU)(INT_PTR)id, NULL, NULL);
    SendMessageW(h, WM_SETFONT, (WPARAM)g_font, TRUE);
    g_controls[id] = h;
    return h;
}

static HWND AddButton(HWND parent, int id, int x, int y, int w, const wchar_t* text) {
    HWND h = CreateWindowW(L"BUTTON", text, WS_CHILD | WS_VISIBLE | BS_PUSHBUTTON, x, y, w, 28, parent, (HMENU)(INT_PTR)id, NULL, NULL);
    SendMessageW(h, WM_SETFONT, (WPARAM)g_font, TRUE);
    return h;
}

static void BuildUi(HWND hWnd) {
    g_font = CreateFontW(18, 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE, DEFAULT_CHARSET, OUT_OUTLINE_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY, VARIABLE_PITCH, L"Microsoft YaHei UI");
    AddLabel(hWnd, 16, 16, 90, L"Base URL"); AddEdit(hWnd, IDC_BASE_URL, 112, 16, 250, L"http://127.0.0.1:8080");
    AddLabel(hWnd, 380, 16, 70, L"AppKey"); AddEdit(hWnd, IDC_APP_KEY, 450, 16, 160, L"demo-app");
    AddLabel(hWnd, 625, 16, 95, L"ClientSecret"); AddEdit(hWnd, IDC_CLIENT_SECRET, 720, 16, 330, L"replace-client-secret");

    AddLabel(hWnd, 16, 52, 90, L"Username"); AddEdit(hWnd, IDC_USERNAME, 112, 52, 160, L"testuser");
    AddLabel(hWnd, 285, 52, 75, L"Password"); AddEdit(hWnd, IDC_PASSWORD, 360, 52, 130, L"123456", true);
    AddLabel(hWnd, 505, 52, 80, L"Machine"); AddEdit(hWnd, IDC_MACHINE, 590, 52, 230, L"WIN64-GUI-PC-001");
    AddLabel(hWnd, 835, 52, 65, L"Version"); AddEdit(hWnd, IDC_CLIENT_VERSION, 900, 52, 150, L"100");

    AddLabel(hWnd, 16, 88, 90, L"Reg Card"); AddEdit(hWnd, IDC_ACCOUNT_CARD, 112, 88, 250, L"");
    AddLabel(hWnd, 380, 88, 95, L"Recharge"); AddEdit(hWnd, IDC_RECHARGE_CARD, 475, 88, 250, L"");
    AddLabel(hWnd, 742, 88, 95, L"Card Login"); AddEdit(hWnd, IDC_STANDALONE_CARD, 837, 88, 213, L"");

    AddLabel(hWnd, 16, 124, 90, L"Acct Token"); AddEdit(hWnd, IDC_ACCOUNT_TOKEN, 112, 124, 435, L"");
    AddLabel(hWnd, 565, 124, 90, L"Card Token"); AddEdit(hWnd, IDC_CARD_TOKEN, 655, 124, 395, L"");

    int y = 166, x = 16, bw = 128, gap = 8;
    AddButton(hWnd, IDC_APPINFO, x, y, bw, L"软件信息"); x += bw + gap;
    AddButton(hWnd, IDC_REGISTER, x, y, bw, L"账号注册"); x += bw + gap;
    AddButton(hWnd, IDC_LOGIN, x, y, bw, L"账号登录"); x += bw + gap;
    AddButton(hWnd, IDC_RECHARGE, x, y, bw, L"账号充值"); x += bw + gap;
    AddButton(hWnd, IDC_ACCOUNT_HEARTBEAT, x, y, bw, L"账号心跳"); x += bw + gap;
    AddButton(hWnd, IDC_ACCOUNT_CLOUD, x, y, bw, L"账号云变量"); x += bw + gap;
    AddButton(hWnd, IDC_TOKEN_UNBIND, x, y, bw, L"Token解绑"); x += bw + gap;
    AddButton(hWnd, IDC_ACCOUNT_UNBIND, x, y, bw, L"密码解绑");

    y += 38; x = 16;
    AddButton(hWnd, IDC_CARD_LOGIN, x, y, bw, L"卡密登录"); x += bw + gap;
    AddButton(hWnd, IDC_CARD_HEARTBEAT, x, y, bw, L"卡密心跳"); x += bw + gap;
    AddButton(hWnd, IDC_CARD_CLOUD, x, y, bw, L"卡密云变量"); x += bw + gap;
    AddButton(hWnd, IDC_CARD_UNBIND, x, y, bw, L"卡密解绑"); x += bw + gap;
    AddButton(hWnd, IDC_CLEAR_LOG, x, y, bw, L"清空日志");

    HWND log = CreateWindowExW(WS_EX_CLIENTEDGE, L"EDIT", L"", WS_CHILD | WS_VISIBLE | WS_VSCROLL | WS_HSCROLL | ES_MULTILINE | ES_READONLY | ES_AUTOVSCROLL | ES_AUTOHSCROLL, 16, 248, 1034, 385, hWnd, (HMENU)(INT_PTR)IDC_LOG, NULL, NULL);
    SendMessageW(log, WM_SETFONT, (WPARAM)g_font, TRUE);
    g_controls[IDC_LOG] = log;
    AppendLog("License Client GUI Demo ready. Fill client_secret and card keys first.");
    AppendLog("Version uses integer now, for example clientVersion=100.");
}

static LRESULT CALLBACK WndProc(HWND hWnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    switch (msg) {
    case WM_CREATE:
        BuildUi(hWnd);
        return 0;
    case WM_COMMAND: {
        int id = LOWORD(wParam);
        if (id == IDC_CLEAR_LOG) { SetWindowTextW(g_controls[IDC_LOG], L""); return 0; }
        if (id >= IDC_APPINFO && id <= IDC_CARD_UNBIND) DoAction(id);
        return 0;
    }
    case WM_DESTROY:
        if (g_font) DeleteObject(g_font);
        PostQuitMessage(0);
        return 0;
    }
    return DefWindowProcW(hWnd, msg, wParam, lParam);
}

int WINAPI wWinMain(HINSTANCE hInst, HINSTANCE, PWSTR, int nCmdShow) {
    INITCOMMONCONTROLSEX icc{ sizeof(icc), ICC_STANDARD_CLASSES };
    InitCommonControlsEx(&icc);
    WNDCLASSW wc{};
    wc.lpfnWndProc = WndProc;
    wc.hInstance = hInst;
    wc.hCursor = LoadCursor(NULL, IDC_ARROW);
    wc.hbrBackground = (HBRUSH)(COLOR_WINDOW + 1);
    wc.lpszClassName = L"LicenseClientGuiDemoWnd";
    RegisterClassW(&wc);
    g_hWnd = CreateWindowW(wc.lpszClassName, L"License SaaS Win64 GUI C++ Demo", WS_OVERLAPPEDWINDOW & ~WS_MAXIMIZEBOX, CW_USEDEFAULT, CW_USEDEFAULT, 1085, 700, NULL, NULL, hInst, NULL);
    ShowWindow(g_hWnd, nCmdShow);
    UpdateWindow(g_hWnd);
    MSG message;
    while (GetMessageW(&message, NULL, 0, 0)) {
        TranslateMessage(&message);
        DispatchMessageW(&message);
    }
    return (int)message.wParam;
}
