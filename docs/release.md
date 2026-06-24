# 发布流程

更新日志由 [git-cliff](https://git-cliff.org/) 从 git 提交记录自动生成，因此提交信息需遵循 [Conventional Commits](https://www.conventionalcommits.org/)。

当前 `cliff.toml` 只把 `feat:`、`fix:`、`perf:`、`revert:` 写入 changelog；`docs:`、`chore:`、`ci:`、`test:` 等开发类提交不会进入发布说明，除非调整 `cliff.toml`。不规范的提交（如 `Merge`、自由文案）会被自动忽略。

## 本地发版

发版步骤推荐使用脚本：

1. 确认当前工作区干净，并确认 `dev` 是准备发布的内容。
2. 执行发布脚本（需先安装 git-cliff，见其 [安装文档](https://git-cliff.org/docs/installation/)）：

   ```bash
   scripts/release/release.sh v0.xx.xx
   ```

也可以用 `patch`、`minor`、`major` 根据当前最新版本自动推导下一个版本：

```bash
scripts/release/release.sh patch
scripts/release/release.sh minor
scripts/release/release.sh major
```

## 发布脚本行为

该脚本会：

- 校验 tag 格式必须为 `v<major>.<minor>.<patch>`，例如 `v0.15.3`。
- 校验 tag 必须大于当前最新版本；如果 minor 或 major 版本增加，还会分别要求输入 `minor yes` 或 `major yes` 额外确认。
- 同步 `main`，并把本地 `dev` 快进合入 `main`。
- 读取上一个 `v*` 标签至今的提交，按类型分组生成 `.changes/v0.xx.xx.md`，作为 GitHub Release 正文。
- 把本版本段落插入 `CHANGELOG.md` 顶部，保留历史内容。
- 拒绝重复版本：如果本地已存在同名 git tag、`.changes/<tag>.md`，或 `CHANGELOG.md` 已包含该版本段落，命令会直接失败。
- 展示 `CHANGELOG.md` 和 `.changes/<tag>.md` 的 diff，等待输入 `yes` 确认。
- 在 `main` 上提交 `chore: release <tag>`。
- 创建 annotated tag：`git tag -a <tag> -m "Release <tag>"`。
- 推送 `main` 和 tag 触发 release workflow。
- 将 release commit 快进同步回 `dev` 并推送 `dev`。

## GitHub Actions 发布

推送 `v*` 标签会触发 GitHub Actions 的 release 流程，生成 Windows/Linux 发布包、可选的飞牛 FPK，并创建 GitHub Release。

GitHub Release 的正文取自上一步提交的 `.changes/v0.xx.xx.md`；release workflow 会拒绝重复 GitHub Release 和缺失 `.changes/<tag>.md` 的发布。

发布流程还会使用 `GITHUB_TOKEN` 推送 GHCR 镜像 `ghcr.io/<owner>/qmediasync:<tag>` 和 `ghcr.io/<owner>/qmediasync:latest`。

也可以在 GitHub Actions 中手动触发 `release` workflow，并输入要发布的 Git tag（同样要求该 tag 对应的 `.changes/<tag>.md` 已提交）。

## 飞牛 FPK

飞牛 FPK 打包依赖飞牛官方工具 `fnpack`（不公开分发）。release workflow 通过仓库 Secret `FNPACK_DOWNLOAD_URL`（指向可下载 `fnpack` 可执行文件的地址）下载安装，再用 `backend/FNOS/` 下的素材执行 `fnpack build` 生成 `*.fpk`。

未配置该 Secret 时，`fpk` job 和 `scripts/release/package-fnos.sh` 会自动跳过 FPK 打包，其余 Windows/Linux 发布包、Docker 镜像不受影响；若希望缺少工具时直接报错（而非静默跳过），可在脚本环境设置 `REQUIRE_FNPACK=1`。

调整 changelog 的分组、过滤规则可编辑仓库根目录的 `cliff.toml`。
