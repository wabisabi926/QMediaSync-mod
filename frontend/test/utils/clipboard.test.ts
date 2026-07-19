// @vitest-environment happy-dom

import { afterEach, describe, expect, it, vi } from 'vitest'
import { copyText } from '@/utils/clipboard'

describe('copyText', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    document.body.innerHTML = ''
  })

  it('优先使用 Clipboard API', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', {
      clipboard: { writeText },
    })

    const copied = await copyText('qms_secret')

    expect(copied).toBe(true)
    expect(writeText).toHaveBeenCalledWith('qms_secret')
  })

  it('Clipboard API 不存在时使用选中文本降级复制', async () => {
    vi.stubGlobal('navigator', {})
    Object.defineProperty(document, 'execCommand', {
      configurable: true,
      value: vi.fn().mockReturnValue(true),
    })

    const copied = await copyText('qms_secret')

    expect(copied).toBe(true)
    expect(document.execCommand).toHaveBeenCalledWith('copy')
    expect(document.querySelector('textarea')).toBeNull()
  })
})
