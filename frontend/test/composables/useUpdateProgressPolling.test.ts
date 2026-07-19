import { describe, expect, it } from 'vitest'

const terminalStatuses = ['completed', 'failed', 'cancelled'] as const

describe('useUpdate 进度状态分类', () => {
  it.each(terminalStatuses)('把 %s 识别为终态', async (status) => {
    const mod = await import('../../src/composables/useUpdate')
    expect(mod.isUpdateTerminalStatus(status)).toBe(true)
  })

  it.each(['downloading', 'install'])('把 %s 识别为进行态', async (status) => {
    const mod = await import('../../src/composables/useUpdate')
    expect(mod.isUpdateRunningStatus(status)).toBe(true)
  })

  it('终态不属于进行态', async () => {
    const mod = await import('../../src/composables/useUpdate')
    expect(mod.isUpdateRunningStatus('completed')).toBe(false)
    expect(mod.isUpdateRunningStatus('failed')).toBe(false)
    expect(mod.isUpdateRunningStatus('cancelled')).toBe(false)
  })
})
