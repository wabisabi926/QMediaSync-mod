<template>
  <div class="main-content-container tmdb-settings-container">
    <el-form :model="formData" :label-position="checkIsMobile ? 'top' : 'left'" :label-width="120" class="tmdb-form">
      <el-form-item label="TMDB接口地址" prop="tmdbUrl">
        <el-input v-model="formData.tmdbUrl" placeholder="不填取默认值：https://api.tmdb.org" :disabled="loading"
          maxlength="255" />
        <div class="form-help">可以输入镜像地址，如果不清楚则留空</div>
      </el-form-item>

      <el-form-item label="TMDB图片地址" prop="tmdbImageUrl">
        <el-input v-model="formData.tmdbImageUrl" placeholder="不填取默认值：https://image.tmdb.org" :disabled="loading"
          maxlength="255" />
        <div class="form-help">可以输入镜像地址，如果不清楚则留空</div>
      </el-form-item>

      <el-form-item label="是否启用代理" prop="tmdbEnableProxy">
        <el-switch v-model="formData.tmdbEnableProxy" active-text="启用" inactive-text="禁用" :disabled="loading" />
        <div class="form-help">如果https://api.tmdb.org可以直连就禁用</div>
      </el-form-item>

      <el-form-item label="API密钥" prop="tmdbApiKey">
        <el-input v-model="formData.tmdbApiKey" placeholder="系统内置，不填也可用，填自己的更稳定" type="password" :disabled="loading"
          show-password maxlength="255" />
        <div class="form-help">如果您没有申请TMDB API KEY则留空</div>
      </el-form-item>
      <el-form-item label="刮削本地文件线程数" prop="local_max_threads">
        <el-input-number v-model="formData.local_max_threads" :disabled="loading || !formData.tmdbApiKey" min="1"
          max="100" step="1" style="width: 100%" />
        <div class="form-help">刮削本地文件时的最大并发线程数，越高越快, 刮削网盘该值无效。默认值为5，只有输入自己的tmdb api key时才能调整该值</div>
      </el-form-item>

      <el-form-item label="首选元数据语言" prop="tmdbLanguage">
        <el-radio-group v-model="formData.tmdbLanguage" :disabled="loading" size="large">
          <el-radio-button label="zh-CN">中文</el-radio-button>
          <el-radio-button label="en-US">英文</el-radio-button>
        </el-radio-group>
        <div class="form-help">如果首选语言没有数据，则获取英文数据</div>
      </el-form-item>
      <el-form-item label="首选图片语言" prop="tmdbImageLanguage">
        <el-radio-group v-model="formData.tmdbImageLanguage" :disabled="loading" size="large">
          <el-radio-button label="zh-CN">中文</el-radio-button>
          <el-radio-button label="en-US">英文</el-radio-button>
        </el-radio-group>
        <div class="form-help">如果首选语言没有数据，则获取英文图片</div>
      </el-form-item>

      <div class="form-actions">
        <el-button type="primary" @click="testConnection" :loading="testing" size="large" :icon="Refresh"
          style="margin-right: 15px">
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
            如果不了解如何设置，请全部留空，会使用默认配置。默认配置可能需要代理才可以访问。<br />
            如果TMDB无法直接访问，请开启使用代理，确保代理配置正确。<br />
          </template>
        </el-alert>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, inject, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Check, Refresh } from '@element-plus/icons-vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { isMobile } from '@/utils/deviceUtils'
import { useAuthStore } from '@/stores/auth'

