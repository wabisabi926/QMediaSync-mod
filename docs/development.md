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

后端默认允许 Vite 开发来源 `http://localhost:5173`、`http://127.0.0.1:5173` 和 `http://[::1]:5173` 携带浏览器登录 Cookie。若修改 Vite 端口、使用自定义开发域名，或让前端跨源访问后端，需要把前端来源加入 `config/config.yaml`：

```yaml
trustedOrigins:
  - http://dev.example.local:5173
```

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
