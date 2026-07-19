import { describe, expect, it } from 'vitest'
import * as syncTaskEventSequence from '@/utils/syncTaskEventSequence'

const { shouldApplySyncTaskEvent } = syncTaskEventSequence

describe('shouldApplySyncTaskEvent', () => {
  it('接受终态清理 sequence 后序号回退的删除事件', () => {
    const sequences = new Map([[42, 8]])

    expect(shouldApplySyncTaskEvent(sequences, 42, 1, true)).toBe(true)
    expect(sequences.has(42)).toBe(false)
  })

  it('继续忽略同一运行中任务的旧 patch', () => {
    const sequences = new Map([[42, 8]])

    expect(shouldApplySyncTaskEvent(sequences, 42, 8, false)).toBe(false)
    expect(shouldApplySyncTaskEvent(sequences, 42, 9, false)).toBe(true)
    expect(sequences.get(42)).toBe(9)
  })

  it('HTTP snapshot 收敛后接受服务重启后重新开始的 sequence', () => {
    const sequences = new Map([[42, 8]])
    const resetSequences = (
      syncTaskEventSequence as typeof syncTaskEventSequence & {
        resetSyncTaskEventSequences?: (sequences: Map<number, number>) => void
      }
    ).resetSyncTaskEventSequences

    expect(resetSequences).toBeTypeOf('function')
    resetSequences?.(sequences)

    expect(shouldApplySyncTaskEvent(sequences, 42, 1, false)).toBe(true)
  })
})
