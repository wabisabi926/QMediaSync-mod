import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'

const readSource = (path: string) => readFileSync(resolve(path), 'utf8')

const extractBlock = (source: string, selector: string) => {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`${escapedSelector}\\s*\\{([^}]+)\\}`))
  expect(match).toBeTruthy()
  return match?.[1] ?? ''
}

const extractMediaBlock = (source: string, query: string) => {
  const start = source.indexOf(query)
  expect(start).toBeGreaterThanOrEqual(0)

  const blockStart = source.indexOf('{', start)
  let depth = 0
  for (let index = blockStart; index < source.length; index += 1) {
    if (source[index] === '{') depth += 1
    if (source[index] === '}') depth -= 1
    if (depth === 0) return source.slice(blockStart + 1, index)
  }

  throw new Error(`未找到 ${query} 的完整样式块`)
}

describe('账号弹窗响应式布局', () => {
  const cloudAccountsSource = readSource('src/components/AppCloudAccounts.vue')

  test('新增和编辑账号在移动端使用一致的垂直居中表单布局', () => {
    expect(cloudAccountsSource.match(/class="account-dialog"/g) ?? []).toHaveLength(2)
    expect(cloudAccountsSource).not.toContain('add-account-dialog')
    expect(cloudAccountsSource).not.toContain(':top=')
    expect(cloudAccountsSource.match(/:align-center="isMobile"/g) ?? []).toHaveLength(2)
    expect(
      cloudAccountsSource.match(
        /:width="isMobile \? 'min\(500px, calc\(100vw - 32px\)\)' : '500px'"/g,
      ) ?? [],
    ).toHaveLength(2)
    expect(
      cloudAccountsSource.match(/:label-position="isMobile \? 'top' : 'right'"/g) ?? [],
    ).toHaveLength(2)
    expect(
      cloudAccountsSource.match(/:label-width="isMobile \? 'auto' : '100px'"/g) ?? [],
    ).toHaveLength(2)

    const mobileBlock = extractMediaBlock(cloudAccountsSource, '@media (max-width: 768px)')
    expect(extractBlock(mobileBlock, '.account-dialog')).toContain(
      'max-height: calc(100dvh - 32px);',
    )
    expect(extractBlock(mobileBlock, '.account-dialog')).toContain('display: flex;')
    expect(extractBlock(mobileBlock, '.account-dialog :deep(.el-dialog__body)')).toContain(
      'overflow-y: auto;',
    )
    expect(mobileBlock).toContain('.account-dialog :deep(.el-dialog__footer)')
    expect(mobileBlock).toContain('flex-shrink: 0;')
    expect(extractBlock(cloudAccountsSource, '.dialog-footer')).toContain(
      'justify-content: center;',
    )
  })
})

describe('115 二维码授权长错误保护', () => {
  const authorizationSource = readSource('src/components/cloud-auth/V115AuthorizationDialog.vue')

  test('状态标签在有限宽度内换行并在超长错误时独立滚动', () => {
    const statusBlock = extractBlock(authorizationSource, '.v115-auth-dialog__status')

    for (const declaration of [
      'width: 100%;',
      'min-width: 0;',
      'height: auto;',
      'white-space: normal;',
      'overflow-wrap: anywhere;',
      'max-height: 96px;',
      'overflow-y: auto;',
      'overscroll-behavior: contain;',
    ]) {
      expect(statusBlock).toContain(declaration)
    }
  })

  test('短状态居中并使用语义色，失败原文保持左对齐', () => {
    expect(authorizationSource).toContain(
      `if (status.value === 'waiting') return 'primary' as const`,
    )
    expect(authorizationSource).toContain(
      `:class="{ 'v115-auth-dialog__status--failed': status === 'failed' }"`,
    )

    const statusBlock = extractBlock(authorizationSource, '.v115-auth-dialog__status')
    for (const declaration of [
      'align-items: center;',
      'justify-content: center;',
      'text-align: center;',
    ]) {
      expect(statusBlock).toContain(declaration)
    }

    const failedBlock = extractBlock(authorizationSource, '.v115-auth-dialog__status--failed')
    for (const declaration of [
      'align-items: flex-start;',
      'justify-content: flex-start;',
      'text-align: left;',
    ]) {
      expect(failedBlock).toContain(declaration)
    }
  })
})

describe('首页操作区样式隔离', () => {
  const homeSource = readSource('src/components/AppHome.vue')

  test('运行日志操作区不再使用全局 header-actions 类', () => {
    expect(homeSource).toContain('class="home-header__actions"')
    expect(homeSource).not.toContain('class="header-actions"')

    const actionsBlock = extractBlock(homeSource, '.home-header__actions')
    expect(actionsBlock).toContain('display: flex;')
    expect(actionsBlock).toContain('align-items: center;')
    expect(actionsBlock).toContain('gap: 12px;')
  })
})
