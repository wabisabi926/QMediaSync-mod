<template>
  <div class="main-content-container user-settings-container">
    <el-form
      :model="formData"
      :label-position="checkIsMobile ? 'top' : 'left'"
      :label-width="90"
      class="user-form"
    >
      <el-form-item label="密码" prop="password" required>
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

        <el-button @click="resetForm" :disabled="loading" size="large" :icon="RefreshLeft">
          重置
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
    <!-- 安全提示 -->
    <el-card class="security-card" shadow="hover">
      <template #header>
        <h3>安全提示</h3>
      </template>

      <div class="security-content">
        <el-alert title="密码安全建议" type="info" :closable="false" show-icon>
          <template #default>
            <ul class="security-tips">
              <li>密码长度至少8位，建议12位以上</li>
              <li>包含大写字母、小写字母、数字和特殊字符</li>
              <li>避免使用常见密码或个人信息</li>
              <li>定期更换密码以提高安全性</li>
              <li>不要在多个系统中使用相同密码</li>
            </ul>
          </template>
        </el-alert>

        <el-divider />

        <div class="warning-section">
          <el-alert title="重要提醒" type="warning" :closable="false" show-icon>
            <template #default>
              修改用户名或密码后，您需要重新登录系统。请确保记住新的登录凭据。
            </template>
          </el-alert>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, inject } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, RefreshLeft } from '@element-plus/icons-vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { isMobile } from '@/utils/deviceUtils'

interface UserSettings {
  password: string
  confirmPassword: string
}

interface SaveStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}

const checkIsMobile = ref(isMobile())
const http: AxiosStatic | undefined = inject('$http')
const loading = ref(false)
const saveStatus = ref<SaveStatus | null>(null)

const formData = reactive<UserSettings>({
  password: '',
  confirmPassword: '',
})

// 表单验证
const validateForm = (): boolean => {
  if (!formData.password) {
    ElMessage.error('请输入密码')
    return false
  }

  if (formData.password.length < 6) {
    ElMessage.error('密码长度至少6个字符')
    return false
  }

  if (formData.password !== formData.confirmPassword) {
    ElMessage.error('两次输入的密码不一致')
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

    const requestData = {
      new_password: formData.password,
      confirm_password: formData.confirmPassword,
    }

    const response = await http?.post(`${SERVER_URL}/change-password`, requestData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success('用户密码已修改')

      saveStatus.value = {
        title: '用户密码已修改',
        type: 'success',
        description: '用户密码已更新，下次登录时请使用新的凭据',
      }

      // 清空确认密码字段
      formData.confirmPassword = ''
    } else {
      ElMessage.error(response?.data.msg || '修改密码失败，请重试')

      saveStatus.value = {
        title: '修改密码失败',
        type: 'error',
        description: response?.data.msg || '无法保存新密码，请检查网络连接后重试',
      }
    }
  } catch (error) {
    console.error('修改密码失败:', error)
    ElMessage.error('修改密码失败，请重试')

    saveStatus.value = {
      title: '修改密码失败',
      type: 'error',
      description: '保存过程中发生错误，请检查网络连接',
    }
  } finally {
    loading.value = false
  }
}

// 重置表单
const resetForm = async () => {
  try {
    await ElMessageBox.confirm('确定要重置表单吗？所有未保存的更改将丢失。', '确认重置', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    // 重置表单数据
    formData.password = ''
    formData.confirmPassword = ''
    saveStatus.value = null
    ElMessage.info('表单已重置')
  } catch {
    // 用户取消
  }
}
</script>

<style scoped>
.user-settings-container {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.user-settings-card,
.security-card {
  width: 100%;
  max-width: none;
  margin: 0;
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
