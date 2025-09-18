<template>
  <div class="main-content-container user-settings-container">
    <el-form
      :model="formData"
      :label-position="checkIsMobile ? 'top' : 'left'"
      :label-width="90"
      class="user-form"
    >
      <el-form-item label="用户名" prop="username">
        <el-input
          v-model="formData.username"
          placeholder="请输入新的管理员用户名"
          :disabled="loading"
          maxlength="50"
        />
        <div class="form-help">用户名长度至少3个字符，留空则不修改</div>
      </el-form-item>

      <el-form-item label="密码" prop="password">
        <el-input
          v-model="formData.password"
          placeholder="请输入管理员密码"
          type="password"
          :disabled="loading"
          show-password
          maxlength="100"
        />
        <div class="form-help">建议使用强密码，包含大小写字母、数字和特殊字符</div>
      </el-form-item>

      <el-form-item label="确认密码" prop="confirmPassword" required>
        <el-input
          v-model="formData.confirmPassword"
          placeholder="请再次输入密码"
          type="password"
          :disabled="loading"
          show-password
          maxlength="100"
        />
        <div class="form-help">请再次输入密码以确认</div>
      </el-form-item>

      <div class="form-actions">
        <el-button
          type="success"
          @click="saveSettings"
          :loading="loading"
          size="large"
          :icon="Check"
        >
          保存设置
        </el-button>
      </div>
    </el-form>

    <!-- 保存状态显示 -->
    <el-alert
      v-if="saveStatus"
      :title="saveStatus.title"
      :type="saveStatus.type"
      :description="saveStatus.description"
      :closable="false"
      show-icon
      class="save-status"
    />
    <div class="security-content">
      <div class="warning-section">
        <el-alert title="重要提醒" type="warning" :closable="false" show-icon>
          <template #default>
            修改用户名或密码后，您需要重新登录系统。请确保记住新的登录凭据。
          </template>
        </el-alert>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, inject, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Check } from '@element-plus/icons-vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { isMobile } from '@/utils/deviceUtils'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'
interface UserSettings {
  username: string
  password: string
  confirmPassword: string
}

interface SaveStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}
const authStore = useAuthStore()
const router = useRouter()
const checkIsMobile = ref(isMobile())
const http: AxiosStatic | undefined = inject('$http')
const loading = ref(false)
const saveStatus = ref<SaveStatus | null>(null)

const formData = reactive<UserSettings>({
  username: '',
  password: '',
  confirmPassword: '',
})

// 表单验证
const validateForm = (): boolean => {
  if (formData.username && formData.username.length < 3) {
    ElMessage.error('用户名长度至少3个字符')
    return false
  }

  if (formData.password && formData.password.length < 6) {
    ElMessage.error('密码长度至少6个字符')
    return false
  }

  if (formData.password !== formData.confirmPassword) {
    ElMessage.error('两次输入的密码不一致')
    return false
  }

  if (!formData.username && !formData.password) {
    ElMessage.error('请至少修改用户名或密码中的一项')
    return false
  }

  return true
}

// 保存设置
const saveSettings = async () => {
  if (!validateForm()) {
    return
  }

  try {
    loading.value = true
    saveStatus.value = null

    const requestData: Record<string, string> = {}
    requestData.username = formData.username
    requestData.new_password = formData.password

    const response = await http?.post(`${SERVER_URL}/user/change`, requestData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success('用户设置已保存')

      saveStatus.value = {
        title: '用户设置已保存',
        type: 'success',
        description: '用户名和密码已更新，下次登录时请使用新的凭据',
      }
      // 清空字段
      formData.confirmPassword = ''
      formData.password = ''
      if (response?.data.data) {
        // 如果为true则需要重新登录
        authStore.logout()
        ElMessage.success('已退出登录')
        router.push('/login')
      }
    } else {
      ElMessage.error(response?.data.message || '保存设置失败，请重试')

      saveStatus.value = {
        title: '保存设置失败',
        type: 'error',
        description: response?.data.message || '无法保存用户设置，请检查网络连接后重试',
      }
    }
  } catch (error) {
    console.error('保存设置失败:', error)
    ElMessage.error('保存设置失败，请重试')

    saveStatus.value = {
      title: '保存设置失败',
      type: 'error',
      description: '保存过程中发生错误，请检查网络连接',
    }
  } finally {
    loading.value = false
  }
}

// 组件挂载时加载当前用户名
onMounted(() => {
  loadCurrentUsername()
})

// 加载当前用户名
const loadCurrentUsername = async () => {
  formData.username = authStore.user?.username || ''
  if (formData.username == '') {
    try {
      const response = await http?.get(`${SERVER_URL}/user/info`)
      if (response?.data.code === 200 && response.data.data?.username) {
        formData.username = response.data.data.username
      }
    } catch (error) {
      console.error('加载当前用户名失败:', error)
    }
  }
}
</script>

<style scoped>
.user-settings-container {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0 10px 10px 10px;
}

.user-settings-card,
.security-card {
  width: 100%;
  max-width: none;
  margin: 0;
  border: 0;
  padding: 0;
}

.card-title {
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.card-subtitle {
  margin: 0;
  font-size: 14px;
  color: #909399;
}

.user-form {
  margin-top: 20px;
  width: 100%;
}

.user-form .el-form-item {
  margin-bottom: 24px;
}

.user-form .el-form-item__label {
  font-weight: 500;
  color: #303133;
  margin-bottom: 8px;
}

.form-help {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

.form-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
  flex-wrap: nowrap;
  margin-top: 20px;
}

.save-status {
  margin-top: 20px;
}

.security-content {
  font-size: 14px;
  line-height: 1.6;
}

.security-tips {
  margin: 8px 0 0 0;
  padding-left: 20px;
  color: #606266;
}

.security-tips li {
  margin-bottom: 6px;
}

.warning-section {
  margin-top: 16px;
}
</style>
