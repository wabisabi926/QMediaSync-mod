export function buildApiWebSocketUrl(
  serverUrl: string,
  apiPath: string,
  currentHref = typeof window === 'undefined' ? 'http://localhost/' : window.location.href,
) {
  const normalizedPath = apiPath.startsWith('/') ? apiPath : `/${apiPath}`
  const current = new URL(currentHref)
  const apiBase = serverUrl.startsWith('http')
    ? new URL(serverUrl)
    : new URL(serverUrl.replace(/^\//, ''), `${current.protocol}//${current.host}/`)
  const protocol = apiBase.protocol === 'https:' ? 'wss:' : 'ws:'
  const basePath = apiBase.pathname.replace(/\/$/, '')
  return `${protocol}//${apiBase.host}${basePath}${normalizedPath}`
}
