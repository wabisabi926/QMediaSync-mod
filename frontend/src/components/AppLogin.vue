<template>
  <div class="login-container">
    <div class="login-box">
      <div class="login-header">
        <h1 class="login-title">QMediaSync</h1>
        <p class="login-subtitle">系统登录</p>
      </div>

      <el-form
        :model="loginForm"
        :rules="loginRules"
        ref="loginFormRef"
        class="login-form"
        @submit.prevent="handleLogin"
      >
        <el-form-item prop="username">
          <el-input
            v-model="loginForm.username"
            size="large"
            placeholder="请输入用户名"
            prefix-icon="User"
            :disabled="loading"
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="loginForm.password"
            type="password"
            size="large"
            placeholder="请输入密码"
            prefix-icon="Lock"
            show-password
            :disabled="loading"
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item>
          <el-checkbox v-model="loginForm.rememberMe" :disabled="loading"> 记住我 </el-checkbox>
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            class="login-button"
            :loading="loading"
            @click="handleLogin"
          >
            {{ loading ? '登录中...' : '登录' }}
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, inject, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import type { AxiosStatic } from 'axios'
import { SERVER_URL } from '@/const'

const router = useRouter()
const authStore = useAuthStore()
const http: AxiosStatic | undefined = inject('$http')

const loginFormRef = ref<FormInstance>()
const loading = ref(false)

const loginForm = reactive({
  username: '',
  password: '',
  rememberMe: false,
})

const loginRules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 20, message: '用户名长度在 2 到 20 个字符', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 20, message: '密码长度在 6 到 20 个字符', trigger: 'blur' },
  ],
}

const handleLogin = async () => {
  if (!loginFormRef.value) return

  try {
    const valid = await loginFormRef.value.validate()
    if (!valid) return

    loading.value = true

    // 创建FormData对象，使用标准form表单格式
    const formData = new FormData()
    formData.append('username', loginForm.username)
    formData.append('password', loginForm.password)

    const response = await http?.post(`${SERVER_URL}/login`, formData, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
    })

    if (response?.data.code === 200) {
      const { token, user } = response.data.data
      // 保存登录状态
      authStore.login(token, user, loginForm.rememberMe)

      ElMessage.success('登录成功')

      // 跳转到首页或原本要访问的页面
      const redirect = router.currentRoute.value.query.redirect as string
      router.push(redirect || '/')
    } else {
      ElMessage.error(response?.data.message || '登录失败')
    }
  } catch (error: unknown) {
    console.error('登录错误:', error)
    let errorMsg = '登录失败，请检查网络连接'

    if (error && typeof error === 'object' && 'response' in error) {
      const response = (error as { response?: { data?: { message?: string } } }).response
      if (response?.data?.message) {
        errorMsg = response.data.message
      }
    }

    ElMessage.error(errorMsg)
  } finally {
    loading.value = false
  }
}

// 检查是否已经登录
onMounted(() => {
  if (authStore.isAuthenticated) {
    router.push('/')
  }
})
</script>

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

.login-form {
  width: 100%;
}

.login-form .el-form-item {
  margin-bottom: 24px;
}

.login-button {
  width: 100%;
  height: 44px;
  font-size: 16px;
  font-weight: 500;
}

/* 移动端适配 */
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

  .login-form .el-form-item {
    margin-bottom: 20px;
  }

  .login-button {
    height: 40px;
    font-size: 15px;
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

  .login-button {
    height: 42px;
  }
}
</style>
