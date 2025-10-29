<template>
  <div class="main-content-container ai-settings-container">
    <el-alert title="" type="error" :closable="false" style="margin-bottom: 20px;">
      <template #default>
        推荐 <a href="https://cloud.siliconflow.cn/i/fNSX73Tt" target="_blank">硅基流动</a> 的模型，新号输我的邀请码送2000万 Tokens:
        <b>fNSX73Tt</b>
      </template>
    </el-alert>
    <el-form :model="formData" :label-position="checkIsMobile ? 'top' : 'left'" :label-width="120" :rules="formRules"
      ref="formRef" class="ai-form">
      <el-form-item label="API接口地址" prop="aiBaseUrl">
        <el-input v-model="formData.aiBaseUrl" placeholder="不填取默认值：https://api.siliconflow.cn" :disabled="loading"
          maxlength="255" />
        <div class="form-help">例如：https://api.deepseek.com</div>
      </el-form-item>

      <el-form-item label="API Key" prop="aiApiKey">
        <el-input v-model="formData.aiApiKey" placeholder="不填用作者的，但可能失败；填了更稳定" type="password" :disabled="loading"
          show-password maxlength="255" />
        <div class="form-help">API服务的访问密钥</div>
      </el-form-item>

      <el-form-item label="模型名称" prop="aiModelName">
        <el-input v-model="formData.aiModelName" placeholder="不填取默认值：deepseek-ai/DeepSeek-R1" :disabled="loading"
          maxlength="100" />
        <div class="form-help">从硅基流动的模型广场找或者模型提供商的API文档中找</div>
      </el-form-item>

      <el-form-item label="请求超时时间" prop="ai_timeout">
        <el-input-number v-model="formData.ai_timeout" :disabled="loading || !formData.aiApiKey" min="5" max="120"
          step="1" style="width: 100%" />
        <div class="form-help">请求超时时间，单位秒，默认值为120秒，减少该值可以极大提高刮削速度，但可能会导致AI识别失败</div>
      </el-form-item>

      <div class="form-actions">
        <el-button type="primary" @click="testConnection" :loading="testing" :disabled="loading" size="large"
          :icon="Refresh" style="margin-right: 15px">
          测试连通性
        </el-button>

        <el-button type="success" @click="saveSettings" :loading="loading" size="large" :icon="Check">
          保存设置
        </el-button>
      </div>
    </el-form>

    <!-- 保存状态显示 -->
    <el-alert v-if="saveStatus" :title="saveStatus.title" :type="saveStatus.type" :description="saveStatus.description"
      :closable="false" show-icon class="save-status" />

    <!-- 测试状态显示 -->
    <el-alert v-if="testStatus" :title="testStatus.title" :type="testStatus.type" :description="testStatus.description"
      :closable="false" show-icon class="test-status" />

    <div class="security-content">
      <div class="warning-section">
        <el-alert title="使用提示" type="warning" :closable="false" show-icon>
          <template #default>
            可以支持所有OpenAI Api兼容的模型。<br />
            项目内置硅基流动的访问权限，如果使用强度不高，建议上面所有输入框留空来使用系统默认值<br />
            使用AI识别会消耗tokens额度, 请合理配置。
          </template>
        </el-alert>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, inject, onMounted, computed } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { Check, Refresh } from '@element-plus/icons-vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { isMobile } from '@/utils/deviceUtils'
import { useAuthStore } from '@/stores/auth'

interface AiSettings {
  aiBaseUrl: string
  aiApiKey: string
  aiModelName: string
  ai_timeout: number
}

interface SaveStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}
const http: AxiosStatic | undefined = inject('$http')
const authStore = useAuthStore()
const checkIsMobile = ref(isMobile())
const loading = ref(false)
const testing = ref(false)
const saveStatus = ref<SaveStatus | null>(null)
const testStatus = ref<SaveStatus | null>(null)
const formRef = ref<FormInstance>()

// 表单数据
const formData = reactive<AiSettings>({
  aiBaseUrl: '',
  aiApiKey: '',
  aiModelName: '',
  ai_timeout: 120
})

