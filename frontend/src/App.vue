<template>
  <!-- 登录页面 -->
  <div v-if="$route.name === 'login'" class="login-layout">
    <router-view />
  </div>

  <!-- 主应用布局 -->
  <div v-else class="common-layout">
    <el-container>
      <!-- 移动端遮罩层 -->
      <div v-if="isMobile && isMenuOpen" class="mobile-overlay" @click="toggleMenu"></div>

      <!-- 侧边栏 -->
      <el-aside :width="isMobile ? '250px' : '200px'"
        :class="{ 'mobile-aside': isMobile, 'mobile-aside-open': isMobile && isMenuOpen }">
        <!-- 用户信息 -->
        <div class="user-info">
          <div class="user-avatar">
            <el-icon size="24">
              <User />
            </el-icon>
          </div>
          <div class="user-details">
            <div class="username">{{ authStore.user?.username || '用户' }}</div>
            <el-button type="text" size="small" class="logout-btn" @click="handleLogout">
              退出登录
            </el-button>
          </div>
        </div>

        <el-menu :default-active="$route.path" :default-openeds="getDefaultOpeneds()" router class="el-menu-vertical"
          @select="handleMenuSelect">
          <el-menu-item index="/">
            <el-icon>
              <House />
            </el-icon>
            <span>首页</span>
          </el-menu-item>
          <el-menu-item index="/accounts">
            <el-icon>
              <User />
            </el-icon>
            <span>网盘账号</span>
          </el-menu-item>
          <el-sub-menu index="/settings">
            <template #title>
              <el-icon>
                <Setting />
              </el-icon>
              <span>系统设置</span>
            </template>
            <el-menu-item index="/settings/emby">
              <el-icon>
                <VideoPlay />
              </el-icon>
              <span>Emby</span>
            </el-menu-item>
            <el-menu-item index="/proxy">
              <el-icon>
                <Link />
              </el-icon>
              <span>网络代理</span>
            </el-menu-item>
            <el-menu-item index="/settings/notification">
              <el-icon>
                <Promotion />
              </el-icon>
              <span>通知管理</span>
            </el-menu-item>
            <el-menu-item index="/settings/threads">
              <el-icon>
                <Operation />
              </el-icon>
              <span>接口速率</span>
            </el-menu-item>
          </el-sub-menu>
          <el-sub-menu index="/upload-queue">
            <template #title>
              <el-icon>
                <Download />
              </el-icon>
              <span>上传下载</span>
            </template>
            <el-menu-item index="/upload-queue">
              <el-icon>
                <Upload />
              </el-icon>
              <span>上传队列</span>
            </el-menu-item>
            <el-menu-item index="/download-queue">
              <el-icon>
                <Download />
              </el-icon>
              <span>下载队列</span>
            </el-menu-item>
          </el-sub-menu>
          <el-sub-menu index="/sync">
            <template #title>
              <el-icon>
                <DocumentCopy />
              </el-icon>
              <span>STRM同步</span>
            </template>
            <el-menu-item index="/sync-directories">
              <el-icon>
                <FolderOpened />
              </el-icon>
              <span>STRM同步目录</span>
            </el-menu-item>
            <el-menu-item index="/sync-records">
              <el-icon>
                <List />
              </el-icon>
              <span>STRM同步记录</span>
            </el-menu-item>

            <el-menu-item index="/settings/strm">
              <el-icon>
                <Setting />
              </el-icon>
              <span>STRM设置</span>
            </el-menu-item>
          </el-sub-menu>

          <el-sub-menu index="/scrape">
            <template #title>
              <el-icon>
                <Film />
              </el-icon>
              <span>刮削 & 整理</span>
            </template>
            <el-menu-item index="/scrape-pathes">
              <el-icon>
                <FolderOpened />
              </el-icon>
              <span>刮削目录</span>
            </el-menu-item>
            <el-menu-item index="/scrape-records">
              <el-icon>
                <List />
              </el-icon>
              <span>刮削记录</span>
            </el-menu-item>
            <el-menu-item index="/settings/tmdb">
              <el-icon>
                <Film />
              </el-icon>
              <span>TMDB设置</span>
            </el-menu-item>
            <el-menu-item index="/settings/ai">
              <el-icon>
                <View />
              </el-icon>
              <span>AI识别设置</span>
            </el-menu-item>
            <el-menu-item index="/settings/category-strategy">
              <el-icon>
                <Operation />
              </el-icon>
              <span>二级分类设置</span>
            </el-menu-item>
          </el-sub-menu>

          <!-- <el-sub-menu index="/scrape">
            <template #title>
              <el-icon><Film /></el-icon>
              <span>简易影视墙</span>
            </template>
            <el-menu-item index="/scrape-pathes">
              <el-icon><FolderOpened /></el-icon>
              <span>电影</span>
            </el-menu-item>
            <el-menu-item index="/media-import">
              <el-icon><List /></el-icon>
              <span>电视剧</span>
            </el-menu-item>
          </el-sub-menu> -->          <el-sub-menu index="database">
            <template #title>
              <el-icon>
                <DataAnalysis />
              </el-icon>
              <span>数据库备份</span>
            </template>
            <el-menu-item index="/database/backup/settings">
              <el-icon>
                <Setting />
              </el-icon>
              <span>备份设置</span>
            </el-menu-item>
            <el-menu-item index="/database/backup/records">
              <el-icon>
                <List />
              </el-icon>
              <span>备份记录</span>
            </el-menu-item>
            <el-menu-item index="/database/backup/restore">
              <el-icon>
                <RefreshLeft />
              </el-icon>
              <span>备份恢复</span>
            </el-menu-item>
          </el-sub-menu>          <el-menu-item index="/settings/user">
            <el-icon>
              <UserFilled />
            </el-icon>
            <span>用户管理</span>
          </el-menu-item>
        </el-menu>
      </el-aside>

      <!-- 主内容区 -->
      <el-main class="main-content">
        <!-- 移动端顶部栏 -->
        <div v-if="isMobile" class="mobile-header">
          <div class="left-section">
            <el-button type="text" class="menu-toggle" @click="toggleMenu">
              <el-icon size="20">
                <Menu />
              </el-icon>
            </el-button>
            <h2 class="page-title">{{ getCurrentPageTitle() }}</h2>
          </div>
          <el-dropdown class="user-dropdown">
            <el-button type="text" class="user-btn">
              <el-icon>
                <User />
              </el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item>{{ authStore.user?.username }}</el-dropdown-item>
                <el-dropdown-item divided @click="handleLogout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <router-view />
      </el-main>
    </el-container>
  </div>

  <!-- 全局备份/恢复进度弹窗 -->
  <el-dialog
    v-model="backupStore.showProgressDialog"
    :title="backupStore.taskType === 'backup' ? '备份进行中' : '数据库恢复中'"
    :width="isMobile ? '90%' : '600px'"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :show-close="false"
    center
  >
    <div class="backup-progress-content">
      <!-- 进度条 -->
      <el-progress
        :percentage="backupStore.progress?.progress || 0"
        :status="getProgressStatus()"
        :stroke-width="20"
      />

      <!-- 当前步骤 -->
      <div v-if="backupStore.progress?.current_step" class="progress-step">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>{{ backupStore.progress.current_step }}</span>
      </div>

      <!-- 表处理进度 -->
      <div v-if="backupStore.progress?.total_tables" class="progress-tables">
        <span>已处理：{{ backupStore.progress.processed_tables || 0 }} / {{ backupStore.progress.total_tables }} 个表</span>
      </div>

      <!-- 时间信息 -->
      <div v-if="backupStore.progress?.elapsed_seconds !== undefined" class="progress-time">
        <div class="time-item">
          <span class="label">已耗时：</span>
          <span class="value">{{ formatDuration(backupStore.progress.elapsed_seconds) }}</span>
        </div>
        <div v-if="backupStore.progress.estimated_seconds" class="time-item">
          <span class="label">预计剩余：</span>
          <span class="value">{{ formatDuration(backupStore.progress.estimated_seconds - backupStore.progress.elapsed_seconds) }}</span>
        </div>
      </div>

      <!-- 错误重试提示 -->
      <el-alert
        v-if="backupStore.errorRetryCount > 0"
        :title="`网络异常，正在重试 (${backupStore.errorRetryCount}/${3})...`"
        type="warning"
        :closable="false"
        style="margin-top: 16px"
      />
    </div>

    <template #footer>
      <!-- 仅备份任务且运行中时显示取消按钮 -->
      <el-button
        v-if="backupStore.canCancel"
        type="danger"
        @click="handleCancelBackup"
      >
        取消备份
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import {
  House,
  User,
  Setting,
  Menu,
  UserFilled,
  VideoPlay,
  DocumentCopy,
  Link,
  FolderOpened,
  List,
  Film,
  View,
  Operation,
  Promotion,
  Loading,
  DataAnalysis,
  RefreshLeft,
} from '@element-plus/icons-vue'
import { ref, onMounted, onUnmounted, inject } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useBackupStore } from '@/stores/backup'
import { ElMessage, ElMessageBox } from 'element-plus'
import { isMobile as checkIsMobile, onDeviceTypeChange } from '@/utils/deviceUtils'
import { formatDuration } from '@/utils/timeUtils'
import type { AxiosStatic } from 'axios'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const backupStore = useBackupStore()
const http = inject<AxiosStatic>('$http')
const isMobile = ref(false)
const isMenuOpen = ref(false)

