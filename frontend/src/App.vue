<template>
  <div v-if="$route.name === 'login'" class="login-layout">
    <router-view />
  </div>

  <div v-else class="common-layout">
    <el-container>
      <div v-if="isMobile && isMenuOpen" class="mobile-overlay" @click="toggleMenu"></div>

      <el-aside
        :width="isMobile ? '250px' : '200px'"
        :class="{ 'mobile-aside': isMobile, 'mobile-aside-open': isMobile && isMenuOpen }"
      >
        <div class="user-info">
          <div class="user-avatar">
            <img src="/qms-icon.png" alt="QMS" class="qms-icon" />
          </div>
          <div class="user-details">
            <div class="username">{{ authStore.user?.username || '用户' }}</div>
            <el-button type="text" size="small" class="logout-btn" @click="handleLogout">
              退出登录
            </el-button>
          </div>
        </div>

        <el-menu
          :default-active="$route.path"
          :default-openeds="defaultOpeneds"
          router
          class="el-menu-vertical"
          @select="handleMenuSelect"
        >
          <template v-for="menu in menuItems" :key="menu.path">
            <el-sub-menu v-if="menu.children && menu.children.length > 0" :index="menu.path">
              <template #title>
                <el-icon>
                  <component :is="menu.iconComponent" :key="menu.meta.icon" />
                </el-icon>
                <span>{{ menu.meta.title }}</span>
              </template>
              <el-menu-item v-for="child in menu.children" :key="child.path" :index="child.path">
                <el-icon>
                  <component :is="child.iconComponent" :key="child.meta.icon" />
                </el-icon>
                <span>{{ child.meta.title }}</span>
              </el-menu-item>
            </el-sub-menu>
            <el-menu-item v-else :index="menu.path">
              <el-icon>
                <component :is="menu.iconComponent" :key="menu.meta.icon" />
              </el-icon>
              <span>{{ menu.meta.title }}</span>
            </el-menu-item>
          </template>
          <el-menu-item index="help" @click="openHelp">
            <el-icon>
              <QuestionFilled />
            </el-icon>
            <span>使用帮助</span>
          </el-menu-item>
        </el-menu>
      </el-aside>

      <el-main class="main-content">
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
      <el-progress
        :percentage="backupStore.progress?.progress || 0"
        :status="getProgressStatus()"
        :stroke-width="20"
      />

      <div v-if="backupStore.progress?.current_step" class="progress-step">
        <el-icon class="is-loading">
          <Loading />
        </el-icon>
        <span>{{ backupStore.progress.current_step }}</span>
      </div>

      <div v-if="backupStore.progress?.total_tables" class="progress-tables">
        <span
          >已处理：{{ backupStore.progress.processed_tables || 0 }} /
          {{ backupStore.progress.total_tables }} 个表</span
        >
      </div>

      <div v-if="backupStore.progress?.elapsed_seconds !== undefined" class="progress-time">
        <div class="time-item">
          <span class="label">已耗时：</span>
          <span class="value">{{ formatDuration(backupStore.progress.elapsed_seconds) }}</span>
        </div>
        <div v-if="backupStore.progress.estimated_seconds" class="time-item">
          <span class="label">预计剩余：</span>
          <span class="value">{{
            formatDuration(
              backupStore.progress.estimated_seconds - backupStore.progress.elapsed_seconds,
            )
          }}</span>
        </div>
      </div>

      <el-alert
        v-if="backupStore.errorRetryCount > 0"
        :title="`网络异常，正在重试 (${backupStore.errorRetryCount}/${3})…`"
        type="warning"
        :closable="false"
        style="margin-top: 16px"
      />
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import {
  User,
  Menu,
  Loading,
  QuestionFilled,
  DataAnalysis,
  DataLine,
  DocumentCopy,
  Download,
  FolderOpened,
  House,
  Key,
  Link,
  List,
  Monitor,
  Operation,
  Promotion,
  RefreshLeft,
  Setting,
  Upload,
  UserFilled,
  VideoPlay,
} from '@element-plus/icons-vue'
import axios from 'axios'
import { ref, onMounted, onUnmounted, computed, markRaw, type Component as VueComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useBackupStore } from '@/stores/backup'
import { ElMessage, ElMessageBox } from 'element-plus'
import { isMobile as checkIsMobile, onDeviceTypeChange } from '@/utils/deviceUtils'
import { formatDuration } from '@/utils/timeUtils'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const backupStore = useBackupStore()
const isMobile = ref(false)
const isMenuOpen = ref(false)

const menuIconMap = {
  DataAnalysis: markRaw(DataAnalysis),
  DataLine: markRaw(DataLine),
  DocumentCopy: markRaw(DocumentCopy),
  Download: markRaw(Download),
  FolderOpened: markRaw(FolderOpened),
  House: markRaw(House),
  Key: markRaw(Key),
  Link: markRaw(Link),
  List: markRaw(List),
  Monitor: markRaw(Monitor),
  Operation: markRaw(Operation),
  Promotion: markRaw(Promotion),
  RefreshLeft: markRaw(RefreshLeft),
  Setting: markRaw(Setting),
  Upload: markRaw(Upload),
  User: markRaw(User),
  UserFilled: markRaw(UserFilled),
  VideoPlay: markRaw(VideoPlay),
} as const

