import axios, { AxiosError, type AxiosAdapter, type InternalAxiosRequestConfig } from 'axios'
import { describe, expect, it, vi } from 'vitest'

import {
  installAuthResponseInterceptor,
  type AuthInvalidationStore,
} from '../../src/http/authInterceptor'

type AuthStoreStub = AuthInvalidationStore & {
  clearAuth: ReturnType<typeof vi.fn<() => void>>
}

const unauthorizedResponse = (config: InternalAxiosRequestConfig) => ({
  config,
  data: { code: 401 },
  headers: {},
  status: 401,
  statusText: 'Unauthorized',
})

const unauthorizedAdapter: AxiosAdapter = async (config) => {
  if (config.url === '/body-code-401') {
    return {
      ...unauthorizedResponse(config),
      status: 200,
      statusText: 'OK',
    }
  }

  return Promise.reject(
    new AxiosError(
      'Request failed with status code 401',
      'ERR_BAD_REQUEST',
      config,
      undefined,
      unauthorizedResponse(config),
    ),
  )
}

const createAuthenticatedStore = (): AuthStoreStub => {
  const store: AuthStoreStub = {
    isAuthenticated: true,
    isLoggingOut: false,
    clearAuth: vi.fn<() => void>(),
  }
  store.clearAuth.mockImplementation(() => {
    store.isAuthenticated = false
  })
  return store
}

describe('auth response interceptor', () => {
  it('并发业务 401 只清理、提示和跳转一次', async () => {
    const http = axios.create({ adapter: unauthorizedAdapter })
    const store = createAuthenticatedStore()
    const notify = vi.fn()
    const replace = vi.fn().mockResolvedValue(undefined)
    const uninstall = installAuthResponseInterceptor(http, {
      getAuthStore: () => store,
      onAuthenticationInvalidated: async () => {
        notify('登录已失效，请重新登录')
        await replace('/login')
      },
    })

    await Promise.allSettled([http.get('/protected-a'), http.get('/protected-b')])

    expect(store.clearAuth).toHaveBeenCalledTimes(1)
    expect(notify).toHaveBeenCalledWith('登录已失效，请重新登录')
    expect(replace).toHaveBeenCalledTimes(1)
    expect(replace).toHaveBeenCalledWith('/login')
    uninstall()
  })

  it('响应体 code 为 401 时也只处理一次认证失效', async () => {
    const http = axios.create({ adapter: unauthorizedAdapter })
    const store = createAuthenticatedStore()
    const notify = vi.fn()
    const replace = vi.fn().mockResolvedValue(undefined)
    const uninstall = installAuthResponseInterceptor(http, {
      getAuthStore: () => store,
      onAuthenticationInvalidated: async () => {
        notify('登录已失效，请重新登录')
        await replace('/login')
      },
    })

    await expect(http.get('/body-code-401')).rejects.toThrow('登录已失效，请重新登录')

    expect(store.clearAuth).toHaveBeenCalledTimes(1)
    expect(notify).toHaveBeenCalledTimes(1)
    expect(replace).toHaveBeenCalledWith('/login')
    uninstall()
  })

  it('跳过认证流程请求和匿名状态不触发认证失效处理', async () => {
    const http = axios.create({ adapter: unauthorizedAdapter })
    const store = createAuthenticatedStore()
    const notify = vi.fn()
    const replace = vi.fn().mockResolvedValue(undefined)
    const uninstall = installAuthResponseInterceptor(http, {
      getAuthStore: () => store,
      onAuthenticationInvalidated: async () => {
        notify('登录已失效，请重新登录')
        await replace('/login')
      },
    })

    await Promise.allSettled([
      http.get('/session', { skipAuthInvalidation: true }),
      (() => {
        store.isAuthenticated = false
        return http.get('/anonymous-request')
      })(),
    ])

    expect(store.clearAuth).not.toHaveBeenCalled()
    expect(notify).not.toHaveBeenCalled()
    expect(replace).not.toHaveBeenCalled()
    uninstall()
  })

  it('认证失效处理不会请求服务端登出', async () => {
    const requestedURLs: string[] = []
    const http = axios.create({
      adapter: async (config) => {
        requestedURLs.push(config.url || '')
        return Promise.reject(
          new AxiosError(
            'Request failed with status code 401',
            'ERR_BAD_REQUEST',
            config,
            undefined,
            unauthorizedResponse(config),
          ),
        )
      },
    })
    const store = createAuthenticatedStore()
    const uninstall = installAuthResponseInterceptor(http, {
      getAuthStore: () => store,
      onAuthenticationInvalidated: vi.fn(),
    })

    await expect(http.get('/protected')).rejects.toThrow('Request failed with status code 401')

    expect(requestedURLs).toEqual(['/protected'])
    uninstall()
  })
})
