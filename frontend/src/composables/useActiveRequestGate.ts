export interface ActiveRequestGate {
  next: () => number
  invalidate: () => void
  isCurrent: (requestId: number) => boolean
}

export function createActiveRequestGate(isActive: () => boolean): ActiveRequestGate {
  let currentRequestId = 0

  const next = () => {
    currentRequestId += 1
    return currentRequestId
  }

  const invalidate = () => {
    currentRequestId += 1
  }

  const isCurrent = (requestId: number) => isActive() && requestId === currentRequestId

  return {
    next,
    invalidate,
    isCurrent,
  }
}
