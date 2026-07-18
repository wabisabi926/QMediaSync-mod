import {
  saveSyncPathAggregate,
  type SaveSyncPathPayload,
  type SaveSyncPathResponseData,
  type SyncPathFieldError,
} from '@/api/syncPaths'
import type { AxiosError, AxiosInstance } from 'axios'
import { readonly, ref } from 'vue'

interface APIEnvelope<T> {
  code: number
  message: string
  data: T
}

interface SaveErrorData {
  error_code?: string
  field_errors?: SyncPathFieldError[]
}

export function useSyncDirectorySave(http: AxiosInstance) {
  const saving = ref(false)
  const errorMessage = ref('')
  const errorCode = ref('')
  const fieldErrors = ref<SyncPathFieldError[]>([])
  const warnings = ref<string[]>([])

  async function save(
    id: number,
    payload: SaveSyncPathPayload,
    idempotencyKey: string,
  ): Promise<SaveSyncPathResponseData | null> {
    errorMessage.value = ''
    errorCode.value = ''
    fieldErrors.value = []
    warnings.value = []
    saving.value = true
    try {
      const response = await saveSyncPathAggregate(http, id, payload, idempotencyKey)
      const envelope = response.data as APIEnvelope<SaveSyncPathResponseData | SaveErrorData>
      if (envelope.code !== 200) {
        const data = envelope.data as SaveErrorData
        errorMessage.value = envelope.message || '保存同步目录失败'
        errorCode.value = data?.error_code || ''
        fieldErrors.value = data?.field_errors || []
        return null
      }
      const data = envelope.data as SaveSyncPathResponseData
      warnings.value = data.warnings || []
      return data
    } catch (error) {
      const response = (error as AxiosError<APIEnvelope<SaveErrorData>>).response?.data
      errorMessage.value = response?.message || '保存同步目录失败'
      errorCode.value = response?.data?.error_code || ''
      fieldErrors.value = response?.data?.field_errors || []
      return null
    } finally {
      saving.value = false
    }
  }

  async function saveAndRun(
    id: number,
    payload: SaveSyncPathPayload,
    idempotencyKey: string,
    onSuccess: (data: SaveSyncPathResponseData) => void | Promise<void>,
  ): Promise<SaveSyncPathResponseData | null> {
    const result = await save(id, payload, idempotencyKey)
    if (!result) {
      return null
    }
    await onSuccess(result)
    return result
  }

  return {
    saving: readonly(saving),
    errorMessage: readonly(errorMessage),
    errorCode: readonly(errorCode),
    fieldErrors: readonly(fieldErrors),
    warnings: readonly(warnings),
    save,
    saveAndRun,
  }
}
