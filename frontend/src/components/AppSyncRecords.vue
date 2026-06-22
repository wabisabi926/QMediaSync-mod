<template>
  <div class="sync-records-container" ref="pageContainerRef">
    <!-- 同步记录卡片 -->
    <div class="sync-records-card">
      <div class="header-left">
        <h2 class="card-title hidden-md-and-down">同步记录</h2>
        <p class="card-subtitle">只会保留3天的记录，每天0点会删除3天前的所有记录</p>
      </div>
      <div class="header-right">
        <!-- <el-button
              type="primary"
              @click="startManualSync"
              :loading="syncLoading"
              :disabled="hasRunningSyncTask"
              size="large"
            >
              <el-icon><Refresh /></el-icon>
              手动同步
            </el-button> -->
      </div>
    </div>

    <div class="sync-content" v-loading="queryLoading">
      <!-- 批量删除控制 -->
      <div style="display: flex; align-items: center; margin-bottom: 8px; gap: 12px">
        <el-checkbox v-model="batchMode" size="large">批量删除</el-checkbox>
        <el-button
          v-if="batchMode"
          type="danger"
          size="small"
          :disabled="selectedIds.length === 0"
          :loading="batchDeleteLoading"
          @click="batchDeleteRecords"
        >
          批量删除
        </el-button>
      </div>
      <ResponsiveRecordTable
        class="sync-table"
        :rows="syncRecords"
        :columns="syncRecordColumns"
        :actions="syncRecordActions"
        :row-key="getSyncRecordRowKey"
        :loading="initialLoading || queryLoading"
        :is-mobile="isMobileView"
        :expanded-row-keys="pageState.expandedRowKeys"
        :show-selection="batchMode"
        :selectable="isDeletableRecord"
        :height="isMobileView ? 'calc(100vh - 250px)' : 'calc(100vh - 350px)'"
        @selection-change="handleSelectionChange"
        @expand-change="handleExpandChange"
        @action="handleSyncRecordAction"
      >
        <template #cell-id="{ row }"> #{{ row.id }} </template>
        <template #cell-status="{ row }">
          <el-tag :type="getStatusType(row.status)" :effect="row.status === 1 ? 'dark' : 'light'">
            {{ getStatusText(row.status) }}
          </el-tag>
        </template>
        <template #cell-sub_status="{ row }">
          <el-tag v-if="row.status === 1" type="primary" size="small" effect="light">
            {{ getSubStatusText(row.sub_status) }}
          </el-tag>
          <span v-else class="sync-sub-status">-</span>
        </template>
        <template #cell-start_time="{ row }">
          {{ row.start_time ? formatDateTime(row.start_time) : '-' }}
        </template>
        <template #cell-local_path="{ row }">
          <div class="sync-path-cell">
            <el-text type="primary" class="sync-path-cell__id hidden-md-and-up"
              >#{{ row.id }}</el-text
            >
            <span class="sync-path-cell__route">
              {{ row.remote_path || '-' }} => {{ row.local_path || '-' }}
            </span>
          </div>
        </template>
      </ResponsiveRecordTable>

      <!-- 分页器 -->
      <el-pagination
        v-if="total > 0"
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        class="sync-pagination"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import ResponsiveRecordTable from '@/components/records/ResponsiveRecordTable.vue'
import { SERVER_URL } from '@/const'
import { createActiveRequestGate } from '@/composables/useActiveRequestGate'
import { useBackgroundRefresh } from '@/composables/useBackgroundRefresh'
import { mergeStableList, retainExistingKeys } from '@/composables/useStableList'
import { usePageStateStore } from '@/stores/pageState'
import type { RecordAction, RecordActionPayload, RecordColumn } from '@/types/recordTable'
import { isMobile as checkIsMobile, onDeviceTypeChange } from '@/utils/deviceUtils'
import { formatDateTime } from '@/utils/timeUtils'
import type { AxiosStatic } from 'axios'
import { Delete, View } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  computed,
  inject,
  nextTick,
  onActivated,
  onDeactivated,
  onMounted,
  onUnmounted,
  ref,
  useTemplateRef,
  watch,
} from 'vue'
import { useRouter } from 'vue-router'

interface SyncRecord {
  id: number
  start_time: number
  end_time: number | null
  status: 0 | 1 | 2 | 3 // 0-待开始，1-运行中，2-完成，3-失败
  sub_status: 0 | 1 | 2 | 3 | 4 // 0-待开始，1-正在处理网盘文件，2-正在处理本地文件
  processed_files: number
  created_strm: number
  downloaded_meta: number
  uploaded_meta: number
  local_path: string
  remote_path: string
  fail_reason: string
}

