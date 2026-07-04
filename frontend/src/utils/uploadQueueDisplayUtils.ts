import { formatFileSize } from '@/utils/fileSizeUtils'

export interface UploadQueueDisplayTask {
  id: string | number
  status?: number
  source?: string
  file_size?: number
  uploaded_bytes?: number
  progress_percent?: number
  upload_speed_bytes?: number
  upload_phase?: string
  upload_result?: string
  resume_state?: string
  rapid_wait_until?: number
  total_parts?: number
  uploaded_parts?: number
  source_cleanup_status?: string
  source_cleanup_error?: string
}

export interface UploadQueuePatch {
  task_id?: string | number
  id?: string | number
  status?: number
  uploaded_bytes?: number
  file_size?: number
  progress_percent?: number
  upload_speed_bytes?: number
  upload_phase?: string
  upload_result?: string
  resume_state?: string
  rapid_wait_until?: number
  total_parts?: number
  uploaded_parts?: number
}

export interface UploadTaskDetailRow {
  label: string
  value: string
}

const phaseLabelMap: Record<string, string> = {
  rapid_waiting: '等待秒传',
  rapid_init: '等待秒传',
  resume_uploading: '恢复上传',
  resumed_uploading: '恢复上传',
  multipart_uploading: '上传中',
  uploading: '上传中',
  completing: '完成处理中',
  complete_callback: '完成处理中',
  rapid_uploaded: '秒传成功',
}

const resultLabelMap: Record<string, string> = {
  rapid_upload: '秒传成功',
  multipart_uploaded: '上传完成',
  remote_exists: '远端已存在',
  skipped_after_rapid_wait: '秒传等待超时',
  unknown: '',
}

const resumeStateLabelMap: Record<string, string> = {
  none: '',
  new_session: '新上传',
  resumed_session: '已恢复上传',
  session_expired_restarted: '续传会话已失效，已重新上传',
  resumed: '已恢复上传',
  resume_available: '可断点续传',
  no_session: '未发现可续传会话',
  new_upload: '新上传',
  stale_session: '续传会话已失效',
}

const cleanupStatusLabelMap: Record<string, string> = {
  none: '未清理',
  pending: '等待清理',
  completed: '清理成功',
  skipped: '未清理',
  success: '清理成功',
  deleted: '清理成功',
  failed: '清理失败',
}

const roundPercent = (value: number): number => Math.round(value * 10) / 10

export const getUploadProgressPercent = (task: UploadQueueDisplayTask): number => {
  if (typeof task.progress_percent === 'number' && Number.isFinite(task.progress_percent)) {
    return Math.min(100, Math.max(0, roundPercent(task.progress_percent)))
  }
  if (task.file_size && task.file_size > 0 && typeof task.uploaded_bytes === 'number') {
    return Math.min(100, Math.max(0, roundPercent((task.uploaded_bytes / task.file_size) * 100)))
  }
  if (task.status === 2) {
    return 100
  }
  return 0
}

export const formatByteRate = (bytesPerSecond?: number): string => {
  if (!bytesPerSecond || bytesPerSecond <= 0) {
    return '-'
  }
  return `${formatFileSize(bytesPerSecond)}/s`
}

export const getUploadPhaseLabel = (task: Pick<UploadQueueDisplayTask, 'upload_phase'>): string => {
  if (!task.upload_phase) {
    return '-'
  }
  return phaseLabelMap[task.upload_phase] ?? task.upload_phase
}

export const getUploadResultLabel = (
  task: Pick<UploadQueueDisplayTask, 'upload_result' | 'status'>,
): string => {
  if (!task.upload_result || task.upload_result === 'unknown') {
    if (task.status === 2) {
      return '上传完成'
    }
    return '-'
  }
  return resultLabelMap[task.upload_result] ?? task.upload_result
}

export const getUploadStageOrResultLabel = (task: UploadQueueDisplayTask): string => {
  if (task.status === 2 || task.upload_result) {
    return getUploadResultLabel(task)
  }
  return getUploadPhaseLabel(task)
}

export const getUploadedSizeLabel = (task: UploadQueueDisplayTask): string => {
  const uploadedBytes = Math.max(0, task.uploaded_bytes || 0)
  const fileSize = Math.max(0, task.file_size || 0)
  if (fileSize > 0) {
    return `${formatFileSize(uploadedBytes)} / ${formatFileSize(fileSize)}`
  }
  if (uploadedBytes > 0) {
    return formatFileSize(uploadedBytes)
  }
  return '-'
}

export const getResumeStateLabel = (resumeState?: string): string => {
  if (!resumeState) {
    return ''
  }
  return resumeStateLabelMap[resumeState] ?? resumeState
}

export const getSourceCleanupStatusLabel = (cleanupStatus?: string): string => {
  if (!cleanupStatus) {
    return ''
  }
  return cleanupStatusLabelMap[cleanupStatus] ?? cleanupStatus
}

export const getUploadTaskDetailRows = (task: UploadQueueDisplayTask): UploadTaskDetailRow[] => {
  const rows: UploadTaskDetailRow[] = []
  const resumeState = getResumeStateLabel(task.resume_state)
  if (resumeState) {
    rows.push({ label: '断点续传', value: resumeState })
  }
  if (task.total_parts && task.total_parts > 0) {
    rows.push({ label: '分片进度', value: `${task.uploaded_parts || 0}/${task.total_parts}` })
  }
  if (task.source === 'directory_monitor') {
    const cleanupStatus = getSourceCleanupStatusLabel(task.source_cleanup_status)
    if (cleanupStatus) {
      rows.push({ label: '源文件清理', value: cleanupStatus })
    }
    if (task.source_cleanup_error) {
      rows.push({ label: '清理失败原因', value: String(task.source_cleanup_error) })
    }
  }
  return rows
}

export const applyUploadQueuePatch = <T extends UploadQueueDisplayTask>(
  rows: T[],
  patch: UploadQueuePatch,
): boolean => {
  const taskId = patch.task_id ?? patch.id
  if (taskId === undefined || taskId === null) {
    return false
  }
  const row = rows.find((item) => String(item.id) === String(taskId))
  if (!row) {
    return false
  }
  if (patch.status !== undefined) {
    row.status = patch.status
  }
  if (patch.uploaded_bytes !== undefined) {
    row.uploaded_bytes = patch.uploaded_bytes
  }
  if (patch.file_size !== undefined) {
    row.file_size = patch.file_size
  }
  if (patch.progress_percent !== undefined) {
    row.progress_percent = patch.progress_percent
  }
  if (patch.upload_speed_bytes !== undefined) {
    row.upload_speed_bytes = patch.upload_speed_bytes
  }
  if (patch.upload_phase !== undefined) {
    row.upload_phase = patch.upload_phase
  }
  if (patch.upload_result !== undefined) {
    row.upload_result = patch.upload_result
  }
  if (patch.resume_state !== undefined) {
    row.resume_state = patch.resume_state
  }
  if (patch.rapid_wait_until !== undefined) {
    row.rapid_wait_until = patch.rapid_wait_until
  }
  if (patch.total_parts !== undefined) {
    row.total_parts = patch.total_parts
  }
  if (patch.uploaded_parts !== undefined) {
    row.uploaded_parts = patch.uploaded_parts
  }
  return true
}
