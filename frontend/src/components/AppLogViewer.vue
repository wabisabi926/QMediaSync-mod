<template>
  <div class="log-viewer-container">
    <el-card class="log-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <h2 class="card-title">日志查看器</h2>
          <LogActionToolbar
            :connected="isConnected"
            :show-realtime-controls="isRealTime"
            :download-disabled="!props.logPath.trim()"
            @connect="connect"
            @disconnect="disconnect"
            @clear="clearLogs"
            @download="downloadLogs"
          />
        </div>
      </template>

      <div class="log-content">
        <!-- 日志显示区域 -->
        <div class="logs" id="logs" ref="logsContainer" @scroll="handleScroll" v-loading="loading">
          <div
            v-for="(log, index) in limitedLogLines"
            :key="log.id || index"
            class="log-line"
            :class="`log-level-${log.level}`"
          >
            <span class="log-timestamp">{{ log.timestamp }}</span>
            <span class="log-level">{{ log.level.toUpperCase() }}</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
          <div v-if="logLines.length === 0" class="empty-logs">暂无日志内容</div>
        </div>

        <!-- 日志信息 -->
        <div class="log-info">
          <el-text size="small">
            当前显示 {{ limitedLogLines.length }} 行日志
            <span v-if="isConnected" class="status-indicator connected">● 已连接</span>
            <span v-else class="status-indicator disconnected">● 已断开</span>
          </el-text>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, useTemplateRef } from 'vue'
import LogActionToolbar from '@/components/log/LogActionToolbar.vue'
import { useLogFileActions } from '@/composables/useLogFileActions'
import { SERVER_URL } from '@/const'
import { formatDateTime } from '@/utils/timeUtils'
import { buildApiWebSocketUrl } from '@/utils/wsUrl'

// 定义组件属性
interface Props {
  logPath: string
  isRealTime: boolean
}

const props = withDefaults(defineProps<Props>(), {
  isRealTime: true,
})

// 定义日志条目类型
interface LogEntry {
  level: 'info' | 'warn' | 'error' | 'debug'
  message: string
  timestamp: string
  id?: string
}

// 组件状态
const ws = ref<WebSocket | null>(null)
const logLines = ref<LogEntry[]>([])
const isConnected = ref(false)
const loading = ref(false)
const logsContainer = useTemplateRef<HTMLElement>('logsContainer')
const { downloadLogFile } = useLogFileActions()

// 日志数量限制配置
const MAX_LOG_ENTRIES = 2000
const CLEANUP_THRESHOLD = 2500
let cleanupTimer: ReturnType<typeof setTimeout> | null = null

// 日志配置
let isLoadingOldLogs = false
let currentOffset = 0
let isAtTop = true
let lastScrollTop = 0
let isManualDisconnect = false
let hasReachedEnd = false
let followLatest = true

// WebSocket URL 配置
const WS_URL = buildApiWebSocketUrl(SERVER_URL, '/logs/ws')
const HTTP_URL = `${SERVER_URL}/logs/old`

const readLogResponseError = async (response: Response) => {
  try {
    const body = await response.json()
    if (body?.error) return String(body.error)
  } catch {
    // 忽略非 JSON 错误响应
  }
  return 'HTTP 请求失败'
}

// 限制显示的日志条目
const limitedLogLines = computed(() => {
  return logLines.value.slice(0, MAX_LOG_ENTRIES)
})

const resetLogState = () => {
  currentOffset = 0
  hasReachedEnd = false
  isAtTop = true
  lastScrollTop = 0
  followLatest = true
  logLines.value = []
}

// 监听日志路径和实时模式变化，自动维护连接
watch(
  () => [props.logPath, props.isRealTime] as const,
  async ([logPath, isRealTime], oldValue) => {
    const oldLogPath = oldValue?.[0] ?? ''
    const normalizedLogPath = logPath.trim()
    if (logPath !== oldLogPath) {
      disconnect({ silent: true })
      resetLogState()
      if (normalizedLogPath) {
        await loadInitialLogs()
      }
    }
    if (isRealTime && normalizedLogPath && !isWebSocketConnected()) {
      connect()
    }
    if (!isRealTime && ws.value) {
      disconnect({ silent: true })
    }
  },
)

// 初始化
onMounted(async () => {
  // 如果提供了日志路径，先加载历史日志
  if (props.logPath) {
    resetLogState()
    // 加载历史日志，设置 limit 为 1000
    await loadInitialLogs()
    // 只有在实时日志模式下才建立 WebSocket 连接
    if (props.isRealTime) {
      connect()
    }
  }
})

