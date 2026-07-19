// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'

import App from '@/App.vue'
import { httpKey } from '@/http/client'
import routes from '@/router'

describe('App 菜单图标', () => {
  it('初始渲染系统设置菜单时立即显示父子菜单图标', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: routes.getRoutes().map((route) => ({
        path: route.path,
        name: route.name,
        meta: route.meta,
        component: { template: '<div />' },
      })),
    })
    await router.push('/settings/emby')
    await router.isReady()

    const wrapper = mount(App, {
      global: {
        plugins: [createPinia(), router, ElementPlus],
        provide: {
          [httpKey]: {},
        },
        stubs: {
          RouterView: { template: '<div />' },
          ElDialog: { template: '<div />' },
        },
      },
    })

    const settingsSubMenu = wrapper
      .findAll('.el-sub-menu')
      .find((subMenu) => subMenu.find('.el-sub-menu__title').text().includes('系统设置'))

    expect(settingsSubMenu).toBeTruthy()
    if (!settingsSubMenu) {
      return
    }
    expect(settingsSubMenu.exists()).toBe(true)
    expect(settingsSubMenu.find('.el-sub-menu__title .el-icon svg').exists()).toBe(true)
    expect(settingsSubMenu.findAll('.el-menu-item .el-icon svg').length).toBeGreaterThan(0)
  })
})
