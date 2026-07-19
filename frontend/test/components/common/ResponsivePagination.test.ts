// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ResponsivePagination from '@/components/common/ResponsivePagination.vue'

describe('ResponsivePagination', () => {
  it('移动端保留每页数量并使用紧凑尺寸', () => {
    const wrapper = mount(ResponsivePagination, {
      props: {
        currentPage: 1,
        pageSize: 100,
        total: 300,
        pageSizes: [100, 200, 500],
        isMobile: true,
      },
      global: {
        stubs: {
          'el-pagination': {
            props: ['layout', 'pagerCount', 'size'],
            template:
              '<div class="el-pagination" :data-layout="layout" :data-pager-count="pagerCount" :data-size="size"></div>',
          },
        },
      },
    })

    const pagination = wrapper.find('.el-pagination')
    expect(pagination.attributes('data-layout')).toBe('total, sizes, prev, pager, next')
    expect(pagination.attributes('data-pager-count')).toBe('5')
    expect(pagination.attributes('data-size')).toBe('small')
  })
})
