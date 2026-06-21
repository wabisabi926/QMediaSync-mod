export type StableRowKey = string | number

export function toStableRowKey(key: StableRowKey): string {
  return String(key)
}

export function mergeStableList<T extends object>(
  currentRows: T[],
  incomingRows: T[],
  getKey: (row: T) => StableRowKey,
): T[] {
  const currentByKey = new Map<string, T>()

  currentRows.forEach((row) => {
    currentByKey.set(toStableRowKey(getKey(row)), row)
  })

  return incomingRows.map((incomingRow) => {
    const key = toStableRowKey(getKey(incomingRow))
    const currentRow = currentByKey.get(key)
    if (!currentRow) {
      return incomingRow
    }

    Object.assign(currentRow, incomingRow)
    return currentRow
  })
}

export function retainExistingKeys<T extends object>(
  keys: string[],
  rows: T[],
  getKey: (row: T) => StableRowKey,
): string[] {
  const existingKeys = new Set(rows.map((row) => toStableRowKey(getKey(row))))
  return keys.filter((key) => existingKeys.has(key))
}
