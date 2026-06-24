# 配置和密钥

## 默认访问

- 默认用户名：`admin`
- 默认密码：`admin123`
- Web 默认端口：HTTP `12333`，HTTPS `12332`
- Emby 代理默认端口：HTTP `8095`，HTTPS `8094`

## 数据库

开源版本不包含 PostgreSQL 数据库二进制文件，需要自行安装。建议使用 PostgreSQL 15 及以上，然后通过环境变量配置使用。

## 需要自备的密钥

- 115 开放平台 AppID：当前改为使用 OAuth 授权方式，开发者需要根据代码自行实现 OAuth 服务端来和 115 通信，或改为二维码扫码登录授权。
- TMDB API Key：可在 Web 页面「刮削设置」填写；刮削实际使用 v3 API Key。
- OpenAI 兼容 API Key：目前使用硅基流动，可在 Web 页面「刮削设置」填写。
- Fanart.tv API Key：可在 Web 页面「刮削设置」填写。

以上 key 可在 `backend/main.go` 开头的变量中设置、编译时通过 ldflags 传入，或运行时通过环境变量 / `config/.env` 注入（变量名 `TMDB_API_KEY`、`TMDB_ACCESS_TOKEN`、`SC_API_KEY`、`FANART_API_KEY`，无 `DEFAULT_` 前缀）。

取值优先级：Web UI > 环境变量 > ldflags。

## 本地敏感数据

两步验证等本机敏感数据使用实例本地密钥：每个实例首次启动自动生成并保存到 `config/encryption.key`。

网盘 OAuth 中转使用共享密钥 `OAUTH_RELAY_ENCRYPTION_KEY`，可编译时通过 ldflags 变量 `main.OAuthRelayEncryptionKey` 传入，或运行时通过环境变量 / `config/.env` 注入。
