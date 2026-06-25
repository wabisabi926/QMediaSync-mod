import './assets/main.css'

import axios from 'axios'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { ElMessage } from 'element-plus'
import 'element-plus/theme-chalk/display.css'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/message-box/style/css'
import App from './App.vue'

import router from './router/index'
import { useAuthStore } from '@/stores/auth'
import { SERVER_URL } from '@/const'
import { getCSRFTokenFromCookie, shouldAttachCSRFToken } from '@/utils/csrf'

// 配置 axios
axios.defaults.timeout = 10000
axios.defaults.headers.common['Content-Type'] = 'application/json'
axios.defaults.withCredentials = true

// 请求拦截器
axios.interceptors.request.use(
  (config) => {
    const headers = config.headers as Record<string, string>
    delete headers.Authorization

    const authStore = useAuthStore()
    if (shouldAttachCSRFToken(config.method)) {
      const csrfToken = authStore.csrfToken || getCSRFTokenFromCookie()
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken
      }
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  },
)

// 响应拦截器
axios.interceptors.response.use(
  (response) => {
    // 检查响应数据中的 code 字段
    if (response.data && response.data.code === 401) {
      const authStore = useAuthStore()
      if (!authStore.isLoggingOut) {
        ElMessage.error('登录已失效，请重新登录')
        void authStore.logoutWithServer(axios)
        void router.push('/login')
      }
      return Promise.reject(new Error('登录已失效，请重新登录'))
    }
    return response
  },
  (error) => {
    const authStore = useAuthStore()
    // 检查 HTTP 状态码 401
    if (error.response?.status === 401) {
      if (!authStore.isLoggingOut) {
        ElMessage.error('登录已失效，请重新登录')
        void authStore.logoutWithServer(axios)
        void router.push('/login')
      }
    }
    // 检查响应数据中的 code 字段
    else if (error.response?.data?.code === 401) {
      if (!authStore.isLoggingOut) {
        ElMessage.error('登录已失效，请重新登录')
        void authStore.logoutWithServer(axios)
        void router.push('/login')
      }
    }
    return Promise.reject(error)
  },
)

const pinia = createPinia()
const app = createApp(App)

app.use(pinia)

const bootstrap = async () => {
  const authStore = useAuthStore()
  await authStore.bootstrapAuth(axios)

  app.use(router)
  app.provide('$http', axios)
  app.provide('SERVER_URL', SERVER_URL)

  await router.isReady()
  app.mount('#app')
}

void bootstrap()
