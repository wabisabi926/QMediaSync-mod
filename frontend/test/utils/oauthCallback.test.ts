import { describe, expect, it } from 'vitest'
import { collectOAuthCallbackParams } from '@/utils/oauthCallback'

describe('collectOAuthCallbackParams', () => {
  it('合并普通 query 和 hash route query 中的授权参数', () => {
    const params = collectOAuthCallbackParams(
      '?access_token=access&refresh_token=refresh&expires_in=3600',
      '#/cloud-accounts?account_id=4&source=115',
    )

    expect(params.get('access_token')).toBe('access')
    expect(params.get('refresh_token')).toBe('refresh')
    expect(params.get('expires_in')).toBe('3600')
    expect(params.get('account_id')).toBe('4')
    expect(params.get('source')).toBe('115')
  })

  it('保留原 hash query 回调格式', () => {
    const params = collectOAuthCallbackParams('', '#/cloud-accounts?account_id=4&token_data=abc')

    expect(params.get('account_id')).toBe('4')
    expect(params.get('token_data')).toBe('abc')
  })
})
