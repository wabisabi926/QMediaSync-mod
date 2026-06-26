# 请求校验约定

本文档记录 QMediaSync 后端请求校验体系的当前实现和后续约定。已迁移的 HTTP 接口使用 `backend/internal/requests` 下的 Request DTO 绑定请求，复用 `backend/internal/validation` 下的通用规则；未迁移或特殊流程接口以实际代码为准。

## 适用边界

- DTO 面向 HTTP 边界：JSON Body、Query/Form 参数，以及需要从控制器传入统一校验结构的请求参数。
- 纯内部 Go 函数、Service 调用、模型方法和后台任务不为了形式统一额外创建 DTO。
- 路径参数仍允许在控制器中就地解析，例如 `c.Param("id")` 后转换为整数；需要复用时可迁移到通用 ID DTO 或独立辅助函数。
- 数据库存在性、账号归属、权限、任务状态、外部服务连通性等依赖运行时状态的校验保留在控制器或业务层。
- 控制器响应风格保持所在模块既有模式，不因为 DTO 迁移统一改 HTTP 状态码或响应结构。

## 控制器流程

JSON Body 和 Query/Form 参数优先使用 DTO 绑定：

```go
var req requests.UpdateStrmConfigRequest
if err := c.ShouldBind(&req); err != nil {
    // 按所在控制器的既有响应风格返回参数错误
    return
}
if err := req.Validate(); err != nil {
    // 按所在控制器的既有响应风格返回校验错误
    return
}
input := req.ToModel()
```

DTO 负责：

- 字段必填、范围、枚举、格式和简单条件规则。
- 将外部字段转换为模型输入，例如 `ToModel()`、`StrmSettingModel()`、`NormalizedIDs()`。
- 与请求兼容性相关的规范化，例如 OpenList URL 自动补全协议、分页默认值、旧字段映射。

控制器仍负责：

- 调用 `ShouldBind`、`ShouldBindJSON` 或 `ShouldBindQuery`。
- 调用 DTO 的 `Validate()` 或场景化方法，例如 `ValidateSave()`、`ValidateTest()`、`ValidateCreate()`、`ValidateUpdate()`。
- 数据库查询、账号类型匹配、权限检查、任务状态检查和外部请求。
- 保持模块既有错误响应。当前代码中同时存在 `APIResponse`、`gin.H`、HTTP 200 业务错误、HTTP 400/401/403/404/409 等模式。

模型层仍负责：

- 持久化。
- 派生字段写入。
- 与数据库结构强相关的转换。

## 通用规则

`backend/internal/validation` 只放跨模块复用规则，不访问数据库，不依赖 `gin.Context`，不处理具体业务流程。

| 函数 | 规则 |
| --- | --- |
| `NonBlank` | 去除首尾空白后不能为空。 |
| `Length` | 按 rune 计数字符串长度，并拒绝控制字符。 |
| `PositiveID` | `uint` ID 必须大于 0。 |
| `RangeInt` / `RangeInt64` | 整数必须落在闭区间内。 |
| `OneOfInt` / `OneOfString` | 值必须属于显式枚举。 |
| `HTTPURL` | 可配置是否允许空值；非空时必须是 `http` 或 `https` URL，并包含 Host。 |
| `ProxyURL` | HTTP 代理 URL 校验，当前只允许 `http` 或 `https`。 |
| `DownloadProxyURL` | 网盘下载反代 URL 校验，只允许 `115cdn.net`、其子域名、`d.pcs.baidu.com`、`baidupcs.com` 及其子域名。 |
| `Cron` | 使用 `robfig/cron/v3` 的标准 5 段 Cron 解析。 |
| `ExtList` | 扩展名数组可配置是否允许空；非空项必须以 `.` 开头，且不能包含空白字符。 |

通用错误使用 `validation.Error`，错误文本格式为 `字段：原因`。新增规则时应同时补充 `backend/internal/validation` 的 table-driven 测试。

## 当前 DTO 覆盖范围

