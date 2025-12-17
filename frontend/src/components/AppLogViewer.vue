<template>
  <div class="log-viewer-container">
    <el-card class="log-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <h2 class="card-title">日志查看器</h2>
          <div class="header-actions">
            <!-- 只有实时日志模式才显示连接、断开、清空按钮 -->
            <template v-if="isRealTime">
              <el-button
                type="primary"
                @click="connect"
                size="small"
                :disabled="isConnected"
              >
                <el-icon v-if="!isConnected"><Connection /></el-icon>
                连接
              </el-button>
              <el-button
                type="danger"
                @click="disconnect"
                size="small"
                :disabled="!isConnected"
              >
                <el-icon v-if="isConnected"><Close /></el-icon>
                断开
              </el-button>
              <el-button
                type="info"
                @click="clearLogs"
                size="small"
              >
                <el-icon><Delete /></el-icon>
                清空
              </el-button>
            </template>
            <!-- 下载按钮始终显示 -->
            <el-button
              type="success"
              @click="downloadLogs"
              size="small"
            >
              <el-icon><Download /></el-icon>
              下载日志
            </el-button>
          </div>
        </div>
      </template>

      <div class="log-content">
        <!-- 日志显示区域 -->
        <div
          class="logs"
          id="logs"
          ref="logsContainer"
          @scroll="handleScroll"
          v-loading="loading"
        >
          <div
            v-for="(log, index) in logLines"
            :key="index"
            class="log-line"
            :class="`log-level-${log.level}`"
          >
            <span class="log-timestamp">{{ log.timestamp }}</span>
            <span class="log-level">{{ log.level.toUpperCase() }}</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
          <div v-if="logLines.length === 0" class="empty-logs">
            暂无日志内容
          </div>
        </div>

        <!-- 日志信息 -->
        <div class="log-info">
          <el-text size="small">
            当前显示 {{ logLines.length }} 行日志
            <span v-if="isConnected" class="status-indicator connected">● 已连接</span>
            <span v-else class="status-indicator disconnected">● 已断开</span>
          </el-text>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Connection, Close, Delete, Download } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { SERVER_URL } from '@/const'

// 定义组件属性
interface Props {
  logPath: string
  isRealTime: boolean
}

const props = withDefaults(defineProps<Props>(), {
  isRealTime: true
})

// 定义日志条目类型
interface LogEntry {
  level: 'info' | 'warn' | 'error' | 'debug'
  message: string
  timestamp: string
}

// 组件状态
const ws = ref<WebSocket | null>(null)
const logLines = ref<LogEntry[]>([])
const isConnected = ref(false)
const loading = ref(false)
const logsContainer = ref<HTMLElement | null>(null)

// 日志配置
let isLoadingOldLogs = false
let currentOffset = 0
let isAtTop = true
let lastScrollTop = 0
let isManualDisconnect = false
let hasReachedEnd = false

// WebSocket URL配置
let wsUrl = SERVER_URL.replace('http', 'ws')
wsUrl = wsUrl.replace('https', 'ws')
const WS_URL = `${wsUrl}/logs/ws`
const HTTP_URL = `${SERVER_URL}/logs/old`

// 监听日志路径变化，自动重新连接
watch(() => props.logPath, (newPath) => {
  if (newPath && isConnected.value) {
    reconnectWebSocket()
  }
})

