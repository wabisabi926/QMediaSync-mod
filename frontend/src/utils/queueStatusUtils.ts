export interface QueueStatusSnapshot {
  running: boolean
  pending: number
  processing: number
  completed: number
  failed: number
  cancelled: number
  total: number
}

export const emptyQueueStatusSnapshot = (): QueueStatusSnapshot => ({
  running: false,
  pending: 0,
  processing: 0,
  completed: 0,
  failed: 0,
  cancelled: 0,
  total: 0,
})

const toNumber = (value: unknown): number => (typeof value === 'number' ? value : 0)

export const normalizeQueueStatusSnapshot = (
  value: unknown,
  fallbackRunning = false,
): QueueStatusSnapshot => {
  if (typeof value === 'boolean') {
    return { ...emptyQueueStatusSnapshot(), running: value }
  }
  if (!value || typeof value !== 'object') {
    return { ...emptyQueueStatusSnapshot(), running: fallbackRunning }
  }

  const record = value as Record<string, unknown>
  return {
    running: typeof record.running === 'boolean' ? record.running : fallbackRunning,
    pending: toNumber(record.pending),
    processing: toNumber(record.processing),
    completed: toNumber(record.completed),
    failed: toNumber(record.failed),
    cancelled: toNumber(record.cancelled),
    total: toNumber(record.total),
  }
}

export const canPauseQueue = (snapshot: QueueStatusSnapshot): boolean => snapshot.running

export const canResumeQueue = (snapshot: QueueStatusSnapshot): boolean => !snapshot.running

export interface QueueRowWithStatus {
  status: number
}

export const removePendingQueueRows = <T extends QueueRowWithStatus>(rows: T[]): T[] =>
  rows.filter((row) => row.status !== 0)
