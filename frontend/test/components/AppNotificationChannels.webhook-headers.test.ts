// @vitest-environment happy-dom
import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import AppNotificationChannels from '@/components/AppNotificationChannels.vue'
import { httpKey } from '@/http/client'

describe('AppNotificationChannels Webhook headers', () => {
  it('创建 Webhook 渠道时提交多个自定义请求头', async () => {
    const http = {
      get: vi.fn().mockResolvedValue({ data: { code: 0, data: [] } }),
      post: vi.fn().mockResolvedValue({ data: { code: 0, data: {} } }),
    }
    const wrapper = shallowMount(AppNotificationChannels, {
      global: {
        provide: { [httpKey]: http },
        stubs: {
          ResponsiveIconButton: true,
        },
      },
    })

    const vm = wrapper.vm as unknown as {
      selectedChannelType: string
      channelForm: {
        channel_name: string
        endpoint: string
        method: string
        format: string
        template: string
        headers: Array<{ key: string; value: string }>
      }
      createChannel: () => Promise<void>
    }
    vm.selectedChannelType = 'webhook'
    Object.assign(vm.channelForm, {
      channel_name: 'Webhook',
      endpoint: 'https://example.com/webhook',
      method: 'POST',
      format: 'json',
      template: '{"title":"{{title}}"}',
      headers: [
        { key: 'X-Trace-ID', value: 'trace-1' },
        { key: 'X-Webhook-Source', value: 'qmediasync' },
      ],
    })

    await vm.createChannel()

    expect(http.post).toHaveBeenCalledWith(
      expect.stringContaining('/setting/notification/channels/webhook'),
      expect.objectContaining({
        headers: {
          'X-Trace-ID': 'trace-1',
          'X-Webhook-Source': 'qmediasync',
        },
      }),
    )
  })
})