// 检测是否为移动设备
const checkMobile = () => {
  isMobile.value = checkIsMobile()
  if (!isMobile.value) {
    isMenuOpen.value = false
  }
}

// 切换菜单显示状态
const toggleMenu = () => {
  isMenuOpen.value = !isMenuOpen.value
}

// 处理菜单选择，移动端选择后关闭菜单
const handleMenuSelect = () => {
  if (isMobile.value) {
    isMenuOpen.value = false
  }
}

// 处理登出
const handleLogout = async () => {
  try {
    await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    authStore.logout()
    ElMessage.success('已退出登录')
    router.push('/login')
  } catch {
    // 用户取消
  }
}

// 获取当前页面标题
const getCurrentPageTitle = (): string => {
  return route.meta.title as string || '首页'
}

// 获取默认展开的子菜单
const getDefaultOpeneds = () => {
  const openeds = []
  if (route.path.startsWith('/settings') || route.path.startsWith('/proxy')) {
    openeds.push('/settings')
  }
  if (route.path.startsWith('/instant-upload') || route.path.startsWith('/media-import')) {
    openeds.push('/instant')
  }
  if (route.path.startsWith('/sync')) {
    openeds.push('/sync')
  }
  if (route.path.startsWith('/scrape')) {
    openeds.push('/scrape')
  }
  if (route.path.includes('upload-queue') || route.path.includes('download-queue')) {
    openeds.push('/upload-queue')
  }
  if (route.path.startsWith('/database/backup')) {
    openeds.push('database')
  }
  return openeds
}

