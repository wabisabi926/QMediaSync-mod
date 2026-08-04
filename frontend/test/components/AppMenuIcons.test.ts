// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { Setting, UserFilled, Key, Monitor } from '@element-plus/icons-vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'

import { httpKey } from '@/http/client'

const menuItems = [
  {
    path: '/settings',
    meta: { title: '系统设置', icon: 'Setting' },
    iconComponent: Setting,
    children: [
      { path: '/settings/user', meta: { title: '用户管理', icon: 'UserFilled' }, iconComponent: UserFilled },
      { path: '/settings/api-keys', meta: { title: 'API Key', icon: 'Key' }, iconComponent: Key },
      { path: '/settings/sessions', meta: { title: '登录设备', icon: 'Monitor' }, iconComponent: Monitor },
    ],
  },
]

const menuTemplate = {
  template: `
    <el-menu :default-active="'/settings/emby'">
      <template v-for="menu in items" :key="menu.path">
        <el-sub-menu v-if="menu.children && menu.children.length > 0" :index="menu.path">
          <template #title>
            <el-icon><component :is="menu.iconComponent" /></el-icon>
            <span>{{ menu.meta.title }}</span>
          </template>
          <el-menu-item v-for="child in menu.children" :key="child.path" :index="child.path">
            <el-icon><component :is="child.iconComponent" /></el-icon>
            <span>{{ child.meta.title }}</span>
          </el-menu-item>
        </el-sub-menu>
        <el-menu-item v-else :index="menu.path">
          <el-icon><component :is="menu.iconComponent" /></el-icon>
          <span>{{ menu.meta.title }}</span>
        </el-menu-item>
      </template>
    </el-menu>
  `,
  props: {
    items: { type: Array, required: true },
  },
}

describe('App 菜单图标', () => {
  it('初始渲染系统设置菜单时立即显示父子菜单图标', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/settings/emby', component: { template: '<div />' } }],
    })

    const wrapper = mount(menuTemplate, {
      props: { items: menuItems },
      global: {
        plugins: [createPinia(), router, ElementPlus],
        provide: {
          [httpKey]: {},
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
