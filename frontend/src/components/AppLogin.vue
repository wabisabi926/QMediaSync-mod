<script setup lang="ts">
import { onMounted, shallowRef } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useHttpClient } from '@/http/client'
import { SERVER_URL } from '@/const'
import LoginForm, { type LoginSubmitPayload } from '@/components/auth/LoginForm.vue'

const router = useRouter()
const authStore = useAuthStore()
const http = useHttpClient()
const loading = shallowRef(false)

const getErrorMessage = (error: unknown, fallback: string) => {
  if (error instanceof Error && error.message) {
    return error.message
  }
  if (error && typeof error === 'object' && 'response' in error) {
    const response = (error as { response?: { data?: { message?: string } } }).response
    if (response?.data?.message) {
      return response.data.message
    }
  }
  return fallback
}

const handleLogin = async (payload: LoginSubmitPayload) => {
  if (loading.value || !http) return

  try {
    loading.value = true
    const response = await http.post(
      `${SERVER_URL}/login`,
      {
        username: payload.username,
        password: payload.password,
        totp_code: payload.totp_code,
        rememberMe: payload.rememberMe,
      },
      {
        headers: {
          'Content-Type': 'application/json',
        },
        skipAuthInvalidation: true,
      },
    )

    if (response?.data.code === 200) {
      const sessionResult = await authStore.refreshSession(http)
      if (sessionResult.state === 'anonymous') {
        ElMessage.error(
          '登录会话未能建立，请允许本站 Cookie 后重试；若问题持续，请清除本站点数据或停用拦截扩展',
        )
        return
      }
      if (sessionResult.state === 'unavailable') {
        ElMessage.error('登录会话验证失败，请检查网络连接或稍后重试')
        return
      }

      ElMessage.success('登录成功')

      const redirect = router.currentRoute.value.query.redirect as string
      router.replace(redirect || '/')
    } else {
      ElMessage.error(response?.data.message || '登录失败')
    }
  } catch (error: unknown) {
    console.error('登录错误：', error)
    ElMessage.error(getErrorMessage(error, '登录失败，请检查网络连接'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (authStore.isAuthenticated) {
    router.replace('/')
    return
  }
})
</script>

<template>
  <div class="login-container">
    <div class="login-box">
      <div class="login-header">
        <h1 class="login-title">QMediaSync</h1>
        <p class="login-subtitle">系统登录</p>
      </div>

      <LoginForm :loading="loading" @submit="handleLogin" />
    </div>
  </div>
</template>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.login-box {
  width: 100%;
  max-width: 500px;
  background: white;
  border-radius: 12px;
  padding: 40px 30px;
  box-shadow: 0 15px 35px rgba(0, 0, 0, 0.1);
}

.login-header {
  text-align: center;
  margin-bottom: 30px;
}

.login-title {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 8px 0;
}

.login-subtitle {
  font-size: 16px;
  color: #909399;
  margin: 0;
}

@media (max-width: 768px) {
  .login-container {
    padding: 15px;
  }

  .login-box {
    max-width: 100%;
    padding: 30px 20px;
    border-radius: 8px;
  }

  .login-title {
    font-size: 24px;
  }

  .login-subtitle {
    font-size: 14px;
  }
}

@media (max-width: 480px) {
  .login-container {
    padding: 10px;
  }

  .login-box {
    max-width: 100%;
    padding: 25px 15px;
  }

  .login-title {
    font-size: 22px;
  }
}
</style>