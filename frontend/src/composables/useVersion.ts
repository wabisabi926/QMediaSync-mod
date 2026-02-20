import { ref, onMounted } from 'vue'
import { inject } from 'vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'

export interface VersionInfo {
  version: string
  date: string
}

export function useVersion() {
  const http = inject<AxiosStatic>('$http')
  const versionInfo = ref<VersionInfo | null>(null)
  const versionLoading = ref(true)

  const loadVersionInfo = async () => {
    try {
      versionLoading.value = true
      const response = await http?.get(`${SERVER_URL}/version`)
      if (response && response.data) {
        versionInfo.value = response.data
      } else {
        versionInfo.value = null
      }
    } catch (error) {
      console.error('加载系统版本信息错误:', error)
      versionInfo.value = null
    } finally {
      versionLoading.value = false
    }
  }

  onMounted(() => {
    loadVersionInfo()
  })

  return {
    versionInfo,
    versionLoading,
    loadVersionInfo
  }
}
