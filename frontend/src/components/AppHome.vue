<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject, onMounted, onUnmounted, ref, computed } from 'vue'
import { formatDateTime } from '@/utils/timeUtils'
import { formatFileSize } from '@/utils/fileSizeUtils'
import MarkdownIt from 'markdown-it'
import 'github-markdown-css'
import { ElMessage } from 'element-plus'
import { CircleCheck, Document } from '@element-plus/icons-vue'
import AppLogViewer from './AppLogViewer.vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, LineChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
  DataZoomComponent
} from 'echarts/components'

use([
  CanvasRenderer,
  BarChart,
  LineChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
  DataZoomComponent
])

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

interface QueueStats {
  avg_response_time_ms: number
  is_throttled: boolean
  last_throttle_time: string | null
  qph_count: number
  qpm_count: number
  qps_count: number
  throttle_recover_time: string
  throttle_wait_time: string
  throttled_count: number
  throttled_elapsed_time: string
  throttled_remaining_time: string
  time_window_seconds: number
  total_requests: number
}

interface HourlyStat {
  hour_ts: number
  total_requests: number
  throttled_requests: number
  avg_duration: string
}

interface HourlyStatsData {
  start_date: string
  end_date: string
  total_requests: number
  total_throttled: number
  hourly_stats: HourlyStat[]
  query_time_range_days: number
}


const http: AxiosStatic | undefined = inject('$http')
const versionInfo = ref<VersionInfo | null>(null)
const versionLoading = ref(true)
const refreshTimer = ref<number | null>(null)
const updateList = ref<UpdateInfo[]>([])
const updateLoading = ref(false)
const activeNames = ref<string[]>(['update-0'])
const isUpdating = ref(false) // 是否正在更新中
let progressTimer: number | null = null
const updatingVersion = ref<string>('') // 当前正在更新的版本
const updateProgress = ref({
  progress: 0, // 进度百分比
  total_size: 0, // 总大小字节
  downloaded: 0, // 已下载字节
  status: '' // 状态：downloading-下载中，install-安装中, failed-失败
})
const showUpdateCompleteDialog = ref(false) // 是否显示更新完成弹窗
const countdown = ref(30) // 倒计时秒数
let countdownTimer: number | null = null

// 115接口请求统计
const queueStats = ref<QueueStats | null>(null)
const queueStatsLoading = ref(false)
let queueStatsTimer: number | null = null

// 每小时请求统计
const hourlyStats = ref<HourlyStatsData | null>(null)
const hourlyStatsLoading = ref(false)

// 日志弹窗相关
const showLogDialog = ref(false) // 是否显示日志弹窗
const logViewerRef = ref<InstanceType<typeof AppLogViewer> | null>(null) // 日志查看器引用

// 处理日志弹窗关闭事件
const handleLogDialogClose = () => {
  // 断开日志连接
  if (logViewerRef.value) {
    // 调用日志查看器的disconnect方法
    logViewerRef.value.disconnect()
  }
}

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
const loadUpdateList = async (force = false) => {
  try {
    updateLoading.value = true
    let url = `${SERVER_URL}/update/last`;
    if (force) {
      url += '?force=1';
    }
    const response = await http?.get(url)
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
  } finally {
    updateLoading.value = false
  }
}

// 加载115接口请求统计
const loadQueueStats = async () => {
  try {
    queueStatsLoading.value = true
    const response = await http?.get(`${SERVER_URL}/115/queue/stats`)
    if (response && response.data && response.data.code === 200) {
      queueStats.value = response.data.data
    } else {
      queueStats.value = null
    }
  } catch (error) {
    console.error('加载115接口请求统计错误:', error)
    queueStats.value = null
  } finally {
    queueStatsLoading.value = false
  }
}

// 开始定时刷新统计数据
const startQueueStatsPolling = () => {
  if (queueStatsTimer) {
    clearInterval(queueStatsTimer)
  }

  // 每3秒刷新一次
  queueStatsTimer = setInterval(() => {
    loadQueueStats()
  }, 3000)
}

