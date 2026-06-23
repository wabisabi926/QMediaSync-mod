import { SERVER_URL } from '@/const'
import { effectScope } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useV115AppIdSearch } from './useV115AppIdSearch'

describe('useV115AppIdSearch', () => {
  it('加载更多使用当前关键词和 offset 追加下一页结果', async () => {
    const get = vi
      .fn()
      .mockResolvedValueOnce({
        data: {
          data: {
            items: [{ app_id: '1001', app_name: '应用1', display_name: '应用1' }],
            total: 2,
          },
        },
      })
      .mockResolvedValueOnce({
        data: {
          data: {
            items: [{ app_id: '1002', app_name: '应用2', display_name: '应用2' }],
            total: 2,
          },
        },
      })

    const scope = effectScope()
    const search = scope.run(() => useV115AppIdSearch({ http: { get } as never, pageSize: 1 }))

    expect(search).toBeDefined()
    search!.keyword.value = '应用'
    await search!.search()

    expect(search!.hasMore.value).toBe(true)
    await search!.loadMore()

    expect(get).toHaveBeenNthCalledWith(1, `${SERVER_URL}/115/appids`, {
      params: { keyword: '应用', offset: 0, limit: 1 },
    })
    expect(get).toHaveBeenNthCalledWith(2, `${SERVER_URL}/115/appids`, {
      params: { keyword: '应用', offset: 1, limit: 1 },
    })
    expect(search!.items.value.map((item) => item.app_id)).toEqual(['1001', '1002'])
    expect(search!.hasMore.value).toBe(false)

    scope.stop()
  })
})
