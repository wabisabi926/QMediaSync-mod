import { onMounted, onUnmounted, readonly, shallowRef } from 'vue'
import { isMobile as checkIsMobile, onDeviceTypeChange } from '@/utils/deviceUtils'

export function useDeviceType() {
  const isMobile = shallowRef(checkIsMobile())
  let cleanup: (() => void) | null = null

  const start = () => {
    if (cleanup) {
      return
    }

    isMobile.value = checkIsMobile()
    cleanup = onDeviceTypeChange((newIsMobile) => {
      isMobile.value = newIsMobile
    })
  }

  const stop = () => {
    if (!cleanup) {
      return
    }

    cleanup()
    cleanup = null
  }

  onMounted(start)
  onUnmounted(stop)

  return {
    isMobile: readonly(isMobile),
    start,
    stop,
  }
}
