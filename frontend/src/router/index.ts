import { createRouter } from 'vue-router'
import { createQMediaSyncHashHistory } from './history'
import { useAuthStore } from '@/stores/auth'
import axios from 'axios'

const AppHome = () => import('@/components/AppHome.vue')
const AppLogin = () => import('@/components/AppLogin.vue')
const AppUserSettings = () => import('@/components/AppUserSettings.vue')
const AppStrmSettings = () => import('@/components/AppStrmSettings.vue')
const AppEmbySettings = () => import('@/components/AppEmbySettings.vue')
const AppSyncRecords = () => import('@/components/AppSyncRecords.vue')
const AppSyncTaskDetail = () => import('@/components/AppSyncTaskDetail.vue')
const AppSyncDirectories = () => import('@/components/AppSyncDirectories.vue')
const AppSyncDirectoryForm = () => import('@/components/AppSyncDirectoryForm.vue')
const AppCloudAccounts = () => import('@/components/AppCloudAccounts.vue')
const AppThreadSettings = () => import('@/components/AppThreadSettings.vue')
const AppTmdbSettings = () => import('@/components/AppTmdbSettings.vue')
const AppAiSettings = () => import('@/components/AppAiSettings.vue')
const AppCategoryStrategy = () => import('@/components/AppCategoryStrategy.vue')
const AppScrapePathes = () => import('@/components/AppScrapePathes.vue')
const AppScrapePathForm = () => import('@/components/AppScrapePathForm.vue')
const AppScrapeRecords = () => import('@/components/AppScrapeRecords.vue')
const AppUploadQueue = () => import('@/components/AppUploadQueue.vue')
const AppDownloadQueue = () => import('@/components/AppDownloadQueue.vue')
const AppNotificationChannels = () => import('@/components/AppNotificationChannels.vue')
const AppApiKeys = () => import('@/components/AppApiKeys.vue')
const AppFileManager = () => import('@/components/AppFileManager.vue')
const AppUpdate = () => import('@/components/AppUpdate.vue')

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
    path: '/sync-directory/add',
    name: 'sync-directory-add',
    component: AppSyncDirectoryForm,
    meta: {
      title: '添加同步目录',
      requiresAuth: true,
      parent: 'sync',
      showInMenu: false,
    },
  },
  {
    path: '/sync-directory/edit/:id',
    name: 'sync-directory-edit',
    component: AppSyncDirectoryForm,
    meta: {
      title: '编辑同步目录',
      requiresAuth: true,
      parent: 'sync',
      showInMenu: false,
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
    path: '/scrape-path/add',
    name: 'scrape-path-add',
    component: AppScrapePathForm,
    meta: {
      title: '添加刮削目录',
      requiresAuth: true,
      parent: 'scrape',
      showInMenu: false,
    },
  },
  {
    path: '/scrape-path/edit/:id',
    name: 'scrape-path-edit',
    component: AppScrapePathForm,
    meta: {
      title: '编辑刮削目录',
      requiresAuth: true,
      parent: 'scrape',
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
      title: '刮削设置',
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
    path: '/transfer',
    name: 'transfer',
    redirect: '/upload-queue',
    meta: {
      title: '上传下载',
      requiresAuth: true,
      icon: 'Download',
      showInMenu: true,
    },
  },
  {
    path: '/upload-queue',
    name: 'upload-queue',
    component: AppUploadQueue,
    meta: {
      title: '上传队列',
      requiresAuth: true,
      parent: 'transfer',
      icon: 'Upload',
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
      parent: 'transfer',
      icon: 'Download',
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
    path: '/settings/update',
    name: 'settings-update',
    component: AppUpdate,
    meta: {
      title: '版本更新',
      requiresAuth: true,
      parent: 'settings',
      icon: 'Upload',
      showInMenu: true,
    },
  },
  {
    path: '/settings/database-repair',
    name: 'settings-database-repair',
    component: () => import('@/components/AppDatabaseRepair.vue'),
    meta: {
      title: '数据库修复',
      requiresAuth: true,
      parent: 'settings',
      icon: 'DataLine',
      showInMenu: true,
    },
  },
]

const router = createRouter({
  history: createQMediaSyncHashHistory(),
  routes,
})

// 路由守卫
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()

  // 如果要访问的页面需要认证
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    const restored = await authStore.restoreSession(axios)
    if (!restored) {
      next({
        name: 'login',
        query: { redirect: to.fullPath },
      })
      return
    }
  }

  if (to.meta.requiresAuth) {
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
  if (to.name === 'login') {
    await authStore.restoreSession(axios)
    if (authStore.isAuthenticated) {
      next({ name: 'home' })
      return
    }
  }

  // 设置页面标题
  if (to.meta.title) {
    document.title = `${to.meta.title} - QMediaSync`
  }

  next()
})

export default router
