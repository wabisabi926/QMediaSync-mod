import { ref, onMounted, onUnmounted } from 'vue'
import { inject } from 'vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { ElMessage } from 'element-plus'

export interface UpdateInfo {
  version: string
  date: string
  note: string
  url: string
  latest?: boolean
  current?: boolean
}

export interface UpdateProgress {
  progress: number
  total_size: number
  downloaded: number
  status: string
  version?: string
}

export function useUpdate() {
  const http = inject<AxiosStatic>('$http')
  
  const updateList = ref<UpdateInfo[]>([])
  const updateLoading = ref(false)
  const isUpdating = ref(false)
  const updatingVersion = ref<string>('')
  const updateProgress = ref<UpdateProgress>({
    progress: 0,
    total_size: 0,
    downloaded: 0,
    status: ''
  })
  const showUpdateCompleteDialog = ref(false)
  const countdown = ref(30)
  
  let progressTimer: number | null = null
  let countdownTimer: number | null = null

  const loadUpdateList = async (force = false) => {
    try {
      updateLoading.value = true
      let url = `${SERVER_URL}/update/last`
      if (force) {
        url += '?force=1'
      }
      const response = await http?.get(url)
      if (response && response.data && response.data.data) {
        updateList.value = response.data.data.map((item: UpdateInfo) => {
          return {
            ...item,
            url: item.url || ''
          }
        })
      } else {
        updateList.value = []
        console.log('未获取到版本列表数据')
      }
    } catch (error) {
      console.error('加载最新版本列表错误:', error)
      updateList.value = []
    } finally {
      updateLoading.value = false
    }
  }

  const checkUpdateStatusOnLoad = async () => {
    try {
      const response = await http?.get(`${SERVER_URL}/update/progress`)

      if (response && response.data && response.data.code === 200) {
        const progressData = response.data.data
        if (progressData && (progressData.progress > 0 || progressData.status === 'downloading' || progressData.status === 'install')) {
          isUpdating.value = true
          updatingVersion.value = progressData.version || ''

          updateProgress.value = {
            progress: progressData.progress || 0,
            total_size: progressData.total_size || 0,
            downloaded: progressData.downloaded || 0,
            status: progressData.status || 'downloading'
          }

          startProgressPolling()
        }
      }
    } catch (error) {
      console.error('检查更新状态错误:', error)
    }
  }

  const updateToVersion = async (version: string) => {
    isUpdating.value = true
    updatingVersion.value = version
    updateProgress.value = {
      progress: 0,
      total_size: 0,
      downloaded: 0,
      status: 'downloading'
    }

    try {
      const response = await http?.post(`${SERVER_URL}/update/to-version`, {
        version: version
      })

      if (response && response.data && response.data.code === 200) {
        startProgressPolling()
      } else {
        isUpdating.value = false
        updatingVersion.value = ''
        updateProgress.value = {
          progress: 0,
          total_size: 0,
          downloaded: 0,
          status: ''
        }
        ElMessage.error(response?.data.message || '触发版本更新失败')
      }
    } catch (error) {
      console.error('触发版本更新错误:', error)
      isUpdating.value = false
      updatingVersion.value = ''
      updateProgress.value = {
        progress: 0,
        total_size: 0,
        downloaded: 0,
        status: ''
      }
      ElMessage.error('触发版本更新失败')
    }
  }

  const startProgressPolling = () => {
    if (progressTimer) {
      clearInterval(progressTimer)
    }

    checkUpdateProgress()

    progressTimer = setInterval(() => {
      checkUpdateProgress()
    }, 1000)
  }

  const stopProgressPolling = () => {
    if (progressTimer) {
      clearInterval(progressTimer)
      progressTimer = null
    }
  }

  const showUpdateCompleteNotification = () => {
    showUpdateCompleteDialog.value = true
    countdown.value = 30

    if (countdownTimer) {
      clearInterval(countdownTimer)
    }

    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        if (countdownTimer) {
          clearInterval(countdownTimer)
          countdownTimer = null
        }
        window.location.reload()
      }
    }, 1000)
  }

  const manuallyRefresh = () => {
    if (countdownTimer) {
      clearInterval(countdownTimer)
      countdownTimer = null
    }
    window.location.reload()
  }

  const checkUpdateProgress = async () => {
    try {
      const response = await http?.get(`${SERVER_URL}/update/progress`)

      if (response && response.data) {
        if (response.data.data.progress !== undefined) {
          const previousProgress = updateProgress.value.progress
          updateProgress.value.progress = response.data.data.progress

          if (previousProgress < 100 && updateProgress.value.progress >= 100) {
            showUpdateCompleteNotification()
          }
        }
        if (response.data.data.total_size !== undefined) {
          updateProgress.value.total_size = response.data.data.total_size
        }
        if (response.data.data.downloaded !== undefined) {
          updateProgress.value.downloaded = response.data.data.downloaded
        }

        if (response.data.data.status !== undefined) {
          updateProgress.value.status = response.data.data.status

          if (response.data.data.status === 'failed') {
            stopProgressPolling()

            if (countdownTimer) {
              clearInterval(countdownTimer)
              countdownTimer = null
            }

            showUpdateCompleteDialog.value = false

            isUpdating.value = false
            updatingVersion.value = ''
            updateProgress.value.progress = 0
            updateProgress.value.total_size = 0
            updateProgress.value.downloaded = 0
            updateProgress.value.status = ''

            ElMessage.error({
              message: '更新失败，请稍后重试或手动下载最新版本',
              duration: 5000
            })

            setTimeout(() => {
              loadUpdateList()
            }, 1000)

            return
          }
        }

        if (response.data.code !== 200) {
          stopProgressPolling()

          setTimeout(() => {
            isUpdating.value = false
            updatingVersion.value = ''
            loadUpdateList()
          }, 2000)
        }
      }
    } catch (error) {
      console.error('查询更新进度错误:', error)
    }
  }

  const cancelUpdate = async () => {
    try {
      await http?.post(`${SERVER_URL}/update/cancel`)

      stopProgressPolling()

      if (countdownTimer) {
        clearInterval(countdownTimer)
        countdownTimer = null
      }

      showUpdateCompleteDialog.value = false

      isUpdating.value = false
      updatingVersion.value = ''
      updateProgress.value = {
        progress: 0,
        total_size: 0,
        downloaded: 0,
        status: ''
      }

      ElMessage.success('已取消更新')

      setTimeout(() => {
        loadUpdateList()
      }, 1000)
    } catch (error) {
      console.error('取消更新错误:', error)
      ElMessage.error('取消更新失败，请稍后重试')
    }
  }

  const handleDownloadClick = (update: UpdateInfo) => {
    if (!update.url) {
      console.error('下载链接不存在:', update)
      alert('下载链接不存在，请稍后重试')
      return false
    }
    window.open(update.url, '_blank')
    return true
  }

  const cleanup = () => {
    stopProgressPolling()
    if (countdownTimer) {
      clearInterval(countdownTimer)
      countdownTimer = null
    }
  }

  onMounted(() => {
    loadUpdateList().then(() => {
      checkUpdateStatusOnLoad()
    })
  })

  onUnmounted(() => {
    cleanup()
  })

  return {
    updateList,
    updateLoading,
    isUpdating,
    updatingVersion,
    updateProgress,
    showUpdateCompleteDialog,
    countdown,
    loadUpdateList,
    updateToVersion,
    cancelUpdate,
    handleDownloadClick,
    manuallyRefresh
  }
}
