import { SERVER_URL } from '@/const'
import type {
  V115AuthStatus,
  V115QrCodePayload,
  V115QrCodeStatusPayload,
} from '@/types/v115Auth'
import type { AxiosStatic } from 'axios'
import { computed, onBeforeUnmount, shallowRef } from 'vue'

export function useV115DeviceAuthorization(http: AxiosStatic | undefined) {
  const qrCode = shallowRef<V115QrCodePayload | null>(null)
  const status = shallowRef<V115AuthStatus>('idle')
  const tip = shallowRef('')
  const loading = shallowRef(false)
  const pollTimer = shallowRef<number | null>(null)
  const accountId = shallowRef<number | null>(null)

  const isPolling = computed(() => pollTimer.value !== null)

  const stopPolling = () => {
    if (pollTimer.value !== null) {
      window.clearInterval(pollTimer.value)
      pollTimer.value = null
    }
  }

  const pollStatus = async () => {
    if (!http || !accountId.value || !qrCode.value) return

    try {
      const response = await http.post(`${SERVER_URL}/auth/115-qrcode-status`, {
        account_id: accountId.value,
        uid: qrCode.value.uid,
      })
      const data = response.data?.data as V115QrCodeStatusPayload | undefined
      if (response.data?.code !== 200 || !data) {
        status.value = 'failed'
        tip.value = response.data?.message || '授权状态查询失败'
        stopPolling()
        return
      }
      status.value = data.status
      tip.value = data.tip
      if (['confirmed', 'expired', 'failed'].includes(data.status)) {
        stopPolling()
      }
    } catch (error) {
      status.value = 'failed'
      tip.value = error instanceof Error ? error.message : '授权状态查询失败'
      stopPolling()
    }
  }

  const startAuthorization = async (nextAccountId: number) => {
    if (!http) return
    stopPolling()
    loading.value = true
    accountId.value = nextAccountId
    status.value = 'waiting'
    tip.value = '正在获取二维码…'
    qrCode.value = null

    try {
      const response = await http.post(`${SERVER_URL}/auth/115-qrcode-open`, {
        account_id: nextAccountId,
      })
      if (response.data?.code !== 200 || !response.data.data) {
        status.value = 'failed'
        tip.value = response.data?.message || '获取二维码失败'
        return
      }
      qrCode.value = response.data.data as V115QrCodePayload
      tip.value = '等待扫码'
      pollTimer.value = window.setInterval(() => void pollStatus(), 3000)
      void pollStatus()
    } catch (error) {
      status.value = 'failed'
      tip.value = error instanceof Error ? error.message : '获取二维码失败'
    } finally {
      loading.value = false
    }
  }

  const resetAuthorization = () => {
    stopPolling()
    qrCode.value = null
    status.value = 'idle'
    tip.value = ''
    accountId.value = null
  }

  onBeforeUnmount(stopPolling)

  return {
    qrCode,
    status,
    tip,
    loading,
    isPolling,
    startAuthorization,
    stopPolling,
    resetAuthorization,
  }
}
