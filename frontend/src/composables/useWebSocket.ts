import { ref, onBeforeUnmount } from 'vue'

export type WSEventType =
  | 'strm_sync_task_start'
  | 'strm_sync_task_queued'
  | 'strm_sync_task_complete'
  | 'sync_task_created'
  | 'sync_task_updated'
  | 'sync_task_deleted'
  | 'upload_queue_status_changed'
  | 'download_queue_status_changed'
  | 'upload_queue_changed'
  | 'download_queue_changed'

export interface WSEvent {
  event_type: WSEventType
  timestamp: string
  data: Record<string, unknown>
}

type EventCallback = (data: Record<string, unknown>) => void

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

export function on(eventType: WSEventType, callback: EventCallback): () => void {
  ensureConnection()
  if (!listeners.has(eventType)) {
    listeners.set(eventType, new Set())
  }
  listeners.get(eventType)!.add(callback)
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

export function off(eventType: WSEventType, callback: EventCallback) {
  const set = listeners.get(eventType)
  if (set) {
    set.delete(callback)
    if (set.size === 0) {
      listeners.delete(eventType)
    }
  }
}

export function useWSEvent(eventType: WSEventType, callback: EventCallback) {
  const unsubscribe = on(eventType, callback)

  onBeforeUnmount(() => {
    unsubscribe()
  })

  return { unsubscribe }
}
