import assert from 'node:assert/strict'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { describe, it } from 'vitest'

import { usePageStateStore } from '../../src/stores/pageState'

const storageKey = 'qmediasync-page-state'
const memory = new Map<string, string>()

function installSessionStorage(storage: Storage | object) {
  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    value: storage,
  })
}

function installThrowingSessionStorageGetter() {
  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    get() {
      throw new DOMException('storage disabled', 'SecurityError')
    },
  })
}

function installMemorySessionStorage() {
  memory.clear()
  installSessionStorage({
    getItem: (key: string) => memory.get(key) ?? null,
    setItem: (key: string, value: string) => memory.set(key, value),
    removeItem: (key: string) => memory.delete(key),
  })
}

function createStore() {
  setActivePinia(createPinia())
  return usePageStateStore()
}

function assertEmptyMap(value: Record<string, unknown>) {
  assert.deepEqual(Object.keys(value), [])
}

async function main() {
  installMemorySessionStorage()

  const store = createStore()
  const state = store.getPageState('download-queue', {
    currentPage: 2,
    pageSize: 20,
    filters: { status: -1 },
  })

  assert.equal(state.currentPage, 2)
  assert.equal(state.pageSize, 20)

  store.setPagination('download-queue', 3, 50)
  store.setFilter('download-queue', 'status', 1)
  store.setExpandedRowKeys('download-queue', ['1', '2', '3'])
  store.pruneExpandedRowKeys('download-queue', ['2', '3', '4'])
  store.setScrollTop('download-queue', 120)

  const updated = store.getPageState('download-queue')
  assert.equal(updated.currentPage, 3)
  assert.equal(updated.pageSize, 50)
  assert.equal(updated.filters.status, 1)
  assert.deepEqual(updated.expandedRowKeys, ['2', '3'])
  assert.equal(updated.scrollTop, 120)

  installThrowingSessionStorageGetter()
  assert.doesNotThrow(() => {
    const storageBlockedStore = createStore()
    const fallback = storageBlockedStore.getPageState('storage-blocked')
    assert.equal(fallback.currentPage, 1)
    assert.equal(fallback.pageSize, 20)
    assertEmptyMap(fallback.filters)
    assert.deepEqual(fallback.expandedRowKeys, [])
    assert.equal(fallback.scrollTop, 0)
  })

  installSessionStorage({
    getItem: () => null,
    setItem: () => {
      throw new DOMException('quota exceeded', 'QuotaExceededError')
    },
    removeItem: () => undefined,
  })
  const writeBlockedStore = createStore()
  writeBlockedStore.setPagination('write-blocked', 4, 100)
  writeBlockedStore.setFilter('write-blocked', 'keyword', 'movie')
  writeBlockedStore.setScrollTop('write-blocked', 240)
  await nextTick()

  const writeBlocked = writeBlockedStore.getPageState('write-blocked')
  assert.equal(writeBlocked.currentPage, 4)
  assert.equal(writeBlocked.pageSize, 100)
  assert.equal(writeBlocked.filters.keyword, 'movie')
  assert.equal(writeBlocked.scrollTop, 240)

  installMemorySessionStorage()
  memory.set(storageKey, JSON.stringify({ broken: null, legacy: { currentPage: 2 } }))
  const invalidStoredStore = createStore()

  const broken = invalidStoredStore.getPageState('broken')
  assert.equal(broken.currentPage, 1)
  assertEmptyMap(broken.filters)
  assert.equal(broken.scrollTop, 0)

  const legacy = invalidStoredStore.getPageState('legacy')
  assert.equal(legacy.currentPage, 2)
  assert.equal(legacy.pageSize, 20)
  assertEmptyMap(legacy.filters)
  assert.deepEqual(legacy.expandedRowKeys, [])
  assert.equal(legacy.scrollTop, 0)

  installMemorySessionStorage()
  const specialKeyStore = createStore()
  const toStringState = specialKeyStore.getPageState('toString', {
    currentPage: 6,
    filters: { status: 2 },
  })
  assert.equal(typeof toStringState, 'object')
  assert.equal(toStringState.currentPage, 6)
  assert.equal(toStringState.filters.status, 2)
  assert.doesNotThrow(() => specialKeyStore.setFilter('toString', 'keyword', 'movie'))
  assert.equal(specialKeyStore.getPageState('toString').filters.keyword, 'movie')

  installMemorySessionStorage()
  memory.set(
    storageKey,
    '{"__proto__":{"currentPage":7,"pageSize":30,"filters":{"status":3},"expandedRowKeys":["10"],"scrollTop":360}}',
  )
  const protoPageStore = createStore()
  assert.equal(Object.getPrototypeOf(protoPageStore.states), null)
  assert.equal(Object.prototype.hasOwnProperty.call(protoPageStore.states, '__proto__'), true)
  const protoPageState = protoPageStore.getPageState('__proto__')
  assert.equal(protoPageState.currentPage, 7)
  assert.equal(protoPageState.pageSize, 30)
  assert.equal(protoPageState.filters.status, 3)
  assert.deepEqual(protoPageState.expandedRowKeys, ['10'])
  assert.equal(protoPageState.scrollTop, 360)

  installMemorySessionStorage()
  memory.set(
    storageKey,
    '{"filters":{"filters":{"__proto__":"proto-filter","toString":"text-filter"}}}',
  )
  const specialFilterStore = createStore()
  const specialFilterState = specialFilterStore.getPageState('filters')
  assert.equal(Object.getPrototypeOf(specialFilterState.filters), null)
  assert.equal(Object.prototype.hasOwnProperty.call(specialFilterState.filters, '__proto__'), true)
  assert.equal(specialFilterState.filters.__proto__, 'proto-filter')
  assert.equal(specialFilterState.filters.toString, 'text-filter')
  assert.doesNotThrow(() => specialFilterStore.setFilter('filters', '__proto__', 'updated-proto'))
  assert.doesNotThrow(() => specialFilterStore.setFilter('filters', 'toString', 'updated-text'))
  assert.equal(specialFilterStore.getPageState('filters').filters.__proto__, 'updated-proto')
  assert.equal(specialFilterStore.getPageState('filters').filters.toString, 'updated-text')
}

describe('usePageStateStore', () => {
  it('在存储不可用、损坏数据和特殊键名下保持页面状态可用', async () => {
    await main()
  })
})
