const normalizeServerURL = (value: string | undefined, fallback: string): string => {
  const normalized = value?.trim()
  if (!normalized || normalized === 'undefined' || normalized === 'null') {
    return fallback
  }
  return normalized.replace(/\/+$/, '')
}

const DEFAULT_SERVER_URL = '/api'
const SERVER_URL = normalizeServerURL(import.meta.env.VITE_SERVER_URL, DEFAULT_SERVER_URL)

export { SERVER_URL, normalizeServerURL }
