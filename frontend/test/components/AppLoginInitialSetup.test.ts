// @vitest-environment happy-dom
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'

import AppLogin from '@/components/AppLogin.vue'
import { httpKey } from '@/http/client'

const router = {
  push: vi.fn(),
  currentRoute: {
    value: {
      query: {},
    },
  },
}

vi.mock('vue-router', () => ({
  useRouter: () => router,
}))

const mountLogin = (http: { get: ReturnType<typeof vi.fn>; post: ReturnType<typeof vi.fn> }) =>
  mount(AppLogin, {
    global: {
      plugins: [createPinia()],
      provide: {
        [httpKey]: http,
      },
      stubs: {
        ElForm: {
          name: 'ElForm',
          props: ['model', 'rules'],
          template: '<form v-bind="$attrs" @submit="$emit(\'submit\', $event)"><slot /></form>',
        },
        ElFormItem: { props: ['prop'], template: '<div><slot /></div>' },
        ElInput: {
          props: ['modelValue', 'type', 'name', 'autocomplete', 'placeholder', 'disabled', 'id'],
          template:
            '<input :id="id" :value="modelValue" :type="type || \'text\'" :name="name" :autocomplete="autocomplete" :placeholder="placeholder" :disabled="disabled" />',
        },
        ElCheckbox: {
          props: ['modelValue', 'disabled'],
          template:
            '<label><input type="checkbox" :checked="modelValue" :disabled="disabled" /> <slot /></label>',
        },
        ElButton: {
          props: ['type', 'nativeType', 'loading'],
          template:
            '<button :type="nativeType || \'button\'" :disabled="loading"><slot /></button>',
        },
      },
    },
  })

describe('AppLogin 初始化模式', () => {
  it('用户表为空时显示创建管理员表单', async () => {
    const http = {
      get: vi.fn().mockResolvedValue({
        data: {
          code: 200,
          data: { required: true },
        },
      }),
      post: vi.fn(),
    }
    const wrapper = mountLogin(http)

    await flushPromises()

    expect(wrapper.text()).toContain('创建管理员')
    expect(wrapper.text()).toContain('初始化码')
  })

  it('用户表已有用户时显示登录表单', async () => {
    const http = {
      get: vi.fn().mockResolvedValue({
        data: {
          code: 200,
          data: { required: false },
        },
      }),
      post: vi.fn(),
    }
    const wrapper = mountLogin(http)

    await flushPromises()

    expect(wrapper.text()).toContain('系统登录')
    expect(wrapper.text()).toContain('登录')
  })
})
