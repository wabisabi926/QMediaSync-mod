// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/const', () => ({
  SERVER_URL: 'https://api.example.test/api',
}))

import AppLogViewer from '@/components/AppLogViewer.vue'

type Listener = (event: MessageEvent<string>) => void

class MockEventSource {
  static instances: MockEventSource[] = []
  onopen: (() => void) | null = null
  onerror: ((event: Event) => void) | null = null
  private readonly listeners = new Map<string, Set<Listener>>()

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

  close() {}
}

describe('AppLogViewer', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    MockEventSource.instances = []
  })

  it('creates an EventSource only after the initial log snapshot succeeds', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: vi.fn().mockResolvedValue({ entries: [], pos: 0 }),
      }),
    )
    vi.stubGlobal('EventSource', MockEventSource)

    const wrapper = mount(AppLogViewer, {
      props: {
        logPath: 'app.log',
        isRealTime: false,
      },
      global: {
        directives: { loading: {} },
        stubs: {
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
          ElButton: { template: '<button><slot /></button>' },
          ElText: { template: '<span><slot /></span>' },
        },
      },
    })

    await flushPromises()
    expect(MockEventSource.instances).toHaveLength(0)

    await wrapper.setProps({ isRealTime: true })
    await flushPromises()

    expect(MockEventSource.instances).toHaveLength(1)
    expect(MockEventSource.instances[0].url).toBe('/api/logs/stream?path=app.log')
  })

  it('keeps a successful stream connected when the server sends a business error', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: vi.fn().mockResolvedValue({ entries: [], pos: 0 }),
      }),
    )
    vi.stubGlobal('EventSource', MockEventSource)

    const wrapper = mount(AppLogViewer, {
      props: {
        logPath: 'app.log',
        isRealTime: false,
      },
      global: {
        directives: { loading: {} },
        stubs: {
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
          ElButton: { template: '<button><slot /></button>' },
          ElText: { template: '<span><slot /></span>' },
        },
      },
    })
    await flushPromises()
    await wrapper.setProps({ isRealTime: true })
    await flushPromises()
    expect(MockEventSource.instances).toHaveLength(1)
    const source = MockEventSource.instances[0]

    expect(source.onopen).toBeTypeOf('function')
    source.onopen?.()
    await flushPromises()
    expect(wrapper.text()).toContain('● 已连接')
    source.emit('error', { reason: 'tailer failed' })
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('● 已连接')
    expect(wrapper.text()).not.toContain('实时日志暂时断开，正在重新连接…')
  })

  it('aborts and ignores a stale snapshot after the log path changes', async () => {
    type PendingRequest = {
      url: string
      signal: AbortSignal | undefined
      resolve: (response: { ok: boolean; json: () => Promise<unknown> }) => void
    }
    const pendingRequests: PendingRequest[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(
        (url: string, options?: RequestInit) =>
          new Promise((resolve) => {
            pendingRequests.push({
              url,
              signal: options?.signal ?? undefined,
              resolve,
            })
          }),
      ),
    )
    vi.stubGlobal('EventSource', MockEventSource)

    const wrapper = mount(AppLogViewer, {
      props: {
        logPath: 'first.log',
        isRealTime: true,
      },
      global: {
        directives: { loading: {} },
        stubs: {
          ElCard: { template: '<div><slot name="header" /><slot /></div>' },
          ElButton: { template: '<button><slot /></button>' },
          ElText: { template: '<span><slot /></span>' },
        },
      },
    })

    await vi.waitFor(() => expect(pendingRequests).toHaveLength(1))
    await wrapper.setProps({ logPath: 'second.log' })
    await vi.waitFor(() => expect(pendingRequests).toHaveLength(2))

    pendingRequests[1].resolve({
      ok: true,
      json: vi.fn().mockResolvedValue({
        entries: [{ level: 'info', message: 'second snapshot', timestamp: 't2' }],
        pos: 2,
      }),
    })
    await flushPromises()
    pendingRequests[0].resolve({
      ok: true,
      json: vi.fn().mockResolvedValue({
        entries: [{ level: 'info', message: 'stale first snapshot', timestamp: 't1' }],
        pos: 1,
      }),
    })
    await flushPromises()

    expect(pendingRequests[0].signal?.aborted).toBe(true)
    expect(wrapper.text()).toContain('second snapshot')
    expect(wrapper.text()).not.toContain('stale first snapshot')
  })
})
