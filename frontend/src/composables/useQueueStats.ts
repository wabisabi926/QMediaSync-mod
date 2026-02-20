import { ref, onMounted, onUnmounted } from 'vue'
import { inject } from 'vue'
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
  let queueStatsTimer: number | null = null

  const loadQueueStats = async () => {
    try {
      queueStatsLoading.value = true
      const response = await http?.get(`${SERVER_URL}/115/queue/stats`)
      if (response && response.data && response.data.code === 200) {
        queueStats.value = response.data.data
      } else {
        queueStats.value = null
      }
    } catch (error) {
      console.error('加载115接口请求统计错误:', error)
      queueStats.value = null
    } finally {
      queueStatsLoading.value = false
    }
  }

  const startPolling = () => {
    if (queueStatsTimer) {
      clearInterval(queueStatsTimer)
    }
    queueStatsTimer = setInterval(() => {
      loadQueueStats()
    }, pollingInterval)
  }

  const stopPolling = () => {
    if (queueStatsTimer) {
      clearInterval(queueStatsTimer)
      queueStatsTimer = null
    }
  }

  onMounted(() => {
    loadQueueStats()
    startPolling()
  })

  onUnmounted(() => {
    stopPolling()
  })

  return {
    queueStats,
    queueStatsLoading,
    loadQueueStats,
    startPolling,
    stopPolling
  }
}
