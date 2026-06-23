export type V115AuthMode = 'qr' | 'oauth'

export type V115AuthSourceType =
  | 'built_in_appid'
  | 'built_in_relay'
  | 'third_party_service'
  | 'custom_appid'

export type V115AuthProvider =
  | 'official_pkce'
  | 'mqfamily'
  | 'qmediasync'
  | 'moviepilot'
  | 'openlist'
  | 'clouddrive'

export interface V115SelectedQrApp {
  appId: string
  appName: string
}

export const pinnedBuiltInAppIDs = [
  { label: 'QMediaSync', value: '100197849', appName: 'QMediaSync' },
  { label: 'Q115-STRM', value: '100197665', appName: 'Q115-STRM' },
  { label: 'MQ的媒体库', value: '100197503', appName: 'MQ的媒体库' },
] as const

export const featuredBuiltInAppIDs = [
  { label: 'MoviePilot-115', value: '100197847', appName: 'MoviePilot-115' },
  { label: 'OpenList', value: '100197303', appName: 'OpenList' },
  { label: 'CloudDrive', value: '100195313', appName: 'CloudDrive' },
  { label: '媒体播放器', value: '100195125', appName: '媒体播放器' },
] as const

export const webAuthProviders = [
  {
    label: 'QMediaSync',
    sourceType: 'built_in_relay',
    provider: 'qmediasync',
    appName: 'QMediaSync',
  },
  {
    label: 'Q115-STRM',
    sourceType: 'built_in_relay',
    provider: 'mqfamily',
    appName: 'Q115-STRM',
  },
  {
    label: 'MQ的媒体库',
    sourceType: 'built_in_relay',
    provider: 'mqfamily',
    appName: 'MQ的媒体库',
  },
  {
    label: 'MoviePilot',
    sourceType: 'third_party_service',
    provider: 'moviepilot',
    appId: '100197847',
    appName: 'MoviePilot-115',
  },
  {
    label: 'OpenList',
    sourceType: 'third_party_service',
    provider: 'openlist',
    appId: '100197303',
    appName: 'OpenList',
  },
  {
    label: 'CloudDrive',
    sourceType: 'third_party_service',
    provider: 'clouddrive',
    appId: '100195313',
    appName: 'CloudDrive',
  },
] as const satisfies readonly {
  label: string
  sourceType: V115AuthSourceType
  provider: V115AuthProvider
  appId?: string
  appName: string
}[]
