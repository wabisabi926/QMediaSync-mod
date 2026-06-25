# 配置和密钥

## 默认访问

- 默认用户名：`admin`
- 默认密码：`admin123`
- Web 默认端口：HTTP `12333`，HTTPS `12332`
- Emby 代理默认端口：HTTP `8095`，HTTPS `8094`

管理员密码使用 bcrypt 哈希保存，新生成和修改后的密码使用成本参数 `12`。旧成本哈希会在用户下一次成功登录后自动升级。

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

`jwtSecret` 是 JWT Cookie 会话票据签名密钥。启动时如果为空或仍为公开默认值，程序会自动生成 32 字节强随机密钥并写回 `config/config.yaml`；如果配置目录不可写，启动会失败。

修改 `jwtSecret` 会让现有登录 Cookie 无法通过签名校验，用户需要重新登录。

网盘 OAuth 中转使用共享密钥 `OAUTH_RELAY_ENCRYPTION_KEY`，可编译时通过 ldflags 变量 `main.OAuthRelayEncryptionKey` 传入，或运行时通过环境变量 / `config/.env` 注入。
