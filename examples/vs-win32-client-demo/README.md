# License SaaS Visual Studio Win32 C++ 客户端示例

这是给授权平台配套的 Windows C++ 示例工程，使用 Visual Studio 编译。

## 功能覆盖

### 账号模式

- 获取软件信息：`GET /api/client/app-info`
- 用户注册并激活卡密：`POST /api/client/register`
- 用户登录：`POST /api/client/login`
- 账号充值/续费：`POST /api/client/recharge`
- 心跳：`POST /api/client/heartbeat`
- 读取云变量：`GET /api/client/cloud-vars`
- token 账号解绑：`POST /api/client/unbind`
- 账号密码自助解绑：`POST /api/client/account-unbind`

### 单卡密模式

- 卡密登录/首次绑定机器码：`POST /api/client/card-login`
- 卡密心跳：`POST /api/client/heartbeat`
- 卡密模式读取云变量：`GET /api/client/cloud-vars`
- 卡密解绑：`POST /api/client/card-unbind`

## 编译环境

- Visual Studio 2022
- Windows SDK
- 平台选择：`Win32` 或 `x64` 都可以
- 配置选择：`Release`

示例不依赖 OpenSSL/libcurl，只使用 Windows 自带：

- WinHTTP：HTTP/HTTPS 请求
- BCrypt：HMAC-SHA256 签名

项目文件里已经链接：

```text
winhttp.lib
bcrypt.lib
```

## 使用步骤

1. 打开：

```text
LicenseClientDemo.sln
```

2. 打开源码：

```text
LicenseClientDemo/main.cpp
```

3. 修改顶部配置：

```cpp
static const std::wstring kBaseUrl = L"http://127.0.0.1:8080";
static const std::string kAppKey = "demo-app";
static const std::string kClientSecret = "请替换成后台软件的client_secret";

static const std::string kAccountCardKey = "请替换成账号注册用卡密";
static const std::string kRechargeCardKey = "请替换成账号充值用卡密";
static const std::string kStandaloneCardKey = "请替换成卡密登录用卡密";
```

其中：

- `kAppKey`：后台软件管理里的 `app_key`
- `kClientSecret`：后台软件管理里的客户端签名密钥/client_secret
- `kAccountCardKey`：用于账号注册激活的未使用卡密
- `kRechargeCardKey`：用于账号续费充值的未使用卡密，可不填，不填则跳过充值演示
- `kStandaloneCardKey`：用于单卡密登录的未使用卡密

4. 编译运行。

## 签名规则

每个 `/api/client/*` 请求都带：

```text
X-App-Key: <app_key>
X-Timestamp: <10位Unix秒时间戳>
X-Nonce: <每次请求唯一随机串>
X-Signature: <HMAC-SHA256小写hex>
Authorization: Bearer <client_token>   // 心跳、云变量等登录后接口需要
```

签名原文：

```text
METHOD + "\n" + PATH_WITH_QUERY + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + BODY_JSON
```

注意：

- `PATH_WITH_QUERY` 只签 `/api/...`，不要签 `http://127.0.0.1:8080`
- GET 请求 `BODY_JSON` 是空字符串，但最后一行换行仍然存在
- POST 请求签名用的 JSON 字符串必须和实际发送 body 完全一致
- 时间戳是 10 位秒，不是 13 位毫秒
- nonce 每次请求都必须换，重复会被服务端拒绝

## 固定签名测试

程序启动会先跑固定测试：

```text
secret = 929df12601c3cdbf423c145c90f7e351a134f77c1761fef4db241021333e5066
canonical = GET\n/api/client/app-info?app_key=demo-app\n1779979787\notsbrssveuywakt\n
expected = 0f489948f6fadd77f63b361dab61fc346b2523ade353e20834cd2775de3e7b9a
```

如果显示：

```text
签名测试: OK
```

说明本地 HMAC 算法没问题。

## 重要提醒

这个示例是控制台 demo，主要给你移植到正式客户端使用。正式产品里不要明文暴露 `client_secret`，建议配合壳、混淆、分段存储、服务端风控、机器码绑定、心跳校验一起使用。
