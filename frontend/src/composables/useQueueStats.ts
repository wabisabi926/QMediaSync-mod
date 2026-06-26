import { inject, onMounted, onUnmounted, ref } from 'vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'

export interface QueueStats {
  avg_response_time_ms: number
  is_throttled: boolean
  last_throttle_time: string | null
  qph_count: number
  qpm_count: number
  qps_count: number
  throttle_recover_time: string
  throttle_wait_time: string
  throttled_count: number
  throttled_elapsed_time: string
  throttled_remaining_time: string
  time_window_seconds: number
  total_requests: number
}

export function useQueueStats(pollingInterval = 3000) {
  const http = inject<AxiosStatic>('$http')
  const queueStats = ref<QueueStats | null>(null)
  const queueStatsLoading = ref(false)
  const hasLoaded = ref(false)
  const maxPollingInterval = 30000
  let queueStatsTimer: ReturnType<typeof setTimeout> | null = null
  let inFlight = false
  let currentPollingInterval = pollingInterval

  const loadQueueStats = async () => {
    if (inFlight) {
      return
    }
    inFlight = true

    try {
      queueStatsLoading.value = !hasLoaded.value
      const response = await http?.get(`${SERVER_URL}/115/queue/stats`)
      if (response && response.data && response.data.code === 200) {
        queueStats.value = response.data.data
        hasLoaded.value = true
        currentPollingInterval = pollingInterval
      } else if (!hasLoaded.value) {
        queueStats.value = null
        currentPollingInterval = Math.min(currentPollingInterval * 2, maxPollingInterval)
      } else {
        currentPollingInterval = Math.min(currentPollingInterval * 2, maxPollingInterval)
      }
    } catch (error) {
      console.error('加载 115 接口请求统计错误：', error)
      if (!hasLoaded.value) {
        queueStats.value = null
      }
      currentPollingInterval = Math.min(currentPollingInterval * 2, maxPollingInterval)
    } finally {
      inFlight = false
      queueStatsLoading.value = false
    }
  }

  const scheduleNextPolling = () => {
    if (queueStatsTimer || document.hidden) {
      return
    }
    queueStatsTimer = setTimeout(async () => {
      queueStatsTimer = null
      await loadQueueStats()
      scheduleNextPolling()
    }, currentPollingInterval)
  }

  const startPolling = () => {
    stopPolling()
    scheduleNextPolling()
  }

  const stopPolling = () => {
    if (queueStatsTimer) {
      clearTimeout(queueStatsTimer)
      queueStatsTimer = null
    }
  }

  const handleVisibilityChange = () => {
    if (document.hidden) {
      stopPolling()
      return
    }
    void loadQueueStats()
    scheduleNextPolling()
  }

  onMounted(() => {
    loadQueueStats()
    document.addEventListener('visibilitychange', handleVisibilityChange)
    startPolling()
  })

  onUnmounted(() => {
    stopPolling()
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  })

  return {
    queueStats,
    queueStatsLoading,
    loadQueueStats,
    startPolling,
    stopPolling,
  }
}
