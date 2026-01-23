<template>
  <div class="backup-records-container">
    <!-- 手动备份按钮 -->
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

    <!-- 备份文件与历史记录 -->
    <div class="records-section">
      <el-tabs v-model="activeTab">
        <!-- 备份文件列表 -->
        <el-tab-pane label="备份文件" name="files">
          <el-alert
            title="备份文件存储在服务器的备份目录中，如需下载请直接访问服务器路径"
            type="info"
            :closable="false"
            style="margin-bottom: 16px"
          />

          <el-table
            :data="backupFiles"
            v-loading="filesLoading"
            :height="isMobile ? 'auto' : 400"
            style="width: 100%"
          >
            <el-table-column
              prop="filename"
              label="文件名"
              :min-width="isMobile ? 150 : 250"
              show-overflow-tooltip
            />
            <el-table-column
              prop="file_size"
              label="文件大小"
              :width="isMobile ? 100 : 120"
            >
              <template #default="{ row }">
                {{ formatFileSize(row.file_size) }}
              </template>
            </el-table-column>
            <el-table-column
              v-if="!isMobile"
              prop="table_count"
              label="表数量"
              width="100"
            />
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
              prop="modified_time"
              label="创建时间"
              :width="isMobile ? 100 : 180"
            >
              <template #default="{ row }">
                {{ formatTimestamp(row.modified_time) }}
              </template>
            </el-table-column>
            <el-table-column
              label="操作"
              :width="isMobile ? 80 : 100"
              fixed="right"
            >
              <template #default="{ row }">
                <el-button
                  type="danger"
                  size="small"
                  link
                  @click="deleteBackupFile(row.filename)"
                >
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 备份历史记录 -->
        <el-tab-pane label="历史记录" name="records">
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
                <el-tooltip
                  v-if="row.status === 'failed' && row.failure_reason"
                  :content="row.failure_reason"
                  placement="top"
                >
                  <el-tag :type="getStatusTagType(row.status)" size="small">
                    {{ getStatusText(row.status) }}
                  </el-tag>
                </el-tooltip>
                <el-tag v-else :type="getStatusTagType(row.status)" size="small">
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
              prop="completed_at"
              label="完成时间"
              :width="isMobile ? 100 : 180"
            >
              <template #default="{ row }">
                {{ row.completed_at ? formatTimestamp(row.completed_at) : '-' }}
              </template>
            </el-table-column>
            <el-table-column
              label="操作"
              :width="isMobile ? 80 : 100"
              fixed="right"
            >
              <template #default="{ row }">
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

          <!-- 分页 -->
          <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :total="totalRecords"
            :layout="isMobile ? 'prev, pager, next' : 'total, prev, pager, next, jumper'"
            :size="isMobile ? 'small' : 'default'"
            style="margin-top: 16px; justify-content: center"
            @current-change="loadBackupRecords"
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
  BackupFile,
  BackupRecord,
  BackupRecordsResponse,
  BackupStatus,
} from '@/typing'
import { formatFileSize } from '@/utils/fileSizeUtils'
import { formatTimestamp, formatDuration } from '@/utils/timeUtils'
import { isMobile as checkIsMobile } from '@/utils/deviceUtils'

const http = inject<AxiosStatic>('$http')
const backupStore = useBackupStore()
const isMobile = checkIsMobile()

const activeTab = ref('files')
const backupStarting = ref(false)
const filesLoading = ref(false)
const recordsLoading = ref(false)
const backupFiles = ref<BackupFile[]>([])
const backupRecords = ref<BackupRecord[]>([])
const currentPage = ref(1)
const pageSize = ref(10)
const totalRecords = ref(0)

// 启动手动备份（默认原因为"手动备份"）
const startManualBackup = async () => {
  if (!http) return

  backupStarting.value = true
  try {
    const res = await http.post(`${SERVER_URL}/database/backup/start`, {
      reason: '手动备份',
    })

    if (res.data.code === 200) {
      ElMessage.success('备份任务已启动')
      const taskId = res.data.data.task_id
      backupStore.startProgressPolling('backup', taskId, http)
      // 刷新列表
      setTimeout(() => {
        loadBackupFiles()
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

// 加载备份文件列表
const loadBackupFiles = async () => {
  if (!http) return

  filesLoading.value = true
  try {
    const res = await http.get<{ code: number; data: BackupFile[] }>(
      `${SERVER_URL}/database/backups`
    )

    if (res.data.code === 200) {
      backupFiles.value = res.data.data
    } else {
      ElMessage.error('加载备份文件列表失败')
    }
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : '加载备份文件列表失败'
    ElMessage.error(errorMsg)
  } finally {
    filesLoading.value = false
  }
}

// 加载备份记录
const loadBackupRecords = async () => {
  if (!http) return

  recordsLoading.value = true
  try {
    const res = await http.get<{ code: number; data: BackupRecordsResponse }>(
      `${SERVER_URL}/database/backup-records`,
      {
        params: {
          page: currentPage.value,
          page_size: pageSize.value,
        },
      }
    )

    if (res.data.code === 200) {
      backupRecords.value = res.data.data.records
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

// 删除备份文件
const deleteBackupFile = async (filename: string) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除备份文件 ${filename} 吗？`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    if (!http) return

    const res = await http.delete(
      `${SERVER_URL}/database/backup?filename=${encodeURIComponent(filename)}`
    )

    if (res.data.code === 200) {
      ElMessage.success('备份文件已删除')
      loadBackupFiles()
    } else {
      ElMessage.error('删除备份文件失败')
    }
  } catch (error: unknown) {
    if (error !== 'cancel') {
      const errorMsg = error instanceof Error ? error.message : '删除备份文件失败'
      ElMessage.error(errorMsg)
    }
  }
}

// 删除备份记录
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

    const res = await http.post(`${SERVER_URL}/database/backup-record/delete`, {
      record_id: recordId,
    })

    if (res.data.code === 200) {
      ElMessage.success('备份记录已删除')
      loadBackupRecords()
      loadBackupFiles()
    } else {
      ElMessage.error('删除备份记录失败')
    }
  } catch (error: unknown) {
    if (error !== 'cancel') {
      const errorMsg = error instanceof Error ? error.message : '删除备份记录失败'
      ElMessage.error(errorMsg)
    }
  }
}

// 获取状态标签类型
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

// 获取状态文本
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
    default:
      return status
  }
}

// 组件挂载时加载数据
onMounted(() => {
  loadBackupFiles()
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
