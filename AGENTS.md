# AGENTS.md - QMediaSync

本文档是 AI 编码助手在本仓库工作的约束文档。优先遵循用户当前请求；当请求未明确细节时，按本文档和现有代码风格做保守实现。

## 工作原则

- 先阅读相关代码、文档和现有调用方，再修改文件。
- 保持改动聚焦在用户请求范围内，不做无关重构、格式化或依赖更新。
- 工作区可能有用户未提交改动；不要回滚、覆盖或整理与任务无关的修改。
- 修改已有接口、模型、配置或前端交互时，优先保持所在模块的既有模式。
- 中文文案、注释、日志和展示性描述遵循中英文排版习惯；品牌名、产品名、协议名使用官方名称、大小写和空格。
- 修改代码、配置、接口、命令或流程时，必须同步更新对应文档，保持文档和实现实时一致，可通过 `docs/README.md` 正式文档索引来查找对应文档，新增或迁移文档时同样需要同步维护其中的链接。
- 新增行为必须有相应验证；无法运行验证时，在最终回复中说明原因和剩余风险。
- 不要把被忽略的本地辅助目录作为正式项目文档入口，例如 `docs/superpowers/`。

## 项目事实

QMediaSync 是媒体同步和刮削系统，用于管理 115 网盘、百度网盘、OpenList 等云存储与 Emby 媒体服务器之间的文件同步、STRM 生成、媒体刮削（TMDB/fanart.tv）等功能。

- 语言：Go 1.25
- 模块名：`Q115-STRM`
- Web 框架：Gin
- ORM：GORM
- 数据库：SQLite / PostgreSQL（支持内嵌和外部）
- 后端目录：`backend/`
- 前端目录：`frontend/`，Vue/Vite，生产构建输出到 `backend/web_statics/`

## 常用命令

按任务选择最小必要命令，不要为了无关验证运行大范围构建。

```bash
# 构建前端静态文件
(cd frontend && corepack enable && corepack prepare pnpm@11 --activate && pnpm install --frozen-lockfile && pnpm run build)

# 前端代码检查 / 格式化
(cd frontend && pnpm lint)
(cd frontend && pnpm format)

# 本地构建后端
(cd backend && go build -o QMediaSync .)

# 构建并注入版本信息
(cd backend && CGO_ENABLED=0 go build -ldflags="-s -w -X main.Version=v1.0.0 -X 'main.PublishDate=2026-01-01'" -o QMediaSync .)

# 下载 / 整理 Go 依赖
(cd backend && go mod tidy)

# 跨平台构建（正式发布由 .github/workflows/release.yaml 执行）
(cd backend && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o QMediaSync.exe .)
(cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o QMediaSync .)

# Docker 构建
docker build -f docker/source.Dockerfile -t qmediasync .
```

## 验证要求

后端常用验证：

```bash
# 运行所有测试
(cd backend && go test ./...)

# 运行单个包测试
(cd backend && go test ./internal/helpers/)
(cd backend && go test ./internal/models/)
(cd backend && go test ./internal/synccron/)

# 运行单个测试函数
(cd backend && go test ./internal/helpers/ -run TestExtractFilename)
(cd backend && go test ./internal/models/ -run TestOldSyntax_BasicMovie)

# 运行测试并显示覆盖率
(cd backend && go test -cover ./...)
```

代码检查：

```bash
(cd backend && go vet ./...)
(cd backend && goimports -local Q115-STRM -w .)
```

后端没有配置 Go lint 工具（无 `.golangci.yml`、无 Makefile）。Go 格式化统一使用 `goimports -local Q115-STRM -w .`，不要手工维护 import 分组。

前端常用验证：

```bash
(cd frontend && pnpm lint)
(cd frontend && pnpm format:check)
(cd frontend && pnpm run type-check)
(cd frontend && pnpm run build)
```

前端测试约束：

- 前端测试文件统一放在 `frontend/test/` 下按类型归类，例如 `components/`、`composables/`、`utils/`、`unit/`、`e2e/`、`regression/`。
- `frontend/test/` 是本地测试目录，保持被 `.gitignore` 忽略，不提交测试文件。
- 不使用 `package.json` 的 `test:unit` 脚本；需要跑前端测试时直接用裸命令。
- 不要直接对整个 `frontend/test` 跑 Vitest；该目录里可能包含 Playwright、源码检查脚本或其他非 Vitest suite。

常用 Vitest 命令：

```bash
(cd frontend && pnpm exec vitest run test/components test/composables test/utils test/unit/const.test.ts)
(cd frontend && pnpm exec vitest run test/components/AppLogin.test.ts)
```

