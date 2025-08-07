<template>
  <div class="cookiecloud-container">
    <el-card class="cookiecloud-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <h2 class="card-title">CookieCloud 设置</h2>
          <p class="card-subtitle">
            从 CookieCloud 中读取115的登录Cookie，只要网页不退出登录就可以一直使用
          </p>
        </div>
      </template>

      <el-form
        :model="formData"
        :rules="formRules"
        ref="formRef"
        label-position="top"
        class="cookiecloud-form"
      >
        <el-form-item label="启用CookieCloud" prop="enabled">
          <el-switch
            v-model="formData.enabled"
            active-text="启用"
            inactive-text="禁用"
            :disabled="loading"
          />
          <div class="form-help">开启后将使用CookieCloud服务获取Cookie</div>
        </el-form-item>

        <el-form-item label="服务器地址" prop="serverUrl" required>
          <el-input
            v-model="formData.serverUrl"
            placeholder="请输入CookieCloud服务器地址"
            :disabled="loading || !formData.enabled"
          >
          </el-input>
          <div class="form-help">
            例如：http://cookiecloud.mynas.com 或 http://192.168.1.100:8088
          </div>
        </el-form-item>

        <el-form-item label="用户Key/UUID" prop="uuid" required>
          <el-input
            v-model="formData.uuid"
            placeholder="请输入UUID"
            :disabled="loading || !formData.enabled"
          >
          </el-input>
          <div class="form-help">用户Key/UUID</div>
        </el-form-item>

        <el-form-item label="端到端加密密码" prop="password" required>
          <el-input
            v-model="formData.password"
            type="password"
            placeholder="请输入加密密码"
            show-password
            :disabled="loading || !formData.enabled"
          />
          <div class="form-help">用于加密传输的密码，请妥善保管</div>
        </el-form-item>

        <el-form-item>
          <div class="form-actions">
            <el-button
              type="primary"
              size="large"
              @click="testConnection"
              :loading="testing"
              :disabled="loading || !formData.enabled"
            >
              <el-icon><Connection /></el-icon>
              测试连接
            </el-button>

            <el-button
              type="success"
              size="large"
              @click="saveSettings"
              :loading="loading"
              :disabled="testing"
            >
              <el-icon><Check /></el-icon>
              保存设置
            </el-button>

            <el-button size="large" @click="resetForm" :disabled="loading || testing">
              <el-icon><RefreshLeft /></el-icon>
              重置
            </el-button>
          </div>
        </el-form-item>
      </el-form>

      <!-- 连接状态显示 -->
      <el-alert
        v-if="connectionStatus"
        :title="connectionStatus.title"
        :type="connectionStatus.type"
        :description="connectionStatus.description"
        :closable="false"
        show-icon
        class="connection-status"
      />
    </el-card>

    <!-- 使用说明 -->
    <el-card class="help-card" shadow="hover">
      <template #header>
        <h3>使用说明</h3>
      </template>

      <div class="help-content">
        <el-steps direction="vertical" :active="4">
          <el-step title="部署CookieCloud服务器" description="在服务器上部署CookieCloud服务" />
          <el-step title="配置服务器地址" description="填入CookieCloud服务器的访问地址" />
          <el-step title="设置UUID和密码" description="生成唯一的UUID并设置加密密码" />
          <el-step title="测试连接" description="点击测试连接按钮确认配置正确" />
          <el-step title="保存设置" description="保存配置并开始使用CookieCloud服务" />
        </el-steps>

        <el-divider />

        <div class="help-tips">
          <h4>配置提示：</h4>
          <ul>
            <li>服务器地址支持域名和IP地址，可包含端口号</li>
            <li>UUID用于标识不同的客户端，建议为每个设备生成不同的UUID</li>
            <li>加密密码用于保护传输数据，请使用强密码</li>
          </ul>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted, inject } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Connection, Check, RefreshLeft } from '@element-plus/icons-vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'

interface CookieCloudSettings {
  enabled: boolean
  serverUrl: string
  uuid: string
  password: string
}

interface ConnectionStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}

const http: AxiosStatic | undefined = inject('$http')
const formRef = ref<FormInstance>()
const loading = ref(false)
const testing = ref(false)
const connectionStatus = ref<ConnectionStatus | null>(null)

const formData = reactive<CookieCloudSettings>({
  enabled: true,
  serverUrl: '',
  uuid: '',
  password: '',
})