interface ApiSyncRecord {
  id: number
  created_at: number
  finish_at: number | null
  status: number
  sub_status: number
  total: number
  new_strm: number
  new_meta: number
  new_upload: number
  local_path: string
  remote_path: string
  fail_reason: string
}

interface SyncRecordDeleteContextSnapshot {
  contextVersion: number
  mode: 'single' | 'batch'
}

const http: AxiosStatic | undefined = inject('$http')
const router = useRouter()

// 数据状态
const pageStateStore = usePageStateStore()
const pageState = pageStateStore.getPageState('sync-records', {
  currentPage: 1,
  pageSize: 20,
})
const { initialLoading, isRefreshing, runRefresh } = useBackgroundRefresh()
const pageContainerRef = useTemplateRef<HTMLElement>('pageContainerRef')
const getPageScrollContainer = () =>
  pageContainerRef.value?.closest<HTMLElement>('.main-content') ?? pageContainerRef.value
const syncRecords = ref<SyncRecord[]>([])
const queryLoading = ref(false)
const isMobileView = ref(checkIsMobile())

// 批量删除相关状态
const batchMode = ref(false)
const selectedIds = ref<number[]>([])

// 删除loading状态
const deleteLoading = ref(false)
const batchDeleteLoading = ref(false)
const deleteOperationContextVersion = ref(0)
const activeDeleteOperationContext = ref<SyncRecordDeleteContextSnapshot | null>(null)

// 分页相关
const currentPage = computed({
  get: () => pageState.currentPage,
  set: (value) => pageStateStore.setPagination('sync-records', value, pageState.pageSize),
})
const pageSize = computed({
  get: () => pageState.pageSize,
  set: (value) => pageStateStore.setPagination('sync-records', pageState.currentPage, value),
})
const total = ref(0)

// 定时器相关 - 已停用，使用WebSocket替代
const refreshTimer = ref<number | null>(null)
const pendingSyncRecordsRefresh = ref(false)
let isPageActive = false
const syncRecordsRequestGate = createActiveRequestGate(() => isPageActive)
let stopDeviceTypeChange: (() => void) | null = null

// WebSocket事件监听
import { useWSEvent } from '@/composables/useWebSocket'

const onSyncStart = () => {
  loadSyncRecords()
}
const onSyncComplete = () => {
  loadSyncRecords()
}

useWSEvent('strm_sync_task_start', onSyncStart)
useWSEvent('strm_sync_task_complete', onSyncComplete)

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
      return '正在处理网盘文件'
    case 2:
      return '正在处理本地文件'
    default:
      return '未知'
  }
}

const syncRecordColumns: RecordColumn<SyncRecord>[] = [
  { key: 'id', label: '任务ID', priority: 'primary', width: 88, align: 'center' },
  { key: 'status', label: '状态', priority: 'primary', width: 96, align: 'center' },
  { key: 'sub_status', label: '子状态', priority: 'secondary', minWidth: 132 },
  {
    key: 'start_time',
    label: '开始时间',
    priority: 'secondary',
    minWidth: 168,
    detailField: {
      key: 'start_time',
      label: '开始时间',
      value: (row) => (row.start_time ? formatDateTime(row.start_time) : '-'),
    },
  },
  {
    key: 'local_path',
    label: '同步路径',
    priority: 'primary',
    minWidth: 260,
    detailField: {
      key: 'local_path',
      label: '本地路径',
      value: (row) => row.local_path,
      span: 2,
      isLongText: true,
    },
  },
  {
    key: 'remote_path',
    label: '网盘路径',
    priority: 'detail',
    detailField: {
      key: 'remote_path',
      label: '网盘路径',
      value: (row) => row.remote_path,
      span: 2,
      isLongText: true,
    },
  },
  {
    key: 'end_time',
    label: '结束时间',
    priority: 'detail',
    detailField: {
      key: 'end_time',
      label: '结束时间',
      value: (row) => (row.end_time ? formatDateTime(row.end_time) : '-'),
    },
  },
  {
    key: 'stats',
    label: '统计',
    priority: 'detail',
    detailField: {
      key: 'stats',
      label: '统计',
      value: (row) =>
        `总文件 ${row.processed_files}，STRM ${row.created_strm}，元数据 ${row.downloaded_meta}，上传 ${row.uploaded_meta}`,
      span: 2,
    },
  },
  {
    key: 'fail_reason',
    label: '失败原因',
    priority: 'detail',
    detailField: {
      key: 'fail_reason',
      label: '失败原因',
      value: (row) => row.fail_reason || '-',
      span: 2,
      isLongText: true,
    },
  },
]

