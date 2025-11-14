<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject, onMounted, onUnmounted, ref } from 'vue'
import { } from '@/utils/timeUtils'
import { formatFileSize } from '@/utils/fileSizeUtils'
import MarkdownIt from 'markdown-it'
import 'github-markdown-css'
import { ElMessage } from 'element-plus'
import { CircleCheck } from '@element-plus/icons-vue'

interface VersionInfo {
  version: string
  date: string
}

interface UpdateInfo {
  version: string
  date: string
  note: string
  url: string
  latest?: boolean
  current?: boolean
}


const http: AxiosStatic | undefined = inject('$http')
const versionInfo = ref<VersionInfo | null>(null)
const versionLoading = ref(true)
const refreshTimer = ref<number | null>(null)
const updateList = ref<UpdateInfo[]>([])
const updateLoading = ref(false)
const activeNames = ref<string[]>(['update-0'])
const isUpdating = ref(false) // 是否正在更新中
let progressTimer: NodeJS.Timeout | null = null
const updatingVersion = ref<string>('') // 当前正在更新的版本
const updateProgress = ref({
  progress: 0, // 进度百分比
  total_size: 0, // 总大小字节
  downloaded: 0, // 已下载字节
  status: '' // 状态：downloading-下载中，install-安装中, failed-失败
})
const showUpdateCompleteDialog = ref(false) // 是否显示更新完成弹窗
const countdown = ref(30) // 倒计时秒数
let countdownTimer: NodeJS.Timeout | null = null

// 创建markdown-it实例
const md = new MarkdownIt({
  html: true,
  breaks: true,
  linkify: true
})

