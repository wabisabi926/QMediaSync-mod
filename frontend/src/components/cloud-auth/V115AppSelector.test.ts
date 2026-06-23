// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import V115AppSelector from '@/components/cloud-auth/V115AppSelector.vue'

describe('V115AppSelector', () => {
  it('展示所有内置 115 开放平台应用', () => {
    const wrapper = mount(V115AppSelector, {
      props: {
        authMode: 'qr',
        selectedQrApp: { appId: '100197849', appName: 'QMediaSync' },
        selectedWebProvider: 'qmediasync',
        customAppId: '',
        customAppName: '',
        'onUpdate:authMode': vi.fn(),
        'onUpdate:selectedQrApp': vi.fn(),
        'onUpdate:selectedWebProvider': vi.fn(),
        'onUpdate:customAppId': vi.fn(),
        'onUpdate:customAppName': vi.fn(),
      },
      global: {
        provide: {
          $http: {
            get: vi.fn().mockResolvedValue({ data: { data: { items: [], total: 0 } } }),
          },
        },
        stubs: {
          ElFormItem: { template: '<div><slot /></div>' },
          ElSegmented: {
            props: ['options'],
            template:
              '<div><button v-for="option in options" :key="option.value">{{ option.label }}</button></div>',
          },
          ElSelect: { template: '<div><slot /></div>' },
          ElOption: { props: ['label'], template: '<div>{{ label }}<slot /></div>' },
          ElInput: { template: '<input />' },
          ElButton: { template: '<button><slot /></button>' },
        },
      },
    })

    expect(wrapper.text()).toContain('扫码授权')
    expect(wrapper.text()).toContain('网页授权')
    expect(wrapper.text()).toContain('QMediaSync')
    expect(wrapper.text()).toContain('Q115-STRM')
    expect(wrapper.text()).toContain('MQ的媒体库')
    expect(wrapper.text()).toContain('自定义 APPID')
  })
})
