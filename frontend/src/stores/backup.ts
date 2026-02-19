import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import type { AxiosStatic } from 'axios'
import type {
  BackupTaskType,
  BackupProgress,
} from '@/typing'
import { SERVER_URL } from '@/const'

export const useBackupStore = defineStore('backup', () => {
  const progress = ref<BackupProgress | null>(null)
  const taskType = ref<BackupTaskType>(null)
  const showProgressDialog = ref(false)
  const pollingTimer = ref<number | null>(null)
  const errorRetryCount = ref(0)

  const MAX_RETRY_COUNT = 3
  const API_SUCCESS_CODE = 0

  const isRunning = computed(() => progress.value?.running === true)
  const canCancel = computed(
    () => taskType.value === 'backup' && progress.value?.status === 'running'
  )

  const checkBackupStatus = async (http?: AxiosStatic) => {
    if (!http) return

    try {
      const res = await http.get(`${SERVER_URL}/backup/status`)
      if (res.data.code === API_SUCCESS_CODE && res.data.data.is_running) {
        progress.value = { running: true, status: 'running' }
        taskType.value = 'backup'
        showProgressDialog.value = true
        startProgressPolling('backup', undefined, http)
      }
    } catch (error) {
      console.error('检查备份状态失败:', error)
    }
  }

  const startProgressPolling = (
    type: 'backup' | 'restore',
    id?: number,
    http?: AxiosStatic
  ) => {
    if (!http) return

    taskType.value = type
    showProgressDialog.value = true
    errorRetryCount.value = 0

    if (pollingTimer.value) {
      clearInterval(pollingTimer.value)
    }

    pollProgress(http)

    pollingTimer.value = window.setInterval(() => {
      pollProgress(http)
    }, 2000)
  }

  const pollProgress = async (http: AxiosStatic) => {
    try {
      if (taskType.value === 'backup') {
        const res = await http.get(`${SERVER_URL}/backup/status`)
        if (res.data.code === API_SUCCESS_CODE) {
          const statusData = res.data.data
          progress.value = {
            running: statusData.is_running,
            status: statusData.is_running ? 'running' : 'completed',
          }
          errorRetryCount.value = 0

          if (!statusData.is_running) {
            stopProgressPolling()
            handleTaskComplete(progress.value.status)
          }
        }
      } else if (taskType.value === 'restore') {
        const res = await http.get(`${SERVER_URL}/backup/status`)
        if (res.data.code === API_SUCCESS_CODE) {
          const statusData = res.data.data
          errorRetryCount.value = 0

          if (!statusData.is_running) {
            stopProgressPolling()
            showProgressDialog.value = false
            ElMessage.success('数据库恢复成功！页面即将刷新...')
            setTimeout(() => {
              location.reload()
            }, 1500)
          } else {
            progress.value = { running: true, status: 'running' }
          }
        }
      }
    } catch (error) {
      console.error('轮询进度失败:', error)
      errorRetryCount.value++

      if (errorRetryCount.value >= MAX_RETRY_COUNT) {
        stopProgressPolling()
        ElMessage.error('网络连接失败，页面即将刷新...')
        setTimeout(() => {
          location.reload()
        }, 2000)
      }
    }
  }

  const handleTaskComplete = (status?: string) => {
    switch (status) {
      case 'completed':
        ElMessage.success('备份任务完成！')
        break
      case 'cancelled':
        ElMessage.info('备份任务已取消')
        break
      case 'timeout':
        ElMessage.warning('备份任务超时')
        break
      case 'failed':
        ElMessage.error('备份任务失败')
        break
    }

    setTimeout(() => {
      showProgressDialog.value = false
      resetState()
    }, 1500)
  }

  const stopProgressPolling = () => {
    if (pollingTimer.value) {
      clearInterval(pollingTimer.value)
      pollingTimer.value = null
    }
  }

  const cancelBackupTask = async (http: AxiosStatic) => {
    try {
      const res = await http.post(`${SERVER_URL}/backup/cancel`)

      if (res.data.code === API_SUCCESS_CODE) {
        ElMessage.success(res.data.message || '备份任务已取消')
        stopProgressPolling()
        showProgressDialog.value = false
        resetState()
      } else {
        ElMessage.error(res.data.message || '取消备份任务失败')
      }
    } catch (error: unknown) {
      const errorMsg = error instanceof Error ? error.message : '取消备份任务失败'
      ElMessage.error(errorMsg)
    }
  }

  const resetState = () => {
    progress.value = null
    taskType.value = null
    errorRetryCount.value = 0
  }

  return {
    progress,
    taskType,
    showProgressDialog,
    errorRetryCount,
    isRunning,
    canCancel,
    checkBackupStatus,
    startProgressPolling,
    stopProgressPolling,
    cancelBackupTask,
    resetState,
  }
})
