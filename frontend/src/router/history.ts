import { createWebHashHistory, type RouterHistory } from 'vue-router'

export function runWithoutHiddenScrollListeners<T>(factory: () => T): T {
  const originalDocumentAddEventListener =
    typeof document === 'undefined' ? undefined : document.addEventListener
  const originalWindowAddEventListener =
    typeof window === 'undefined' ? undefined : window.addEventListener

  if (!originalDocumentAddEventListener && !originalWindowAddEventListener) {
    return factory()
  }

  if (originalDocumentAddEventListener) {
    document.addEventListener = ((
      type: string,
      listener: EventListenerOrEventListenerObject | null,
      options?: boolean | AddEventListenerOptions,
    ) => {
      if (type === 'visibilitychange') {
        return
      }

      return originalDocumentAddEventListener.call(
        document,
        type,
        listener as EventListenerOrEventListenerObject,
        options,
      )
    }) as typeof document.addEventListener
  }

  if (originalWindowAddEventListener) {
    window.addEventListener = ((
      type: string,
      listener: EventListenerOrEventListenerObject | null,
      options?: boolean | AddEventListenerOptions,
    ) => {
      if (type === 'pagehide') {
        return
      }

      return originalWindowAddEventListener.call(
        window,
        type,
        listener as EventListenerOrEventListenerObject,
        options,
      )
    }) as typeof window.addEventListener
  }

  try {
    return factory()
  } finally {
    if (originalDocumentAddEventListener) {
      document.addEventListener = originalDocumentAddEventListener
    }
    if (originalWindowAddEventListener) {
      window.addEventListener = originalWindowAddEventListener
    }
  }
}

// Vue Router 会在页面 hidden 时写入 history.state.scroll，部分浏览器窗口最小化会被该写入打断。
export function createQMediaSyncHashHistory(): RouterHistory {
  return runWithoutHiddenScrollListeners(() => createWebHashHistory())
}
