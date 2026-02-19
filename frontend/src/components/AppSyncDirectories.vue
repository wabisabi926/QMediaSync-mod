<template>
  <div class="sync-directories-page">
    <div class="page-header">
      <div class="header-content">
        <div class="header-title-section">
          <h1 class="page-title">
            <el-icon class="title-icon"><FolderOpened /></el-icon>
            同步目录管理
          </h1>
          <p class="page-subtitle">管理您的云盘与本地目录的同步配置</p>
        </div>
        <div class="header-actions">
          <el-button type="primary" class="add-btn" @click="handleAdd">
            <el-icon><Plus /></el-icon>
            <span class="btn-text">添加同步目录</span>
          </el-button>
        </div>
      </div>
      <div class="stats-bar mobile-hidden">
        <div class="stat-item">
          <div class="stat-icon total">
            <el-icon><Files /></el-icon>
          </div>
          <div class="stat-info">
            <span class="stat-value">{{ directories.length }}</span>
            <span class="stat-label">总目录数</span>
          </div>
        </div>
        <div class="stat-item">
          <div class="stat-icon running">
            <el-icon><Loading /></el-icon>
          </div>
          <div class="stat-info">
            <span class="stat-value">{{ runningCount }}</span>
            <span class="stat-label">运行中</span>
          </div>
        </div>
        <div class="stat-item">
          <div class="stat-icon waiting">
            <el-icon><Clock /></el-icon>
          </div>
          <div class="stat-info">
            <span class="stat-value">{{ waitingCount }}</span>
            <span class="stat-label">等待中</span>
          </div>
        </div>
        <div class="stat-item">
          <div class="stat-icon cron">
            <el-icon><Timer /></el-icon>
          </div>
          <div class="stat-info">
            <span class="stat-value">{{ cronEnabledCount }}</span>
            <span class="stat-label">定时同步</span>
          </div>
        </div>
      </div>
    </div>

    <div class="directories-content">
      <div class="directories-grid" v-if="directories.length > 0">
        <div
          class="directory-card"
          v-for="(row, index) in directories"
          :key="row.id || index"
          :class="{ 'is-running': row.is_running === 2, 'is-waiting': row.is_running === 1 }"
        >
          <div class="card-status-bar" :class="getStatusClass(row)"></div>
          <div class="card-main">
            <div class="card-header">
              <div class="card-title-wrapper">
                <el-tooltip :content="'目录ID：' + row.base_cid" placement="bottom">
                  <span class="card-id">#{{ row.id }}</span>
                </el-tooltip>
                <span class="card-path">{{ row.remote_path }}</span>
              </div>
              <el-tag :type="sourceTypeTagMap[row.source_type]" class="source-tag" effect="light">
                {{ sourceTypeMap[row.source_type] }}
              </el-tag>
            </div>

            <div class="card-body">
              <div class="info-row" v-if="row.source_type !== 'local'">
                <div class="info-icon">
                  <el-icon><User /></el-icon>
                </div>
                <div class="info-content">
                  <span class="info-label">关联账号</span>
                  <span class="info-value">{{ row.account_name }}</span>
                </div>
              </div>

              <div class="info-row">
                <div class="info-icon">
                  <el-icon><Folder /></el-icon>
                </div>
                <div class="info-content">
                  <span class="info-label">目标路径</span>
                  <span class="info-value path-value">{{ GetFullPath(row) }}</span>
                </div>
              </div>

              <div class="info-row">
                <div class="info-icon">
                  <el-icon><Calendar /></el-icon>
                </div>
                <div class="info-content">
                  <span class="info-label">添加时间</span>
                  <span class="info-value">{{ formatTime(row.created_at) }}</span>
                </div>
              </div>

              <div class="info-row">
                <div class="info-icon">
                  <el-icon><Refresh /></el-icon>
                </div>
                <div class="info-content">
                  <span class="info-label">最后同步</span>
                  <span class="info-value">{{ formatTime(row.last_sync_at) || '从未同步' }}</span>
                </div>
              </div>

              <div class="info-row toggle-row">
                <div class="info-icon">
                  <el-icon><Timer /></el-icon>
                </div>
                <div class="info-content">
                  <el-tooltip
                    effect="dark"
                    content="开启后会根据strm设置中的cron表达式定时同步数据，如果该同步目录内的资源变动概率较小，建议关闭定时同步"
                    placement="top"
                  >
                    <span class="info-label with-tooltip">
                      定时同步
                      <el-icon class="help-icon"><QuestionFilled /></el-icon>
                    </span>
                  </el-tooltip>
                  <el-switch
                    v-model="row.enable_cron"
                    :active-value="true"
                    :inactive-value="false"
                    @change="toggleCron(row)"
                    active-color="#67c23a"
                    inactive-color="#dcdfe6"
                  />
                </div>
              </div>

              <div class="status-row">
                <div class="status-indicator" :class="getStatusClass(row)">
                  <el-icon v-if="row.is_running === 2" class="rotating"><Loading /></el-icon>
                  <el-icon v-else-if="row.is_running === 1"><Clock /></el-icon>
                  <el-icon v-else><CircleCheck /></el-icon>
                  <span>{{ getStatusText(row) }}</span>
                </div>
              </div>
            </div>

            <div class="card-footer">
              <el-tooltip content="删除所有缓存数据后执行同步，可处理所有网盘文件变更" placement="top">
                <el-button
                  type="warning"
                  size="small"
                  plain
                  @click="handleFullStart(row, index)"
                  :loading="row.starting"
                  v-if="(row.source_type === '115' || row.source_type === 'baidupan') && row.is_running === 0"
                >
                  <el-icon><RefreshRight /></el-icon>
                  全量同步
                </el-button>
              </el-tooltip>

              <el-button
                type="success"
                size="small"
                plain
                @click="handleStart(row, index)"
                :loading="row.starting"
                v-if="row.is_running === 0"
              >
                <el-icon><VideoPlay /></el-icon>
                同步
              </el-button>

              <el-button
                type="info"
                size="small"
                plain
                @click="handleStop(row, index)"
                :loading="row.stopping"
                v-if="row.is_running !== 0"
              >
                <el-icon><VideoPause /></el-icon>
                停止
              </el-button>

              <el-button
                type="primary"
                size="small"
                plain
                @click="handleEdit(row)"
              >
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>

              <el-button
                type="danger"
                size="small"
                plain
                @click="handleDelete(row, index)"
                :loading="row.deleting"
              >
                <el-icon><Delete /></el-icon>
                删除
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <div class="empty-state" v-else-if="!loading">
        <div class="empty-illustration">
          <el-icon class="empty-icon"><FolderOpened /></el-icon>
          <div class="empty-dots">
            <span></span>
            <span></span>
            <span></span>
          </div>
        </div>
        <h3 class="empty-title">暂无同步目录</h3>
        <p class="empty-description">点击上方按钮添加您的第一个同步目录</p>
        <el-button type="primary" @click="handleAdd">
          <el-icon><Plus /></el-icon>
          添加同步目录
        </el-button>
      </div>

      <div class="loading-state" v-if="loading">
        <el-icon class="loading-icon rotating"><Loading /></el-icon>
        <span>加载中...</span>
      </div>

      <div class="page-footer-tips">
        <div class="tips-header">
          <el-icon class="tips-icon"><InfoFilled /></el-icon>
          <span>使用说明</span>
        </div>
        <div class="tips-content">
          <div class="tip-group">
            <div class="tip-group-title">
              <el-icon><Warning /></el-icon>
              <span>115网盘</span>
            </div>
            <div class="tip-group-items">
              <div class="tip-item">
                <span class="tip-bullet">•</span>
                <span>无法感知网盘的文件夹重命名等操作，如果发现文件夹名字不对可以手动点击：<strong>重置&同步</strong></span>
              </div>
              <div class="tip-item">
                <span class="tip-bullet">•</span>
                <span>"全量同步"会删除所有缓存数据（不会删除本地文件），然后执行同步，可以处理所有网盘文件变更</span>
              </div>
              <div class="tip-item tip-highlight">
                <span class="tip-bullet">★</span>
                <span>请按照电影和电视剧分开添加同步目录，电影的同步速度非常快，电视剧的同步速度较慢</span>
              </div>
            </div>
          </div>
          <div class="tip-group">
            <div class="tip-group-title">
              <el-icon><Timer /></el-icon>
              <span>百度网盘</span>
            </div>
            <div class="tip-group-items">
              <div class="tip-item">
                <span class="tip-bullet">•</span>
                <span>只查询上次同步时间之后修改的文件列表，不会查询所有文件、无法感知文件和文件夹删除</span>
              </div>
              <div class="tip-item">
                <span class="tip-bullet">•</span>
                <span>增量同步只能单线程，每分钟最多执行8次请求，每次请求1000个文件，如果单次变更文件数量大于8000，同步就会很慢</span>
              </div>
              <div class="tip-item">
                <span class="tip-bullet">•</span>
                <span>"全量同步"会删除所有缓存数据（不会删除本地文件），然后递归查询所有文件</span>
              </div>
              <div class="tip-item tip-highlight">
                <span class="tip-bullet">★</span>
                <span>每天的第一次同步会执行"全量同步"，后续同步会执行"增量同步"</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject, onMounted, onUnmounted, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Loading, Folder, VideoPlay, Edit, Delete, Warning, VideoPause,
  InfoFilled, Timer, FolderOpened, Files, Clock, User, Calendar, Refresh,
  RefreshRight, CircleCheck, QuestionFilled
} from '@element-plus/icons-vue'
import { formatTime } from '@/utils/timeUtils'
import { isMobile, onDeviceTypeChange } from '@/utils/deviceUtils'
import { sourceTypeTagMap, sourceTypeMap } from '@/utils/sourceTypeUtils'

