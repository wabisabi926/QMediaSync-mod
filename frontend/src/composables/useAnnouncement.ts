import { ref, onMounted } from 'vue'
import { inject } from 'vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'

export interface Announcement {
  id?: number | string
  time: string
  title: string
  content: string
}

export function useAnnouncement() {
  const http = inject<AxiosStatic>('$http')
  const announcementList = ref<Announcement[]>([])
  const announcementLoading = ref(false)

  const loadAnnouncements = async () => {
    try {
      announcementLoading.value = true
      const response = await http?.get(`${SERVER_URL}/announce`)
      if (response && response.data) {
        if (response.data.code === 200 && response.data.data) {
          announcementList.value = response.data.data
        } else if (Array.isArray(response.data)) {
          announcementList.value = response.data
        } else {
          announcementList.value = []
        }
      } else {
        announcementList.value = []
      }
    } catch (error) {
      console.error('加载公告列表错误:', error)
      announcementList.value = []
    } finally {
      announcementLoading.value = false
    }
  }

  onMounted(() => {
    loadAnnouncements()
  })

  return {
    announcementList,
    announcementLoading,
    loadAnnouncements
  }
}
