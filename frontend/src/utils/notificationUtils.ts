// 渠道类型
export type ChannelType = 'telegram' | 'meow' | 'bark' | 'serverchan' | 'webhook'

// 事件类型
export type EventType =
  | 'sync_finish'
  | 'sync_error'
  | 'scrape_finish'
  | 'system_alert'
  | 'media_added'
  | 'media_removed'

// 通知渠道接口
export interface NotificationChannel {
  id: number
  channel_type: ChannelType
  channel_name: string
  is_enabled: boolean
  created_at: string
  updated_at: string
  config?: NotificationConfig
  rules?: NotificationRule[]
}

// 通知配置接口（不同类型的配置）
export interface NotificationConfig {
  // Telegram
  bot_token?: string
  chat_id?: string
  proxy_url?: string

  // MeoW
  nickname?: string
  endpoint?: string

  // Bark
  device_key?: string
  server_url?: string
  sound?: string
  icon?: string

  // Server酱
  sc_key?: string

  // Webhook
  method?: string
  format?: string
  template?: string
  query_param?: string
  auth_type?: string
  auth_token?: string
  auth_user?: string
  auth_pass?: string
  auth_header_key?: string
  auth_query_key?: string
  headers?: Record<string, string>
  description?: string
}

// 通知规则接口
export interface NotificationRule {
  id: number
  channel_id: number
  event_type: EventType
  is_enabled: boolean
  created_at: string
  updated_at: string
}

// 获取渠道类型名称
export function getChannelTypeName(type: ChannelType): string {
  const nameMap: Record<ChannelType, string> = {
    telegram: 'Telegram',
    meow: 'MeoW',
    bark: 'Bark',
    serverchan: 'Server酱',
    webhook: 'Webhook'
  }
  return nameMap[type] || type
}

// 获取渠道类型颜色
export function getChannelTypeColor(
  type: ChannelType
): 'primary' | 'success' | 'info' | 'warning' | 'danger' {
  const colorMap: Record<ChannelType, 'primary' | 'success' | 'info' | 'warning' | 'danger'> = {
    telegram: 'primary',
    meow: 'success',
    bark: 'warning',
    serverchan: 'info',
    webhook: 'danger'
  }
  return colorMap[type] || 'info'
}

// 获取事件类型名称
export function getEventTypeName(type: EventType): string {
  const nameMap: Record<EventType, string> = {
    sync_finish: '同步完成',
    sync_error: '同步错误',
    scrape_finish: '刮削完成',
    system_alert: '系统警告',
    media_added: '媒体添加',
    media_removed: '媒体移除'
  }
  return nameMap[type] || type
}

// 获取事件类型描述
export function getEventTypeDescription(type: EventType): string {
  const descMap: Record<EventType, string> = {
    sync_finish: '同步任务成功完成时发送通知',
    sync_error: '同步任务出现错误时发送通知',
    scrape_finish: '刮削任务完成时发送通知',
    system_alert: '系统出现重要警告时发送通知',
    media_added: '新媒体添加到媒体库时发送通知',
    media_removed: '媒体从媒体库移除时发送通知'
  }
  return descMap[type] || ''
}
