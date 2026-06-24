# 开发调试

## 后端启动

```bash
cd backend
go run .
```

如果没有 `config/config.yaml`，后端会先启动配置向导，默认监听 `http://127.0.0.1:12333`。配置保存后需要重新启动后端。

## 前端启动

```bash
cd frontend
corepack enable
corepack prepare pnpm@11 --activate
pnpm install
pnpm run dev
```

前端开发环境默认连接 `http://localhost:12333/api`。

## 常用验证

```bash
(cd backend && go test ./...)
(cd frontend && pnpm lint)
(cd frontend && pnpm format:check)
(cd frontend && pnpm run type-check)
(cd frontend && pnpm run build)
```

## 退出

- Linux：按 `Ctrl+C` 退出。
- Windows：在系统托盘找到 QMediaSync 图标，右键退出。
