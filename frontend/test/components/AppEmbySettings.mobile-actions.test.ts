import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('AppEmbySettings 移动端操作和媒体库图标', () => {
  it('使用响应式操作栏并定义媒体库图标背景', () => {
    const source = readFileSync(resolve('src/components/AppEmbySettings.vue'), 'utf8')

    expect(source).toContain(
      "import ResponsiveActionBar from '@/components/common/ResponsiveActionBar.vue'",
    )
    expect(source).toContain('<ResponsiveActionBar')
    expect(source).toContain('.library-icon')
    expect(source).toMatch(/\.library-icon\s*\{[\s\S]*background:/)
  })
})
