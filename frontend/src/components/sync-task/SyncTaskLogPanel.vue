<script setup lang="ts">
import LogActionToolbar from '@/components/log/LogActionToolbar.vue'
import type { SyncTaskLogEntry } from '@/types/syncTaskStream'
import { computed, nextTick, shallowRef, useTemplateRef, watch } from 'vue'

const props = defineProps<{
  logs: readonly SyncTaskLogEntry[]
  connected: boolean
  logPath: string
}>()

const emit = defineEmits<{
  connect: []
  disconnect: []
  clear: []
  download: []
}>()

const logsContainer = useTemplateRef<HTMLElement>('logsContainer')
const followLatest = shallowRef(true)
const visibleLogs = computed(() => props.logs)

const handleScroll = () => {
  if (!logsContainer.value) return
  followLatest.value = logsContainer.value.scrollTop <= 40
}

watch(
  () => props.logs[0]?.id,
  async () => {
    if (!followLatest.value) return
    await nextTick()
    if (logsContainer.value) {
      logsContainer.value.scrollTop = 0
    }
  },
)
</script>

<template>
  <div class="sync-task-log-panel">
    <div class="log-panel-header">
      <h3 class="log-panel-title">同步日志</h3>
      <LogActionToolbar
        compact
        :connected="connected"
        :download-disabled="!logPath"
        @connect="emit('connect')"
        @disconnect="emit('disconnect')"
        @clear="emit('clear')"
        @download="emit('download')"
      />
    </div>
    <div ref="logsContainer" class="logs" @scroll="handleScroll">
      <div
        v-for="log in visibleLogs"
        :key="log.id"
        class="log-line"
        :class="`log-level-${log.level}`"
      >
        <span class="log-timestamp">{{ log.timestamp }}</span>
        <span class="log-level">{{ log.level.toUpperCase() }}</span>
        <span class="log-message">{{ log.message }}</span>
      </div>
      <div v-if="visibleLogs.length === 0" class="empty-logs">暂无日志内容</div>
    </div>
    <div class="log-info">
      <el-text size="small">
        当前显示 {{ visibleLogs.length }} 行日志
        <span v-if="connected" class="status-indicator connected">● 已连接</span>
        <span v-else class="status-indicator disconnected">● 已断开</span>
      </el-text>
    </div>
  </div>
</template>

<style scoped>
.sync-task-log-panel {
  width: 100%;
}

.log-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.log-panel-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.logs {
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 10px;
  height: calc(100vh - 320px);
  min-height: 320px;
  overflow-y: auto;
  background-color: #fafafa;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  line-height: 1.5;
}

.log-line {
  display: flex;
  gap: 8px;
  padding: 2px 0;
  word-break: break-all;
  white-space: pre-wrap;
}

.log-timestamp {
  color: #909399;
  flex: 0 0 auto;
}

.log-level {
  flex: 0 0 52px;
  font-weight: 600;
  text-align: center;
}

.log-message {
  flex: 1;
  min-width: 0;
}

.log-level-info .log-level {
  color: #409eff;
}

.log-level-warn .log-level {
  color: #e6a23c;
}

.log-level-error .log-level {
  color: #f56c6c;
}

.log-level-debug .log-level {
  color: #909399;
}

.empty-logs {
  padding: 32px 0;
  color: #909399;
  text-align: center;
}

.log-info {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.status-indicator {
  margin-left: 12px;
}

.status-indicator.connected {
  color: #67c23a;
}

.status-indicator.disconnected {
  color: #909399;
}

@media (max-width: 768px) {
  .log-panel-header {
    align-items: stretch;
    flex-direction: column;
  }

  .logs {
    height: calc(100dvh - 300px);
    min-height: 360px;
    font-size: 12px;
  }

  .log-line {
    display: grid;
    grid-template-columns: 48px minmax(0, 1fr);
    gap: 2px 8px;
    padding: 4px 0;
  }

  .log-timestamp {
    grid-column: 1 / -1;
    font-size: 11px;
    white-space: normal;
    overflow-wrap: anywhere;
  }

  .log-level {
    flex: none;
    width: 48px;
    font-size: 11px;
  }

  .log-message {
    min-width: 0;
    overflow-wrap: anywhere;
  }
}
</style>