// 加载每小时请求统计
const loadHourlyStats = async () => {
  try {
    hourlyStatsLoading.value = true
    const response = await http?.get(`${SERVER_URL}/115/stats/hourly`)
    if (response && response.data && response.data.code === 200) {
      hourlyStats.value = response.data.data
    } else {
      hourlyStats.value = null
    }
  } catch (error) {
    console.error('加载每小时请求统计错误:', error)
    hourlyStats.value = null
  } finally {
    hourlyStatsLoading.value = false
  }
}

// 图表配置
const chartOption = computed(() => {
  if (!hourlyStats.value || !hourlyStats.value.hourly_stats) {
    return {}
  }

  const hours = hourlyStats.value.hourly_stats.map(item => formatDateTime(item.hour_ts))
  const requestCounts = hourlyStats.value.hourly_stats.map(item => item.total_requests)
  const throttledCounts = hourlyStats.value.hourly_stats.map(item => item.throttled_requests)
  const avgDurations = hourlyStats.value.hourly_stats.map(item => {
    const value = parseFloat(item.avg_duration)
    return isNaN(value) ? 0 : Math.round(value)
  })

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['请求数', '限流次数', '平均响应时间(ms)'],
      top: 10
    },
    grid: {
      left: '50px',
      right: '4%',
      bottom: '60px',
      top: '60px'
    },
    xAxis: {
      type: 'category',
      data: hours,
      axisLabel: {
        rotate: 45,
        interval: 0,
        fontSize: 10
      }
    },
    yAxis: [
      {
        type: 'value',
        name: '次数',
        position: 'left'
      },
      {
        type: 'value',
        name: '响应时间(ms)',
        position: 'right'
      }
    ],
    series: [
      {
        name: '请求数',
        type: 'bar',
        yAxisIndex: 0,
        data: requestCounts,
        itemStyle: {
          color: '#409eff'
        }
      },
      {
        name: '限流次数',
        type: 'bar',
        yAxisIndex: 0,
        data: throttledCounts,
        itemStyle: {
          color: '#f56c6c'
        }
      },
      {
        name: '平均响应时间(ms)',
        type: 'line',
        yAxisIndex: 1,
        data: avgDurations,
        itemStyle: {
          color: '#67c23a'
        },
        lineStyle: {
          width: 2
        },
        smooth: true
      }
    ],
    dataZoom: [
      {
        type: 'inside',
        start: 0,
        end: 100
      },
      {
        start: 0,
        end: 100
      }
    ]
  }
})