Go 测试约束：

- 使用 table-driven 测试模式。
- 测试名称应清楚描述场景；业务场景优先使用中文，既有英文测试名保持不变。
- 测试文件位于对应包目录下，例如 `internal/helpers/extract_test.go`。

## 项目结构

```text
backend/
  main.go                # 入口，初始化环境、数据库、路由、启动服务
  internal/              # Gin 控制器、模型、同步、刮削、通知等后端逻辑
  emby302/               # 嵌入的 Emby 302 代理子项目
  openxpanapi/           # 自动生成的百度网盘 OpenAPI 客户端
  web_statics/           # 前端生产构建产物
  FNOS/                  # 飞牛应用打包目录
frontend/                # Vue/Vite 前端源码
frontend/test/           # 前端本地测试目录（被忽略，不提交）
docker/                  # Dockerfile、容器入口脚本和在线更新监视脚本
scripts/release/         # GitHub Actions 发布打包辅助脚本
scripts/install/         # Linux 裸机安装辅助脚本
.github/workflows/       # CI/CD 工作流
```

## Go 代码约束

### 导入和格式化

- Go 文件使用 `goimports` 维护 import，不要手工调整 import 分组。
- `-local Q115-STRM` 用于识别本项目包。
- 由于模块名 `Q115-STRM` 不是域名式路径，导入分组以 `goimports -local Q115-STRM` 的实际输出为准。

常见输出为标准库、本项目包、第三方包三组：

```go
import (
    "context"
    "fmt"
    "net/http"

    "Q115-STRM/internal/db"
    "Q115-STRM/internal/helpers"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)
```

### 命名

- 新增标识符优先遵循 Go 惯用命名；存量命名不做无关批量重命名。
- 导出类型 / 函数使用 `PascalCase`，例如 `SyncPath`、`GetSyncPathByID`、`CreateSyncPath`。
- 未导出字段 / 函数使用 `camelCase`，例如 `isRelease`、`httpServer`、`loadYaml`。
- 常量导出用 `PascalCase`，未导出用 `camelCase`。
- 常见 initialism 使用 Go 约定的大写形式，例如 `ID`、`URL`、`API`、`HTTP`、`JSON`、`SQL`、`OAuth`。
- 修改已有接口、模型、配置字段、JSON 字段或跨包导出符号时，优先保持兼容；可以顺便询问用户是否需要按命名规范调整，但只有在用户明确要求时才改动公共 API。
- 接口按行为或职责命名，例如 `Reader`、`Writer`、`EventHandler`、`Scraper`、`Driver`；不强制使用 `Impl` 后缀。
- 布尔字段优先使用自然状态名，例如 `IsRunning`、`HasRemoteSeasonPath`、`CronEnabled`；已有 `EnableXxx` 命名可继续保持。
- 包级变量按可见性命名：跨包需要访问时使用 `PascalCase`，仅包内使用时使用 `camelCase`。
- 所有 Go 标识符使用英文，中文仅出现在注释和字符串字面量中。

### 注释

- 注释使用中文，这是项目的主要约定。
- 导出函数应有中文文档注释，例如 `// 修改同步路径`。
- Swagger 注解使用英文标签 + 中文描述。
- 结构体字段注释使用中文，例如 `// 是否自定义配置`。
- 不写无意义注释。

### 错误处理和控制器响应

常见错误处理模式：

```go
if err := tx.Where("id = ?", id).Delete(&SyncPath{}).Error; err != nil {
    tx.Rollback()
    return err
}
```

```go
if result.Error != nil {
    helpers.AppLogger.Errorf("创建同步路径失败: %v", result.Error)
    return nil
}
```

`panic` 仅用于不可恢复的启动错误：

```go
panic("创建数据目录失败")
```

控制器中的错误响应以现有控制器模式为准。多数业务接口使用 `APIResponse` 包装响应，并通过 `Code` 字段表达业务成功 / 失败；但当前代码并不是所有错误都返回 HTTP 200。认证、参数校验、文件 / 日志、部分历史接口会直接使用 `http.StatusBadRequest`、`http.StatusUnauthorized`、`http.StatusNotFound`、`http.StatusConflict` 或 `gin.H`。

修改现有接口时，先保持所在控制器的既有响应模式。常见 `APIResponse` 写法：

```go
c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
```

### 数据模型

所有表嵌入基础模型：

