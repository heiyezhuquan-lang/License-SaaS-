# 易语言客户端签名说明

客户端接口必须带 4 个签名头：

```text
X-Timestamp: 当前 Unix 秒
X-Nonce: 每次请求不同的随机字符串
X-Signature: HMAC-SHA256 十六进制结果
X-App-Key: demo-app
```

签名原文格式固定：

```text
请求方法 + 换行符 + 路径含查询 + 换行符 + 时间戳 + 换行符 + nonce + 换行符 + JSON正文
```

例如登录：

```text
POST
/api/client/login
1779950000
abc-random-001
{"appKey":"demo-app","username":"u1","password":"123456","machineCode":"PC-001"}
```

然后计算：

```text
签名 = HMAC_SHA256_HEX("demo-client-secret-change-me", 上面的签名原文)
```

HTTP 请求头：

```text
Content-Type: application/json
X-App-Key: demo-app
X-Timestamp: 1779950000
X-Nonce: abc-random-001
X-Signature: 签名结果
```

伪代码：

```text
body = "{\"appKey\":\"demo-app\",\"username\":\"u1\",\"password\":\"123456\",\"machineCode\":\"PC-001\"}"
timestamp = 到文本(取现行时间戳())
nonce = 取随机UUID()
canonical = "POST" + #换行符 + "/api/client/login" + #换行符 + timestamp + #换行符 + nonce + #换行符 + body
signature = HMAC_SHA256_HEX("demo-client-secret-change-me", canonical)

headers = "Content-Type: application/json" + #换行符
headers = headers + "X-App-Key: demo-app" + #换行符
headers = headers + "X-Timestamp: " + timestamp + #换行符
headers = headers + "X-Nonce: " + nonce + #换行符
headers = headers + "X-Signature: " + signature

返回 = HTTP_POST("http://127.0.0.1:8080/api/client/login", body, headers)
```

注意：

- JSON 字符串参与签名的内容必须和实际发送的 body 完全一致。
- 同一个 nonce 只能用一次，重复会返回重放请求错误。
- 时间戳允许误差默认约 5 分钟。
- 生产环境必须更换 `APP_CLIENT_SIGN_SECRET`。
