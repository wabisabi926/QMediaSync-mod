// @vitest-environment happy-dom
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import RecordActionButtons from '@/components/records/RecordActionButtons.vue'

describe('RecordActionButtons', () => {
  it('渲染所有可见操作按钮', () => {
    const wrapper = mount(RecordActionButtons, {
      props: {
        row: { id: 1 },
        actions: [
          { key: 'detail', label: '详情' },
          { key: 'delete', label: '删除' },
        ],
      },
      global: {
        stubs: {
          ElButton: { template: '<button><slot /></button>' },
          ElIcon: { template: '<span><slot /></span>' },
        },
      },
    })

    expect(wrapper.text()).toContain('详情')
    expect(wrapper.text()).toContain('删除')
  })

  it('覆盖 Element Plus 相邻按钮默认左边距，避免换行错位', () => {
    const filename = resolve(process.cwd(), 'src/components/records/RecordActionButtons.vue')
    const source = readFileSync(filename, 'utf8')

    expect(source).toContain('.record-actions :deep(.el-button + .el-button)')
    expect(source).toMatch(/margin-left:\s*0/)
  })
})
