import 'axios'

declare module 'axios' {
  interface AxiosRequestConfig {
    skipAuthInvalidation?: boolean
  }
}
