import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const readSource = (path) => readFileSync(resolve(frontendRoot, path), 'utf8')

const expectRecordTableContract = (source, pageKey, rowKey, columns, actions) => {
  expect(source).toMatch(new RegExp(`getPageState\\s*\\(\\s*['"]${pageKey}['"]`))
  expect(source).toContain('ResponsiveRecordTable')
  expect(source).toMatch(new RegExp(`:columns\\s*=\\s*["']${columns}["']`))
  expect(source).toMatch(new RegExp(`:actions\\s*=\\s*["']${actions}["']`))
  expect(source).toMatch(new RegExp(`:row-key\\s*=\\s*["']${rowKey}["']`))
  expect(source).toMatch(/:loading\s*=\s*["']initialLoading\s*\|\|\s*queryLoading["']/)
  expect(source).toMatch(/:expanded-row-keys\s*=\s*["']pageState\.expandedRowKeys["']/)
  expect(source).toMatch(/@expand-change\s*=\s*["']handleExpandChange["']/)
  expect(source).toContain('createActiveRequestGate')
}

describe('记录与文件页的公共状态契约', () => {
  it('同步记录通过响应式记录表公开稳定行键、展开状态和失败详情', () => {
    const source = readSource('src/components/AppSyncRecords.vue')

    expectRecordTableContract(
      source,
      'sync-records',
      'getSyncRecordRowKey',
      'syncRecordColumns',
      'syncRecordActions',
    )
    expect(source).toMatch(/const\s+getSyncRecordRowKey\s*=\s*\(row:\s*SyncRecord\)\s*=>\s*row\.id/)
    expect(source).toMatch(
      /key:\s*['"]fail_reason['"][\s\S]*?value:\s*\(row\)\s*=>\s*row\.fail_reason\s*\|\|\s*['"]-['"]/,
    )
    expect(source).toMatch(
      /setExpandedRowKeys\s*\(\s*['"]sync-records['"][\s\S]*?String\s*\(\s*item\.id\s*\)/,
    )
  })
})
