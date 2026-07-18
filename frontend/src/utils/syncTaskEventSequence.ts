// shouldApplySyncTaskEvent 判断同步任务事件是否比当前已处理事件更新。
export function shouldApplySyncTaskEvent(
  sequences: Map<number, number>,
  syncID: number,
  sequence: number | undefined,
  deleted: boolean,
): boolean {
  if (deleted) {
    sequences.delete(syncID)
    return true
  }

  const lastSequence = sequences.get(syncID) || 0
  if (sequence && sequence <= lastSequence) {
    return false
  }
  if (sequence) {
    sequences.set(syncID, sequence)
  }
  return true
}

// resetSyncTaskEventSequences 在 HTTP snapshot 收敛前清空进程内事件水位。
export function resetSyncTaskEventSequences(sequences: Map<number, number>) {
  sequences.clear()
}
