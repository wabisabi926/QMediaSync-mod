export interface SyncRefreshDecisionInput {
  createdStrm: number
  downloadedMeta: number
  status?: number
}

export type SyncRefreshDecisionTagType = 'success' | 'info'

export interface SyncRefreshDecision {
  willRequestRefresh: boolean
  label: string
  type: SyncRefreshDecisionTagType
  reason: string
}

export const getEmbyRefreshDecision = ({
  createdStrm,
  downloadedMeta,
  status = 2,
}: SyncRefreshDecisionInput): SyncRefreshDecision => {
  if (createdStrm > 0 || downloadedMeta > 0) {
    return {
      willRequestRefresh: true,
      label: status === 2 ? '已提交媒体库刷新' : '完成后将提交媒体库刷新',
      type: 'success',
      reason: '有新增 STRM 或下载元数据',
    }
  }

  return {
    willRequestRefresh: false,
    label: status === 2 ? '未提交媒体库刷新' : '暂无媒体库刷新变更',
    type: 'info',
    reason: '无新增 STRM 或下载元数据',
  }
}