// 渲染markdown内容
const renderMarkdown = (content: string): string => {
  return md.render(content || '')
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

// 处理下载按钮点击
const handleDownloadClick = (update: UpdateInfo) => {
  if (!update.url) {
    console.error('下载链接不存在:', update)
    alert('下载链接不存在，请稍后重试')
    return false
  }
  window.open(update.url, '_blank')
  return true
}

// 加载最新版本列表
const loadUpdateList = async () => {
  try {
    updateLoading.value = true
    const response = await http?.get(`${SERVER_URL}/update/last`)
    if (response && response.data && response.data.data) {
      updateList.value = response.data.data.map((item: UpdateInfo) => {
        // 确保url字段存在
        return {
          ...item,
          url: item.url || '',
          // latest: index === 0, // 第一个版本标记为最新版
          // current: versionInfo.value && item.version === versionInfo.value.version // 与当前系统版本匹配的标记为当前版本
        }
      })
      // console.log('版本列表加载成功:', updateList.value)
    } else {
      updateList.value = []
      console.log('未获取到版本列表数据')
    }
  } catch (error) {
    console.error('加载最新版本列表错误:', error)
    updateList.value = []
    // 如果是接口调用失败，使用mock数据进行测试
    console.log('使用模拟数据进行测试')
    updateList.value = [
      {
        version: '1.0.0',
        date: '2023-12-01',
        note: '测试版本',
        url: 'https://github.com/qicfan/qmediasync/releases',
        latest: true
      },
      {
        version: '0.9.9',
        date: '2023-11-15',
        note: '上一个版本',
        url: 'https://github.com/qicfan/qmediasync/releases',
        current: true
      }
    ]
  } finally {
    updateLoading.value = false
  }
}

onMounted(() => {
  loadVersionInfo()
  loadUpdateList().then(() => {
    // 加载完成后检查是否正在更新
    checkUpdateStatusOnLoad()
  })
})

// 页面加载时检查更新状态
const checkUpdateStatusOnLoad = async () => {
  try {
    const response = await http?.get(`${SERVER_URL}/update/progress`)

    if (response && response.data && response.data.code === 200) {
      // 如果正在更新，同步更新状态
      const progressData = response.data.data
      if (progressData && (progressData.progress > 0 || progressData.status === 'downloading' || progressData.status === 'install')) {
        isUpdating.value = true
        updatingVersion.value = progressData.version || '' // 这里可能需要从其他接口获取当前正在更新的版本

        // 更新进度信息
        updateProgress.value = {
          progress: progressData.progress || 0,
          total_size: progressData.total_size || 0,
          downloaded: progressData.downloaded || 0,
          status: progressData.status || 'downloading'
        }

        // 开始轮询进度
        startProgressPolling()
      }
    }
  } catch (error) {
    console.error('检查更新状态错误:', error)
  }
}

// 触发版本更新
const updateToVersion = async (version: string) => {
  // 立即设置更新状态为true，禁用所有更新按钮
  isUpdating.value = true
  updatingVersion.value = version
  // 重置进度信息
  updateProgress.value = {
    progress: 0,
    total_size: 0,
    downloaded: 0,
    status: 'downloading'
  }

  try {
    const response = await http?.post(`${SERVER_URL}/update/to-version`, {
      version: version
    })

    if (response && response.data && response.data.code === 200) {
      // 开始轮询更新进度
      startProgressPolling()
    } else {
      // 如果更新失败，重置状态
      isUpdating.value = false
      updatingVersion.value = ''
      updateProgress.value = {
        progress: 0,
        total_size: 0,
        downloaded: 0,
        status: ''
      }
      ElMessage.error(response?.data.message || '触发版本更新失败')
    }
  } catch (error) {
    console.error('触发版本更新错误:', error)
    // 如果出现异常，重置状态
    isUpdating.value = false
    updatingVersion.value = ''
    updateProgress.value = {
      progress: 0,
      total_size: 0,
      downloaded: 0,
      status: ''
    }
    ElMessage.error('触发版本更新失败')
  }
}

// 开始轮询更新进度
const startProgressPolling = () => {
  // 清除之前的定时器（如果存在）
  if (progressTimer) {
    clearInterval(progressTimer)
  }

  // 立即查询一次进度
  checkUpdateProgress()

  // 设置定时器，每隔3秒查询一次进度
  progressTimer = setInterval(() => {
    checkUpdateProgress()
  }, 1000)
}

// 显示更新完成弹窗并开始倒计时
const showUpdateCompleteNotification = () => {
  showUpdateCompleteDialog.value = true
  countdown.value = 30

  // 清除可能存在的定时器
  if (countdownTimer) {
    clearInterval(countdownTimer)
  }

  // 开始倒计时
  countdownTimer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      if (countdownTimer) {
        clearInterval(countdownTimer)
        countdownTimer = null
      }
      // 刷新页面
      window.location.reload()
    }
  }, 1000)
}

// 手动刷新页面
const manuallyRefresh = () => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  window.location.reload()
}

// 查询更新进度
const checkUpdateProgress = async () => {
  try {
    const response = await http?.get(`${SERVER_URL}/update/progress`)

    if (response && response.data) {
      // 更新进度信息
      if (response.data.data.progress !== undefined) {
        const previousProgress = updateProgress.value.progress
        updateProgress.value.progress = response.data.data.progress

        // 检测进度是否达到100%
        if (previousProgress < 100 && updateProgress.value.progress >= 100) {
          // 显示更新完成弹窗
          showUpdateCompleteNotification()
        }
      }
      if (response.data.data.total_size !== undefined) {
        updateProgress.value.total_size = response.data.data.total_size
      }
      if (response.data.data.downloaded !== undefined) {
        updateProgress.value.downloaded = response.data.data.downloaded
      }

      // 处理status字段
      if (response.data.data.status !== undefined) {
        updateProgress.value.status = response.data.data.status

        // 如果状态为failed，重置所有更新操作并提示用户
        if (response.data.data.status === 'failed') {
          // 清除定时器
          if (progressTimer) {
            clearInterval(progressTimer)
            progressTimer = null
          }

          if (countdownTimer) {
            clearInterval(countdownTimer)
            countdownTimer = null
          }

          // 关闭弹窗
          showUpdateCompleteDialog.value = false

          // 重置更新状态
          isUpdating.value = false
          updatingVersion.value = ''
          updateProgress.value.progress = 0
          updateProgress.value.total_size = 0
          updateProgress.value.downloaded = 0
          updateProgress.value.status = ''

          // 提示用户更新失败
          ElMessage.error({
            message: '更新失败，请稍后重试或手动下载最新版本',
            duration: 5000
          })

          // 刷新版本列表
          setTimeout(() => {
            loadUpdateList()
          }, 1000)

          return // 提前返回，不再执行后续逻辑
        }
      }

      // 如果接口返回code为200，说明正在更新中，保持isUpdating为true
      if (response.data.code !== 200) {
        // 更新完成或失败，清除定时器
        if (progressTimer) {
          clearInterval(progressTimer)
          progressTimer = null
        }

        // 重置更新状态
        setTimeout(() => {
          isUpdating.value = false
          updatingVersion.value = ''
          // 刷新版本列表
          loadUpdateList()
        }, 2000)
      }
    }
  } catch (error) {
    console.error('查询更新进度错误:', error)
  }
}