interface SyncDirectory {
  id: number
  base_cid: string
  local_path: string
  remote_path: string
  strm_path: string
  created_at: number
  updated_at: number
  last_sync_at: number
  deleting?: boolean
  starting?: boolean
  source_type: string
  account_id: number
  account_name: string
  enable_cron: boolean
  is_running: number
  stopping?: boolean
}

const http: AxiosStatic | undefined = inject('$http')
const router = useRouter()

const directories = ref<SyncDirectory[]>([])
const loading = ref(false)
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(9999)

const checkIsMobile = ref(isMobile())

const runningCount = computed(() => directories.value.filter(d => d.is_running === 2).length)
const waitingCount = computed(() => directories.value.filter(d => d.is_running === 1).length)
const cronEnabledCount = computed(() => directories.value.filter(d => d.enable_cron).length)

const getStatusClass = (row: SyncDirectory) => {
  if (row.is_running === 2) return 'status-running'
  if (row.is_running === 1) return 'status-waiting'
  return 'status-idle'
}

const getStatusText = (row: SyncDirectory) => {
  if (row.is_running === 2) return '运行中'
  if (row.is_running === 1) return '等待中'
  return '空闲'
}

const checkMobile = () => {
  checkIsMobile.value = isMobile()
}

const GetFullPath = (row: SyncDirectory) => {
  const pathSeparator = row.local_path.startsWith('/') ? '/' : '\\'
  if (row.source_type == 'local') {
    return row.local_path
  }
  let remotePath = row.remote_path
  if (pathSeparator === '/') {
    remotePath = remotePath.replace(/\\/g, pathSeparator)
  } else {
    remotePath = remotePath.replace(/\//g, pathSeparator)
  }
  return `${row.local_path}${pathSeparator}${remotePath}`
}

const loadDirectories = async () => {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/path-list`, {
      timeout: 5000,
      params: {
        page: currentPage.value,
        page_size: pageSize.value,
      },
    })

    if (response?.data.code === 200) {
      directories.value = response.data.data.list || []
      total.value = response.data.data.total || 0
    } else {
      ElMessage.error(response?.data.message || '加载同步目录失败')
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

const updatePathesStatus = async () => {
  const response = await http?.get(`${SERVER_URL}/sync/path-list`)

  if (response?.data.code === 200) {
    for (const p of response.data.data.list || []) {
      const path = directories.value.find(pa => pa.id === p.id)
      if (path) {
        path.is_running = p.is_running
      }
    }
  }
  autoRefreshEnabled = true
}

const handleAdd = () => {
  router.push({ name: 'sync-directory-add' })
}

const handleEdit = (row: SyncDirectory) => {
  router.push({ name: 'sync-directory-edit', params: { id: row.id } })
}

const handleDelete = async (row: SyncDirectory, index: number) => {
  try {
    await ElMessageBox.confirm(
      `不会删除已经同步的元数据和STRM文件，确定要删除同步目录 "${row.local_path}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )

    directories.value[index].deleting = true

    const formData = {
      id: row.id || '',
    }

    const response = await http?.post(`${SERVER_URL}/sync/path-delete`, formData, {
      headers: {
        'Content-Type': 'application/json',
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

const handleFullStart = async (row: SyncDirectory, index: number) => {
  try {
    directories.value[index].starting = true

    const formData = {
      id: row.id || '',
    }

    const response = await http?.post(`${SERVER_URL}/sync/path/full-start`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success(`同步目录 "${row.local_path}" 启动成功`)
    } else {
      ElMessage.error(response?.data.message || '启动同步目录失败')
    }
  } catch {
    console.error('启动同步目录错误')
    ElMessage.error('启动同步目录失败')
  } finally {
    if (directories.value[index]) {
      directories.value[index].starting = false
    }
  }
}

const handleStart = async (row: SyncDirectory, index: number) => {
  try {
    directories.value[index].starting = true

    const formData = {
      id: row.id || '',
    }

    const response = await http?.post(`${SERVER_URL}/sync/path/start`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success(`同步目录 "${row.local_path}" 启动成功`)
    } else {
      ElMessage.error(response?.data.message || '启动同步目录失败')
    }
  } catch {
    console.error('启动同步目录错误')
    ElMessage.error('启动同步目录失败')
  } finally {
    if (directories.value[index]) {
      directories.value[index].starting = false
    }
  }
}

const handleStop = async (row: SyncDirectory, index: number) => {
  try {
    directories.value[index].stopping = true

    const formData = {
      id: row.id || '',
    }

    const response = await http?.post(`${SERVER_URL}/sync/path/stop`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success(`同步目录 "${row.local_path}" 停止成功`)
    } else {
      ElMessage.error(response?.data.message || '停止同步目录失败')
    }
  } catch {
    console.error('停止同步目录错误')
    ElMessage.error('停止同步目录失败')
  } finally {
    if (directories.value[index]) {
      directories.value[index].stopping = false
    }
  }
}

const toggleCron = async (row: SyncDirectory) => {
  try {
    const formData = {
      id: row.id || '',
    }

    const response = await http?.post(`${SERVER_URL}/sync/path/toggle-cron`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success(row.enable_cron ? '开启定时同步成功' : '关闭定时同步成功')
    } else {
      row.enable_cron = !row.enable_cron
      ElMessage.error(response?.data.message || '切换定时同步状态失败')
    }
  } catch {
    console.error('切换定时同步状态错误')
    row.enable_cron = !row.enable_cron
    ElMessage.error('切换定时同步状态失败')
  }
}

let autoRefreshEnabled = true
const autoRefreshTimer = ref<number | null>(null)

const checkAndSetAutoRefresh = () => {
  if (autoRefreshTimer.value) {
    clearInterval(autoRefreshTimer.value)
    autoRefreshTimer.value = null
  }

  autoRefreshTimer.value = window.setInterval(() => {
    if (!autoRefreshEnabled) {
      return
    }
    autoRefreshEnabled = false
    updatePathesStatus()
  }, 2000)
}

const clearAutoRefreshTimer = () => {
  if (autoRefreshTimer.value) {
    clearInterval(autoRefreshTimer.value)
    autoRefreshTimer.value = null
  }
}

let removeDeviceTypeListener: (() => void) | null = null

onMounted(() => {
  checkMobile()
  removeDeviceTypeListener = onDeviceTypeChange((newIsMobile) => {
    checkIsMobile.value = newIsMobile
  })
  loadDirectories()
  checkAndSetAutoRefresh()
})

onUnmounted(() => {
  if (removeDeviceTypeListener) {
    removeDeviceTypeListener()
  }
  clearAutoRefreshTimer()
})
</script>

<style scoped>
.sync-directories-page {
  min-height: 100%;
  background: #f5f7fa;
  padding: 0;
}

.page-header {
  background: #fff;
  padding: 24px;
  border-bottom: 1px solid #ebeef5;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 20px;
}

.header-title-section {
  flex: 1;
}

.page-title {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.title-icon {
  font-size: 28px;
  color: #409eff;
}

.page-subtitle {
  margin: 0;
  font-size: 14px;
  color: #909399;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.add-btn {
  background: #409eff !important;
  border-color: #409eff !important;
  transition: all 0.3s ease;
}

.add-btn:hover {
  background: #66b1ff !important;
  border-color: #66b1ff !important;
}

.stats-bar {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #f5f7fa;
  padding: 12px 16px;
  border-radius: 8px;
  min-width: 140px;
}

.stat-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
}

.stat-icon.total {
  background: #ecf5ff;
  color: #409eff;
}

.stat-icon.running {
  background: #f0f9eb;
  color: #67c23a;
}

.stat-icon.waiting {
  background: #fdf6ec;
  color: #e6a23c;
}

.stat-icon.cron {
  background: #f4f4f5;
  color: #909399;
}

.stat-info {
  display: flex;
  flex-direction: column;
}

.stat-value {
  font-size: 20px;
  font-weight: 600;
  line-height: 1.2;
  color: #303133;
}

.stat-label {
  font-size: 12px;
  color: #909399;
}

.directories-content {
  padding: 24px;
}

.directories-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
}

.directory-card {
  background: #fff;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  transition: all 0.3s ease;
  position: relative;
}

.directory-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
}

.directory-card.is-running {
  box-shadow: 0 2px 12px rgba(103, 194, 58, 0.2);
}

.directory-card.is-running:hover {
  box-shadow: 0 8px 24px rgba(103, 194, 58, 0.3);
}

.directory-card.is-waiting {
  box-shadow: 0 2px 12px rgba(230, 162, 60, 0.2);
}

.directory-card.is-waiting:hover {
  box-shadow: 0 8px 24px rgba(230, 162, 60, 0.3);
}

.card-status-bar {
  height: 4px;
  background: #e4e7ed;
}

.card-status-bar.status-running {
  background: linear-gradient(90deg, #67c23a, #95d475);
  animation: pulse 2s infinite;
}

.card-status-bar.status-waiting {
  background: linear-gradient(90deg, #e6a23c, #f0c78a);
  animation: pulse 2s infinite;
}

.card-status-bar.status-idle {
  background: linear-gradient(90deg, #909399, #c0c4cc);
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.card-main {
  padding: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f2f5;
}

.card-title-wrapper {
  flex: 1;
  min-width: 0;
}

.card-id {
  display: inline-block;
  font-size: 12px;
  color: #909399;
  background: #f5f7fa;
  padding: 2px 8px;
  border-radius: 4px;
  margin-right: 8px;
}

.card-path {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  word-break: break-all;
}

.source-tag {
  flex-shrink: 0;
  margin-left: 8px;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.info-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
  flex-shrink: 0;
}

.info-content {
  flex: 1;
  display: flex;
  justify-content: space-between;
  align-items: center;
  min-width: 0;
}

.info-label {
  font-size: 13px;
  color: #909399;
  display: flex;
  align-items: center;
  gap: 4px;
}

.info-label.with-tooltip {
  cursor: help;
}

.help-icon {
  font-size: 14px;
  color: #c0c4cc;
}

.info-value {
  font-size: 14px;
  color: #303133;
  text-align: right;
}

.path-value {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 12px;
  word-break: break-all;
  max-width: 200px;
}

.toggle-row .info-content {
  justify-content: space-between;
}

.status-row {
  margin-top: 8px;
  padding-top: 12px;
  border-top: 1px dashed #ebeef5;
}

.status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
}

.status-indicator.status-running {
  background: #f0f9eb;
  color: #67c23a;
}

.status-indicator.status-waiting {
  background: #fdf6ec;
  color: #e6a23c;
}

.status-indicator.status-idle {
  background: #f5f7fa;
  color: #909399;
}

.rotating {
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.card-footer {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 16px;
  margin-top: 12px;
  border-top: 1px solid #f0f2f5;
}

.card-footer .el-button {
  flex: 1;
  min-width: 70px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  margin-bottom: 24px;
}

.empty-illustration {
  position: relative;
  margin-bottom: 24px;
}

.empty-icon {
  font-size: 80px;
  color: #dcdfe6;
}

.empty-dots {
  display: flex;
  justify-content: center;
  gap: 8px;
  margin-top: 16px;
}

.empty-dots span {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #dcdfe6;
  animation: bounce 1.4s infinite ease-in-out both;
}

.empty-dots span:nth-child(1) { animation-delay: -0.32s; }
.empty-dots span:nth-child(2) { animation-delay: -0.16s; }

@keyframes bounce {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}

.empty-title {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.empty-description {
  margin: 0 0 24px 0;
  font-size: 14px;
  color: #909399;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  background: #fff;
  border-radius: 16px;
  color: #909399;
  gap: 12px;
}

.loading-icon {
  font-size: 32px;
  color: #409eff;
}

.page-footer-tips {
  border: none;
  border-radius: 16px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.tips-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
  font-size: 15px;
  font-weight: 600;
}

.tips-icon {
  font-size: 18px;
}

.tips-content {
  display: flex;
  flex-wrap: wrap;
  gap: 0;
}

.tip-group {
  flex: 1;
  min-width: 300px;
  padding: 20px;
  border-right: 1px solid #f0f2f5;
}

.tip-group:last-child {
  border-right: none;
}

.tip-group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding-bottom: 10px;
  border-bottom: 2px solid #f0f2f5;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.tip-group-title .el-icon {
  color: #e6a23c;
  font-size: 18px;
}

.tip-group-items {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tip-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 13px;
  color: #606266;
  line-height: 1.6;
}

.tip-bullet {
  flex-shrink: 0;
  width: 16px;
  color: #c0c4cc;
  text-align: center;
}

.tip-item strong {
  color: #409eff;
}

.tip-highlight {
  background: linear-gradient(135deg, #fdf6ec 0%, #fef8eb 100%);
  margin: 6px -12px;
  padding: 12px;
  border-radius: 8px;
  border-left: 3px solid #e6a23c;
}

.tip-highlight .tip-bullet {
  color: #e6a23c;
}

.tip-highlight span:last-child {
  color: #8b6b3d;
}

@media (max-width: 768px) {
  .page-header {
    padding: 12px;
    background: #fff;
  }

  .header-title-section {
    display: none;
  }

  .header-content {
    margin-bottom: 0;
  }

  .header-actions {
    justify-content: stretch;
  }

  .header-actions .add-btn {
    width: 100%;
    background: #409eff !important;
    border-color: #409eff !important;
    color: #fff !important;
  }

  .header-actions .add-btn:hover {
    background: #66b1ff !important;
    border-color: #66b1ff !important;
    transform: none;
  }

  .mobile-hidden {
    display: none !important;
  }

  .directories-content {
    padding: 12px;
  }

  .directories-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .directory-card {
    border-radius: 12px;
  }

  .card-main {
    padding: 12px;
  }

  .card-header {
    margin-bottom: 12px;
    padding-bottom: 10px;
  }

  .card-id {
    font-size: 11px;
    padding: 2px 6px;
  }

  .card-path {
    font-size: 14px;
  }

  .source-tag {
    font-size: 11px;
  }

  .card-body {
    gap: 10px;
  }

  .info-row {
    gap: 10px;
  }

  .info-icon {
    width: 28px;
    height: 28px;
    font-size: 14px;
  }

  .info-label {
    font-size: 12px;
  }

  .info-value {
    font-size: 13px;
  }

  .path-value {
    font-size: 11px;
    max-width: 140px;
  }

  .status-row {
    margin-top: 6px;
    padding-top: 10px;
  }

  .status-indicator {
    padding: 5px 10px;
    font-size: 12px;
  }

  .card-footer {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
    padding-top: 12px;
    margin-top: 10px;
  }

  .card-footer .el-button {
    flex: none;
    min-width: 0;
    width: 100%;
    margin: 0;
  }

  .card-footer .el-button :deep(.el-icon) {
    margin-right: 4px;
  }

  .empty-state {
    padding: 40px 16px;
    border-radius: 12px;
  }

  .empty-icon {
    font-size: 60px;
  }

  .empty-title {
    font-size: 16px;
  }

  .empty-description {
    font-size: 13px;
    margin-bottom: 20px;
  }

  .page-footer-tips {
    border-radius: 12px;
  }

  .tips-header {
    padding: 12px 14px;
    font-size: 14px;
  }

  .tip-group {
    padding: 14px;
    border-right: none;
    border-bottom: 1px solid #f0f2f5;
  }

  .tip-group:last-child {
    border-bottom: none;
  }

  .tip-group-title {
    font-size: 14px;
    margin-bottom: 12px;
    padding-bottom: 8px;
  }

  .tip-group-items {
    gap: 8px;
  }

  .tip-item {
    font-size: 12px;
  }

  .tip-highlight {
    margin: 4px -8px;
    padding: 10px;
  }
}

@media (max-width: 480px) {
  .info-content {
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
  }

  .info-value {
    text-align: left;
  }

  .path-value {
    max-width: 100%;
  }

  .toggle-row .info-content {
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
    width: 100%;
  }

  .card-footer {
    grid-template-columns: 1fr;
    gap: 6px;
  }
}
</style>
