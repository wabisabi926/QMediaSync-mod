<template>
  <div class="media-import-container">
    <!-- 媒体库导入卡片 -->
    <el-card class="media-import-card" shadow="hover">
      <template #header>
        <h2 class="card-title">媒体库导入</h2>
        <p class="card-subtitle">从媒体库管理工具导入文件到115网盘</p>
      </template>

      <div class="import-content">
        <!-- 导入功能 -->
        <div class="import-section">
          <h3 class="section-title">
            <el-icon><FolderOpened /></el-icon>
            媒体库导入
          </h3>
          <p class="section-description">支持从Jellyfin、Emby、Plex等媒体库管理工具导入文件</p>

          <el-form :model="importData" :label-position="'top'" class="import-form">
            <el-form-item label="媒体库路径" required>
              <el-input
                v-model="importData.library_path"
                type="textarea"
                :rows="3"
                placeholder="请输入媒体库路径，支持多个路径，每行一个"
                :disabled="importing"
              />
              <div class="form-help">输入媒体库的文件夹路径，系统将扫描这些路径下的媒体文件</div>
            </el-form-item>

            <el-form-item label="文件类型过滤">
              <el-checkbox-group v-model="importData.file_types" :disabled="importing">
                <el-checkbox label="video">视频文件 (.mp4, .mkv, .avi, .mov等)</el-checkbox>
                <el-checkbox label="audio">音频文件 (.mp3, .flac, .wav, .aac等)</el-checkbox>
                <el-checkbox label="subtitle">字幕文件 (.srt, .ass, .vtt等)</el-checkbox>
                <el-checkbox label="image">图像文件 (.jpg, .png, .gif等)</el-checkbox>
              </el-checkbox-group>
              <div class="form-help">选择要导入的文件类型，不选择表示导入所有支持的文件类型</div>
            </el-form-item>

            <el-form-item label="目标目录">
              <el-input
                v-model="importData.target_dir"
                placeholder="目标目录路径，留空表示根目录"
                :disabled="importing"
                clearable
              />
              <div class="form-help">
                上传到115网盘的目标目录，例如：/媒体库，留空表示上传到根目录
              </div>
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="startImport" :loading="importing" size="large">
                <el-icon><Upload /></el-icon>
                开始导入
              </el-button>

              <el-button type="warning" @click="resetForm" :disabled="importing" size="large">
                <el-icon><RefreshLeft /></el-icon>
                重置表单
              </el-button>
            </el-form-item>
          </el-form>
        </div>

        <!-- 操作状态显示 -->
        <el-alert
          v-if="importStatus"
          :title="importStatus.title"
          :type="importStatus.type"
          :description="importStatus.description"
          :closable="false"
          show-icon
          class="import-status"
        />

        <!-- 导入进度 -->
        <div v-if="importProgress.visible" class="progress-section">
          <h4>导入进度</h4>
          <el-progress
            :percentage="importProgress.percentage"
            :status="importProgress.status"
            :show-text="true"
          />
          <p class="progress-text">{{ importProgress.text }}</p>
        </div>
      </div>
    </el-card>

    <!-- 使用说明卡片 -->
    <el-card class="usage-card" shadow="hover">
      <template #header>
        <h2 class="card-title">使用说明</h2>
        <p class="card-subtitle">如何配置媒体库导入</p>
      </template>

      <div class="usage-content">
        <div class="usage-section">
          <h4>支持的媒体库：</h4>
          <ul>
            <li><strong>Jellyfin：</strong>开源媒体服务器</li>
            <li><strong>Emby：</strong>媒体服务器</li>
            <li><strong>Plex：</strong>媒体服务器</li>
            <li><strong>本地文件夹：</strong>任意包含媒体文件的文件夹</li>
          </ul>
        </div>

        <div class="usage-section">
          <h4>路径格式示例：</h4>
          <ul>
            <li>Windows: <code>D:\Movies</code>, <code>E:\TV Shows</code></li>
            <li>Linux/macOS: <code>/media/movies</code>, <code>/media/tv</code></li>
            <li>网络路径: <code>\\\\192.168.1.100\\media</code></li>
          </ul>
        </div>

        <div class="usage-section">
          <h4>注意事项：</h4>
          <ul>
            <li>确保服务器有权限访问指定的媒体库路径</li>
            <li>大型媒体库的扫描可能需要较长时间</li>
            <li>建议在网络空闲时进行大批量导入</li>
            <li>重复文件会自动跳过，不会重复上传</li>
          </ul>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { Upload, FolderOpened, RefreshLeft } from '@element-plus/icons-vue'
import { inject, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

interface ImportData {
  library_path: string
  file_types: string[]
  target_dir: string
}

interface ImportStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}

interface ImportProgress {
  visible: boolean
  percentage: number
  status: 'success' | 'exception' | 'warning' | undefined
  text: string
}

