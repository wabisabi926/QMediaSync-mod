import { describe, expect, it } from 'vitest'
import {
  canPauseQueue,
  canResumeQueue,
  hasActiveQueueTasks,
  normalizeQueueStatusSnapshot,
  removePendingQueueRows,
} from '@/utils/queueStatusUtils'

describe('queueStatusUtils', () => {
  it('运行中且有等待任务时允许全部暂停', () => {
    const snapshot = normalizeQueueStatusSnapshot({ running: true, pending: 2, total: 2 })
    expect(canPauseQueue(snapshot)).toBe(true)
    expect(canResumeQueue(snapshot)).toBe(false)
  })

  it('运行中但当前没有可处理任务时仍允许全部暂停', () => {
    const snapshot = normalizeQueueStatusSnapshot({ running: true, completed: 3, total: 3 })
    expect(canPauseQueue(snapshot)).toBe(true)
  })

  it('已暂停且有等待任务时允许全部恢复', () => {
    const snapshot = normalizeQueueStatusSnapshot({ running: false, pending: 1, total: 1 })
    expect(canPauseQueue(snapshot)).toBe(false)
    expect(canResumeQueue(snapshot)).toBe(true)
  })

  it('已暂停且当前为空时仍允许全部恢复', () => {
    const snapshot = normalizeQueueStatusSnapshot({ running: false })
    expect(canPauseQueue(snapshot)).toBe(false)
    expect(canResumeQueue(snapshot)).toBe(true)
  })

  it('兼容旧布尔状态响应', () => {
    expect(normalizeQueueStatusSnapshot(true).running).toBe(true)
    expect(normalizeQueueStatusSnapshot(false).running).toBe(false)
  })

  it('只乐观移除当前页等待任务', () => {
    const rows = [
      { id: 1, status: 0 },
      { id: 2, status: 1 },
      { id: 3, status: 2 },
    ]
    expect(removePendingQueueRows(rows)).toEqual([
      { id: 2, status: 1 },
      { id: 3, status: 2 },
    ])
  })

  it('队列已启动但没有等待或处理任务时不视为活跃', () => {
    const snapshot = normalizeQueueStatusSnapshot({ running: true, completed: 3, total: 3 })
    expect(hasActiveQueueTasks(snapshot)).toBe(false)
  })

  it('有等待或处理任务时视为活跃', () => {
    expect(hasActiveQueueTasks(normalizeQueueStatusSnapshot({ running: true, pending: 1 }))).toBe(
      true,
    )
    expect(
      hasActiveQueueTasks(normalizeQueueStatusSnapshot({ running: false, processing: 1 })),
    ).toBe(true)
  })
})
