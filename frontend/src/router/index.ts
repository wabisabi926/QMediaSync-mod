import AppHome from '@/components/AppHome.vue'
import AppSettings from '@/components/AppSettings.vue'
import AppLogin from '@/components/AppLogin.vue'
import AppCookieCloud from '@/components/AppCookieCloud.vue'
import AppTelegramSettings from '@/components/AppTelegramSettings.vue'
import AppUserSettings from '@/components/AppUserSettings.vue'
import AppStrmSettings from '@/components/AppStrmSettings.vue'
import AppSyncRecords from '@/components/AppSyncRecords.vue'
import AppSyncTaskDetail from '@/components/AppSyncTaskDetail.vue'
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
    path: '/settings',
    name: 'settings',
    component: AppSettings,
    meta: {
      title: '115开放平台授权',
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
      title: '任务详情',
      requiresAuth: true,
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
    path: '/settings/cookiecloud',
    name: 'settings-cookiecloud',
    component: AppCookieCloud,
    meta: {
      title: 'CookieCloud设置',
      requiresAuth: true,
      parent: 'settings',
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

  next()
})

export default router
