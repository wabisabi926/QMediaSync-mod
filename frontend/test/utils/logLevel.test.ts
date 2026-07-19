import { describe, expect, it } from 'vitest'

import {
  DEFAULT_VISIBLE_LOG_LEVELS,
  LOG_LEVEL_OPTIONS,
  filterLogEntriesByLevels,
} from '@/utils/logLevel'

describe('logLevel utils', () => {
  const entries = [
    { level: 'debug', message: 'debug' },
    { level: 'info', message: 'info' },
    { level: 'warn', message: 'warn' },
    { level: 'error', message: 'error' },
  ] as const

  it('keeps all levels visible by default', () => {
    const got = filterLogEntriesByLevels(entries, DEFAULT_VISIBLE_LOG_LEVELS)

    expect(got.map((entry) => entry.message)).toEqual(['debug', 'info', 'warn', 'error'])
  })

  it('filters entries by selected levels', () => {
    const got = filterLogEntriesByLevels(entries, ['warn', 'error'])

    expect(got.map((entry) => entry.message)).toEqual(['warn', 'error'])
  })

  it('exposes stable level option order', () => {
    expect(LOG_LEVEL_OPTIONS.map((option) => option.value)).toEqual([
      'debug',
      'info',
      'warn',
      'error',
    ])
  })
})
