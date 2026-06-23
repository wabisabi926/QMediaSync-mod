import { describe, expect, it } from 'vitest'

import {
  buildV115CreatePayload,
  getV115AuthAction,
  type V115AccountAuthInfo,
} from './v115AuthSources'

describe('v115AuthSources', () => {
  it('扫码授权内置 APPID 创建 payload 使用数字 APPID', () => {
    expect(
      buildV115CreatePayload({
        authMode: 'qr',
        selectedQrApp: { appId: '100197849', appName: 'QMediaSync' },
        selectedWebProvider: 'qmediasync',
        customAppId: '',
        customAppName: '',
      }),
    ).toEqual({
      auth_source_type: 'built_in_appid',
      auth_provider: 'official_pkce',
      app_id: '100197849',
      app_id_name: 'QMediaSync',
    })
  })

  it('网页授权创建 payload 使用 provider 来源', () => {
    expect(
      buildV115CreatePayload({
        authMode: 'oauth',
        selectedQrApp: { appId: '100197849', appName: 'QMediaSync' },
        selectedWebProvider: 'moviepilot',
        customAppId: '',
        customAppName: '',
      }),
    ).toEqual({
      auth_source_type: 'third_party_service',
      auth_provider: 'moviepilot',
      app_id: '100197847',
      app_id_name: 'MoviePilot-115',
    })
  })

  it('授权动作按 auth_provider 分发', () => {
    const qrAccount: V115AccountAuthInfo = {
      source_type: '115',
      auth_source_type: 'built_in_appid',
      auth_provider: 'official_pkce',
      app_id: '100197849',
    }
    const oauthAccount: V115AccountAuthInfo = {
      source_type: '115',
      auth_source_type: 'built_in_relay',
      auth_provider: 'qmediasync',
      app_id_name: 'QMediaSync',
    }
    const thirdPartyAccount: V115AccountAuthInfo = {
      source_type: '115',
      auth_source_type: 'third_party_service',
      auth_provider: 'moviepilot',
      app_id: '100197847',
    }

    expect(getV115AuthAction(qrAccount)).toBe('pkce')
    expect(getV115AuthAction(oauthAccount)).toBe('oauth')
    expect(getV115AuthAction(thirdPartyAccount)).toBe('oauth')
  })
})
