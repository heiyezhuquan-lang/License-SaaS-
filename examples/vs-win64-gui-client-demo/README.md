# License SaaS Win64 GUI C++ 客户端示例

这是带 Windows 窗口界面的 C++ 示例，不是控制台程序。

## 工程

```text
LicenseClientGuiDemo.sln
LicenseClientGuiDemo/LicenseClientGuiDemo.vcxproj
LicenseClientGuiDemo/main.cpp
```

## 编译

使用 Visual Studio 2022 打开：

```text
LicenseClientGuiDemo.sln
```

选择：

```text
Release | x64
```

然后点击：

```text
生成 -> 生成解决方案
```

这是 Win64 GUI 程序，子系统是 Windows：

```text
<Link><SubSystem>Windows</SubSystem></Link>
```

## 依赖

只用 Windows SDK 自带库，不需要 OpenSSL，不需要 libcurl：

```text
winhttp.lib
bcrypt.lib
comctl32.lib
```

## 界面功能

窗口里有这些输入框：

- Base URL：例如 `http://127.0.0.1:8080`
- AppKey：后台软件的 App Key
- ClientSecret：后台软件的客户端签名密钥
- Username / Password
- Machine：机器码
- Version：客户端整数版本号，例如 `100`
- Reg Card：账号注册用卡密
- Recharge：账号充值用卡密
- Card Login：单卡密登录用卡密
- Acct Token：账号登录/注册返回的 token，成功后自动填入
- Card Token：卡密登录返回的 token，成功后自动填入
- 日志框：显示 HTTP 状态码和返回 JSON

按钮包括：

### 账号模式

- 软件信息
- 账号注册
- 账号登录
- 账号充值
- 账号心跳
- 账号云变量
- Token解绑
- 密码解绑

### 卡密模式

- 卡密登录
- 卡密心跳
- 卡密云变量
- 卡密解绑

## 使用步骤

1. 先在后台创建软件。
2. 复制软件的 AppKey 和 ClientSecret。
3. 在后台生成卡密。
4. 打开本示例程序。
5. 填写：

```text
Base URL
AppKey
ClientSecret
机器码
卡密
```

6. 点击对应按钮测试。

## 版本号说明

现在平台的软件版本号已经改成整数。

示例界面里的 Version 输入框默认是：

```text
100
```

心跳提交的是：

```json
{"machineCode":"...","clientVersion":"100"}
```

客户端判断更新时建议：

```cpp
int localVersion = 100;
int minVersion = 120;

if (localVersion < minVersion) {
    // 版本太低，需要强制更新
}
```

## 签名规则

每个 `/api/client/*` 请求都会自动带：

```text
X-App-Key
X-Timestamp
X-Nonce
X-Signature
Authorization: Bearer <client_token>   // 需要 token 的接口才带
```

签名原文：

```text
METHOD + "\n" + PATH_WITH_QUERY + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + BODY_JSON
```

注意：

- 只签 `/api/...` 路径和 query，不签完整 URL。
- GET 的 body 是空字符串。
- POST 签名用的 JSON 必须和实际发送的 JSON 完全一致。
- 时间戳是 10 位秒级 Unix 时间，不是毫秒。
- nonce 每次请求自动随机生成。

## 常见问题

### 1. 提示签名失败

检查：

- ClientSecret 是否复制完整。
- AppKey 是否和软件一致。
- 服务器时间和电脑时间是否差太多。

### 2. 账号云变量/心跳提示未登录

先点：

```text
账号登录
```

成功后 Acct Token 会自动填入，再点账号心跳/账号云变量。

### 3. 卡密云变量/心跳提示未登录

先点：

```text
卡密登录
```

成功后 Card Token 会自动填入，再点卡密心跳/卡密云变量。

### 4. HTTP 连接失败

确认平台服务正在运行，并且 Base URL 正确，例如：

```text
http://127.0.0.1:8080
```
