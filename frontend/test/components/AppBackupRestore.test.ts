// @vitest-environment happy-dom
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

describe('AppBackupRestore', () => {
  it('上传恢复页面和确认弹窗都会提示恢复后重启服务', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/components/AppBackupRestore.vue'),
      'utf8',
    )

    expect(source).toContain('提示：恢复成功后请重启服务，让所有数据和配置生效')
    expect(source).toContain('恢复成功后请重启服务让所有数据和配置生效')
    expect(source).toContain('dangerouslyUseHTMLString: true')
  })
})
