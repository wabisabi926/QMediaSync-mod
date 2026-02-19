import AppHome from '@/components/AppHome.vue'
import AppLogin from '@/components/AppLogin.vue'
import AppUserSettings from '@/components/AppUserSettings.vue'
import AppStrmSettings from '@/components/AppStrmSettings.vue'
import AppEmbySettings from '@/components/AppEmbySettings.vue'
import AppSyncRecords from '@/components/AppSyncRecords.vue'
import AppSyncTaskDetail from '@/components/AppSyncTaskDetail.vue'
import AppSyncDirectories from '@/components/AppSyncDirectories.vue'
import AppCloudAccounts from '@/components/AppCloudAccounts.vue'
import AppThreadSettings from '@/components/AppThreadSettings.vue'
import AppTmdbSettings from '@/components/AppTmdbSettings.vue'
import AppAiSettings from '@/components/AppAiSettings.vue'
import AppCategoryStrategy from '@/components/AppCategoryStrategy.vue'
import AppScrapePathes from '@/components/AppScrapePathes.vue'
import AppScrapeRecords from '@/components/AppScrapeRecords.vue'
import AppUploadQueue from '@/components/AppUploadQueue.vue'
import AppDownloadQueue from '@/components/AppDownloadQueue.vue'
import AppScrapePathAdd from '@/components/AppScrapePathAdd.vue'
import AppNotificationChannels from '@/components/AppNotificationChannels.vue'
import AppApiKeys from '@/components/AppApiKeys.vue'
import AppFileManager from '@/components/AppFileManager.vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

// 定义路由元信息类型
declare module 'vue-router' {
  interface RouteMeta {
    title: string
    requiresAuth: boolean
    parent?: string
    icon?: string
    showInMenu?: boolean
  }
}

