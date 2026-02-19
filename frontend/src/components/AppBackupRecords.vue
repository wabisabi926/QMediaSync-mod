<template>
  <div class="backup-records-container">
    <div class="action-section">
      <el-button
        type="primary"
        size="large"
        :loading="backupStarting"
        :disabled="backupStore.isRunning"
        @click="startManualBackup"
      >
        <el-icon><Upload /></el-icon>
        <span>手动备份</span>
      </el-button>
      <span v-if="backupStore.isRunning" style="margin-left: 12px; color: #909399">
        备份正在进行中...
      </span>
    </div>

    <div class="records-section">
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <el-tab-pane label="备份记录" name="records">
          <el-table
            :data="backupRecords"
            v-loading="recordsLoading"
            :height="isMobile ? 'auto' : 400"
            style="width: 100%"
          >
            <el-table-column
              prop="id"
              label="ID"
              width="80"
            />
            <el-table-column
              prop="status"
              label="状态"
              width="100"
            >
              <template #default="{ row }">
                <el-tag :type="getStatusTagType(row.status)" size="small">
                  {{ getStatusText(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column
              v-if="!isMobile"
              prop="file_size"
              label="文件大小"
              width="120"
            >
              <template #default="{ row }">
                {{ row.file_size ? formatFileSize(row.file_size) : '-' }}
              </template>
            </el-table-column>
            <el-table-column
              prop="backup_duration"
              label="耗时"
              width="100"
            >
              <template #default="{ row }">
                {{ row.backup_duration ? formatDuration(row.backup_duration) : '-' }}
              </template>
            </el-table-column>
            <el-table-column
              prop="backup_type"
              label="类型"
              width="100"
            >
              <template #default="{ row }">
                <el-tag :type="row.backup_type === 'manual' ? 'primary' : 'info'" size="small">
                  {{ row.backup_type === 'manual' ? '手动' : '自动' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column
              v-if="!isMobile"
              prop="created_reason"
              label="原因"
              min-width="120"
              show-overflow-tooltip
            />
            <el-table-column
              prop="created_at"
              label="创建时间"
              :width="isMobile ? 100 : 180"
            >
              <template #default="{ row }">
                {{ formatTimestamp(row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column
              label="操作"
              :width="isMobile ? 120 : 150"
              fixed="right"
            >
              <template #default="{ row }">
                <el-button
                  v-if="row.status === 'completed'"
                  type="primary"
                  size="small"
                  link
                  @click="downloadBackup(row.id, getFilenameFromPath(row.file_path))"
                >
                  下载
                </el-button>
                <el-button
                  type="danger"
                  size="small"
                  link
                  @click="deleteBackupRecord(row.id)"
                >
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :total="totalRecords"
            :layout="isMobile ? 'prev, pager, next' : 'total, prev, pager, next, jumper'"
            :size="isMobile ? 'small' : 'default'"
            :page-sizes="[10, 20, 50, 100]"
            style="margin-top: 16px; justify-content: center"
            @current-change="loadBackupRecords"
            @size-change="handlePageSizeChange"
          />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, inject } from 'vue'
import { Upload } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { AxiosStatic } from 'axios'
import { SERVER_URL } from '@/const'
import { useBackupStore } from '@/stores/backup'
import type {
  BackupRecordListItem,
  BackupRecordsResponse,
  BackupStatus,
} from '@/typing'
import { formatFileSize } from '@/utils/fileSizeUtils'
import { formatTimestamp, formatDuration } from '@/utils/timeUtils'
import { isMobile as checkIsMobile } from '@/utils/deviceUtils'

const http = inject<AxiosStatic>('$http')
const backupStore = useBackupStore()
const isMobile = checkIsMobile()
const API_SUCCESS_CODE = 200

const activeTab = ref('records')
const backupStarting = ref(false)
const recordsLoading = ref(false)
const backupRecords = ref<BackupRecordListItem[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const totalRecords = ref(0)

const startManualBackup = async () => {
  if (!http) return

  backupStarting.value = true
  try {
    const res = await http.post(`${SERVER_URL}/backup/create`, {
      reason: '手动备份',
    })

    if (res.data.code === API_SUCCESS_CODE) {
      ElMessage.success('备份任务已启动')
      backupStore.startProgressPolling('backup', undefined, http)
      setTimeout(() => {
        loadBackupRecords()
      }, 2000)
    } else {
      ElMessage.error(res.data.message || '启动备份任务失败')
    }
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : '启动备份任务失败'
    ElMessage.error(errorMsg)
  } finally {
    backupStarting.value = false
  }
}

const loadBackupRecords = async () => {
  if (!http) return

  recordsLoading.value = true
  try {
    const res = await http.get<{ code: number; data: BackupRecordsResponse }>(
      `${SERVER_URL}/backup/list`,
      {
        params: {
          page: currentPage.value,
          page_size: pageSize.value,
          type: 'all',
        },
      }
    )

    if (res.data.code === API_SUCCESS_CODE) {
      backupRecords.value = res.data.data.list
      totalRecords.value = res.data.data.total
    } else {
      ElMessage.error('加载备份记录失败')
    }
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : '加载备份记录失败'
    ElMessage.error(errorMsg)
  } finally {
    recordsLoading.value = false
  }
}

const handlePageSizeChange = () => {
  currentPage.value = 1
  loadBackupRecords()
}

const handleTabChange = () => {
  loadBackupRecords()
}

const getFilenameFromPath = (filePath: string): string => {
  if (!filePath) return 'backup.sql.zip'
  return filePath.split('/').pop() || 'backup.sql.zip'
}

const downloadBackup = async (recordId: number, filename: string) => {
  if (!http) return

  try {
    const res = await http.get(`${SERVER_URL}/backup/download/${recordId}`, {
      responseType: 'blob',
    })

    const url = window.URL.createObjectURL(new Blob([res.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', filename)
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : '下载备份文件失败'
    ElMessage.error(errorMsg)
  }
}

const deleteBackupRecord = async (recordId: number) => {
  try {
    await ElMessageBox.confirm(
      '确定要删除此备份记录吗？相关的备份文件也将被删除。',
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    if (!http) return

    const res = await http.delete(`${SERVER_URL}/backup/records/${recordId}`)

    if (res.data.code === API_SUCCESS_CODE) {
      ElMessage.success('备份记录已删除')
      loadBackupRecords()
    } else {
      ElMessage.error(res.data.message || '删除备份记录失败')
    }
  } catch (error: unknown) {
    if (error !== 'cancel') {
      const errorMsg = error instanceof Error ? error.message : '删除备份记录失败'
      ElMessage.error(errorMsg)
    }
  }
}

const getStatusTagType = (status: BackupStatus): string => {
  switch (status) {
    case 'completed':
      return 'success'
    case 'failed':
      return 'danger'
    case 'cancelled':
      return 'info'
    case 'timeout':
      return 'warning'
    default:
      return ''
  }
}

const getStatusText = (status: BackupStatus): string => {
  switch (status) {
    case 'completed':
      return '成功'
    case 'failed':
      return '失败'
    case 'cancelled':
      return '已取消'
    case 'timeout':
      return '超时'
    case 'running':
      return '运行中'
    case 'pending':
      return '等待中'
    default:
      return status
  }
}

onMounted(() => {
  loadBackupRecords()
})
</script>

<style scoped>
.backup-records-container {
  padding: 20px;
}

.action-section {
  margin-bottom: 20px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 4px;
  max-width: 1200px;
}

.records-section {
  margin-bottom: 20px;
  max-width: 1400px;
}

@media (max-width: 768px) {
  .backup-records-container {
    padding: 10px;
  }

  .action-section {
    padding: 12px;
  }
}
</style>
