const unsafeMethods = new Set(['post', 'put', 'patch', 'delete'])

export const shouldAttachCSRFToken = (method?: string) => {
  return unsafeMethods.has((method || 'get').toLowerCase())
}

export const getCSRFTokenFromCookie = () => {
  const item = document.cookie
    .split(';')
    .map((part) => part.trim())
    .find((part) => part.startsWith('csrf_token='))
  return item ? decodeURIComponent(item.slice('csrf_token='.length)) : ''
}