const http: AxiosStatic | undefined = inject('$http')

// 状态管理
const importing = ref(false)
const importStatus = ref<ImportStatus | null>(null)
const importProgress = ref<ImportProgress>({
  visible: false,
  percentage: 0,
  status: undefined,
  text: '',
})

// 导入数据
const importData = reactive<ImportData>({
  library_path: '',
  file_types: ['video', 'audio'],
  target_dir: '',
})

// 开始导入
const startImport = async () => {
  if (!validateImport()) {
    return
  }

  try {
    importing.value = true
    importStatus.value = null
    importProgress.value.visible = true
    importProgress.value.percentage = 0
    importProgress.value.text = '准备扫描媒体库...'

    const requestData = {
      library_path: importData.library_path,
      file_types: importData.file_types.join(','),
      target_dir: importData.target_dir || '/',
    }

    importProgress.value.percentage = 30
    importProgress.value.text = '正在扫描文件...'

    const response = await http?.post(`${SERVER_URL}/import/media-library`, requestData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    importProgress.value.percentage = 100

    if (response?.data.code === 200) {
      importProgress.value.status = 'success'
      importProgress.value.text = '媒体库导入完成！'

      const successCount = response.data.data?.success_count || 0
      const totalCount = response.data.data?.total_count || 0
      const skippedCount = response.data.data?.skipped_count || 0

      importStatus.value = {
        title: '媒体库导入完成',
        type: successCount === totalCount ? 'success' : 'warning',
        description: `共扫描 ${totalCount} 个文件，成功导入 ${successCount} 个，跳过 ${skippedCount} 个`,
      }
      ElMessage.success(`媒体库导入完成！成功 ${successCount}/${totalCount} 个文件`)
    } else {
      importProgress.value.status = 'exception'
      importProgress.value.text = '媒体库导入失败'
      importStatus.value = {
        title: '媒体库导入失败',
        type: 'error',
        description: response?.data.message || '导入过程中发生错误，请检查媒体库路径',
      }
    }
  } catch (error) {
    console.error('媒体库导入错误:', error)
    importProgress.value.status = 'exception'
    importProgress.value.text = '媒体库导入失败'
    importStatus.value = {
      title: '媒体库导入出错',
      type: 'error',
      description: '导入过程中发生网络错误，请检查网络连接',
    }
  } finally {
    importing.value = false
    setTimeout(() => {
      importProgress.value.visible = false
    }, 3000)
  }
}

// 验证导入数据
const validateImport = (): boolean => {
  if (!importData.library_path.trim()) {
    ElMessage.warning('请输入媒体库路径')
    return false
  }

  if (importData.file_types.length === 0) {
    ElMessage.warning('请选择至少一种文件类型')
    return false
  }

  return true
}

// 重置表单
const resetForm = () => {
  importData.library_path = ''
  importData.file_types = ['video', 'audio']
  importData.target_dir = ''
  importStatus.value = null
  importProgress.value.visible = false
}
</script>

<style scoped>
.media-import-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.media-import-card,
.usage-card {
  width: 100%;
  max-width: none;
  margin: 0;
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

.import-content {
  margin-top: 20px;
}

.import-section {
  margin-bottom: 24px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 12px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.section-description {
  margin: 0 0 16px 0;
  font-size: 14px;
  color: #606266;
  line-height: 1.5;
}

.import-form {
  margin-top: 16px;
}

.import-form .el-form-item {
  margin-bottom: 20px;
}

.form-help {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

.import-status {
  margin-top: 16px;
}

.progress-section {
  margin-top: 20px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
}

.progress-section h4 {
  margin: 0 0 12px 0;
  font-size: 16px;
  color: #303133;
}

.progress-text {
  margin: 8px 0 0 0;
  font-size: 14px;
  color: #606266;
  text-align: center;
}

.usage-content {
  margin-top: 20px;
}

.usage-section {
  margin-bottom: 20px;
}

.usage-section h4 {
  margin: 16px 0 8px 0;
  font-size: 16px;
  color: #303133;
}

.usage-section p {
  margin: 8px 0;
  color: #606266;
  line-height: 1.5;
}

.usage-section ul {
  margin: 8px 0;
  padding-left: 20px;
}

.usage-section li {
  margin: 4px 0;
  color: #606266;
  line-height: 1.4;
}

.usage-section li strong {
  color: #409eff;
  font-weight: 600;
}

.usage-section li code {
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .media-import-card,
  .usage-card {
    margin: 0;
  }

  .card-title {
    font-size: 20px;
  }

  .card-subtitle {
    font-size: 13px;
  }

  .section-title {
    font-size: 16px;
  }

  .import-form .el-button {
    width: 100%;
    margin-bottom: 8px;
  }
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .card-title {
    font-size: 18px;
  }

  .section-title {
    font-size: 15px;
  }

  .section-description {
    font-size: 13px;
  }
}
</style>