interface MenuItem {
  path: string
  name?: string
  meta: {
    title: string
    icon: string
    showInMenu: boolean
    parent?: string
  }
  iconComponent: VueComponent
  children?: MenuItem[]
}

const menuItems = computed(() => {
  type RouteConfig = {
    path: string
    name?: string
    meta?: {
      title: string
      requiresAuth: boolean
      parent?: string
      icon?: string
      showInMenu?: boolean
    }
    redirect?: string
  }
  const routes = router.options.routes as RouteConfig[]
  const allRoutes = routes.filter((route) => route.meta?.showInMenu)

  const menuMap = new Map<string, MenuItem>()
  const rootMenus: MenuItem[] = []

  allRoutes.forEach((route) => {
    if (route.meta?.parent) {
      if (!menuMap.has(route.meta.parent)) {
        const parentRoute = routes.find((r) => r.name === route.meta?.parent)
        if (parentRoute && parentRoute.meta) {
          const parentMenuItem: MenuItem = {
            path: parentRoute.path,
            name: typeof parentRoute.name === 'string' ? parentRoute.name : undefined,
            meta: {
              title: parentRoute.meta.title,
              icon: parentRoute.meta.icon || 'Setting',
              showInMenu: parentRoute.meta.showInMenu || false,
              parent: parentRoute.meta.parent,
            },
            iconComponent: menuIconMap[parentRoute.meta.icon as keyof typeof menuIconMap] || Setting,
            children: [],
          }
          menuMap.set(route.meta.parent, parentMenuItem)
          rootMenus.push(parentMenuItem)
        }
      }
      if (route.meta.parent && menuMap.has(route.meta.parent) && route.meta) {
        const childMenuItem: MenuItem = {
          path: route.path,
          name: typeof route.name === 'string' ? route.name : undefined,
          meta: {
            title: route.meta.title,
            icon: route.meta.icon || 'Setting',
            showInMenu: route.meta.showInMenu || false,
            parent: route.meta.parent,
          },
          iconComponent: menuIconMap[route.meta.icon as keyof typeof menuIconMap] || Setting,
        }
        menuMap.get(route.meta.parent)?.children?.push(childMenuItem)
      }
    } else if (route.meta) {
      const routeNameKey = typeof route.name === 'string' ? route.name : ''
      if (!menuMap.has(routeNameKey)) {
        const menuItem: MenuItem = {
          path: route.path,
          name: typeof route.name === 'string' ? route.name : undefined,
          meta: {
            title: route.meta.title,
            icon: route.meta.icon || 'Setting',
            showInMenu: route.meta.showInMenu || false,
            parent: route.meta.parent,
          },
          iconComponent: menuIconMap[route.meta.icon as keyof typeof menuIconMap] || Setting,
          children: [],
        }
        menuMap.set(routeNameKey, menuItem)
        rootMenus.push(menuItem)
      }
    }
  })

  return rootMenus
})

const checkMobile = () => {
  isMobile.value = checkIsMobile()
  if (!isMobile.value) {
    isMenuOpen.value = false
  }
}

const toggleMenu = () => {
  isMenuOpen.value = !isMenuOpen.value
}

const handleMenuSelect = () => {
  if (isMobile.value) {
    isMenuOpen.value = false
  }
}

const handleLogout = async () => {
  try {
    await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    await authStore.logoutWithServer(axios)
    ElMessage.success('已退出登录')
    router.replace('/login')
  } catch {
  }
}

const getCurrentPageTitle = (): string => {
  return (route.meta.title as string) || '首页'
}

const defaultOpeneds = computed(() => {
  const openeds: string[] = []
  if (route.path.startsWith('/settings') || route.path.startsWith('/proxy'))
    openeds.push('/settings')
  if (route.path.startsWith('/instant-upload') || route.path.startsWith('/media-import'))
    openeds.push('/instant')
  if (route.path.startsWith('/sync')) openeds.push('/sync')

  if (route.path.includes('upload-queue') || route.path.includes('download-queue')) {
    openeds.push('/upload-queue')
  }
  if (route.path.startsWith('/database/backup')) openeds.push('database')
  return openeds
})

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

const openHelp = () => {
  window.open('https://qmediasync.cn/', '_blank')
}

let removeDeviceTypeListener: (() => void) | null = null

onMounted(() => {
  checkMobile()
  removeDeviceTypeListener = onDeviceTypeChange((newIsMobile) => {
    isMobile.value = newIsMobile
    if (!newIsMobile) {
      isMenuOpen.value = false
    }
  })
})

onUnmounted(() => {
  if (removeDeviceTypeListener) {
    removeDeviceTypeListener()
  }
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
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  overflow: hidden;
}

.qms-icon {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.user-details {
  flex: 1;
  min-width: 0;
}

.username {
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
  flex: 1;
}

.main-content {
  padding: 20px;
  background-color: #ffffff;
  overflow-y: auto;
  transition: margin-left 0.3s ease;
}

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

@media (min-width: 769px) {
  .mobile-header {
    display: none;
  }
}

@media (min-width: 769px) and (max-width: 1024px) {
  .main-content {
    padding: 15px;
  }
}

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
