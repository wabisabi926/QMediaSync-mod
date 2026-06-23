// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
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
          ElFormItem: { props: ['label'], template: '<div><span>{{ label }}</span><slot /></div>' },
          ElSegmented: {
            props: ['options'],
            template:
              '<div><button v-for="option in options" :key="option.value">{{ option.label }}</button></div>',
          },
          ElSelect: { template: '<div><slot /></div>' },
          ElOption: { props: ['label'], template: '<div>{{ label }}<slot /></div>' },
          ElInput: {
            props: ['name', 'placeholder', 'modelValue'],
            template:
              '<input :name="name" :placeholder="placeholder" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value); $emit(\'input\', $event.target.value)" />',
          },
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

  it('扫码授权把 APPID 搜索内置在下拉框里', () => {
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
          ElFormItem: { props: ['label'], template: '<div><span>{{ label }}</span><slot /></div>' },
          ElSegmented: {
            props: ['options'],
            template:
              '<div><button v-for="option in options" :key="option.value">{{ option.label }}</button></div>',
          },
          ElSelect: {
            props: ['placeholder'],
            template:
              '<div><input name="v115-appid-search" :placeholder="placeholder" /><slot /></div>',
          },
          ElOption: { props: ['label'], template: '<div>{{ label }}<slot /></div>' },
          ElInput: {
            props: ['name', 'placeholder', 'modelValue'],
            template:
              '<input :name="name" :placeholder="placeholder" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value); $emit(\'input\', $event.target.value)" />',
          },
          ElButton: { template: '<button><slot /></button>' },
        },
      },
    })

    expect(wrapper.text()).not.toContain('搜索 APPID')
    expect(wrapper.find('input[name="v115-appid-search"]').exists()).toBe(true)
    expect(wrapper.find('input[name="v115-appid-search"]').attributes('placeholder')).toBe(
      '搜索应用名或 APPID',
    )
  })

  it('扫码授权搜索结果存在下一页时显示加载更多入口', async () => {
    const get = vi.fn().mockResolvedValue({
      data: {
        data: {
          items: [{ app_id: '1001', app_name: '测试应用', display_name: '测试应用' }],
          total: 2,
        },
      },
    })
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
          $http: { get },
        },
        stubs: {
          ElFormItem: { props: ['label'], template: '<div><span>{{ label }}</span><slot /></div>' },
          ElSegmented: {
            props: ['options'],
            template:
              '<div><button v-for="option in options" :key="option.value">{{ option.label }}</button></div>',
          },
          ElSelect: {
            props: ['placeholder', 'remoteMethod'],
            template:
              '<div><input name="v115-appid-search" :placeholder="placeholder" @input="remoteMethod?.($event.target.value)" /><slot /><slot name="footer" /></div>',
          },
          ElOption: { props: ['label'], template: '<div>{{ label }}<slot /></div>' },
          ElInput: {
            props: ['name', 'placeholder', 'modelValue'],
            template:
              '<input :name="name" :placeholder="placeholder" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value); $emit(\'input\', $event.target.value)" />',
          },
          ElButton: { template: '<button><slot /></button>' },
        },
      },
    })

    await wrapper.find('input[name="v115-appid-search"]').setValue('测试')
    await flushPromises()

    expect(get).toHaveBeenCalled()
    expect(wrapper.text()).toContain('加载更多')
  })
})
