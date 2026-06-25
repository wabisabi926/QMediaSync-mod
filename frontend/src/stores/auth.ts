import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { defineStore } from 'pinia'
import { computed, shallowRef } from 'vue'

export interface User {
  id: string
  username: string
  email?: string
  role?: string
}

export interface UserSession {
  session_id: string
  current?: boolean
  ip_address?: string
  user_agent?: string
  created_at?: number
  last_seen_at?: number
  expires_at: number
}

export interface LoginPayload {
  user: User
  csrfToken: string
  session?: UserSession
}

type AuthStatus = 'checking' | 'authenticated' | 'anonymous'

type SessionResponseData = {
  user?: User
  csrf_token?: string
  session?: UserSession
}

const clearLegacyWebStorage = () => {
  localStorage.removeItem('auth_token')
  localStorage.removeItem('auth_user')
  sessionStorage.removeItem('auth_token')
  sessionStorage.removeItem('auth_user')
}

export const useAuthStore = defineStore('auth', () => {
  const user = shallowRef<User | null>(null)
  const session = shallowRef<UserSession | null>(null)
  const csrfToken = shallowRef<string | null>(null)
  const authStatus = shallowRef<AuthStatus>('checking')
  const isLoading = shallowRef(false)
  const isLoggingOut = shallowRef(false)
  const hasInitialized = shallowRef(false)
  let bootstrapPromise: Promise<boolean> | null = null

  const isAuthenticated = computed(() => authStatus.value === 'authenticated' && !!user.value)

  const applySession = (payload: LoginPayload) => {
    user.value = payload.user
    session.value = payload.session || null
    csrfToken.value = payload.csrfToken
    authStatus.value = 'authenticated'
    hasInitialized.value = true
    clearLegacyWebStorage()
  }

  const clearAuth = () => {
    user.value = null
    session.value = null
    csrfToken.value = null
    authStatus.value = 'anonymous'
    hasInitialized.value = true
    bootstrapPromise = null
    clearLegacyWebStorage()
  }

  const applySessionResponse = (data?: SessionResponseData) => {
    if (!data?.user || !data.csrf_token) return false
    applySession({
      user: data.user,
      csrfToken: data.csrf_token,
      session: data.session,
    })
    return true
  }

  const refreshSession = async (http: AxiosStatic) => {
    authStatus.value = 'checking'
    try {
      const response = await http.get(`${SERVER_URL}/session`, { withCredentials: true })
      if (response.data?.code === 200 && applySessionResponse(response.data.data)) {
        return true
      }
    } catch (error) {
      console.error('恢复登录会话失败：', error)
    }
    clearAuth()
    return false
  }

  const bootstrapAuth = async (http: AxiosStatic) => {
    if (bootstrapPromise) return bootstrapPromise

    bootstrapPromise = (async () => {
      const ok = await refreshSession(http)
      bootstrapPromise = null
      return ok
    })()

    return bootstrapPromise
  }

  const restoreSession = async (http: AxiosStatic) => {
    return bootstrapAuth(http)
  }

  const initAuth = () => {
    if (hasInitialized.value) return
    clearAuth()
  }

  const login = (payload: LoginPayload) => {
    applySession(payload)
  }

  const logout = () => {
    if (isLoggingOut.value) return
    isLoggingOut.value = true
    clearAuth()
    setTimeout(() => {
      isLoggingOut.value = false
    }, 1000)
  }

  const logoutWithServer = async (http: AxiosStatic) => {
    if (isLoggingOut.value) return
    isLoggingOut.value = true
    try {
      await http.post(`${SERVER_URL}/logout`, undefined, { withCredentials: true })
    } catch (error) {
      console.error('服务端退出登录失败：', error)
    } finally {
      clearAuth()
      setTimeout(() => {
        isLoggingOut.value = false
      }, 1000)
    }
  }

  const updateUser = (userData: Partial<User>) => {
    if (user.value) {
      user.value = { ...user.value, ...userData }
    }
  }

  return {
    user,
    session,
    csrfToken,
    authStatus,
    isLoading,
    isLoggingOut,
    hasInitialized,
    isAuthenticated,
    initAuth,
    bootstrapAuth,
    refreshSession,
    restoreSession,
    login,
    logout,
    logoutWithServer,
    clearAuth,
    updateUser,
  }
})
