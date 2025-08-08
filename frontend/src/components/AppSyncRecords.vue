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
            <el-button
              type="primary"
              @click="startManualSync"
              :loading="syncLoading"
              :disabled="hasRunningSyncTask"
              size="large"
            >
              <el-icon><Refresh /></el-icon>
              手动同步
            </el-button>
          </div>
        </div>
      </template>

      <div class="sync-content">
        <!-- 同步记录表格 -->
        <el-table
          :data="syncRecords"
          v-loading="tableLoading"
          stripe
          class="sync-table"
          empty-text="暂无同步记录"
        >
          <el-table-column prop="id" label="任务ID" width="80" />
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
          <el-table-column prop="status" label="状态" width="120">
            <template #default="scope">
              <el-tag
                :type="getStatusType(scope.row.status)"
                :effect="scope.row.status === 1 ? 'dark' : 'light'"
              >
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="sub_status" label="子状态" width="180">
            <template #default="scope">
              <el-tag v-if="scope.row.status === 1" type="primary" size="small" effect="light">
                {{ getSubStatusText(scope.row.sub_status) }}
              </el-tag>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column prop="processed_files" label="总文件数" width="120" align="center" />
          <el-table-column prop="created_strm" label="新增STRM数" width="120" align="center" />
          <el-table-column prop="downloaded_meta" label="下载元数据数" width="140" align="center" />
          <el-table-column label="操作" width="100" align="center">
            <template #default="scope">
              <el-button type="primary" size="small" @click="viewTaskDetail(scope.row.id)" link>
                查看
              </el-button>
            </template>
          </el-table-column>
        </el-table>

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
    </el-card>

    <!-- 同步状态提示 -->
    <el-alert
      v-if="syncStatus"
      :title="syncStatus.title"
      :type="syncStatus.type"
      :description="syncStatus.description"
      :closable="false"
      show-icon
      class="sync-status"
    />
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { Refresh } from '@element-plus/icons-vue'
import { inject, onMounted, onUnmounted, ref, computed } from 'vue'
import { useRouter } from 'vue-router'

interface SyncRecord {
  id: number
  start_time: number
  end_time: number | null
  status: 0 | 1 | 2 | 3 // 0-待开始，1-运行中，2-完成，3-失败
  sub_status: 0 | 1 | 2 | 3 | 4 | 5 | 6 // 0-未开始，1-正在收集要同步的文件，2-正在查询基础目录结构，3-正在补全文件的路径，4-正在对比文件，5-正在新增和更新文件，6-正在下载元数据
  processed_files: number
  created_strm: number
  downloaded_meta: number
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
}

const http: AxiosStatic | undefined = inject('$http')
const router = useRouter()

// 数据状态
const syncRecords = ref<SyncRecord[]>([])
const tableLoading = ref(false)
const syncLoading = ref(false)
const syncStatus = ref<SyncStatus | null>(null)

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
      return '未开始'
    case 1:
      return '正在收集要同步的文件'
    case 2:
      return '正在查询基础目录结构'
    case 3:
      return '正在补全文件的路径'
    case 4:
      return '正在对比文件'
    case 5:
      return '正在新增和更新文件'
    case 6:
      return '正在下载元数据'
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
        sub_status: item.sub_status as 0 | 1 | 2 | 3 | 4 | 5 | 6,
        processed_files: item.total,
        created_strm: item.new_strm,
        downloaded_meta: item.new_meta || 0,
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

// 手动开始同步
const startManualSync = async () => {
  try {
    syncLoading.value = true
    syncStatus.value = null

    const response = await http?.post(`${SERVER_URL}/sync/start`)

    if (response?.data.code === 200) {
      syncStatus.value = {
        title: '同步任务已启动',
        type: 'success',
        description: '手动同步任务已成功启动，请稍后查看同步记录',
      }
      // 重新加载记录
      await loadSyncRecords()
      // 启动自动刷新
      startAutoRefresh()
    } else {
      syncStatus.value = {
        title: '启动同步失败',
        type: 'error',
        description: response?.data.msg || '启动同步任务失败，请重试',
      }
    }
  } catch (error) {
    console.error('启动同步错误:', error)
    syncStatus.value = {
      title: '启动同步出错',
      type: 'error',
      description: '启动同步过程中发生错误，请检查网络连接',
    }
  } finally {
    syncLoading.value = false
  }
}

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
