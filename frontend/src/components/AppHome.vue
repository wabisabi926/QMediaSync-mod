<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject, onMounted, ref } from 'vue'
import {} from '@/utils/timeUtils'

interface VersionInfo {
  version: string
  date: string
}

interface QueueStatus {
  download_status: string
  upload_status: string
  download_active_len: number
  upload_active_len: number
  download_wait_len: number
  upload_wait_len: number
}

const http: AxiosStatic | undefined = inject('$http')
const versionInfo = ref<VersionInfo | null>(null)
const queueStatus = ref<QueueStatus | null>(null)
const versionLoading = ref(true)
const queueLoading = ref(true)

// 获取队列状态文本
const getQueueStatusText = (status: string): string => {
  switch (status) {
    case 'active':
      return '运行中'
    case 'idle':
      return '等待中'
    case 'paused':
      return '暂停'
    case 'error':
      return '错误'
    default:
      return status || '未知'
  }
}

// 获取队列状态样式类
const getQueueStatusClass = (status: string): string => {
  switch (status) {
    case 'active':
      return 'status-running'
    case 'idle':
      return 'status-idle'
    case 'paused':
      return 'status-paused'
    case 'error':
      return 'status-error'
    default:
      return 'status-unknown'
  }
}

// 加载系统版本信息
const loadVersionInfo = async () => {
  try {
    versionLoading.value = true
    const response = await http?.get(`${SERVER_URL}/version`)
    if (response && response.data) {
      versionInfo.value = response.data
    } else {
      versionInfo.value = null
    }
  } catch (error) {
    console.error('加载系统版本信息错误:', error)
    versionInfo.value = null
  } finally {
    versionLoading.value = false
  }
}