const formRules: FormRules = {
  serverUrl: [
    {
      required: true,
      message: '请输入服务器地址',
      trigger: 'blur',
      validator: (rule, value, callback) => {
        if (!formData.enabled) {
          callback()
          return
        }
        if (!value) {
          callback(new Error('请输入服务器地址'))
          return
        }
        const pattern =
          /^http[s]{0,1}\:\/\/[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})*(:[0-9]{1,5})?$|^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(:[0-9]{1,5})?$/
        if (!pattern.test(value)) {
          callback(new Error('请输入有效的服务器地址'))
          return
        }
        callback()
      },
    },
  ],
  uuid: [
    {
      required: true,
      message: '请输入UUID',
      trigger: 'blur',
      validator: (rule, value, callback) => {
        if (!formData.enabled) {
          callback()
          return
        }
        if (!value) {
          callback(new Error('请输入UUID'))
          return
        }
        callback()
      },
    },
  ],
  password: [
    {
      required: true,
      message: '请输入加密密码',
      trigger: 'blur',
      validator: (rule, value, callback) => {
        if (!formData.enabled) {
          callback()
          return
        }
        if (!value) {
          callback(new Error('请输入加密密码'))
          return
        }
        if (value.length < 6 || value.length > 50) {
          callback(new Error('密码长度在 6 到 50 个字符'))
          return
        }
        callback()
      },
    },
  ],
}

// 测试连接
const testConnection = async () => {
  if (!formRef.value) return

  if (!formData.enabled) {
    ElMessage.warning('请先启用CookieCloud功能')
    return
  }

  try {
    const valid = await formRef.value.validate()
    if (!valid) return

    testing.value = true
    connectionStatus.value = null
    const reqFormData = new FormData()
    reqFormData.append('serverUrl', formData.serverUrl)
    reqFormData.append('uuid', formData.uuid)
    reqFormData.append('password', formData.password)

    const response = await http?.post(`${SERVER_URL}/cookiecloud/test`, reqFormData, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
    })

    if (response?.data.code === 200) {
      connectionStatus.value = {
        title: '连接测试成功',
        type: 'success',
        description: `成功连接到 CookieCloud 服务器 (${formData.serverUrl})`,
      }
      ElMessage.success('连接测试成功')
    } else {
      connectionStatus.value = {
        title: '连接测试失败',
        type: 'error',
        description:
          response?.data.msg || '无法连接到指定的服务器，请检查服务器地址、网络连接或防火墙设置',
      }
      ElMessage.error(response?.data.msg || '连接测试失败')
    }
  } catch (error) {
    console.error('连接测试错误:', error)
    connectionStatus.value = {
      title: '连接测试出错',
      type: 'error',
      description: '测试过程中发生错误，请检查配置信息',
    }
    ElMessage.error('连接测试出错')
  } finally {
    testing.value = false
  }
}

// 保存设置
const saveSettings = async () => {
  if (!formRef.value) return

  try {
    const valid = await formRef.value.validate()
    if (!valid) return

    loading.value = true
    // 构造标准 form 表单数据
    const reqFormData = new URLSearchParams()
    reqFormData.append('enabled', formData.enabled ? '1' : '0')
    reqFormData.append('serverUrl', formData.serverUrl)
    reqFormData.append('uuid', formData.uuid)
    reqFormData.append('password', formData.password)

    const response = await http?.post(`${SERVER_URL}/setting/update-cookie-cloud`, reqFormData, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
    })
    if (response?.data.code === 200) {
      ElMessage.success('CookieCloud 设置已保存')

      connectionStatus.value = {
        title: '设置已保存',
        type: 'info',
        description: '配置已成功保存，你可以开始使用 CookieCloud 服务了',
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

    formRef.value?.resetFields()
    connectionStatus.value = null
    ElMessage.info('表单已重置')
  } catch {
    // 用户取消
  }
}

// 加载设置
const loadSettings = async () => {
  try {
    loading.value = true

    const response = await http?.get(`${SERVER_URL}/setting/get-cookie-cloud`)

    if (response?.data.code === 200 && response.data.data) {
      formData.enabled = response.data.data.enabled === '1'
      formData.serverUrl = response.data.data.host || ''
      formData.uuid = response.data.data.uuid || ''
      formData.password = response.data.data.password || ''
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
.cookiecloud-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.cookiecloud-card {
  width: 100%;
  max-width: none;
  margin: 0;
}

.card-header {
  text-align: center;
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

.cookiecloud-form {
  margin-top: 20px;
  width: 100%;
}

.cookiecloud-form .el-form-item {
  margin-bottom: 24px;
}

.cookiecloud-form .el-form-item__label {
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
  flex-wrap: wrap;
  margin-top: 20px;
}

.connection-status {
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

/* 移动端适配 */
@media (max-width: 768px) {
  .cookiecloud-card,
  .help-card {
    margin: 0;
  }

  .card-title {
    font-size: 20px;
  }

  .card-subtitle {
    font-size: 13px;
  }

  .form-actions {
    flex-direction: column;
    gap: 8px;
  }

  .form-actions .el-button {
    width: 100%;
  }

  .el-input {
    font-size: 16px; /* 防止iOS缩放 */
  }

  .el-steps {
    padding: 0 10px;
  }

  .el-form-item {
    margin-bottom: 20px;
  }

  .el-form-item__label {
    font-size: 14px !important;
    margin-bottom: 8px !important;
    font-weight: 500;
  }
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .card-title {
    font-size: 18px;
  }

  .cookiecloud-form {
    margin-top: 15px;
  }

  .help-tips ul {
    padding-left: 16px;
  }

  .help-tips li {
    font-size: 13px;
  }

  .el-form-item {
    margin-bottom: 18px;
  }
}
</style>
