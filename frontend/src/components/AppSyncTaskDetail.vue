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
            <el-descriptions-item label="开始时间">
              {{ taskInfo?.start_time ? formatDateTime(taskInfo.start_time) : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="结束时间">
              {{ taskInfo?.end_time ? formatDateTime(taskInfo.end_time) : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="处理文件数">
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

        <!-- 执行日志列表 -->
        <div class="task-logs">
          <h3>执行日志</h3>
          <el-table
            :data="taskLogs"
            v-loading="logsLoading"
            stripe
            class="logs-table"
            empty-text="暂无执行日志"
            max-height="500"
          >
            <el-table-column prop="timestamp" label="时间" width="180">
              <template #default="scope">
                {{ formatDateTime(scope.row.timestamp) }}
              </template>
            </el-table-column>
            <el-table-column prop="level" label="级别" width="100">
              <template #default="scope">
                <el-tag :type="getLogLevelType(scope.row.level)" size="small">
                  {{ scope.row.level }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="message" label="日志信息" min-width="400" />
          </el-table>

          <!-- 分页器 -->
          <el-pagination
            v-if="logsTotal > 0"
            v-model:current-page="logsCurrentPage"
            v-model:page-size="logsPageSize"
            :total="logsTotal"
            :page-sizes="[20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            class="logs-pagination"
            @size-change="handleLogsPageSizeChange"
            @current-change="handleLogsPageChange"
          />
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { ArrowLeft } from '@element-plus/icons-vue'
import { inject, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

interface TaskInfo {
  id: number
  start_time: number
  end_time: number | null
  status: 0 | 1 | 2 | 3 // 0-待开始，1-运行中，2-完成，3-失败
  processed_files: number
  created_strm: number
  downloaded_meta: number
}

interface TaskLog {
  id: number
  timestamp: number
  level: 'INFO' | 'WARNING' | 'ERROR' | 'DEBUG'
  message: string
}

const http: AxiosStatic | undefined = inject('$http')
const route = useRoute()
const router = useRouter()

// 获取任务ID
const taskId = ref(route.params.id as string)

// 数据状态
const taskInfo = ref<TaskInfo | null>(null)
const taskLogs = ref<TaskLog[]>([])
const infoLoading = ref(false)
const logsLoading = ref(false)

// 日志分页
const logsCurrentPage = ref(1)
const logsPageSize = ref(20)
const logsTotal = ref(0)

// 返回上一页
const goBack = () => {
  router.push({ name: 'sync-records' })
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

// 获取日志级别类型
const getLogLevelType = (level: string) => {
  switch (level) {
    case 'INFO':
      return 'primary'
    case 'WARNING':
      return 'warning'
    case 'ERROR':
      return 'danger'
    case 'DEBUG':
      return 'info'
    default:
      return 'info'
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

  const startTime = taskInfo.value.start_time
  const endTime = taskInfo.value.end_time || Math.floor(Date.now() / 1000)
  const duration = endTime - startTime

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

// 加载任务信息
const loadTaskInfo = async () => {
  try {
    infoLoading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/task/${taskId.value}`)

    if (response?.data.code === 200) {
      const data = response.data.data
      taskInfo.value = {
        id: data.id,
        start_time: data.created_at,
        end_time: data.finish_at,
        status: data.status as 0 | 1 | 2 | 3,
        processed_files: data.total,
        created_strm: data.new_strm,
        downloaded_meta: data.downloaded_meta || 0,
      }
    }
  } catch (error) {
    console.error('加载任务信息错误:', error)
  } finally {
    infoLoading.value = false
  }
}

// 加载任务日志
const loadTaskLogs = async () => {
  try {
    logsLoading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/task/${taskId.value}/logs`, {
      params: {
        page: logsCurrentPage.value,
        page_size: logsPageSize.value,
      },
    })

    if (response?.data.code === 200) {
      // 暂时使用空数据，等待后端接口实现
      taskLogs.value = []
      logsTotal.value = 0
    }
  } catch (error) {
    console.error('加载任务日志错误:', error)
    // 暂时使用空数据
    taskLogs.value = []
    logsTotal.value = 0
  } finally {
    logsLoading.value = false
  }
}

// 日志分页处理
const handleLogsPageSizeChange = (newSize: number) => {
  logsPageSize.value = newSize
  logsCurrentPage.value = 1
  loadTaskLogs()
}

const handleLogsPageChange = (newPage: number) => {
  logsCurrentPage.value = newPage
  loadTaskLogs()
}

// 页面挂载时加载数据
onMounted(() => {
  loadTaskInfo()
  loadTaskLogs()
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

.task-logs h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.logs-table {
  width: 100%;
  margin-bottom: 20px;
}

.logs-pagination {
  display: flex;
  justify-content: center;
  margin-top: 20px;
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
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .card-title {
    font-size: 18px;
  }
}
</style>
