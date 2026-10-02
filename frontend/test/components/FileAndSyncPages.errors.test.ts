import { browseSortOptions } from '../support/browseSort'
import axios, { AxiosError, CanceledError } from 'axios'
import { enableAutoUnmount, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { ElMessage, ElMessageBox, type MessageBoxData } from 'element-plus'
import { createPinia } from 'pinia'
import { defineComponent, nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AppSyncRecords from '@/components/AppSyncRecords.vue'
import DirectorySelector from '@/components/DirectorySelector.vue'
import { httpKey } from '@/http/client'
import { HttpResponseError, markAuthInvalidationHandled } from '@/http/errors'
import { createDeferred } from '../support/deferred'

vi.mock('@/composables/useRealtimeEvents', () => ({ useRealtimeEvent: vi.fn() }))

const DialogStub = defineComponent({
  props: { modelValue: Boolean, title: String },
  template:
    '<section v-if="modelValue" role="dialog" :aria-label="title"><slot /><slot name="footer" /></section>',
})
const recordTableStub = defineComponent({
  props: ['rows', 'showSelection'],
  emits: ['action', 'selectionChange'],
  template: `<div>
    <div v-for="row in rows" :key="row.id" class="record-row">
      <span>{{ row.remote_path }}</span>
      <button @click="$emit('action', { actionKey: 'delete', row })">删除记录</button>
    </div>
    <button v-if="showSelection" @click="$emit('selectionChange', rows)">选择全部记录</button>
  </div>`,
})
const pageStub = defineComponent({
  props: ['total', 'currentPage'],
  emits: ['currentChange'],
  template: '<button @click="$emit(\'currentChange\', 2)">第二页</button>',
})
const envelope = (data: unknown, code = 200, message = '') => ({ code, data, message })
const directory = { id: '/strm', name: '目标目录', path: '/strm' }
const record = {
  id: 1,
  created_at: 1,
  finish_at: 2,
  status: 2,
  sub_status: 0,
  total: 3,
  new_strm: 2,
  new_meta: 0,
  new_upload: 0,
  local_path: '/strm',
  remote_path: '原记录',
  fail_reason: '',
}
const accounts = [
  {
    id: 1,
    name: '媒体',
    username: '媒体',
    user_id: '1',
    source_type: '115',
    authorized: true,
    created_at: 1,
  },
  {
    id: 2,
    name: '第二账号',
    username: '第二账号',
    user_id: '2',
    source_type: 'openlist',
    authorized: true,
    created_at: 1,
  },
]
const failure = (errorCode: string, status = 403) =>
  new HttpResponseError({
    status,
    data: { code: 500, error_code: errorCode, message: 'private SQL token' },
    config: { method: 'post', url: '/api/path/create?token=private' },
  })
const handled401 = () => {
  const error = failure('AUTHENTICATION_REQUIRED', 401)
  markAuthInvalidationHandled(error)
  return error
}
const button = (wrapper: VueWrapper, label: string) =>
  wrapper.findAll('button').find((item) => item.text() === label)!
const click = async (wrapper: VueWrapper, label: string) => {
  await button(wrapper, label).trigger('click')
  await flushPromises()
}
const mountPage = async (kind: 'records' | 'directories') => {
  const readReply = vi.fn(
    async (url: string, params?: Record<string, unknown>): Promise<unknown> => {
      if (url.endsWith('/path/sort-options'))
        return envelope(
          browseSortOptions(
            String(params?.source_type),
            params?.scope === 'directories' ? 'directories' : 'files',
          ),
        )
      if (url.endsWith('/account/list')) return envelope(accounts)
      if (url.endsWith('/path/list')) return envelope(params?.parent_id ? [] : [directory])
      return envelope({ records: [record], total: 30 })
    },
  )
  const writeReply = vi.fn(async (): Promise<unknown> => envelope(null))
  const adapter = vi.fn(async (config) => ({
    config,
    status: 200,
    statusText: 'OK',
    headers: {},
    data: config.method === 'get' ? await readReply(config.url, config.params) : await writeReply(),
  }))
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div />' } }],
  })
  await router.push('/')
  await router.isReady()
  const wrapper = mount(kind === 'records' ? AppSyncRecords : DirectorySelector, {
    props: kind === 'directories' ? { sourceType: 'local' } : {},
    global: {
      plugins: [createPinia(), router],
      provide: { [httpKey]: axios.create({ adapter }) },
      stubs: {
        PageHeader: true,
        ElDialog: DialogStub,
        ResponsiveRecordTable: recordTableStub,
        ResponsivePagination: pageStub,
      },
    },
  })
  await flushPromises()
  return { wrapper, adapter, readReply, writeReply }
}
const openCreate = async (wrapper: VueWrapper) => {
  await click(wrapper, '新建文件夹')
  await wrapper.get('input[placeholder="请输入文件夹名称"]').setValue(' 保留名称 ')
}

enableAutoUnmount(afterEach)
beforeEach(() => {
  sessionStorage.clear()
  vi.restoreAllMocks()
  vi.spyOn(ElMessage, 'error').mockImplementation(() => ({ close: vi.fn() }))
  vi.spyOn(ElMessage, 'success').mockImplementation(() => ({ close: vi.fn() }))
  vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as MessageBoxData)
  vi.spyOn(console, 'error').mockImplementation(() => {})
})

