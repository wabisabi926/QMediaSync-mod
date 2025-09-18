<template>
  <div class="instant-upload-container">
    <!-- 秒传功能卡片 -->
    <el-card class="instant-upload-card" shadow="hover">
      <template #header>
        <h2 class="card-title">秒传功能</h2>
        <p class="card-subtitle">通过文件哈希值快速上传文件到115网盘</p>
      </template>

      <div class="upload-content">
        <!-- 批量文件秒传 -->
        <div class="upload-section">
          <h3 class="section-title">
            <el-icon><FolderOpened /></el-icon>
            批量秒传
          </h3>
          <p class="section-description">通过文本格式批量上传多个文件</p>

          <el-form :model="batchUploadData" :label-position="'top'" class="upload-form">
            <!-- 根目录选择 -->
            <el-form-item label="115网盘根目录">
              <div class="dir-selector-inline">
                <el-input
                  v-model="selectedDirId"
                  placeholder="目录ID"
                  :disabled="uploading"
                  style="width: 200px; margin-right: 12px"
                />
                <el-button type="primary" @click="openDirSelector" :disabled="uploading">
                  选择目录
                </el-button>
              </div>
              <div v-if="selectedDirPath" class="selected-path-inline">
                <span class="path-label">选中目录路径：</span>
                <code class="path-url">{{ selectedDirPath }}</code>
              </div>
              <div class="form-help">选择115网盘中的根目录，文件将上传到此目录下</div>
            </el-form-item>

            <el-form-item label="文件列表" required>
              <el-input
                v-model="batchUploadData.file_list"
                type="textarea"
                :rows="10"
                placeholder="请按照指定URL格式输入文件信息，每行一个URL"
                :disabled="uploading"
              />
              <div class="form-help">
                格式：http://ip:port/115/url?name=$path&pick_code=$pick_code&sha1=$sha1&size=$size<br />
                示例：<br />
                http://192.168.1.10:12333/115/url?name=%E7%94%B5%E8%A7%86%E5%89%A7%2F%E5%9B%BD%E4%BA%A7%E5%89%A7%2F%E9%97%AE%E5%BF%83%2F%E9%97%AE%E5%BF%83+01.mp4&pick_code=azg8910t6xesohfw6&sha1=22938B78CD5E8EB4D82A3181CD1ED210B2C06F25&size=5249192463<br />
                注意：文件路径需要进行URL编码
              </div>
            </el-form-item>

            <el-form-item>
              <el-button type="success" @click="uploadBatchFiles" :loading="uploading" size="large">
                <el-icon><Upload /></el-icon>
                批量秒传
              </el-button>

              <el-button type="warning" @click="resetBatchForm" :disabled="uploading" size="large">
                <el-icon><RefreshLeft /></el-icon>
                重置表单
              </el-button>
            </el-form-item>
          </el-form>
        </div>

        <!-- 操作状态显示 -->
        <el-alert
          v-if="uploadStatus"
          :title="uploadStatus.title"
          :type="uploadStatus.type"
          :description="uploadStatus.description"
          :closable="false"
          show-icon
          class="upload-status"
        />

        <!-- 上传进度 -->
        <div v-if="uploadProgress.visible" class="progress-section">
          <h4>上传进度</h4>
          <el-progress
            :percentage="uploadProgress.percentage"
            :status="uploadProgress.status"
            :show-text="true"
          />
          <p class="progress-text">{{ uploadProgress.text }}</p>
        </div>
      </div>
    </el-card>

    <!-- 使用说明卡片 -->
    <el-card class="usage-card" shadow="hover">
      <template #header>
        <h2 class="card-title">使用说明</h2>
        <p class="card-subtitle">如何获取URL</p>
      </template>

      <div class="usage-content">
        <div class="usage-section notice-section">
          <h4>注意事项：</h4>
          <p>
            文件将被上传到115网盘根目录+name，比如115网盘根目录选择了Media，name=电影/国产电影/哪吒(2015)/哪吒.mkv，那么该文件将被上传到/Media/电影/国产电影/哪吒(2015)/哪吒.mkv，如果路径不存在则会自动创建。
          </p>
        </div>

        <div class="usage-section">
          <h4>URL格式说明：</h4>
          <p>每行包含一个完整的115文件URL，格式如下：</p>
          <el-code class="code-block">
            http://ip:port/115/url?name=$path&pick_code=$pick_code&sha1=$sha1&size=$size
          </el-code>
          <ul>
            <li><strong>ip:port：</strong>服务器地址和端口</li>
            <li><strong>name：</strong>文件路径</li>
            <li><strong>pick_code：</strong>115网盘文件提取码</li>
            <li><strong>sha1：</strong>文件SHA1哈希值（40位十六进制）</li>
            <li><strong>size：</strong>文件大小（字节）</li>
          </ul>
        </div>

        <div class="usage-section">
          <h4>如何获取URL：</h4>
          <p>请使用本项目生成的STRM文件的内容</p>
        </div>

        <div class="usage-section">
          <h4>接口数据格式：</h4>
          <p>提交时会发送以下参数：</p>
          <ul>
            <li><strong>file_list：</strong>JSON格式的字符串数组，包含所有URL</li>
            <li><strong>root_dir_id：</strong>115网盘根目录ID</li>
            <li><strong>root_dir_path：</strong>115网盘根目录路径</li>
          </ul>
          <p>示例：</p>
          <el-code class="code-block">
            file_list: ["http://ip:port/115/url?name=file1...",
            "http://ip:port/115/url?name=file2..."]<br />
            root_dir_id: "123456789"<br />
            root_dir_path: "/Media"
          </el-code>
        </div>
      </div>
    </el-card>

    <!-- 115网盘目录选择对话框 -->
    <el-dialog
      v-model="showDirDialog"
      title="选择115网盘目录"
      width="60%"
      :close-on-click-modal="false"
    >
      <div class="dir-selector">
        <el-scrollbar height="400px">
          <div v-if="dirTreeLoading" class="loading-container">
            <el-icon class="is-loading"><Loading /></el-icon>
            <p>加载中...</p>
          </div>
          <div v-else-if="dirTreeData.length === 0" class="empty-container">
            <p>暂无目录</p>
          </div>
          <div v-else class="dir-list">
            <div
              v-for="dir in dirTreeData"
              :key="dir.id"
              class="dir-item"
              @click="selectTempDir(dir)"
            >
              <el-icon><Folder /></el-icon>
              <span class="dir-name">{{ dir.name }}</span>
              <el-icon class="enter-icon"><ArrowRight /></el-icon>
            </div>
          </div>
        </el-scrollbar>

        <!-- 选中目录显示和确认区域 -->
        <div v-if="tempSelectedDir" class="selected-dir-section">
          <div class="selected-dir-info">
            <div class="selected-dir-label">当前选中目录：</div>
            <div class="selected-dir-path">{{ tempSelectedDir.path || tempSelectedDir.name }}</div>
          </div>
        </div>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showDirDialog = false">取消</el-button>
          <el-button type="primary" @click="confirmSelectDir" :disabled="!tempSelectedDir">
            确定选择
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import {
  Upload,
  FolderOpened,
  RefreshLeft,
  Folder,
  ArrowRight,
  Loading,
} from '@element-plus/icons-vue'
import { inject, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

interface BatchUploadData {
  file_list: string
}

interface DirInfo {
  id: string
  name: string
  path?: string
}

interface UploadStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}