```go
type BaseModel struct {
    ID        uint  `gorm:"primaryKey" json:"id"`
    CreatedAt int64 `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt int64 `gorm:"autoUpdateTime" json:"updated_at"`
}
```

- 时间戳使用 `int64`（Unix 秒），不使用 `time.Time`。
- JSON 标签使用 `snake_case`。
- GORM 标签指定约束：`gorm:"primaryKey"`、`gorm:"unique"`、`gorm:"index"`、`gorm:"type:text;size:512"`。
- 数组字段使用双存储模式（JSON 切片 + JSON 字符串）：

  ```go
  VideoExt    []string `json:"video_ext_arr" gorm:"-"`
  VideoExtStr string   `json:"-"`
  ```

- 枚举类型使用 `string` 类型 + 常量：

  ```go
  type SourceType string

  const (
      SourceType115      SourceType = "115"
      SourceTypeBaiduPan SourceType = "baidupan"
  )
  ```

- 非标准表名使用 `TableName()` 方法。

### API 响应和绑定

多数业务接口使用泛型响应类型，通过 `Code` 字段区分成功 / 失败：

```go
type APIResponse[T any] struct {
    Code    APIResponseCode `json:"code"`
    Message string          `json:"message"`
    Data    T               `json:"data"`
}
```

请求参数绑定：

- 查询参数：`c.ShouldBindQuery(&req)`，struct 标签使用 `form:"field"`。
- JSON Body：`c.ShouldBindJSON(&req)`，struct 标签使用 `json:"field"`。
- 路径参数：`c.Param("id")`。

## 配置和日志

- YAML 配置文件：`config/config.yaml`（通过 `helpers.InitConfig()` 加载，兼容旧 `config.yml`）。
- 敏感信息：`config/.env`（通过 `helpers.LoadEnvFromFile()` 加载，会覆盖真实环境变量）。
- 默认 API 密钥：`FANART_API_KEY` / `TMDB_API_KEY` / `TMDB_ACCESS_TOKEN` / `SC_API_KEY`，可编译期 ldflags（`-X main.<名>`）注入或运行时环境变量设置（无 `DEFAULT_` 前缀）。
- 默认 API 密钥取值优先级：UI 配置（DB）> 环境变量 / `config/.env` > ldflags；`config/.env` 会覆盖真实环境变量。
- 本机敏感数据密钥：由 `helpers.InitEncryptionKey()` 每实例自动生成并保存到 `config/encryption.key`，不使用 `ENCRYPTION_KEY` 环境变量，也不走 ldflags。
- OAuth 中转共享密钥：`OAUTH_RELAY_ENCRYPTION_KEY`，可通过环境变量 / `config/.env` 设置，或通过 ldflags 变量 `main.OAuthRelayEncryptionKey` 注入；环境变量优先于 ldflags。
- 全局配置对象：`helpers.GlobalConfig`。

使用自定义 `QLogger`（封装 `log.Logger` + lumberjack 日志轮转）：

```go
helpers.AppLogger.Infof("操作成功: %s", name)
helpers.AppLogger.Errorf("操作失败: %v", err)
helpers.V115Log.Debugf("115 请求详情: %s", url)
```

日志级别：`Infof`、`Warnf`、`Errorf`、`Debugf`、`Fatalf`。

按子系统使用不同 logger：`AppLogger`（主日志）、`V115Log`、`OpenListLog`、`BaiduPanLog`、`TMDBLog`。

## 前端约束

- Vue 代码使用 Vue 3 Composition API 和 `<script setup lang="ts">`。
- 组合式函数接收 maybe-reactive 参数时，普通值（如 `pageSize`）可以使用 `MaybeRefOrGetter` + `toValue()`。
- axios 实例、回调函数、SDK client、handler 等“值本身可能是函数或可调用对象”的参数，不要使用 `MaybeRefOrGetter` + `toValue()`；应使用 `MaybeRef` + `unref()`，避免把可调用对象当 getter 执行。

## 文档约束

- README 只保留项目介绍、快速入口和 `docs/README.md` 索引入口；完整文档索引统一维护在 `docs/README.md`。
- 使用者、开发、配置、发布类说明放入 `docs/`。
- 不要在正式文档索引中引用被忽略的本地工作目录。
- 中文文档遵循中英文、中文与数字之间加空格的排版习惯。

## CI/CD

- `feature/**` 分支推送触发构建，生成 GHCR 镜像 `ghcr.io/<owner>/qmediasync:<branch-name>`。
- `dev` 分支推送触发构建，生成 GHCR 镜像 `ghcr.io/<owner>/qmediasync:beta`。
- 正式发布通过推送 `v*` 标签或手动触发 `.github/workflows/release.yaml` 执行。
- 变更日志使用 `git-cliff` 从 conventional commits 生成，发布脚本会更新 `CHANGELOG.md` 并写入 `.changes/<tag>.md`。