| 文件 | 覆盖接口类型 | 主要校验 |
| --- | --- | --- |
| `requests/settings.go` | 线程配置、全局 STRM 配置 | 线程范围、页面大小范围、STRM Base URL、Cron、扩展名、STRM 开关枚举。 |
| `requests/sync.go` | 同步路径创建和更新、自定义 STRM 配置 | 来源类型、非本地来源账号 ID、路径必填、自定义配置、继承值 `-1`、远程路径规范化。 |
| `requests/scrape_path.go` | 刮削路径保存 | 来源类型、媒体类型、刮削类型、重命名类型、非本地来源只允许移动重命名、源/目标路径、扩展名、最小文件大小、线程上限、Cron。 |
| `requests/scrape_settings.go` | TMDB、AI、分类和 TMDB 搜索 | URL、语言代码、国家代码、AI 动作枚举、模型名长度、超时范围、分类名称、Genre ID、年份范围。 |
| `requests/accounts.go` | 账号、OpenList 账号、API Key | 账号来源类型、名称长度、115 授权来源组合、OpenList URL 规范化、用户名/密码或 Token、API Key 状态。 |
| `requests/connections.go` | HTTP 代理、OAuth、二维码、远程直链、反代、请求队列限制和统计 | 代理 URL、账号 ID、OAuth 回调 URL、`data`/`payload` 条件必填、二维码 UID、PickCode、反代下载域名白名单、QPS/QPM/QPH、统计窗口和清理天数。 |
| `requests/emby.go` | Emby 配置 | Emby URL、同步 Cron、布尔开关枚举、媒体库 JSON 字符串。 |
| `requests/backup.go` | 备份创建、列表、记录 ID、恢复和配置 | 手动备份原因默认值、分页默认值、备份记录 ID、启用开关、Cron、保留天数、最大备份数、压缩开关。 |
| `requests/notification.go` | Telegram、MeoW、Bark、ServerChan、自定义 Webhook 渠道 | 渠道名称、必填凭据、URL、Webhook 方法、格式、认证方式和模板格式。 |
| `requests/users.go` | 登录、启用/关闭两步验证、当前用户用户名/密码修改 | 登录用户名和密码必填、两步验证码必填、用户名长度、密码最小长度。 |
| `requests/operations.go` | 分页、ID、路径浏览、网盘文件、目录操作、队列、同步/刮削关联、日志、临时图片、版本更新 | 分页默认值和范围、ID 列表、CSV ID、来源类型、文件夹名、路径穿越防护、日志文件名限制、版本号格式、日期范围。 |

迁移临时服务是启动期流程，不纳入公共 `backend/internal/requests` 目录。它在 `backend/internal/migrate` 包内使用私有 DTO 校验 PostgreSQL 测试连接和保存配置请求。

## 重要兼容性规则

- `SyncPathRequest` 同时支持计划中的嵌套 `setting` 字段和旧前端使用的顶层 STRM 字段；存在非零嵌套配置时优先使用 `setting`。
- 同步路径自定义 STRM 配置使用 `-1` 表示继承全局值；全局 STRM 配置不接受 `-1`。
- `SaveRelScrapePathRequest` 同时支持旧字段 `id`、`scrape_path_id` 和新字段 `sync_path_id`、`scrape_path_ids`。
- 同步路径和刮削路径的关联保存允许空 ID 列表，用于清空关联；通用 `IDListRequest` 不允许空列表。
- `IDCSVRequest` 保留 `ids=1,2` 的 Query 格式，用于刮削记录批量操作。
- `QueueListRequest.Status` 当前只绑定为 `int`，不做枚举限制，继续兼容现有前端和模型状态值。
- `AISettingsRequest.EnableAI` 允许空值，避免旧前端或局部保存请求被误拒。
- `CreateOpenListAccountRequest` 会自动补全缺失的 `http://` 协议，并去掉末尾 `/`。
- `LoginRequest` 只做基础必填校验；控制器仍统一返回「登录失败」，不向客户端暴露用户名、密码或验证码的具体失败原因。
- `BackupCreateRequest` 在原因为空时默认使用「手动备份」，与旧控制器行为一致。
- `BackupListRequest` 保留旧分页兼容策略：页码小于 1 时回退为 1，每页数量小于 1 或大于 100 时回退为 20，类型为空时回退为 `all`。
- 备份配置中 `backup_retention` 为 0 时表示不更新或使用既有值；大于 0 时限制为 1 到 365。
- `internal/migrate` 的测试连接请求允许 `database` 为空，并继续固定连接 `dbname=postgres`；保存配置请求要求 `database` 非空。

## 安全敏感校验

