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
                    <div v-for="(path, index) in scope.row.left" :key="index" class="file-path">
                      {{ path }}
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="right" label="本地文件" min-width="300">
                <template #default="scope">
                  <div class="file-paths">
                    <div v-for="(path, index) in scope.row.right" :key="index" class="file-path">
                      {{ path }}
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

        <!-- 元数据下载列表 -->
        <div class="download-table">
          <h3>元数据下载</h3>
          <div class="table-container" v-loading="downloadLoading">
            <el-table
              :data="downloadData"
              stripe
              class="download-list-table"
              empty-text="暂无下载数据"
              max-height="500"
            >
              <el-table-column prop="local_path" label="文件路径" min-width="400">
                <template #default="scope">
                  <div class="file-path">
                    {{ scope.row.local_path }}
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="size" label="文件大小" width="120" align="center">
                <template #default="scope">
                  {{ formatFileSize(scope.row.size) }}
                </template>
              </el-table-column>
              <el-table-column prop="download_status" label="下载状态" width="120" align="center">
                <template #default="scope">
                  <el-tag :type="getDownloadStatusType(scope.row.download_status)" size="small">
                    {{ getDownloadStatusText(scope.row.download_status) }}
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>

            <!-- 分页器 -->
            <el-pagination
              v-if="downloadTotal > 0"
              v-model:current-page="downloadCurrentPage"
              v-model:page-size="downloadPageSize"
              :total="downloadTotal"
              :page-sizes="[50, 100, 200]"
              layout="total, sizes, prev, pager, next, jumper"
              class="download-pagination"
              @size-change="handleDownloadPageSizeChange"
              @current-change="handleDownloadPageChange"
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
import { ArrowLeft } from '@element-plus/icons-vue'
import { inject, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

// 任务详情数据结构
interface TaskInfo {
  id: number
  start_time: number
  end_time: number | null
  status: 0 | 1 | 2 | 3 // 0-待开始，1-运行中，2-完成，3-失败
  sub_status: 0 | 1 | 2 | 3 | 4 | 5 | 6 // 0-未开始，1-正在收集要同步的文件，2-正在查询基础目录结构，3-正在补全文件的路径，4-正在对比文件，5-正在新增和更新文件，6-正在下载元数据
  processed_files: number
  created_strm: number
  downloaded_meta: number
}

// 定义文件对比项接口
interface CompareItem {
  left: string[]
  right: string[]
  status: number
}

// 定义元数据下载项接口
interface DownloadItem {
  local_path: string
  size: number
  download_status: number
}

const http: AxiosStatic | undefined = inject('$http')
const route = useRoute()
const router = useRouter()

// 获取任务ID
const taskId = ref(route.params.id as string)

// 数据状态
const taskInfo = ref<TaskInfo | null>(null)
const compareData = ref<CompareItem[]>([])
const downloadData = ref<DownloadItem[]>([])
const infoLoading = ref(false)
const logsLoading = ref(false)
const downloadLoading = ref(false)

// 对比数据分页
const compareCurrentPage = ref(1)
const comparePageSize = ref(100)
const compareTotal = ref(0)

// 下载数据分页
const downloadCurrentPage = ref(1)
const downloadPageSize = ref(100)
const downloadTotal = ref(0)

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
const getCompareStatusType = (status: number) => {
  switch (status) {
    case 0:
      return 'success' // 新增
    case 1:
      return 'warning' // 更新
    case 2:
      return 'danger' // 删除
    case 3:
      return 'info' // 不处理
    default:
      return 'info'
  }
}

// 获取对比状态文本
const getCompareStatusText = (status: number) => {
  switch (status) {
    case 0:
      return '新增'
    case 1:
      return '更新'
    case 2:
      return '删除'
    case 3:
      return '不处理'
    default:
      return '未知'
  }
}

// 获取下载状态标签类型
const getDownloadStatusType = (status: number) => {
  switch (status) {
    case 0:
      return 'info' // 不用下载
    case 1:
      return 'warning' // 待下载
    case 2:
      return 'success' // 下载完成
    case 3:
      return 'danger' // 下载失败
    case 4:
      return 'primary' // 下载中
    default:
      return 'info'
  }
}

// 获取下载状态文本
const getDownloadStatusText = (status: number) => {
  switch (status) {
    case 0:
      return '不用下载'
    case 1:
      return '待下载'
    case 2:
      return '下载完成'
    case 3:
      return '下载失败'
    case 4:
      return '下载中'
    default:
      return '未知'
  }
}

// 格式化文件大小
const formatFileSize = (size: number) => {
  if (size === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(size) / Math.log(k))
  return parseFloat((size / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
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
        sub_status: data.sub_status as 0 | 1 | 2 | 3 | 4 | 5 | 6,
        processed_files: data.total,
        created_strm: data.new_strm,
        downloaded_meta: data.new_meta || 0,
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

// 加载元数据下载数据
const loadDownloadData = async () => {
  try {
    downloadLoading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/task/download?sync_id=${taskId.value}`, {
      params: {
        page: downloadCurrentPage.value,
        pageSize: downloadPageSize.value,
      },
    })

    if (response?.data.code === 200) {
      const data = response.data.data
      downloadData.value = data.list || []
      downloadTotal.value = data.total || 0
    } else {
      downloadData.value = []
      downloadTotal.value = 0
    }
  } catch (error) {
    console.error('加载元数据下载数据错误:', error)
    downloadData.value = []
    downloadTotal.value = 0
  } finally {
    downloadLoading.value = false
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

// 下载数据分页处理
const handleDownloadPageSizeChange = (newSize: number) => {
  downloadPageSize.value = newSize
  downloadCurrentPage.value = 1
  loadDownloadData()
}

const handleDownloadPageChange = (newPage: number) => {
  downloadCurrentPage.value = newPage
  loadDownloadData()
}

// 页面挂载时加载数据
onMounted(() => {
  loadTaskInfo()
  loadCompareData()
  loadDownloadData()
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

.file-compare-table h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.download-table {
  margin-bottom: 30px;
}

.download-table h3 {
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

.download-list-table {
  width: 100%;
  margin-bottom: 20px;
}

/* 为下载表格列添加竖线分隔 */
.download-list-table :deep(.el-table__cell) {
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

.download-pagination {
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