onMounted(() => {
  loadVersionInfo()
  loadUpdateList().then(() => {
    // 加载完成后检查是否正在更新
    checkUpdateStatusOnLoad()
  })
  loadQueueStats()
  startQueueStatsPolling()
  loadHourlyStats()
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

  if (queueStatsTimer) {
    clearInterval(queueStatsTimer)
    queueStatsTimer = null
  }
})
</script>
<template>
  <div class="home-container">
    <div class="header-section">
      <div class="header-title">
        <h1>控制台</h1>
        <p>系统运行状态监控与管理</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="showLogDialog = true" :icon="Document" round>
          运行日志
        </el-button>
      </div>
    </div>

    <div class="stats-section">
      <div class="stats-row">
        <div class="stats-card-main" v-loading="queueStatsLoading">
          <div class="stats-card-header">
            <div class="stats-card-title">
              <span class="title-icon">📊</span>
              <span>115接口监控</span>
            </div>
            <div class="status-badge" :class="queueStats?.is_throttled ? 'status-warning' : 'status-success'">
              {{ queueStats?.is_throttled ? '限流中' : '运行正常' }}
            </div>
          </div>

          <div v-if="queueStats" class="stats-content">
            <div v-if="queueStats.is_throttled" class="throttle-warning">
              <div class="throttle-icon">⚠️</div>
              <div class="throttle-details">
                <div class="throttle-item">
                  <span class="label">等待时间</span>
                  <span class="value">{{ queueStats.throttle_wait_time }}</span>
                </div>
                <div class="throttle-item">
                  <span class="label">已过时间</span>
                  <span class="value">{{ queueStats.throttled_elapsed_time }}</span>
                </div>
                <div class="throttle-item">
                  <span class="label">剩余时间</span>
                  <span class="value">{{ queueStats.throttled_remaining_time }}</span>
                </div>
              </div>
            </div>

            <div class="metrics-grid">
              <div class="metric-item" :class="{ 'metric-warning': queueStats.qps_count > 3 }">
                <div class="metric-value">{{ queueStats.qps_count }}</div>
                <div class="metric-label">QPS</div>
              </div>
              <div class="metric-item">
                <div class="metric-value">{{ queueStats.qpm_count }}</div>
                <div class="metric-label">QPM</div>
              </div>
              <div class="metric-item">
                <div class="metric-value">{{ queueStats.qph_count }}</div>
                <div class="metric-label">QPH</div>
              </div>
              <div class="metric-item">
                <div class="metric-value">{{ queueStats.avg_response_time_ms }}</div>
                <div class="metric-label">响应(ms)</div>
              </div>
              <div class="metric-item" :class="{ 'metric-danger': queueStats.throttled_count > 0 }">
                <div class="metric-value">{{ queueStats.throttled_count }}</div>
                <div class="metric-label">限流次数</div>
              </div>
            </div>
          </div>

          <div v-else class="empty-state">
            <el-empty description="暂无统计数据" :image-size="60" />
          </div>
        </div>

        <div class="chart-card" v-loading="hourlyStatsLoading">
          <div class="chart-header">
            <div class="chart-title">
              <span class="title-icon">📈</span>
              <span>请求趋势</span>
              <span class="chart-period">{{ hourlyStats?.start_date }} ~ {{ hourlyStats?.end_date }}</span>
            </div>
            <el-button type="primary" size="small" @click="loadHourlyStats" :loading="hourlyStatsLoading" round>
              刷新
            </el-button>
          </div>

          <div v-if="hourlyStats" class="chart-content">
            <div class="chart-summary">
              <div class="summary-item">
                <div class="summary-value">{{ hourlyStats.total_requests }}</div>
                <div class="summary-label">总请求</div>
              </div>
              <div class="summary-item" :class="{ 'summary-danger': hourlyStats.total_throttled > 0 }">
                <div class="summary-value">{{ hourlyStats.total_throttled }}</div>
                <div class="summary-label">总限流</div>
              </div>
            </div>
            <div class="chart-wrapper">
              <v-chart class="chart" :option="chartOption" autoresize />
            </div>
          </div>

          <div v-else class="empty-state">
            <el-empty description="暂无统计数据" :image-size="60" />
          </div>
        </div>
      </div>
    </div>

    <div class="info-section">
      <div class="info-grid">
        <div class="info-card system-info" v-loading="versionLoading">
          <div class="info-card-header">
            <span class="info-icon">⚙️</span>
            <span>系统信息</span>
          </div>
          <div v-if="versionInfo" class="info-content">
            <div class="info-row">
              <span class="info-label">版本</span>
              <span class="info-value version-tag">{{ versionInfo.version }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">编译时间</span>
              <span class="info-value">{{ versionInfo.date }}</span>
            </div>
          </div>
          <div v-else class="empty-state-small">
            <el-empty description="暂无信息" :image-size="40" />
          </div>
        </div>

        <div class="info-card sponsor-card">
          <div class="info-card-header">
            <span class="info-icon">☕</span>
            <span>支持作者</span>
          </div>
          <div class="sponsor-content">
            <img src="https://s.mqfamily.top/alipay_wechat.jpg" alt="请作者喝杯咖啡" />
          </div>
        </div>

        <div class="info-card notice-card">
          <div class="info-card-header">
            <span class="info-icon">📝</span>
            <span>使用须知</span>
          </div>
          <div class="notice-list">
            <div class="notice-item notice-important">
              <span class="notice-number">1</span>
              <span>本项目使用115开放平台，QPS受限，介意勿用</span>
            </div>
            <div class="notice-item">
              <span class="notice-number">2</span>
              <span>播放、下载、媒体提取等操作并发总和勿超5</span>
            </div>
            <div class="notice-item">
              <span class="notice-number">3</span>
              <span>神医助手线程数建议调整为1或2</span>
            </div>
            <div class="notice-item">
              <span class="notice-number">4</span>
              <span>刮削和STRM同步为独立功能</span>
            </div>
            <div class="notice-item">
              <span class="notice-number">5</span>
              <span>问题请在
                <a href="https://github.com/qicfan/qmediasync" target="_blank">GitHub</a> 提交issue
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="update-section">
      <div class="section-header">
        <div class="section-title">
          <span class="title-icon">🚀</span>
          <span>版本更新</span>
        </div>
        <el-button type="primary" size="small" @click="loadUpdateList(true)" :loading="updateLoading" round>
          刷新
        </el-button>
      </div>

      <div v-if="updateList.length > 0" class="update-list">
        <el-collapse v-model="activeNames" class="update-collapse">
          <el-collapse-item v-for="(update, index) in updateList" :key="index" :name="`update-${index}`">
            <template #title>
              <div class="update-title-row">
                <div class="update-version">
                  <span class="version-number">v{{ update.version }}</span>
                  <span class="version-date">{{ update.date }}</span>
                </div>
                <div class="update-tags">
                  <el-tag v-if="update.latest" type="success" size="small" effect="dark">最新</el-tag>
                  <el-tag v-if="update.current" type="primary" size="small" effect="dark">当前</el-tag>
                </div>
              </div>
            </template>
            <div class="update-detail">
              <div class="update-note markdown-body" v-html="renderMarkdown(update.note)"></div>
              <div class="update-actions" v-if="!update.current">
                <el-button type="default" size="small" @click="handleDownloadClick(update)" round>
                  手动下载
                </el-button>
                <el-button type="primary" size="small" @click="updateToVersion(update.version)" :disabled="isUpdating" round>
                  在线更新
                </el-button>
              </div>

              <div v-if="isUpdating && update.version === updatingVersion" class="update-progress">
                <el-progress :percentage="updateProgress.progress" :stroke-width="8" :show-text="false" />
                <div class="progress-info">
                  <span>{{ formatFileSize(updateProgress.downloaded) }} / {{ formatFileSize(updateProgress.total_size) }}</span>
                  <span>{{ updateProgress.status === 'downloading' ? '下载中' : updateProgress.status === 'install' ? '安装中' : '' }}</span>
                </div>
                <el-button type="danger" size="small" @click="cancelUpdate" round>
                  取消
                </el-button>
              </div>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>

      <div v-else class="empty-state">
        <el-empty description="暂无版本信息" :image-size="80" />
      </div>
    </div>
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

  <!-- 日志查看弹窗 -->
  <el-dialog v-model="showLogDialog" title="运行日志" class="log-dialog" :fullscreen="true" :close-on-click-modal="true"
    :close-on-press-escape="true" show-close="true" :destroy-on-close="true" @close="handleLogDialogClose">
    <div class="log-dialog-content">
      <AppLogViewer ref="logViewerRef" log-path="app.log" :is-real-time="true" />
    </div>
  </el-dialog>
</template>

<style scoped>
.home-container {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 0;
}

.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px;
  color: white;
}