// 动态表单验证规则
const formRules = computed(() => {
  return {
    aiBaseUrl: [
      {
        message: '请输入API接口地址',
        trigger: 'blur'
      }
    ],
    aiApiKey: [
      {
        message: '请输入API Key',
        trigger: 'blur'
      }
    ],
    aiModelName: [
      {
        message: '请输入模型名称',
        trigger: 'blur'
      }
    ],
    ai_timeout: [
      {
        message: '请输入请求超时时间',
        trigger: 'blur'
      }
    ]
  }
})

// 页面挂载时获取当前设置
onMounted(async () => {
  await fetchAiSettings()
})

// 获取AI设置
async function fetchAiSettings() {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/scrape/ai-settings`, {
      headers: {
        Authorization: `Bearer ${authStore.token}`
      }
    })
    formData.aiBaseUrl = response?.data.data.ai_base_url || ''
    formData.aiApiKey = response?.data.data.ai_api_key || ''
    formData.aiModelName = response?.data.data.ai_model_name || ''
    formData.ai_timeout = response?.data.data.ai_timeout || 120
  } catch (error) {
    console.error('获取AI设置失败:', error)
    ElMessage.error('获取AI设置失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

// 保存AI设置
async function saveSettings() {
  try {
    // 执行表单验证
    await formRef.value?.validate()
    if (formData.aiModelName && !formData.aiApiKey) {
      ElMessage.error('如果填写了模型名称，必须填写API Key')
      return
    }
    if (formData.aiBaseUrl && (!formData.aiApiKey || !formData.aiModelName)) {
      ElMessage.error('如果填写了API接口地址，必须填写模型名称和API Key')
      return
    }
    loading.value = true

    const payload = {
      ai_base_url: formData.aiBaseUrl,
      ai_api_key: formData.aiApiKey,
      ai_model_name: formData.aiModelName,
      ai_timeout: formData.ai_timeout
    }

    await http?.post(`${SERVER_URL}/scrape/ai-settings`, payload, {
      headers: {
        Authorization: `Bearer ${authStore.token}`
      }
    })

    saveStatus.value = {
      title: '保存成功',
      type: 'success',
      description: 'AI识别设置已成功保存'
    }

    // 3秒后清除状态提示
    setTimeout(() => {
      saveStatus.value = null
    }, 3000)

  } catch (error) {
    // 如果是验证错误，则不显示保存失败的消息
    if (error !== false) {
      console.error('保存AI设置失败:', error)
      saveStatus.value = {
        title: '保存失败',
        type: 'error',
        description: '保存AI设置失败，请稍后重试'
      }
    }
  } finally {
    loading.value = false
  }
}

// 测试AI连通性
async function testConnection() {
  try {
    // 执行表单验证
    await formRef.value?.validate()

    testing.value = true
    testStatus.value = null

    const payload = {
      ai_base_url: formData.aiBaseUrl,
      ai_api_key: formData.aiApiKey,
      ai_model_name: formData.aiModelName,
    }

    const response = await http?.post(`${SERVER_URL}/scrape/ai-test`, payload, {
      timeout: 120000,
      headers: {
        Authorization: `Bearer ${authStore.token}`
      }
    })

    // 根据接口返回结果显示不同的状态
    if (response?.data?.code === 200) {
      testStatus.value = {
        title: '测试成功',
        type: 'success',
        description: response.data.message || 'AI服务连通性测试成功'
      }
    } else {
      testStatus.value = {
        title: '测试失败',
        type: 'error',
        description: response?.data?.message || 'AI服务连通性测试失败，请检查设置'
      }
    }

    // 5秒后清除状态提示
    setTimeout(() => {
      testStatus.value = null
    }, 5000)

  } catch (error) {
    // 如果是验证错误，则不显示测试失败的消息
    if (error !== false) {
      console.error('测试AI连通性失败:', error)
      testStatus.value = {
        title: '连接失败',
        type: 'error',
        description: '测试过程中发生错误，请检查网络连接和设置'
      }
    }
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.ai-settings-container {
  padding: 20px;
  max-width: 800px;
}

.ai-form {
  background: #fff;
  padding: 20px;
}

.form-help {
  color: #909399;
  font-size: 12px;
  margin-top: 5px;
}

.form-actions {
  margin-top: 30px;
  text-align: center;
}

.save-status,
.test-status {
  margin-top: 20px;
}

.security-content {
  margin-top: 30px;
}

.warning-section {
  margin-bottom: 20px;
}
</style>
