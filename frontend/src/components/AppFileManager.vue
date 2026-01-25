<template>
  <div class="main-content-container file-manager-container full-width-container">
    <el-card shadow="none" class="full-width-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <h2 class="card-title hidden-md-and-down">网盘文件浏览器（仅测试UI，暂无功能）</h2>
            <p class="card-subtitle">
              浏览和管理媒体文件，支持STRM生成、刮削整理和ED2K生成操作
            </p>
          </div>
          <div class="header-right">
            <el-select
              v-model="pageSize"
              style="width: 100px; margin-right: 10px"
              @change="handlePageSizeChange"
            >
              <el-option label="100" :value="100" />
              <el-option label="200" :value="200" />
              <el-option label="500" :value="500" />
            </el-select>
            <el-checkbox v-model="batchMode">批量操作</el-checkbox>
          </div>
        </div>
      </template>

      <!-- 面包屑导航 -->
      <el-breadcrumb separator="/" style="margin-bottom: 16px">
        <el-breadcrumb-item @click="navigateToPath('')" style="cursor: pointer">根目录</el-breadcrumb-item>
        <el-breadcrumb-item
          v-for="(segment, index) in pathSegments"
          :key="index"
          @click="navigateToPath(pathSegments.slice(0, index + 1).join('/'))"
          style="cursor: pointer"
        >
          {{ segment }}
        </el-breadcrumb-item>
      </el-breadcrumb>

      <!-- 批量操作按钮组 -->
      <div v-if="batchMode" style="margin-bottom: 16px">
        <el-button
          type="primary"
          :disabled="selectedItems.length === 0"
          @click="handleBatchOperation('STRM_GENERATE')"
        >
          <el-icon><VideoPlay /></el-icon>
          批量STRM生成 ({{ selectedItems.length }})
        </el-button>
        <el-button
          type="success"
          :disabled="selectedItems.length === 0"
          @click="handleBatchOperation('SCRAPE_ORGANIZE')"
        >
          <el-icon><FolderOpened /></el-icon>
          批量刮削整理 ({{ selectedItems.length }})
        </el-button>
        <el-button
          type="warning"
          :disabled="selectedVideoItems.length === 0"
          @click="handleBatchOperation('GENERATE_ED2K')"
        >
          <el-icon><Link /></el-icon>
          批量生成ED2K ({{ selectedVideoItems.length }})
        </el-button>
      </div>

      <!-- 桌面端表格 -->
      <el-table
        class="hidden-md-and-down"
        v-loading="loading"
        :data="fileList"
        style="width: 100%"
        @selection-change="handleSelectionChange"
        @row-dblclick="handleRowDoubleClick"
      >
        <el-table-column
          v-if="batchMode"
          type="selection"
          width="50"
          :selectable="isFileSelectable"
        />
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
            {{ formatDateTime(row.modified_time * 1000) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" align="center">
          <template #default="{ row }">
            <el-dropdown trigger="click" @command="(command: string) => handleSingleOperation(command as FileOperationType, row)">
              <el-button type="primary" size="small">
                操作 <el-icon class="el-icon--right"><arrow-down /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="STRM_GENERATE">STRM生成</el-dropdown-item>
                  <el-dropdown-item command="SCRAPE_ORGANIZE">刮削整理</el-dropdown-item>
                  <el-dropdown-item
                    v-if="!row.is_directory && (getFileType(row.name) === 'video' || getFileType(row.name) === 'image')"
                    command="GENERATE_ED2K"
                  >
                    生成ED2K
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>

      <!-- 移动端表格 -->
      <el-table
        class="hidden-md-and-up"
        v-loading="loading"
        :data="fileList"
        style="width: 100%"
        @selection-change="handleSelectionChange"
        @row-dblclick="handleRowDoubleClick"
      >
        <el-table-column
          v-if="batchMode"
          type="selection"
          width="50"
          :selectable="isFileSelectable"
        />
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
                  size="small"
                  type="warning"
                  @click="handleSingleOperation('GENERATE_ED2K', row)"
                >
                  生成ED2K
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
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[100, 200, 500]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handlePageSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, VideoPlay, FolderOpened, Link, Folder, Document, Picture } from '@element-plus/icons-vue'
import type { FileSystemItem, FileOperationType, FileListResponse } from '@/typing'
import { getFileType, getFileIconByName } from '@/utils/fileIconUtils'
import { formatFileSize } from '@/utils/fileSizeUtils'
import { formatDateTime } from '@/utils/timeUtils'
import { isMobile } from '@/utils/deviceUtils'

// 响应式数据
const loading = ref(false)
const batchMode = ref(false)
const currentPath = ref('')
const currentPage = ref(1)
const pageSize = ref(100)
const total = ref(0)
const fileList = ref<FileSystemItem[]>([])
const selectedItems = ref<FileSystemItem[]>([])

// 计算属性
const pathSegments = computed(() => {
  return currentPath.value ? currentPath.value.split('/').filter(Boolean) : []
})

const selectedVideoItems = computed(() => {
  return selectedItems.value.filter(item =>
    !item.is_directory && (getFileType(item.name) === 'video' || getFileType(item.name) === 'image')
  )
})

const isMobileDevice = computed(() => isMobile())

// 创建模拟数据
function createMockData(path: string, page: number, size: number): FileListResponse {
  // 模拟不同层级的文件结构
  const mockItems: FileSystemItem[] = []

  if (path === '') {
    // 根目录
    mockItems.push(
      { name: '电影', path: '电影', type: 'directory', size: 0, modified_time: Date.now() / 1000, is_directory: true },
      { name: '电视剧', path: '电视剧', type: 'directory', size: 0, modified_time: Date.now() / 1000, is_directory: true },
      { name: '纪录片', path: '纪录片', type: 'directory', size: 0, modified_time: Date.now() / 1000, is_directory: true }
    )
  } else if (path === '电影') {
    // 电影目录
    for (let i = 1; i <= 50; i++) {
      mockItems.push({
        name: `电影${i}`,
        path: `电影/电影${i}`,
        type: 'directory',
        size: 0,
        modified_time: Date.now() / 1000 - Math.random() * 86400 * 30,
        is_directory: true
      })
    }
  } else if (path.startsWith('电影/电影')) {
    // 具体电影目录
    const movieName = path.split('/')[1]
    mockItems.push(
      {
        name: `${movieName}.mp4`,
        path: `${path}/${movieName}.mp4`,
        type: 'video',
        size: Math.floor(Math.random() * 2000000000) + 500000000,
        modified_time: Date.now() / 1000 - Math.random() * 86400 * 7,
        is_directory: false
      },
      {
        name: `${movieName}.nfo`,
        path: `${path}/${movieName}.nfo`,
        type: 'nfo',
        size: Math.floor(Math.random() * 5000) + 1000,
        modified_time: Date.now() / 1000 - Math.random() * 86400 * 7,
        is_directory: false
      },
      {
        name: 'poster.jpg',
        path: `${path}/poster.jpg`,
        type: 'image',
        size: Math.floor(Math.random() * 500000) + 100000,
        modified_time: Date.now() / 1000 - Math.random() * 86400 * 7,
        is_directory: false
      },
      {
        name: 'fanart.jpg',
        path: `${path}/fanart.jpg`,
        type: 'image',
        size: Math.floor(Math.random() * 1000000) + 200000,
        modified_time: Date.now() / 1000 - Math.random() * 86400 * 7,
        is_directory: false
      }
    )
  }

  // 分页处理
  const start = (page - 1) * size
  const end = start + size
  const paginatedItems = mockItems.slice(start, end)

  return {
    total: mockItems.length,
    items: paginatedItems,
    current_path: path
  }
}

// 加载文件列表
async function loadFileList() {
  loading.value = true
  try {
    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 300))
    const response = createMockData(currentPath.value, currentPage.value, pageSize.value)

    fileList.value = response.items
    total.value = response.total
  } catch (error) {
    ElMessage.error('加载文件列表失败')
  } finally {
    loading.value = false
  }
}