interface UploadProgress {
  visible: boolean
  percentage: number
  status: 'success' | 'exception' | 'warning' | undefined
  text: string
}

const http: AxiosStatic | undefined = inject('$http')

// 状态管理
const uploading = ref(false)
const uploadStatus = ref<UploadStatus | null>(null)
const uploadProgress = ref<UploadProgress>({
  visible: false,
  percentage: 0,
  status: undefined,
  text: '',
})

// 批量上传数据
const batchUploadData = reactive<BatchUploadData>({
  file_list: '',
})

// 目录选择相关状态
const showDirDialog = ref(false)
const dirTreeData = ref<DirInfo[]>([])
const dirTreeLoading = ref(false)
const tempSelectedDir = ref<DirInfo | null>(null)
const currentDir = ref<DirInfo | null>(null)
const selectedDirPath = ref('')
const selectedDirId = ref('')

// 单文件秒传
// 批量文件秒传
const uploadBatchFiles = async () => {
  if (!validateBatchFiles()) {
    return
  }

  try {
    uploading.value = true
    uploadStatus.value = null
    uploadProgress.value.visible = true
    uploadProgress.value.percentage = 0
    uploadProgress.value.text = '准备批量上传...'

    // 将每行URL转换为字符串数组，然后以JSON格式发送
    const lines = batchUploadData.file_list.trim().split('\n')
    const urlArray = lines.map((line) => line.trim()).filter((line) => line.length > 0) // 过滤空行

    const requestData = {
      file_list: urlArray,
      root_dir_id: selectedDirId.value,
      root_dir_path: selectedDirPath.value,
    }

    uploadProgress.value.percentage = 50
    uploadProgress.value.text = '正在处理批量文件...'

    const response = await http?.post(`${SERVER_URL}/upload/batch-instant`, requestData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    uploadProgress.value.percentage = 100

    if (response?.data.code === 200) {
      uploadProgress.value.status = 'success'
      uploadProgress.value.text = '批量秒传完成！'

      const successCount = response.data.data?.success_count || 0
      const totalCount = response.data.data?.total_count || 0

      uploadStatus.value = {
        title: '批量秒传完成',
        type: successCount === totalCount ? 'success' : 'warning',
        description: `共处理 ${totalCount} 个文件，成功 ${successCount} 个，失败 ${totalCount - successCount} 个`,
      }
      ElMessage.success(`批量秒传完成！成功 ${successCount}/${totalCount} 个文件`)

      // 上传完成后清空文件列表
      batchUploadData.file_list = ''
    } else {
      uploadProgress.value.status = 'exception'
      uploadProgress.value.text = '批量秒传失败'
      uploadStatus.value = {
        title: '批量秒传失败',
        type: 'error',
        description: response?.data.message || '批量秒传过程中发生错误，请检查文件列表格式',
      }
    }
  } catch (error) {
    console.error('批量秒传错误:', error)
    uploadProgress.value.status = 'exception'
    uploadProgress.value.text = '批量秒传失败'
    uploadStatus.value = {
      title: '批量秒传出错',
      type: 'error',
      description: '批量秒传过程中发生网络错误，请检查网络连接',
    }
  } finally {
    uploading.value = false
    setTimeout(() => {
      uploadProgress.value.visible = false
    }, 3000)
  }
}

// 验证批量文件数据
const validateBatchFiles = (): boolean => {
  // 验证根目录是否已选择
  if (!selectedDirId.value || !selectedDirPath.value) {
    ElMessage.warning('请选择115网盘根目录')
    return false
  }

  if (!batchUploadData.file_list.trim()) {
    ElMessage.warning('请输入文件列表')
    return false
  }

  const lines = batchUploadData.file_list.trim().split('\n')
  const invalidLines: number[] = []

  lines.forEach((line, index) => {
    const url = line.trim()
    if (!url) {
      return // 跳过空行
    }

    // 验证URL格式
    try {
      const urlObj = new URL(url)

      // 检查是否为115格式的URL
      if (!urlObj.pathname.includes('/115/url')) {
        invalidLines.push(index + 1)
        return
      }

      // 检查必需的查询参数
      const searchParams = urlObj.searchParams
      const name = searchParams.get('name')
      const pickCode = searchParams.get('pick_code')
      const sha1 = searchParams.get('sha1')
      const size = searchParams.get('size')

      if (!name || !pickCode || !sha1 || !size) {
        invalidLines.push(index + 1)
        return
      }

      // 验证sha1格式
      if (!/^[a-fA-F0-9]{40}$/i.test(sha1)) {
        invalidLines.push(index + 1)
        return
      }

      // 验证size格式
      if (!/^\d+$/.test(size)) {
        invalidLines.push(index + 1)
        return
      }
    } catch {
      invalidLines.push(index + 1)
      return
    }
  })

  if (invalidLines.length > 0) {
    ElMessage.warning(`第 ${invalidLines.join(', ')} 行URL格式不正确`)
    return false
  }

  return true
}

// 重置批量表单
const resetBatchForm = () => {
  batchUploadData.file_list = ''
  uploadStatus.value = null
  uploadProgress.value.visible = false
}

// 打开目录选择器
const openDirSelector = async () => {
  showDirDialog.value = true
  tempSelectedDir.value = null
  currentDir.value = null
  await loadDirTree('0') // 加载根目录
}

// 加载目录树
const loadDirTree = async (dirId: string) => {
  try {
    dirTreeLoading.value = true
    const response = await http?.get(`${SERVER_URL}/115/dir-path`, {
      params: { parent_id: dirId },
    })

    if (response?.data.code === 200) {
      dirTreeData.value = response.data.data || []
    }
  } catch (error) {
    console.error('加载目录树错误:', error)
  } finally {
    dirTreeLoading.value = false
  }
}

// 临时选择目录（点击目录时）
const selectTempDir = async (dir: DirInfo) => {
  tempSelectedDir.value = dir
  currentDir.value = dir
  // 加载该目录的子目录
  await loadDirTree(dir.id)
}

// 确认选择目录
const confirmSelectDir = async () => {
  if (!tempSelectedDir.value) return

  const selectedDir = tempSelectedDir.value
  // 设置目录ID和路径
  selectedDirId.value = selectedDir.id
  selectedDirPath.value = selectedDir.path || selectedDir.name
  showDirDialog.value = false
}
</script>

<style scoped>
.instant-upload-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.instant-upload-card,
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

.upload-content,
.usage-content {
  margin-top: 20px;
}

.upload-section,
.usage-section {
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

.upload-form {
  margin-top: 16px;
}

.upload-form .el-form-item {
  margin-bottom: 20px;
}

.form-help {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

.upload-status {
  margin-top: 20px;
}

.progress-section {
  margin-top: 20px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
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

.code-block {
  display: block;
  background: #f5f7fa;
  padding: 8px 12px;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  margin: 8px 0;
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

.hash-commands {
  margin-top: 12px;
}

.hash-commands p {
  margin: 8px 0 4px 0;
  font-weight: 600;
}

.usage-section li strong {
  color: #409eff;
  font-weight: 600;
}

/* 注意事项样式 */
.notice-section {
  background: #fff7e6;
  border: 1px solid #ffc069;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 20px;
}

.notice-section h4 {
  color: #fa8c16;
  margin-bottom: 8px;
  font-weight: 600;
}

.notice-section p {
  color: #8c4a00;
  margin: 0;
  line-height: 1.6;
}

/* 目录选择相关样式 */
.dir-selector-inline {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.selected-path-inline {
  margin-top: 8px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
  border: 1px solid #e4e7ed;
}

.path-label {
  font-size: 12px;
  color: #909399;
  margin-right: 8px;
}

.path-url {
  background: #fff;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #409eff;
}

.dir-selector {
  padding: 12px 0;
}

.loading-container,
.empty-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: #909399;
}

.loading-container .el-icon {
  font-size: 24px;
  margin-bottom: 8px;
}

.dir-list {
  padding: 0;
}

.dir-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  margin-bottom: 4px;
  background: #f8f9fa;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.dir-item:hover {
  background: #ecf5ff;
  border-color: #b3d8ff;
}

.dir-item .el-icon {
  margin-right: 8px;
  color: #409eff;
}

.dir-name {
  flex: 1;
  font-size: 14px;
  color: #303133;
}

.enter-icon {
  color: #c0c4cc;
}

.selected-dir-section {
  margin-top: 16px;
  padding: 12px;
  background: #f0f9ff;
  border: 1px solid #b3d8ff;
  border-radius: 6px;
}

.selected-dir-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.selected-dir-label {
  font-size: 12px;
  color: #909399;
  font-weight: 600;
}

.selected-dir-path {
  font-size: 14px;
  color: #409eff;
  font-family: 'Courier New', monospace;
  background: #fff;
  padding: 4px 8px;
  border-radius: 4px;
  border: 1px solid #d9ecff;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .instant-upload-card,
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

  .upload-form .el-button {
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
