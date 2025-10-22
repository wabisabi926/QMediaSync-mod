/**
 * 文件大小格式化工具函数
 */

/**
 * 将字节大小格式化为合适的单位显示
 * @param bytes 字节大小
 * @returns 格式化后的文件大小字符串
 */
export function formatFileSize(bytes: number): string {
  // 处理无效输入
  if (bytes === null || bytes === undefined || isNaN(bytes) || bytes < 0) {
    return 'N/A'
  }
  
  if (bytes === 0) return '0 Bytes'
  
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  // 确保索引在有效范围内
  const index = Math.min(i, sizes.length - 1)
  
  return parseFloat((bytes / Math.pow(k, index)).toFixed(2)) + ' ' + sizes[index]
}