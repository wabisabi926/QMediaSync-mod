<template>
  <div class="main-content-container thread-settings-container">
    <el-form
      :model="formData"
      :label-position="checkIsMobile ? 'top' : 'left'"
      :label-width="200"
      class="thread-form"
    >
      <el-form-item label="下载队列每秒处理数量" prop="downloadThreads">
        <el-input-number
          v-model="formData.downloadThreads"
          :min="1"
          :max="10"
          :disabled="loading"
          size="large"
        />
        <div class="form-help">
          控制每秒加入“下载中”的任务数，并不是实际每秒下载数。因此“下载中”数量可能高于该值，这是正常的。建议保留余量给播放和刮削，范围
          1 到 10。
        </div>
      </el-form-item>

      <el-form-item label="网盘接口每秒请求数量" prop="fileDetailThreads">
        <el-input-number
          v-model="formData.fileDetailThreads"
          :min="2"
          :max="10"
          :disabled="loading"
          size="large"
        />
        <div class="form-help">
          控制 115 或百度网盘开放平台每秒请求数，影响同步速度，范围 2 到 10。
        </div>
      </el-form-item>

      <el-form-item label="OpenList 接口请求 QPS" prop="openlistQPS">
        <el-input-number
          v-model="formData.openlistQPS"
          :min="2"
          :max="10"
          :disabled="loading"
          size="large"
        />
        <div class="form-help">控制 OpenList 接口每秒请求数，影响同步速度，范围 2 到 10。</div>
      </el-form-item>

      <el-form-item label="OpenList 接口重试次数" prop="openlistRetryCount">
        <el-input-number
          v-model="formData.openlistRetryCount"
          :min="1"
          :max="10"
          :disabled="loading"
          size="large"
        />
        <div class="form-help">
          OpenList
          接口请求失败后的重试次数。超过次数仍失败时会停止同步；次数越多，失败场景下耗时越长。范围 1
          到 10。
        </div>
      </el-form-item>

      <el-form-item label="OpenList 接口重试间隔秒数" prop="openlistRetryDelay">
        <el-input-number
          v-model="formData.openlistRetryDelay"
          :min="30"
          :max="3600"
          :disabled="loading"
          size="large"
        />
        <div class="form-help">
          OpenList 接口每次重试的间隔时间，单位为秒。间隔越大，失败后恢复越慢，范围 30 到 3600。
        </div>
      </el-form-item>

      <el-form-item label="115 文件列表每页查询数量" prop="fileListPageSize">
        <el-input-number
          v-model="formData.fileListPageSize"
          :min="100"
          :max="1150"
          :disabled="loading"
          size="large"
        />
        <div class="form-help">
          115 网盘文件列表接口每页返回的文件数量。网络较慢时可减小此值避免超时，范围 100 到
          1150，默认 1150。
        </div>
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
        <el-alert title="使用提示" type="warning" :closable="false" show-icon>
          <template #default>
            线程数过高可能导致 115 提示访问量过高，一般建议总线程数不超过 10。<br />
            如果还授权了其他第三方平台，也要把它们的线程数一起算进去。<br />
            如果日志提示访问量过高，请降低接口请求 QPS。<br />
            如果下载文件时出现 403，请降低下载 QPS。<br />
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

interface ThreadSettings {
  downloadThreads: number
  fileDetailThreads: number
  openlistQPS: number
  openlistRetryCount: number
  openlistRetryDelay: number
  fileListPageSize: number
}

interface SaveStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}
const http: AxiosStatic | undefined = inject('$http')
const checkIsMobile = ref(isMobile())
const loading = ref(false)
const saveStatus = ref<SaveStatus | null>(null)

// 表单数据
const formData = reactive<ThreadSettings>({
  downloadThreads: 1,
  fileDetailThreads: 3,
  openlistQPS: 2,
  openlistRetryCount: 1,
  openlistRetryDelay: 30,
  fileListPageSize: 1150,
})

// 页面挂载时获取当前设置
onMounted(async () => {
  await fetchThreadSettings()
})

// 获取线程设置
async function fetchThreadSettings() {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/setting/threads`)

    formData.downloadThreads = response?.data.data.download_threads
    formData.fileDetailThreads = response?.data.data.file_detail_threads
    formData.openlistQPS = response?.data.data.openlist_qps
    formData.openlistRetryCount = response?.data.data.openlist_retry
    formData.openlistRetryDelay = response?.data.data.openlist_retry_delay
    formData.fileListPageSize = response?.data.data.file_list_page_size || 1150
  } catch (error) {
    console.error('获取线程设置失败：', error)
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
      file_detail_threads: formData.fileDetailThreads,
      openlist_qps: formData.openlistQPS,
      openlist_retry: formData.openlistRetryCount,
      openlist_retry_delay: formData.openlistRetryDelay,
      file_list_page_size: formData.fileListPageSize,
    }

    await http?.post(`${SERVER_URL}/setting/threads`, payload)

    saveStatus.value = {
      title: '保存成功',
      type: 'success',
      description: '线程设置已成功保存',
    }

    // 3 秒后清除状态提示
    setTimeout(() => {
      saveStatus.value = null
    }, 3000)
  } catch (error) {
    console.error('保存线程设置失败：', error)
    saveStatus.value = {
      title: '保存失败',
      type: 'error',
      description: '保存线程设置失败，请稍后重试',
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
