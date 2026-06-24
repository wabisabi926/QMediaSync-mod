# 开发调试

## 后端启动

```bash
cd backend
go run .
```

## 前端启动

```bash
cd frontend
corepack enable
corepack prepare pnpm@11 --activate
pnpm install
pnpm run dev
```

前端开发环境默认连接 `http://localhost:12333/api`。

## 退出

- Linux：按 `Ctrl+C` 退出。
- Windows：在系统托盘找到 QMediaSync 图标，右键退出。
