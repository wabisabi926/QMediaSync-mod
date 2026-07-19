// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { defineComponent, h, KeepAlive, nextTick, ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { useDeviceType } from '@/composables/useDeviceType'

const setWindowWidth = (width: number) => {
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: width })
}

const resizeListenerCalls = (spy: ReturnType<typeof vi.spyOn>) =>
  spy.mock.calls.filter((call: unknown[]) => call[0] === 'resize')

afterEach(() => {
  vi.restoreAllMocks()
  setWindowWidth(1024)
})

describe('useDeviceType', () => {
  it('在组件挂载时订阅 resize，并在卸载后停止更新', async () => {
    setWindowWidth(1024)
    const addEventListener = vi.spyOn(window, 'addEventListener')
    const removeEventListener = vi.spyOn(window, 'removeEventListener')
    let device: ReturnType<typeof useDeviceType> | undefined

    const Consumer = defineComponent({
      setup() {
        device = useDeviceType()
        return () => h('span', device?.isMobile.value ? 'mobile' : 'desktop')
      },
    })

    const wrapper = mount(Consumer)
    expect(wrapper.text()).toBe('desktop')
    expect(resizeListenerCalls(addEventListener)).toHaveLength(1)

    setWindowWidth(375)
    window.dispatchEvent(new Event('resize'))
    await nextTick()
    expect(wrapper.text()).toBe('mobile')

    wrapper.unmount()
    expect(resizeListenerCalls(removeEventListener)).toHaveLength(1)

    setWindowWidth(1024)
    window.dispatchEvent(new Event('resize'))
    await nextTick()
    expect(device?.isMobile.value).toBe(true)
  })

  it('KeepAlive 停用期间保持单一订阅，并在重新激活后使用最新状态', async () => {
    setWindowWidth(1024)
    const addEventListener = vi.spyOn(window, 'addEventListener')
    const removeEventListener = vi.spyOn(window, 'removeEventListener')
    const visible = ref(true)
    let device: ReturnType<typeof useDeviceType> | undefined

    const Consumer = defineComponent({
      setup() {
        device = useDeviceType()
        return () =>
          h('span', { class: 'device-type' }, device?.isMobile.value ? 'mobile' : 'desktop')
      },
    })
    const Host = defineComponent({
      setup() {
        return () => h(KeepAlive, null, { default: () => (visible.value ? h(Consumer) : null) })
      },
    })

    const wrapper = mount(Host)
    expect(wrapper.find('.device-type').text()).toBe('desktop')
    expect(resizeListenerCalls(addEventListener)).toHaveLength(1)

    setWindowWidth(375)
    window.dispatchEvent(new Event('resize'))
    await nextTick()
    expect(device?.isMobile.value).toBe(true)

    visible.value = false
    await nextTick()
    setWindowWidth(1024)
    window.dispatchEvent(new Event('resize'))
    await nextTick()
    expect(device?.isMobile.value).toBe(false)

    visible.value = true
    await nextTick()
    expect(wrapper.find('.device-type').text()).toBe('desktop')
    expect(resizeListenerCalls(addEventListener)).toHaveLength(1)

    wrapper.unmount()
    expect(resizeListenerCalls(removeEventListener)).toHaveLength(1)
  })
})
