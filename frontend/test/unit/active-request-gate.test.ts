import { describe, expect, it } from 'vitest'

import { createActiveRequestGate } from '../../src/composables/useActiveRequestGate'

describe('createActiveRequestGate', () => {
  it('只接受页面仍活跃时的最新请求', () => {
    let active = true
    const gate = createActiveRequestGate(() => active)

    const firstRequest = gate.next()
    expect(gate.isCurrent(firstRequest)).toBe(true)

    const secondRequest = gate.next()
    expect(gate.isCurrent(firstRequest)).toBe(false)
    expect(gate.isCurrent(secondRequest)).toBe(true)

    active = false
    expect(gate.isCurrent(secondRequest)).toBe(false)

    active = true
    const thirdRequest = gate.next()
    gate.invalidate()
    expect(gate.isCurrent(thirdRequest)).toBe(false)

    const fourthRequest = gate.next()
    expect(gate.isCurrent(fourthRequest)).toBe(true)
  })
})
