import type { FileType } from '@/typing'

/**
 * 根据文件名获取文件类型
 * @param filename 文件名
 * @returns 文件类型
 */
export function getFileType(filename: string): FileType {
  if (!filename) return 'other'

  const ext = filename.toLowerCase().split('.').pop()
  if (!ext) return 'other'

  // 视频文件
  const videoExtensions = ['mp4', 'mkv', 'avi', 'mov', 'wmv', 'flv', 'm4v', 'webm', 'ts', 'rmvb', 'rm', '3gp', 'mpg', 'mpeg']
  if (videoExtensions.includes(ext)) return 'video'

  // 图片文件
  const imageExtensions = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg', 'ico', 'tiff', 'tga']
  if (imageExtensions.includes(ext)) return 'image'

  // NFO信息文件
  if (ext === 'nfo') return 'nfo'

  return 'other'
}

/**
 * 根据文件类型获取对应的Element Plus图标
 * @param type 文件类型
 * @param isDirectory 是否为目录
 * @returns Element Plus图标名称
 */
export function getFileIcon(type: FileType, isDirectory = false): string {
  if (isDirectory) return 'Folder'

  switch (type) {
    case 'video':
      return 'VideoPlay'
    case 'image':
      return 'Picture'
    case 'nfo':
      return 'Document'
    default:
      return 'Document'
  }
}

/**
 * 根据文件名直接获取图标
 * @param filename 文件名
 * @param isDirectory 是否为目录
 * @returns Element Plus图标名称
 */
export function getFileIconByName(filename: string, isDirectory = false): string {
  const type = getFileType(filename)
  return getFileIcon(type, isDirectory)
}
