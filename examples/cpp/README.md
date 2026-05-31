# C++ 客户端对接示例

这个目录提供一个可编译的 C++17 示例，用来对接 License SaaS 客户端接口。

示例内容：

- HMAC-SHA256 签名
- `GET /api/client/app-info`
- `POST /api/client/login`
- `POST /api/client/card-login`
- `POST /api/client/heartbeat` 调用方式示例

## 依赖

Linux / Ubuntu / Debian：

```bash
sudo apt install build-essential libcurl4-openssl-dev libssl-dev
```

Windows：

- 用 vcpkg 安装 `curl` 和 `openssl`
- 或者在 Visual Studio 工程里链接 libcurl + OpenSSL

## 编译

```bash
g++ -std=c++17 client_demo.cpp -o client_demo -lcurl -lssl -lcrypto
```

## 运行

```bash
./client_demo http://127.0.0.1:8080 demo-app 929df12601c3cdbf423c145c90f7e351a134f77c1761fef4db241021333e5066
```

参数分别是：

```text
服务地址
app_key
客户端密钥 client_secret
```

## 签名规则

客户端接口必须带这些 Header：

```text
X-App-Key: demo-app
X-Timestamp: unix 秒
X-Nonce: 每次请求唯一随机串
X-Signature: hex(HMAC-SHA256(client_secret, canonical))
```

签名原文：

```text
METHOD + "\n" + PATH_WITH_QUERY + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + BODY_JSON
```

注意：

- `PATH_WITH_QUERY` 只写 `/api/...`，不要带 `http://host:port`
- GET 没有 body，最后一行为空字符串
- POST 的 `BODY_JSON` 必须和实际发送的 JSON 完全一致
- `X-Nonce` 每次请求都要换，重复会返回 `replay request`
- `X-Timestamp` 要用 10 位 Unix 秒

## 固定测试向量

如果使用这个密钥：

```text
929df12601c3cdbf423c145c90f7e351a134f77c1761fef4db241021333e5066
```

签名原文：

```text
GET
/api/client/app-info?app_key=demo-app
1779979787
otsbrssveuywakt

```

正确签名：

```text
0f489948f6fadd77f63b361dab61fc346b2523ade353e20834cd2775de3e7b9a
```

## 文件

- `client_demo.cpp`：完整 C++ 示例代码