// 初始化
onMounted(async () => {
  console.log("isRealTime:", props.isRealTime)
  // 如果提供了日志路径，先加载历史日志
  if (props.logPath) {
    // 设置初始偏移量为0
    currentOffset = 0
    // 清空现有日志
    logLines.value = []
    // 加载历史日志，设置limit为1000
    await loadInitialLogs()
    // 只有在实时日志模式下才建立WebSocket连接
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

  // 构建HTTP请求URL，加载前1000条日志
  const apiUrl = `${HTTP_URL}?path=${encodeURIComponent(logPath)}&pos=-1&direction=forward&limit=1000`

  try {
    const response = await fetch(apiUrl)
    if (!response.ok) {
      throw new Error('HTTP请求失败')
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
    console.error('加载初始日志失败:', error)
    addSystemLog(`加载初始日志失败: ${error instanceof Error ? error.message : '未知错误'}`, 'error')
  } finally {
    isLoadingOldLogs = false
    loading.value = false
  }
}

// 清理资源
onUnmounted(() => {
  disconnect()
})

// 处理滚动事件
const handleScroll = () => {
  if (!logsContainer.value) return

  const scrollTop = logsContainer.value.scrollTop
  const scrollHeight = logsContainer.value.scrollHeight
  const clientHeight = logsContainer.value.clientHeight

  // 判断是否在顶部
  isAtTop = scrollTop === 0

  // 检查是否需要重新连接WebSocket（当回到顶部时）
  if (isAtTop && lastScrollTop > 0) {
    reconnectWebSocket()
  }

  // 检查是否滚动到底部，需要加载更多旧日志
  if (scrollHeight - scrollTop - clientHeight < 50 && !isLoadingOldLogs && !hasReachedEnd) {
    loadOldLogs()
  }

  lastScrollTop = scrollTop
}

// 添加日志条目
const addLogEntry = (entry: LogEntry) => {
  // 添加到日志数组前面（最新的日志在最前面）
  logLines.value = [entry, ...logLines.value]
}

// 添加系统日志
const addSystemLog = (message: string, level: LogEntry['level'] = 'info') => {
  addLogEntry({
    level,
    message,
    timestamp: new Date().toISOString(),
  })
}

// 检查WebSocket是否已连接
const isWebSocketConnected = (): boolean => {
  return ws.value !== null && ws.value.readyState === WebSocket.OPEN
}

// 连接WebSocket
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

  // 构建WebSocket URL
  const wsUrl = `${WS_URL}?path=${encodeURIComponent(logPath)}`

  try {
    ws.value = new WebSocket(wsUrl)

    ws.value.onopen = () => {
      isConnected.value = true
      addSystemLog('WebSocket连接已建立', 'info')
    }

    ws.value.onmessage = (event) => {
      try {
        // 尝试解析JSON格式的日志条目
        const entry = JSON.parse(event.data) as LogEntry
        addLogEntry(entry)
      } catch (error) {
        // 如果不是JSON格式，作为纯文本处理
        console.error('解析日志条目失败:', error)
        addSystemLog(event.data, 'info')
      }
    }

    ws.value.onclose = () => {
      isConnected.value = false
      // 只有在主动断开连接时才显示关闭信息
      if (isManualDisconnect) {
        addSystemLog('WebSocket连接已关闭', 'info')
        isManualDisconnect = false
      }
    }

    ws.value.onerror = (event) => {
      let errorMsg = 'WebSocket错误: '
      if (event instanceof ErrorEvent && event.message) {
        errorMsg += event.message
      } else {
        errorMsg += JSON.stringify(event)
      }
      addSystemLog(errorMsg, 'error')
    }
  } catch (error) {
    addSystemLog(`连接失败: ${error instanceof Error ? error.message : '未知错误'}`, 'error')
  }
}

// 断开WebSocket连接
const disconnect = () => {
  if (ws.value) {
    isManualDisconnect = true
    ws.value.close()
    ws.value = null
  }
}

// 重新连接WebSocket
const reconnectWebSocket = () => {

  // 如果已经连接，不需要重新连接
  if (isWebSocketConnected()) {
    return
  }

  // 断开现有连接（不显示关闭信息）
  if (ws.value) {
    ws.value.close()
    ws.value = null
  }

  // 加载初始化日志
  loadInitialLogs()
  if (!props.isRealTime) {
    return
  }
  // 重新连接
  setTimeout(() => {
    connect()
  }, 500)
}

// 通过HTTP接口加载旧日志
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

  // 加载旧日志时断开WebSocket连接，避免新日志干扰
  if (isWebSocketConnected()) {
    ws.value?.close()
    ws.value = null
    isConnected.value = false
  }

  // 构建HTTP请求URL
  const apiUrl = `${HTTP_URL}?path=${encodeURIComponent(logPath)}&pos=${currentOffset}&direction=forward&limit=100`

  // 发送HTTP请求
  fetch(apiUrl)
    .then(response => {
      if (!response.ok) {
        throw new Error('HTTP请求失败')
      }
      return response.json() // 解析为JSON格式
    })
    .then(rs => {
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
    .catch(error => {
      console.error('加载旧日志失败:', error)
      addSystemLog(`加载旧日志失败: ${error instanceof Error ? error.message : '未知错误'}`, 'error')
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
  const logPath = props.logPath.trim()
  if (!logPath) {
    ElMessage.error('请输入日志文件路径')
    return
  }

  try {
    // 构建下载链接
    const downloadUrl = `${SERVER_URL}/logs/download?path=${encodeURIComponent(logPath)}`

    // 创建一个临时的a标签来触发下载
    const link = document.createElement('a')
    link.href = downloadUrl
    link.download = logPath.split('/').pop() || 'logfile.log' // 使用文件名作为下载文件名
    link.target = '_blank'

    // 触发下载
    document.body.appendChild(link)
    link.click()

    // 清理
    setTimeout(() => {
      document.body.removeChild(link)
    }, 100)

    ElMessage.success('开始下载日志文件')
  } catch (error) {
    console.error('下载日志失败:', error)
    ElMessage.error(`下载日志失败: ${error instanceof Error ? error.message : '未知错误'}`)
  }
}

// 暴露方法给父组件
defineExpose({
  disconnect
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
  gap: 20px;
}

.card-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.header-actions {
  display: flex;
  gap: 8px;
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

  .log-timestamp {
    min-width: 140px;
    margin-right: 8px;
  }

  .log-level {
    min-width: 50px;
    margin-right: 8px;
    font-size: 10px;
  }
}
</style>
