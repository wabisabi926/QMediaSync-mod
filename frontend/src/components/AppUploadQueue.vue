<template>
  <div class="upload-queue-container">
    <div class="card-header">
      <div>
        <h2>上传队列</h2>
        <p>这里包含strm同步时产生的元数据的上传和刮削产生的上传任务。</p>
        <p>列表中只有待上传的记录，如果记录变为上传中，这里就看不见了。</p>
        <p>最多显示100条记录</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="refreshQueue">刷新</el-button>
        <el-button type="danger" @click="clearQueue">清空队列</el-button>
      </div>
    </div>

    <div class="queue-stats">
      <el-statistic :value="total">
        <template #title>
          <div style="display: inline-flex; align-items: center">
            <el-text class="mx-1" type="primary">上传队列中的任务总数</el-text>
          </div>
        </template>
      </el-statistic>
      <el-statistic :value="uploading">
        <template #title>
          <div style="display: inline-flex; align-items: center">
            <el-text class="mx-1" type="info">正在上传的任务总数</el-text>
          </div>
        </template>
      </el-statistic>
    </div>

    <el-table :data="queueData" style="width: 100%" v-loading="loading" empty-text="暂无上传任务"
      :row-class-name="tableRowClassName">
      <!-- <el-table-column prop="ID" label="任务ID" width="80" /> -->
      <el-table-column prop="file_name" label="文件名" min-width="100"></el-table-column>
      <el-table-column prop="status" label="状态" width="80">
        <template #default="scope">
          <el-tag :type="getStatusTagType(scope.row.status)">
            {{ getStatusText(scope.row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="size" label="文件大小" width="100">
        <template #default="scope">
          {{ formatFileSize(scope.row.size) }}
        </template>
      </el-table-column>
      <el-table-column prop="start_time" label="开始时间" width="140" />
      <el-table-column prop="local_path" label="本地文件"></el-table-column>
      <el-table-column prop="remote_path" label="目标文件"></el-table-column>

      <el-table-column label="操作" width="150" fixed="right">
        <!-- <template #default="scope">
            <el-button
              size="small"
              type="primary"
              @click="pauseTask(scope.row)"
              :disabled="scope.row.status !== 'uploading' && scope.row.status !== 'waiting'"
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
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject } from 'vue'
import { formatFileSize } from '@/utils/fileSizeUtils'

interface UploadTask {
  id: string
  file_name: string
  local_path: string
  remote_path: string
  status: number
  size: string
  start_time: string
}

const http: AxiosStatic | undefined = inject('$http')

// 数据状态
const queueData = ref<UploadTask[]>([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const uploading = ref(0)

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

// 加载队列数据
const loadQueueData = async () => {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/upload/queue`, {
      params: {
        page: currentPage.value,
        size: pageSize.value,
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
    await ElMessageBox.confirm('上传中的任务不会删除，只能清空待上传，是否继续？此操作不可恢复。', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })

    const response = await http?.post(`${SERVER_URL}/upload/qudeue/clear-pending`)

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
.upload-queue-container {
  width: 100%;
  height: 100%;
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
