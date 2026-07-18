const sources = new Set<() => void>()

// registerRealtimeSource 登记一个实时连接关闭函数，并返回幂等注销函数。
export function registerRealtimeSource(close: () => void): () => void {
  let registered = true
  sources.add(close)

  return () => {
    if (!registered) return
    registered = false
    sources.delete(close)
  }
}

// closeAllRealtimeSources 关闭当前已登记的所有实时连接。
export function closeAllRealtimeSources() {
  const closers = [...sources]
  sources.clear()
  closers.forEach((close) => close())
}