const handleExpandChange = (row: SyncRecord, expandedRows: SyncRecord[]) => {
  pageStateStore.setExpandedRowKeys(
    'sync-records',
    expandedRows.map((item) => String(item.id)),
  )
}

function invalidateDeleteOperationContext() {
  deleteOperationContextVersion.value += 1
  activeDeleteOperationContext.value = null
  deleteLoading.value = false
  batchDeleteLoading.value = false
}

function startDeleteOperationContext(
  mode: SyncRecordDeleteContextSnapshot['mode'],
): SyncRecordDeleteContextSnapshot {
  invalidateDeleteOperationContext()
  const snapshot = {
    contextVersion: deleteOperationContextVersion.value,
    mode,
  }
  activeDeleteOperationContext.value = snapshot
  return snapshot
}

function isDeleteOperationContextCurrent(
  snapshot: SyncRecordDeleteContextSnapshot | null,
  mode?: SyncRecordDeleteContextSnapshot['mode'],
): snapshot is SyncRecordDeleteContextSnapshot {
  return (
    isPageActive &&
    !!snapshot &&
    !!activeDeleteOperationContext.value &&
    activeDeleteOperationContext.value.contextVersion === snapshot.contextVersion &&
    snapshot.contextVersion === deleteOperationContextVersion.value &&
    (mode === undefined || snapshot.mode === mode)
  )
}

function finishDeleteOperationContext(
  snapshot: SyncRecordDeleteContextSnapshot,
  mode: SyncRecordDeleteContextSnapshot['mode'],
) {
  if (isDeleteOperationContextCurrent(snapshot, mode)) {
    activeDeleteOperationContext.value = null
  }
}

function clearSyncRecordsForQuerySwitch() {
  queryLoading.value = true
  syncRecordsRequestGate.invalidate()
  invalidateDeleteOperationContext()
  syncRecords.value = []
  total.value = 0
  selectedIds.value = []
  pageStateStore.setExpandedRowKeys('sync-records', [])
}

function finishSyncRecordsQuerySwitchLoading() {
  if (!isRefreshing.value && !pendingSyncRecordsRefresh.value) {
    queryLoading.value = false
  }
}

// 加载同步记录
const loadSyncRecords = async () => {
  if (!isPageActive) {
    finishSyncRecordsQuerySwitchLoading()
    return
  }

  const requestId = syncRecordsRequestGate.next()

  if (isRefreshing.value) {
    pendingSyncRecordsRefresh.value = true
    return
  }

  try {
    await runRefresh(async () => {
      try {
        const response = await http?.get(`${SERVER_URL}/sync/records`, {
          params: {
            page: currentPage.value,
            page_size: pageSize.value,
          },
        })

        if (!syncRecordsRequestGate.isCurrent(requestId)) {
          return
        }

        if (response?.data.code === 200) {
          const rows = (response.data.data.records || []).map((item: ApiSyncRecord) => ({
            id: item.id,
            start_time: item.created_at,
            end_time: item.finish_at,
            status: item.status as 0 | 1 | 2 | 3,
            sub_status: item.sub_status as 0 | 1 | 2 | 3 | 4,
            processed_files: item.total,
            created_strm: item.new_strm,
            downloaded_meta: item.new_meta || 0,
            uploaded_meta: item.new_upload || 0,
            local_path: item.local_path || '',
            remote_path: item.remote_path || '',
            fail_reason: item.fail_reason || '',
          }))

          syncRecords.value = mergeStableList(syncRecords.value, rows, (row) => row.id)
          pageStateStore.setExpandedRowKeys(
            'sync-records',
            retainExistingKeys(pageState.expandedRowKeys, syncRecords.value, (row) => row.id),
          )
          total.value = response.data.data.total || 0
        }
      } catch (error) {
        if (!syncRecordsRequestGate.isCurrent(requestId)) {
          return
        }
        console.error('加载同步记录错误:', error)
      }
    })
  } finally {
    if (pendingSyncRecordsRefresh.value && isPageActive) {
      pendingSyncRecordsRefresh.value = false
      await loadSyncRecords()
    }
    finishSyncRecordsQuerySwitchLoading()
  }
}

