import type { LogLevel } from '@/types/log'

export interface LogLevelOption {
  value: LogLevel
  label: string
}

export const LOG_LEVEL_OPTIONS: readonly LogLevelOption[] = [
  { value: 'debug', label: 'Debug' },
  { value: 'info', label: 'Info' },
  { value: 'warn', label: 'Warn' },
  { value: 'error', label: 'Error' },
]

export const DEFAULT_VISIBLE_LOG_LEVELS: LogLevel[] = LOG_LEVEL_OPTIONS.map(
  (option) => option.value,
)

export function isLogLevel(value: unknown): value is LogLevel {
  return LOG_LEVEL_OPTIONS.some((option) => option.value === value)
}

export function filterLogEntriesByLevels<T extends { level: LogLevel }>(
  entries: readonly T[],
  levels: readonly LogLevel[],
): T[] {
  if (levels.length === 0) {
    return []
  }
  const selectedLevels = new Set(levels)
  return entries.filter((entry) => selectedLevels.has(entry.level))
}
