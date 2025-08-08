/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_SERVER_URL: string
  // 更多环境变量可以在这里添加
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
