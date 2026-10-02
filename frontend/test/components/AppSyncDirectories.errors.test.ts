import axios, { CanceledError } from 'axios'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { ElMessage, ElMessageBox, type MessageBoxData } from 'element-plus'
import { defineComponent } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AppSyncDirectories from '@/components/AppSyncDirectories.vue'
import { httpKey } from '@/http/client'
import { HttpResponseError, markAuthInvalidationHandled } from '@/http/errors'
import type { APIResponse } from '@/api/types'
import { createDeferred } from '../support/deferred'

const events = vi.hoisted(
  () =>
    new Map<
      string,
      { update: (payload: Record<string, unknown>) => void; reconnect?: () => void }
    >(),
)
vi.mock('@/composables/useRealtimeEvents', () => ({
  useRealtimeEvent: (
    name: string,
    update: (payload: Record<string, unknown>) => void,
    reconnect?: () => void,
  ) => events.set(name, { update, reconnect }),
}))
const DialogStub = defineComponent({
  props: { modelValue: Boolean },
  template: '<section v-if="modelValue" role="dialog"><slot /><slot name="footer" /></section>',
})
const rejection = (code: string, status = 403) =>
  new HttpResponseError({
    status,
    data: { code: 500, error_code: code, message: 'private token' },
    config: { method: 'post', url: '/api/sync/path/start?token=private' },
  })
const mountDirectories = async (running = 0, directoryIds = [12]) => {
  const getReply = vi.fn(async (url: string): Promise<APIResponse<unknown>> => {
    if (url.endsWith('/sync/path-list'))
      return {
        code: 200,
        message: '',
        data: {
          list: directoryIds.map((id) => ({
            id,
            base_cid: 'root',
            local_path: '/strm',
            remote_path: '/remote',
            source_type: '115',
            account_id: 1,
            account_name: 'my-account',
            enable_cron: false,
            directory_upload_enabled: true,
            is_running: running,
          })),
          total: directoryIds.length,
        },
      }
    if (url.endsWith('/directory-upload/rules'))
      return {
        code: 200,
        message: '',
        data: {
          list: [
            {
              id: 4,
              sync_path_id: 12,
              enabled: true,
              monitor_path: '/watch',
              remote_root_path: '/remote/upload',
              recursive: true,
            },
          ],
        },
      }
    return { code: 200, message: '', data: [5] }
  })
  const reply = vi
    .fn<() => Promise<APIResponse<unknown>>>()
    .mockResolvedValue({ code: 200, message: '', data: null })
  const adapter = vi.fn(async (config) => ({
    config,
    status: 200,
    statusText: 'OK',
    headers: {},
    data: config.method === 'get' ? await getReply(config.url) : await reply(),
  }))
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div />' } }],
  })
  await router.push('/')
  const wrapper = mount(AppSyncDirectories, {
    global: {
      plugins: [router],
      provide: { [httpKey]: axios.create({ adapter }) },
      stubs: { PageHeader: true, PageStats: true, ElDialog: DialogStub },
    },
  })
  await flushPromises()
  const act = async (ariaLabel: string, index = 0) => {
    await wrapper.findAll(`button[aria-label="${ariaLabel}"]`)[index]!.trigger('click')
    await flushPromises()
  }
  return { wrapper, reply, getReply, adapter, act }
}

const relationSuccess = (data: unknown = null): APIResponse<unknown> => ({
  code: 200,
  message: '',
  data,
})
const relationFailure = (): APIResponse<unknown> => ({
  code: 500,
  message: '同步目录操作失败，请稍后重试',
  data: null,
})

enableAutoUnmount(afterEach)
beforeEach(() => {
  events.clear()
  vi.spyOn(ElMessage, 'error').mockImplementation(() => ({ close: vi.fn() }))
  vi.spyOn(ElMessage, 'success').mockImplementation(() => ({ close: vi.fn() }))
  vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as MessageBoxData)
  vi.spyOn(console, 'error').mockImplementation(() => {})
})
afterEach(() => vi.restoreAllMocks())

