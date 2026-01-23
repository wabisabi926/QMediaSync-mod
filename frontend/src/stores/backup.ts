import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { AxiosStatic } from 'axios'
import type {
  BackupTaskType,
  BackupProgress,
  BackupConfig,
} from '@/typing'
import { SERVER_URL } from '@/const'

export const useBackupStore = defineStore('backup', () => {
  // 状态
  const progress = ref<BackupProgress | null>(null)
  const taskType = ref<BackupTaskType>(null)
  const taskId = ref<number | null>(null)
  const isMaintenanceMode = ref(false)
  const showProgressDialog = ref(false)
  const pollingTimer = ref<number | null>(null)
  const errorRetryCount = ref(0)
  const pollingStartTime = ref<number | null>(null)

  // 最大重试次数
  const MAX_RETRY_COUNT = 3
  // 最大轮询时长（65分钟，单位：毫秒）
  const MAX_POLLING_DURATION = 65 * 60 * 1000

  // 计算属性
  const isRunning = computed(() => progress.value?.running === true)
  const canCancel = computed(
    () => taskType.value === 'backup' && progress.value?.status === 'running'
  )

  // 检查备份状态
  const checkBackupStatus = async (http?: AxiosStatic) => {
    if (!http) return

    try {
      // 检查是否有正在运行的备份任务
      const progressRes = await http.get(`${SERVER_URL}/database/backup/progress`)
      if (progressRes.data.code === 200 && progressRes.data.data.running) {
        progress.value = progressRes.data.data
        taskType.value = 'backup'
        showProgressDialog.value = true
        startProgressPolling('backup', undefined, http)
        return
      }

      // 检查维护模式
      const configRes = await http.get(`${SERVER_URL}/database/backup-config`)
      if (configRes.data.code === 200 && configRes.data.data.exists) {
        const config: BackupConfig = configRes.data.data.config
        if (config.maintenance_mode === 1) {
          isMaintenanceMode.value = true
          taskType.value = 'restore'
          showProgressDialog.value = true
          startProgressPolling('restore', undefined, http)
        }
      }
    } catch (error) {
      console.error('检查备份状态失败:', error)
    }
  }

  // 开始轮询进度
  const startProgressPolling = (
    type: 'backup' | 'restore',
    id?: number,
    http?: AxiosStatic
  ) => {
    if (!http) return

    taskType.value = type
    if (id) taskId.value = id
    showProgressDialog.value = true
    errorRetryCount.value = 0
    pollingStartTime.value = Date.now()

    // 清除已存在的定时器
    if (pollingTimer.value) {
      clearInterval(pollingTimer.value)
    }

    // 立即执行一次
    pollProgress(http)

    // 每2秒轮询一次
    pollingTimer.value = window.setInterval(() => {
      pollProgress(http)
    }, 2000)
  }

  // 轮询进度
  const pollProgress = async (http: AxiosStatic) => {
    try {
      // 检查是否超过最大轮询时长
      if (pollingStartTime.value && taskType.value === 'restore') {
        const elapsedTime = Date.now() - pollingStartTime.value
        if (elapsedTime > MAX_POLLING_DURATION) {
          stopProgressPolling()
          ElMessageBox.alert(
            '数据库恢复操作已超过最大等待时间（65分钟），请手动刷新页面查看恢复结果。',
            '提示',
            {
              confirmButtonText: '刷新页面',
              callback: () => {
                location.reload()
              },
            }
          )
          return
        }
      }

      if (taskType.value === 'backup') {
        // 轮询备份进度
        const res = await http.get(`${SERVER_URL}/database/backup/progress`)
        if (res.data.code === 200) {
          progress.value = res.data.data
          errorRetryCount.value = 0 // 重置错误计数

          // 检查任务状态
          if (progress.value && !progress.value.running) {
            stopProgressPolling()
            handleTaskComplete(progress.value.status)
          }
        }
      } else if (taskType.value === 'restore') {
        // 轮询维护模式状态
        const res = await http.get(`${SERVER_URL}/database/backup-config`)
        if (res.data.code === 200 && res.data.data.exists) {
          const config: BackupConfig = res.data.data.config
          errorRetryCount.value = 0 // 重置错误计数

          if (config.maintenance_mode === 0) {
            // 维护模式已解除，恢复完成
            stopProgressPolling()
            isMaintenanceMode.value = false
            showProgressDialog.value = false
            ElMessage.success('数据库恢复成功！页面即将刷新...')
            setTimeout(() => {
              location.reload()
            }, 1500)
          } else {
            // 更新进度信息（如果有的话）
            const progressRes = await http.get(`${SERVER_URL}/database/backup/progress`)
            if (progressRes.data.code === 200 && progressRes.data.data.running) {
              progress.value = progressRes.data.data
            }
          }
        }
      }
    } catch (error) {
      console.error('轮询进度失败:', error)
      errorRetryCount.value++

      // 超过最大重试次数，自动刷新页面
      if (errorRetryCount.value >= MAX_RETRY_COUNT) {
        stopProgressPolling()
        ElMessage.error('网络连接失败，页面即将刷新...')
        setTimeout(() => {
          location.reload()
        }, 2000)
      }
    }
  }

  // 处理任务完成
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

    // 延迟关闭对话框
    setTimeout(() => {
      showProgressDialog.value = false
      resetState()
    }, 1500)
  }

  // 停止轮询
  const stopProgressPolling = () => {
    if (pollingTimer.value) {
      clearInterval(pollingTimer.value)
      pollingTimer.value = null
    }
    pollingStartTime.value = null
  }

  // 取消备份任务
  const cancelBackupTask = async (http: AxiosStatic) => {
    if (!taskId.value) {
      ElMessage.error('无法获取任务ID')
      return
    }

    try {
      const res = await http.post(`${SERVER_URL}/database/backup/cancel`, {
        task_id: taskId.value,
      })

      if (res.data.code === 200) {
        ElMessage.success('备份任务已取消')
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

  // 重置状态
  const resetState = () => {
    progress.value = null
    taskType.value = null
    taskId.value = null
    errorRetryCount.value = 0
    pollingStartTime.value = null
  }

  return {
    // 状态
    progress,
    taskType,
    taskId,
    isMaintenanceMode,
    showProgressDialog,
    errorRetryCount,

    // 计算属性
    isRunning,
    canCancel,

    // 方法
    checkBackupStatus,
    startProgressPolling,
    stopProgressPolling,
    cancelBackupTask,
    resetState,
  }
})
