// @vitest-environment happy-dom
import { describe, expect, it } from 'vitest'
import { getCSRFTokenFromCookie, shouldAttachCSRFToken } from '../../src/utils/csrf'

describe('axios csrf helpers', () => {
  it('unsafe method 需要 CSRF', () => {
    expect(shouldAttachCSRFToken('post')).toBe(true)
    expect(shouldAttachCSRFToken('PUT')).toBe(true)
    expect(shouldAttachCSRFToken('delete')).toBe(true)
    expect(shouldAttachCSRFToken('get')).toBe(false)
  })

  it('从 csrf_token cookie 读取 token', () => {
    document.cookie = 'csrf_token=abc123; path=/'
    expect(getCSRFTokenFromCookie()).toBe('abc123')
  })
})
