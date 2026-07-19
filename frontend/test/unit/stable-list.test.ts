import assert from 'node:assert/strict'
import { describe, it } from 'vitest'

import { mergeStableList, retainExistingKeys } from '../../src/composables/useStableList'

type Row = {
  id: number
  status: string
  name: string
}

describe('稳定列表工具', () => {
  it('保留已有行引用并按当前列表过滤展开键', () => {
    const first: Row[] = [
      { id: 1, status: 'pending', name: 'A' },
      { id: 2, status: 'running', name: 'B' },
    ]

    const originalFirstRow = first[0]
    const merged = mergeStableList(
      first,
      [
        { id: 1, status: 'done', name: 'A1' },
        { id: 3, status: 'pending', name: 'C' },
      ],
      (row) => row.id,
    )

    assert.equal(merged.length, 2)
    assert.equal(merged[0], originalFirstRow)
    assert.deepEqual(merged[0], { id: 1, status: 'done', name: 'A1' })
    assert.deepEqual(merged[1], { id: 3, status: 'pending', name: 'C' })
    assert.deepEqual(
      retainExistingKeys(['1', '2', '3'], merged, (row) => row.id),
      ['1', '3'],
    )
  })
})
