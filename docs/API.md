# API 文档

Base URL: `http://127.0.0.1:8080`

## 客户端签名协议

所有 `/api/client/...` 接口现在都必须带 HMAC-SHA256 签名。

Header：

```text
X-Timestamp: unix 秒
X-Nonce: 每次请求唯一随机字符串
X-Signature: hex(HMAC-SHA256(secret, canonical))
X-App-Key: demo-app
```

签名原文：

```text
METHOD + "\n" + PATH_WITH_QUERY + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + BODY_JSON
```

示例：

```text
POST
/api/client/login
1779950000
nonce-001
{"appKey":"demo-app","username":"u1","password":"123456","machineCode":"PC-001"}
```

默认开发密钥：

```text
demo-client-secret-change-me
```

生产环境请设置：

```bash
APP_CLIENT_SIGN_SECRET='换成强随机密钥'
```

服务端校验：

- 缺签名返回 `401 missing client signature`
- 签名错误返回 `401 bad signature`
- 时间戳过期返回 `401 timestamp expired`
- nonce 重复返回 `409 replay request`

## 管理端

### 登录

`POST /api/admin/login`

```json
{"username":"admin","password":"admin123"}
```

返回：

```json
{"ok":true,"token":"...","username":"admin"}
```

后续管理端请求 Header：

```text
Authorization: Bearer ***
```

### 软件

- `GET /api/admin/apps`
- `POST /api/admin/apps`
- `PUT /api/admin/apps/:id`

创建：

```json
{"name":"我的软件","appKey":"my-app","version":"1.0.0","announcement":"公告"}
```

### 用户

- `GET /api/admin/users?keyword=&status=&app_id=`
- `POST /api/admin/users`
- `PUT /api/admin/users/:id`
- `PUT /api/admin/users/:id/password`
- `PUT /api/admin/users/:id/unbind`
- `GET /api/admin/users/:id/devices`

创建用户：

```json
{"appId":1,"username":"u1","password":"123456","maxDevices":1}
```

编辑用户：

```json
{"status":"active","expireAt":"2027-01-01 00:00:00","machineCode":"PC-001","maxDevices":2}
```

修改密码：

```json
{"password":"654321"}
```

解绑会清空用户机器码，并禁用该用户历史设备记录。

### 卡类

- `GET /api/admin/card-types`
- `POST /api/admin/card-types`

### 卡密

- `GET /api/admin/cards?keyword=&status=&app_id=&agent_id=`
- `GET /api/admin/cards/export?keyword=&status=&app_id=&agent_id=`
- `POST /api/admin/cards/generate`
- `PUT /api/admin/cards/:id/disable`
- `PUT /api/admin/cards/:id/enable`

生成卡密：

```json
{"appId":1,"cardTypeId":1,"count":10}
```

### 代理管理

- `GET /api/admin/agents`
- `POST /api/admin/agents`
- `PUT /api/admin/agents/:id`
- `POST /api/admin/agents/:id/balance`
- `GET /api/admin/agents/:id/balance-logs`

创建代理：

```json
{
  "username":"agent001",
  "password":"123456",
  "balance":100,
  "remark":"一级代理",
  "appIds":[1],
  "cardTypeIds":[1]
}
```

余额调整：

```json
{"type":"add","amount":100,"remark":"线下收款后加余额"}
```

扣余额：

```json
{"type":"deduct","amount":20,"remark":"人工扣除"}
```

### 设备管理

- `GET /api/admin/devices?keyword=&status=&app_id=`
- `PUT /api/admin/devices/:id/status`

更新设备状态：

```json
{"status":"banned"}
```

可用状态：`active`、`banned`、`disabled`。

## 代理端

### 登录

`POST /api/agent/login`

```json
{"username":"agent001","password":"123456"}
```

返回：

```json
{"ok":true,"token":"...","username":"agent001","id":1}
```

后续代理端请求 Header：

```text
Authorization: Bearer ***
```

### 当前代理信息和权限范围

- `GET /api/agent/me`
- `GET /api/agent/scopes`
- `GET /api/agent/stats`
- `GET /api/agent/balance-logs`

### 代理发卡

`POST /api/agent/cards/generate`

```json
{"appId":1,"cardTypeId":1,"count":1}
```

规则：

- 代理只能给管理员授权的软件和卡类发卡。
- 发卡按卡类价格自动扣代理余额。
- 余额不足返回错误，不生成卡密。
- 生成的卡密会绑定 `agent_id`，代理只能看到自己的卡密。

### 代理卡密列表

`GET /api/agent/cards`

## 客户端

### 获取软件信息

`GET /api/client/app-info?app_key=demo-app`

返回公告、版本号、强更、下载地址。

### 注册

`POST /api/client/register`

```json
{"appKey":"demo-app","username":"u1","password":"123456"}
```

### 登录

`POST /api/client/login`

```json
{"appKey":"demo-app","username":"u1","password":"123456","machineCode":"PC-001"}
```

返回：

```json
{"ok":true,"client_token":"...","expire_at":"..."}
```

### 卡密充值

`POST /api/client/recharge`

```json
{"appKey":"demo-app","username":"u1","cardKey":"LS-XXXX-XXXX-XXXX"}
```

### 心跳

`POST /api/client/heartbeat`

Header：

```text
Authorization: Bearer ***
```

Body：

```json
{"machineCode":"PC-001","clientVersion":"0.1.0"}
```
