export const downloadSourceNameMap: Record<string, string> = {
  strm_sync: 'STRM 同步',
  local_file: '本地文件',
  emby_media: 'Emby 媒体信息提取',
}

export const uploadSourceNameMap: Record<string, string> = {
  strm_sync: 'STRM 同步',
  scrape_organize: '刮削整理',
}

export const syncTaskTypeNameMap: Record<string, string> = {
  strm_sync: 'STRM 同步',
  scrape_organize: '刮削整理',
}

const taskSourceTypeNameMap: Record<string, string> = {
  '115': '115 网盘',
  baidupan: '百度网盘',
  openlist: 'OpenList',
  local: '本地文件',
  '123': '123 网盘',
  emby_media: 'Emby 媒体信息提取',
}

const taskSourceTypeTagTypeMap: Record<string, string> = {
  '115': 'primary',
  baidupan: 'danger',
  openlist: 'success',
  local: 'warning',
  '123': 'info',
  emby_media: 'info',
}

export const getDownloadSourceName = (source: string): string => {
  return downloadSourceNameMap[source] ?? source
}

export const getUploadSourceName = (source: string): string => {
  return uploadSourceNameMap[source] ?? source
}

export const getSyncTaskTypeName = (taskType: string): string => {
  return syncTaskTypeNameMap[taskType] ?? taskType
}

export const getTaskSourceTypeName = (type: string): string => {
  return taskSourceTypeNameMap[type] ?? '其他'
}

export const getTaskSourceTypeTagType = (type: string): string => {
  return taskSourceTypeTagTypeMap[type] ?? 'info'
}
