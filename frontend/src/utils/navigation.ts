import type { RouteLocationRaw, Router } from 'vue-router'

interface NavigateBackOrReplaceOptions {
  historyState?: unknown
}

const hasOwn = (value: object, key: string) => Object.prototype.hasOwnProperty.call(value, key)

const readHistoryState = (): unknown => {
  try {
    return globalThis.history?.state ?? null
  } catch {
    return null
  }
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null

// Vue Router 的 history.state.back 只记录应用内上一页；直接打开深链时为 null。
export const hasAppBackHistory = (historyState: unknown = readHistoryState()): boolean => {
  return isRecord(historyState) && typeof historyState.back === 'string' && historyState.back !== ''
}

export const navigateBackOrReplace = async (
  router: Pick<Router, 'back' | 'replace'>,
  fallback: RouteLocationRaw,
  options: NavigateBackOrReplaceOptions = {},
): Promise<void> => {
  const historyState = hasOwn(options, 'historyState') ? options.historyState : readHistoryState()

  if (hasAppBackHistory(historyState)) {
    router.back()
    return
  }

  await router.replace(fallback)
}