// 取消更新
const cancelUpdate = async () => {
  try {
    await http?.post(`${SERVER_URL}/update/cancel`)

    // 清除定时器
    if (progressTimer) {
      clearInterval(progressTimer)
      progressTimer = null
    }

    if (countdownTimer) {
      clearInterval(countdownTimer)
      countdownTimer = null
    }

    // 关闭弹窗
    showUpdateCompleteDialog.value = false

    // 重置更新状态
    isUpdating.value = false
    updatingVersion.value = ''
    updateProgress.value = {
      progress: 0,
      total_size: 0,
      downloaded: 0,
      status: ''
    }

    ElMessage.success('已取消更新')

    // 刷新版本列表
    setTimeout(() => {
      loadUpdateList()
    }, 1000)
  } catch (error) {
    console.error('取消更新错误:', error)
    ElMessage.error('取消更新失败，请稍后重试')
  }
}

// 组件卸载时清除定时器
onUnmounted(() => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }

  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }

  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
})
</script>
<template>
  <div class="home-container">
    <!-- 账号信息和队列状态行 -->
    <el-row :gutter="20" class="top-row">
      <el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24">

        <el-card class="version-card" shadow="hover" v-loading="versionLoading">
          <template #header>
            <h2 class="card-title">使用注意事项：</h2>
          </template>
          <div class="notice-content">
            <p class="notice-item">
              <el-text class="mx-1" size="large" type="danger">
                1. 本项目使用115开放平台，所以各方面受限颇多，QPS也不能太高，介意勿用。
              </el-text>
            </p>
            <p class="notice-item">
              <el-text class="mx-1" size="large">
                2. 影片播放、元数据下载、神医助手、本项目的媒体信息提取等全部占用下载额度，请确保这些东西相加不要大于5，否则115会报403错误。
              </el-text>
            </p>
            <p class="notice-item">
              <el-text class="mx-1" size="large">
                3. 请将神医助手的线程数调整到2或者1，如果使用的话。
              </el-text>
            </p>
            <p class="notice-item">
              <el-text class="mx-1" size="large">
                4. 刮削和STRM同步是两个独立的功能，不会联动。刮削可以处理网盘，也可以处理本地文件。
              </el-text>
            </p>
            <p class="notice-item">
              <el-text class="mx-1" size="large">
                5. 有问题请在github提交issue，带上截图或日志，日志请将config/logs目录打包压缩后上传。
              </el-text>
            </p>
            <p class="notice-item">
              <el-text class="mx-1" size="large">
                6. 仓库地址：<a href="https://github.com/qicfan/qmediasync"
                  target="_blank">https://github.com/qicfan/qmediasync</a>
              </el-text>
            </p>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="8" :lg="8" :xl="8">
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

      <el-col :xs="24" :sm="24" :md="16" :lg="16" :xl="16">
        <!-- 赞助版块 -->
        <el-card class="version-card" shadow="hover">
          <template #header>
            <h2 class="card-title">请作者喝杯咖啡</h2>
          </template>
          <img src="https://s.mqfamily.top/alipay_wechat.jpg" alt="请作者喝杯咖啡" class="coffee-image"
            style="max-width: 100%" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 最新版本列表 -->
    <el-row :gutter="20">
      <el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24">
        <el-card class="version-card" shadow="hover" v-loading="updateLoading">
          <template #header>
            <h2 class="card-title">最新版本列表</h2>
            <p class="card-subtitle">最新发布的版本信息， 可以选择要更新的版本</p>
          </template>

          <div v-if="updateList.length > 0" class="update-list">
            <el-collapse v-model="activeNames">
              <el-collapse-item v-for="(update, index) in updateList" :key="index" :name="`update-${index}`">
                <template #title>
                  <div class="version-title-container">
                    <div style="display:flex; flex-wrap: wrap; font-size:14px;">
                      <span>版本 {{ update.version }}</span>
                      <span>({{ update.date }})</span>
                    </div>
                    <div class="version-tags">
                      <el-tag v-if="update.latest" type="success" size="small" class="mr-2">
                        最新版本
                      </el-tag>
                      <el-tag v-if="update.current" type="primary" size="small">
                        当前版本
                      </el-tag>
                    </div>
                  </div>
                </template>
                <div class="update-item">
                  <div class="update-content">
                    <h4>更新内容:</h4>
                    <div class="update-note markdown-body" v-html="renderMarkdown(update.note)"></div>
                  </div>
                  <div class="update-actions">
                    <el-button v-if="!update.current" type="info" size="small" :href="update.url || '#'" target="_blank"
                      @click="handleDownloadClick(update)">
                      去下载
                    </el-button>
                    <el-button v-if="!update.current" type="primary" size="large"
                      @click="updateToVersion(update.version)" :disabled="isUpdating">
                      更新到此版本
                    </el-button>
                  </div>

                  <!-- 下载进度条 -->
                  <div v-if="isUpdating && update.version === updatingVersion" class="update-progress">
                    <div style="display: flex; align-items: center; gap: 16px;">
                      <el-progress :percentage="updateProgress.progress"
                        :status="updateProgress.progress < 100 ? 'primary' : 'success'" :stroke-width="24"
                        :text-inside="true" style="flex: 1;"></el-progress>
                      <el-button type="danger" @click="cancelUpdate" :loading="updateProgress.progress >= 100">
                        取消更新
                      </el-button>
                    </div>
                    <div class="progress-info">
                      <span>已下载: {{ formatFileSize(updateProgress.downloaded) }}</span>
                      <span v-if="updateProgress.total_size > 0">
                        / 总大小: {{ formatFileSize(updateProgress.total_size) }}
                      </span>
                      <span v-if="updateProgress.status">
                        / 状态: {{ updateProgress.status === 'downloading' ? '下载中' : updateProgress.status === 'install' ?
                          '安装中' : updateProgress.status }}
                      </span>
                    </div>
                  </div>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>

          <div v-else class="no-update">
            <el-empty description="暂未获取到版本更新信息" />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>

  <!-- 更新完成弹窗 -->
  <el-dialog v-model="showUpdateCompleteDialog" title="正在安装更新" class="update-complete-dialog"
    :close-on-click-modal="false" :close-on-press-escape="false" show-close="false" :destroy-on-close="true">
    <div class="dialog-content">
      <el-icon>
        <CircleCheck />
      </el-icon>
      <h3>安装包已下载，正在更新中</h3>
      <p>系统将在 <strong>{{ countdown }}</strong> 秒后自动刷新页面</p>
      <div class="dialog-tips">
        <p>提示：刷新页面后，新版本将生效。如未生效，请手动刷新或手动下载最新版本，如果是docker可以更新镜像</p>
      </div>
    </div>
    <template #footer>
      <div class="dialog-footer">
        <el-button type="primary" @click="manuallyRefresh">
          立即刷新
        </el-button>
      </div>
    </template>
  </el-dialog>
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

