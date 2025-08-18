<template>
  <div class="sync-task-detail-container">
    <!-- 任务详情卡片 -->
    <el-card class="task-detail-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-button type="primary" @click="goBack" size="small" link>
              <el-icon><ArrowLeft /></el-icon>
              返回同步记录
            </el-button>
            <h2 class="card-title">同步任务详情 #{{ taskId }}</h2>
            <p class="card-subtitle">查看同步任务的执行情况和详细信息</p>
          </div>
        </div>
      </template>

      <div class="task-content">
        <!-- 任务基本信息 -->
        <div class="task-info" v-loading="infoLoading">
          <h3>任务信息</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="任务ID">{{ taskId }}</el-descriptions-item>
            <el-descriptions-item label="任务状态">
              <el-tag v-if="taskInfo" :type="getStatusType(taskInfo.status)">
                {{ getStatusText(taskInfo.status) }}
              </el-tag>
              <span v-else>-</span>
            </el-descriptions-item>
            <el-descriptions-item label="子状态">
              <el-tag
                v-if="taskInfo && taskInfo.status === 1"
                type="primary"
                size="small"
                effect="light"
              >
                {{ getSubStatusText(taskInfo.sub_status) }}
              </el-tag>
              <span v-else>-</span>
            </el-descriptions-item>
            <el-descriptions-item label="开始时间">
              {{ taskInfo?.start_time ? formatDateTime(taskInfo.start_time) : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="结束时间">
              {{ taskInfo?.end_time ? formatDateTime(taskInfo.end_time) : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="总文件数">
              {{ taskInfo?.processed_files || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="生成STRM">
              {{ taskInfo?.created_strm || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="下载元数据数">
              {{ taskInfo?.downloaded_meta || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="执行时长">
              {{ getExecutionDuration() }}
            </el-descriptions-item>
          </el-descriptions>
        </div>

        <!-- 执行时间线 -->
        <div class="execution-timeline" v-if="taskInfo">
          <h3>执行时间线</h3>
          <div class="horizontal-timeline">
            <div
              v-for="(item, index) in getTimelineItems()"
              :key="index"
              class="timeline-step"
              :class="{ completed: item.completed, current: item.current }"
            >
              <div class="step-icon">
                <el-icon :class="{ loading: item.current && !item.completed }">
                  <component :is="item.icon" />
                </el-icon>
              </div>
              <div class="step-content">
                <div class="step-title">{{ item.title }}</div>
                <div class="step-time" v-if="item.time !== '进行中...'">{{ item.time }}</div>
                <div class="step-duration" v-if="item.duration && item.completed">
                  耗时: {{ item.duration }}
                </div>
              </div>
              <div
                v-if="index < getTimelineItems().length - 1"
                class="step-connector"
                :class="{ active: item.completed }"
              ></div>
            </div>
          </div>
        </div>

        <!-- 文件对比表格 -->
        <div class="file-compare-table">
          <h3>文件对比</h3>
          <div class="table-container" v-loading="logsLoading">
            <el-table
              :data="compareData"
              stripe
              class="compare-table"
              empty-text="暂无对比数据"
              max-height="500"
            >
              <el-table-column prop="left" label="网盘文件" min-width="300">
                <template #default="scope">
                  <div class="file-paths">
                    <div class="file-path">
                      {{ scope.row.left }}
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="right" label="本地文件" min-width="300">
                <template #default="scope">
                  <div class="file-paths">
                    <div class="file-path">
                      {{ scope.row.right }}
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="status" label="状态" width="120" align="center">
                <template #default="scope">
                  <el-tag :type="getCompareStatusType(scope.row.status)" size="small">
                    {{ getCompareStatusText(scope.row.status) }}
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>

            <!-- 分页器 -->
            <el-pagination
              v-if="compareTotal > 0"
              v-model:current-page="compareCurrentPage"
              v-model:page-size="comparePageSize"
              :total="compareTotal"
              :page-sizes="[50, 100, 200]"
              layout="total, sizes, prev, pager, next, jumper"
              class="compare-pagination"
              @size-change="handleComparePageSizeChange"
              @current-change="handleComparePageChange"
            />
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import {
  ArrowLeft,
  Clock,
  Loading,
  Document,
  Folder,
  Connection,
  Download,
  SuccessFilled,
} from '@element-plus/icons-vue'
import { inject, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

// 任务详情数据结构
interface TaskInfo {
  id: number
  start_time: number
  end_time: number | null
  status: 0 | 1 | 2 | 3 // 0-待开始，1-运行中，2-完成，3-失败
  sub_status: 0 | 1 | 2 | 3 | 4 // 0-待开始，1-正在收集目录结构，2-正在收集文件列表，3-正在比对文件结构，4-正在生成或下载文件
  processed_files: number
  created_strm: number
  downloaded_meta: number
  // 时间线相关字段
  fetch_file_finish_at: number | null
  fetch_dir_finish_at: number | null
  compare_finish_at: number | null
  new_finish_at: number | null
}

// 定义文件对比项接口
interface CompareItem {
  left: string
  right: string
  status: 'same' | 'create' | 'update' | 'delete' | 'upload' | 'download'
}

const http: AxiosStatic | undefined = inject('$http')
const route = useRoute()
const router = useRouter()

// 获取任务ID
const taskId = ref(route.params.id as string)

// 数据状态
const taskInfo = ref<TaskInfo | null>(null)
const compareData = ref<CompareItem[]>([])
const infoLoading = ref(false)
const logsLoading = ref(false)

// 对比数据分页
const compareCurrentPage = ref(1)
const comparePageSize = ref(100)
const compareTotal = ref(0)

// 定时器相关
const refreshTimer = ref<number | null>(null)
const shouldAutoRefresh = ref(false)

// 返回上一页
const goBack = () => {
  router.push({ name: 'sync-records' })
}

// 启动定时器
const startAutoRefresh = () => {
  stopAutoRefresh() // 先清除现有定时器
  shouldAutoRefresh.value = true
  refreshTimer.value = window.setInterval(() => {
    if (shouldAutoRefresh.value) {
      loadTaskInfo()
    }
  }, 5000) // 每5秒刷新一次
}

// 停止定时器
const stopAutoRefresh = () => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }
  shouldAutoRefresh.value = false
}

// 检查是否需要自动刷新
const checkAutoRefresh = () => {
  // 如果任务状态为运行中(1)，启动自动刷新
  if (taskInfo.value?.status === 1) {
    if (!shouldAutoRefresh.value) {
      startAutoRefresh()
    }
  } else {
    stopAutoRefresh()
  }
}

// 获取状态标签类型
const getStatusType = (status: number) => {
  switch (status) {
    case 0:
      return 'info' // 待开始
    case 1:
      return 'primary' // 运行中
    case 2:
      return 'success' // 完成
    case 3:
      return 'danger' // 失败
    default:
      return 'info'
  }
}

// 获取状态文本
const getStatusText = (status: number) => {
  switch (status) {
    case 0:
      return '待开始'
    case 1:
      return '运行中'
    case 2:
      return '已完成'
    case 3:
      return '失败'
    default:
      return '未知'
  }
}

// 获取子状态文本
const getSubStatusText = (subStatus: number) => {
  switch (subStatus) {
    case 0:
      return '待开始'
    case 1:
      return '正在收集目录结构'
    case 2:
      return '正在收集文件列表'
    case 4:
      return '正在生成或下载文件'
    default:
      return '未知'
  }
}

// 格式化时间戳为 YYYY-MM-DD HH:mm:ss 格式
const formatDateTime = (timestamp: number) => {
  if (!timestamp) return '-'
  const date = new Date(timestamp * 1000) // 时间戳转毫秒
  return date
    .toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    })
    .replace(/\//g, '-')
}

// 计算执行时长
const getExecutionDuration = () => {
  if (!taskInfo.value?.start_time) return '-'

  // 如果任务未完成，使用当前时间计算
  if (!taskInfo.value.end_time) {
    const currentTime = Math.floor(Date.now() / 1000)
    const duration = currentTime - taskInfo.value.start_time
    return formatDuration(duration)
  }

  // 已完成任务使用 finish_at - created_at 计算
  const duration = taskInfo.value.end_time - taskInfo.value.start_time
  return formatDuration(duration)
}

// 格式化时长
const formatDuration = (duration: number) => {
  if (duration < 60) {
    return `${duration}秒`
  } else if (duration < 3600) {
    const minutes = Math.floor(duration / 60)
    const seconds = duration % 60
    return `${minutes}分${seconds}秒`
  } else {
    const hours = Math.floor(duration / 3600)
    const minutes = Math.floor((duration % 3600) / 60)
    return `${hours}小时${minutes}分`
  }
}

// 获取对比状态标签类型
const getCompareStatusType = (status: string) => {
  switch (status) {
    case 'create':
      return 'success' // 新增
    case 'download':
      return 'success' // 下载
    case 'upload':
      return 'success' // 上传
    case 'update':
      return 'warning' // 更新
    case 'delete':
      return 'danger' // 删除
    case 'same':
      return 'info' // 不处理
    default:
      return 'info'
  }
}

// 获取对比状态文本
const getCompareStatusText = (status: string) => {
  switch (status) {
    case 'create':
      return '新增'
    case 'update':
      return '更新'
    case 'delete':
      return '删除'
    case 'same':
      return '不处理'
    case 'download':
      return '下载'
    case 'upload':
      return '上传'
    default:
      return '未知'
  }
}

// 获取时间线项目
const getTimelineItems = () => {
  if (!taskInfo.value) return []

  const allSteps = [
    {
      key: 'start',
      title: '开始任务',
      icon: Clock,
      timeField: 'start_time',
    },
    {
      key: 'fetch_dir',
      title: '收集目录结构',
      icon: Folder,
      timeField: 'fetch_dir_finish_at',
    },
    {
      key: 'fetch_file',
      title: '收集文件列表',
      icon: Document,
      timeField: 'fetch_file_finish_at',
    },
    {
      key: 'generate',
      title: '生成或下载文件',
      icon: Download,
      timeField: 'new_finish_at',
    },
    {
      key: 'finish',
      title: '完成任务',
      icon: SuccessFilled,
      timeField: 'end_time',
    },
  ]

  const items = []
  let previousTime = 0

  for (let i = 0; i < allSteps.length; i++) {
    const step = allSteps[i]
    const currentTime =
      step.timeField === 'start_time'
        ? taskInfo.value.start_time
        : (taskInfo.value[step.timeField as keyof TaskInfo] as number | null)

    const completed = currentTime !== null && currentTime > 0

    // 判断是否为当前正在执行的步骤
    let current = false
    if (taskInfo.value.status === 1) {
      // 任务运行中
      if (step.key === 'start') {
        current = taskInfo.value.sub_status === 0
      } else if (step.key === 'fetch_dir') {
        current = taskInfo.value.sub_status === 1
      } else if (step.key === 'fetch_file') {
        current = taskInfo.value.sub_status === 2
      } else if (step.key === 'generate') {
        current = taskInfo.value.sub_status === 4
      }
    }

    // 检查前一个节点是否已完成，如果是且当前节点未开始，则设为进行中
    if (!completed && !current && i > 0) {
      const prevItem = items[i - 1]
      if (prevItem && prevItem.completed) {
        current = true
      }
    }

    let duration = null
    if (completed && previousTime > 0) {
      duration = formatDuration(currentTime! - previousTime)
    }

    let title = step.title
    let time = '未开始'

    if (completed) {
      // 对开始任务和完成任务进行特殊处理
      if (step.key === 'start' || step.key === 'finish') {
        title = step.title
      } else {
        title = `已完成${step.title}`
      }
      time = formatDateTime(currentTime!)
    } else if (current) {
      title = `正在${step.title}`
      time = '进行中...'
    }

    items.push({
      title,
      time,
      icon: current ? Loading : step.icon,
      duration,
      completed,
      current,
    })

    if (completed) {
      previousTime = currentTime!
    }
  }

  return items
}

// 加载任务信息
const loadTaskInfo = async () => {
  try {
    infoLoading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/task?sync_id=${taskId.value}`)

    if (response?.data.code === 200) {
      const data = response.data.data
      taskInfo.value = {
        id: data.id,
        start_time: data.created_at,
        end_time: data.finish_at,
        status: data.status as 0 | 1 | 2 | 3,
        sub_status: data.sub_status as 0 | 1 | 2 | 3 | 4,
        processed_files: data.total,
        created_strm: data.new_strm,
        downloaded_meta: data.new_meta || 0,
        fetch_file_finish_at: data.fetch_file_finish_at || null,
        fetch_dir_finish_at: data.fetch_dir_finish_at || null,
        compare_finish_at: data.compare_finish_at || null,
        new_finish_at: data.new_finish_at || null,
      }
    }
  } catch (error) {
    console.error('加载任务信息错误:', error)
  } finally {
    infoLoading.value = false
    // 检查是否需要自动刷新
    checkAutoRefresh()
  }
}

// 加载文件对比数据
const loadCompareData = async () => {
  try {
    logsLoading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/task/compare?sync_id=${taskId.value}`, {
      params: {
        page: compareCurrentPage.value,
        pageSize: comparePageSize.value,
      },
    })

    if (response?.data.code === 200) {
      const data = response.data.data
      compareData.value = data.list || []
      compareTotal.value = data.total || 0
    } else {
      compareData.value = []
      compareTotal.value = 0
    }
  } catch (error) {
    console.error('加载文件对比数据错误:', error)
    compareData.value = []
    compareTotal.value = 0
  } finally {
    logsLoading.value = false
  }
}

// 对比数据分页处理
const handleComparePageSizeChange = (newSize: number) => {
  comparePageSize.value = newSize
  compareCurrentPage.value = 1
  loadCompareData()
}

const handleComparePageChange = (newPage: number) => {
  compareCurrentPage.value = newPage
  loadCompareData()
}

// 页面挂载时加载数据
onMounted(() => {
  loadTaskInfo()
  loadCompareData()
})

// 页面卸载时清理定时器
onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.sync-task-detail-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.task-detail-card {
  width: 100%;
  max-width: none;
  margin: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
}

.header-left {
  flex: 1;
}

.card-title {
  margin: 8px 0 8px 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.card-subtitle {
  margin: 0;
  font-size: 14px;
  color: #909399;
}

.task-content {
  margin-top: 20px;
}

.task-info {
  margin-bottom: 30px;
}

.task-info h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.execution-timeline {
  margin-bottom: 30px;
}

.execution-timeline h3 {
  margin: 0 0 20px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.horizontal-timeline {
  display: flex;
  align-items: flex-start;
  gap: 0;
  padding: 20px 0;
  overflow-x: auto;
}

.timeline-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  min-width: 140px;
  flex-shrink: 0;
}

.timeline-step.completed .step-icon {
  background-color: #67c23a;
  border-color: #67c23a;
  color: #fff;
}

.timeline-step.current .step-icon {
  background-color: #409eff;
  border-color: #409eff;
  color: #fff;
  animation: pulse 2s infinite;
}

.timeline-step:not(.completed):not(.current) .step-icon {
  background-color: #f5f7fa;
  border-color: #dcdfe6;
  color: #c0c4cc;
}

.step-icon {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: 2px solid;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
  transition: all 0.3s ease;
  z-index: 2;
  position: relative;
}

.step-icon .el-icon {
  font-size: 20px;
}

.step-icon .el-icon.loading {
  animation: rotate 2s linear infinite;
}

.step-content {
  text-align: center;
  max-width: 120px;
}

.step-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
  line-height: 1.4;
  word-break: break-word;
}

.timeline-step:not(.completed):not(.current) .step-title {
  color: #c0c4cc;
}

.step-time {
  font-size: 12px;
  color: #909399;
  margin-bottom: 2px;
}

.timeline-step:not(.completed):not(.current) .step-time {
  color: #c0c4cc;
}

.step-duration {
  font-size: 12px;
  color: #67c23a;
  font-weight: 500;
}

.step-connector {
  position: absolute;
  top: 19px;
  left: 50%;
  right: -50%;
  height: 2px;
  background-color: #dcdfe6;
  z-index: 1;
}

.step-connector.active {
  background-color: #67c23a;
}

.timeline-step:last-child .step-connector {
  display: none;
}

@keyframes pulse {
  0% {
    box-shadow: 0 0 0 0 rgba(64, 158, 255, 0.4);
  }
  70% {
    box-shadow: 0 0 0 10px rgba(64, 158, 255, 0);
  }
  100% {
    box-shadow: 0 0 0 0 rgba(64, 158, 255, 0);
  }
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.timeline-content {
  padding: 8px 0;
}

.timeline-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.timeline-duration {
  font-size: 14px;
  color: #909399;
}

.file-compare-table h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.table-container {
  width: 100%;
  overflow-x: auto;
}

.compare-table {
  width: 100%;
  margin-bottom: 20px;
}

/* 为表格列添加竖线分隔 */
.compare-table :deep(.el-table__cell) {
  border-right: 1px solid #ebeef5;
}

.file-paths {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.file-path {
  font-size: 13px;
  color: #606266;
  word-break: break-all;
  line-height: 1.4;
}

.compare-pagination {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

/* 自定义分页器中文文本 */
.compare-pagination :deep(.el-pagination__total) {
  position: relative;
}

.compare-pagination :deep(.el-pagination__total)::before {
  content: '总数 ' attr(data-total) ' 条';
  position: absolute;
  left: 0;
  top: 0;
  background: white;
  width: 100%;
  height: 100%;
  line-height: inherit;
}

.compare-pagination :deep(.el-pagination__jump) {
  position: relative;
}

.compare-pagination :deep(.el-pagination__jump)::before {
  content: '跳转到';
  margin-right: 8px;
}

.compare-pagination :deep(.el-pagination__sizes .el-select .el-input__inner) {
  font-size: 14px;
}

.compare-pagination :deep(.el-pagination__sizes::after) {
  content: '条/页';
  margin-left: 8px;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .card-title {
    font-size: 20px;
  }

  .card-subtitle {
    font-size: 13px;
  }

  .task-info h3,
  .task-logs h3 {
    font-size: 16px;
  }

  .logs-table {
    font-size: 12px;
  }

  .logs-pagination {
    flex-wrap: wrap;
    gap: 8px;
  }

  .horizontal-timeline {
    padding: 15px 0;
  }

  .timeline-step {
    min-width: 100px;
  }

  .step-content {
    max-width: 90px;
  }

  .step-title {
    font-size: 12px;
  }

  .step-time {
    font-size: 11px;
  }

  .step-duration {
    font-size: 11px;
  }

  .step-icon {
    width: 35px;
    height: 35px;
    margin-bottom: 8px;
  }

  .step-icon .el-icon {
    font-size: 18px;
  }

  .step-connector {
    top: 16px;
  }
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .card-title {
    font-size: 18px;
  }
}
</style>
