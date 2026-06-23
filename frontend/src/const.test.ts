import { describe, expect, it } from 'vitest'
import { normalizeServerURL } from './const'

describe('normalizeServerURL', () => {
  it('忽略字符串 undefined 并使用回退地址', () => {
    expect(normalizeServerURL('undefined', '/api')).toBe('/api')
  })

  it('保留有效地址并去掉尾部斜杠', () => {
    expect(normalizeServerURL('http://localhost:12333/api/', '/api')).toBe(
      'http://localhost:12333/api',
    )
  })
})