/* 最新版本列表样式 */
.update-list {
  margin-top: 16px;
}

.update-item {
  padding: 12px 0;
}

.update-content h4 {
  font-size: 16px;
  color: #303133;
  margin: 0 0 8px 0;
}

.update-note {
  background: #f8f9fa;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 16px;
  font-size: 14px;
  line-height: 1.6;
  color: #606266;
  margin: 0 0 16px 0;
  overflow-x: auto;
}

/* 为markdown内容添加一些额外样式 */
.update-note :deep(.markdown-body) {
  font-size: 14px;
  line-height: 1.6;
}

.update-note :deep(.markdown-body pre) {
  background-color: #f6f8fa;
  border-radius: 3px;
  padding: 16px;
  overflow: auto;
}

.update-note :deep(.markdown-body code) {
  background-color: #f1f1f1;
  border-radius: 3px;
  padding: 2px 4px;
  font-size: 0.9em;
}

.update-note :deep(.markdown-body pre code) {
  background-color: transparent;
  padding: 0;
}

.update-note :deep(.markdown-body a) {
  color: #0366d6;
  text-decoration: none;
}

.update-note :deep(.markdown-body a:hover) {
  text-decoration: underline;
}

.update-note :deep(.markdown-body h1),
.update-note :deep(.markdown-body h2),
.update-note :deep(.markdown-body h3),
.update-note :deep(.markdown-body h4),
.update-note :deep(.markdown-body h5),
.update-note :deep(.markdown-body h6) {
  margin-top: 24px;
  margin-bottom: 16px;
  font-weight: 600;
  line-height: 1.25;
}