// 加载初始历史日志
const loadInitialLogs = async () => {
  const logPath = props.logPath.trim()
  if (!logPath) {
    return
  }

  isLoadingOldLogs = true
  loading.value = true

  // 构建 HTTP 请求 URL，加载前 1000 条日志
  const apiUrl = `${HTTP_URL}?path=${encodeURIComponent(logPath)}&pos=-1&direction=forward&limit=1000`

  try {
    const response = await fetch(apiUrl, { credentials: 'include' })
    if (!response.ok) {
      throw new Error(await readLogResponseError(response))
    }
    const rs = await response.json()
    const entries = rs.entries || []
    currentOffset = rs.pos
    // 处理返回的旧日志条目
    if (Array.isArray(entries) && entries.length > 0) {
      // 添加到日志数组（旧日志在后面）
      logLines.value = [...entries]
    }
  } catch (error) {
    console.error('加载初始日志失败：', error)
    addSystemLog(
      `加载初始日志失败：${error instanceof Error ? error.message : '未知错误'}`,
      'error',
    )
  } finally {
    isLoadingOldLogs = false
    loading.value = false
  }
}

// 清理资源
onUnmounted(() => {
  disconnect({ silent: true })
  if (cleanupTimer) {
    clearTimeout(cleanupTimer)
  }
})

// 处理滚动事件
const handleScroll = () => {
  if (!logsContainer.value) return

  const scrollTop = logsContainer.value.scrollTop
  const scrollHeight = logsContainer.value.scrollHeight
  const clientHeight = logsContainer.value.clientHeight

  // 判断是否在顶部
  isAtTop = scrollTop === 0
  followLatest = scrollTop <= 20

  // 检查是否需要重新连接 WebSocket（当回到顶部时）
  if (isAtTop && lastScrollTop > 0) {
    reconnectWebSocket()
  }

  // 检查是否滚动到底部，需要加载更多旧日志
  if (scrollHeight - scrollTop - clientHeight < 50 && !isLoadingOldLogs && !hasReachedEnd) {
    loadOldLogs()
  }

  lastScrollTop = scrollTop
}

// 防抖清理函数
const debouncedCleanupLogs = () => {
  if (cleanupTimer) {
    clearTimeout(cleanupTimer)
  }

  cleanupTimer = setTimeout(() => {
    if (logLines.value.length > CLEANUP_THRESHOLD) {
      logLines.value = logLines.value.slice(0, MAX_LOG_ENTRIES)
    }
  }, 300)
}

// 添加日志条目
const addLogEntry = (entry: LogEntry) => {
  // 为日志条目添加唯一 ID
  const entryWithId = {
    ...entry,
    id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
  }

  // 添加到日志数组前面（最新的日志在最前面）
  logLines.value = [entryWithId, ...logLines.value]

  // 防抖清理检查
  debouncedCleanupLogs()

  if (followLatest) {
    requestAnimationFrame(() => {
      if (logsContainer.value) {
        logsContainer.value.scrollTop = 0
      }
    })
  }
}

// 添加系统日志
const addSystemLog = (message: string, level: LogEntry['level'] = 'info') => {
  addLogEntry({
    level,
    message,
    timestamp: formatDateTime(Math.floor(Date.now() / 1000)),
  })
}

// 检查 WebSocket 是否已连接
const isWebSocketConnected = (): boolean => {
  return ws.value !== null && ws.value.readyState === WebSocket.OPEN
}

// 连接 WebSocket
const connect = () => {
  // 如果已经连接，直接返回
  if (isWebSocketConnected()) {
    return
  }

  const logPath = props.logPath.trim()
  if (!logPath) {
    addSystemLog('请输入日志文件路径', 'error')
    return
  }

  // 构建 WebSocket URL
  if (ws.value) {
    disconnect({ silent: true })
  }
  const wsUrl = `${WS_URL}?path=${encodeURIComponent(logPath)}`

  try {
    isManualDisconnect = false
    ws.value = new WebSocket(wsUrl)

    ws.value.onopen = () => {
      isConnected.value = true
      addSystemLog('WebSocket 连接已建立', 'info')
    }

    ws.value.onmessage = (event) => {
      try {
        // 尝试解析 JSON 格式的日志条目
        const entry = JSON.parse(event.data) as LogEntry
        addLogEntry(entry)
      } catch (error) {
        // 如果不是 JSON 格式，作为纯文本处理
        console.error('解析日志条目失败：', error)
        addSystemLog(event.data, 'info')
      }
    }

    ws.value.onclose = () => {
      isConnected.value = false
      // 只有在主动断开连接时才显示关闭信息
      if (isManualDisconnect) {
        addSystemLog('WebSocket 连接已关闭', 'info')
        isManualDisconnect = false
      }
    }

    ws.value.onerror = (event) => {
      let errorMsg = 'WebSocket 错误：'
      if (event instanceof ErrorEvent && event.message) {
        errorMsg += event.message
      } else {
        errorMsg += JSON.stringify(event)
      }
      addSystemLog(errorMsg, 'error')
    }
  } catch (error) {
    addSystemLog(`连接失败：${error instanceof Error ? error.message : '未知错误'}`, 'error')
  }
}

