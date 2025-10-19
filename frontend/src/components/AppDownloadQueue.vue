<template>
  <div class="download-queue-container">
    <el-card class="download-queue-card" shadow="never">
      <template #header>
        <div class="card-header">
          <h2>下载队列</h2>
          <div class="header-actions">
            <el-button type="primary" @click="refreshQueue">刷新</el-button>
            <el-button type="danger" @click="clearQueue" :disabled="queueData.length === 0">清空队列</el-button>
          </div>
        </div>
      </template>

      <el-table
        :data="queueData"
        style="width: 100%"
        v-loading="loading"
        empty-text="暂无下载任务"
        :row-class-name="tableRowClassName"
      >
        <el-table-column prop="ID" label="任务ID" width="180" />
        <el-table-column prop="source_type" label="下载来源" width="80" />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="scope">
            <el-tag :type="getStatusTagType(scope.row.status)">
              {{ getStatusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="file_size" label="文件大小" width="120" />
        <el-table-column prop="file_name" label="文件名">
          <template #default="scope">
            <span class="filename">{{ scope.row.file_name }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="speed" label="下载链接">
          <template #default="scope">
            <span>{{ scope.row.url }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="dest_path" label="保存路径" />
        <!-- <el-table-column label="操作" width="150" fixed="right"> -->
          <!-- <template #default="scope">
            <el-button
              size="small"
              type="primary"
              @click="pauseTask(scope.row)"
              :disabled="scope.row.status !== 'downloading' && scope.row.status !== 'waiting'"
            >
              {{ scope.row.status === 'paused' ? '继续' : '暂停' }}
            </el-button>
            <el-button
              size="small"
              type="danger"
              @click="cancelTask(scope.row)"
            >
              取消
            </el-button>
          </template> -->
        <!-- </el-table-column> -->
      </el-table>

      <div class="pagination-container">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject } from 'vue'

interface DownloadTask {
  ID: string
  file_name: string
  dest_path: string
  status: 0 | 1 | 2 | 3 | 4
  file_size: number
  progress: number
  start_time: string
  end_time: string
  url: string
}

const http: AxiosStatic | undefined = inject('$http')

// 数据状态
const queueData = ref<DownloadTask[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 定时器
const refreshTimer = ref<number | null>(null)

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
const getStatusTagType = (status: number): 'primary' | 'success' | 'warning' | 'danger' | 'info' => {
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

// 加载队列数据
const loadQueueData = async () => {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/download/queue`, {
      params: {
        page: currentPage.value,
        size: pageSize.value
      }
    })

    if (response?.data.code === 200) {
      queueData.value = response.data.data
      total.value = response.data.data.length
    } else {
      ElMessage.error('获取下载队列数据失败')
    }
  } catch (error) {
    console.error('加载下载队列数据错误:', error)
    ElMessage.error('加载下载队列数据失败')
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
    await ElMessageBox.confirm('确定要清空所有下载任务吗？此操作不可恢复。', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })

    const response = await http?.post(`${SERVER_URL}/download/queue/clear`)

    if (response?.data.code === 200) {
      ElMessage.success('队列已清空')
      loadQueueData()
    } else {
      ElMessage.error('清空队列失败')
    }
  } catch {
    // 用户取消或请求失败
  }
}

// // 暂停/继续任务
// const pauseTask = async (task: DownloadTask) => {
//   try {
//     const action = task.status === 'paused' ? 'resume' : 'pause'
//     const response = await http?.post(`${SERVER_URL}/download/task/${task.id}/${action}`)

//     if (response?.data.code === 200) {
//       ElMessage.success(`${task.status === 'paused' ? '继续' : '暂停'}任务成功`)
//       loadQueueData()
//     } else {
//       ElMessage.error(`${task.status === 'paused' ? '继续' : '暂停'}任务失败`)
//     }
//   } catch (error) {
//     console.error('操作任务失败:', error)
//     ElMessage.error(`${task.status === 'paused' ? '继续' : '暂停'}任务失败`)
//   }
// }

// // 取消任务
// const cancelTask = async (task: DownloadTask) => {
//   try {
//     await ElMessageBox.confirm(`确定要取消下载任务 "${task.filename}" 吗？`, '提示', {
//       confirmButtonText: '确定',
//       cancelButtonText: '取消',
//       type: 'warning'
//     })

//     const response = await http?.post(`${SERVER_URL}/download/task/${task.id}/cancel`)

//     if (response?.data.code === 200) {
//       ElMessage.success('任务已取消')
//       loadQueueData()
//     } else {
//       ElMessage.error('取消任务失败')
//     }
//   } catch {
//     // 用户取消或请求失败
//   }
// }

// 分页处理
const handleSizeChange = (val: number) => {
  pageSize.value = val
  loadQueueData()
}

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

// 页面生命周期
onMounted(() => {
  loadQueueData()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.download-queue-container {
  width: 100%;
  height: 100%;
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
  gap: 12px;
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
    justify-content: space-between;
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
