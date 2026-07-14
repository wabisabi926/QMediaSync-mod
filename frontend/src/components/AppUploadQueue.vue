<template>
  <div class="upload-queue-container">
    <div class="card-header">
      <div>
        <h2 class="hidden-md-and-down">上传队列</h2>
        <p class="queue-description">
          上传队列包含 STRM
          同步和刮削流程产生的元数据上传任务，可在这里查看进度、重试失败任务或清理记录。
        </p>
      </div>
      <div class="header-actions">
        <div class="queue-control-actions">
          <el-button
            type="info"
            :size="queueControlSize"
            @click="refreshQueue"
            :loading="backgroundRefreshing"
            >刷新</el-button
          >
          <el-button
            type="warning"
            :size="queueControlSize"
            @click="pauseAllTasks"
            :disabled="!canPauseAllTasks"
            >全部暂停</el-button
          >
          <el-button
            type="success"
            :size="queueControlSize"
            @click="resumeAllTasks"
            :disabled="!canResumeAllTasks"
            >全部恢复</el-button
          >
        </div>
        <div class="queue-cleanup-actions">
          <el-button type="warning" :size="queueControlSize" @click="retryAllFailedTasks"
            >重试失败</el-button
          >
          <el-button type="warning" :size="queueControlSize" @click="clearQueue"
            >清空等待</el-button
          >
          <el-button type="danger" :size="queueControlSize" @click="clearSuccessAndFailedTasks"
            >清空完成/失败</el-button
          >
        </div>
      </div>
    </div>

    <div class="queue-toolbar-row">
      <div class="filter-container">
        <el-select
          v-model="statusFilter"
          :size="queueControlSize"
          placeholder="请选择状态"
          @change="handleStatusChange"
        >
          <el-option label="全部状态" :value="-1"></el-option>
          <el-option label="等待中" :value="0"></el-option>
          <el-option label="上传中" :value="1"></el-option>
          <el-option label="已完成" :value="2"></el-option>
          <el-option label="失败" :value="3"></el-option>
          <el-option label="已取消" :value="4"></el-option>
          <el-option label="待收尾" :value="5"></el-option>
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

    <el-table
      :data="queueData"
      style="width: 100%"
      v-loading="initialLoading"
      empty-text="暂无上传任务"
      :row-key="(row: UploadTask) => String(row.id)"
      :expand-row-keys="pageState.expandedRowKeys"
      @expand-change="handleExpandChange"
      :row-class-name="tableRowClassName"
      :height="tableHeight"
      class="hidden-md-and-up queue-table-mobile"
    >
      <el-table-column type="expand" width="30">
        <template #default="scope">
          <el-descriptions class="margin-top" :column="2" border size="small">
            <el-descriptions-item label="来源">{{
              getUploadSourceName(scope.row.source)
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
            <el-descriptions-item label="进度">
              {{ getUploadedSizeLabel(scope.row) }}
            </el-descriptions-item>
            <el-descriptions-item label="速度">
              {{ formatByteRate(scope.row.upload_speed_bytes) }}
            </el-descriptions-item>
            <el-descriptions-item label="阶段">
              {{ getUploadPhaseLabel(scope.row) }}
            </el-descriptions-item>
            <el-descriptions-item label="结果">
              {{ getUploadResultLabel(scope.row) }}
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
            <el-descriptions-item label="重试次数">
              {{ scope.row.retry_count || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="失败原因" v-if="scope.row.error" :span="2">
              {{ scope.row.error ? scope.row.error : '-' }}
            </el-descriptions-item>
            <el-descriptions-item
              v-for="detail in getUploadTaskDetailRows(scope.row)"
              :key="detail.label"
              :label="detail.label"
              :span="2"
            >
              {{ detail.value }}
            </el-descriptions-item>
          </el-descriptions>
        </template>
      </el-table-column>

      <el-table-column prop="speed" label="上传文件">
        <template #default="scope">
          <div class="mobile-task-summary">
            <div class="mobile-task-meta">
              <el-text type="primary"># {{ scope.row.id }}</el-text>
              <el-tag size="small" effect="plain">{{
                getUploadSourceName(scope.row.source)
              }}</el-tag>
              <el-tag size="small" :type="getStatusTagType(scope.row.status)">
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </div>
            <div class="queue-path-text mobile-task-path">
              {{ scope.row.file_name || scope.row.local_full_path }}
            </div>
            <el-progress
              class="queue-progress mobile-progress"
              :percentage="getUploadProgressPercent(scope.row)"
              :stroke-width="6"
              :show-text="false"
            />
          </div>
        </template>
      </el-table-column>
    </el-table>
    <el-table
      :data="queueData"
      style="width: 100%"
      v-loading="initialLoading"
      empty-text="暂无上传任务"
      :row-key="(row: UploadTask) => String(row.id)"
      :expand-row-keys="pageState.expandedRowKeys"
      @expand-change="handleExpandChange"
      :row-class-name="tableRowClassName"
      :height="tableHeight"
      class="hidden-md-and-down"
    >
      <el-table-column prop="id" label="ID" width="64" />
      <el-table-column prop="source" label="来源" width="128" show-overflow-tooltip>
        <template #default="scope">
          {{ getUploadSourceName(scope.row.source) }}
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
      <el-table-column prop="progress_percent" label="进度" width="220">
        <template #default="scope">
          <div class="progress-cell">
            <el-progress
              class="queue-progress"
              :percentage="getUploadProgressPercent(scope.row)"
              :stroke-width="8"
            />
            <div class="progress-meta">
              <span>{{ getUploadedSizeLabel(scope.row) }}</span>
              <span v-if="scope.row.status === 1 && scope.row.upload_speed_bytes">
                {{ formatByteRate(scope.row.upload_speed_bytes) }}
              </span>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="upload_phase" label="阶段 / 结果" width="148">
        <template #default="scope">
          <div class="stage-result-cell">
            <el-tag :type="getStageResultTagType(scope.row)" effect="light">
              {{ getUploadStageOrResultLabel(scope.row) }}
            </el-tag>
            <el-tooltip
              v-if="getUploadDetailSummary(scope.row)"
              :content="getUploadDetailSummary(scope.row)"
              placement="top"
            >
              <el-text class="stage-detail-link" type="info">详情</el-text>
            </el-tooltip>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="start_time" label="时间" width="180">
        <template #default="scope">
          开始时间：{{ scope.row.start_time ? formatDateTime(scope.row.start_time) : '-' }} <br />
          完成时间：{{ scope.row.end_time ? formatDateTime(scope.row.end_time) : '-' }} <br />
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
      <el-table-column prop="speed" label="上传文件" min-width="300">
        <template #default="scope">
          <p class="queue-path-text">{{ scope.row.local_full_path }}</p>
          <p>
            => <span class="queue-path-text">{{ scope.row.remote_file_id }}</span>
          </p>
        </template>
      </el-table-column>
    </el-table>
    <ResponsivePagination
      v-model:current-page="currentPage"
      v-model:page-size="pageSize"
      :page-sizes="[10, 20, 50, 100]"
      :total="total"
      :is-mobile="isMobileView"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
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
import ResponsivePagination from '@/components/common/ResponsivePagination.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { WarningFilled } from '@element-plus/icons-vue'
import { SERVER_URL } from '@/const'
import { createActiveRequestGate } from '@/composables/useActiveRequestGate'
import { useBackgroundRefresh } from '@/composables/useBackgroundRefresh'
import { mergeStableList, retainExistingKeys } from '@/composables/useStableList'
import { useWSEvent } from '@/composables/useWebSocket'
import { usePageStateStore } from '@/stores/pageState'
import type { AxiosStatic } from 'axios'
import { formatFileSize } from '@/utils/fileSizeUtils'
import {
  getTaskSourceTypeName,
  getTaskSourceTypeTagType,
  getUploadSourceName,
} from '@/utils/taskSourceUtils'
import { formatDateTime } from '@/utils/timeUtils'
import {
  applyUploadQueuePatch,
  formatByteRate,
  getUploadedSizeLabel,
  getUploadPhaseLabel,
  getUploadProgressPercent,
  getUploadResultLabel,
  getUploadStageOrResultLabel,
  getUploadTaskDetailRows,
  type UploadQueuePatch,
} from '@/utils/uploadQueueDisplayUtils'
import { isMobile as checkIsMobile, onDeviceTypeChange } from '@/utils/deviceUtils'
import {
  canPauseQueue,
  canResumeQueue,
  emptyQueueStatusSnapshot,
  normalizeQueueStatusSnapshot,
  removePendingQueueRows,
  type QueueStatusSnapshot,
} from '@/utils/queueStatusUtils'
import 'element-plus/theme-chalk/display.css'

interface UploadTask {
  id: string
  source: string
  source_type: string
  file_name: string
  local_full_path: string
  remote_path: string
  status: 0 | 1 | 2 | 3 | 4 | 5
  file_size: number
  start_time: number
  end_time: number
  remote_file_id: string
  error: string
  retry_count: number
  last_retry_time: number
  is_season_or_tvshow_file: boolean
  uploaded_bytes?: number
  upload_result?: string
  resume_state?: string
  rapid_wait_until?: number
  upload_phase?: string
  upload_speed_bytes?: number
  progress_percent?: number
  total_parts?: number
  uploaded_parts?: number
  source_cleanup_status?: string
  source_cleanup_error?: string
}

interface QueueMutationContextSnapshot {
  contextVersion: number
}

const http: AxiosStatic | undefined = inject('$http')

// 数据状态
const pageStateStore = usePageStateStore()
const pageState = pageStateStore.getPageState('upload-queue', {
  currentPage: 1,
  pageSize: 20,
  filters: { status: -1 },
})
const { initialLoading, backgroundRefreshing, isRefreshing, runRefresh } = useBackgroundRefresh()
const queueData = ref<UploadTask[]>([])
const total = ref(0)
const uploading = ref(0)
const queueStatusSnapshot = ref<QueueStatusSnapshot>(emptyQueueStatusSnapshot())
const queueStatus = computed<0 | 1>(() => (queueStatusSnapshot.value.running ? 1 : 0))
const canPauseAllTasks = computed(() => canPauseQueue(queueStatusSnapshot.value))
const canResumeAllTasks = computed(() => canResumeQueue(queueStatusSnapshot.value))
const isMobileView = ref(checkIsMobile())
const tableHeight = computed(() => (isMobileView.value ? undefined : 'calc(100vh - 300px)'))
const queueControlSize = computed<'small' | 'default'>(() =>
  isMobileView.value ? 'small' : 'default',
)
const currentPage = computed({
  get: () => pageState.currentPage,
  set: (value) => pageStateStore.setPagination('upload-queue', value, pageState.pageSize),
})
const pageSize = computed({
  get: () => pageState.pageSize,
  set: (value) => pageStateStore.setPagination('upload-queue', pageState.currentPage, value),
})
const statusFilter = computed({
  get: () => Number(pageState.filters.status ?? -1),
  set: (value) => pageStateStore.setFilter('upload-queue', 'status', value),
})
const hasActiveQueueWork = computed(
  () =>
    queueStatus.value === 1 ||
    uploading.value > 0 ||
    queueData.value.some((task) => task.status === 0 || task.status === 1 || task.status === 5),
)

const taskMatchesCurrentStatusFilter = (task: UploadTask): boolean => {
  return statusFilter.value === -1 || task.status === statusFilter.value
}

const removeQueueRowByTaskId = (taskId: string | number | undefined): boolean => {
  if (taskId === undefined) {
    return false
  }
  const beforeCount = queueData.value.length
  queueData.value = queueData.value.filter((task) => String(task.id) !== String(taskId))
  const removed = beforeCount - queueData.value.length
  if (removed > 0) {
    total.value = Math.max(0, total.value - removed)
    pageStateStore.setExpandedRowKeys(
      'upload-queue',
      retainExistingKeys(pageState.expandedRowKeys, queueData.value, (row) => row.id),
    )
    return true
  }
  return false
}

// 定时器
const refreshTimer = ref<number | null>(null)
const pendingQueueDataRefresh = ref(false)
let isPageActive = false
let removeDeviceTypeListener: (() => void) | null = null
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
      return '上传中'
    case 2:
      return '已完成'
    case 3:
      return '失败'
    case 4:
      return '已取消'
    case 5:
      return '待收尾'
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
    case 5:
      return 'primary'
    default:
      return 'info'
  }
}

const getStageResultTagType = (
  task: UploadTask,
): 'primary' | 'success' | 'warning' | 'danger' | 'info' => {
  if (task.upload_result === 'rapid_upload' || task.upload_result === 'multipart_uploaded') {
    return 'success'
  }
  if (task.upload_result === 'remote_exists') {
    return 'info'
  }
  if (task.upload_result === 'skipped_after_rapid_wait') {
    return 'warning'
  }
  if (task.upload_phase === 'rapid_waiting') {
    return 'warning'
  }
  if (task.status === 5 || task.upload_phase === 'remote_completed_pending_finalize') {
    return 'primary'
  }
  if (task.status === 2) {
    return 'success'
  }
  if (task.status === 3) {
    return 'danger'
  }
  if (task.status === 1) {
    return 'primary'
  }
  return 'info'
}

const getUploadDetailSummary = (task: UploadTask): string => {
  return getUploadTaskDetailRows(task)
    .map((item) => `${item.label}：${item.value}`)
    .join('；')
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

const handleExpandChange = (row: UploadTask, expandedRows: UploadTask[]) => {
  pageStateStore.setExpandedRowKeys(
    'upload-queue',
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
        const response = await http?.get(`${SERVER_URL}/upload/queue`, {
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
          uploading.value = response.data.data.uploading || 0
          queueStatusSnapshot.value = normalizeQueueStatusSnapshot(
            response.data.data.queue_status,
            queueStatusSnapshot.value.running,
          )
          pageStateStore.setExpandedRowKeys(
            'upload-queue',
            retainExistingKeys(pageState.expandedRowKeys, queueData.value, (row) => row.id),
          )
          if (hasActiveQueueWork.value) {
            startAutoRefresh()
          } else {
            stopAutoRefresh()
          }
        } else {
          ElMessage.error('获取上传队列数据失败')
        }
      } catch (error) {
        if (!queueDataRequestGate.isCurrent(requestId)) {
          return
        }
        console.error('加载上传队列数据错误：', error)
        ElMessage.error('获取上传队列数据失败')
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
    await ElMessageBox.confirm('只能清空等待中的记录，是否继续？此操作不可恢复。', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    const response = await http?.post(`${SERVER_URL}/upload/queue/clear-pending`)

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    if (response?.data.code === 200) {
      const beforeCount = queueData.value.length
      queueData.value = removePendingQueueRows(queueData.value)
      const removedCount = beforeCount - queueData.value.length
      total.value = Math.max(0, total.value - removedCount)
      queueStatusSnapshot.value = {
        ...queueStatusSnapshot.value,
        pending: 0,
        total: Math.max(0, queueStatusSnapshot.value.total - removedCount),
      }
      ElMessage.success('队列已清空')
      await loadQueueData()
      await loadQueueStatus()
    } else {
      ElMessage.error('清空队列失败')
    }
  } catch (error) {
    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }
    if (!isMessageBoxCancelError(error)) {
      console.error('清空队列错误：', error)
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

    const response = await http?.post(`${SERVER_URL}/upload/queue/clear-success-failed`)

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

// 重试所有失败的任务
const retryAllFailedTasks = async () => {
  const operationContext = startQueueMutationContext()

  try {
    await ElMessageBox.confirm('是否重试所有失败的任务？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    const response = await http?.post(`${SERVER_URL}/upload/queue/retry-failed`)

    if (!isQueueMutationContextCurrent(operationContext)) {
      return
    }

    if (response?.data.code === 200) {
      ElMessage.success('已开始重试所有失败任务')
      loadQueueData()
    } else {
      ElMessage.error(`重试失败任务时出错：${response?.data.message || ''}`)
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
    const response = await http?.post(`${SERVER_URL}/upload/queue/stop`)

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
    const response = await http?.post(`${SERVER_URL}/upload/queue/start`)

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
    const response = await http?.get(`${SERVER_URL}/upload/queue/status`)

    if (!queueStatusRequestGate.isCurrent(requestId)) {
      return
    }

    if (response?.data.code === 200) {
      queueStatusSnapshot.value = normalizeQueueStatusSnapshot(
        response.data.data,
        queueStatusSnapshot.value.running,
      )
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

// 启动定时刷新
const startAutoRefresh = () => {
  if (refreshTimer.value || !hasActiveQueueWork.value) {
    return
  }

  refreshTimer.value = window.setInterval(() => {
    if (!document.hidden && hasActiveQueueWork.value) {
      loadQueueData()
    }
    if (!hasActiveQueueWork.value) {
      stopAutoRefresh()
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

const activateQueuePage = () => {
  if (isPageActive) {
    return
  }
  isPageActive = true
  loadQueueStatus()
  loadQueueData()
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

// 处理状态筛选变更
const handleStatusChange = (val: number) => {
  statusFilter.value = val
  currentPage.value = 1
  loadQueueData()
}

const progressPatchFields = [
  'uploaded_bytes',
  'file_size',
  'progress_percent',
  'upload_speed_bytes',
  'upload_phase',
  'upload_result',
  'resume_state',
  'rapid_wait_until',
  'total_parts',
  'uploaded_parts',
  'source_cleanup_status',
  'source_cleanup_error',
] as const

const uploadQueuePatchReasons = ['progress', 'source_cleanup_changed'] as const

const isUploadQueuePatch = (data: Record<string, unknown>): boolean => {
  if (!uploadQueuePatchReasons.includes(data.reason as (typeof uploadQueuePatchReasons)[number])) {
    return false
  }
  if (data.task_id === undefined && data.id === undefined) {
    return false
  }
  return progressPatchFields.some((field) => data[field] !== undefined)
}

const applyUploadQueuePatchForCurrentFilter = (patch: UploadQueuePatch): boolean => {
  const taskId = patch.task_id ?? patch.id
  if (!applyUploadQueuePatch(queueData.value, patch)) {
    return false
  }
  const row = queueData.value.find((item) => String(item.id) === String(taskId))
  if (row && !taskMatchesCurrentStatusFilter(row)) {
    removeQueueRowByTaskId(taskId)
  }
  return true
}

useWSEvent('upload_queue_status_changed', (data) => {
  if (typeof data.running === 'boolean') {
    queueStatusSnapshot.value = {
      ...queueStatusSnapshot.value,
      running: data.running,
    }
  }
  if (isPageActive) {
    loadQueueData()
  }
})

useWSEvent('upload_queue_changed', (data) => {
  if (!isPageActive || document.hidden) {
    return
  }
  if (isUploadQueuePatch(data) && applyUploadQueuePatchForCurrentFilter(data as UploadQueuePatch)) {
    if (hasActiveQueueWork.value) {
      startAutoRefresh()
    } else {
      stopAutoRefresh()
    }
    return
  }
  if (isPageActive && !document.hidden) {
    loadQueueData()
  }
})

// 页面生命周期
onMounted(() => {
  activateQueuePage()
  removeDeviceTypeListener = onDeviceTypeChange((newIsMobile) => {
    isMobileView.value = newIsMobile
  })
})

onActivated(activateQueuePage)

onActivated(() => {
  if (queueData.value.length > 0) {
    pageStateStore.pruneExpandedRowKeys(
      'upload-queue',
      queueData.value.map((row) => String(row.id)),
    )
  }
  nextTick(() => {
    window.dispatchEvent(new Event('resize'))
  })
})

onDeactivated(deactivateQueuePage)

onUnmounted(() => {
  if (removeDeviceTypeListener) {
    removeDeviceTypeListener()
  }
  isPageActive = false
  pendingQueueDataRefresh.value = false
  queueDataRequestGate.invalidate()
  queueStatusRequestGate.invalidate()
  invalidateQueueMutationContext()
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

.queue-control-actions,
.queue-cleanup-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.header-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.queue-description {
  margin: 0;
  color: #606266;
}

.queue-toolbar-row {
  display: flex;
  gap: 20px;
  align-items: center;
}

.filter-container {
  width: 120px;
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

.queue-path-text {
  overflow-wrap: anywhere;
  word-break: break-word;
}

.progress-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.queue-progress {
  width: 100%;
}

.progress-meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  line-height: 1.4;
  color: #606266;
}

.stage-result-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.stage-detail-link {
  cursor: help;
  font-size: 12px;
}

.mobile-task-summary {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.mobile-task-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.mobile-task-path {
  display: -webkit-box;
  max-width: 100%;
  overflow: hidden;
  line-height: 1.4;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.mobile-progress {
  max-width: 100%;
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

  .card-header p {
    margin: 0;
    font-size: 12px;
    line-height: 1.4;
  }

  .card-header {
    flex-direction: column;
    gap: 8px;
    align-items: flex-start;
  }

  .header-actions {
    width: 100%;
    align-self: stretch;
    display: grid;
    justify-content: stretch;
    gap: 6px;
  }

  .queue-control-actions {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    width: 100%;
    gap: 6px;
  }

  .queue-cleanup-actions {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    width: 100%;
    gap: 6px;
  }

  .queue-control-actions :deep(.el-button),
  .queue-cleanup-actions :deep(.el-button) {
    width: 100%;
    min-width: 0;
  }

  .queue-toolbar-row {
    gap: 8px;
    align-items: stretch;
  }

  .filter-container {
    width: 112px !important;
    margin: 8px 0;
  }

  .queue-stats {
    margin: 8px 0;
    gap: 8px;
  }

  .queue-stats :deep(.el-statistic__head) {
    font-size: 12px;
  }

  .queue-stats :deep(.el-statistic__content) {
    font-size: 20px;
  }

  .queue-table-mobile {
    margin-top: 4px;
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
