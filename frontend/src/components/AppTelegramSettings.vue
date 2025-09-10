<template>
  <div class="telegram-settings-container">
    <el-card class="telegram-settings-card" shadow="hover">
      <template #header>
        <h2 class="card-title">Telegram通知设置</h2>
        <p class="card-subtitle">配置Telegram机器人用于接收系统通知</p>
      </template>

      <el-form
        :model="formData"
        :label-position="checkIsMobile ? 'top' : 'left'"
        :label-width="100"
        class="telegram-form"
      >
        <el-form-item label="启用" prop="enabled">
          <div class="enable-switch">
            <el-switch
              v-model="formData.enabled"
              :loading="loading"
              size="large"
              active-text="已启用"
              inactive-text="已禁用"
            />
            <div class="form-help">开启后，系统将通过Telegram机器人发送重要通知</div>
          </div>
        </el-form-item>

        <el-form-item label="机器人Token" prop="telegram_bot_token">
          <el-input
            v-model="formData.telegram_bot_token"
            placeholder="搜索@BotFather创建机器人，找到TOKEN"
            :disabled="loading || !formData.enabled"
          />
          <div class="form-help">在Telegram中搜索@BotFather，创建机器人后获取TOKEN</div>
        </el-form-item>

        <el-form-item label="用户ID" prop="telegram_user_id">
          <el-input
            v-model="formData.telegram_user_id"
            placeholder="搜索@get_id_bot，点开始，找到Your Chat Id=后面的数字"
            :disabled="loading || !formData.enabled"
          />
          <div class="form-help">在Telegram中搜索@get_id_bot，点击开始获取您的Chat ID</div>
        </el-form-item>

        <el-form-item>
          <div class="form-actions">
            <el-button
              type="primary"
              @click="testBot"
              :loading="testing"
              :disabled="loading || !formData.enabled"
              :icon="Message"
            >
              测试机器人
            </el-button>

            <el-button
              type="success"
              @click="saveSettings"
              :loading="loading"
              :disabled="testing"
              :icon="Check"
            >
              保存设置
            </el-button>

            <el-button @click="resetForm" :disabled="loading || testing" :icon="RefreshLeft">
              重置
            </el-button>
          </div>
        </el-form-item>
      </el-form>

      <!-- 测试状态显示 -->
      <el-alert
        v-if="testStatus"
        :title="testStatus.title"
        :type="testStatus.type"
        :description="testStatus.description"
        :closable="false"
        show-icon
        class="test-status"
      />
    </el-card>

    <!-- 使用说明 -->
    <el-card class="help-card" shadow="hover">
      <template #header>
        <h3>注意事项：</h3>
      </template>

      <div class="help-content">
        <div class="help-tips">
          <ul>
            <li>如果您在中国大陆地区，那么需要设置网络代理才可以访问Telegram接口</li>
            <li>需要先启用Telegram通知功能才能配置和测试机器人</li>
            <li>机器人Token是敏感信息，请妥善保管</li>
            <li>确保您已经与机器人进行过对话，否则无法接收消息</li>
            <li>用户ID是数字格式，不是用户名</li>
            <li>配置完成后建议先测试机器人功能</li>
            <li>禁用功能后仍会保存配置信息，重新启用时无需重新配置</li>
          </ul>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted, inject } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Message, Check, RefreshLeft } from '@element-plus/icons-vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { isMobile } from '@/utils/deviceUtils'

interface TelegramSettings {
  enabled: boolean
  telegram_bot_token: string
  telegram_user_id: string
}

interface TestStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}
const checkIsMobile = ref(isMobile())
const http: AxiosStatic | undefined = inject('$http')
const loading = ref(false)
const testing = ref(false)
const testStatus = ref<TestStatus | null>(null)

const formData = reactive<TelegramSettings>({
  enabled: false,
  telegram_bot_token: '',
  telegram_user_id: '',
})

