<template>
  <div class="main-content-container thread-settings-container">
    <el-alert title="重要提示" type="error" :closable="false" show-icon style="margin-bottom: 20px;">
      <template #default>
        修改无法立即生效，请重启容器(docker部署)或者重启应用(非docker部署)，重启会丢失现在的下载队列，谨慎操作。
      </template>
    </el-alert>
    <el-form :model="formData" :label-position="checkIsMobile ? 'top' : 'left'" :label-width="150" class="thread-form">
      <el-form-item label="下载QPS" prop="downloadThreads">
        <el-input-number v-model="formData.downloadThreads" :min="1" :max="3" :disabled="loading" size="large" />
        <div class="form-help">下载队列的每秒处理数量，合理设置可提高下载速度，最大3，最小1，最大是因为要给播放和刮削留出空余。</div>
      </el-form-item>

      <el-form-item label="115 接口请求QPS" prop="fileDetailThreads">
        <el-input-number v-model="formData.fileDetailThreads" :min="1" :max="10" :disabled="loading" size="large" />
        <div class="form-help">115开放平台接口的每秒请求数量，影响同步速度，最大10，最小1</div>
      </el-form-item>

      <div class="form-actions">
        <el-button type="success" @click="saveSettings" :loading="loading" size="large" :icon="Check">
          保存设置
        </el-button>
      </div>
    </el-form>

    <!-- 保存状态显示 -->
    <el-alert v-if="saveStatus" :title="saveStatus.title" :type="saveStatus.type" :description="saveStatus.description"
      :closable="false" show-icon class="save-status" />
    <div class="security-content">
      <div class="warning-section">
        <el-alert title="使用提示" type="warning" :closable="false" show-icon>
          <template #default>
            线程数设置过高可能导致115提示访问量过高，请根据实际情况合理设置。一般建议总线程数不超过10。<br />
            如果您授权了多个第三方平台，请保证其他第三方的线程数+本项目的线程数不要过高。<br />
            如果日志中提示访问量过高，请适当减少接口请求QPS。<br />
            如果日志中下载文件提示403，请适当减少下载QPS。<br />
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

interface ThreadSettings {
  downloadThreads: number
  fileDetailThreads: number
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
const saveStatus = ref<SaveStatus | null>(null)

// 表单数据
const formData = reactive<ThreadSettings>({
  downloadThreads: 5,
  fileDetailThreads: 3
})

// 页面挂载时获取当前设置
onMounted(async () => {
  await fetchThreadSettings()
})

// 获取线程设置
async function fetchThreadSettings() {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/setting/threads`, {
      headers: {
        Authorization: `Bearer ${authStore.token}`
      }
    })

    formData.downloadThreads = response?.data.data.download_threads || 5
    formData.fileDetailThreads = response?.data.data.file_detail_threads || 3
  } catch (error) {
    console.error('获取线程设置失败:', error)
    ElMessage.error('获取线程设置失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

// 保存线程设置
async function saveSettings() {
  try {
    loading.value = true

    const payload = {
      download_threads: formData.downloadThreads,
      file_detail_threads: formData.fileDetailThreads
    }

    await http?.post(`${SERVER_URL}/setting/threads`, payload, {
      headers: {
        Authorization: `Bearer ${authStore.token}`
      }
    })

    saveStatus.value = {
      title: '保存成功',
      type: 'success',
      description: '线程设置已成功保存'
    }

    // 3秒后清除状态提示
    setTimeout(() => {
      saveStatus.value = null
    }, 3000)

  } catch (error) {
    console.error('保存线程设置失败:', error)
    saveStatus.value = {
      title: '保存失败',
      type: 'error',
      description: '保存线程设置失败，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.thread-settings-container {
  padding: 20px;
  max-width: 800px;
  /* margin: 0 auto; */
}

/* .thread-form { */
/* background: #fff;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1); */
/* } */

.form-help {
  color: #909399;
  font-size: 12px;
  margin-top: 5px;
}

.form-actions {
  margin-top: 30px;
  text-align: center;
}

.save-status {
  margin-top: 20px;
}

.security-content {
  margin-top: 30px;
}

.warning-section {
  margin-bottom: 20px;
}
</style>
