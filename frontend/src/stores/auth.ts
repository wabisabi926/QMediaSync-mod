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

  // 计算属性
  const isAuthenticated = computed(() => !!token.value)

  // 从localStorage恢复登录状态
  const initAuth = () => {
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
  }

  // 登录
  const login = (authToken: string, userData: User, rememberMe: boolean = false) => {
    token.value = authToken
    user.value = userData
    console.log('登录成功1:', authToken, userData)
    const jsonUser = JSON.stringify(userData)
    console.log('登录成功2:', jsonUser)
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
    clearAuth()
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

    // 计算属性
    isAuthenticated,

    // 方法
    initAuth,
    login,
    logout,
    clearAuth,
    updateUser,
    checkTokenValidity,
  }
})