// // 手动开始同步
// const startManualSync = async () => {
//   try {
//     syncLoading.value = true
//     syncStatus.value = null

//     const response = await http?.post(`${SERVER_URL}/sync/start`)

//     if (response?.data.code === 200) {
//       syncStatus.value = {
//         title: '同步任务已启动',
//         type: 'success',
//         description: '手动同步任务已成功启动，请稍后查看同步记录',
//       }
//       // 重新加载记录
//       await loadSyncRecords()
//       // 启动自动刷新
//       startAutoRefresh()
//     } else {
//       syncStatus.value = {
//         title: '启动同步失败',
//         type: 'error',
//         description: response?.data.message || '启动同步任务失败，请重试',
//       }
//     }
//   } catch (error) {
//     console.error('启动同步错误:', error)
//     syncStatus.value = {
//       title: '启动同步出错',
//       type: 'error',
//       description: '启动同步过程中发生错误，请检查网络连接',
//     }
//   } finally {
//     syncLoading.value = false
//   }
// }

// 查看任务详情
const viewTaskDetail = (taskId: number) => {
  router.push({
    name: 'sync-task-detail',
    params: { id: taskId.toString() },
  })
}

const getSyncRecordRowKey = (row: SyncRecord) => row.id

// 分页大小改变
const handleSizeChange = (newSize: number) => {
  pageStateStore.setPagination('sync-records', 1, newSize)
  clearSyncRecordsForQuerySwitch()
  loadSyncRecords()
}

// 当前页改变
const handleCurrentChange = (newPage: number) => {
  pageStateStore.setPagination('sync-records', newPage, pageState.pageSize)
  clearSyncRecordsForQuerySwitch()
  loadSyncRecords()
}

const activateSyncRecordsPage = () => {
  if (isPageActive) {
    return
  }
  isPageActive = true
  loadSyncRecords()
}

const deactivateSyncRecordsPage = () => {
  if (!isPageActive) {
    return
  }
  isPageActive = false
  pendingSyncRecordsRefresh.value = false
  queryLoading.value = false
  syncRecordsRequestGate.invalidate()
  invalidateDeleteOperationContext()
}

// 页面生命周期
onMounted(activateSyncRecordsPage)

onMounted(() => {
  stopDeviceTypeChange = onDeviceTypeChange((nextIsMobile) => {
    isMobileView.value = nextIsMobile
  })
})

onActivated(activateSyncRecordsPage)

onActivated(() => {
  nextTick(() => {
    const scrollContainer = getPageScrollContainer()
    if (scrollContainer) {
      scrollContainer.scrollTop = pageState.scrollTop
    }
  })
})

onDeactivated(() => {
  const scrollContainer = getPageScrollContainer()
  pageStateStore.setScrollTop('sync-records', scrollContainer?.scrollTop || 0)
})

onDeactivated(deactivateSyncRecordsPage)

// 页面卸载时清理定时器（已停用）
onUnmounted(() => {
  isPageActive = false
  pendingSyncRecordsRefresh.value = false
  queryLoading.value = false
  stopDeviceTypeChange?.()
  stopDeviceTypeChange = null
  syncRecordsRequestGate.invalidate()
  invalidateDeleteOperationContext()
  // 旧的定时器清理，已不再需要
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }
})

// 判断记录是否可删除（完成或失败）
const isDeletableRecord = (row: SyncRecord) =>
  row.status === 2 || row.status === 3 || row.status === 0

const syncRecordActions: RecordAction<SyncRecord>[] = [
  { key: 'detail', label: '详情', type: 'primary', icon: View },
  {
    key: 'delete',
    label: '删除',
    type: 'danger',
    icon: Delete,
    visible: (row) => !batchMode.value && isDeletableRecord(row),
    disabled: () => deleteLoading.value,
  },
]

const handleSyncRecordAction = ({ actionKey, row }: RecordActionPayload<SyncRecord>) => {
  if (actionKey === 'detail') {
    viewTaskDetail(row.id)
    return
  }
  if (actionKey === 'delete') {
    void deleteRecord(row.id)
  }
}

