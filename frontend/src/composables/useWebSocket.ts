import { ref, onBeforeUnmount } from 'vue'
import { useAuthStore } from '@/stores/auth'

// WebSocket 事件类型
export type WSEventType =
  | 'scraper_task_start'
  | 'scraper_task_complete'
  | 'scraper_item_complete'
  | 'strm_sync_task_start'
  | 'strm_sync_task_complete'

// WebSocket 事件结构
export interface WSEvent {
  event_type: WSEventType
  timestamp: string
  data: Record<string, unknown>
}

type EventCallback = (data: Record<string, unknown>) => void

// 全局单例状态
let wsInstance: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempts = 0
const MAX_RECONNECT_ATTEMPTS = 5
const BASE_RECONNECT_DELAY = 1000
const listeners = new Map<WSEventType, Set<EventCallback>>()
export const wsConnected = ref(false)

function getWsUrl(): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  return `${protocol}//${host}/api/events/ws`
}

function connect() {
  if (wsInstance && (wsInstance.readyState === WebSocket.OPEN || wsInstance.readyState === WebSocket.CONNECTING)) {
    return
  }

  const authStore = useAuthStore()
  const url = getWsUrl()

  try {
    wsInstance = new WebSocket(url)
  } catch {
    scheduleReconnect()
    return
  }

  wsInstance.onopen = () => {
    wsConnected.value = true
    reconnectAttempts = 0
    // 发送认证消息
    if (authStore.token) {
      wsInstance?.send(JSON.stringify({ type: 'auth', token: authStore.token }))
    }
  }

  wsInstance.onmessage = (event) => {
    try {
      const wsEvent: WSEvent = JSON.parse(event.data)
      const eventType = wsEvent.event_type as WSEventType
      const callbacks = listeners.get(eventType)
      if (callbacks) {
        callbacks.forEach(cb => cb(wsEvent.data))
      }
    } catch {
      // 忽略解析失败的消息
    }
  }

  wsInstance.onclose = () => {
    wsConnected.value = false
    wsInstance = null
    scheduleReconnect()
  }

  wsInstance.onerror = () => {
    wsConnected.value = false
  }
}

function scheduleReconnect() {
  if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
    return
  }
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
  }
  const delay = BASE_RECONNECT_DELAY * Math.pow(2, reconnectAttempts)
  reconnectAttempts++
  reconnectTimer = setTimeout(() => {
    connect()
  }, delay)
}

function ensureConnection() {
  if (!wsInstance || wsInstance.readyState === WebSocket.CLOSED) {
    connect()
  }
}

/**
 * 注册 WebSocket 事件监听
 * @param eventType 事件类型
 * @param callback 回调函数
 * @returns 注销函数
 */
export function on(eventType: WSEventType, callback: EventCallback): () => void {
  ensureConnection()
  if (!listeners.has(eventType)) {
    listeners.set(eventType, new Set())
  }
  listeners.get(eventType)!.add(callback)
  // 返回注销函数
  return () => {
    const set = listeners.get(eventType)
    if (set) {
      set.delete(callback)
      if (set.size === 0) {
        listeners.delete(eventType)
      }
    }
  }
}

/**
 * 注销 WebSocket 事件监听
 */
export function off(eventType: WSEventType, callback: EventCallback) {
  const set = listeners.get(eventType)
  if (set) {
    set.delete(callback)
    if (set.size === 0) {
      listeners.delete(eventType)
    }
  }
}

/**
 * Vue composable：自动管理 WebSocket 事件监听的生命周期
 * @param eventType 事件类型
 * @param callback 回调函数
 */
export function useWSEvent(eventType: WSEventType, callback: EventCallback) {
  const unsubscribe = on(eventType, callback)

  onBeforeUnmount(() => {
    unsubscribe()
  })

  return { unsubscribe }
}
