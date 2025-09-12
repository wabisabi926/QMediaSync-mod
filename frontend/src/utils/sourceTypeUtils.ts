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
    label: '本地目录',
    value: 'local',
  },
]

export const sourceTypeTagMap: Record<string, string> = {
  '115': 'success',
  // '123': 'primary',
  local: 'info',
}

export const sourceTypeMap: Record<string, string> = {
  '115': '115网盘',
  // '123': '123云盘',
  local: '本地目录',
}
