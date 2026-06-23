import { describe, expect, it } from 'vitest'

import { getV115AppInfoRows, isCustomV115App } from './cloudAccountUtils'

describe('cloudAccountUtils', () => {
  it('识别自定义 115 应用', () => {
    expect(isCustomV115App({ source_type: '115', app_id: 'custom-app-id' })).toBe(true)
    expect(isCustomV115App({ source_type: '115', app_id_name: 'QMediaSync' })).toBe(false)
    expect(isCustomV115App({ source_type: 'openlist', app_id: 'custom-app-id' })).toBe(false)
  })

  it('数字内置 APPID 不应按自定义应用处理', () => {
    expect(
      isCustomV115App({
        source_type: '115',
        app_id: '100197849',
        app_id_name: 'QMediaSync',
        auth_source_type: 'built_in_appid',
        auth_provider: 'official_pkce',
      }),
    ).toBe(false)
  })

  it('自定义 115 应用展示应用名和 APPID', () => {
    expect(
      getV115AppInfoRows({
        source_type: '115',
        app_id: 'custom-app-id',
        app_id_name: '家庭影音',
      }),
    ).toEqual([
      { label: '应用名', value: '家庭影音' },
      { label: 'APPID', value: 'custom-app-id' },
    ])
  })

  it('自定义应用名为空时使用自定义占位', () => {
    expect(
      getV115AppInfoRows({
        source_type: '115',
        app_id: 'custom-app-id',
        app_id_name: '',
      }),
    ).toEqual([
      { label: '应用名', value: '自定义' },
      { label: 'APPID', value: 'custom-app-id' },
    ])
  })
})
