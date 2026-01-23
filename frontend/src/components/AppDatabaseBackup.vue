<template>
  <div class="database-backup-container">
    <el-row :gutter="20">
      <!-- 备份配置区 -->
      <el-col :xs="24" :sm="24" :md="24" :lg="24">
        <el-card class="config-card">
          <template #header>
            <div class="card-header">
              <el-icon><Setting /></el-icon>
              <span>备份配置</span>
            </div>
          </template>

          <el-form
            ref="configFormRef"
            :model="configForm"
            label-width="120px"
            :label-position="isMobile ? 'top' : 'right'"
          >
            <el-form-item label="启用自动备份">
              <el-switch
                v-model="configForm.backup_enabled"
                :active-value="1"
                :inactive-value="0"
              />
            </el-form-item>

            <el-form-item label="定时策略" required>
              <cron-selector v-model="configForm.backup_cron" />
            </el-form-item>

            <el-form-item label="备份路径" required>
              <el-input
                v-model="configForm.backup_path"
                placeholder="例如: config/backups/"
              />
            </el-form-item>

            <el-form-item label="保留天数" required>
              <el-input-number
                v-model="configForm.backup_retention"
                :min="1"
                :max="365"
                controls-position="right"
              />
              <span style="margin-left: 8px; color: #909399">天</span>
            </el-form-item>

            <el-form-item label="最大备份数" required>
              <el-input-number
                v-model="configForm.backup_max_count"
                :min="1"
                :max="100"
                controls-position="right"
              />
              <span style="margin-left: 8px; color: #909399">个</span>
            </el-form-item>

            <el-form-item label="压缩备份">
              <el-switch
                v-model="configForm.backup_compress"
                :active-value="1"
                :inactive-value="0"
              />
            </el-form-item>

            <el-form-item>
              <el-button
                type="primary"
                :loading="configSaving"
                @click="saveConfig"
              >
                保存配置
              </el-button>
            </el-form-item>
          </el-form>

          <!-- 下次执行时间提示 -->
          <el-alert
            v-if="nextBackupTime"
            :title="`下次自动备份时间：${nextBackupTime}`"
            type="info"
            :closable="false"
          />
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <!-- 手动备份区 -->
      <el-col :xs="24" :sm="24" :md="12" :lg="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <el-icon><Document /></el-icon>
              <span>手动备份</span>
            </div>
          </template>

          <el-form>
            <el-form-item label="备份原因" label-width="80px">
              <el-input
                v-model="backupReason"
                placeholder="请输入备份原因（必填，最多50字）"
                maxlength="50"
                show-word-limit
              />
            </el-form-item>

            <el-form-item>
              <el-button
                type="primary"
                :loading="backupStarting"
                :disabled="!backupReason || backupStore.isRunning"
                @click="startManualBackup"
              >
                <el-icon><Upload /></el-icon>
                <span>开始备份</span>
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 数据库恢复区 -->
      <el-col :xs="24" :sm="24" :md="12" :lg="12" :style="isMobile ? 'margin-top: 20px' : ''">
        <el-card>
          <template #header>
            <div class="card-header">
              <el-icon><RefreshRight /></el-icon>
              <span>数据库恢复</span>
            </div>
          </template>

          <el-alert
            title="危险操作警告"
            type="warning"
            :closable="false"
            style="margin-bottom: 16px"
          >
            <p>恢复操作将删除所有现有数据！</p>
            <p>系统会自动创建回滚备份。</p>
          </el-alert>

          <el-upload
            ref="uploadRef"
            :auto-upload="false"
            :limit="1"
            :on-change="handleFileChange"
            :on-exceed="handleExceed"
            accept=".sql,.gz"
            drag
          >
            <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
            <div class="el-upload__text">
              拖拽文件到此处或<em>点击选择</em>
            </div>
            <template #tip>
              <div class="el-upload__tip">
                仅支持 .sql 或 .sql.gz 文件，大小不超过1GB
              </div>
            </template>
          </el-upload>

          <el-button
            type="danger"
            :loading="restoreUploading"
            :disabled="!selectedFile || backupStore.isRunning"
            @click="startRestore"
            style="margin-top: 16px; width: 100%"
          >
            <el-icon><RefreshRight /></el-icon>
            <span>开始恢复</span>
          </el-button>
        </el-card>
      </el-col>
    </el-row>

    <!-- 文件管理区 -->
    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :xs="24" :sm="24" :md="24" :lg="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <el-icon><FolderOpened /></el-icon>
              <span>备份文件管理</span>
            </div>
          </template>

          <!-- 提示信息 -->
          <el-alert
            title="备份文件存储在服务器的备份目录中，如需下载请直接访问服务器路径"
            type="info"
            :closable="false"
            style="margin-bottom: 16px"
          />

          <el-tabs v-model="activeTab">
            <!-- 备份文件列表 -->
            <el-tab-pane label="备份文件" name="files">
              <el-button
                type="primary"
                :icon="Refresh"
                @click="loadBackupFiles"
                :loading="filesLoading"
                style="margin-bottom: 16px"
              >
                刷新列表
              </el-button>

              <el-table
                :data="backupFiles"
                v-loading="filesLoading"
                stripe
                style="width: 100%"
              >
                <el-table-column prop="filename" label="文件名" min-width="200" />
                <el-table-column label="文件大小" width="120">
                  <template #default="{ row }">
                    {{ formatFileSize(row.file_size) }}
                  </template>
                </el-table-column>
                <el-table-column label="类型" width="100" v-if="!isMobile">
                  <template #default="{ row }">
                    <el-tag :type="row.backup_type === 'manual' ? 'primary' : 'info'" size="small">
                      {{ row.backup_type === 'manual' ? '手动' : '自动' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="created_reason" label="备份原因" min-width="150" v-if="!isMobile" />
                <el-table-column label="数据库大小" width="120" v-if="!isMobile">
                  <template #default="{ row }">
                    {{ formatFileSize(row.database_size) }}
                  </template>
                </el-table-column>
                <el-table-column label="表数量" width="100" v-if="!isMobile">
                  <template #default="{ row }">
                    {{ row.table_count }}
                  </template>
                </el-table-column>
                <el-table-column label="创建时间" width="180">
                  <template #default="{ row }">
                    {{ formatTimestamp(row.modified_time) }}
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100" fixed="right">
                  <template #default="{ row }">
                    <el-button
                      type="danger"
                      size="small"
                      @click="deleteBackupFile(row.filename)"
                    >
                      删除
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-tab-pane>

            <!-- 历史记录 -->
            <el-tab-pane label="历史记录" name="records">
              <el-button
                type="primary"
                :icon="Refresh"
                @click="loadBackupRecords"
                :loading="recordsLoading"
                style="margin-bottom: 16px"
              >
                刷新列表
              </el-button>

              <el-table
                :data="backupRecords"
                v-loading="recordsLoading"
                stripe
                style="width: 100%"
              >
                <el-table-column prop="id" label="ID" width="80" />
                <el-table-column label="状态" width="120">
                  <template #default="{ row }">
                    <el-tooltip
                      v-if="row.status === 'failed' && row.failure_reason"
                      :content="row.failure_reason"
                      placement="top"
                    >
                      <el-tag :type="getStatusType(row.status)" size="small">
                        {{ getStatusText(row.status) }}
                      </el-tag>
                    </el-tooltip>
                    <el-tag v-else :type="getStatusType(row.status)" size="small">
                      {{ getStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="类型" width="100" v-if="!isMobile">
                  <template #default="{ row }">
                    <el-tag :type="row.backup_type === 'manual' ? 'primary' : 'info'" size="small">
                      {{ row.backup_type === 'manual' ? '手动' : '自动' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="created_reason" label="原因" min-width="150" />
                <el-table-column label="耗时" width="100" v-if="!isMobile">
                  <template #default="{ row }">
                    {{ formatDuration(row.backup_duration) }}
                  </template>
                </el-table-column>
                <el-table-column label="文件大小" width="120" v-if="!isMobile">
                  <template #default="{ row }">
                    {{ row.file_size ? formatFileSize(row.file_size) : '-' }}
                  </template>
                </el-table-column>
                <el-table-column label="压缩比" width="100" v-if="!isMobile">
                  <template #default="{ row }">
                    {{ row.compression_ratio ? (row.compression_ratio * 100).toFixed(1) + '%' : '-' }}
                  </template>
                </el-table-column>
                <el-table-column label="完成时间" width="180">
                  <template #default="{ row }">
                    {{ row.completed_at ? formatTimestamp(row.completed_at) : '-' }}
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100" fixed="right">
                  <template #default="{ row }">
                    <el-button
                      type="danger"
                      size="small"
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
                :page-sizes="[10, 20, 50]"
                :layout="isMobile ? 'prev, pager, next' : 'total, sizes, prev, pager, next, jumper'"
                :size="isMobile ? 'small' : 'default'"
                @current-change="loadBackupRecords"
                @size-change="loadBackupRecords"
                style="margin-top: 16px; justify-content: center"
              />
            </el-tab-pane>
          </el-tabs>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, inject } from 'vue'
import {
  Setting,
  Document,
  Upload,
  RefreshRight,
  UploadFilled,
  FolderOpened,
  Refresh,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type UploadFile, type UploadInstance } from 'element-plus'
import type { AxiosStatic } from 'axios'
import { SERVER_URL } from '@/const'
import { useBackupStore } from '@/stores/backup'
import { formatFileSize } from '@/utils/fileSizeUtils'
import { formatTimestamp, formatDuration } from '@/utils/timeUtils'
import { isMobile as checkIsMobile } from '@/utils/deviceUtils'
import CronSelector from './CronSelector.vue'

type CronParserClass = {
  new (expr: string): {
    reset(): void
    next(): { toDate(): Date }
  }
}

let CronParser: CronParserClass | null = null

// 懒罗加载 cron-parser
const loadCronParser = async () => {
  if (!CronParser) {
    const cronParser = await import('cron-parser')
    CronParser = cronParser.default as unknown as CronParserClass
  }
}
import type {
  BackupConfigResponse,
  BackupFile,
  BackupRecord,
  BackupRecordsResponse,
  BackupStatus,
} from '@/typing'

// 依赖注入
const http = inject<AxiosStatic>('$http')
const backupStore = useBackupStore()
const isMobile = checkIsMobile()

// 配置表单
const configFormRef = ref()
const configForm = reactive({
  backup_enabled: 1,
  backup_cron: '0 2 * * *',
  backup_path: 'config/backups/',
  backup_retention: 7,
  backup_max_count: 10,
  backup_compress: 1,
})
const configSaving = ref(false)
const nextBackupTime = ref('')

// 手动备份
const backupReason = ref('')
const backupStarting = ref(false)

// 数据库恢复
const uploadRef = ref<UploadInstance>()
const selectedFile = ref<File | null>(null)
const restoreUploading = ref(false)

// 文件管理
const activeTab = ref('files')
const backupFiles = ref<BackupFile[]>([])
const filesLoading = ref(false)

// 历史记录
const backupRecords = ref<BackupRecord[]>([])
const recordsLoading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const totalRecords = ref(0)

// 计算下次备份时间
const calculateNextBackupTime = () => {
  try {
    if (!CronParser || !configForm.backup_enabled || !configForm.backup_cron) {
      nextBackupTime.value = ''
      return
    }

    const expression = new CronParser(configForm.backup_cron)
    const next = expression.next().toDate()
    nextBackupTime.value = next.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    })
  } catch {
    nextBackupTime.value = ''
  }
}

// 加载备份配置
const loadBackupConfig = async () => {
  if (!http) return

  try {
    const res = await http.get<{ code: number; data: BackupConfigResponse }>(
      `${SERVER_URL}/database/backup-config`
    )

    if (res.data.code === 200 && res.data.data.exists && res.data.data.config) {
      const config = res.data.data.config
      Object.assign(configForm, {
        backup_enabled: config.backup_enabled,
        backup_cron: config.backup_cron,
        backup_path: config.backup_path,
        backup_retention: config.backup_retention,
        backup_max_count: config.backup_max_count,
        backup_compress: config.backup_compress,
      })
      calculateNextBackupTime()
    }
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : '加载备份配置失败'
    ElMessage.error(errorMsg)
  }
}

// 保存备份配置
const saveConfig = async () => {
  if (!http) return

  configSaving.value = true
  try {
    const res = await http.post(`${SERVER_URL}/database/backup-config`, configForm)

    if (res.data.code === 200) {
      ElMessage.success('备份配置保存成功')
      calculateNextBackupTime()
    } else {
      ElMessage.error('保存配置失败')
    }
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : '保存配置失败'
    ElMessage.error(errorMsg)
  } finally {
    configSaving.value = false
  }
}

// 开始手动备份
const startManualBackup = async () => {
  if (!http) return
  if (!backupReason.value.trim()) {
    ElMessage.warning('请输入备份原因')
    return
  }

  backupStarting.value = true
  try {
    const res = await http.post(`${SERVER_URL}/database/backup/start`, {
      reason: backupReason.value.trim(),
    })

    if (res.data.code === 200) {
      ElMessage.success('备份任务已启动')
      const taskId = res.data.data.task_id
      backupStore.startProgressPolling('backup', taskId, http)
      backupReason.value = ''
    } else {
      ElMessage.error('启动备份任务失败')
    }
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : '启动备份任务失败'
    ElMessage.error(errorMsg)
  } finally {
    backupStarting.value = false
  }
}

// 处理文件选择
const handleFileChange = (file: UploadFile) => {
  selectedFile.value = file.raw || null
}

// 处理文件数量超限
const handleExceed = () => {
  ElMessage.warning('只能上传一个备份文件')
}

// 验证备份文件
const validateBackupFile = (file: File): { valid: boolean; error?: string } => {
  // 验证文件扩展名
  const fileName = file.name.toLowerCase()
  if (!fileName.endsWith('.sql') && !fileName.endsWith('.sql.gz')) {
    return { valid: false, error: '只支持 .sql 或 .sql.gz 格式的备份文件' }
  }

  // 验证文件大小（1GB = 1073741824 bytes）
  if (file.size > 1073741824) {
    return { valid: false, error: '备份文件过大，最大支持1GB' }
  }

  return { valid: true }
}

// 开始恢复
const startRestore = async () => {
  if (!http || !selectedFile.value) return

  // 验证文件
  const validation = validateBackupFile(selectedFile.value)
  if (!validation.valid) {
    ElMessage.error(validation.error || '文件验证失败')
    return
  }

  try {
    // 显示二次确认对话框
    const { value } = await ElMessageBox.prompt(
      '此操作将删除所有现有数据并从备份恢复，不可撤销！\n系统会自动创建回滚备份，如恢复失败将自动回滚。\n\n请输入 RESTORE 确认操作：',
      '危险操作确认',
      {
        confirmButtonText: '确认恢复',
        cancelButtonText: '取消',
        inputPattern: /^RESTORE$/,
        inputErrorMessage: '请输入 RESTORE',
        type: 'warning',
        dangerouslyUseHTMLString: false,
      }
    )

    if (value !== 'RESTORE') {
      return
    }

    // 上传文件并开始恢复
    restoreUploading.value = true
    const formData = new FormData()
    formData.append('backup_file', selectedFile.value)

    const res = await http.post(`${SERVER_URL}/database/restore`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })

    if (res.data.code === 200) {
      ElMessage.success('恢复任务已启动，系统进入维护模式')
      backupStore.startProgressPolling('restore', undefined, http)
      // 清空选择的文件
      selectedFile.value = null
      uploadRef.value?.clearFiles()
    } else {
      ElMessage.error('启动恢复任务失败')
    }
  } catch (error: unknown) {
    const errorMsg = error === 'cancel' ? '' : (error instanceof Error ? error.message : '启动恢复任务失败')
    if (errorMsg) {
      ElMessage.error(errorMsg)
    }
  } finally {
    restoreUploading.value = false
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

// 删除备份文件
const deleteBackupFile = async (filename: string) => {
  if (!http) return

  try {
    await ElMessageBox.confirm(
      `确定要删除备份文件 ${filename} 吗？此操作不可恢复！`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    const res = await http.delete(`${SERVER_URL}/database/backup`, {
      params: { filename },
    })

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

// 删除备份记录
const deleteBackupRecord = async (recordId: number) => {
  if (!http) return

  try {
    await ElMessageBox.confirm(
      '确定要删除该备份记录吗？此操作将同时删除备份文件和关联的回滚备份！',
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    const res = await http.post(`${SERVER_URL}/database/backup-record/delete`, {
      record_id: recordId,
    })

    if (res.data.code === 200) {
      ElMessage.success('备份记录已删除')
      loadBackupRecords()
      loadBackupFiles() // 同时刷新文件列表
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

// 获取状态类型
const getStatusType = (status: BackupStatus) => {
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
      return 'info'
  }
}

// 获取状态文本
const getStatusText = (status: BackupStatus) => {
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
onMounted(async () => {
  await loadCronParser()
  loadBackupConfig()
  loadBackupFiles()
  loadBackupRecords()
})
</script>

<style scoped>
.database-backup-container {
  padding: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
}

.config-card :deep(.el-form-item__label) {
  font-weight: 500;
}

@media (max-width: 768px) {
  .database-backup-container {
    padding: 10px;
  }
}
</style>
