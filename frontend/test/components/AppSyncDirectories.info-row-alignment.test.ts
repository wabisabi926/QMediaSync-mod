import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'

const source = readFileSync(resolve('src/components/AppSyncDirectories.vue'), 'utf8')

function extractBlock(css: string, selector: string) {
  const pattern = new RegExp(`${selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*\\{([^}]+)\\}`)
  const match = css.match(pattern)
  expect(match).toBeTruthy()
  return match?.[1] ?? ''
}

function extractMediaBlock(css: string, query: string) {
  const start = css.indexOf(query)
  expect(start).toBeGreaterThanOrEqual(0)

  const blockStart = css.indexOf('{', start)
  expect(blockStart).toBeGreaterThanOrEqual(0)

  let depth = 0
  for (let index = blockStart; index < css.length; index += 1) {
    const char = css[index]
    if (char === '{') {
      depth += 1
    }
    if (char === '}') {
      depth -= 1
      if (depth === 0) {
        return css.slice(blockStart + 1, index)
      }
    }
  }

  throw new Error(`未找到 ${query} 的完整样式块`)
}

describe('AppSyncDirectories 信息行对齐', () => {
  test('桌面端信息行内容按 32px 图标高度垂直居中', () => {
    expect(extractBlock(source, '.info-row')).toContain('align-items: center;')
    expect(extractBlock(source, '.info-content')).toContain('min-height: 32px;')
  })

  test('移动端信息行内容按 28px 图标高度垂直居中', () => {
    const mobileBlock = extractMediaBlock(source, '@media (max-width: 768px)')

    expect(extractBlock(mobileBlock, '.info-content')).toContain('min-height: 28px;')
  })
})
