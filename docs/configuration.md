# 配置和密钥

## 默认访问

- 默认用户名：`admin`
- 默认密码：`admin123`
- Web 默认端口：HTTP `12333`，HTTPS `12332`
- Emby 代理默认端口：HTTP `8095`，HTTPS `8094`

管理员密码使用 bcrypt 哈希保存，新生成和修改后的密码使用成本参数 `12`。旧成本哈希会在用户下一次成功登录后自动升级。

## 浏览器登录会话

- 浏览器登录使用 `auth_token` HttpOnly Cookie，不在 Web Storage 保存 JWT。
- 服务端通过 `user_sessions` 表控制会话有效性，退出登录、修改密码、两步验证变更和登录设备撤销都会更新该表。
- `csrf_token` Cookie 可被前端读取，前端会在 `POST`、`PUT`、`PATCH`、`DELETE` 请求中通过 `X-CSRF-Token` 发送，服务端同时校验请求来源和 session 中的 CSRF 哈希。
- CORS 和 CSRF 共享可信来源判断：同源请求自动允许，默认允许 Vite 开发来源 `http://localhost:5173`、`http://127.0.0.1:5173` 和 `http://[::1]:5173`。自定义前后端跨源部署时，在 `config/config.yaml` 中配置精确来源：

  ```yaml
  trustedOrigins:
    - https://qms.example.com
  ```

  `trustedOrigins` 按 `scheme://host[:port]` 精确匹配，显式默认端口 `http:80`、`https:443` 会按无端口来源处理。前端和 API 使用同一个域名访问时不需要配置；旧配置缺少该字段会按空列表处理。
  通过 Nginx / Caddy 等反向代理绑定域名时，应保留原始 `Host` 并传递 `X-Forwarded-Proto`，这样同源判断可以按用户访问的域名生效。
- API Key 调用支持 `X-API-Key` header 和 `?api_key=` 查询参数，不需要 CSRF。

## 数据库

首次启动且不存在 `config/config.yaml` 时，后端会先启动配置向导。向导当前提供 SQLite 和外部 PostgreSQL 两种选择；保存后会生成 `config/config.yaml`，旧版 `config.yml` 仍可读取。

代码默认配置是 `postgres + embedded`。Docker 镜像会安装 `postgresql15`，可以直接配合内嵌 PostgreSQL 使用；裸二进制和本地开发环境不随仓库携带 PostgreSQL 二进制，如果要使用 PostgreSQL，建议安装 PostgreSQL 15 及以上并配置为外部数据库，或自行保证内嵌模式所需的 PostgreSQL 命令可用。

数据库引擎、配置项、迁移和维护入口的完整说明见 [数据库](database.md)。

## 需要自备的密钥

- 115 开放平台 APPID：前端支持扫码授权和网页授权；自定义 APPID 走扫码授权 。
- TMDB API Key / Access Token：可在 Web 页面「刮削设置」填写；留空时使用默认值。
- OpenAI 兼容 API Key：默认对接硅基流动（SiliconFlow），可在 Web 页面「刮削设置」填写。
- fanart.tv API Key：可在 Web 页面「刮削设置」填写。

以上默认密钥可在 `backend/main.go` 开头的变量中设置、编译时通过 ldflags 传入，或运行时通过环境变量 / `config/.env` 注入（变量名 `TMDB_API_KEY`、`TMDB_ACCESS_TOKEN`、`SC_API_KEY`、`FANART_API_KEY`）。

取值优先级：Web UI > 环境变量 / `config/.env` > ldflags。`config/.env` 会覆盖真实环境变量。

## 本地敏感数据

两步验证等本机敏感数据使用实例本地密钥：每个实例首次启动自动生成并保存到 `config/encryption.key`。

`jwtSecret` 是 JWT Cookie 会话票据签名密钥。启动时如果为空、仍为当前公开默认值，或仍为历史版本的公开默认值，程序会自动生成 32 字节强随机密钥并写回 `config/config.yaml`；如果配置目录不可写，启动会失败。

修改 `jwtSecret` 会让现有登录 Cookie 无法通过签名校验，用户需要重新登录。

网盘 OAuth 中转使用共享密钥 `OAUTH_RELAY_ENCRYPTION_KEY`，可编译时通过 ldflags 变量 `main.OAuthRelayEncryptionKey` 传入，或运行时通过环境变量 / `config/.env` 注入。