.update-note :deep(.markdown-body p) {
  margin-top: 0;
  margin-bottom: 16px;
}

.update-note :deep(.markdown-body ul),
.update-note :deep(.markdown-body ol) {
  padding-left: 2em;
  margin-top: 0;
  margin-bottom: 16px;
}

.update-note :deep(.markdown-body li) {
  margin-bottom: 4px;
}

.update-note :deep(.markdown-body blockquote) {
  padding: 0 1em;
  color: #6a737d;
  border-left: 0.25em solid #dfe2e5;
  margin: 0 0 16px 0;
}

.update-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  ;
  gap: 12px;
  margin-bottom: 16px;
}

.update-progress {
  margin-top: 16px;
}

.progress-info {
  display: flex;
  justify-content: center;
  align-items: center;
  margin-top: 8px;
  font-size: 14px;
  color: #606266;
  gap: 8px;
}

/* 更新完成弹窗样式 */
.update-complete-dialog {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  margin: 0;
  width: 100% !important;
  max-width: none !important;
  background-color: rgba(0, 0, 0, 0.8);
}

.update-complete-dialog :deep(.el-dialog) {
  width: 500px;
  max-width: 90vw;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.dialog-content {
  text-align: center;
  padding: 20px 0;
}

.success-icon {
  font-size: 24px;
  color: #67c23a;
  margin-bottom: 20px;
}

.dialog-content h3 {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 16px;
  color: #303133;
}

.dialog-content p {
  font-size: 16px;
  color: #606266;
  margin-bottom: 20px;
}

.dialog-tips {
  padding: 12px 20px;
  background-color: #f0f9ff;
  border: 1px solid #e6f7ff;
  border-radius: 6px;
  margin-top: 20px;
}

.dialog-tips p {
  font-size: 14px;
  color: #909399;
  margin: 0;
}

.dialog-footer {
  display: flex;
  justify-content: center;
  gap: 16px;
  padding: 20px;
  border-top: 1px solid #ebeef5;
}

.dialog-footer .el-button {
  min-width: 120px;
}

.version-title-container {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  flex-wrap: wrap
}

.version-tags {
  display: flex;
  gap: 8px;
}

.mr-2 {
  margin-right: 8px;
}

.no-update {
  padding: 40px 20px;
  text-align: center;
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
