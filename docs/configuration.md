# 配置和密钥

## 默认端口

- Web 默认端口：HTTP `12333`，HTTPS `12332`
- Emby 代理默认端口：HTTP `8095`，HTTPS `8094`

管理员用户名和密码在创建和修改时使用严格校验规则：用户名去除首尾空白后长度必须为 3 到 20 个字符且只能包含英文和数字，密码长度至少 6 个字符，且不能是纯数字或纯字母。登录时校验用户名和密码非空，用户名 20 个字符上限。管理员密码使用 bcrypt 哈希保存，新生成和修改后的密码使用成本参数 `12`。旧成本哈希会在用户下一次成功登录后自动升级。

## 首次管理员初始化

`config.yaml` 不保存管理员用户名和密码。首次启动并完成数据库初始化后，如果 `users` 表为空，程序会生成一次性初始化码并写入启动日志。打开 Web 登录页后，系统会显示创建管理员表单；输入启动日志中的初始化码、管理员用户名和密码后，后端会使用 bcrypt 哈希密码并创建首个管理员。

初始化码只在用户表为空时生成，创建管理员成功后立即失效。`users` 表通过唯一约束保证系统只存在一个登录用户；若尚未创建管理员就重启程序，会生成新的初始化码。首次初始化应在可信网络内完成，避免无关人员访问 Web 页面。

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
- API Key 调用支持 `X-API-Key` header 和 `?api_key=` 查询参数，不需要 CSRF。`/emby/webhook` 配置默认启用鉴权，优先读取 `X-API-Key`，并保留 `?api_key=` 兼容只能配置 URL 的 Emby Webhook 场景。
- 在「系统设置 - API Key」创建密钥后，后端会生成 `qms_` 前缀加 24 位随机字符的完整密钥。完整密钥只在创建响应中返回一次；数据库 `api_keys` 表保存 `user_id`、`name`、`key_hash`、前 8 位 `key_prefix`、`is_active`、时间戳和 `last_used_at`，不保存完整明文。校验时对请求中的 Key 再做同样 SHA256，用 `key_hash` + `is_active=true` 查表。
- 本地下载反代 `/proxy-115` 仅允许访问 115 CDN 和百度网盘下载域名，用于 115 / 百度网盘播放代理和媒体信息提取；初始目标和每次重定向目标都会执行同一白名单校验，其他目标地址会被拒绝。

## 数据库

首次启动且不存在 `config/config.yaml` 时，后端会先启动配置向导。向导当前提供 SQLite 和外部 PostgreSQL 两种选择；保存后会生成 `config/config.yaml`，旧版 `config.yml` 仍可读取。

完整配置项示例见 [config.yaml](examples/config.yaml)。示例文件只用于说明字段含义，运行时仍以 `config/config.yaml` 为准。

代码默认配置是 `postgres + embedded`。Docker 镜像会安装 `postgresql15`，可以直接配合内嵌 PostgreSQL 使用；裸二进制和本地开发环境不随仓库携带 PostgreSQL 二进制，如果要使用 PostgreSQL，建议安装 PostgreSQL 15 及以上并配置为外部数据库，或自行保证内嵌模式所需的 PostgreSQL 命令可用。

数据库引擎、配置项、迁移和维护入口的完整说明见 [数据库](database.md)。

## 115 接口监控统计

首页“115 接口监控”卡片中的请求数、QPS、QPM、QPH、平均响应时间和限流次数来自 `request_stats` 表，会在程序重启后继续按时间窗口聚合展示。

“限流中 / 运行正常”和剩余限流时间来自当前进程的 115 请求队列限流管理器。当前限流暂停时长为 1 分钟，程序重启后该运行态会恢复为未限流。

## Emby 302 出站 HTTPS

Emby 302 代理请求 Emby、OpenList、m3u8 和下载资源时，默认启用 HTTPS 证书校验。共享 HTTP client 会复用空闲连接，避免同一上游的连续请求反复建立 TCP / TLS 连接。

只有在受控内网自签名证书或临时排障场景下，才建议显式跳过证书校验：

```yaml
emby302:
  insecure_skip_verify: false
```

将 `emby302.insecure_skip_verify` 改为 `true` 后，Emby 302 出站 HTTPS 请求会接受无法验证的证书。程序启动时会写入风险提示日志；该模式存在中间人攻击风险，不建议在公网或长期生产环境开启。

## 日志行为和脱敏

日志文件路径由 `config/config.yaml` 的 `log` 配置决定，默认相对于配置目录写入：

| 配置项 | 默认文件 | 用途 |
| --- | --- | --- |
| `log.file` | `logs/app.log` | 主程序日志，包含 Web、控制器、模型、Emby 302 等通用日志。 |
| `log.v115` | `logs/115.log` | 115 开放平台相关请求和队列日志。 |
| `log.openList` | `logs/openList.log` | OpenList 相关日志。 |
| `log.tmdb` | `logs/tmdb.log` | TMDB 刮削相关日志。 |
| `log.baiduPan` | `logs/baidupan.log` | 百度网盘相关日志。 |
| `log.web` | `logs/web.log` | 预留 Web 日志配置。 |
| `log.syncLogDir` | `logs/sync` | 同步任务独立日志目录配置。 |

当前自定义 `QLogger` 不提供运行时日志等级过滤；`Info`、`Warn`、`Error`、`Debug` 和显式敏感 `SensitiveDebug` 都会写入对应日志。日志前缀用于区分等级，但不会因为运行模式自动屏蔽 `Debug`。`gin.ReleaseMode` 只影响 Gin 自身模式，不控制 `QLogger` 的输出。

运行日志默认会在写入前完全脱敏常见敏感字段，包括 `api_key`、`X-Emby-Token`、`Authorization`、`X-Emby-Authorization`、`X-API-Key`、`password`、`access_token`、`refresh_token`、`AccessKeySecret`、`SecurityToken`、`Cookie` 等。普通 `Info`、`Warn`、`Error` 和 `Debug` 日志都会执行脱敏，不保留敏感值开头或结尾字符。脱敏后的敏感值统一显示为 `******`。

需要临时排查 Emby 302 等链路的完整请求信息时，可以在本地调试环境设置 `QMS_UNSAFE_SENSITIVE_LOG=1`。该开关只影响显式标记为敏感的 `SensitiveDebug` 日志；启用后这类 Debug 日志可能包含 API Key、Token、Cookie 或密码，程序启动时会写入风险提示。不应在生产环境长期打开，也不应分享对应日志文件。

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
