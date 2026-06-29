export const THREAD_LIMITS = {
  downloadThreads: { min: 1, max: 10 },
  fileDetailThreads: { min: 2, max: 10 },
  openlistQPS: { min: 2, max: 10 },
  openlistRetry: { min: 1, max: 10 },
  openlistRetryDelay: { min: 30, max: 3600 },
  fileListPageSize: { min: 100, max: 1150 },
} as const

export const SCRAPE_THREAD_LIMITS = {
  localMax: 20,
  remoteMax: 5,
  min: 1,
} as const

export const STRM_GLOBAL_OPTIONS = {
  localProxy: [0, 1],
  uploadMeta: [0, 1, 2],
  downloadMeta: [0, 1],
  deleteDir: [0, 1],
  addPath: [1, 2, 3],
  checkMetaMtime: [0, 1],
} as const

export const STRM_CUSTOM_OPTIONS = {
  localProxy: [-1, 0, 1],
  uploadMeta: [-1, 0, 1, 2],
  downloadMeta: [-1, 0, 1],
  deleteDir: [-1, 0, 1],
  addPath: [-1, 1, 2, 3],
  checkMetaMtime: [-1, 0, 1],
} as const

export const HTTP_URL_PATTERN = /^(http|https):\/\/[^\s/$.?#].[^\s]*$/

export const CRON_DEFAULTS = {
  embySync: '0 * * * *',
} as const
