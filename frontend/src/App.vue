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
      <el-aside
        :width="isMobile ? '250px' : '200px'"
        :class="{ 'mobile-aside': isMobile, 'mobile-aside-open': isMobile && isMenuOpen }"
      >
        <!-- 用户信息 -->
        <div class="user-info">
          <div class="user-avatar">
            <el-icon size="24"><User /></el-icon>
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
          :default-openeds="getDefaultOpeneds()"
          router
          class="el-menu-vertical"
          background-color="#545c64"
          text-color="#fff"
          active-text-color="#ffd04b"
          @select="handleMenuSelect"
        >
          <el-menu-item index="/">
            <el-icon><House /></el-icon>
            <span>首页</span>
          </el-menu-item>
          <el-sub-menu index="/settings">
            <template #title>
              <el-icon><Setting /></el-icon>
              <span>系统设置</span>
            </template>
            <el-menu-item index="/settings">
              <el-icon><Tools /></el-icon>
              <span>115开放平台授权</span>
            </el-menu-item>
            <el-menu-item index="/proxy">
              <el-icon><Link /></el-icon>
              <span>网络代理</span>
            </el-menu-item>
            <el-menu-item index="/settings/strm">
              <el-icon><VideoPlay /></el-icon>
              <span>STRM配置</span>
            </el-menu-item>
            <el-menu-item index="/settings/user">
              <el-icon><UserFilled /></el-icon>
              <span>用户账号设置</span>
            </el-menu-item>
            <el-menu-item index="/settings/telegram">
              <el-icon><ChatLineRound /></el-icon>
              <span>Telegram通知</span>
            </el-menu-item>
          </el-sub-menu>

          <el-sub-menu index="/sync">
            <template #title>
              <el-icon><DocumentCopy /></el-icon>
              <span>同步</span>
            </template>
            <el-menu-item index="/sync-records">
              <el-icon><List /></el-icon>
              <span>同步记录</span>
            </el-menu-item>
            <el-menu-item index="/sync-directories">
              <el-icon><FolderOpened /></el-icon>
              <span>同步目录</span>
            </el-menu-item>
          </el-sub-menu>

          <el-sub-menu index="/instant" v-if="false">
            <template #title>
              <el-icon><Upload /></el-icon>
              <span>秒传</span>
            </template>
            <el-menu-item index="/instant-upload">
              <el-icon><Link /></el-icon>
              <span>URL秒传</span>
            </el-menu-item>
            <el-menu-item index="/media-import">
              <el-icon><FolderOpened /></el-icon>
              <span>媒体库导入</span>
            </el-menu-item>
          </el-sub-menu>
        </el-menu>
      </el-aside>

      <!-- 主内容区 -->
      <el-main class="main-content">
        <!-- 移动端顶部栏 -->
        <div v-if="isMobile" class="mobile-header">
          <div class="left-section">
            <el-button type="text" class="menu-toggle" @click="toggleMenu">
              <el-icon size="20"><Menu /></el-icon>
            </el-button>
            <h2 class="page-title">{{ getCurrentPageTitle() }}</h2>
          </div>
          <el-dropdown class="user-dropdown">
            <el-button type="text" class="user-btn">
              <el-icon><User /></el-icon>
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
</template>

<script setup lang="ts">
import {
  House,
  User,
  Setting,
  Menu,
  Upload,
  Tools,
  ChatLineRound,
  UserFilled,
  VideoPlay,
  DocumentCopy,
  Link,
  FolderOpened,
  List,
} from '@element-plus/icons-vue'
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const isMobile = ref(false)
const isMenuOpen = ref(false)

// 检测是否为移动设备
const checkMobile = () => {
  isMobile.value = window.innerWidth <= 768
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
  const titleMap: Record<string, string> = {
    '/settings': '115开放平台授权',
    '/settings/strm': 'STRM配置',
    '/settings/user': '用户账号设置',
    '/settings/telegram': 'Telegram通知设置',
    '/instant-upload': 'URL秒传',
    '/media-import': '媒体库导入',
    '/proxy': '网络代理',
    '/sync-records': '同步记录',
    '/sync-directories': '同步目录',
  }
  return titleMap[route.path] || '首页'
}

// 获取默认展开的子菜单
const getDefaultOpeneds = () => {
  const openeds = []
  if (route.path.startsWith('/settings')) {
    openeds.push('/settings')
  }
  if (route.path.startsWith('/instant-upload') || route.path.startsWith('/media-import')) {
    openeds.push('/instant')
  }
  if (route.path.startsWith('/sync')) {
    openeds.push('/sync')
  }
  return openeds
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
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
  background-color: #545c64;
  transition: transform 0.3s ease;
  z-index: 1000;
  display: flex;
  flex-direction: column;
}

.user-info {
  padding: 20px 15px;
  border-bottom: 1px solid #434a50;
  background-color: #434a50;
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
  color: #fff;
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
  border-right: none;
  flex: 1;
}

.main-content {
  padding: 20px;
  background-color: #f5f5f5;
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
</style>