describe('同步目录列表请求行为', () => {
  it.each(['增量同步', '全量同步', '删除同步目录', '目录监控扫描'])(
    '%s 业务失败不改变卡片、不刷新或报成功',
    async (action) => {
      const { wrapper, reply, getReply, act } = await mountDirectories()
      reply.mockResolvedValueOnce({
        code: 500,
        message: '同步目录操作失败，请稍后重试',
        data: { is_running: 2, accepted: 1 },
      })
      const reads = getReply.mock.calls.length
      await act(action)
      expect(reply).toHaveBeenCalledOnce()
      expect(wrapper.find('.directory-card').exists()).toBe(true)
      expect(wrapper.find('.is-running').exists()).toBe(false)
      expect(wrapper.find('.is-waiting').exists()).toBe(false)
      expect(getReply).toHaveBeenCalledTimes(reads)
      expect(ElMessage.success).not.toHaveBeenCalled()
      expect(ElMessage.error).toHaveBeenCalledOnce()
      expect(JSON.stringify(vi.mocked(ElMessage.error).mock.calls)).not.toContain('private')
      expect(JSON.stringify(vi.mocked(console.error).mock.calls)).not.toContain('private')
    },
  )

  it.each(['success', 'failure'])('新列表开始后忽略旧列表的 %s 结果', async (outcome) => {
    const { wrapper, getReply } = await mountDirectories()
    const original = getReply.getMockImplementation()!
    const old = createDeferred<APIResponse<unknown>>()
    const current = createDeferred<APIResponse<unknown>>()
    getReply.mockReturnValueOnce(old.promise).mockReturnValueOnce(current.promise)
    const reload = events.get('sync_task_created')!.reconnect!
    reload()
    reload()
    current.resolve(await original('/api/sync/path-list'))
    await flushPromises()
    const reads = getReply.mock.calls.length
    old.resolve(outcome === 'success' ? relationSuccess({ list: [], total: 0 }) : relationFailure())
    await flushPromises()
    expect(wrapper.find('.directory-card').exists()).toBe(true)
    expect(getReply).toHaveBeenCalledTimes(reads)
    expect(ElMessage.error).not.toHaveBeenCalled()
    expect(console.error).not.toHaveBeenCalled()
  })

  it('新列表的上传规则不会被旧规则请求失败清空或改为失败状态', async () => {
    const { wrapper, getReply } = await mountDirectories()
    const original = getReply.getMockImplementation()!
    const oldRules = createDeferred<APIResponse<unknown>>()
    getReply
      .mockResolvedValueOnce(await original('/api/sync/path-list'))
      .mockReturnValueOnce(oldRules.promise)
    const reload = events.get('sync_task_created')!.reconnect!
    reload()
    await flushPromises()
    reload()
    await flushPromises()
    oldRules.resolve(relationFailure())
    await flushPromises()
    expect(wrapper.text()).toContain('/watch')
    expect(wrapper.text()).not.toContain('加载失败')
    expect(ElMessage.error).not.toHaveBeenCalled()
    expect(console.error).not.toHaveBeenCalled()
  })

  it('列表替换后旧状态读取不回写新列表，卸载后失败不提示', async () => {
    const { wrapper, getReply } = await mountDirectories()
    const oldStatus = createDeferred<APIResponse<unknown>>()
    getReply.mockReturnValueOnce(oldStatus.promise)
    const refreshStatus = events.get('strm_sync_task_start')!.update
    refreshStatus({})
    events.get('sync_task_created')!.reconnect!()
    await flushPromises()
    oldStatus.resolve(relationSuccess({ list: [{ id: 12, is_running: 2 }], total: 1 }))
    await flushPromises()
    expect(wrapper.get('.directory-card').classes()).not.toContain('is-running')
    const lateStatus = createDeferred<APIResponse<unknown>>()
    getReply.mockReturnValueOnce(lateStatus.promise)
    refreshStatus({})
    wrapper.unmount()
    lateStatus.resolve(relationFailure())
    await flushPromises()
    expect(ElMessage.error).not.toHaveBeenCalled()
    expect(console.error).not.toHaveBeenCalled()
  })

  it('后台状态事件合并读取，连续失败只提示一次，成功后重新允许提示', async () => {
    const { getReply } = await mountDirectories()
    const original = getReply.getMockImplementation()!
    const first = createDeferred<APIResponse<unknown>>()
    getReply.mockClear()
    getReply.mockReturnValueOnce(first.promise).mockResolvedValueOnce(relationFailure())
    const refreshStatus = events.get('strm_sync_task_start')!.update
    refreshStatus({})
    refreshStatus({})
    refreshStatus({})
    await flushPromises()
    expect(getReply).toHaveBeenCalledTimes(1)
    first.resolve(relationFailure())
    await flushPromises()
    expect(getReply).toHaveBeenCalledTimes(2)
    expect(ElMessage.error).toHaveBeenCalledTimes(1)
    getReply.mockResolvedValueOnce(await original('/api/sync/path-list'))
    refreshStatus({})
    await flushPromises()
    getReply.mockResolvedValueOnce(relationFailure())
    refreshStatus({})
    await flushPromises()
    expect(ElMessage.error).toHaveBeenCalledTimes(2)
  })

  it('成功启动应用服务端运行状态；停止失败保持运行状态', async () => {
    const { wrapper, reply, act } = await mountDirectories()
    reply.mockResolvedValueOnce({ code: 200, message: '', data: { is_running: 2 } })
    await act('增量同步')
    expect(wrapper.get('.directory-card').classes()).toContain('is-running')
    vi.mocked(ElMessage.success).mockClear()
    reply.mockRejectedValueOnce(rejection('REQUEST_ORIGIN_INVALID'))
    await act('停止同步')
    expect(wrapper.get('.directory-card').classes()).toContain('is-running')
    expect(ElMessage.error).toHaveBeenCalledWith(expect.stringContaining('访问地址校验失败'))
    expect(ElMessage.success).not.toHaveBeenCalled()
  })

  it('定时开关安全校验失败回滚开关并展示真实原因', async () => {
    const { wrapper, reply } = await mountDirectories()
    reply.mockRejectedValueOnce(rejection('CSRF_TOKEN_INVALID'))
    await wrapper.get('[role="switch"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[role="switch"]').attributes('aria-checked')).toBe('false')
    expect(ElMessage.error).toHaveBeenCalledWith(expect.stringContaining('请求安全校验失败'))
    expect(ElMessage.success).not.toHaveBeenCalled()
  })

  it.each(['增量同步', '删除同步目录', '目录监控扫描'])(
    '%s 的取消和已处理 401 静默',
    async (action) => {
      const { reply, act } = await mountDirectories()
      const handled = rejection('SESSION_INVALID', 401)
      markAuthInvalidationHandled(handled)
      for (const error of [new CanceledError(), handled]) {
        reply.mockRejectedValueOnce(error)
        await act(action)
      }
      expect(reply).toHaveBeenCalledTimes(2)
      expect(ElMessage.error).not.toHaveBeenCalled()
      expect(ElMessage.success).not.toHaveBeenCalled()
      expect(console.error).not.toHaveBeenCalled()
    },
  )

  it('重连刷新失败保留原目录，旧任务事件不会覆盖较新状态', async () => {
    const { wrapper, getReply } = await mountDirectories()
    const update = events.get('sync_task_updated')!.update
    update({ sync_path_id: 12, sync_id: 1, sequence: 2, is_running: 2, event_time: 10 })
    update({ sync_path_id: 12, sync_id: 1, sequence: 1, is_running: 0, event_time: 9 })
    await flushPromises()
    expect(wrapper.get('.directory-card').classes()).toContain('is-running')
    getReply.mockRejectedValueOnce(rejection('REQUEST_ORIGIN_INVALID'))
    events.get('sync_task_created')!.reconnect!()
    await flushPromises()
    expect(wrapper.find('.directory-card').exists()).toBe(true)
    expect(wrapper.get('.directory-card').classes()).toContain('is-running')
  })
  it('停止响应成功只表示请求已受理，等待实时状态确认停止', async () => {
    const { wrapper, act } = await mountDirectories(2)
    await act('停止同步')
    expect(ElMessage.success).toHaveBeenCalledWith('已请求停止同步任务')
    expect(wrapper.get('.directory-card').classes()).toContain('is-running')
    events
      .get('sync_task_updated')!
      .update({ sync_path_id: 12, sync_id: 1, sequence: 1, is_running: 0 })
    await flushPromises()
    expect(wrapper.get('.directory-card').classes()).not.toContain('is-running')
  })
})
