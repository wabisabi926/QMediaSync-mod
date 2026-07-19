// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ResponsiveActionBar from '@/components/common/ResponsiveActionBar.vue'

describe('ResponsiveActionBar', () => {
  it('移动端添加纵向布局 class 并渲染帮助内容', () => {
    const wrapper = mount(ResponsiveActionBar, {
      props: { isMobile: true },
      slots: {
        actions: '<button>保存设置</button><button>提取媒体信息</button>',
        help: '<p>查看下载队列页</p>',
      },
    })

    expect(wrapper.classes()).toContain('responsive-action-bar--mobile')
    expect(wrapper.text()).toContain('保存设置')
    expect(wrapper.text()).toContain('提取媒体信息')
    expect(wrapper.text()).toContain('查看下载队列页')
  })
})
