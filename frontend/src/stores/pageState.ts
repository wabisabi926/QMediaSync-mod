import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

type PageStatePrimitive = string | number | boolean | null

export interface PageState {
  currentPage: number
  pageSize: number
  filters: Record<string, PageStatePrimitive>
  expandedRowKeys: string[]
  scrollTop: number
}

type PageStateDefaults = Partial<Omit<PageState, 'filters'>> & {
  filters?: Record<string, PageStatePrimitive>
}

const storageKey = 'qmediasync-page-state'

function createPageStateMap(): Record<string, PageState> {
  return Object.create(null) as Record<string, PageState>
}

function createFilterMap(
  filters: Record<string, PageStatePrimitive> = {},
): Record<string, PageStatePrimitive> {
  const filterMap = Object.create(null) as Record<string, PageStatePrimitive>
  for (const [key, value] of Object.entries(filters)) {
    filterMap[key] = value
  }
  return filterMap
}

function hasOwnPageState(states: Record<string, PageState>, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(states, key)
}

function createPageState(defaults: PageStateDefaults = {}): PageState {
  return {
    currentPage: defaults.currentPage ?? 1,
    pageSize: defaults.pageSize ?? 20,
    filters: createFilterMap(defaults.filters),
    expandedRowKeys: [...(defaults.expandedRowKeys ?? [])],
    scrollTop: defaults.scrollTop ?? 0,
  }
}

function safeGetSessionStorage(): Storage | null {
  try {
    return globalThis.sessionStorage ?? null
  } catch {
    return null
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isPageStatePrimitive(value: unknown): value is PageStatePrimitive {
  return typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean' || value === null
}

function normalizePageState(value: unknown): PageState | null {
  if (!isRecord(value)) {
    return null
  }

  const filters = createFilterMap()
  if (isRecord(value.filters)) {
    for (const [key, filterValue] of Object.entries(value.filters)) {
      if (isPageStatePrimitive(filterValue)) {
        filters[key] = filterValue
      }
    }
  }

  return createPageState({
    currentPage: typeof value.currentPage === 'number' ? value.currentPage : undefined,
    pageSize: typeof value.pageSize === 'number' ? value.pageSize : undefined,
    filters,
    expandedRowKeys: Array.isArray(value.expandedRowKeys)
      ? value.expandedRowKeys.filter((rowKey): rowKey is string => typeof rowKey === 'string')
      : undefined,
    scrollTop: typeof value.scrollTop === 'number' ? value.scrollTop : undefined,
  })
}

function normalizeStoredStates(value: unknown): Record<string, PageState> {
  if (!isRecord(value)) {
    return createPageStateMap()
  }

  const states = createPageStateMap()
  for (const [key, stateValue] of Object.entries(value)) {
    const state = normalizePageState(stateValue)
    if (state) {
      states[key] = state
    }
  }
  return states
}

function readStoredStates(): Record<string, PageState> {
  const storage = safeGetSessionStorage()
  if (!storage) {
    return createPageStateMap()
  }

  try {
    const rawValue = storage.getItem(storageKey)
    if (!rawValue) {
      return createPageStateMap()
    }
    return normalizeStoredStates(JSON.parse(rawValue))
  } catch {
    try {
      storage.removeItem(storageKey)
    } catch {
      // 页面状态缓存为 best-effort，storage 异常时降级为内存态。
    }
    return createPageStateMap()
  }
}

export const usePageStateStore = defineStore('page-state', () => {
  const states = ref<Record<string, PageState>>(readStoredStates())

  const getPageState = (key: string, defaults: PageStateDefaults = {}) => {
    if (!hasOwnPageState(states.value, key)) {
      states.value[key] = createPageState(defaults)
    }
    return states.value[key]
  }

  const setPagination = (key: string, currentPage: number, pageSize: number) => {
    const state = getPageState(key)
    state.currentPage = currentPage
    state.pageSize = pageSize
  }

  const setFilter = (key: string, name: string, value: PageStatePrimitive) => {
    const state = getPageState(key)
    state.filters[name] = value
  }

  const setExpandedRowKeys = (key: string, rowKeys: string[]) => {
    getPageState(key).expandedRowKeys = [...new Set(rowKeys)]
  }

  const pruneExpandedRowKeys = (key: string, existingRowKeys: string[]) => {
    const existing = new Set(existingRowKeys)
    const state = getPageState(key)
    state.expandedRowKeys = state.expandedRowKeys.filter((rowKey) => existing.has(rowKey))
  }

  const setScrollTop = (key: string, scrollTop: number) => {
    getPageState(key).scrollTop = scrollTop
  }

  watch(
    states,
    (value) => {
      const storage = safeGetSessionStorage()
      if (storage) {
        try {
          storage.setItem(storageKey, JSON.stringify(value))
        } catch {
          // 页面状态缓存写入失败不应影响页面状态更新。
        }
      }
    },
    { deep: true },
  )

  return {
    states,
    getPageState,
    setPagination,
    setFilter,
    setExpandedRowKeys,
    pruneExpandedRowKeys,
    setScrollTop,
  }
})
