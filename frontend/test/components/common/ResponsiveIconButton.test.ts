// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { Plus } from '@element-plus/icons-vue'
import ResponsiveIconButton from '@/components/common/ResponsiveIconButton.vue'

describe('ResponsiveIconButton', () => {
  it('移动端隐藏文字并保留 aria-label', () => {
    const wrapper = mount(ResponsiveIconButton, {
      props: {
        icon: Plus,
        label: '添加渠道',
        mobileIconOnly: true,
        isMobile: true,
      },
      global: {
        stubs: {
          'el-button': {
            props: ['ariaLabel'],
            template:
              '<button :aria-label="ariaLabel" class="el-button"><slot /><slot name="icon" /></button>',
          },
          'el-icon': { template: '<i><slot /></i>' },
        },
      },
    })

    expect(wrapper.find('button').attributes('aria-label')).toBe('添加渠道')
    expect(wrapper.find('.responsive-icon-button__label').exists()).toBe(false)
  })

  it('桌面端显示按钮文字', () => {
    const wrapper = mount(ResponsiveIconButton, {
      props: {
        icon: Plus,
        label: '添加渠道',
        isMobile: false,
      },
      global: {
        stubs: {
          'el-button': { template: '<button class="el-button"><slot /></button>' },
          'el-icon': { template: '<i><slot /></i>' },
        },
      },
    })

    expect(wrapper.text()).toContain('添加渠道')
  })
})