- `/proxy-115` 使用 `Proxy115Request` 和 `DownloadProxyURL` 限制目标下载域名，并在重定向时重新校验 Location，避免通过跳转绕过反代白名单。
- 日志读取相关请求只接受日志文件名，不接受绝对路径、路径分隔符或 `..` 路径穿越。
- 临时图片读取请求只接受相对路径，并拒绝绝对路径和路径穿越。
- 创建目录请求拒绝空名称、`.`、`..`、路径分隔符和控制字符。
- Webhook JSON 模板会先替换内置变量再做 JSON 解析；Form 模板必须符合 `key=value&key2=value2` 格式。

## 当前例外

以下接口或参数仍是特殊实现，不应作为新增接口的默认写法：

- 用户会话撤销使用 `session_id` 路径参数，当前直接从 `c.Param("session_id")` 读取。
- 同步记录、同步任务详情、同步路径列表查询仍在 `controllers/sync.go` 使用控制器内局部 Request 结构。
- Cron 预览和 Cron 验证工具接口仍在 `controllers/settings.go` 使用控制器内局部 Request 结构。
- 备份上传恢复使用 multipart 文件流，文件读取、扩展名和临时文件处理仍保留在控制器中。
- 迁移临时服务 `internal/migrate/server.go` 使用独立的启动期接口和包内私有 DTO，不纳入常规 API DTO 目录。
- 部分只读或触发型接口没有外部参数，或只做运行状态检查，不需要 DTO。

新增或改造接口时，不应继续扩大这些例外；如果改动触及上述接口，可以顺手迁移到 `backend/internal/requests`，但要保持外部响应兼容。

## 前端规则

前端校验用于即时反馈和减少误操作，不能替代后端校验，也不作为安全边界。与后端一致的范围和枚举常量放在 `frontend/src/constants/validation.ts`：

- `THREAD_LIMITS`：下载线程、文件详情线程、OpenList QPS、重试次数、重试延迟、文件列表分页大小。
- `SCRAPE_THREAD_LIMITS`：本地刮削最大线程 20、远程刮削最大线程 5、最小线程 1。
- `STRM_GLOBAL_OPTIONS` 和 `STRM_CUSTOM_OPTIONS`：全局配置与自定义配置的 STRM 开关枚举。
- `HTTP_URL_PATTERN`：前端 URL 输入提示使用，后端仍以 `validation.HTTPURL` 为准。
- `CRON_DEFAULTS`：前端默认 Cron 值。

调整后端范围或枚举时，必须同步检查该文件和相关表单组件，避免前后端提示不一致。

## 测试要求

- 通用规则测试放在 `backend/internal/validation`。
- Request DTO 测试放在 `backend/internal/requests`，按模块拆分。
- DTO 测试至少覆盖合法请求、必填缺失、枚举错误、范围错误、格式错误和关键条件规则。
- 涉及兼容性逻辑时必须补充回归测试，例如旧字段映射、空关联列表、默认分页值和 URL 规范化。
- 修改后端校验时运行 `(cd backend && go test ./...)`。
- 修改前端校验常量或表单时至少运行 `(cd frontend && pnpm run type-check)`；影响构建链路时运行 `(cd frontend && pnpm run build)`。

## 新增或迁移接口清单

1. 在 `backend/internal/requests` 新增或复用 Request DTO。
2. 使用 `form`、`json` 标签匹配现有外部字段名，避免破坏前端兼容性。
3. 在 `Validate()` 中优先复用 `backend/internal/validation`；只有业务条件规则留在 DTO 内。
4. 需要进入模型层时提供 `ToModel()` 或语义明确的转换方法。
5. 控制器绑定请求后立即调用 `Validate()`，再执行数据库或外部服务校验。
6. 保持原控制器响应风格，除非明确要做 API 行为变更。
7. 补充 DTO 和通用规则测试。
8. 如果改动字段范围、枚举或默认值，同步更新 `frontend/src/constants/validation.ts` 和本文档。

## 后续可补充内容

- 按路由生成「接口 → DTO → 测试文件」映射表，便于审查覆盖率。
- 统一路径参数 ID 的小型辅助函数或 DTO，减少控制器内重复 `strconv.ParseUint`。
- 为 Swagger 注解补充 DTO 字段说明，避免接口文档仍停留在旧的散字段描述。
