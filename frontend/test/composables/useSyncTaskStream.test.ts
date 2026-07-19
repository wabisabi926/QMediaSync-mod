// @vitest-environment happy-dom

import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick, shallowRef, type App } from 'vue'

vi.mock('@/const', () => ({
  SERVER_URL: 'https://api.example.test',
}))

import { useSyncTaskStream } from '@/composables/useSyncTaskStream'

const mountedApps: App[] = []

const withSetup = <T>(composable: () => T): T => {
  let result!: T
  const app = createApp({
    setup() {
      result = composable()
      return () => null
    },
  })
  app.mount(document.createElement('div'))
  mountedApps.push(app)
  return result
}

type Listener = (event: MessageEvent<string>) => void

class MockEventSource {
  static instances: MockEventSource[] = []
  onopen: (() => void) | null = null
  onerror: ((event: Event) => void) | null = null
  private readonly listeners = new Map<string, Set<Listener>>()
  closed = false

  constructor(public readonly url: string) {
    MockEventSource.instances.push(this)
  }

  addEventListener(type: string, listener: Listener) {
    const listeners = this.listeners.get(type) ?? new Set<Listener>()
    listeners.add(listener)
    this.listeners.set(type, listeners)
  }

  removeEventListener(type: string, listener: Listener) {
    this.listeners.get(type)?.delete(listener)
  }

  emit(type: string, data: unknown) {
    const event = { data: JSON.stringify(data) } as MessageEvent<string>
    if (type === 'error') this.onerror?.(event)
    this.listeners.get(type)?.forEach((listener) => listener(event))
  }

  close() {
    this.closed = true
  }
}

const snapshot = {
  task: {
    id: 8,
    sync_path_id: 2,
    created_at: 1,
    updated_at: 1,
    finish_at: 0,
    status: 1,
    sub_status: 1,
    total: 2,
    new_strm: 1,
    new_meta: 0,
    new_upload: 0,
    net_file_start_at: 0,
    net_file_finish_at: 0,
    local_file_start_at: 0,
    local_file_finish_at: 0,
    local_path: '/media',
    remote_path: '/cloud',
    fail_reason: '',
  },
  logs: [{ level: 'info', message: 'start', timestamp: 't', cursor: 12 }],
  log_cursor: 12,
  log_path: 'libs/sync_8.log',
}

describe('useSyncTaskStream', () => {
  afterEach(() => {
    mountedApps.splice(0).forEach((app) => app.unmount())
    vi.unstubAllGlobals()
    MockEventSource.instances = []
  })

  it('applies snapshot, patches and log events from EventSource', async () => {
    vi.stubGlobal('EventSource', MockEventSource)
    const syncId = shallowRef(8)
    const stream = withSetup(() => useSyncTaskStream(syncId))
    await nextTick()
    const source = MockEventSource.instances[0]

    expect(source.url).toBe('/api/sync/tasks/8/stream')
    expect(stream.connectionState.value).toBe('connecting')
    source.onopen?.()
    expect(stream.connectionState.value).toBe('connected')
    source.emit('snapshot', { type: 'snapshot', version: 1, sync_id: 8, data: snapshot })
    source.emit('task_patch', {
      type: 'task_patch',
      version: 1,
      sync_id: 8,
      data: { sync_id: 8, status: 1, sub_status: 2, total: 2, net_file_start_at: 101 },
    })
    source.emit('log_append', {
      type: 'log_append',
      version: 1,
      sync_id: 8,
      data: { entry: { level: 'info', message: 'next', timestamp: 't2', cursor: 30 }, cursor: 30 },
    })

    await nextTick()
    expect(stream.task.value?.sub_status).toBe(2)
    expect(stream.task.value?.net_file_start_at).toBe(101)
    expect(stream.logs.value.map((line) => line.message)).toEqual(['next', 'start'])
    expect(stream.connected.value).toBe(true)
  })

  it('enters reconnecting only after a native EventSource error', async () => {
    vi.stubGlobal('EventSource', MockEventSource)
    const stream = withSetup(() => useSyncTaskStream(8))
    await nextTick()
    const source = MockEventSource.instances[0]

    expect(stream.connectionState.value).toBe('connecting')
    source.onerror?.(new Event('error'))

    expect(stream.connectionState.value).toBe('reconnecting')
    expect(stream.connected.value).toBe(false)
  })

  it('closes terminal sources and ignores callbacks from a replaced source', async () => {
    vi.stubGlobal('EventSource', MockEventSource)
    const syncId = shallowRef(8)
    const stream = withSetup(() => useSyncTaskStream(syncId))
    await nextTick()
    const source = MockEventSource.instances[0]

    syncId.value = 9
    await nextTick()
    const replacement = MockEventSource.instances[1]
    source.emit('snapshot', { type: 'snapshot', version: 1, sync_id: 8, data: snapshot })
    replacement.emit('snapshot', {
      type: 'snapshot',
      version: 1,
      sync_id: 9,
      data: { ...snapshot, task: { ...snapshot.task, id: 9 } },
    })
    replacement.emit('complete', {
      type: 'complete',
      version: 1,
      sync_id: 9,
      data: { sync_id: 9, status: 2 },
    })

    await nextTick()
    expect(source.closed).toBe(true)
    expect(stream.task.value?.id).toBe(9)
    expect(replacement.closed).toBe(true)
    expect(stream.terminal.value).toBe(true)
  })

  it('falls back to running-task HTTP polling when EventSource is unsupported', async () => {
    vi.stubGlobal('EventSource', undefined)
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: vi.fn().mockResolvedValue({ data: { ...snapshot.task } }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const stream = withSetup(() => useSyncTaskStream(8))
    await nextTick()
    await Promise.resolve()

    expect(MockEventSource.instances).toHaveLength(0)
    expect(stream.unsupported.value).toBe(true)
    expect(stream.task.value?.id).toBe(8)
    expect(fetchMock).toHaveBeenCalledWith('/api/sync/task?sync_id=8', {
      credentials: 'include',
    })
  })

  it('keeps the stream connected for a server-sent business error', async () => {
    vi.stubGlobal('EventSource', MockEventSource)
    const stream = withSetup(() => useSyncTaskStream(8))
    await nextTick()
    const source = MockEventSource.instances[0]

    source.onopen?.()
    source.emit('error', {
      type: 'error',
      version: 1,
      sync_id: 8,
      data: { reason: 'tailer failed' },
    })
    await nextTick()

    expect(stream.connected.value).toBe(true)
    expect(stream.errorMessage.value).toBe('同步任务实时流返回错误')
  })
})