// 加载上传下载队列状态
const loadQueueStatus = async () => {
  try {
    queueLoading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/status`)

    if (response?.data.code === 200 && response.data.data) {
      queueStatus.value = response.data.data
    } else {
      queueStatus.value = null
    }
  } catch (error) {
    console.error('加载队列状态错误:', error)
    queueStatus.value = null
  } finally {
    queueLoading.value = false
  }
}

onMounted(() => {
  loadVersionInfo()
  loadQueueStatus()
})
</script>
<template>
  <div class="home-container">
    <!-- 账号信息和队列状态行 -->
    <el-row :gutter="20" class="top-row">
      <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
        <!-- 系统版本信息卡片 -->
        <el-card class="version-card" shadow="hover" v-loading="versionLoading">
          <template #header>
            <h2 class="card-title">系统信息</h2>
            <p class="card-subtitle">当前系统版本和编译信息</p>
          </template>

          <div v-if="versionInfo" class="version-info">
            <div class="version-item">
              <span class="version-label">系统版本:</span>
              <span class="version-value">{{ versionInfo.version }}</span>
            </div>
            <div class="version-item">
              <span class="version-label">编译时间:</span>
              <span class="version-value">{{ versionInfo.date }}</span>
            </div>
          </div>

          <div v-else class="no-version">
            <el-empty description="暂未获取到系统版本信息" />
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
        <!-- 上传下载状态卡片 -->
        <el-card class="queue-card" shadow="hover" v-loading="queueLoading">
          <template #header>
            <h2 class="card-title">上传下载状态</h2>
            <p class="card-subtitle">当前队列和任务状态</p>
          </template>

          <div v-if="queueStatus" class="queue-info">
            <el-row :gutter="20">
              <el-col :span="12">
                <div class="queue-section">
                  <h3 class="queue-section-title">下载状态</h3>
                  <div class="queue-stats">
                    <div class="stat-item">
                      <span class="stat-label">当前状态:</span>
                      <span
                        class="stat-value"
                        :class="getQueueStatusClass(queueStatus.download_status)"
                      >
                        {{ getQueueStatusText(queueStatus.download_status) }}
                      </span>
                    </div>
                    <div class="stat-item">
                      <span class="stat-label">活跃任务:</span>
                      <span class="stat-value active-count">{{
                        queueStatus.download_active_len
                      }}</span>
                    </div>
                    <div class="stat-item">
                      <span class="stat-label">等待任务:</span>
                      <span class="stat-value wait-count">{{ queueStatus.download_wait_len }}</span>
                    </div>
                  </div>
                </div>
              </el-col>
              <el-col :span="12">
                <div class="queue-section">
                  <h3 class="queue-section-title">上传状态</h3>
                  <div class="queue-stats">
                    <div class="stat-item">
                      <span class="stat-label">当前状态:</span>
                      <span
                        class="stat-value"
                        :class="getQueueStatusClass(queueStatus.upload_status)"
                      >
                        {{ getQueueStatusText(queueStatus.upload_status) }}
                      </span>
                    </div>
                    <div class="stat-item">
                      <span class="stat-label">活跃任务:</span>
                      <span class="stat-value active-count">{{
                        queueStatus.upload_active_len
                      }}</span>
                    </div>
                    <div class="stat-item">
                      <span class="stat-label">等待任务:</span>
                      <span class="stat-value wait-count">{{ queueStatus.upload_wait_len }}</span>
                    </div>
                  </div>
                </div>
              </el-col>
            </el-row>
          </div>

          <div v-else class="no-queue">
            <el-empty description="暂未获取到队列状态信息" />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.home-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

/* 顶部行样式 */
.top-row {
  margin-bottom: 0;
}

.top-row .el-col {
  margin-bottom: 20px;
}

.account-card,
.queue-card,
.version-card,
.intro-card {
  width: 100%;
  max-width: none;
  margin: 0;
  height: 100%;
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

/* 导航链接样式 */
.navigation-links {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 16px;
}

.nav-link-btn {
  flex: 1;
  min-width: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

/* 115账号信息样式 */
.account-info {
  margin-top: 16px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.info-label {
  font-size: 14px;
  font-weight: 600;
  color: #606266;
}

.info-value {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
}

.member-vip {
  color: #f56c6c !important;
  font-weight: 600;
}

.member-normal {
  color: #909399;
}

.expire-normal {
  color: #67c23a;
}

.expire-notice {
  color: #e6a23c;
}

.expire-warning {
  color: #f56c6c;
}

.expire-expired {
  color: #f56c6c;
  font-weight: 600;
}

.expire-unknown {
  color: #909399;
}

.storage-progress {
  margin-top: 16px;
}

.no-account {
  padding: 40px 20px;
  text-align: center;
}

/* 系统版本信息样式 */
.version-info {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.version-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.version-label {
  font-size: 14px;
  font-weight: 600;
  color: #606266;
  margin-right: 16px;
  min-width: 80px;
}

.version-value {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.no-version {
  padding: 40px 20px;
  text-align: center;
}

/* 项目介绍样式 */
.intro-content {
  margin-top: 16px;
  line-height: 1.6;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .top-row .el-col:last-child {
    margin-bottom: 0;
  }

  .account-card,
  .version-card,
  .intro-card {
    margin: 0;
  }

  .card-title {
    font-size: 20px;
  }

  .card-subtitle {
    font-size: 13px;
  }

  .info-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .info-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .info-label {
    font-size: 13px;
  }

  .info-value {
    font-size: 13px;
  }

  .version-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .version-label {
    font-size: 13px;
    margin-right: 0;
    min-width: auto;
  }

  .version-value {
    font-size: 13px;
  }

  .intro-content {
    font-size: 14px;
  }

  .intro-content h2 {
    font-size: 18px;
    margin-top: 20px;
    margin-bottom: 12px;
  }

  .intro-content h2:first-child {
    margin-top: 0;
  }

  .intro-content ol,
  .intro-content ul {
    padding-left: 20px;
  }

  .intro-content li {
    margin-bottom: 8px;
  }

  .intro-content blockquote {
    margin: 10px 0;
    padding: 8px 12px;
    border-left: 3px solid #409eff;
    background-color: #f4f4f5;
    font-size: 13px;
  }

  .intro-content code {
    padding: 2px 4px;
    background-color: #f1f1f1;
    border-radius: 3px;
    font-size: 12px;
  }
}

/* 队列状态卡片样式 */
.queue-card {
  height: 100%;
}

.queue-info {
  margin: 0;
}

.queue-section {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 20px;
  height: 100%;
}

.queue-section-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin: 0 0 16px 0;
  text-align: center;
  border-bottom: 2px solid #e0e6ed;
  padding-bottom: 8px;
}

.queue-stats {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
}

.stat-label {
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

.stat-value {
  font-size: 14px;
  font-weight: 600;
}

/* 队列状态颜色 */
.status-idle {
  color: #909399;
}

.status-running {
  color: #67c23a;
}

.status-paused {
  color: #e6a23c;
}

.status-error {
  color: #f56c6c;
}

.status-unknown {
  color: #909399;
}

.active-count {
  color: #67c23a;
}

.wait-count {
  color: #e6a23c;
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .card-title {
    font-size: 18px;
  }

  .intro-content {
    font-size: 13px;
  }

  .intro-content h2 {
    font-size: 16px;
  }

  .info-item,
  .version-item {
    padding: 10px 12px;
  }

  .queue-section {
    padding: 16px;
  }

  .queue-section-title {
    font-size: 14px;
  }

  .stat-item {
    padding: 6px 10px;
  }

  .stat-label,
  .stat-value {
    font-size: 13px;
  }
}
</style>
