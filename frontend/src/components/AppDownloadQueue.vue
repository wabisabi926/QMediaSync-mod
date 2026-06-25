<template>
  <div class="download-queue-container">
    <div class="card-header">
      <div>
        <h2 class="hidden-md-and-down">下载队列</h2>
        <p>
          STRM 同步会把需要的元数据加入下载队列，可在这里查看进度、重试失败任务或清理已完成记录。
        </p>
        <p>来源为“Emby 媒体信息提取”的记录只用于触发提取流程，不会产生实际文件下载。</p>
      </div>
      <div class="header-actions">
        <div class="queue-control-actions">
          <el-button type="info" @click="refreshQueue" :loading="backgroundRefreshing"
            >刷新</el-button
          >
          <el-button type="success" @click="pauseAllTasks" :disabled="queueStatus === 0"
            >全部暂停</el-button
          >
          <el-button type="primary" @click="resumeAllTasks" :disabled="queueStatus === 1"
            >全部恢复</el-button
          >
        </div>
        <div class="queue-cleanup-actions">
          <el-button type="warning" @click="retryFailedTasks">重试失败</el-button>
          <el-button type="warning" @click="clearQueue">清空等待</el-button>
          <el-button type="danger" @click="clearSuccessAndFailedTasks">清空完成/失败</el-button>
        </div>
      </div>
    </div>

    <div style="display: flex; gap: 20px; align-items: center">
      <div class="filter-container" style="width: 120px">
        <el-select v-model="statusFilter" placeholder="请选择状态" @change="handleStatusChange">
          <el-option label="全部状态" :value="-1"></el-option>
          <el-option label="等待中" :value="0"></el-option>
          <el-option label="下载中" :value="1"></el-option>
          <el-option label="已完成" :value="2"></el-option>
          <el-option label="失败" :value="3"></el-option>
          <el-option label="已取消" :value="4"></el-option>
        </el-select>
      </div>

      <div class="queue-stats">
        <el-statistic :value="downloading">
          <template #title>
            <div style="display: inline-flex; align-items: center">
              <el-text class="mx-1" type="primary">正在下载的任务总数</el-text>
            </div>
          </template>
        </el-statistic>
      </div>
    </div>
    <el-table
      :data="queueData"
      style="width: 100%"
      v-loading="initialLoading"
      empty-text="暂无下载任务"
      :row-key="(row: DownloadTask) => String(row.id)"
      :expand-row-keys="pageState.expandedRowKeys"
      @expand-change="handleExpandChange"
      :row-class-name="tableRowClassName"
      height="calc(100vh - 500px)"
      class="hidden-md-and-up"
    >
      <el-table-column type="expand" width="30">
        <template #default="scope">
          <el-descriptions class="margin-top" :column="2" border size="small">
            <el-descriptions-item label="来源">{{
              getDownloadSourceName(scope.row.source)
            }}</el-descriptions-item>
            <el-descriptions-item label="类型">
              <el-tag :type="getTaskSourceTypeTagType(scope.row.source_type)">
                {{ getTaskSourceTypeName(scope.row.source_type) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="getStatusTagType(scope.row.status)">
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="文件大小">
              {{ formatFileSize(scope.row.size) }}
            </el-descriptions-item>
            <el-descriptions-item label="开始时间">
              {{ scope.row.start_time ? formatDateTime(scope.row.start_time) : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="完成时间">
              {{ scope.row.end_time ? formatDateTime(scope.row.end_time) : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="重试次数">
              {{ scope.row.retry_count || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="失败原因" v-if="scope.row.error" :span="2">
              {{ scope.row.error ? scope.row.error : '-' }}
            </el-descriptions-item>
          </el-descriptions>
        </template>
      </el-table-column>
      <el-table-column prop="speed" label="下载文件">
        <template #default="scope">
          <span class="queue-path-text">{{ scope.row.remote_file_id }}</span> <br />
          => <br />
          <span class="queue-path-text">{{ scope.row.local_full_path }}</span>
        </template>
      </el-table-column>
    </el-table>
    <el-table
      :data="queueData"
      style="width: 100%"
      v-loading="initialLoading"
      empty-text="暂无下载任务"
      :row-key="(row: DownloadTask) => String(row.id)"
      :expand-row-keys="pageState.expandedRowKeys"
      @expand-change="handleExpandChange"
      :row-class-name="tableRowClassName"
      height="calc(100vh - 300px)"
      class="hidden-md-and-down"
    >
      <el-table-column prop="id" label="ID" width="64" />
      <el-table-column prop="source" label="来源" width="128" show-overflow-tooltip>
        <template #default="scope">
          {{ getDownloadSourceName(scope.row.source) }}
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="104">
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
      <el-table-column prop="size" label="大小" width="104">
        <template #default="scope">
          {{ formatFileSize(scope.row.size) }}
        </template>
      </el-table-column>
      <el-table-column prop="file_name" label="文件名" min-width="220" show-overflow-tooltip>
        <template #default="scope">
          <span class="filename">{{ scope.row.file_name }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="start_time" label="时间" width="180">
        <template #default="scope">
          开始时间：{{ scope.row.start_time ? formatDateTime(scope.row.start_time) : '-' }}<br />
          结束时间：{{ scope.row.end_time ? formatDateTime(scope.row.end_time) : '-' }}<br />
          <span v-if="scope.row.retry_count">重试次数：{{ scope.row.retry_count }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="source_type" label="类型" width="72">
        <template #default="scope">
          <el-tag :type="getTaskSourceTypeTagType(scope.row.source_type)">
            {{ getTaskSourceTypeName(scope.row.source_type) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="speed" label="下载文件" min-width="240">
        <template #default="scope">
          <span class="queue-path-text">{{ scope.row.remote_file_id }}</span> <br />
          => <br />
          <span class="queue-path-text">{{ scope.row.local_full_path }}</span>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-model:current-page="currentPage"
      v-model:page-size="pageSize"
      :page-sizes="[10, 20, 50, 100]"
      :small="false"
      :disabled="false"
      :background="true"
      layout="total, sizes, prev, pager, next, jumper"
      :total="total"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
      class="pagination-container"
    />
  </div>
</template>

<script setup lang="ts">
import {
  computed,
  inject,
  nextTick,
  onActivated,
  onDeactivated,
  onMounted,
  onUnmounted,
  ref,
} from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { WarningFilled } from '@element-plus/icons-vue'
import { SERVER_URL } from '@/const'
import { createActiveRequestGate } from '@/composables/useActiveRequestGate'
import { useBackgroundRefresh } from '@/composables/useBackgroundRefresh'
import { mergeStableList, retainExistingKeys } from '@/composables/useStableList'
import { usePageStateStore } from '@/stores/pageState'
import type { AxiosStatic } from 'axios'
import { formatFileSize } from '@/utils/fileSizeUtils'
import {
  getDownloadSourceName,
  getTaskSourceTypeName,
  getTaskSourceTypeTagType,
} from '@/utils/taskSourceUtils'
import { formatDateTime } from '@/utils/timeUtils'
import 'element-plus/theme-chalk/display.css'

interface DownloadTask {
  id: string
  source: string
  file_name: string
  local_full_path: string
  remote_path: string
  status: 0 | 1 | 2 | 3 | 4
  size: number
  start_time: number
  end_time: number
  remote_file_id: string
  error: string
  source_type: string
  retry_count: number
  last_retry_time: number
}

interface QueueMutationContextSnapshot {
  contextVersion: number
}

const http: AxiosStatic | undefined = inject('$http')

// 数据状态
const pageStateStore = usePageStateStore()
const pageState = pageStateStore.getPageState('download-queue', {
  currentPage: 1,
  pageSize: 20,
  filters: { status: -1 },
})
const { initialLoading, backgroundRefreshing, isRefreshing, runRefresh } = useBackgroundRefresh()
const queueData = ref<DownloadTask[]>([])
const total = ref(0)
const downloading = ref(0)
const queueStatus = ref<0 | 1>(1) // 0-停止，1-运行中
const currentPage = computed({
  get: () => pageState.currentPage,
  set: (value) => pageStateStore.setPagination('download-queue', value, pageState.pageSize),
})
const pageSize = computed({
  get: () => pageState.pageSize,
  set: (value) => pageStateStore.setPagination('download-queue', pageState.currentPage, value),
})
const statusFilter = computed({
  get: () => Number(pageState.filters.status ?? -1),
  set: (value) => pageStateStore.setFilter('download-queue', 'status', value),
})

// 定时器
const refreshTimer = ref<number | null>(null)
const statusRefreshTimer = ref<number | null>(null)
const pendingQueueDataRefresh = ref(false)
let isPageActive = false
const queueDataRequestGate = createActiveRequestGate(() => isPageActive)
const queueStatusRequestGate = createActiveRequestGate(() => isPageActive)
const queueMutationContextVersion = ref(0)
const activeQueueMutationContext = ref<QueueMutationContextSnapshot | null>(null)

const invalidateQueueMutationContext = () => {
  queueMutationContextVersion.value += 1
  activeQueueMutationContext.value = null
}

const startQueueMutationContext = (): QueueMutationContextSnapshot => {
  invalidateQueueMutationContext()
  const snapshot = {
    contextVersion: queueMutationContextVersion.value,
  }
  activeQueueMutationContext.value = snapshot
  return snapshot
}

const isQueueMutationContextCurrent = (snapshot: QueueMutationContextSnapshot | null) => {
  return (
    isPageActive &&
    !!snapshot &&
    !!activeQueueMutationContext.value &&
    activeQueueMutationContext.value.contextVersion === snapshot.contextVersion &&
    snapshot.contextVersion === queueMutationContextVersion.value
  )
}

const finishQueueMutationContext = (snapshot: QueueMutationContextSnapshot) => {
  if (isQueueMutationContextCurrent(snapshot)) {
    activeQueueMutationContext.value = null
  }
}

const isMessageBoxCancelError = (error: unknown): boolean => {
  if (error === 'cancel' || error === 'close') {
    return true
  }

  const errorMessage = error instanceof Error ? error.message : String(error)
  return errorMessage.includes('用户取消操作')
}

// 获取状态文本
const getStatusText = (status: number): string => {
  switch (status) {
    case 0:
      return '等待中'
    case 1:
      return '下载中'
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
      return 'info'
    default:
      return 'info'
  }
}

// 表格行类名
const tableRowClassName = ({ row }: { row: DownloadTask }) => {
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

const handleExpandChange = (row: DownloadTask, expandedRows: DownloadTask[]) => {
  pageStateStore.setExpandedRowKeys(
    'download-queue',
    expandedRows.map((item) => String(item.id)),
  )
}

// 加载队列数据
const loadQueueData = async () => {
  if (!isPageActive) {
    return
  }

  const requestId = queueDataRequestGate.next()

  if (isRefreshing.value) {
    pendingQueueDataRefresh.value = true
    return
  }

  try {
    await runRefresh(async () => {
      try {
        const response = await http?.get(`${SERVER_URL}/download/queue`, {
          params: {
            page: currentPage.value,
            page_size: pageSize.value,
            status: statusFilter.value,
          },
        })

        if (!queueDataRequestGate.isCurrent(requestId)) {
          return
        }

        if (response?.data.code === 200) {
          const rows = response.data.data.list || []
          queueData.value = mergeStableList(queueData.value, rows, (row) => row.id)
          total.value = response.data.data.total
          downloading.value = response.data.data.downloading || 0
          pageStateStore.setExpandedRowKeys(
            'download-queue',
            retainExistingKeys(pageState.expandedRowKeys, queueData.value, (row) => row.id),
          )
        } else {
          ElMessage.error('获取下载队列数据失败')
        }
      } catch (error) {
        if (!queueDataRequestGate.isCurrent(requestId)) {
          return
        }
        console.error('加载下载队列数据错误：', error)
        ElMessage.error('获取下载队列数据失败')
      }
    })
  } finally {
    if (pendingQueueDataRefresh.value && isPageActive) {
      pendingQueueDataRefresh.value = false
      await loadQueueData()
    }
  }
}

// 刷新队列
const refreshQueue = () => {
  loadQueueData()
}

// 清空队列
const clearQueue = async () => {
  const operationContext = startQueueMutationContext()

  try {
    await ElMessageBox.confirm('只能清空所有等待下载的数据，此操作不可恢复，是否继续？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    const response = await http?.post(`${SERVER_URL}/download/queue/clear-pending`)

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    if (response?.data.code === 200) {
      ElMessage.success('队列已清空')
      loadQueueData()
    } else {
      ElMessage.error('清空队列失败')
    }
  } catch (error) {
    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }
    if (!isMessageBoxCancelError(error)) {
      console.error('清空队列失败：', error)
      ElMessage.error('清空队列失败')
    }
  } finally {
    if (isQueueMutationContextCurrent(operationContext)) {
      finishQueueMutationContext(operationContext)
    }
  }
}

const clearSuccessAndFailedTasks = async () => {
  const operationContext = startQueueMutationContext()

  try {
    await ElMessageBox.confirm(
      '只能清空所有已完成和失败的数据，此操作不可恢复，是否继续？',
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    const response = await http?.post(`${SERVER_URL}/download/queue/clear-success-failed`)

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    if (response?.data.code === 200) {
      ElMessage.success('队列已清空')
      loadQueueData()
    } else {
      ElMessage.error(`清空队列失败：${response?.data.message || ''}`)
    }
  } catch (error) {
    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }
    if (!isMessageBoxCancelError(error)) {
      console.error('清空队列失败：', error)
      ElMessage.error('清空队列失败')
    }
  } finally {
    if (isQueueMutationContextCurrent(operationContext)) {
      finishQueueMutationContext(operationContext)
    }
  }
}

const retryFailedTasks = async () => {
  const operationContext = startQueueMutationContext()

  try {
    await ElMessageBox.confirm('是否重试所有失败任务？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    const response = await http?.post(`${SERVER_URL}/download/queue/retry-failed`)

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    if (response?.data.code === 200) {
      ElMessage.success('失败任务已重新加入队列')
      loadQueueData()
    } else {
      ElMessage.error(response?.data.message || '重试失败任务时出错')
    }
  } catch (error) {
    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }
    if (!isMessageBoxCancelError(error)) {
      console.error('重试失败任务时出错：', error)
      ElMessage.error('重试失败任务时出错')
    }
  } finally {
    if (isQueueMutationContextCurrent(operationContext)) {
      finishQueueMutationContext(operationContext)
    }
  }
}

// 全局暂停所有任务
const pauseAllTasks = async () => {
  const operationContext = startQueueMutationContext()

  try {
    const response = await http?.post(`${SERVER_URL}/download/queue/stop`)

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    if (response?.data.code === 200) {
      ElMessage.success('已暂停所有任务')
      loadQueueData()
    } else {
      ElMessage.error(`暂停所有任务失败：${response?.data.message || ''}`)
    }
  } catch (error) {
    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }
    console.error('暂停所有任务失败：', error)
    ElMessage.error('暂停所有任务失败')
  } finally {
    if (isQueueMutationContextCurrent(operationContext)) {
      finishQueueMutationContext(operationContext)
    }
  }
}

// 全局继续所有任务
const resumeAllTasks = async () => {
  const operationContext = startQueueMutationContext()

  try {
    const response = await http?.post(`${SERVER_URL}/download/queue/start`)

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    if (response?.data.code === 200) {
      ElMessage.success('已恢复所有任务')
      loadQueueData()
    } else {
      ElMessage.error(`恢复所有任务失败：${response?.data.message || ''}`)
    }
  } catch (error) {
    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }
    console.error('恢复所有任务失败：', error)
    ElMessage.error('恢复所有任务失败')
  } finally {
    if (isQueueMutationContextCurrent(operationContext)) {
      finishQueueMutationContext(operationContext)
    }
  }
}

// 获取队列状态
const loadQueueStatus = async () => {
  const requestId = queueStatusRequestGate.next()

  try {
    const response = await http?.get(`${SERVER_URL}/download/queue/status`)

    if (!queueStatusRequestGate.isCurrent(requestId)) {
      return
    }

    if (response?.data.code === 200) {
      // 0-停止，1-运行中
      queueStatus.value = response.data.data ? 1 : 0
    } else {
      console.error('获取队列状态失败：', response?.data.message)
    }
  } catch (error) {
    if (!queueStatusRequestGate.isCurrent(requestId)) {
      return
    }
    console.error('获取队列状态错误：', error)
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

// 处理状态筛选变更
const handleStatusChange = (val: number) => {
  statusFilter.value = val
  currentPage.value = 1
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
    }
  }, 5000)

  // 启动队列状态定时刷新
  if (statusRefreshTimer.value) {
    clearInterval(statusRefreshTimer.value)
  }

  statusRefreshTimer.value = window.setInterval(() => {
    // 只有在页面可见时才刷新
    if (!document.hidden) {
      loadQueueStatus()
    }
  }, 3000)
}

// 停止定时刷新
const stopAutoRefresh = () => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }

  if (statusRefreshTimer.value) {
    clearInterval(statusRefreshTimer.value)
    statusRefreshTimer.value = null
  }
}

const activateQueuePage = () => {
  if (isPageActive) {
    return
  }
  isPageActive = true
  loadQueueData()
  loadQueueStatus()
  startAutoRefresh()
}

const deactivateQueuePage = () => {
  if (!isPageActive) {
    return
  }
  isPageActive = false
  pendingQueueDataRefresh.value = false
  queueDataRequestGate.invalidate()
  queueStatusRequestGate.invalidate()
  invalidateQueueMutationContext()
  stopAutoRefresh()
}

// 页面生命周期
onMounted(activateQueuePage)

onActivated(activateQueuePage)

onActivated(() => {
  if (queueData.value.length > 0) {
    pageStateStore.pruneExpandedRowKeys(
      'download-queue',
      queueData.value.map((row) => String(row.id)),
    )
  }
  nextTick(() => {
    window.dispatchEvent(new Event('resize'))
  })
})

onDeactivated(deactivateQueuePage)

onUnmounted(() => {
  isPageActive = false
  pendingQueueDataRefresh.value = false
  queueDataRequestGate.invalidate()
  queueStatusRequestGate.invalidate()
  invalidateQueueMutationContext()
  stopAutoRefresh()
})
</script>

<style scoped>
.el-statistic {
  --el-statistic-content-font-size: 28px;
}

.download-queue-container {
  width: 100%;
  padding: 20px;
  box-sizing: border-box;
}

.download-queue-card {
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
  flex-wrap: nowrap;
  justify-content: flex-start;
  align-items: center;
  gap: 12px;
}

.queue-control-actions,
.queue-cleanup-actions {
  display: flex;
  flex-wrap: nowrap;
  gap: 12px;
}

.header-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.queue-stats {
  display: flex;
  gap: 16px;
  margin: 16px 0;
  flex-wrap: wrap;
}

.filter-container {
  margin: 16px 0;
}

.filename {
  font-weight: 500;
}

.queue-path-text {
  overflow-wrap: anywhere;
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
  .download-queue-container {
    padding: 12px;
  }

  .card-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .header-actions {
    width: 100%;
    display: grid;
    align-items: stretch;
    gap: 8px;
  }

  .queue-control-actions {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
  }

  .queue-cleanup-actions {
    display: grid;
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .queue-control-actions :deep(.el-button),
  .queue-cleanup-actions :deep(.el-button) {
    width: 100%;
    min-width: 0;
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