const routes = [
  {
    path: '/login',
    name: 'login',
    component: AppLogin,
    meta: {
      title: '登录',
      requiresAuth: false,
      showInMenu: false,
    },
  },
  {
    path: '/',
    name: 'home',
    component: AppHome,
    meta: {
      title: '首页',
      requiresAuth: true,
      icon: 'House',
      showInMenu: true,
    },
  },
  {
    path: '/accounts',
    name: 'accounts',
    component: AppCloudAccounts,
    meta: {
      title: '网盘账号',
      requiresAuth: true,
      icon: 'User',
      showInMenu: true,
    },
  },
  {
    path: '/settings',
    name: 'settings',
    component: AppUserSettings,
    meta: {
      title: '系统设置',
      requiresAuth: true,
      icon: 'Setting',
      showInMenu: true,
    },
  },
  {
    path: '/settings/user',
    name: 'settings-user',
    component: AppUserSettings,
    meta: {
      title: '用户管理',
      requiresAuth: true,
      parent: 'settings',
      icon: 'UserFilled',
      showInMenu: true,
    },
  },
  {
    path: '/settings/api-keys',
    name: 'settings-api-keys',
    component: AppApiKeys,
    meta: {
      title: 'API Key',
      requiresAuth: true,
      parent: 'settings',
      icon: 'Key',
      showInMenu: true,
    },
  },
  {
    path: '/settings/notification',
    name: 'settings-notification',
    component: AppNotificationChannels,
    meta: {
      title: '通知管理',
      requiresAuth: true,
      parent: 'settings',
      icon: 'Promotion',
      showInMenu: true,
    },
  },
  {
    path: '/settings/emby',
    name: 'settings-emby',
    component: AppEmbySettings,
    meta: {
      title: 'Emby',
      requiresAuth: true,
      parent: 'settings',
      icon: 'VideoPlay',
      showInMenu: true,
    },
  },
  {
    path: '/settings/threads',
    name: 'settings-threads',
    component: AppThreadSettings,
    meta: {
      title: '接口速率',
      requiresAuth: true,
      parent: 'settings',
      icon: 'Operation',
      showInMenu: true,
    },
  },
  {
    path: '/proxy',
    name: 'proxy',
    component: () => import('@/components/AppProxySettings.vue'),
    meta: {
      title: '网络代理',
      requiresAuth: true,
      parent: 'settings',
      icon: 'Link',
      showInMenu: true,
    },
  },
  {
    path: '/upload-queue',
    name: 'upload-queue',
    component: AppUploadQueue,
    meta: {
      title: '上传下载',
      requiresAuth: true,
      icon: 'Download',
      showInMenu: true,
    },
  },
  {
    path: '/download-queue',
    name: 'download-queue',
    component: AppDownloadQueue,
    meta: {
      title: '下载队列',
      requiresAuth: true,
      parent: 'upload-queue',
      icon: 'Download',
      showInMenu: true,
    },
  },
  {
    path: '/sync',
    name: 'sync',
    redirect: '/sync-directories',
    meta: {
      title: 'STRM同步',
      requiresAuth: true,
      icon: 'DocumentCopy',
      showInMenu: true,
    },
  },
  {
    path: '/sync-directories',
    name: 'sync-directories',
    component: AppSyncDirectories,
    meta: {
      title: 'STRM同步目录',
      requiresAuth: true,
      parent: 'sync',
      icon: 'FolderOpened',
      showInMenu: true,
    },
  },
  {
    path: '/sync-records',
    name: 'sync-records',
    component: AppSyncRecords,
    meta: {
      title: 'STRM同步记录',
      requiresAuth: true,
      parent: 'sync',
      icon: 'List',
      showInMenu: true,
    },
  },
    {
    path: '/settings/strm',
    name: 'settings-strm',
    component: AppStrmSettings,
    meta: {
      title: 'STRM设置',
      requiresAuth: true,
      parent: 'sync',
      icon: 'Setting',
      showInMenu: true,
    },
  },
  {
    path: '/sync-records/:id',
    name: 'sync-task-detail',
    component: AppSyncTaskDetail,
    meta: {
      title: '同步任务详情',
      requiresAuth: true,
      showInMenu: false,
    },
  },
  {
    path: '/scrape',
    name: 'scrape',
    redirect: '/scrape-pathes',
    meta: {
      title: '刮削 & 整理',
      requiresAuth: true,
      icon: 'Film',
      showInMenu: true,
    },
  },
  {
    path: '/scrape-pathes',
    name: 'scrape-pathes',
    component: AppScrapePathes,
    meta: {
      title: '刮削目录',
      requiresAuth: true,
      parent: 'scrape',
      icon: 'FolderOpened',
      showInMenu: true,
    },
  },
  {
    path: '/scrape-add',
    name: 'scrape-add',
    component: AppScrapePathAdd,
    meta: {
      title: '添加刮削目录',
      requiresAuth: true,
      showInMenu: false,
    },
  },
  {
    path: '/scrape-records',
    name: 'scrape-records',
    component: AppScrapeRecords,
    meta: {
      title: '刮削记录',
      requiresAuth: true,
      parent: 'scrape',
      icon: 'List',
      showInMenu: true,
    },
  },
  {
    path: '/settings/tmdb',
    name: 'settings-tmdb',
    component: AppTmdbSettings,
    meta: {
      title: 'TMDB设置',
      requiresAuth: true,
      parent: 'scrape',
      icon: 'Film',
      showInMenu: true,
    },
  },
  {
    path: '/settings/ai',
    name: 'settings-ai',
    component: AppAiSettings,
    meta: {
      title: 'AI识别设置',
      requiresAuth: true,
      parent: 'scrape',
      icon: 'View',
      showInMenu: true,
    },
  },
  {
    path: '/settings/category-strategy',
    name: 'settings-category-strategy',
    component: AppCategoryStrategy,
    meta: {
      title: '二级分类设置',
      requiresAuth: true,
      parent: 'scrape',
      icon: 'Operation',
      showInMenu: true,
    },
  },
  {
    path: '/file-manager',
    name: 'file-manager',
    component: AppFileManager,
    meta: {
      title: '网盘文件管理',
      requiresAuth: true,
      icon: 'Folder',
      showInMenu: true,
    },
  },
    {
    path: '/database',
    name: 'database',
    component: () => import('@/components/AppBackupSettings.vue'),
    meta: {
      title: '备份恢复',
      requiresAuth: true,
      icon: 'DataAnalysis',
      showInMenu: true,
    },
  },
  {
    path: '/database/backup/settings',
    name: 'database-backup-settings',
    component: () => import('@/components/AppBackupSettings.vue'),
    meta: {
      title: '备份设置',
      requiresAuth: true,
      parent: 'database',
      showInMenu: true,
    },
  },
  {
    path: '/database/backup/records',
    name: 'database-backup-records',
    component: () => import('@/components/AppBackupRecords.vue'),
    meta: {
      title: '备份记录',
      requiresAuth: true,
      parent: 'database',
      icon: 'List',
      showInMenu: true,
    },
  },
  {
    path: '/database/backup/restore',
    name: 'database-backup-restore',
    component: () => import('@/components/AppBackupRestore.vue'),
    meta: {
      title: '备份恢复',
      requiresAuth: true,
      parent: 'database',
      icon: 'RefreshLeft',
      showInMenu: true,
    },
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// 路由守卫
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()

  // 初始化认证状态
  authStore.initAuth()

  // 如果要访问的页面需要认证
  if (to.meta.requiresAuth) {
    if (!authStore.isAuthenticated) {
      // 未登录，跳转到登录页面
      next({
        name: 'login',
        query: { redirect: to.fullPath },
      })
      return
    }

    // 验证token有效性
    const isValid = await authStore.checkTokenValidity()
    if (!isValid) {
      next({
        name: 'login',
        query: { redirect: to.fullPath },
      })
      return
    }
  }

  // 如果已登录用户访问登录页面，重定向到首页
  if (to.name === 'login' && authStore.isAuthenticated) {
    next({ name: 'home' })
    return
  }

  // 设置页面标题
  if (to.meta.title) {
    document.title = `${to.meta.title} - QMediaSync`
  }

  next()
})

export default router
