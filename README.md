# License SaaS v1.1

轻量级软件授权 / 卡密 / 代理商 / 客户端安全协议管理系统。

- 后端：Go + Gin + SQLite
- 前端：Vue 3 + Vite + Naive UI
- 部署：单端口 same-origin，后端同时提供 `/api/...` 和前端页面
- 客户端：HMAC-SHA256 签名、nonce 防重放、账号模式、卡密模式、心跳、云变量、解绑

> 当前开源版本：`1.1`

## 功能概览

### 管理端

- 首次安装初始化管理员
- 总览仪表盘
- 软件管理 / 软件配置
  - App Key
  - 客户端签名密钥 `client_secret`
  - 整数版本号 `version` / `min_version`
  - 公告
  - 强制更新
  - 下载地址
  - 心跳间隔 / 超时配置
- 用户管理
  - 客户端注册账号
  - 状态管理：正常 / 禁用 / 封禁
  - 到期时间
  - 机器码
  - 设备限制
  - 管理端解绑
  - 删除用户
- 卡类套餐
  - 授权小时
  - 最大设备数
  - 解绑策略
  - 零售价 / 代理成本
- 卡密管理
  - 生成卡密
  - 禁用 / 启用
  - 按软件、代理、卡类、状态筛选
- 代理管理
  - 创建代理账号
  - 启用 / 禁用 / 删除
  - 手动加扣余额
  - 授权可售软件和卡类
- 设备管理
  - 在线设备
  - 禁用 / 封禁 / 恢复
  - 账号授权和卡密授权设备
- 云变量
  - 软件级文本 key/value 配置
  - 客户端登录后读取
- 销量查询
  - 管理端 / 代理端销售统计
  - 支持生成时间 / 激活时间筛选
  - 支持卡类、日期范围、是否包含禁用卡密
- 客户端接口 / 安全协议文档页

### 代理端

- 代理登录
- 查看余额
- 生成授权卡密
- 查看自己的卡密
- 查看销售记录和余额流水

### 客户端接口

所有 `/api/client/*` 接口使用 HMAC-SHA256 签名。

支持：

- 获取软件信息：`GET /api/client/app-info`
- 账号注册：`POST /api/client/register`
- 账号登录：`POST /api/client/login`
- 账号充值：`POST /api/client/recharge`
- 账号心跳：`POST /api/client/heartbeat`
- 账号读取云变量：`GET /api/client/cloud-vars`
- Token 解绑：`POST /api/client/unbind`
- 账号密码自助解绑：`POST /api/client/account-unbind`
- 单卡登录：`POST /api/client/card-login`
- 单卡解绑：`POST /api/client/card-unbind`

## 目录结构

```text
backend/        Go/Gin 后端
frontend/       Vue 3 前端
scripts/        本地启动和验证脚本
docs/           API 文档
examples/       C++ / 易语言 / Visual Studio 示例
```

## 环境要求

- Go 1.22+
- Node.js 18+
- npm
- SQLite 由 Go 驱动自动使用，不需要单独部署数据库服务

## 本地开发启动

### 1. 安装前端依赖

```bash
cd frontend
npm install
```

### 2. 构建前端

```bash
npm run build
```

### 3. 启动后端

```bash
cd ../backend
APP_FRONTEND_DIST=../frontend/dist \
APP_DB=../license-saas.db \
APP_ADDR=:8080 \
go run ./cmd/server
```

然后打开：

```text
http://127.0.0.1:8080
```

首次运行会进入初始化管理员页面，请自行设置管理员账号和密码。

## 单文件内嵌前端构建

如果要把前端静态文件内嵌进 Go 程序：

```bash
cd frontend
npm run build

cd ../backend
rm -rf cmd/server/webdist
cp -a ../frontend/dist cmd/server/webdist
go build -o ../license-saas-server ./cmd/server
```

运行：

```bash
cd ..
./license-saas-server
```

访问：

```text
http://127.0.0.1:8080
```

## 验证

后端启动后：

```bash
python3 scripts/verify.py
```

或者：

```bash
./scripts/verify.sh
```

后端测试：

```bash
cd backend
go test ./...
```

前端构建：

```bash
cd frontend
npm run build
```

## 重要环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `APP_ADDR` | `:8080` | 监听地址 |
| `APP_DB` | `./license-saas.db` | SQLite 数据库路径 |
| `APP_JWT_SECRET` | `dev-secret-change-me` | JWT 密钥，生产环境必须修改 |
| `APP_FRONTEND_DIST` | 空 | 指向前端 `dist` 目录；为空时使用内嵌前端 |
| `APP_ADMIN_USER` | 空 | 可选：首次启动自动创建管理员用户名 |
| `APP_ADMIN_PASS` | 空 | 可选：首次启动自动创建管理员密码 |

生产环境建议显式设置强随机 `APP_JWT_SECRET`。

## 客户端签名规则

请求头：

```text
X-App-Key: <软件 App Key>
X-Timestamp: <Unix 秒级时间戳>
X-Nonce: <每次请求唯一随机串>
X-Signature: <HMAC-SHA256 小写 hex>
Content-Type: application/json
```

签名原文：

```text
METHOD + "\n" + PATH_WITH_QUERY + "\n" + TIMESTAMP + "\n" + NONCE + "\n" + BODY_JSON
```

说明：

- `client_secret` 是 HMAC key。
- GET 请求 `BODY_JSON` 为空字符串。
- `PATH_WITH_QUERY` 只包含路径和查询参数，不包含域名。
- 换行符必须是 LF，也就是 `\n`。
- 签名输出为 64 字符小写 hex。

## 示例客户端

```text
examples/cpp/                         C++ 控制台示例
examples/e-lang/                      易语言签名说明
examples/vs-win32-client-demo/        Visual Studio Win32 控制台示例
examples/vs-win64-gui-client-demo/    Visual Studio Win64 带界面示例
```

其中 Win64 GUI 示例包含窗口、输入框、按钮和日志框，适合直接测试客户端接口。


## License

MIT
