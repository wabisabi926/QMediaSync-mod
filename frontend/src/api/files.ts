import type { AxiosInstance } from 'axios'
import { SERVER_URL } from '@/const'
import type { DirInfo } from '@/typing'
import { unwrapResponse } from './response'
import type { APIResponse } from './types'

export type NetFileSortBy = 'default' | 'name' | 'time' | 'size' | 'type'
export type NetFileSortOrder = 'asc' | 'desc'
export type BrowseScope = 'files' | 'directories'

export interface BrowseSortValue {
  sort_by: NetFileSortBy
  sort_order: NetFileSortOrder
  folders_first?: boolean
}

export interface BrowseSortOptions {
  fields: NetFileSortBy[]
  folders_first: boolean
  default: BrowseSortValue
}

export async function fetchBrowseSortOptions(
  http: AxiosInstance,
  source_type: string,
  scope: BrowseScope,
) {
  const options = unwrapResponse(
    await http.get<APIResponse<BrowseSortOptions>>(`${SERVER_URL}/path/sort-options`, {
      params: { source_type, scope },
    }),
  )
  if (
    !options ||
    !Array.isArray(options.fields) ||
    !options.fields.length ||
    !options.default ||
    !options.fields.includes(options.default.sort_by) ||
    !['asc', 'desc'].includes(options.default.sort_order) ||
    typeof options.folders_first !== 'boolean'
  ) {
    throw new Error('排序能力响应不完整')
  }
  return options
}

export interface DirectoryQuery {
  parent_id: string
  parent_path: string
  source_type: string
  account_id: number
}

export interface DirectoryListQuery extends DirectoryQuery {
  sort_by?: NetFileSortBy
  sort_order?: NetFileSortOrder
  refresh?: 0 | 1
}

export interface CreateDirectoryPayload extends DirectoryQuery {
  name: string
}

export interface ManualStrmPayload {
  path_id: string
  target_path: string
  account_id: number
}

// 仅改写需要本地化的业务字段和文案，其他原因使用服务端消息。
export const filePublicMessages: Readonly<Record<string, string>> = {
  'account_id：必须大于 0': '请先选择网盘账号',
  'source_type：不是允许的取值': '未知的同步源类型',
  'path_id：不能为空': '请选择源文件或目录',
  'target_path：不能为空': '请选择目标目录',
  'name：不能为空': '请输入文件夹名称',
  'name：文件夹名不合法': '文件夹名不合法',
  'name：不能包含路径分隔符': '文件夹名不能包含路径分隔符',
  'name：不能包含控制字符': '文件夹名不能包含控制字符',
  'sort_by：不支持的排序字段': '不支持的排序字段',
  'sort_order：不支持的排序方向': '不支持的排序方向',
}

export async function fetchDirectories(
  http: AxiosInstance,
  params: DirectoryListQuery,
  signal?: AbortSignal,
) {
  return unwrapResponse(
    await http.get<APIResponse<DirInfo[] | null>>(`${SERVER_URL}/path/list`, {
      timeout: 60000,
      params,
      ...(signal ? { signal } : {}),
    }),
  )
}

export async function createDirectory(http: AxiosInstance, payload: CreateDirectoryPayload) {
  return unwrapResponse(await http.post<APIResponse<DirInfo>>(`${SERVER_URL}/path/create`, payload))
}

export async function generateManualStrm(
  http: AxiosInstance,
  payload: ManualStrmPayload,
): Promise<void> {
  unwrapResponse(await http.post<APIResponse<null>>(`${SERVER_URL}/sync/manual`, payload))
}
