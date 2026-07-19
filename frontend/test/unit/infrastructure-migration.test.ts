import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const readSource = (path: string) =>
  readFileSync(fileURLToPath(new URL(path, import.meta.url)), 'utf8')

const deviceTypeConsumerPaths = [
  '../../src/App.vue',
  '../../src/components/AppCloudAccounts.vue',
  '../../src/components/AppDownloadQueue.vue',
  '../../src/components/AppScrapePathForm.vue',
  '../../src/components/AppScrapeRecords.vue',
  '../../src/components/AppSyncDirectoryForm.vue',
  '../../src/components/AppSyncRecords.vue',
  '../../src/components/AppUploadQueue.vue',
]

describe('基础设施迁移', () => {
  it('应用与路由都通过 HTTP 客户端模块使用默认客户端', () => {
    for (const path of ['../../src/App.vue', '../../src/router/index.ts']) {
      const source = readSource(path)

      expect(source).toContain("from '@/http/client'")
      expect(source).not.toContain("from 'axios'")
    }
  })

  it('所有设备类型订阅者统一使用 useDeviceType', () => {
    for (const path of deviceTypeConsumerPaths) {
      const source = readSource(path)

      expect(source).toContain("from '@/composables/useDeviceType'")
      expect(source).toContain('useDeviceType()')
      expect(source).not.toContain('onDeviceTypeChange')
    }
  })

  it('移除未被消费的设备类型订阅', () => {
    for (const path of [
      '../../src/components/AppScrapePathes.vue',
      '../../src/components/AppSyncDirectories.vue',
    ]) {
      const source = readSource(path)

      expect(source).not.toContain('onDeviceTypeChange')
      expect(source).not.toContain('useDeviceType')
    }
  })
})
