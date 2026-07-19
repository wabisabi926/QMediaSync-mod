// @vitest-environment happy-dom
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAuthStore } from '../../src/stores/auth'

describe('cookie-only auth store', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    setActivePinia(createPinia())
  })

  it('登录状态不写入 Web Storage token', () => {
    const store = useAuthStore()

    store.login({
      user: { id: '1', username: 'admin', role: 'admin' },
      csrfToken: 'csrf-token',
      session: { session_id: 'sid', expires_at: 1 },
    })

    expect(store.isAuthenticated).toBe(true)
    expect(store.csrfToken).toBe('csrf-token')
    expect(localStorage.getItem('auth_token')).toBeNull()
    expect(sessionStorage.getItem('auth_token')).toBeNull()
  })

  it('通过 /session 恢复服务端会话', async () => {
    const store = useAuthStore()
    const http = {
      get: vi.fn().mockResolvedValue({
        data: {
          code: 200,
          data: {
            authenticated: true,
            user: { id: '1', username: 'admin', role: 'admin' },
            csrf_token: 'csrf-token',
            session: { session_id: 'sid', expires_at: 1 },
          },
        },
      }),
    }

    await store.bootstrapAuth(http as never)

    expect(http.get).toHaveBeenCalledWith(expect.stringContaining('/session'), {
      skipAuthInvalidation: true,
      withCredentials: true,
    })
    expect(store.authStatus).toBe('authenticated')
    expect(store.csrfToken).toBe('csrf-token')
  })
})