.header-title h1 {
  margin: 0 0 4px 0;
  font-size: 28px;
  font-weight: 700;
}

.header-title p {
  margin: 0;
  font-size: 14px;
  opacity: 0.9;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.stats-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.stats-row {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: 20px;
}

.stats-card-main,
.chart-card {
  background: white;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #f0f0f0;
}

.stats-card-header,
.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.stats-card-title,
.chart-title,
.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.title-icon {
  font-size: 20px;
}

.chart-period {
  font-size: 12px;
  color: #909399;
  font-weight: 400;
  margin-left: 8px;
}

.status-badge {
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
}

.status-success {
  background: #e8f5e9;
  color: #2e7d32;
}

.status-warning {
  background: #fff3e0;
  color: #e65100;
  animation: pulse-bg 2s ease-in-out infinite;
}

@keyframes pulse-bg {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.throttle-warning {
  display: flex;
  gap: 16px;
  padding: 16px;
  background: linear-gradient(135deg, #fff8e1 0%, #ffecb3 100%);
  border-radius: 12px;
  margin-bottom: 16px;
}

.throttle-icon {
  font-size: 24px;
}

.throttle-details {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  flex: 1;
}

.throttle-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.throttle-item .label {
  font-size: 12px;
  color: #909399;
}

.throttle-item .value {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(80px, 1fr));
  gap: 12px;
}

.metric-item {
  text-align: center;
  padding: 16px 8px;
  background: #f8f9fa;
  border-radius: 12px;
  transition: all 0.3s ease;
}

.metric-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.metric-item.metric-warning {
  background: linear-gradient(135deg, #fff8e1 0%, #ffe082 100%);
}

.metric-item.metric-danger {
  background: linear-gradient(135deg, #ffebee 0%, #ffcdd2 100%);
}

.metric-value {
  font-size: 24px;
  font-weight: 700;
  color: #303133;
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
}

.metric-item.metric-warning .metric-value {
  color: #e65100;
}

.metric-item.metric-danger .metric-value {
  color: #c62828;
}

.metric-label {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.chart-summary {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.summary-item {
  flex: 1;
  padding: 16px;
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e7ed 100%);
  border-radius: 12px;
  text-align: center;
}

.summary-item.summary-danger {
  background: linear-gradient(135deg, #ffebee 0%, #ffcdd2 100%);
}

.summary-value {
  font-size: 28px;
  font-weight: 700;
  color: #303133;
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
}

.summary-item.summary-danger .summary-value {
  color: #c62828;
}

.summary-label {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.chart-wrapper {
  width: 100%;
  height: 350px;
}

.chart {
  width: 100%;
  height: 100%;
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
}

.empty-state-small {
  padding: 20px;
  text-align: center;
}

.info-section {
  display: flex;
  flex-direction: column;
}

.info-grid {
  display: grid;
  grid-template-columns: 280px 280px 1fr;
  gap: 20px;
}

.info-card {
  background: white;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #f0f0f0;
}

.info-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.info-icon {
  font-size: 18px;
}

.info-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.info-label {
  font-size: 13px;
  color: #909399;
}

.info-value {
  font-size: 13px;
  color: #303133;
  font-weight: 500;
}

.version-tag {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.sponsor-content {
  display: flex;
  justify-content: center;
}

.sponsor-content img {
  max-width: 100%;
  border-radius: 8px;
}

.notice-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.notice-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}

.notice-item.notice-important {
  color: #c62828;
}

.notice-number {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  background: #f0f0f0;
  border-radius: 50%;
  font-size: 11px;
  font-weight: 600;
  color: #606266;
  flex-shrink: 0;
}

.notice-item.notice-important .notice-number {
  background: #ffebee;
  color: #c62828;
}

.notice-item a {
  color: #409eff;
  text-decoration: none;
}

.notice-item a:hover {
  text-decoration: underline;
}

.update-section {
  background: white;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #f0f0f0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.update-collapse {
  border: none;
}

.update-collapse :deep(.el-collapse-item__header) {
  background: #f8f9fa;
  border-radius: 12px;
  padding: 0 16px;
  margin-bottom: 8px;
  border: none;
  height: 56px;
}

.update-collapse :deep(.el-collapse-item__wrap) {
  border: none;
}

.update-collapse :deep(.el-collapse-item__content) {
  padding: 16px;
}

.update-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.update-version {
  display: flex;
  align-items: center;
  gap: 12px;
}

.version-number {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.version-date {
  font-size: 13px;
  color: #909399;
}

.update-tags {
  display: flex;
  gap: 8px;
}

.update-detail {
  background: #fafafa;
  border-radius: 12px;
  padding: 16px;
}

.update-note {
  background: white;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
  font-size: 14px;
  line-height: 1.6;
  color: #606266;
}

.update-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.update-progress {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.update-progress .el-progress {
  flex: 1;
}

.progress-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 12px;
  color: #909399;
  min-width: 120px;
}

.log-dialog {
  display: flex;
  align-items: center;
  justify-content: center;
}

.log-dialog-content {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.log-dialog-content :deep(.el-dialog__body) {
  padding: 0;
  overflow: hidden;
  height: calc(100% - 60px);
}

.log-dialog-content :deep(.el-dialog__header) {
  padding: 10px 20px;
  border-bottom: 1px solid #ebeef5;
}

.update-complete-dialog :deep(.el-dialog) {
  width: 500px;
  max-width: 90vw;
  border-radius: 16px;
}

.dialog-content {
  text-align: center;
  padding: 30px 20px;
}

.dialog-content .el-icon {
  font-size: 48px;
  color: #67c23a;
  margin-bottom: 20px;
}

.dialog-content h3 {
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 12px;
  color: #303133;
}

.dialog-content p {
  font-size: 15px;
  color: #606266;
  margin-bottom: 16px;
}

.dialog-tips {
  padding: 12px 16px;
  background: #f0f9ff;
  border-radius: 8px;
}

.dialog-tips p {
  font-size: 13px;
  color: #909399;
  margin: 0;
}

.dialog-footer {
  display: flex;
  justify-content: center;
  padding: 16px;
  border-top: 1px solid #ebeef5;
}

.update-note :deep(.markdown-body) {
  font-size: 14px;
  line-height: 1.6;
}

.update-note :deep(.markdown-body pre) {
  background-color: #f6f8fa;
  border-radius: 6px;
  padding: 12px;
  overflow: auto;
}

.update-note :deep(.markdown-body code) {
  background-color: #f1f1f1;
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 13px;
}

.update-note :deep(.markdown-body pre code) {
  background-color: transparent;
  padding: 0;
}

.update-note :deep(.markdown-body a) {
  color: #409eff;
  text-decoration: none;
}

.update-note :deep(.markdown-body a:hover) {
  text-decoration: underline;
}

.update-note :deep(.markdown-body ul),
.update-note :deep(.markdown-body ol) {
  padding-left: 1.5em;
  margin: 8px 0;
}

.update-note :deep(.markdown-body li) {
  margin-bottom: 4px;
}

@media (max-width: 1200px) {
  .stats-row {
    grid-template-columns: 1fr;
  }

  .info-grid {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 768px) {
  .header-section {
    flex-direction: column;
    gap: 16px;
    text-align: center;
    padding: 16px;
  }

  .header-title h1 {
    font-size: 24px;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }

  .chart-wrapper {
    height: 280px;
  }

  .metrics-grid {
    grid-template-columns: repeat(3, 1fr);
  }

  .metric-value {
    font-size: 20px;
  }

  .summary-value {
    font-size: 24px;
  }
}

@media (max-width: 480px) {
  .header-title h1 {
    font-size: 20px;
  }

  .header-title p {
    font-size: 12px;
  }

  .stats-card-main,
  .chart-card,
  .info-card,
  .update-section {
    padding: 16px;
    border-radius: 12px;
  }

  .metrics-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }

  .metric-item {
    padding: 12px 6px;
  }

  .metric-value {
    font-size: 18px;
  }

  .metric-label {
    font-size: 10px;
  }

  .chart-summary {
    flex-direction: column;
    gap: 8px;
  }

  .chart-wrapper {
    height: 220px;
  }

  .update-collapse :deep(.el-collapse-item__header) {
    padding: 0 12px;
    height: 48px;
  }

  .update-detail {
    padding: 12px;
  }

  .update-note {
    padding: 12px;
    font-size: 13px;
  }

  .update-actions {
    flex-direction: column;
  }

  .update-actions .el-button {
    width: 100%;
  }
}
</style>
