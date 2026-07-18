import { SERVER_URL } from '@/const'
import type { AxiosInstance } from 'axios'

export interface SetupStatus {
  required: boolean
}

export interface CreateInitialAdminPayload {
  setup_token: string
  username: string
  password: string
}

export const fetchSetupStatus = async (http: AxiosInstance): Promise<SetupStatus> => {
  const response = await http.get(`${SERVER_URL}/setup/status`)
  if (response.data?.code !== 200) {
    throw new Error(response.data?.message || '查询初始化状态失败')
  }
  return response.data.data
}

export const createInitialAdmin = async (
  http: AxiosInstance,
  payload: CreateInitialAdminPayload,
): Promise<void> => {
  const response = await http.post(`${SERVER_URL}/setup/admin`, payload)
  if (response.data?.code !== 200) {
    throw new Error(response.data?.message || '创建管理员失败')
  }
}
