<template>
  <div class="main-content-container file-manager-container full-width-container">
    <el-card shadow="none" class="full-width-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <h2 class="card-title hidden-md-and-down">网盘文件浏览器（实装功能：查看列表、创建文件夹、删除）</h2>
            <p class="card-subtitle">
              浏览和管理媒体文件，支持STRM生成、刮削整理和ED2K生成操作
            </p>
          </div>
          <div class="header-right">
            <el-select v-model="pageSize" style="width: 100px; margin-right: 10px" @change="handlePageSizeChange">
              <el-option label="100" :value="100" />
              <el-option label="200" :value="200" />
              <el-option label="500" :value="500" />
            </el-select>
          </div>
        </div>
      </template>

      <!-- 左右布局 -->
      <div class="file-manager-layout">
        <!-- 左侧：网盘账号列表 -->
        <div class="account-sidebar">
          <div class="sidebar-header">
            <h3>网盘账号</h3>
          </div>
          <div class="account-list">
            <div v-for="account in accountList" :key="account.id"
              :class="['account-item', { active: selectedAccountId === account.id }]" @click="selectAccount(account)">
              <div class="account-info">
                <el-icon class="account-icon">
                  <component :is="getAccountIcon()" />
                </el-icon>
                <div class="account-details">
                  <div class="account-name">{{ account.username }} <span v-if="account.source_type === '115'">({{ account.user_id }})</span></div>
                  <div class="account-type">{{ getAccountTypeName(account.source_type) }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 右侧：文件列表 -->
        <div class="file-content">
          <!-- 未选择账号时的提示 -->
          <div v-if="!selectedAccountId" class="no-account-selected">
            <el-empty description="选择一个网盘账号" />
          </div>

          <!-- 文件列表内容 -->
          <template v-else>
            <!-- 面包屑导航 -->
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px">
              <el-breadcrumb separator="/">
                <el-breadcrumb-item @click="navigateToPath(-1)" style="cursor: pointer">根目录</el-breadcrumb-item>
                <el-breadcrumb-item v-for="(item, index) in pathItems" :key="item.id"
                  @click="navigateToPath(index)" style="cursor: pointer">
                  {{ item.name }}
                </el-breadcrumb-item>
              </el-breadcrumb>
              <el-button type="primary" @click="openCreateDialog" :disabled="!selectedAccountId">
                <el-icon><FolderAdd /></el-icon>
                新建文件夹
              </el-button>
            </div>

            <!-- 桌面端表格 -->
            <el-table class="hidden-md-and-down" v-loading="loading" :data="fileList" style="width: 100%"
              @row-dblclick="handleRowDoubleClick">
              <el-table-column label="名称" min-width="300">
                <template #default="{ row }">
                  <div style="display: flex; align-items: center; gap: 8px">
                    <el-icon :size="18">
                      <component :is="getFileIconByName(row.name, row.is_directory)" />
                    </el-icon>
                    <span>{{ row.name }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="大小" width="120" align="right">
                <template #default="{ row }">
                  <span v-if="!row.is_directory">{{ formatFileSize(row.size) }}</span>
                  <span v-else>--</span>
                </template>
              </el-table-column>
              <el-table-column label="修改时间" width="180">
                <template #default="{ row }">
                  {{ formatDateTime(row.modified_time) }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="120" align="center">
                <template #default="{ row }">
                  <el-dropdown trigger="click"
                    @command="(command: string) => handleSingleOperation(command as FileOperationType, row)">
                    <el-button type="primary" size="small">
                      操作 <el-icon class="el-icon--right"><arrow-down /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="STRM_GENERATE">STRM生成</el-dropdown-item>
                        <el-dropdown-item command="SCRAPE_ORGANIZE">刮削整理</el-dropdown-item>
                        <el-dropdown-item
                          v-if="!row.is_directory && (getFileType(row.name) === 'video' || getFileType(row.name) === 'image')"
                          command="GENERATE_ED2K">
                          生成ED2K
                        </el-dropdown-item>
                        <el-dropdown-item command="DELETE" divided>删除</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </template>
              </el-table-column>
            </el-table>

            <!-- 移动端表格 -->
            <el-table class="hidden-md-and-up" v-loading="loading" :data="fileList" style="width: 100%"
              @row-dblclick="handleRowDoubleClick">
              <el-table-column type="expand" width="30">
                <template #default="{ row }">
                  <div style="padding: 0 20px">
                    <p><strong>大小：</strong>{{ row.is_directory ? '--' : formatFileSize(row.size) }}</p>
                    <p><strong>修改时间：</strong>{{ formatDateTime(row.modified_time * 1000) }}</p>
                    <div style="margin-top: 10px">
                      <el-button size="small" type="primary" @click="handleSingleOperation('STRM_GENERATE', row)">
                        STRM生成
                      </el-button>
                      <el-button size="small" type="success" @click="handleSingleOperation('SCRAPE_ORGANIZE', row)">
                        刮削整理
                      </el-button>
                      <el-button
                        v-if="!row.is_directory && (getFileType(row.name) === 'video' || getFileType(row.name) === 'image')"
                        size="small" type="warning" @click="handleSingleOperation('GENERATE_ED2K', row)">
                        生成ED2K
                      </el-button>
                      <el-button size="small" type="danger" @click="handleSingleOperation('DELETE', row)">
                        删除
                      </el-button>
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="文件">
                <template #default="{ row }">
                  <div style="display: flex; align-items: center; gap: 8px">
                    <el-icon :size="18">
                      <component :is="getFileIconByName(row.name, row.is_directory)" />
                    </el-icon>
                    <span>{{ row.name }}</span>
                  </div>
                </template>
              </el-table-column>
            </el-table>

            <!-- 空状态 -->
            <el-empty v-if="!loading && fileList.length === 0" description="当前目录为空" />

            <!-- 分页器 -->
            <div class="pagination-container" style="margin-top: 20px; text-align: center">
              <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize"
                :page-sizes="[100, 200, 500]" :total="total" layout="total, sizes, prev, pager, next, jumper"
                @size-change="handlePageSizeChange" @current-change="handlePageChange" />
            </div>
          </template>
        </div>
      </div>
    </el-card>

    <el-dialog v-model="showCreateDialog" title="新建文件夹" width="400px" :close-on-click-modal="false">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="80px">
        <el-form-item label="文件夹名称" prop="name">
          <el-input v-model="createForm.name" placeholder="请输入文件夹名称" clearable />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showCreateDialog = false">取消</el-button>
          <el-button type="primary" @click="handleCreateDirectory" :loading="createLoading">
            确定
          </el-button>
        </span>
      </template>
    </el-dialog>

    <el-dialog v-model="showStrmTargetDialog" title="选择STRM目标目录" width="600px" :close-on-click-modal="false">
      <div class="strm-target-dialog-content">
        <p class="dialog-tip">请选择STRM文件的目标存放目录：</p>
        <div v-if="strmSourceItem" class="strm-source-info">
          <span class="source-label">源文件：</span>
          <span class="source-name">{{ strmSourceItem.name }}</span>
        </div>
        <div class="dir-selector-container">
          <DirectorySelector
            v-model="strmTargetDir"
            source-type="local"
            @cancel="showStrmTargetDialog = false"
            @select="confirmStrmGenerate"
          />
        </div>
        <div v-if="strmStorePath" class="strm-store-path">
          <span class="store-label">STRM存放路径：</span>
          <code class="store-path">{{ strmStorePath }}</code>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, inject } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, Files, FolderAdd } from '@element-plus/icons-vue'
import type { FileSystemItem, FileOperationType, DirInfo } from '@/typing'
import type { FormInstance, FormRules } from 'element-plus'
import { getFileType, getFileIconByName } from '@/utils/fileIconUtils'
import { formatFileSize } from '@/utils/fileSizeUtils'
import { formatDateTime } from '@/utils/timeUtils'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import DirectorySelector from './DirectorySelector.vue'

interface NetdiskAccount {
  id: number
  name: string
  username: string
  user_id: string
  source_type: '115' | '123' | 'openlist'
  token: string
  created_at: number
  base_url?: string
  password?: string
  app_id_name?: string
  app_id?: string
  token_failed_reason?: string
}

// 响应式数据
const loading = ref(false)
const currentPath = ref('')
const currentPage = ref(1)
const pageSize = ref(100)
const total = ref(0)
const fileList = ref<FileSystemItem[]>([])
const pathItems = ref<FileSystemItem[]>([])

const http: AxiosStatic | undefined = inject('$http')
const accountList = ref<NetdiskAccount[]>([])
const selectedAccountId = ref<number | null>(null)

const showCreateDialog = ref(false)
const createLoading = ref(false)
const createFormRef = ref<FormInstance>()
const createForm = ref({ name: '' })
const createRules = ref<FormRules>({
  name: [
    { required: true, message: '请输入文件夹名称', trigger: 'blur' },
    { min: 1, max: 255, message: '文件夹名称长度在 1 到 255 个字符', trigger: 'blur' }
  ]
})

const showStrmTargetDialog = ref(false)
const strmTargetDir = ref<DirInfo | null>(null)
const strmSourceItem = ref<FileSystemItem | null>(null)
const strmGenerateLoading = ref(false)

// 计算属性
const strmStorePath = computed(() => {
  if (!strmTargetDir.value || !strmSourceItem.value) return ''
  const currentPathStr = pathItems.value.map(p => p.name).join('/')
  const itemPath = currentPathStr ? `${currentPathStr}/${strmSourceItem.value.name}` : strmSourceItem.value.name
  return `${strmTargetDir.value.path}/${itemPath}`
})

// const isMobileDevice = computed(() => isMobile())

// 加载网盘账号列表
async function loadAccountList() {
  if (!http) {
    console.warn('HTTP客户端未注入，无法加载账号列表')
    return
  }

  try {
    const response = await http.get(`${SERVER_URL}/account/list`)

    if (response?.data.code === 200) {
      const data = response.data.data
      accountList.value = data.map((item: NetdiskAccount) => ({
        id: item.id,
        name: item.name,
        username: item.username,
        user_id: item.user_id,
        source_type: item.source_type,
        token: item.token,
        created_at: item.created_at,
        base_url: item.base_url,
        password: item.password,
        app_id_name: item.app_id_name,
        app_id: item.app_id,
        token_failed_reason: item.token_failed_reason || ''
      }))
    } else {
      console.error('加载账号列表失败:', response?.data.message || '未知错误')
      accountList.value = []
    }
  } catch (error) {
    console.error('加载账号列表失败:', error)
    accountList.value = []
  }
}

// 选择账号
function selectAccount(account: NetdiskAccount) {
  selectedAccountId.value = account.id
  pathItems.value = []
  currentPage.value = 1
  loadFileList()
}

// 获取账号图标
function getAccountIcon() {
  return Files
}

// 获取账号类型名称
function getAccountTypeName(sourceType: string): string {
  switch (sourceType) {
    case '115':
      return '115网盘'
    case '123':
      return '123网盘'
    case 'openlist':
      return 'OpenList'
    case 'baidupan':
      return '百度网盘'
    default:
      return '其他'
  }
}

// 加载文件列表
async function loadFileList() {
  if (!selectedAccountId.value) {
    fileList.value = []
    total.value = 0
    return
  }

  if (!http) {
    console.warn('HTTP客户端未注入，无法加载文件列表')
    return
  }

  loading.value = true
  try {
    const currentItemId = pathItems.value.length > 0 ? pathItems.value[pathItems.value.length - 1].id : ''

    const response = await http.get(`${SERVER_URL}/path/files`, {
      params: {
        account_id: selectedAccountId.value,
        path: currentItemId,
        page: currentPage.value,
        page_size: pageSize.value
      },
      timeout: 60000
    })

    if (response?.data.code === 200) {
      const items = response.data.data || []

      fileList.value = items.map((item: FileSystemItem) => ({
        id: item.id,
        name: item.name,
        path: currentPath.value ? `${currentPath.value}/${item.name}` : item.name,
        type: item.is_directory ? 'directory' : getFileType(item.name),
        size: item.size,
        modified_time: item.modified_time,
        is_directory: item.is_directory
      }))
      total.value = items.length
    } else {
      console.error('加载文件列表失败:', response?.data.message || '未知错误')
      fileList.value = []
      total.value = 0
    }
  } catch {
    ElMessage.error('加载文件列表失败')
  } finally {
    loading.value = false
  }
}

// 导航到指定路径
function navigateToPath(index: number) {
  pathItems.value = pathItems.value.slice(0, index + 1)
  currentPage.value = 1
  loadFileList()
}

// 处理行双击事件（进入目录）
function handleRowDoubleClick(row: FileSystemItem) {
  if (row.is_directory) {
    pathItems.value = [...pathItems.value, row]
    currentPage.value = 1
    loadFileList()
  }
}

// 处理分页大小变化
function handlePageSizeChange(newSize: number) {
  pageSize.value = newSize
  currentPage.value = 1
  loadFileList()
}

// 处理页码变化
function handlePageChange(newPage: number) {
  currentPage.value = newPage
  loadFileList()
}

// 处理单个操作
async function handleSingleOperation(operation: FileOperationType, item: FileSystemItem) {
  if (operation === 'DELETE') {
    await handleDeleteItem(item)
    return
  }

  if (operation === 'STRM_GENERATE') {
    strmSourceItem.value = item
    strmTargetDir.value = null
    showStrmTargetDialog.value = true
    return
  }

  try {
    const operationMap = {
      'STRM_GENERATE': 'STRM生成',
      'SCRAPE_ORGANIZE': '刮削整理',
      'GENERATE_ED2K': '生成ED2K'
    }

    await ElMessageBox.confirm(
      `确认对文件 "${item.name}" 执行 ${operationMap[operation]} 操作吗？`,
      '确认操作',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    ElMessage.info(`${operationMap[operation]} 功能开发中...`)
  } catch {
  }
}

async function handleDeleteItem(item: FileSystemItem) {
  try {
    await ElMessageBox.confirm(
      `确认删除 "${item.name}" 吗？${item.is_directory ? '文件夹内的所有内容也将被删除。' : ''}`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    if (!selectedAccountId.value) {
      ElMessage.warning('请先选择网盘账号')
      return
    }

    const currentParentId = pathItems.value.length > 0 ? pathItems.value[pathItems.value.length - 1].id : ''

    const response = await http?.delete(`${SERVER_URL}/path`, {
      params: {
        parent_id: currentParentId,
        file_id: item.id,
        account_id: selectedAccountId.value
      }
    })

    if (response?.data.code === 200) {
      ElMessage.success('删除成功')
      loadFileList()
    } else {
      ElMessage.error(response?.data.message || '删除失败')
    }
  } catch {
    // 用户取消操作或删除失败
  }
}

function openCreateDialog() {
  createForm.value.name = ''
  showCreateDialog.value = true
}

async function handleCreateDirectory() {
  if (!createFormRef.value) return
  if (!selectedAccountId.value) {
    ElMessage.warning('请先选择网盘账号')
    return
  }

  try {
    await createFormRef.value.validate()
    createLoading.value = true

    const currentParentId = pathItems.value.length > 0 ? pathItems.value[pathItems.value.length - 1].id : ''
    const currentParentPath = pathItems.value.length > 0 ? pathItems.value[pathItems.value.length - 1].path : ''

    const account = accountList.value.find(a => a.id === selectedAccountId.value)

    const response = await http?.post(`${SERVER_URL}/path/create`, {
      parent_id: currentParentId,
      parent_path: currentParentPath,
      name: createForm.value.name.trim(),
      source_type: account?.source_type || '115',
      account_id: selectedAccountId.value,
    })

    if (response?.data.code === 200) {
      ElMessage.success('创建文件夹成功')
      showCreateDialog.value = false
      createForm.value.name = ''
      loadFileList()
    } else {
      ElMessage.error(response?.data.message || '创建文件夹失败')
    }
  } catch {
    ElMessage.error('创建文件夹失败')
  } finally {
    createLoading.value = false
  }
}

async function confirmStrmGenerate() {
  if (!strmTargetDir.value || !strmSourceItem.value) {
    ElMessage.warning('请选择目标目录')
    return
  }

  if (!selectedAccountId.value) {
    ElMessage.warning('请先选择网盘账号')
    return
  }

  try {
    strmGenerateLoading.value = true

    // const currentPathStr = pathItems.value.map(p => p.name).join('/')
    // const itemPath = currentPathStr ? `${currentPathStr}/${strmSourceItem.value.name}` : strmSourceItem.value.name

    const response = await http?.post(`${SERVER_URL}/sync/manual`, {
      path_id: strmSourceItem.value.id,
      // path: itemPath,
      target_path: strmTargetDir.value.path,
      // is_file: !strmSourceItem.value.is_directory,
      account_id: selectedAccountId.value,
    })

    if (response?.data.code === 200) {
      ElMessage.success('STRM生成任务已提交')
      showStrmTargetDialog.value = false
      strmSourceItem.value = null
      strmTargetDir.value = null
    } else {
      ElMessage.error(response?.data.message || 'STRM生成失败')
    }
  } catch {
    ElMessage.error('STRM生成失败')
  } finally {
    strmGenerateLoading.value = false
  }
}

// 监听路径变化
watch([currentPath, currentPage, pageSize], () => {
  loadFileList()
}, { immediate: false })

// 组件挂载时加载数据
onMounted(async () => {
  await loadAccountList()
})
</script>

<style scoped>
.file-manager-container {
  padding: 20px;
}

.file-manager-layout {
  display: flex;
  width: 100%;
  gap: 20px;
  min-height: calc(100vh - 300px);
}

.account-sidebar {
  width: 280px;
  flex-shrink: 0;
  background: #f5f7fa;
  border-radius: 4px;
  overflow: hidden;
}

.file-content {
  flex: 1;
  min-width: 0;
  background: #fff;
  border-radius: 4px;
  padding: 20px;
}

.sidebar-header {
  padding: 16px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
}

.sidebar-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.account-list {
  max-height: calc(100vh - 400px);
  overflow-y: auto;
}

.account-item {
  padding: 16px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  cursor: pointer;
  transition: all 0.3s;
}

.account-item:hover {
  background: #ecf5ff;
}

.account-item.active {
  background: #e6f7ff;
  border-left: 3px solid #409eff;
}

.account-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.account-icon {
  font-size: 24px;
  color: #409eff;
}

.account-details {
  flex: 1;
}

.account-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 4px;
}

.no-account-selected {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 400px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
}

.header-left .card-title {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.header-left .card-subtitle {
  margin: 4px 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

@media (max-width: 768px) {
  .file-manager-layout {
    flex-direction: column;
  }

  .account-sidebar {
    width: 100%;
    max-height: 300px;
  }

  .file-manager-container {
    padding: 10px;
  }

  .card-header {
    flex-direction: column;
    align-items: stretch;
  }

  .header-right {
    justify-content: space-between;
  }
}

.strm-target-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.strm-target-dialog-content .dialog-tip {
  margin: 0;
  color: #606266;
  font-size: 14px;
}

.strm-source-info {
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 14px;
}

.strm-source-info .source-label {
  color: #909399;
  margin-right: 8px;
}

.strm-source-info .source-name {
  color: #303133;
  font-weight: 500;
}

.strm-store-path {
  padding: 12px;
  background: #f0f9eb;
  border-radius: 4px;
  font-size: 14px;
  border: 1px solid #e1f3d8;
}

.strm-store-path .store-label {
  color: #67c23a;
  margin-right: 8px;
  font-weight: 500;
}

.strm-store-path .store-path {
  color: #303133;
  background: #fff;
  padding: 2px 6px;
  border-radius: 2px;
  border: 1px solid #dcdfe6;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  word-break: break-all;
}

.dir-selector-container {
  height: 400px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 12px;
}
</style>