interface TmdbSettings {
  tmdbUrl: string
  tmdbImageUrl: string
  tmdbEnableProxy: boolean
  tmdbApiKey: string
  tmdbAccessToken: string
  tmdbLanguage: string
  tmdbImageLanguage: string
  local_max_threads: number
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

// 表单数据
const formData = reactive<TmdbSettings>({
  tmdbUrl: '',
  tmdbImageUrl: '',
  tmdbEnableProxy: false,
  tmdbApiKey: '',
  tmdbAccessToken: '',
  tmdbLanguage: 'zh-CN',
  tmdbImageLanguage: 'en-US',
  local_max_threads: 5
})

// 页面挂载时获取当前设置
onMounted(async () => {
  await fetchTmdbSettings()
})

// 获取TMDB设置
async function fetchTmdbSettings() {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/scrape/tmdb`, {
      headers: {
        Authorization: `Bearer ${authStore.token}`
      }
    })

    formData.tmdbUrl = response?.data.data.tmdb_url || ''
    formData.tmdbImageUrl = response?.data.data.tmdb_image_url || ''
    formData.tmdbEnableProxy = response?.data.data.tmdb_enable_proxy || false
    formData.tmdbApiKey = response?.data.data.tmdb_api_key || ''
    formData.tmdbAccessToken = response?.data.data.tmdb_access_token || ''
    formData.tmdbLanguage = response?.data.data.tmdb_language || 'zh-CN'
    formData.tmdbImageLanguage = response?.data.data.tmdb_image_language || 'en-US'
    formData.local_max_threads = response?.data.data.local_max_threads || 5
  } catch (error) {
    console.error('获取TMDB设置失败:', error)
    ElMessage.error('获取TMDB设置失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

// 保存TMDB设置
async function saveSettings() {
  try {
    loading.value = true

    const payload = {
      tmdb_url: formData.tmdbUrl,
      tmdb_image_url: formData.tmdbImageUrl,
      tmdb_enable_proxy: formData.tmdbEnableProxy,
      tmdb_api_key: formData.tmdbApiKey,
      tmdb_access_token: formData.tmdbAccessToken,
      tmdb_language: formData.tmdbLanguage,
      tmdb_image_language: formData.tmdbImageLanguage,
      local_max_threads: formData.local_max_threads
    }
    if (!formData.tmdbApiKey) {
      payload.local_max_threads = 5
    }

    await http?.post(`${SERVER_URL}/scrape/tmdb`, payload, {
      headers: {
        Authorization: `Bearer ${authStore.token}`
      }
    })

    saveStatus.value = {
      title: '保存成功',
      type: 'success',
      description: 'TMDB设置已成功保存'
    }

    // 3秒后清除状态提示
    setTimeout(() => {
      saveStatus.value = null
    }, 3000)

  } catch (error) {
    console.error('保存TMDB设置失败:', error)
    saveStatus.value = {
      title: '保存失败',
      type: 'error',
      description: '保存TMDB设置失败，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}

// 测试TMDB连通性
async function testConnection() {
  try {
    testing.value = true
    testStatus.value = null

    const payload = {
      tmdb_url: formData.tmdbUrl,
      tmdb_image_url: formData.tmdbImageUrl,
      tmdb_enable_proxy: formData.tmdbEnableProxy,
      tmdb_api_key: formData.tmdbApiKey,
      tmdb_access_token: formData.tmdbAccessToken,
      tmdb_language: formData.tmdbLanguage,
      tmdb_image_language: formData.tmdbImageLanguage
    }

    const response = await http?.post(`${SERVER_URL}/scrape/tmdb-test`, payload, {
      timeout: 20000,
      headers: {
        Authorization: `Bearer ${authStore.token}`
      }
    })

    // 根据接口返回结果显示不同的状态
    if (response?.data?.data) {
      testStatus.value = {
        title: '连接成功',
        type: 'success',
        description: response.data.message || 'TMDB连通性测试成功'
      }
    } else {
      testStatus.value = {
        title: '连接失败',
        type: 'error',
        description: response?.data?.message || 'TMDB连通性测试失败，请检查设置'
      }
    }

    // 5秒后清除状态提示
    setTimeout(() => {
      testStatus.value = null
    }, 5000)

  } catch (error) {
    console.error('测试TMDB连通性失败:', error)
    testStatus.value = {
      title: '连接失败',
      type: 'error',
      description: '测试过程中发生错误，请检查网络连接和设置'
    }
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.tmdb-settings-container {
  padding: 20px;
  max-width: 800px;
}

.tmdb-form {
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