// 获取进度状态样式
const getProgressStatus = () => {
  if (!backupStore.progress?.status) return undefined
  switch (backupStore.progress.status) {
    case 'completed':
      return 'success'
    case 'failed':
    case 'timeout':
      return 'exception'
    case 'cancelled':
      return 'warning'
    default:
      return undefined
  }
}

// 处理取消备份
const handleCancelBackup = async () => {
  try {
    await ElMessageBox.confirm('确定要取消备份任务吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    if (http) {
      await backupStore.cancelBackupTask(http)
    }
  } catch {
    // 用户取消
  }
}

// 组件挂载时加载数据
let removeDeviceTypeListener: (() => void) | null = null

onMounted(() => {
  checkMobile()
  removeDeviceTypeListener = onDeviceTypeChange((newIsMobile) => {
    isMobile.value = newIsMobile
    if (!newIsMobile) {
      isMenuOpen.value = false
    }
  })

  // 检查备份状态
  if (http) {
    backupStore.checkBackupStatus(http)
  }
})

onUnmounted(() => {
  if (removeDeviceTypeListener) {
    removeDeviceTypeListener()
  }
  // 清理轮询定时器
  backupStore.stopProgressPolling()
})
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #2c3e50;
  height: 100vh;
  overflow: hidden;
}

