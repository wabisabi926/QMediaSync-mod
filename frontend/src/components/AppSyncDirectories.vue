<template>
  <div class="sync-directories-container full-width-container">
    <el-card shadow="hover" class="full-width-card">
      <template #header>
        <div class="card-header">
          <div class="header-content">
            <div class="header-info">
              <h2 class="card-title">同步目录管理</h2>
              <p class="card-subtitle">管理本地目录与115网盘目录的同步配置</p>
              <div class="header-actions">
                <el-button type="primary" @click="showAddDialog = true">
                  <el-icon><Plus /></el-icon>
                  添加同步目录
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- 数据表格 -->
      <el-table
        :data="directories"
        v-loading="loading"
        stripe
        class="directories-table"
        empty-text="暂无同步目录"
        :show-header="!isMobile"
        size="default"
      >
        <el-table-column prop="base_cid" label="CID" min-width="120" :show-overflow-tooltip="true">
          <template #default="{ row }">
            <div class="table-cell-wrapper">
              <span class="cell-label" v-if="isMobile">CID:</span>
              <span class="cid-text">{{ row.base_cid }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column
          prop="local_path"
          label="本地目录"
          min-width="200"
          :show-overflow-tooltip="true"
        >
          <template #default="{ row }">
            <div class="table-cell-wrapper">
              <span class="cell-label" v-if="isMobile">本地目录:</span>
              <span class="path-text">{{ row.local_path }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column
          prop="remote_path"
          label="网盘目录"
          min-width="200"
          :show-overflow-tooltip="true"
        >
          <template #default="{ row }">
            <div class="table-cell-wrapper">
              <span class="cell-label" v-if="isMobile">网盘目录:</span>
              <span class="path-text">{{ row.remote_path }}</span>
              <!-- 移动端显示时间信息 -->
              <div v-if="isMobile" class="mobile-time-info">
                <div class="time-item">
                  <span class="time-label">添加:</span>
                  <span class="time-value">{{ formatTime(row.created_at) }}</span>
                </div>
                <div class="time-item">
                  <span class="time-label">修改:</span>
                  <span class="time-value">{{ formatTime(row.updated_at) }}</span>
                </div>
                <div class="time-item">
                  <span class="time-label">深度:</span>
                  <span class="time-value">{{ row.dir_depth || '-' }}</span>
                </div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="dir_depth" label="目录深度" width="100" align="center" v-if="!isMobile">
          <template #default="{ row }">
            <span>{{ row.dir_depth || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="created_at" label="添加时间" width="180" v-if="!isMobile">
          <template #default="{ row }">
            <span>{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="updated_at" label="修改时间" width="180" v-if="!isMobile">
          <template #default="{ row }">
            <span>{{ formatTime(row.updated_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" :width="isMobile ? 120 : 180" fixed="right">
          <template #default="{ row, $index }">
            <div class="action-buttons">
              <el-button
                type="primary"
                size="small"
                @click="handleEdit(row)"
                :loading="row.editing"
              >
                编辑
              </el-button>
              <el-button
                type="danger"
                size="small"
                @click="handleDelete(row, $index)"
                :loading="row.deleting"
              >
                删除
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container" v-if="total > 0">
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

    <!-- 添加同步目录对话框 -->
    <el-dialog
      v-model="showAddDialog"
      title="添加同步目录"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form ref="addFormRef" :model="addForm" :rules="addFormRules" label-width="100px">
        <el-form-item label="本地目录" prop="local_path">
          <el-input v-model="addForm.local_path" placeholder="请输入本地目录路径" clearable />
          <div class="form-tip">本地同步目录的绝对路径</div>
        </el-form-item>

        <el-form-item label="115网盘目录" prop="base_cid">
          <div class="pan-dir-input">
            <el-input
              v-model="addForm.base_cid"
              placeholder="点击选择按钮选择115网盘目录"
              :disabled="addLoading"
              readonly
            />
            <el-button type="primary" @click="openDirSelector" :disabled="addLoading">
              选择目录
            </el-button>
          </div>
          <div v-if="selectedDirPath" class="selected-path-inline">
            <span class="path-label">选中目录路径：</span>
            <code class="path-url">{{ selectedDirPath }}</code>
          </div>
          <div class="form-tip">选择115网盘中要同步的目录</div>
        </el-form-item>

        <el-form-item label="STRM存放目录">
          <el-input
            v-model="addForm.strm_path"
            placeholder="自动计算：本地目录 + 选中目录路径"
            :disabled="true"
            readonly
          />
          <div class="form-tip">STRM和元数据实际存放目录（自动生成）</div>
        </el-form-item>

        <el-form-item label="目录深度" prop="dir_depth">
          <el-input-number
            v-model="addForm.dir_depth"
            :min="1"
            :max="10"
            :step="1"
            :disabled="addLoading"
            placeholder="初始化的目录深度"
            style="width: 100%"
          />
          <p>该设置会严重影响首次同步所需时间，建议认真设置；如果不是很清楚，建议设置成2</p>
          <div class="form-tip">
            <p>首次获取目录的深度，建议设置为 1-3 层（默认 2层）</p>
            <p>
              如果所选网盘目录是AV类型的根目录，下面的目录结构如：小姐姐名/番号/番号.mkv，那就输入2。
            </p>
            <p>
              如果所选网盘目录是影视剧的根目录，下面的目录结构如：电影/动画电影/哪吒/哪吒.mkv，那就输入3；
            </p>
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showAddDialog = false">取消</el-button>
          <el-button type="primary" @click="handleAdd" :loading="addLoading"> 确定 </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑同步目录对话框 -->
    <el-dialog
      v-model="showEditDialog"
      title="编辑同步目录"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form ref="editFormRef" :model="editForm" :rules="editFormRules" label-width="100px">
        <el-form-item label="本地目录" prop="local_path">
          <el-input v-model="editForm.local_path" placeholder="请输入本地目录路径" clearable />
          <div class="form-tip">本地同步目录的绝对路径</div>
        </el-form-item>

        <el-form-item label="115网盘目录" prop="base_cid">
          <div class="pan-dir-input">
            <el-input
              v-model="editForm.base_cid"
              placeholder="点击选择按钮选择115网盘目录"
              :disabled="editLoading"
              readonly
            />
            <el-button type="primary" @click="openEditDirSelector" :disabled="editLoading">
              选择目录
            </el-button>
          </div>
          <div v-if="editSelectedDirPath" class="selected-path-inline">
            <span class="path-label">选中目录路径：</span>
            <code class="path-url">{{ editSelectedDirPath }}</code>
          </div>
          <div class="form-tip">选择115网盘中要同步的目录</div>
        </el-form-item>

        <el-form-item label="STRM存放目录">
          <el-input
            v-model="editForm.strm_path"
            placeholder="自动计算：本地目录 + 选中目录路径"
            :disabled="true"
            readonly
          />
          <div class="form-tip">STRM和元数据实际存放目录（自动生成）</div>
        </el-form-item>

        <el-form-item label="目录深度" prop="dir_depth">
          <el-input-number
            v-model="editForm.dir_depth"
            :min="1"
            :max="10"
            :step="1"
            :disabled="editLoading"
            placeholder="目录深度"
            style="width: 100%"
          />
          <div class="form-tip">目录深度，建议设置为 1-3 层（默认 2层）</div>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showEditDialog = false">取消</el-button>
          <el-button type="primary" @click="handleEditSave" :loading="editLoading">
            确定
          </el-button>
        </div>
      </template>
    </el-dialog>

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
import { inject, onMounted, onUnmounted, ref, reactive, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Loading, Folder, ArrowRight } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'

interface SyncDirectory {
  id?: number
  base_cid: string
  local_path: string
  remote_path: string
  strm_path?: string
  dir_depth?: number
  created_at: string
  updated_at: string
  deleting?: boolean
  editing?: boolean
}

interface DirInfo {
  id: string
  name: string
  path?: string
}

const http: AxiosStatic | undefined = inject('$http')

// 数据状态
const directories = ref<SyncDirectory[]>([])
const loading = ref(false)
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)

// 移动端检测
const isMobile = ref(false)

// 目录选择相关状态
const showDirDialog = ref(false)
const dirTreeData = ref<DirInfo[]>([])
const dirTreeLoading = ref(false)
const selectedDirPath = ref('')
const currentDir = ref<DirInfo | null>(null)
const tempSelectedDir = ref<DirInfo | null>(null)
const isEditMode = ref(false) // 标记是否为编辑模式

// 检测是否为移动设备
const checkMobile = () => {
  isMobile.value = window.innerWidth <= 768
}

// 添加对话框状态
const showAddDialog = ref(false)
const addLoading = ref(false)
const addFormRef = ref<FormInstance>()
const addForm = reactive({
  local_path: '',
  base_cid: '',
  strm_path: '',
  dir_depth: 2,
})

// 编辑对话框状态
const showEditDialog = ref(false)
const editLoading = ref(false)
const editFormRef = ref<FormInstance>()
const editForm = reactive({
  id: 0,
  local_path: '',
  base_cid: '',
  strm_path: '',
  dir_depth: 2,
})
const editSelectedDirPath = ref('')

// 表单验证规则
const addFormRules: FormRules = {
  local_path: [
    { required: true, message: '请输入本地目录路径', trigger: 'blur' },
    { min: 1, max: 500, message: '长度在 1 到 500 个字符', trigger: 'blur' },
  ],
  base_cid: [
    { required: true, message: '请输入CID', trigger: 'blur' },
    { min: 1, max: 100, message: '长度在 1 到 100 个字符', trigger: 'blur' },
  ],
  dir_depth: [
    { required: true, message: '请输入目录深度', trigger: 'blur' },
    { type: 'number', min: 1, max: 10, message: '目录深度必须在 1 到 10 之间', trigger: 'blur' },
  ],
}

// 编辑表单验证规则
const editFormRules: FormRules = {
  local_path: [
    { required: true, message: '请输入本地目录路径', trigger: 'blur' },
    { min: 1, max: 500, message: '长度在 1 到 500 个字符', trigger: 'blur' },
  ],
  base_cid: [
    { required: true, message: '请输入CID', trigger: 'blur' },
    { min: 1, max: 100, message: '长度在 1 到 100 个字符', trigger: 'blur' },
  ],
  dir_depth: [
    { required: true, message: '请输入目录深度', trigger: 'blur' },
    { type: 'number', min: 1, max: 10, message: '目录深度必须在 1 到 10 之间', trigger: 'blur' },
  ],
}

// 格式化时间
const formatTime = (timestamp: string | number): string => {
  if (!timestamp) return '-'

  try {
    // 将秒级时间戳转换为毫秒级时间戳
    const timestampMs =
      typeof timestamp === 'string' ? parseInt(timestamp) * 1000 : timestamp * 1000
    const date = new Date(timestampMs)

    // 检查日期是否有效
    if (isNaN(date.getTime())) {
      return '-'
    }

    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  } catch {
    return '-'
  }
}

// 加载同步目录列表
const loadDirectories = async () => {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/path-list`, {
      params: {
        page: currentPage.value,
        size: pageSize.value,
      },
    })

    if (response?.data.code === 200) {
      directories.value = response.data.data.list || []
      total.value = response.data.data.total || 0
    } else {
      ElMessage.error(response?.data.msg || '加载同步目录失败')
      directories.value = []
      total.value = 0
    }
  } catch {
    console.error('加载同步目录错误')
    ElMessage.error('加载同步目录失败')
    directories.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

// 处理页面大小变化
const handleSizeChange = (newSize: number) => {
  pageSize.value = newSize
  currentPage.value = 1
  loadDirectories()
}

// 处理页码变化
const handleCurrentChange = (newPage: number) => {
  currentPage.value = newPage
  loadDirectories()
}

// 处理添加同步目录
const handleAdd = async () => {
  if (!addFormRef.value) return

  try {
    await addFormRef.value.validate()
    addLoading.value = true

    const formData = new FormData()
    formData.append('local_path', addForm.local_path.trim())
    formData.append('base_cid', addForm.base_cid.trim())
    formData.append('strm_path', addForm.strm_path.trim())
    formData.append('dir_depth', addForm.dir_depth.toString())

    const response = await http?.post(`${SERVER_URL}/sync/path-add`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success('添加同步目录成功')
      showAddDialog.value = false
      addForm.local_path = ''
      addForm.base_cid = ''
      addForm.strm_path = ''
      addForm.dir_depth = 2
      selectedDirPath.value = ''
      loadDirectories()
    } else {
      ElMessage.error(response?.data.msg || '添加同步目录失败')
    }
  } catch {
    console.error('添加同步目录错误')
    ElMessage.error('添加同步目录失败')
  } finally {
    addLoading.value = false
  }
}

// 处理编辑同步目录
const handleEdit = async (row: SyncDirectory) => {
  editForm.id = row.id || 0
  editForm.local_path = row.local_path
  editForm.base_cid = row.base_cid
  editForm.dir_depth = row.dir_depth || 2
  editSelectedDirPath.value = row.remote_path || ''

  // 初始化STRM路径
  updateEditStrmPath()

  showEditDialog.value = true
}

// 处理编辑保存
const handleEditSave = async () => {
  if (!editFormRef.value) return

  try {
    await editFormRef.value.validate()
    editLoading.value = true

    const formData = new FormData()
    formData.append('id', editForm.id.toString())
    formData.append('local_path', editForm.local_path.trim())
    formData.append('base_cid', editForm.base_cid.trim())
    formData.append('strm_path', editForm.strm_path.trim())
    formData.append('dir_depth', editForm.dir_depth.toString())

    const response = await http?.post(`${SERVER_URL}/sync/path-update`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success('编辑同步目录成功')
      showEditDialog.value = false
      editForm.id = 0
      editForm.local_path = ''
      editForm.base_cid = ''
      editForm.strm_path = ''
      editForm.dir_depth = 2
      editSelectedDirPath.value = ''
      loadDirectories()
    } else {
      ElMessage.error(response?.data.msg || '编辑同步目录失败')
    }
  } catch {
    console.error('编辑同步目录错误')
    ElMessage.error('编辑同步目录失败')
  } finally {
    editLoading.value = false
  }
}

// 处理删除同步目录
const handleDelete = async (row: SyncDirectory, index: number) => {
  try {
    await ElMessageBox.confirm(`确定要删除同步目录 "${row.local_path}" 吗？`, '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    directories.value[index].deleting = true

    const formData = new FormData()
    formData.append('id', row.id?.toString() || '')

    const response = await http?.post(`${SERVER_URL}/sync/path-delete`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success('删除同步目录成功')
      loadDirectories()
    } else {
      ElMessage.error(response?.data.message || '删除同步目录失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除同步目录错误')
      ElMessage.error('删除同步目录失败')
    }
  } finally {
    if (directories.value[index]) {
      directories.value[index].deleting = false
    }
  }
}

// 打开目录选择器
const openDirSelector = async () => {
  isEditMode.value = false
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

// 计算STRM存放目录
const calculateStrmPath = (localPath: string, dirPath: string): string => {
  if (!localPath || !dirPath) return ''

  // 移除本地路径末尾的斜杠
  const cleanLocalPath = localPath.replace(/[/\\]+$/, '')
  // 移除目录路径开头的斜杠并规范化路径分隔符
  const cleanDirPath = dirPath.replace(/^[/\\]+/, '').replace(/\//g, '\\')

  return cleanDirPath ? `${cleanLocalPath}\\${cleanDirPath}` : cleanLocalPath
}

// 更新添加表单的STRM路径
const updateAddStrmPath = () => {
  addForm.strm_path = calculateStrmPath(addForm.local_path, selectedDirPath.value)
}

// 更新编辑表单的STRM路径
const updateEditStrmPath = () => {
  editForm.strm_path = calculateStrmPath(editForm.local_path, editSelectedDirPath.value)
}

// 确认选择目录
const confirmSelectDir = async () => {
  if (!tempSelectedDir.value) return

  const selectedDir = tempSelectedDir.value

  if (isEditMode.value) {
    // 编辑模式：设置编辑表单的CID值和显示路径
    editForm.base_cid = selectedDir.id
    editSelectedDirPath.value = selectedDir.path ? selectedDir.path : selectedDir.name
    // 更新编辑表单的STRM路径
    updateEditStrmPath()
  } else {
    // 添加模式：设置添加表单的CID值和显示路径
    addForm.base_cid = selectedDir.id
    selectedDirPath.value = selectedDir.path ? selectedDir.path : selectedDir.name
    // 更新添加表单的STRM路径
    updateAddStrmPath()
  }

  showDirDialog.value = false
  tempSelectedDir.value = null
  currentDir.value = null
}

// 编辑时打开目录选择器
const openEditDirSelector = async () => {
  isEditMode.value = true
  showDirDialog.value = true
  tempSelectedDir.value = null
  currentDir.value = null
  await loadDirTree('0') // 加载根目录
}

// 监听添加表单本地路径变化
watch(
  () => addForm.local_path,
  () => {
    updateAddStrmPath()
  },
)

// 监听编辑表单本地路径变化
watch(
  () => editForm.local_path,
  () => {
    updateEditStrmPath()
  },
)

// 组件挂载时加载数据
onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
  loadDirectories()
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<style scoped>
.sync-directories-container {
  width: 100% !important;
  max-width: 100% !important;
  margin: 0;
  padding: 0;
}

/* 全宽度容器，突破父容器的padding限制 */
.full-width-container {
  margin: -20px !important;
  padding: 20px !important;
  width: calc(100% + 40px) !important;
  max-width: calc(100% + 40px) !important;
}

/* 确保卡片组件也占满宽度 */
.full-width-card,
.sync-directories-container :deep(.el-card) {
  width: 100% !important;
  max-width: 100% !important;
  border-radius: 0 !important;
}

.card-header {
  margin: 0;
  padding: 0;
}

.header-content {
  display: flex;
  align-items: flex-start;
}

.header-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.header-actions {
  margin-top: 16px;
  display: flex;
  align-items: center;
}

.card-title {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.card-subtitle {
  margin: 0;
  font-size: 14px;
  color: #909399;
}

.directories-table {
  width: 100% !important;
  margin-bottom: 20px;
  overflow-x: auto;
}

/* 确保表格容器也占满宽度 */
.directories-table :deep(.el-table) {
  width: 100% !important;
}

.directories-table :deep(.el-table__inner-wrapper) {
  width: 100% !important;
}

.table-cell-wrapper {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.cell-label {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
}

.cid-text {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  color: #606266;
  word-break: break-all;
}

.path-text {
  color: #303133;
  word-break: break-all;
  font-size: 13px;
  line-height: 1.4;
}

.mobile-time-info {
  display: flex;
  gap: 12px;
  margin-top: 4px;
  font-size: 12px;
}

.time-item {
  display: flex;
  gap: 2px;
}

.time-label {
  color: #909399;
  font-weight: 500;
}

.time-value {
  color: #606266;
}

.pagination-container {
  display: flex;
  justify-content: center;
  padding: 20px 0;
  overflow-x: auto;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

/* 操作按钮组样式 */
.action-buttons {
  display: flex;
  gap: 8px;
}

/* 115网盘目录选择相关样式 */
.pan-dir-input {
  display: flex;
  gap: 8px;
  align-items: flex-start;
}

.pan-dir-input .el-input {
  flex: 1;
}

.selected-path-inline {
  margin-top: 8px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
}

.path-label {
  color: #909399;
  font-weight: 500;
}

.path-url {
  color: #606266;
  background: #fff;
  padding: 2px 6px;
  border-radius: 2px;
  border: 1px solid #dcdfe6;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

/* 目录选择对话框样式 */
.dir-selector {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.loading-container,
.empty-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #909399;
}

.loading-container .el-icon {
  font-size: 32px;
  margin-bottom: 8px;
}

.dir-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.dir-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
}

.dir-item:hover {
  background: #f5f7fa;
  border-color: #e4e7ed;
}

.dir-item .el-icon:first-child {
  color: #409eff;
  font-size: 16px;
  margin-right: 8px;
}

.dir-name {
  flex: 1;
  font-size: 14px;
  color: #303133;
  word-break: break-all;
}

.enter-icon {
  color: #c0c4cc;
  font-size: 14px;
  margin-left: 8px;
}

.selected-dir-section {
  padding: 16px;
  background: #f5f7fa;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
}

.selected-dir-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.selected-dir-label {
  font-size: 14px;
  font-weight: 500;
  color: #606266;
}

.selected-dir-path {
  font-size: 13px;
  color: #303133;
  padding: 8px 12px;
  background: #fff;
  border-radius: 4px;
  border: 1px solid #dcdfe6;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  word-break: break-all;
  line-height: 1.4;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .full-width-container {
    margin: -20px !important;
    padding: 15px !important;
    width: calc(100% + 40px) !important;
  }

  .sync-directories-container {
    padding: 0;
  }

  .card-header {
    gap: 8px;
  }

  .header-content {
    align-items: stretch;
  }

  .header-info {
    gap: 6px;
  }

  .header-actions {
    margin-top: 12px;
    align-self: flex-start;
  }

  .card-title {
    font-size: 20px;
  }

  .card-subtitle {
    font-size: 13px;
  }

  .directories-table {
    font-size: 14px;
    margin-bottom: 16px;
  }

  .table-cell-wrapper {
    gap: 4px;
  }

  .cell-label {
    font-size: 13px;
    font-weight: 600;
  }

  .cid-text {
    font-size: 13px;
  }

  .path-text {
    font-size: 13px;
    line-height: 1.4;
  }

  .pagination-container {
    padding: 16px 0;
  }

  /* 移动端操作按钮适配 */
  .action-buttons {
    gap: 4px;
    flex-direction: column;
  }

  .action-buttons .el-button {
    padding: 2px 6px !important;
    font-size: 11px !important;
    height: 24px !important;
    min-height: 24px !important;
    min-width: 40px !important;
  }

  /* 对话框在移动端的适配 */
  :deep(.el-dialog) {
    width: 95% !important;
    margin: 0 auto;
  }

  :deep(.el-dialog__body) {
    padding: 15px 20px;
  }
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .full-width-container {
    margin: -20px !important;
    padding: 10px !important;
    width: calc(100% + 40px) !important;
  }

  .card-title {
    font-size: 18px;
  }

  .card-subtitle {
    font-size: 12px;
  }

  .header-content {
    gap: 8px;
  }

  .card-header {
    gap: 6px;
  }

  .directories-table {
    font-size: 13px;
  }

  .cell-label {
    font-size: 12px;
  }

  .cid-text,
  .path-text {
    font-size: 12px;
  }

  .mobile-time-info {
    font-size: 11px;
  }

  /* 进一步优化操作按钮 */
  .action-buttons {
    gap: 3px;
    flex-direction: column;
  }

  .action-buttons .el-button {
    padding: 1px 4px !important;
    font-size: 10px !important;
    height: 22px !important;
    min-height: 22px !important;
    min-width: 36px !important;
  }

  .pagination-container {
    padding: 12px 0;
  }

  /* 分页组件在小屏下的适配 */
  :deep(.el-pagination) {
    font-size: 12px;
  }

  :deep(.el-pagination .btn-prev),
  :deep(.el-pagination .btn-next),
  :deep(.el-pagination .el-pager li) {
    min-width: 28px;
    height: 28px;
    line-height: 28px;
  }
}

/* 表格行的移动端适配 */
@media (max-width: 768px) {
  :deep(.el-table .cell) {
    padding: 8px 4px;
    line-height: 1.3;
  }

  :deep(.el-table th) {
    padding: 8px 0;
  }

  :deep(.el-table td) {
    padding: 8px 0;
  }
}

/* 极小屏设备优化 */
@media (max-width: 360px) {
  .action-buttons {
    gap: 2px;
    flex-direction: column;
  }

  .action-buttons .el-button {
    padding: 1px 3px !important;
    font-size: 9px !important;
    height: 20px !important;
    min-height: 20px !important;
    min-width: 32px !important;
  }

  /* 进一步减少操作列宽度 */
  :deep(.el-table__fixed-right) {
    width: 100px !important;
  }

  :deep(.el-table__fixed-right .el-table__cell) {
    width: 100px !important;
  }
}
</style>
