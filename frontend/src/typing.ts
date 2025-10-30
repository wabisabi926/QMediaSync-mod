type defType = '本地路径' | 'WebDAV' | 'alist302'
type CopyMetaFileType = '关闭' | '复制' | '软链接'
type SyncType = '手动' | '定时' | '监控变更'
type CloudType = '115' | 'other'
type DeleteType = '是' | '否'

interface oo5Account {
  key: string
  name: string
  cookie: string
  created_at: string
  updated_at: string
  status: 0 | 1
}

interface LibForm {
  id_of_115: string
  key: string
  cloud_type: CloudType
  name: string
  path: string
  type: defType
  strm_root_path: string
  mount_path: string
  alist_server: string
  alist_115_path: string
  path_of_115: string
  copy_meta_file: CopyMetaFileType
  copy_delay: number
  webdav_url: string
  webdav_username: string
  webdav_password: string
  sync_type: SyncType
  cron_str: string
  meta_ext: Array<string>
  strm_ext: Array<string>
  delete: DeleteType
}
interface Lib extends LibForm {
  extra: {
    status: 1 | 2 | 3
    pid: string
    last_sync_at: string
    last_sync_result: {
      strm: Array<number>
      meta: Array<number>
      delete: Array<number>
    }
  }
}

interface Setting {
  username: string
  password: string
  telegram_bot_token?: string
  telegram_user_id?: string
}

interface BasicSetting {
  username: string
  password: string
}

interface TelegramSetting {
  enabled: boolean
  telegram_bot_token: string
  telegram_user_id: string
}

interface DirInfo {
  id: string
  name: string
  path: string
}

// 账户信息接口
interface CloudAccount {
  id: number
  name: string
  source_type: string
  user_id: string
  username: string
  created_at: number
  token: string
}

export type {
  oo5Account,
  LibForm,
  Lib,
  defType,
  CopyMetaFileType,
  SyncType,
  CloudType,
  Setting,
  BasicSetting,
  TelegramSetting,
  DirInfo,
  CloudAccount,
}
