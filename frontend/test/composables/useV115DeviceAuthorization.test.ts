import { SERVER_URL } from '@/const'
import { effectScope } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  V115_QR_STATUS_TIMEOUT_MS,
  useV115DeviceAuthorization,
} from '@/composables/useV115DeviceAuthorization'

describe('useV115DeviceAuthorization', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('uses a long timeout for 115 QR status polling', async () => {
    const setTimeoutMock = vi.fn(() => 1)
    const clearTimeoutMock = vi.fn()
    vi.stubGlobal('window', {
      setInterval: vi.fn(() => 1),
      clearInterval: vi.fn(),
      setTimeout: setTimeoutMock,
      clearTimeout: clearTimeoutMock,
    })

    const post = vi
      .fn()
      .mockResolvedValueOnce({
        data: {
          code: 200,
          data: {
            uid: 'qr-uid',
            time: 1,
            sign: 'sign',
            qrcode: '115://auth-content',
            expires: 300,
          },
        },
      })
      .mockResolvedValueOnce({
        data: {
          code: 200,
          data: {
            status: 'waiting',
            tip: '等待扫码',
          },
        },
      })

    const scope = effectScope()
    const authorization = scope.run(() => useV115DeviceAuthorization({ post } as never))

    expect(authorization).toBeDefined()
    await authorization!.startAuthorization(12)

    expect(post).toHaveBeenNthCalledWith(
      2,
      `${SERVER_URL}/auth/115-qrcode-status`,
      {
        account_id: 12,
        uid: 'qr-uid',
      },
      {
        timeout: V115_QR_STATUS_TIMEOUT_MS,
      },
    )
    expect(V115_QR_STATUS_TIMEOUT_MS).toBeGreaterThan(60_000)

    scope.stop()
  })
})
