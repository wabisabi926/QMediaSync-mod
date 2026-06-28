import { SERVER_URL } from '@/const'
import type {
  SyncTask,
  SyncTaskEventPayload,
  SyncTaskLogEntry,
  SyncTaskSnapshot,
  SyncTaskStreamMessage,
} from '@/types/syncTaskStream'
import { buildApiWebSocketUrl } from '@/utils/wsUrl'
import {
  computed,
  onBeforeUnmount,
  readonly,
  shallowRef,
  toRef,
  toValue,
  watch,
  type MaybeRefOrGetter,
} from 'vue'

interface UseSyncTaskStreamOptions {
  immediate?: boolean
  reconnect?: boolean
  maxLogs?: number
}

export function useSyncTaskStream(
  syncId: MaybeRefOrGetter<number | string>,
  options: UseSyncTaskStreamOptions = {},
) {
  const { immediate = true, reconnect = true, maxLogs = 2000 } = options
  const task = shallowRef<SyncTask | null>(null)
  const logs = shallowRef<SyncTaskLogEntry[]>([])
  const loading = shallowRef(false)
  const connected = shallowRef(false)
  const terminal = shallowRef(false)
  const errorMessage = shallowRef('')
  const lastSequence = shallowRef(0)
  const logCursor = shallowRef(0)
  const logPath = shallowRef('')
  const reconnectAttempts = shallowRef(0)
  const socket = shallowRef<WebSocket | null>(null)
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let intentionalClose = false

  const isRunning = computed(() => task.value?.status === 0 || task.value?.status === 1)

  const connect = () => {
    const currentId = Number(toValue(syncId))
    if (!currentId || terminal.value) return
    closeSocket({ intentional: true })
    loading.value = !task.value
    errorMessage.value = ''
    intentionalClose = false
    const url = buildApiWebSocketUrl(SERVER_URL, `/sync/tasks/${currentId}/stream`)
    const ws = new WebSocket(url)
    socket.value = ws
    ws.onopen = () => {
      connected.value = true
      reconnectAttempts.value = 0
    }
    ws.onmessage = (event) => {
      try {
        handleMessage(JSON.parse(event.data) as SyncTaskStreamMessage)
      } catch {
        errorMessage.value = '同步任务实时数据解析失败'
      }
    }
    ws.onclose = () => {
      connected.value = false
      socket.value = null
      loading.value = false
      if (!intentionalClose && reconnect && !terminal.value) scheduleReconnect()
      intentionalClose = false
    }
    ws.onerror = () => {
      errorMessage.value = '同步任务实时连接异常'
    }
  }

  const disconnect = () => {
    closeSocket({ intentional: true })
  }

  const closeSocket = (options: { intentional: boolean }) => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (socket.value) {
      const currentSocket = socket.value
      intentionalClose = options.intentional
      currentSocket.onclose = null
      currentSocket.onerror = null
      currentSocket.onmessage = null
      currentSocket.close()
      socket.value = null
    }
    connected.value = false
  }

  const scheduleReconnect = () => {
    if (reconnectAttempts.value >= 5) {
      errorMessage.value = '同步任务实时连接已断开'
      return
    }
    const delay = 1000 * Math.pow(2, reconnectAttempts.value)
    reconnectAttempts.value += 1
    reconnectTimer = setTimeout(connect, delay)
  }

  const handleMessage = (message: SyncTaskStreamMessage) => {
    if (message.type === 'snapshot') {
      applySnapshot(message.data as SyncTaskSnapshot)
      return
    }
    if (message.type === 'task_patch' || message.type === 'complete') {
      applyTaskPatch(message.data as SyncTaskEventPayload)
      if (message.type === 'complete') terminal.value = true
      return
    }
    if (message.type === 'log_append') {
      const data = message.data as { entry?: SyncTaskLogEntry; cursor?: number }
      if (data.entry) appendLog(data.entry)
      if (typeof data.cursor === 'number') logCursor.value = data.cursor
      return
    }
    if (message.type === 'resync_required') {
      connect()
      return
    }
    if (message.type === 'error') {
      errorMessage.value = '同步任务实时流返回错误'
    }
  }

  const applySnapshot = (snapshot: SyncTaskSnapshot) => {
    task.value = snapshot.task
    logs.value = normalizeSnapshotLogs(snapshot.logs)
    logCursor.value = snapshot.log_cursor
    logPath.value = snapshot.log_path
    loading.value = false
    terminal.value = snapshot.task.status === 2 || snapshot.task.status === 3
  }

  const applyTaskPatch = (payload: SyncTaskEventPayload) => {
    if (payload.sequence && payload.sequence <= lastSequence.value) return
    lastSequence.value = payload.sequence || lastSequence.value
    if (payload.deleted) {
      terminal.value = true
      errorMessage.value = '同步记录已删除'
      return
    }
    if (!task.value) return
    task.value = {
      ...task.value,
      status: payload.status as SyncTask['status'],
      sub_status: payload.sub_status as SyncTask['sub_status'],
      total: payload.total,
      new_strm: payload.new_strm,
      new_meta: payload.new_meta,
      new_upload: payload.new_upload,
      finish_at: payload.finish_at,
      net_file_start_at: payload.net_file_start_at,
      net_file_finish_at: payload.net_file_finish_at,
      local_file_start_at: payload.local_file_start_at,
      local_file_finish_at: payload.local_file_finish_at,
      updated_at: payload.updated_at || task.value.updated_at,
      fail_reason: payload.fail_reason ?? task.value.fail_reason,
    }
  }

  const appendLog = (entry: SyncTaskLogEntry) => {
    logs.value = [withLogID(entry), ...logs.value].slice(0, maxLogs)
  }

  const normalizeSnapshotLogs = (entries: SyncTaskLogEntry[]) => {
    return entries.map(withLogID).reverse().slice(0, maxLogs)
  }

  const clearLogs = () => {
    logs.value = []
  }

  const withLogID = (entry: SyncTaskLogEntry): SyncTaskLogEntry => ({
    ...entry,
    id: entry.id || `${entry.cursor || Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
  })

  watch(
    toRef(syncId),
    () => {
      task.value = null
      logs.value = []
      terminal.value = false
      lastSequence.value = 0
      logCursor.value = 0
      logPath.value = ''
      if (immediate) connect()
    },
    { immediate },
  )

  onBeforeUnmount(disconnect)

  return {
    task: readonly(task),
    logs: readonly(logs),
    loading: readonly(loading),
    connected: readonly(connected),
    terminal: readonly(terminal),
    errorMessage: readonly(errorMessage),
    isRunning,
    logCursor: readonly(logCursor),
    logPath: readonly(logPath),
    clearLogs,
    reconnect: connect,
    disconnect,
  }
}