// 导航到指定路径
function navigateToPath(path: string) {
  currentPath.value = path
  currentPage.value = 1
  selectedItems.value = []
  loadFileList()
}

// 处理行双击事件（进入目录）
function handleRowDoubleClick(row: FileSystemItem) {
  if (row.is_directory) {
    navigateToPath(row.path)
  }
}

// 处理选择变化
function handleSelectionChange(selection: FileSystemItem[]) {
  selectedItems.value = selection
}

// 判断文件是否可选择
function isFileSelectable(row: FileSystemItem): boolean {
  return true // 所有文件和目录都可选择
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

    // 执行操作的占位函数
    await performFileOperation([item.path], operation)

    ElMessage.success(`${operationMap[operation]} 操作已提交`)
  } catch (error) {
    // 用户取消操作，不显示错误
  }
}

// 处理批量操作
async function handleBatchOperation(operation: FileOperationType) {
  try {
    const operationMap = {
      'STRM_GENERATE': 'STRM生成',
      'SCRAPE_ORGANIZE': '刮削整理',
      'GENERATE_ED2K': '生成ED2K'
    }

    const targetItems = operation === 'GENERATE_ED2K' ? selectedVideoItems.value : selectedItems.value

    if (targetItems.length === 0) {
      ElMessage.warning('请先选择要操作的文件')
      return
    }

    await ElMessageBox.confirm(
      `确认对选中的 ${targetItems.length} 个项目执行 ${operationMap[operation]} 操作吗？`,
      '确认批量操作',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    // 执行操作的占位函数
    const paths = targetItems.map(item => item.path)
    await performFileOperation(paths, operation)

    ElMessage.success(`${operationMap[operation]} 批量操作已提交，共处理 ${targetItems.length} 个项目`)

    // 清空选择
    selectedItems.value = []
  } catch (error) {
    // 用户取消操作，不显示错误
  }
}

// 执行文件操作的占位函数
async function performFileOperation(paths: string[], operation: FileOperationType) {
  // 这里是占位函数，后续接入真实API
  console.log('执行文件操作:', { paths, operation })

  // 模拟API调用延时
  await new Promise(resolve => setTimeout(resolve, 500))

  // TODO: 实现真实的API调用
  // const response = await $http.post('/api/files/operate', {
  //   paths: paths,
  //   operation: operation
  // })
}

// 监听路径变化
watch([currentPath, currentPage, pageSize], () => {
  loadFileList()
}, { immediate: false })

// 组件挂载时加载数据
onMounted(() => {
  loadFileList()
})
</script>

<style scoped>
.file-manager-container {
  padding: 20px;
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
</style>
