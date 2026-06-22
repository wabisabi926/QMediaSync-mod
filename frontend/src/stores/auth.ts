import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface User {
  id: string
  username: string
  email?: string
  role?: string
}

export const useAuthStore = defineStore('auth', () => {
  // 状态
  const token = ref<string | null>(null)
  const user = ref<User | null>(null)
  const isLoading = ref(false)
  const isLoggingOut = ref(false) // 防止重复登出
  const hasInitialized = ref(false)

  // 计算属性
  const isAuthenticated = computed(() => !!token.value)

  // 从localStorage恢复登录状态
  const initAuth = () => {
    if (hasInitialized.value) return
    const savedToken = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')
    const savedUser = localStorage.getItem('auth_user') || sessionStorage.getItem('auth_user')

    if (savedToken && savedUser) {
      token.value = savedToken
      try {
        user.value = JSON.parse(savedUser)
      } catch (error) {
        console.error('解析用户信息失败:', error)
        clearAuth()
      }
    }
    hasInitialized.value = true
  }

  const restoreSession = async (http: AxiosStatic) => {
    initAuth()
    if (token.value) return true
    try {
      const response = await http.get(`${SERVER_URL}/session`, { withCredentials: true })
      if (response.data?.code === 200 && response.data.data?.token && response.data.data?.user) {
        login(response.data.data.token, response.data.data.user, false)
        return true
      }
    } catch (error) {
      console.error('恢复登录会话失败:', error)
    }
    clearAuth()
    return false
  }

  // 登录
  const login = (authToken: string, userData: User, rememberMe: boolean = false) => {
    token.value = authToken
    user.value = userData
    const jsonUser = JSON.stringify(userData)
    const storage = rememberMe ? localStorage : sessionStorage
    storage.setItem('auth_token', authToken)
    storage.setItem('auth_user', jsonUser)

    // 如果选择记住我，清除sessionStorage中的数据
    if (rememberMe) {
      sessionStorage.removeItem('auth_token')
      sessionStorage.removeItem('auth_user')
    } else {
      // 如果没选择记住我，清除localStorage中的数据
      localStorage.removeItem('auth_token')
      localStorage.removeItem('auth_user')
    }
  }

  // 登出
  const logout = () => {
    if (isLoggingOut.value) return // 防止重复登出
    isLoggingOut.value = true
    clearAuth()
    setTimeout(() => {
      isLoggingOut.value = false
    }, 1000) // 1秒后重置标志
  }

  const logoutWithServer = async (http: AxiosStatic) => {
    if (isLoggingOut.value) return
    isLoggingOut.value = true
    try {
      await http.post(`${SERVER_URL}/logout`, undefined, { withCredentials: true })
    } catch (error) {
      console.error('服务端退出登录失败:', error)
    } finally {
      clearAuth()
      setTimeout(() => {
        isLoggingOut.value = false
      }, 1000)
    }
  }

  // 清除认证信息
  const clearAuth = () => {
    token.value = null
    user.value = null

    // 清除所有存储的认证信息
    localStorage.removeItem('auth_token')
    localStorage.removeItem('auth_user')
    sessionStorage.removeItem('auth_token')
    sessionStorage.removeItem('auth_user')
  }

  // 更新用户信息
  const updateUser = (userData: Partial<User>) => {
    if (user.value) {
      user.value = { ...user.value, ...userData }

      // 更新存储中的用户信息
      const storage = localStorage.getItem('auth_token') ? localStorage : sessionStorage
      storage.setItem('auth_user', JSON.stringify(user.value))
    }
  }

  // 检查token是否有效
  const checkTokenValidity = async () => {
    if (!token.value) return false

    try {
      // 这里可以调用API验证token有效性
      // const response = await api.validateToken(token.value)
      // return response.valid

      // 临时返回true，实际项目中应该调用API验证
      return true
    } catch (error) {
      console.error('Token验证失败:', error)
      clearAuth()
      return false
    }
  }

  return {
    // 状态
    token,
    user,
    isLoading,
    isLoggingOut,
    hasInitialized,

    // 计算属性
    isAuthenticated,

    // 方法
    initAuth,
    restoreSession,
    login,
    logout,
    logoutWithServer,
    clearAuth,
    updateUser,
    checkTokenValidity,
  }
})
