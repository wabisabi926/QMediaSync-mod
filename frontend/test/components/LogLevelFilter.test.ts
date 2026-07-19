// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import LogLevelFilter from '@/components/log/LogLevelFilter.vue'
import type { LogLevel } from '@/types/log'

describe('LogLevelFilter', () => {
  it('renders compact level chips and emits selected levels', async () => {
    const wrapper = mount(LogLevelFilter, {
      props: {
        modelValue: ['debug', 'info', 'warn', 'error'] satisfies LogLevel[],
        'onUpdate:modelValue': (value: LogLevel[]) => {
          void wrapper.setProps({ modelValue: value })
        },
      },
    })

    const chips = wrapper.findAll('.log-level-chip')
    expect(chips).toHaveLength(5)
    expect(chips.map((chip) => chip.text())).toEqual(['全部', 'Debug', 'Info', 'Warn', 'Error'])
    expect(chips.every((chip) => chip.attributes('type') === 'button')).toBe(true)

    await chips[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual(['info', 'warn', 'error'])

    await wrapper.setProps({ modelValue: ['info', 'warn', 'error'] satisfies LogLevel[] })
    await chips[0].trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([
      'debug',
      'info',
      'warn',
      'error',
    ])
  })
})
