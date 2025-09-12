/**
 * 检测是否为移动端设备
 * @returns {boolean} 如果是移动端设备返回true，否则返回false
 */
export function isMobile(): boolean {
  if (typeof window !== 'undefined') {
    return window.innerWidth <= 768
  }
  return false
}

/**
 * 监听窗口大小变化，更新是否为移动端设备的状态
 * @param callback 回调函数，当设备类型发生变化时调用
 * @returns 清除监听器的函数
 */
export function onDeviceTypeChange(callback: (isMobile: boolean) => void): () => void {
  let currentIsMobile = isMobile()
  
  const handleResize = () => {
    const newIsMobile = isMobile()
    if (newIsMobile !== currentIsMobile) {
      currentIsMobile = newIsMobile
      callback(newIsMobile)
    }
  }

  window.addEventListener('resize', handleResize)
  
  // 返回清除监听器的函数
  return () => {
    window.removeEventListener('resize', handleResize)
  }
}