// 单条删除，无需确认
const deleteRecord = async (id: number) => {
  const operationContext = startDeleteOperationContext('single')

  try {
    if (!isDeleteOperationContextCurrent(operationContext, 'single')) {
      return
    }

    deleteLoading.value = true
    const response = await http?.post(
      `${SERVER_URL}/sync/delete-records`,
      {
        ids: [id],
      },
      {
        headers: {
          'Content-Type': 'application/json',
        },
      },
    )
    if (!isDeleteOperationContextCurrent(operationContext, 'single')) {
      return
    }

    if (response?.data.code === 200) {
      ElMessage.success('删除成功')
      await loadSyncRecords()
    } else {
      ElMessage.error(response?.data.message || '删除失败')
    }
  } catch {
    if (!isDeleteOperationContextCurrent(operationContext, 'single')) {
      return
    }
    ElMessage.error('删除出错')
  } finally {
    if (isDeleteOperationContextCurrent(operationContext, 'single')) {
      deleteLoading.value = false
      finishDeleteOperationContext(operationContext, 'single')
    }
  }
}

const isMessageBoxCancelError = (error: unknown): boolean => {
  if (error === 'cancel' || error === 'close') {
    return true
  }

  const errorMessage = error instanceof Error ? error.message : String(error)
  return errorMessage.includes('用户取消操作')
}

// 批量删除
const batchDeleteRecords = async () => {
  if (selectedIds.value.length === 0) return

  const ids = [...selectedIds.value]
  const operationContext = startDeleteOperationContext('batch')

  try {
    await ElMessageBox.confirm(
      `确定要批量删除选中的 ${ids.length} 条同步记录吗？此操作不可恢复。`,
      '确认批量删除',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )

    if (!isDeleteOperationContextCurrent(operationContext, 'batch')) {
      return
    }

    batchDeleteLoading.value = true
    const response = await http?.post(
      `${SERVER_URL}/sync/delete-records`,
      {
        ids,
      },
      {
        headers: {
          'Content-Type': 'application/json',
        },
        timeout: 60000, // 1分钟超时
      },
    )
    if (!isDeleteOperationContextCurrent(operationContext, 'batch')) {
      return
    }

    if (response?.data.code === 200) {
      ElMessage.success('批量删除成功')
      selectedIds.value = []
      await loadSyncRecords()
    } else {
      ElMessage.error(response?.data.message || '批量删除失败')
    }
  } catch (error) {
    if (!isDeleteOperationContextCurrent(operationContext, 'batch')) {
      return
    }
    if (!isMessageBoxCancelError(error)) {
      ElMessage.error('批量删除出错')
    }
  } finally {
    if (isDeleteOperationContextCurrent(operationContext, 'batch')) {
      batchDeleteLoading.value = false
      finishDeleteOperationContext(operationContext, 'batch')
    }
  }
}

// 处理多选变化
const handleSelectionChange = (selection: SyncRecord[]) => {
  selectedIds.value = selection.map((r) => r.id)
}

// 切换批量模式时清空选择
watch(batchMode, (val) => {
  if (!val) selectedIds.value = []
})
</script>

<style scoped>
.sync-records-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.sync-records-card {
  width: 100%;
  max-width: none;
  margin: 0;
  border: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}

.header-left {
  flex: 1;
}

.header-right {
  flex-shrink: 0;
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

.sync-content {
  margin-top: 12px;
}

.sync-table {
  width: 100%;
  margin-bottom: 20px;
}

.sync-pagination {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

/* 自定义分页器中文文本 */
.sync-pagination :deep(.el-pagination__total) {
  position: relative;
}

.sync-pagination :deep(.el-pagination__total)::before {
  content: '总数 ' attr(data-total) ' 条';
  position: absolute;
  left: 0;
  top: 0;
  background: white;
  width: 100%;
  height: 100%;
  line-height: inherit;
}

.sync-pagination :deep(.el-pagination__jump) {
  position: relative;
}

.sync-pagination :deep(.el-pagination__jump)::before {
  content: '跳转到';
  margin-right: 8px;
}

/* 修改页数显示文本 */
.sync-pagination :deep(.el-pagination__sizes .el-select .el-input__inner) {
  font-size: 14px;
}

.sync-pagination :deep(.el-pagination__sizes::after) {
  content: '条/页';
  margin-left: 8px;
}

.sync-path-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.sync-path-cell__id {
  flex: 0 0 auto;
}

.sync-path-cell__route {
  min-width: 0;
  overflow-wrap: anywhere;
}

.sync-sub-status {
  color: #909399;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .header-right {
    width: 100%;
  }

  .header-right .el-button {
    width: 100%;
  }

  .card-title {
    font-size: 20px;
  }

  .card-subtitle {
    font-size: 13px;
  }

  .sync-table {
    font-size: 12px;
  }

  .sync-pagination {
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
