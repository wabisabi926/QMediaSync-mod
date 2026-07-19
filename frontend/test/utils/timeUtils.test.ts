import { describe, expect, it } from 'vitest'
import {
  formatRelativeTime,
  formatMaybeUnixDateTime,
  formatUnixDate,
  formatUnixDateTime,
  normalizeLegacyDateTime,
} from '@/utils/timeUtils'

const formatByRuntime = (timestamp: number): string =>
  new Date(timestamp * 1000)
    .toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    })
    .replace(/\//g, '-')

describe('timeUtils', () => {
  it('优先把 Unix 秒格式化为浏览器本地日期时间', () => {
    const timestamp = 1782634303

    expect(formatUnixDateTime(timestamp)).toBe(formatByRuntime(timestamp))
    expect(formatMaybeUnixDateTime(timestamp)).toBe(formatByRuntime(timestamp))
  })

  it('兼容 RFC3339 UTC 字符串并按浏览器本地时间显示', () => {
    const rfc3339 = '2026-06-28T13:31:43Z'
    const timestamp = Math.floor(Date.parse(rfc3339) / 1000)

    expect(formatMaybeUnixDateTime(rfc3339)).toBe(formatByRuntime(timestamp))
  })

  it('兼容旧无时区 date 字符串', () => {
    expect(formatMaybeUnixDateTime('2026-06-28 13:34:38')).toBe('2026-06-28 13:34:38')
    expect(normalizeLegacyDateTime('2026/06/28 13:34:38')).toBe('2026-06-28 13:34:38')
  })

  it('空值和非法值返回占位符', () => {
    expect(formatMaybeUnixDateTime(0)).toBe('-')
    expect(formatMaybeUnixDateTime(null)).toBe('-')
    expect(formatMaybeUnixDateTime(undefined)).toBe('-')
    expect(formatMaybeUnixDateTime('not a date')).toBe('-')
  })

  it('格式化 Unix 秒日期', () => {
    const timestamp = Math.floor(Date.parse('2026-06-28T13:31:43Z') / 1000)

    expect(formatUnixDate(timestamp)).toMatch(/^2026-06-(28|29)$/)
    expect(formatUnixDate(0)).toBe('-')
  })

  it('格式化相对时间', () => {
    const now = Math.floor(Date.parse('2026-06-28T14:31:43Z') / 1000)
    const oneHourAgo = now - 3600
    const future = now + 60

    expect(formatRelativeTime(oneHourAgo, now)).toBe('1 小时前')
    expect(formatRelativeTime(future, now)).toBe(formatByRuntime(future))
    expect(formatRelativeTime(null, now)).toBe('-')
    expect(formatRelativeTime('not a date', now)).toBe('-')
  })
})
