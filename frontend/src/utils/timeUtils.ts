// 时间格式化工具函数

export type MaybeUnixDateTime = number | string | null | undefined

const DATE_TIME_PLACEHOLDER = '-'

const dateTimeFormatterOptions: Intl.DateTimeFormatOptions = {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
}

const normalizeDateTimeParts = (value: string): string =>
  value.trim().replace(/\//g, '-').replace('T', ' ').replace(/Z$/, '')

const formatDateObject = (date: Date): string => {
  if (Number.isNaN(date.getTime())) {
    return DATE_TIME_PLACEHOLDER
  }

  return date.toLocaleString('zh-CN', dateTimeFormatterOptions).replace(/\//g, '-')
}

export const normalizeLegacyDateTime = (value: string): string => {
  const normalized = normalizeDateTimeParts(value)
  if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(normalized)) {
    return normalized
  }

  return DATE_TIME_PLACEHOLDER
}

export const formatUnixDateTime = (timestamp: number | null | undefined): string => {
  if (!timestamp) {
    return DATE_TIME_PLACEHOLDER
  }

  return formatDateObject(new Date(timestamp * 1000))
}

export const formatMaybeUnixDateTime = (value: MaybeUnixDateTime): string => {
  if (value === null || value === undefined || value === '') {
    return DATE_TIME_PLACEHOLDER
  }

  if (typeof value === 'number') {
    return formatUnixDateTime(value)
  }

  const trimmed = value.trim()
  if (!trimmed) {
    return DATE_TIME_PLACEHOLDER
  }

  if (/^\d+$/.test(trimmed)) {
    return formatUnixDateTime(Number(trimmed))
  }

  if (trimmed.includes('T') || /(?:Z|[+-]\d{2}:\d{2})$/.test(trimmed)) {
    return formatDateObject(new Date(trimmed))
  }

  return normalizeLegacyDateTime(trimmed)
}

/**
 * 格式化时间戳为日期时间字符串 (YYYY-MM-DD HH:MM:SS)
 * @param timestamp 时间戳 (秒)
 * @returns 格式化后的日期时间字符串
 */
export const formatTimestamp = (timestamp: number): string => {
  return formatUnixDateTime(timestamp)
}

/**
 * 格式化日期时间戳为可读字符串
 * @param timestamp 时间戳 (秒)
 * @returns 格式化后的日期时间字符串
 */
export const formatDateTime = (timestamp: number): string => {
  if (!timestamp) return '-'
  const date = new Date(timestamp * 1000)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}

/**
 * 格式化时间戳为时间字符串
 * @param timestamp 时间戳 (秒)
 * @returns 格式化后的时间字符串
 */
export const formatTime = (timestamp: number): string => {
  if (!timestamp) return '-'
  const date = new Date(timestamp * 1000)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}

/**
 * 格式化存储空间大小
 * @param bytes 字节数
 * @returns 格式化后的存储空间字符串
 */
export const formatStorage = (bytes: number): string => {
  if (!bytes || bytes === 0) return '0 B'

  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const size = bytes / Math.pow(1024, i)

  return `${size.toFixed(i === 0 ? 0 : 2)} ${sizes[i]}`
}

/**
 * 计算存储使用百分比
 * @param used 已使用空间
 * @param total 总空间
 * @returns 百分比
 */
export const getStoragePercent = (used: number, total: number): number => {
  if (!total || total === 0) return 0
  return Math.round((used / total) * 100)
}

/**
 * 获取进度条颜色
 * @param used 已使用空间
 * @param total 总空间
 * @returns 颜色字符串
 */
export const getProgressColor = (used: number, total: number): string => {
  const percent = getStoragePercent(used, total)
  if (percent >= 90) return '#f56c6c'
  if (percent >= 70) return '#e6a23c'
  return '#67c23a'
}

/**
 * 获取会员等级样式类
 * @param level 会员等级
 * @returns 样式类名
 */
export const getMemberClass = (level: string): string => {
  const lowerLevel = level.toLowerCase()
  if (lowerLevel.includes('vip') || lowerLevel.includes('会员')) {
    return 'member-vip'
  }
  return 'member-normal'
}

/**
 * 格式化到期时间
 * @param expireTime 到期时间字符串
 * @returns 格式化后的到期时间
 */
export const formatExpireTime = (expireTime: string): string => {
  if (!expireTime) return '未知'

  const date = new Date(expireTime)
  if (isNaN(date.getTime())) return expireTime

  const now = new Date()
  const diffTime = date.getTime() - now.getTime()
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24))

  if (diffDays < 0) return '已过期'
  if (diffDays === 0) return '今天到期'
  if (diffDays <= 30) return `${diffDays} 天后到期`

  return date.toLocaleDateString('zh-CN')
}

/**
 * 获取到期时间样式类
 * @param expireTime 到期时间字符串
 * @returns 样式类名
 */
export const getExpireClass = (expireTime: string): string => {
  if (!expireTime) return 'expire-unknown'

  const date = new Date(expireTime)
  if (isNaN(date.getTime())) return 'expire-unknown'

  const now = new Date()
  const diffTime = date.getTime() - now.getTime()
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24))

  if (diffDays < 0) return 'expire-expired'
  if (diffDays <= 7) return 'expire-warning'
  if (diffDays <= 30) return 'expire-notice'

  return 'expire-normal'
}

/**
 * 格式化秒数为可读的时间段
 * @param seconds 秒数
 * @returns 格式化后的时间字符串
 */
export const formatDuration = (seconds: number): string => {
  if (!seconds || seconds <= 0) return '0 秒'

  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)

  const parts: string[] = []
  if (hours > 0) parts.push(`${hours} 小时`)
  if (minutes > 0) parts.push(`${minutes} 分`)
  if (secs > 0 || parts.length === 0) parts.push(`${secs} 秒`)

  return parts.join(' ')
}
