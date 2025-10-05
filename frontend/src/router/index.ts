import AppHome from '@/components/AppHome.vue'
import AppLogin from '@/components/AppLogin.vue'
import AppTelegramSettings from '@/components/AppTelegramSettings.vue'
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
import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: AppLogin,
    meta: {
      title: '登录',
      requiresAuth: false,
    },
  },
  {
    path: '/',
    name: 'home',
    component: AppHome,
    meta: {
      title: '首页',
      requiresAuth: true,
    },
  },
  {
    path: '/accounts',
    name: 'accounts',
    component: AppCloudAccounts,
    meta: {
      title: '网盘账号管理',
      requiresAuth: true,
    },
  },
  {
    path: '/settings/strm',
    name: 'settings-strm',
    component: AppStrmSettings,
    meta: {
      title: 'STRM配置',
      requiresAuth: true,
      parent: 'settings',
    },
  },
  {
    path: '/settings/user',
    name: 'settings-user',
    component: AppUserSettings,
    meta: {
      title: '用户账号设置',
      requiresAuth: true,
      parent: 'settings',
    },
  },
  {
    path: '/sync-records',
    name: 'sync-records',
    component: AppSyncRecords,
    meta: {
      title: '同步记录',
      requiresAuth: true,
    },
  },
  {
    path: '/sync-records/:id',
    name: 'sync-task-detail',
    component: AppSyncTaskDetail,
    meta: {
      title: '同步任务详情',
      requiresAuth: true,
    },
  },
  {
    path: '/sync-directories',
    name: 'sync-directories',
    component: AppSyncDirectories,
    meta: {
      title: '同步目录',
      requiresAuth: true,
    },
  },
  {
    path: '/scrape-pathes',
    name: 'scrape-pathes',
    component: AppScrapePathes,
    meta: {
      title: '刮削目录',
      requiresAuth: true,
    },
  },
  {
    path: '/settings/telegram',
    name: 'settings-telegram',
    component: AppTelegramSettings,
    meta: {
      title: 'Telegram通知设置',
      requiresAuth: true,
      parent: 'settings',
    },
  },
  {
    path: '/settings/emby',
    name: 'settings-emby',
    component: AppEmbySettings,
    meta: {
      title: 'Emby配置',
      requiresAuth: true,
      parent: 'settings',
    },
  },
  {
    path: '/settings/threads',
    name: 'settings-threads',
    component: AppThreadSettings,
    meta: {
      title: '115并发线程设置',
      requiresAuth: true,
      parent: 'settings',
    },
  },
  {
    path: '/settings/tmdb',
    name: 'settings-tmdb',
    component: AppTmdbSettings,
    meta: {
      title: 'TMDB设置',
      requiresAuth: true,
      parent: 'settings',
    },
  },
    {
      path: '/settings/ai',
      name: 'settings-ai',
      component: AppAiSettings,
      meta: {
        title: 'AI识别设置',
        requiresAuth: true,
        parent: 'settings',
      },
    },
  {
    path: '/settings/category-strategy',
    name: 'settings-category-strategy',
    component: AppCategoryStrategy,
    meta: {
      title: '二级分类策略',
      requiresAuth: true,
      parent: 'settings',
    },
  },
  {
    path: '/instant-upload',
    name: 'instant-upload',
    component: () => import('@/components/AppInstantUpload.vue'),
    meta: {
      title: 'URL秒传',
      requiresAuth: true,
    },
  },
  {
    path: '/media-import',
    name: 'media-import',
    component: () => import('@/components/AppMediaImport.vue'),
    meta: {
      title: '媒体库导入',
      requiresAuth: true,
    },
  },
  {
    path: '/proxy',
    name: 'proxy',
    component: () => import('@/components/AppProxySettings.vue'),
    meta: {
      title: '网络代理',
      requiresAuth: true,
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
