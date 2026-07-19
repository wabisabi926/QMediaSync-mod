// @vitest-environment happy-dom

import { mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { defineComponent } from 'vue'
import { describe, expect, it } from 'vitest'

import ResponsivePagination from '@/components/common/ResponsivePagination.vue'

const ElPaginationStub = defineComponent({
  name: 'ElPagination',
  props: {
    currentPage: Number,
    pageSize: Number,
    pageSizes: Array,
    pagerCount: Number,
    size: String,
    background: Boolean,
    layout: String,
    total: Number,
  },
  emits: ['update:currentPage', 'update:pageSize', 'size-change', 'current-change'],
  template: `
    <div
      data-test="el-pagination"
      :data-layout="layout"
      :data-pager-count="String(pagerCount)"
      :data-size="size"
    />
  `,
})

describe('ResponsivePagination', () => {
  it('分页控件默认居中排列', () => {
    const source = readFileSync('src/components/common/ResponsivePagination.vue', 'utf8')

    expect(source).toContain('justify-content: center;')
    expect(source).not.toContain('justify-content: flex-end;')
  })

  it('移动端默认保留每页数量选择', () => {
    const wrapper = mount(ResponsivePagination, {
      props: {
        currentPage: 1,
        pageSize: 50,
        total: 500,
        isMobile: true,
        'onUpdate:currentPage': () => undefined,
        'onUpdate:pageSize': () => undefined,
      },
      global: {
        stubs: {
          ElPagination: ElPaginationStub,
          'el-pagination': ElPaginationStub,
        },
      },
    })

    const pagination = wrapper.get('[data-test="el-pagination"]')

    expect(pagination.attributes('data-layout')).toBe('total, sizes, prev, pager, next')
    expect(pagination.attributes('data-pager-count')).toBe('5')
    expect(pagination.attributes('data-size')).toBe('small')
  })
})
