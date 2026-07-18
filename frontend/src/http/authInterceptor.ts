import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from 'axios'

const authenticationExpiredMessage = '登录已失效，请重新登录'

export type AuthInvalidationStore = {
  isAuthenticated: boolean
  isLoggingOut: boolean
  clearAuth: () => void
}

type AuthResponseInterceptorOptions = {
  getAuthStore: () => AuthInvalidationStore
  onAuthenticationInvalidated: () => void | Promise<void>
}

const isAuthenticationFailure = (status?: number, data?: unknown) => {
  if (status === 401) return true
  if (!data || typeof data !== 'object' || !('code' in data)) return false
  return data.code === 401
}

const canHandleAuthenticationFailure = (
  config: InternalAxiosRequestConfig | undefined,
  authStore: AuthInvalidationStore,
) => {
  return !config?.skipAuthInvalidation && authStore.isAuthenticated && !authStore.isLoggingOut
}

export const installAuthResponseInterceptor = (
  http: AxiosInstance,
  options: AuthResponseInterceptorOptions,
) => {
  let invalidationPromise: Promise<void> | null = null

  const handleAuthenticationFailure = async (config?: InternalAxiosRequestConfig) => {
    const authStore = options.getAuthStore()
    if (!canHandleAuthenticationFailure(config, authStore)) return false

    if (!invalidationPromise) {
      authStore.clearAuth()
      invalidationPromise = Promise.resolve(options.onAuthenticationInvalidated()).finally(() => {
        invalidationPromise = null
      })
    }

    await invalidationPromise
    return true
  }

  const interceptorID = http.interceptors.response.use(
    async (response: AxiosResponse) => {
      if (!isAuthenticationFailure(response.status, response.data)) return response

      if (await handleAuthenticationFailure(response.config)) {
        return Promise.reject(new Error(authenticationExpiredMessage))
      }

      return response
    },
    async (error) => {
      if (isAuthenticationFailure(error.response?.status, error.response?.data)) {
        await handleAuthenticationFailure(error.config)
      }
      return Promise.reject(error)
    },
  )

  return () => {
    http.interceptors.response.eject(interceptorID)
  }
}
