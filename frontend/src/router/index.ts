import axios from 'axios'
import { createRouter } from 'vue-router'
import { createQMediaSyncHashHistory } from './history'
import { useAuthStore } from '@/stores/auth'

import AppLogin from '@/components/AppLogin.vue'
const AppHome = () => import('@/components/AppHome.vue')
const AppUserSettings = () => import('@/components/AppUserSettings.vue')
const AppStrmSettings = () => import('@/components/AppStrmSettings.vue')
const AppEmbySettings = () => import('@/components/AppEmbySettings.vue')
const AppSyncRecords = () => import('@/components/AppSyncRecords.vue')
const AppSyncTaskDetail = () => import('@/components/AppSyncTaskDetail.vue')
const AppSyncDirectories = () => import('@/components/AppSyncDirectories.vue')
const AppSyncDirectoryForm = () => import('@/components/AppSyncDirectoryForm.vue')
const AppCloudAccounts = () => import('@/components/AppCloudAccounts.vue')
const AppThreadSettings = () => import('@/components/AppThreadSettings.vue')
const AppLogSettings = () => import('@/components/AppLogSettings.vue')
const AppUploadQueue = () => import('@/components/AppUploadQueue.vue')
const AppDownloadQueue = () => import('@/components/AppDownloadQueue.vue')
const AppNotificationChannels = () => import('@/components/AppNotificationChannels.vue')
const AppApiKeys = () => import('@/components/AppApiKeys.vue')
const AppLoginSessions = () => import('@/components/user-settings/LoginSessions.vue')

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
    path: '/upload-queue',
    name: 'upload-queue',
    component: AppUploadQueue,
    meta: {
      title: '上传队列',
      requiresAuth: true,
      parent: 'upload-queue',
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
      parent: 'upload-queue',
      icon: 'Download',
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
    path: '/settings/sessions',
    name: 'settings-sessions',
    component: AppLoginSessions,
    meta: {
      title: '登录设备',
      requiresAuth: true,
      parent: 'settings',
      icon: 'Monitor',
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
    path: '/settings/log',
    name: 'settings-log',
    component: AppLogSettings,
    meta: {
      title: '日志设置',
      requiresAuth: true,
      parent: 'settings',
      icon: 'List',
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

router.beforeEach(async (to) => {
  const authStore = useAuthStore()

  await authStore.bootstrapAuth(axios)

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    return {
      name: 'login',
      query: { redirect: to.fullPath },
      replace: true,
    }
  }

  if (to.name === 'login' && authStore.isAuthenticated) {
    return { name: 'home', replace: true }
  }

  return true
})

router.afterEach((to, _from, failure) => {
  if (!failure && to.meta.title) {
    document.title = `${to.meta.title} - QMediaSync`
  }
})

export default router