.common-layout {
  height: 100vh;
}

.login-layout {
  height: 100vh;
}

.el-container {
  height: 100%;
  position: relative;
}

.el-aside {
  background-color: rgb(244 244 245);
  transition: transform 0.3s ease;
  z-index: 1000;
  display: flex;
  flex-direction: column;
}

.user-info {
  padding: 20px 15px;
  /* border-bottom: 1px solid #6e7072; */
  /* background-color: #9ea2a5; */
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background-color: #409eff;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.user-details {
  flex: 1;
  min-width: 0;
}

.username {
  /* color: #fff; */
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 4px;
  word-break: break-all;
}

.logout-btn {
  color: #909399 !important;
  font-size: 12px;
  padding: 0 !important;
  height: auto !important;
}

.logout-btn:hover {
  color: #ffd04b !important;
}

.el-menu-vertical {
  background-color: rgb(244 244 245);
  /* border-right: none; */
  flex: 1;
}

.main-content {
  padding: 20px;
  background-color: #ffffff;
  overflow-y: auto;
  transition: margin-left 0.3s ease;
}

/* 移动端样式 */
@media (max-width: 768px) {
  .mobile-aside {
    position: fixed !important;
    top: 0;
    left: 0;
    height: 100vh !important;
    transform: translateX(-100%);
    z-index: 1001;
  }

  .mobile-aside-open {
    transform: translateX(0) !important;
  }

  .mobile-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background-color: rgba(0, 0, 0, 0.5);
    z-index: 1000;
  }

  .main-content {
    padding: 10px;
    margin-left: 0 !important;
    width: 100% !important;
  }

  .mobile-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background-color: #fff;
    padding: 10px 15px;
    margin: -10px -10px 20px -10px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    border-radius: 0 0 8px 8px;
  }

  .mobile-header .left-section {
    display: flex;
    align-items: center;
    flex: 1;
  }

  .menu-toggle {
    margin-right: 15px;
    font-size: 18px;
    color: #545c64 !important;
  }

  .page-title {
    margin: 0;
    font-size: 18px;
    font-weight: 500;
    color: #303133;
    flex: 1;
  }

  .user-dropdown {
    margin-left: 10px;
  }

  .user-btn {
    color: #545c64 !important;
    font-size: 18px;
  }

  .el-menu-item {
    padding: 0 20px !important;
    height: 56px !important;
    line-height: 56px !important;
  }

  .el-menu-item .el-icon {
    margin-right: 15px;
  }
}

/* 桌面端样式 */
@media (min-width: 769px) {
  .mobile-header {
    display: none;
  }
}

/* 平板端适配 */
@media (min-width: 769px) and (max-width: 1024px) {
  .main-content {
    padding: 15px;
  }
}

/* 小屏移动设备优化 */
@media (max-width: 480px) {
  .mobile-aside {
    width: 280px !important;
  }

  .main-content {
    padding: 8px;
  }

  .mobile-header {
    padding: 8px 12px;
    margin: -8px -8px 15px -8px;
  }

  .page-title {
    font-size: 16px;
  }
}

nav {
  padding: 30px;
}

nav a {
  font-weight: bold;
  color: #2c3e50;
}

nav a.router-link-exact-active {
  color: #42b983;
}

/* 滚动条样式优化 */
::-webkit-scrollbar {
  width: 6px;
}

::-webkit-scrollbar-track {
  background: #f1f1f1;
}

::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: #a1a1a1;
}

/* 备份进度弹窗样式 */
.backup-progress-content {
  padding: 20px 0;
}

.progress-step {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 20px;
  font-size: 14px;
  color: #606266;
}

.progress-tables {
  margin-top: 12px;
  font-size: 13px;
  color: #909399;
  text-align: center;
}

.progress-time {
  display: flex;
  justify-content: space-around;
  margin-top: 16px;
  padding: 12px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.progress-time .time-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.progress-time .label {
  font-size: 12px;
  color: #909399;
}

.progress-time .value {
  font-size: 16px;
  font-weight: 600;
  color: #409eff;
}
</style>
