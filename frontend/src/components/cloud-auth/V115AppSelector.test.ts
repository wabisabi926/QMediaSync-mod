// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import V115AppSelector from '@/components/cloud-auth/V115AppSelector.vue'

describe('V115AppSelector', () => {
  it('展示所有内置 115 开放平台应用', () => {
    const wrapper = mount(V115AppSelector, {
      props: {
        appName: 'QMediaSync',
        appId: '',
        customAppName: '',
      },
      global: {
        stubs: {
          ElFormItem: { template: '<div><slot /></div>' },
          ElSegmented: {
            props: ['options'],
            template:
              '<div><button v-for="option in options" :key="option.value">{{ option.label }}</button></div>',
          },
        },
      },
    })

    expect(wrapper.text()).toContain('QMediaSync')
    expect(wrapper.text()).toContain('Q115-STRM')
    expect(wrapper.text()).toContain('MQ的媒体库')
    expect(wrapper.text()).toContain('自定义 APPID')
  })
})
