import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('App 菜单图标结构', () => {
  it('菜单数据预解析图标组件，避免初始展开时动态函数调用导致图标延迟显示', () => {
    const source = readFileSync(resolve('src/App.vue'), 'utf8')

    expect(source).toContain('iconComponent')
    expect(source).toContain(':is="menu.iconComponent"')
    expect(source).toContain(':is="child.iconComponent"')
    expect(source).not.toContain(':is="getMenuIcon(menu.meta.icon)"')
    expect(source).not.toContain(':is="getMenuIcon(child.meta.icon)"')
  })
})
