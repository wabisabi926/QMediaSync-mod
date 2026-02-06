export const sourceTypeOptions: Array<Record<string, string>> = [
  {
    label: '115网盘',
    value: '115',
  },
  // {
  //   label: '123云盘',
  //   value: '123',
  // },
    {
    label: '百度网盘',
    value: 'baidupan',
  },
  {
    label: 'OpenList',
    value: 'openlist',
  },
  {
    label: '本地目录',
    value: 'local',
  },
]

export const sourceTypeTagMap: Record<string, string> = {
  '115': 'success',
  // '123': 'primary',
  'baidupan': 'danger',
  'openlist': 'warning',
  'local': 'info',
}

export const sourceTypeMap: Record<string, string> = {
  '115': '115网盘',
  // '123': '123云盘',
  'baidupan': '百度网盘',
  openlist: 'OpenList',
  local: '本地目录',
}