describe('directory selector HTTP feedback', () => {
  it('keeps folder name and selection on create failure', async () => {
    const { wrapper, writeReply } = await mountPage('directories')
    await wrapper.get('[role="treeitem"]').trigger('click')
    await flushPromises()
    await openCreate(wrapper)
    writeReply.mockResolvedValueOnce(envelope(null, 500, '创建目录失败：名称已存在'))
    await click(wrapper, '确定')
    expect(ElMessage.error).toHaveBeenCalledWith('创建目录失败：名称已存在')
    expect(ElMessage.success).not.toHaveBeenCalled()
    expect(
      (wrapper.get('input[placeholder="请输入文件夹名称"]').element as HTMLInputElement).value,
    ).toBe(' 保留名称 ')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(true)
  })

  it.each([new CanceledError(), handled401()])(
    'does not replace a cancelled/auth child refresh with a generic error',
    async (error) => {
      const { wrapper, readReply } = await mountPage('directories')
      await wrapper.get('[role="treeitem"]').trigger('click')
      await flushPromises()
      readReply.mockResolvedValueOnce(envelope([directory])).mockRejectedValueOnce(error)
      await click(wrapper, '刷新')
      expect(ElMessage.error).not.toHaveBeenCalled()
      expect(wrapper.text()).toContain('目标目录')
    },
  )

  it('classifies child load failure and keeps retry available', async () => {
    const { wrapper, readReply } = await mountPage('directories')
    readReply.mockRejectedValueOnce(new AxiosError('private', 'ERR_NETWORK'))
    await wrapper.get('[role="treeitem"]').trigger('click')
    await flushPromises()
    expect(ElMessage.error).toHaveBeenCalledWith(expect.stringContaining('无法获取服务器响应'))
    expect(wrapper.find('[role="treeitem"]').attributes('aria-expanded')).toBe('true')
    expect(wrapper.text()).toContain('重试')
  })
})

describe('sync records HTTP feedback', () => {
  it('rejects single deletion business failure without reporting success', async () => {
    const { wrapper, writeReply, readReply } = await mountPage('records')
    writeReply.mockResolvedValueOnce(envelope(null, 500, '记录不存在'))
    await click(wrapper, '删除记录')
    expect(ElMessage.error).toHaveBeenCalledExactlyOnceWith('记录不存在')
    expect(ElMessage.success).not.toHaveBeenCalled()
    expect(readReply).toHaveBeenCalledTimes(1)
  })

  it('keeps batch selection after partial cleanup failure and uses safe complete message', async () => {
    const { wrapper, writeReply, adapter } = await mountPage('records')
    await wrapper.get('input[type="checkbox"]').setValue(true)
    await click(wrapper, '选择全部记录')
    writeReply.mockResolvedValueOnce(
      envelope(
        { deleted_ids: [], failures: [{ id: 1, reason: 'private log path' }] },
        500,
        '部分同步记录删除失败',
      ),
    )
    await click(wrapper, '批量删除')
    expect(ElMessage.error).toHaveBeenCalledExactlyOnceWith('部分同步记录删除失败')
    expect(ElMessage.success).not.toHaveBeenCalled()
    expect(button(wrapper, '批量删除').attributes('disabled')).toBeUndefined()
    const request = adapter.mock.calls.find(([config]) => config.method === 'post')![0]
    expect(request.timeout).toBe(60000)
    expect(request.headers.get('Content-Type')).toBe('application/json')
    await click(wrapper, '批量删除')
    expect(ElMessage.success).toHaveBeenCalledWith('批量删除成功')
    expect(button(wrapper, '批量删除').attributes('disabled')).toBeDefined()
  })

  it.each([new CanceledError(), handled401()])(
    'silences cancellation and handled 401 during deletion',
    async (error) => {
      const { wrapper, writeReply } = await mountPage('records')
      writeReply.mockRejectedValueOnce(error)
      await click(wrapper, '删除记录')
      expect(ElMessage.error).not.toHaveBeenCalled()
      expect(ElMessage.success).not.toHaveBeenCalled()
    },
  )

  it('reports successful deletion separately from a failed snapshot reload', async () => {
    const { wrapper, readReply } = await mountPage('records')
    readReply.mockResolvedValueOnce(envelope(null, 500, '数据库繁忙'))
    await click(wrapper, '删除记录')
    expect(ElMessage.success).toHaveBeenCalledWith('删除成功')
    expect(ElMessage.error).toHaveBeenCalledExactlyOnceWith('加载同步记录失败：数据库繁忙')
    expect(wrapper.text()).toContain('原记录')
  })

  it('ignores stale read failure when a newer page is pending', async () => {
    const { wrapper, readReply } = await mountPage('records')
    const old = createDeferred<unknown>()
    readReply
      .mockReturnValueOnce(old.promise)
      .mockResolvedValueOnce(
        envelope({ records: [{ ...record, remote_path: '新页面' }], total: 30 }),
      )
    await click(wrapper, '删除记录')
    await click(wrapper, '第二页')
    old.resolve(envelope(null, 500, 'private old SQL'))
    await flushPromises()
    await nextTick()
    expect(wrapper.text()).toContain('新页面')
    expect(ElMessage.error).not.toHaveBeenCalled()
  })
})
