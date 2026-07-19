import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __dirname = dirname(fileURLToPath(import.meta.url))

describe('AppSyncTaskDetail stream mode', () => {
  it('uses the dedicated stream source and task log panel', () => {
    const source = readFileSync(
      resolve(__dirname, '../../src/components/AppSyncTaskDetail.vue'),
      'utf-8',
    )

    expect(source).toContain('useSyncTaskStream')
    expect(source).toContain('SyncTaskLogPanel')
    expect(source).not.toContain('AppLogViewer')
    expect(source).not.toContain('loadTaskInfo')
  })
})
