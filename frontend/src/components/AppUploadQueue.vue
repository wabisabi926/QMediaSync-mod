<template>
  <div class="upload-queue-container">
    <div class="card-header">
      <div>
        <h2 class="hidden-md-and-down">上传队列</h2>
        <p>这里包含strm同步时产生的元数据的上传和刮削产生的上传任务。</p>
      </div>
      <div class="header-actions">
        <el-button type="info" @click="refreshQueue">刷新</el-button>
        <el-button type="success" @click="pauseAllTasks" :disabled="queueStatus === 0">全部暂停</el-button>
        <el-button type="primary" @click="resumeAllTasks" :disabled="queueStatus === 1">全部开始</el-button>
        <el-button type="warning" @click="clearQueue">清空等待中的任务</el-button>
        <el-button type="danger" @click="clearSuccessAndFailedTasks">清空成功和失败的任务</el-button>
      </div>
    </div>

    <div style="display:flex; gap: 20px; align-items: center;">
      <div class="filter-container" style="width: 120px;">
        <el-select v-model="statusFilter" placeholder="请选择状态" @change="handleStatusChange">
          <el-option label="全部状态" :value="-1"></el-option>
          <el-option label="等待中" :value="0"></el-option>
          <el-option label="上传中" :value="1"></el-option>
          <el-option label="已完成" :value="2"></el-option>
          <el-option label="失败" :value="3"></el-option>
          <el-option label="已取消" :value="4"></el-option>
        </el-select>
      </div>


      <div class="queue-stats">
        <el-statistic :value="uploading">
          <template #title>
            <div style="display: inline-flex; align-items: center">
              <el-text class="mx-1" type="primary">正在上传的任务总数</el-text>
            </div>
          </template>
        </el-statistic>
      </div>
    </div>

    <el-table :data="queueData" style="width: 100%" v-loading="loading" empty-text="暂无上传任务"
      :row-class-name="tableRowClassName" height="calc(100vh - 420px)" class="hidden-md-and-up">
      <el-table-column type="expand" width="30">
        <template #default="scope">
          <el-descriptions class="margin-top" :column="2" border size="small">
            <el-descriptions-item label="来源">{{ scope.row.source }}</el-descriptions-item>
            <el-descriptions-item label="类型">
              <el-tag :type="getSourceTypeTagType(scope.row.source_type)">
                {{ getSourceTypeName(scope.row.source_type) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="getStatusTagType(scope.row.status)">
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="文件大小">
              {{ formatFileSize(scope.row.file_size) }}
            </el-descriptions-item>
            <el-descriptions-item label="开始时间">
              {{ scope.row.start_time ? formatDateTime(scope.row.start_time) : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="完成时间">
              {{ scope.row.end_time ? formatDateTime(scope.row.end_time) : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="失败原因" v-if="scope.row.error" :span="2">
              {{ scope.row.error ? scope.row.error : '-' }}
            </el-descriptions-item>
          </el-descriptions>
        </template>
      </el-table-column>

      <el-table-column prop="speed" label="上传文件">
        <template #default="scope">
          <p><el-text type="primary"># {{ scope.row.id }}</el-text> {{ scope.row.local_full_path }}</p>
          <p> => {{ scope.row.remote_file_id }}</p>
        </template>
      </el-table-column>
    </el-table>
    <el-table :data="queueData" style="width: 100%" v-loading="loading" empty-text="暂无上传任务"
      :row-class-name="tableRowClassName" height="calc(100vh - 300px)" class="hidden-md-and-down">
      <el-table-column prop="id" label="任务ID" width="60" />
      <el-table-column prop="source" label="来源类型" width="80" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="scope">
          <div v-if="scope.row.error">
            <el-tooltip :content="scope.row.error" placement="top">
              <el-tag :type="getStatusTagType(scope.row.status)">
                <el-icon>
                  <WarningFilled />
                </el-icon>
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </el-tooltip>
          </div>
          <div v-else>
            <el-tag :type="getStatusTagType(scope.row.status)">
              {{ getStatusText(scope.row.status) }}
            </el-tag>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="file_size" label="文件大小" width="100">
        <template #default="scope">
          {{ formatFileSize(scope.row.file_size) }}
        </template>
      </el-table-column>
      <el-table-column prop="start_time" label="时间" width="260">
        <template #default="scope">
          开始时间：{{ scope.row.start_time ? formatDateTime(scope.row.start_time) : '-' }} <br />
          完成时间：{{ scope.row.end_time ? formatDateTime(scope.row.end_time) : '-' }} <br />
        </template>
      </el-table-column>
      <el-table-column prop="source_type" label="类型" width="80">
        <template #default="scope">
          <el-tag :type="getSourceTypeTagType(scope.row.source_type)">
            {{ getSourceTypeName(scope.row.source_type) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="speed" label="上传文件">
        <template #default="scope">
          <p>{{ scope.row.local_full_path }}</p>
          <p> => {{ scope.row.remote_file_id }}</p>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :page-sizes="[10, 20, 50, 100]"
      :small="false" :disabled="false" :background="true" layout="total, sizes, prev, pager, next, jumper"
      :total="total" @size-change="handleSizeChange" @current-change="handleCurrentChange"
      class="pagination-container" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject } from 'vue'
import { formatFileSize } from '@/utils/fileSizeUtils'
import { formatDateTime } from '@/utils/timeUtils'
import 'element-plus/theme-chalk/display.css'

interface UploadTask {
  id: string
  source: string
  source_type: string
  file_name: string
  local_full_path: string
  remote_path: string
  status: 0 | 1 | 2 | 3 | 4
  file_size: number
  start_time: number
  end_time: number
  remote_file_id: string
  error: string
  is_season_or_tvshow_file: boolean
}

const http: AxiosStatic | undefined = inject('$http')

// 数据状态
const queueData = ref<UploadTask[]>([])
const loading = ref(false)
const total = ref(0)
const uploading = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const statusFilter = ref(-1)
const queueStatus = ref<0 | 1>(1) // 0-停止，1-运行中

// 定时器
const refreshTimer = ref<number | null>(null)

// 获取状态文本
const getStatusText = (status: number): string => {
  switch (status) {
    case 0:
      return '等待中'
    case 1:
      return '上传中'
    case 2:
      return '已完成'
    case 3:
      return '失败'
    case 4:
      return '已取消'
    default:
      return '未知'
  }
}

// 获取状态标签类型
const getStatusTagType = (
  status: number,
): 'primary' | 'success' | 'warning' | 'danger' | 'info' => {
  switch (status) {
    case 0:
      return 'info'
    case 1:
      return 'primary'
    case 2:
      return 'success'
    case 3:
      return 'danger'
    case 4:
      return 'warning'
    default:
      return 'info'
  }
}

// 表格行类名
const tableRowClassName = ({ row }: { row: UploadTask }) => {
  switch (row.status) {
    case 2:
      return 'success-row'
    case 3:
      return 'error-row'
    case 4:
      return 'cancelled-row'
    default:
      return ''
  }
}

const getSourceTypeName = (type: string): string => {
  switch (type) {
    case 'local':
      return '本地文件'
    case '115':
      return '115云盘'
    case 'openlist':
      return 'OpenList'
    case '123':
      return '123云盘'
    default:
      return '其他'
  }
}

// 获取类型标签类型
const getSourceTypeTagType = (type: string): string => {
  switch (type) {
    case 'local':
      return 'warning'
    case '115':
      return 'primary'
    case 'openlist':
      return 'success'
    case '123':
      return 'info'
    default:
      return 'info'
  }
}

// 加载队列数据
const loadQueueData = async () => {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/upload/queue`, {
      params: {
        page: currentPage.value,
        page_size: pageSize.value,
        status: statusFilter.value
      },
    })

    if (response?.data.code === 200) {
      queueData.value = response.data.data.list
      total.value = response.data.data.total
      uploading.value = response.data.data.uploading || 0
    } else {
      ElMessage.error('获取上传队列数据失败')
    }
  } catch (error) {
    console.error('加载上传队列数据错误:', error)
    ElMessage.error('加载上传队列数据失败')
  } finally {
    loading.value = false
  }
}

// 刷新队列
const refreshQueue = () => {
  loadQueueData()
}

// 清空队列
const clearQueue = async () => {
  try {
    await ElMessageBox.confirm('只能清空等待中的记录，是否继续？此操作不可恢复。', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })

    const response = await http?.post(`${SERVER_URL}/upload/queue/clear-pending`)

    if (response?.data.code === 200) {
      ElMessage.success('队列已清空')
      loadQueueData()
    } else {
      ElMessage.error('清空队列失败')
    }
  } catch (error) {
    // 用户取消或请求失败
    console.error('清空队列错误:', error)
    ElMessage.error('清空队列失败')
  }
}

const clearSuccessAndFailedTasks = async () => {
  try {
    await ElMessageBox.confirm('只能清空所有已完成和失败的数据，此操作不可恢复，是否继续？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })

    const response = await http?.post(`${SERVER_URL}/upload/queue/clear-success-failed`)

    if (response?.data.code === 200) {
      ElMessage.success('队列已清空')
      loadQueueData()
    } else {
      ElMessage.error(`清空队列失败: ${response?.data.message || ''}`)
    }
  } catch {
    // 用户取消或请求失败
  }
}


// 全局暂停所有任务
const pauseAllTasks = async () => {
  try {
    const response = await http?.post(`${SERVER_URL}/upload/queue/stop`)

    if (response?.data.code === 200) {
      ElMessage.success('已暂停所有任务')
      loadQueueData()
    } else {
      ElMessage.error(`暂停所有任务失败: ${response?.data.message || ''}`)
    }
  } catch (error) {
    console.error('暂停所有任务失败:', error)
    ElMessage.error('暂停所有任务失败')
  }
}

// 全局继续所有任务
const resumeAllTasks = async () => {
  try {
    const response = await http?.post(`${SERVER_URL}/upload/queue/start`)

    if (response?.data.code === 200) {
      ElMessage.success('已开始所有任务')
      loadQueueData()
    } else {
      ElMessage.error(`开始所有任务失败: ${response?.data.message || ''}`)
    }
  } catch (error) {
    console.error('开始所有任务失败:', error)
    ElMessage.error('开始所有任务失败')
  }
}

// 获取队列状态
const loadQueueStatus = async () => {
  try {
    const response = await http?.get(`${SERVER_URL}/upload/queue/status`)

    if (response?.data.code === 200) {
      // 0-停止，1-运行中
      queueStatus.value = response.data.data ? 1 : 0
    } else {
      console.error('获取队列状态失败:', response?.data.message)
    }
  } catch (error) {
    console.error('获取队列状态错误:', error)
  }
}

// 处理每页数量变更
const handleSizeChange = (val: number) => {
  pageSize.value = val
  currentPage.value = 1
  loadQueueData()
}

// 处理当前页变更
const handleCurrentChange = (val: number) => {
  currentPage.value = val
  loadQueueData()
}

// 启动定时刷新
const startAutoRefresh = () => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
  }

  refreshTimer.value = window.setInterval(() => {
    // 只有在页面可见时才刷新
    if (!document.hidden) {
      loadQueueData()
      loadQueueStatus()
    }
  }, 5000)
}

// 停止定时刷新
const stopAutoRefresh = () => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }
}

// 处理状态筛选变更
const handleStatusChange = (val: number) => {
  statusFilter.value = val
  currentPage.value = 1
  loadQueueData()
}

// 页面生命周期
onMounted(() => {
  loadQueueStatus()
  loadQueueData()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.upload-queue-container {
  width: 100%;
  padding: 20px;
  box-sizing: border-box;
}

.upload-queue-card {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
}

.header-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-start;
  gap: 12px;
}

.queue-stats {
  display: flex;
  gap: 16px;
  margin: 16px 0;
  flex-wrap: wrap;
}

.filename {
  font-weight: 500;
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

/* 表格行样式 */
:deep(.success-row) {
  background-color: #f0f9ff;
}

:deep(.error-row) {
  background-color: #fef0f0;
}

:deep(.cancelled-row) {
  background-color: #f5f7fa;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .upload-queue-container {
    padding: 12px;
  }

  .card-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .header-actions {
    width: 100%;
    justify-content: space-between;
  }

  .queue-stats {
    gap: 12px;
  }

  :deep(.el-table) {
    font-size: 12px;
  }

  :deep(.el-table th) {
    padding: 8px 0;
  }

  :deep(.el-table td) {
    padding: 6px 0;
  }
}
</style>