// 测试机器人
const testBot = async () => {
  if (!formData.enabled) {
    ElMessage.warning('请先启用Telegram通知功能')
    return
  }

  if (!formData.telegram_bot_token || !formData.telegram_user_id) {
    ElMessage.warning('请先填写机器人Token和用户ID')
    return
  }

  try {
    testing.value = true
    testStatus.value = null

    const requestData = {
      enabled: formData.enabled ? 1 : 0,
      token: formData.telegram_bot_token,
      chat_id: formData.telegram_user_id,
    }

    const response = await http?.post(`${SERVER_URL}/telegram/test`, requestData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      testStatus.value = {
        title: '机器人测试成功',
        type: 'success',
        description: '测试消息已发送到您的Telegram，请检查是否收到消息',
      }
      ElMessage.success('机器人测试成功')
    } else {
      testStatus.value = {
        title: '机器人测试失败',
        type: 'error',
        description: response?.data.msg || '无法发送测试消息，请检查Token和用户ID是否正确',
      }
      ElMessage.error(response?.data.msg || '机器人测试失败')
    }
  } catch (error) {
    console.error('机器人测试错误:', error)
    testStatus.value = {
      title: '机器人测试出错',
      type: 'error',
      description: '测试过程中发生错误，请检查网络连接和配置信息',
    }
    ElMessage.error('机器人测试出错')
  } finally {
    testing.value = false
  }
}

// 保存设置
const saveSettings = async () => {
  try {
    loading.value = true

    const requestData = {
      enabled: formData.enabled ? 1 : 0,
      token: formData.telegram_bot_token,
      chat_id: formData.telegram_user_id,
    }

    const response = await http?.post(`${SERVER_URL}/setting/telegram`, requestData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      const statusMessage = formData.enabled ? 'Telegram通知已启用并保存' : 'Telegram通知已禁用'
      ElMessage.success(statusMessage)

      testStatus.value = {
        title: '设置已保存',
        type: 'info',
        description: formData.enabled
          ? '配置已成功保存，您可以开始接收Telegram通知了'
          : 'Telegram通知功能已禁用，不会发送任何通知',
      }
    } else {
      ElMessage.error(response?.data.msg || '保存设置失败，请重试')
    }
  } catch (error) {
    console.error('保存设置错误:', error)
    ElMessage.error('保存设置失败，请重试')
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

    formData.enabled = false
    formData.telegram_bot_token = ''
    formData.telegram_user_id = ''
    testStatus.value = null
    ElMessage.info('表单已重置')
  } catch {
    // 用户取消
  }
}

// 加载设置
const loadSettings = async () => {
  try {
    loading.value = true

    const response = await http?.get(`${SERVER_URL}/setting/telegram`)

    if (response?.data.code === 200 && response.data.data) {
      formData.enabled = response.data.data.enabled === '1'
      formData.telegram_bot_token = response.data.data.token || ''
      formData.telegram_user_id = response.data.data.chat_id || ''
    }
  } catch (error) {
    console.error('加载设置错误:', error)
    ElMessage.warning('加载已保存的设置失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.telegram-settings-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.telegram-settings-card {
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

.telegram-form {
  margin-top: 20px;
  width: 100%;
}

.telegram-form .el-form-item {
  margin-bottom: 24px;
}

.telegram-form .el-form-item__label {
  font-weight: 500;
  color: #303133;
  margin-bottom: 8px;
}

.enable-switch {
  display: flex;
  flex-direction: column;
  gap: 8px;
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

.test-status {
  margin-top: 20px;
}

.help-card {
  width: 100%;
  max-width: none;
  margin: 0;
}

.help-content {
  font-size: 14px;
  line-height: 1.6;
}

.help-tips {
  margin-top: 20px;
}

.help-tips h4 {
  margin: 0 0 12px 0;
  color: #303133;
  font-size: 16px;
}

.help-tips ul {
  margin: 0;
  padding-left: 20px;
  color: #606266;
}

.help-tips li {
  margin-bottom: 8px;
}
</style>
