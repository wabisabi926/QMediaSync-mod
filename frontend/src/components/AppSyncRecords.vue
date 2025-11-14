<template>
  <div class="sync-records-container">
    <!-- 同步记录卡片 -->
    <el-card class="sync-records-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <h2 class="card-title">同步记录</h2>
            <p class="card-subtitle">查看STRM文件同步历史记录和状态</p>
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
      </template>

      <div class="sync-content">
        <!-- 批量删除控制 -->
        <div style="display: flex; align-items: center; margin-bottom: 8px; gap: 12px">
          <el-checkbox v-model="batchMode" size="large">批量删除</el-checkbox>
          <el-button v-if="batchMode" type="danger" size="small" :disabled="selectedIds.length === 0"
            :loading="batchDeleteLoading" @click="batchDeleteRecords">
            批量删除
          </el-button>
        </div>
        <!-- 同步记录表格 -->
        <el-table :data="syncRecords" v-loading="tableLoading" stripe class="sync-table" empty-text="暂无同步记录"
          :show-overflow-tooltip="true" @selection-change="handleSelectionChange">
          <el-table-column v-if="batchMode" type="selection" width="50" align="center"
            :selectable="isDeletableRecord" />
          <el-table-column prop="id" label="任务ID" width="80" />
          <el-table-column prop="status" label="状态" width="90">
            <template #default="scope">
              <el-tag :type="getStatusType(scope.row.status)" :effect="scope.row.status === 1 ? 'dark' : 'light'">
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="sub_status" label="子状态" width="120" class-name="hidden-xs">
            <template #default="scope">
              <el-tag v-if="scope.row.status === 1" type="primary" size="small" effect="light">
                {{ getSubStatusText(scope.row.sub_status) }}
              </el-tag>
              <span v-else>-</span>
            </template>
          </el-table-column>

          <el-table-column prop="start_time" label="开始时间" width="180">
            <template #default="scope">
              {{ formatDateTime(scope.row.start_time) }}
            </template>
          </el-table-column>
          <el-table-column prop="end_time" label="结束时间" width="180">
            <template #default="scope">
              {{ scope.row.end_time ? formatDateTime(scope.row.end_time) : '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="local_path" label="本地路径" width="150" show-overflow-tooltip>
            <template #default="scope">
              {{ scope.row.local_path || '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="remote_path" label="网盘路径" width="150" show-overflow-tooltip>
            <template #default="scope">
              {{ scope.row.remote_path || '-' }}
            </template>
          </el-table-column>

          <el-table-column prop="processed_files" label="总文件数" width="120" align="center" />
          <el-table-column prop="created_strm" label="新增STRM数" width="120" align="center" />
          <el-table-column prop="downloaded_meta" label="下载元数据数" width="140" align="center" class-name="hidden-xs" />
          <el-table-column prop="fail_reason" label="失败原因" width="200" show-overflow-tooltip>
            <template #default="scope">
              <span v-if="scope.row.status === 3 && scope.row.fail_reason" class="fail-reason-text">
                {{ scope.row.fail_reason }}
              </span>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120" align="center" fixed="right">
            <template #default="scope">
              <el-button type="primary" size="small" @click="viewTaskDetail(scope.row.id)" link>
                查看
              </el-button>
              <el-button v-if="isDeletableRecord(scope.row) && !batchMode" type="danger" size="small"
                style="margin-left: 4px" :loading="deleteLoading" @click="deleteRecord(scope.row.id)" link>
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 分页器 -->
        <el-pagination v-if="total > 0" v-model:current-page="currentPage" v-model:page-size="pageSize" :total="total"
          :page-sizes="[10, 20, 50, 100]" layout="total, sizes, prev, pager, next, jumper" class="sync-pagination"
          @size-change="handleSizeChange" @current-change="handleCurrentChange" />
      </div>
    </el-card>

    <!-- 同步状态提示 -->
    <el-alert v-if="syncStatus" :title="syncStatus.title" :type="syncStatus.type" :description="syncStatus.description"
      :closable="false" show-icon class="sync-status" />
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject, onMounted, onUnmounted, ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDateTime } from '@/utils/timeUtils'

interface SyncRecord {
  id: number
  start_time: number
  end_time: number | null
  status: 0 | 1 | 2 | 3 // 0-待开始，1-运行中，2-完成，3-失败
  sub_status: 0 | 1 | 2// 0-待开始，1-正在处理网盘文件，2-正在处理本地文件
  processed_files: number
  created_strm: number
  downloaded_meta: number
  local_path: string
  remote_path: string
  fail_reason: string
}

interface SyncStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
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
  local_path: string
  remote_path: string
  fail_reason: string
}

const http: AxiosStatic | undefined = inject('$http')
const router = useRouter()

// 数据状态
const syncRecords = ref<SyncRecord[]>([])
const tableLoading = ref(false)
const syncStatus = ref<SyncStatus | null>(null)

// 批量删除相关状态
const batchMode = ref(false)
const selectedIds = ref<number[]>([])

// 删除loading状态
const deleteLoading = ref(false)
const batchDeleteLoading = ref(false)

// 分页相关
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 定时器相关
const refreshTimer = ref<number | null>(null)
const shouldAutoRefresh = ref(false)

// 计算是否有正在运行的同步任务
const hasRunningSyncTask = computed(() => {
  return syncRecords.value.some((record) => record.status === 1) // 1-运行中
})

// 启动定时器
const startAutoRefresh = () => {
  stopAutoRefresh() // 先清除现有定时器
  shouldAutoRefresh.value = true
  refreshTimer.value = window.setInterval(() => {
    if (shouldAutoRefresh.value) {
      loadSyncRecords()
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
  if (hasRunningSyncTask.value) {
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
      return '正在处理网盘文件'
    case 2:
      return '正在处理本地文件'
    default:
      return '未知'
  }
}

// 加载同步记录
const loadSyncRecords = async () => {
  try {
    tableLoading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/records`, {
      params: {
        page: currentPage.value,
        page_size: pageSize.value,
      },
    })

    if (response?.data.code === 200) {
      syncRecords.value = (response.data.data.records || []).map((item: ApiSyncRecord) => ({
        id: item.id,
        start_time: item.created_at,
        end_time: item.finish_at,
        status: item.status as 0 | 1 | 2 | 3,
        sub_status: item.sub_status as 0 | 1 | 2 | 3 | 4,
        processed_files: item.total,
        created_strm: item.new_strm,
        downloaded_meta: item.new_meta || 0,
        local_path: item.local_path || '',
        remote_path: item.remote_path || '',
        fail_reason: item.fail_reason || '',
      }))
      total.value = response.data.data.total || 0
    }
  } catch (error) {
    console.error('加载同步记录错误:', error)
  } finally {
    tableLoading.value = false
    // 检查是否需要自动刷新
    checkAutoRefresh()
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

// 分页大小改变
const handleSizeChange = (newSize: number) => {
  pageSize.value = newSize
  currentPage.value = 1
  loadSyncRecords()
}

// 当前页改变
const handleCurrentChange = (newPage: number) => {
  currentPage.value = newPage
  loadSyncRecords()
}

// 页面挂载时加载数据
onMounted(() => {
  loadSyncRecords()
})

// 页面卸载时清理定时器
onUnmounted(() => {
  stopAutoRefresh()
})

// 判断记录是否可删除（完成或失败）
const isDeletableRecord = (row: SyncRecord) => row.status === 2 || row.status === 3

// 单条删除，无需确认
const deleteRecord = async (id: number) => {
  try {
    deleteLoading.value = true
    const formData = new FormData()
    formData.append('ids', id.toString())
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
    if (response?.data.code === 200) {
      ElMessage.success('删除成功')
      await loadSyncRecords()
    } else {
      ElMessage.error(response?.data.message || '删除失败')
    }
  } catch {
    ElMessage.error('删除出错')
  } finally {
    deleteLoading.value = false
  }
}

// 批量删除
const batchDeleteRecords = async () => {
  if (selectedIds.value.length === 0) return

  try {
    await ElMessageBox.confirm(
      `确定要批量删除选中的 ${selectedIds.value.length} 条同步记录吗？此操作不可恢复。`,
      '确认批量删除',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )

    batchDeleteLoading.value = true
    const response = await http?.post(
      `${SERVER_URL}/sync/delete-records`,
      {
        ids: selectedIds.value,
      },
      {
        headers: {
          'Content-Type': 'application/json',
        },
        timeout: 60000, // 1分钟超时
      },
    )
    if (response?.data.code === 200) {
      ElMessage.success('批量删除成功')
      selectedIds.value = []
      await loadSyncRecords()
    } else {
      ElMessage.error(response?.data.message || '批量删除失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除出错')
    }
  } finally {
    batchDeleteLoading.value = false
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
  margin-top: 20px;
}

.sync-table {
  width: 100%;
  margin-bottom: 20px;
}

/* 强制固定操作列 */
.sync-table :deep(.el-table__fixed-right) {
  position: sticky !important;
  right: 0 !important;
  z-index: 10 !important;
}

/* 路径列样式 */
.sync-table :deep(.el-table__cell) .cell {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.sync-status {
  margin-top: 16px;
}

.fail-reason-text {
  color: #f56c6c;
  font-size: 12px;
  line-height: 1.4;
  word-break: break-all;
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

  /* 移动端表格滚动优化 */
  .sync-table :deep(.el-table__body-wrapper) {
    overflow-x: auto;
  }

  /* 固定操作列的样式优化 */
  .sync-table :deep(.el-table__fixed-right) {
    box-shadow: -2px 0 8px rgba(0, 0, 0, 0.1);
    background-color: #fff;
  }

  .sync-table :deep(.el-table__fixed-right-patch) {
    background-color: #f5f7fa;
    border-left: 1px solid #ebeef5;
  }

  /* 确保固定列在移动端正确显示 */
  .sync-table :deep(.el-table__fixed-right .el-table__cell) {
    background-color: #fff;
  }

  /* 在移动端隐藏子状态和下载元数据列以节省空间 */
  .sync-table :deep(.hidden-xs) {
    display: none !important;
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

  /* 进一步优化固定列在小屏幕上的显示 */
  .sync-table :deep(.el-table__fixed-right) {
    right: 0 !important;
    box-shadow: -1px 0 4px rgba(0, 0, 0, 0.08);
    z-index: 3;
  }

  .sync-table :deep(.el-table__fixed-right .el-table__cell) {
    background-color: #fff !important;
    border-left: 1px solid #ebeef5;
  }

  /* 确保操作按钮在小屏幕上的大小合适 */
  .sync-table :deep(.el-table__fixed-right .el-button) {
    padding: 4px 8px;
    font-size: 11px;
  }
}
</style>