// 断开 WebSocket 连接
const disconnect = (options: { silent?: boolean } = {}) => {
  if (ws.value) {
    isManualDisconnect = !options.silent
    ws.value.close()
    ws.value = null
  }
  isConnected.value = false
}

// 重新连接 WebSocket
const reconnectWebSocket = () => {
  // 如果已经连接，不需要重新连接
  if (isWebSocketConnected()) {
    return
  }

  disconnect({ silent: true })
  if (!props.isRealTime) {
    return
  }
  // 重新连接
  setTimeout(() => {
    connect()
  }, 500)
}

// 通过 HTTP 接口加载旧日志
const loadOldLogs = () => {
  const logPath = props.logPath.trim()
  if (!logPath) {
    return
  }

  // 如果已经到达日志文件末尾，不再加载
  if (hasReachedEnd) {
    return
  }

  isLoadingOldLogs = true
  loading.value = true

  // 加载旧日志时断开 WebSocket 连接，避免新日志干扰
  if (isWebSocketConnected()) {
    disconnect({ silent: true })
  }

  // 构建 HTTP 请求 URL
  const apiUrl = `${HTTP_URL}?path=${encodeURIComponent(logPath)}&pos=${currentOffset}&direction=forward&limit=100`

  // 发送 HTTP 请求
  fetch(apiUrl, { credentials: 'include' })
    .then(async (response) => {
      if (!response.ok) {
        throw new Error(await readLogResponseError(response))
      }
      return response.json() // 解析为 JSON 格式
    })
    .then((rs) => {
      const entries = rs.entries || []
      currentOffset = rs.pos
      // 处理返回的旧日志条目
      if (Array.isArray(entries) && entries.length > 0) {
        // 直接添加新日志
        logLines.value = [...logLines.value, ...entries]
      } else {
        // 返回数据为空，说明已经到达日志文件末尾
        hasReachedEnd = true
      }
    })
    .catch((error) => {
      console.error('加载旧日志失败：', error)
      addSystemLog(
        `加载旧日志失败：${error instanceof Error ? error.message : '未知错误'}`,
        'error',
      )
    })
    .finally(() => {
      isLoadingOldLogs = false
      loading.value = false
    })
}

// 清空日志
const clearLogs = () => {
  logLines.value = []
}

// 下载日志文件
const downloadLogs = () => {
  downloadLogFile(props.logPath, {
    emptyMessage: '请输入日志文件路径',
    errorPrefix: '下载日志失败',
  })
}

// 暴露方法给父组件
defineExpose({
  disconnect,
})
</script>

<style scoped>
.log-viewer-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.log-card {
  width: 100%;
  max-width: none;
  margin: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 20px;
}

.card-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.log-content {
  margin-top: 20px;
}

.logs {
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 10px;
  height: calc(100vh - 320px);
  overflow-y: auto;
  background-color: #fafafa;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  line-height: 1.5;
}

.log-line {
  padding: 2px 0;
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  word-break: break-all;
  white-space: pre-wrap;
}

.log-timestamp {
  color: #909399;
  margin-right: 12px;
  min-width: 180px;
  flex-shrink: 0;
}

.log-level {
  font-weight: bold;
  margin-right: 12px;
  min-width: 60px;
  text-align: center;
  border-radius: 3px;
  padding: 0 4px;
  font-size: 11px;
  line-height: 16px;
  flex-shrink: 0;
}

.log-level-info {
  color: #606266;
}

.log-level-info .log-level {
  background-color: #ecf5ff;
  color: #409eff;
}

.log-level-warn {
  color: #e6a23c;
}

.log-level-warn .log-level {
  background-color: #fdf6ec;
  color: #e6a23c;
}

.log-level-error {
  color: #f56c6c;
}

.log-level-error .log-level {
  background-color: #fef0f0;
  color: #f56c6c;
}

.log-level-debug {
  color: #909399;
}

.log-level-debug .log-level {
  background-color: #f4f4f5;
  color: #909399;
}

.log-message {
  flex: 1;
  color: #303133;
}

.empty-logs {
  text-align: center;
  color: #909399;
  padding: 20px 0;
}

.log-info {
  margin-top: 10px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.status-indicator {
  margin-left: 10px;
  font-size: 12px;
  font-weight: 500;
}

.status-indicator.connected {
  color: #67c23a;
}

.status-indicator.disconnected {
  color: #f56c6c;
}

/* 滚动条样式 */
.logs::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.logs::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 4px;
}

.logs::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 4px;
}

.logs::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .card-title {
    font-size: 18px;
  }

  .logs {
    height: 300px;
    font-size: 12px;
  }

  .log-line {
    flex-wrap: wrap;
  }

  .log-timestamp {
    min-width: auto;
    width: 100%;
    margin-right: 0;
    margin-bottom: 2px;
    font-size: 11px;
  }

  .log-level {
    min-width: 40px;
    margin-right: 8px;
    font-size: 10px;
  }

  .log-message {
    width: calc(100% - 60px);
  }
}
</style>
