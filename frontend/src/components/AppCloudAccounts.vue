<template>
  <div class="cloud-accounts-page">
    <div class="page-header">
      <div class="header-content">
        <div class="header-title-section">
          <h1 class="page-title">
            <el-icon class="title-icon"><Cloudy /></el-icon>
            网盘账号管理
          </h1>
          <p class="page-subtitle">管理您的网盘账号授权与绑定</p>
        </div>
        <div class="header-actions">
          <el-button type="primary" class="add-btn" @click="showAddAccountDialog = true">
            <el-icon><Plus /></el-icon>
            <span class="btn-text">添加账号</span>
          </el-button>
        </div>
      </div>
      <div class="stats-bar mobile-hidden">
        <div class="stat-item">
          <div class="stat-icon total">
            <el-icon><User /></el-icon>
          </div>
          <div class="stat-info">
            <span class="stat-value">{{ accounts.length }}</span>
            <span class="stat-label">总账号数</span>
          </div>
        </div>
        <div class="stat-item">
          <div class="stat-icon authorized">
            <el-icon><CircleCheck /></el-icon>
          </div>
          <div class="stat-info">
            <span class="stat-value">{{ authorizedCount }}</span>
            <span class="stat-label">已授权</span>
          </div>
        </div>
        <div class="stat-item">
          <div class="stat-icon unauthorized">
            <el-icon><WarningFilled /></el-icon>
          </div>
          <div class="stat-info">
            <span class="stat-value">{{ unauthorizedCount }}</span>
            <span class="stat-label">未授权</span>
          </div>
        </div>
        <div class="stat-item">
          <div class="stat-icon failed">
            <el-icon><CircleClose /></el-icon>
          </div>
          <div class="stat-info">
            <span class="stat-value">{{ failedCount }}</span>
            <span class="stat-label">授权失败</span>
          </div>
        </div>
      </div>
    </div>

    <div class="accounts-content">
      <div class="accounts-grid" v-if="accounts.length > 0">
        <div
          class="account-card"
          v-for="account in accounts"
          :key="account.id"
          :class="getCardStatusClass(account)"
        >
          <div class="card-status-bar" :class="getStatusClass(account)"></div>
          <div class="card-main">
            <div class="card-header">
              <div class="card-title-wrapper">
                <el-tooltip :content="'账号ID：' + account.id" placement="bottom">
                  <span class="card-id">#{{ account.id }}</span>
                </el-tooltip>
                <span class="card-name">{{ account.name }}</span>
              </div>
              <el-tag :type="sourceTypeTagMap[account.source_type]" class="source-tag" effect="light">
                {{ sourceTypeMap[account.source_type] }}
              </el-tag>
            </div>

            <div class="card-body">
              <div class="info-row" v-if="account.source_type === '115'">
                <div class="info-icon">
                  <el-icon><User /></el-icon>
                </div>
                <div class="info-content">
                  <span class="info-label">115账号</span>
                  <span class="info-value">{{ account.username || '-' }}</span>
                </div>
              </div>

              <div class="info-row" v-if="account.source_type === '115'">
                <div class="info-icon">
                  <el-icon><Key /></el-icon>
                </div>
                <div class="info-content">
                  <span class="info-label">开放平台应用</span>
                  <span class="info-value">{{ account.app_id_name || '-' }}</span>
                </div>
              </div>

              <div class="info-row" v-if="account.source_type === 'baidupan'">
                <div class="info-icon">
                  <el-icon><Key /></el-icon>
                </div>
                <div class="info-content">
                  <span class="info-label">APP ID</span>
                  <span class="info-value">{{ account.app_id || '-' }}</span>
                </div>
              </div>

              <template v-if="account.source_type === 'openlist'">
                <div class="info-row">
                  <div class="info-icon">
                    <el-icon><Link /></el-icon>
                  </div>
                  <div class="info-content">
                    <span class="info-label">访问地址</span>
                    <span class="info-value path-value">{{ account.base_url }}</span>
                  </div>
                </div>
                <div class="info-row">
                  <div class="info-icon">
                    <el-icon><User /></el-icon>
                  </div>
                  <div class="info-content">
                    <span class="info-label">用户名</span>
                    <span class="info-value">{{ account.name }}</span>
                  </div>
                </div>
                <div class="info-row">
                  <div class="info-icon">
                    <el-icon><Postcard /></el-icon>
                  </div>
                  <div class="info-content">
                    <span class="info-label">用户ID</span>
                    <span class="info-value">{{ account.user_id || '-' }}</span>
                  </div>
                </div>
              </template>

              <div class="info-row">
                <div class="info-icon">
                  <el-icon><Calendar /></el-icon>
                </div>
                <div class="info-content">
                  <span class="info-label">添加时间</span>
                  <span class="info-value">{{ formatTimestamp(account.created_at) }}</span>
                </div>
              </div>

              <template v-if="(account.source_type === '115' || account.source_type === 'baidupan') && account.token">
                <div class="status-divider"></div>
                <div class="disk-status-section">
                  <div class="status-header">
                    <span class="status-title">网盘状态</span>
                    <el-button
                      type="primary"
                      size="small"
                      text
                      :loading="account.statusLoading"
                      @click="loadAccountStatus(account)"
                    >
                      <el-icon v-if="!account.statusLoading"><RefreshRight /></el-icon>
                      刷新
                    </el-button>
                  </div>
                  <template v-if="account.status">
                    <div class="info-row">
                      <div class="info-icon">
                        <el-icon><User /></el-icon>
                      </div>
                      <div class="info-content">
                        <span class="info-label">网盘用户</span>
                        <span class="info-value">{{ account.status.username || '-' }}</span>
                      </div>
                    </div>
                    <div class="info-row">
                      <div class="info-icon space-icon">
                        <el-icon><Cloudy /></el-icon>
                      </div>
                      <div class="info-content">
                        <span class="info-label">空间使用</span>
                        <div class="space-info">
                          <el-progress
                            :percentage="account.status.total_space > 0 ? Math.round((account.status.used_space / account.status.total_space) * 100) : 0"
                            :stroke-width="6"
                            :show-text="false"
                            :color="getSpaceColor(account.status.used_space, account.status.total_space)"
                          />
                          <span class="space-text">{{ formatFileSize(account.status.used_space) }} / {{ formatFileSize(account.status.total_space) }}</span>
                        </div>
                      </div>
                    </div>
                    <div class="info-row" v-if="account.status.member_level">
                      <div class="info-icon member-icon">
                        <el-icon><Postcard /></el-icon>
                      </div>
                      <div class="info-content">
                        <span class="info-label">会员等级</span>
                        <el-tag size="small" type="warning" effect="plain">{{ account.status.member_level }}</el-tag>
                      </div>
                    </div>
                    <div class="info-row" v-if="account.status.expire_time && account.status.expire_time !== '0001-01-01T00:00:00Z'">
                      <div class="info-icon expire-icon">
                        <el-icon><Calendar /></el-icon>
                      </div>
                      <div class="info-content">
                        <span class="info-label">到期时间</span>
                        <span class="info-value" :class="{ 'expire-warning': isExpiringSoon(account.status.expire_time) }">
                          {{ formatExpireTime(account.status.expire_time) }}
                        </span>
                      </div>
                    </div>
                  </template>
                  <template v-else-if="!account.statusLoading">
                    <div class="no-status">暂无状态信息</div>
                  </template>
                </div>
              </template>

              <div class="status-row">
                <div class="status-indicator" :class="getStatusClass(account)">
                  <el-icon v-if="account.token_failed_reason && !account.token"><CircleClose /></el-icon>
                  <el-icon v-else-if="account.token"><CircleCheck /></el-icon>
                  <el-icon v-else><WarningFilled /></el-icon>
                  <span>{{ getStatusText(account) }}</span>
                </div>
                <el-tooltip v-if="account.token_failed_reason && !account.token" :content="account.token_failed_reason" placement="top">
                  <el-icon class="error-help-icon"><QuestionFilled /></el-icon>
                </el-tooltip>
              </div>
            </div>

            <div class="card-footer">
              <el-button
                type="danger"
                size="small"
                plain
                @click="handleDelete(account)"
              >
                <el-icon><Delete /></el-icon>
                删除
              </el-button>

              <el-button
                type="warning"
                size="small"
                plain
                @click="handleAuthorize(account)"
                v-if="account.source_type !== 'openlist'"
              >
                <el-icon><Key /></el-icon>
                授权
              </el-button>

              <el-button
                type="primary"
                size="small"
                plain
                @click="handleEdit(account)"
                v-if="account.source_type === 'openlist'"
              >
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <div class="empty-state" v-else-if="!loading">
        <div class="empty-illustration">
          <el-icon class="empty-icon"><Cloudy /></el-icon>
          <div class="empty-dots">
            <span></span>
            <span></span>
            <span></span>
          </div>
        </div>
        <h3 class="empty-title">暂无网盘账号</h3>
        <p class="empty-description">点击上方按钮添加您的第一个网盘账号</p>
        <el-button type="primary" @click="showAddAccountDialog = true">
          <el-icon><Plus /></el-icon>
          添加账号
        </el-button>
      </div>

      <div class="loading-state" v-if="loading">
        <el-icon class="loading-icon rotating"><Loading /></el-icon>
        <span>加载中...</span>
      </div>

      <div class="page-footer-tips">
        <div class="tips-header">
          <el-icon class="tips-icon"><InfoFilled /></el-icon>
          <span>使用说明</span>
        </div>
        <div class="tips-content">
          <div class="tip-group">
            <div class="tip-group-title">
              <el-icon><Warning /></el-icon>
              <span>操作流程</span>
            </div>
            <div class="tip-group-items">
              <div class="tip-item">
                <span class="tip-bullet">1.</span>
                <span>点击"添加账号"按钮，选择网盘类型并填写相关信息</span>
              </div>
              <div class="tip-item">
                <span class="tip-bullet">2.</span>
                <span>添加成功后，点击列表中的"授权"按钮完成账号绑定</span>
              </div>
              <div class="tip-item tip-highlight">
                <span class="tip-bullet">★</span>
                <span>只有已授权的账号才能用于STRM同步目录配置</span>
              </div>
            </div>
          </div>
          <div class="tip-group">
            <div class="tip-group-title">
              <el-icon><Key /></el-icon>
              <span>授权说明</span>
            </div>
            <div class="tip-group-items">
              <div class="tip-item">
                <span class="tip-bullet">•</span>
                <span>115网盘：通过OAuth授权，会打开新窗口进行授权</span>
              </div>
              <div class="tip-item">
                <span class="tip-bullet">•</span>
                <span>百度网盘：通过OAuth授权，会打开新窗口进行授权</span>
              </div>
              <div class="tip-item">
                <span class="tip-bullet">•</span>
                <span>OpenList：支持用户名密码或令牌两种认证方式</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <el-dialog v-model="showAddAccountDialog" title="添加账号" :width="isMobile ? '90%' : '500px'">
    <el-form :model="newAccountForm" label-width="120px">
      <el-form-item label="网盘类型">
        <el-select v-model="newAccountForm.type" placeholder="请选择网盘类型">
          <template v-for="typeItem in sourceTypeOptions" :key="typeItem.value">
            <el-option v-if="typeItem.value !== 'local'" :label="typeItem.label" :value="typeItem.value"></el-option>
          </template>
        </el-select>
      </el-form-item>
      <el-form-item label="账号备注" v-if="newAccountForm.type !== 'openlist'">
        <el-input v-model="newAccountForm.name" placeholder="请输入账号备注" />
      </el-form-item>
      <el-form-item label="访问地址" v-if="newAccountForm.type === 'openlist'">
        <el-input v-model="newAccountForm.base_url" placeholder="请输入OpenList地址:http://ip:5244" />
      </el-form-item>
      <el-form-item label="认证方式" v-if="newAccountForm.type === 'openlist'">
        <el-select v-model="newAccountForm.auth_type" placeholder="请选择认证方式">
          <el-option label="用户名密码" value="password"></el-option>
          <el-option label="令牌" value="token"></el-option>
        </el-select>
      </el-form-item>
      <template v-if="newAccountForm.type === 'openlist' && newAccountForm.auth_type === 'password'">
        <el-form-item label="用户名">
          <el-input v-model="newAccountForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input type="password" v-model="newAccountForm.password" placeholder="请输入密码" />
        </el-form-item>
      </template>
      <el-form-item label="令牌" v-if="newAccountForm.type === 'openlist' && newAccountForm.auth_type === 'token'">
        <el-input type="password" v-model="newAccountForm.token" placeholder="请输入令牌" />
      </el-form-item>
      <el-form-item label="115开放平台应用" v-if="newAccountForm.type === '115'">
        <el-select v-model="newAccountForm.app_id_name" placeholder="请选择APP ID">
          <el-option label="Q115-STRM" value="Q115-STRM"></el-option>
          <el-option label="MQ的媒体库" value="MQ的媒体库"></el-option>
        </el-select>
      </el-form-item>
      <el-form-item label="APP ID" v-if="newAccountForm.type === '115' && newAccountForm.app_id_name === '自定义'">
        <el-input v-model="newAccountForm.app_id" placeholder="请输入自定义APP ID" />
      </el-form-item>
    </el-form>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="showAddAccountDialog = false">取消</el-button>
        <el-button type="primary" @click="handleAddAccount" :loading="addAccountLoading">确定</el-button>
      </span>
    </template>
  </el-dialog>

  <el-dialog v-model="showEditAccountDialog" title="编辑OpenList账号" :width="isMobile ? '90%' : '500px'">
    <el-form :model="editAccountForm" label-width="80px">
      <el-form-item label="访问地址" prop="baseUrl">
        <el-input v-model="editAccountForm.base_url" placeholder="请输入OpenList地址:http://ip:5244" />
      </el-form-item>
      <el-form-item label="认证方式">
        <el-select v-model="editAccountForm.auth_type" placeholder="请选择认证方式">
          <el-option label="用户名密码" value="password"></el-option>
          <el-option label="令牌" value="token"></el-option>
        </el-select>
      </el-form-item>
      <template v-if="editAccountForm.auth_type === 'password'">
        <el-form-item label="用户名">
          <el-input v-model="editAccountForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input type="password" v-model="editAccountForm.password" placeholder="请输入密码（留空则不修改）" />
        </el-form-item>
      </template>
      <el-form-item label="令牌" v-if="editAccountForm.auth_type === 'token'">
        <el-input type="password" v-model="editAccountForm.token" placeholder="请输入令牌" />
      </el-form-item>
    </el-form>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="showEditAccountDialog = false">取消</el-button>
        <el-button type="primary" @click="handleUpdateAccount">确定</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosError, AxiosStatic } from 'axios'
import { inject, ref, computed, onMounted, onUnmounted } from 'vue'

import {
  WarningFilled,
  Plus,
  Loading,
  User,
  Key,
  Link,
  Calendar,
  Delete,
  Edit,
  Cloudy,
  CircleCheck,
  CircleClose,
  InfoFilled,
  QuestionFilled,
  Postcard,
  RefreshRight,
} from '@element-plus/icons-vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { formatTimestamp } from '@/utils/timeUtils'
import { sourceTypeMap, sourceTypeOptions, sourceTypeTagMap } from '@/utils/sourceTypeUtils'
import { isMobile as checkIsMobile, onDeviceTypeChange } from '@/utils/deviceUtils'

const isMobile = ref(checkIsMobile())

interface CloudDiskStatus {
  user_id: string
  username: string
  used_space: number
  total_space: number
  member_level: string
  expire_time: string
}

interface CloudAccount {
  id: number
  source_type: string
  name: string
  user_id: string
  username: string
  password: string
  base_url: string
  created_at: number
  token: string
  auth_type?: string
  app_id_name?: string
  app_id?: string
  token_failed_reason?: string
  status?: CloudDiskStatus
  statusLoading?: boolean
}

const http: AxiosStatic | undefined = inject('$http')

const accounts = ref<CloudAccount[]>([])
const loading = ref(false)
const showAddAccountDialog = ref(false)
const addAccountLoading = ref(false)

const newAccountForm = ref({
  type: '',
  name: '',
  base_url: '',
  username: '',
  password: '',
  token: '',
  auth_type: 'password',
  app_id_name: 'Q115-STRM',
  app_id: '',
})

const showEditAccountDialog = ref(false)
const currentEditAccount = ref<CloudAccount | null>(null)
const editAccountForm = ref({
  id: 0,
  source_type: '',
  base_url: '',
  username: '',
  password: '',
  token: '',
  auth_type: 'password',
  token_failed_reason: '',
})

const currentAccountId = ref<number | undefined>(undefined)

const authorizedCount = computed(() => accounts.value.filter(a => a.token && !a.token_failed_reason).length)
const unauthorizedCount = computed(() => accounts.value.filter(a => !a.token && !a.token_failed_reason).length)
const failedCount = computed(() => accounts.value.filter(a => a.token_failed_reason && !a.token).length)

const getStatusClass = (account: CloudAccount) => {
  if (account.token_failed_reason && !account.token) return 'status-failed'
  if (account.token) return 'status-authorized'
  return 'status-unauthorized'
}

const getStatusText = (account: CloudAccount) => {
  if (account.token_failed_reason && !account.token) return '授权失败'
  if (account.token) return '已授权'
  return '未授权'
}

const getCardStatusClass = (account: CloudAccount) => {
  if (account.token_failed_reason && !account.token) return 'is-failed'
  if (account.token) return 'is-authorized'
  return 'is-unauthorized'
}

const loadAccounts = async () => {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/account/list`)

    if (response?.data.code === 200) {
      const data = response.data.data
      accounts.value = data.map((item: CloudAccount) => ({
        id: item.id,
        source_type: item.source_type,
        name: item.name,
        user_id: item.user_id,
        username: item.username,
        created_at: item.created_at,
        token: item.token,
        base_url: item.base_url,
        password: item.password,
        auth_type: item.auth_type,
        app_id_name: item.app_id_name,
        app_id: item.app_id,
        token_failed_reason: item.token_failed_reason || '',
        status: undefined,
        statusLoading: false,
      }))
      accounts.value.forEach(account => {
        if ((account.source_type === '115' || account.source_type === 'baidupan') && account.token) {
          loadAccountStatus(account)
        }
      })
    } else {
      console.error('加载账号列表失败:', response?.data.message || '未知错误')
      accounts.value = []
    }
  } catch (error) {
    console.error('加载账号列表失败:', error)
    accounts.value = []
  } finally {
    loading.value = false
  }
}

const loadAccountStatus = async (account: CloudAccount) => {
  const index = accounts.value.findIndex(a => a.id === account.id)
  if (index === -1) return

  accounts.value[index].statusLoading = true

  try {
    let url = ''
    if (account.source_type === '115') {
      url = `${SERVER_URL}/115/status`
    } else if (account.source_type === 'baidupan') {
      url = `${SERVER_URL}/baidupan/status`
    } else {
      return
    }

    const response = await http?.get(url, {
      params: { account_id: account.id }
    })

    if (response?.data.code === 200 && response.data.data) {
      accounts.value[index].status = response.data.data
    }
  } catch (error) {
    console.error(`获取${account.source_type}状态失败:`, error)
  } finally {
    accounts.value[index].statusLoading = false
  }
}

const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + units[i]
}

const formatExpireTime = (expireTime: string): string => {
  if (!expireTime || expireTime === '0001-01-01T00:00:00Z' || expireTime === '1970-01-01T00:00:00Z') {
    return '-'
  }
  try {
    const date = new Date(expireTime)
    if (isNaN(date.getTime())) return '-'
    return date.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
  } catch {
    return '-'
  }
}

const getSpaceColor = (used: number, total: number): string => {
  if (total === 0) return '#67c23a'
  const percentage = used / total
  if (percentage < 0.5) return '#67c23a'
  if (percentage < 0.8) return '#e6a23c'
  return '#f56c6c'
}

const isExpiringSoon = (expireTime: string): boolean => {
  if (!expireTime) return false
  try {
    const date = new Date(expireTime)
    if (isNaN(date.getTime())) return false
    const now = new Date()
    const diffDays = Math.ceil((date.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
    return diffDays <= 30
  } catch {
    return false
  }
}

const handleDelete = async (row: CloudAccount) => {
  try {
    await ElMessageBox.confirm(`确定要删除账号 "${row.name}" 吗？此操作不可恢复。`, '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    const response = await http?.post(
      `${SERVER_URL}/account/delete`,
      { id: row.id },
      {
        headers: {
          'Content-Type': 'application/json',
        },
      },
    )

    if (response?.data.code === 200) {
      ElMessage.success('账号删除成功')
      loadAccounts()
    } else {
      ElMessage.error(response?.data.message || '删除账号失败')
    }
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      console.error('删除账号失败:', error)
      ElMessage.error('删除账号失败')
    }
  }
}

const handleEdit = (account: CloudAccount) => {
  currentEditAccount.value = account

  const authType =
    account.auth_type || (account.username && account.password ? 'password' : 'token')

  editAccountForm.value = {
    id: account.id,
    source_type: account.source_type,
    base_url: account.base_url,
    username: account.username,
    password: account.password,
    token: account.token || '',
    auth_type: authType,
    token_failed_reason: account.token_failed_reason || '',
  }
  showEditAccountDialog.value = true
}

const handleUpdateAccount = async () => {
  try {
    const requestData = {
      id: editAccountForm.value.id,
      base_url: editAccountForm.value.base_url,
      auth_type: editAccountForm.value.auth_type,
      ...(editAccountForm.value.auth_type === 'token'
        ? { token: editAccountForm.value.token }
        : { username: editAccountForm.value.username, password: editAccountForm.value.password }),
    }

    const response = await http?.post(`${SERVER_URL}/account/openlist`, requestData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      showEditAccountDialog.value = false
      loadAccounts()
      ElMessage.success('账号更新成功')
    } else {
      console.error('更新账号失败:', response?.data.message || '未知错误')
      ElMessage.error(response?.data.message || '更新账号失败')
    }
  } catch (error) {
    console.error('更新账号错误:', error)
    ElMessage.error('更新账号失败')
  }
}

const handleAuthorize = (row: CloudAccount) => {
  if (row.source_type === '115') {
    handle115OAuth(row.id)
  } else if (row.source_type === '123') {
    selectedAccountId.value = row.id
    show123AuthDialog.value = true
  } else if (row.source_type === 'baidupan') {
    handleBaiduOAuth(row.id)
  }
}

const handle115OAuth = async (accountId?: number) => {
  try {
    const response = await http?.get(`${SERVER_URL}/115/oauth-url?account_id=${accountId}`)

    if (response?.data.code === 200 && response.data.data) {
      const oauthUrl = response.data.data
      currentAccountId.value = accountId
      window.open(oauthUrl, '_blank', 'width=600,height=700')
    } else {
      ElMessage.error(response?.data.message || '获取授权地址失败')
    }
  } catch (error) {
    console.error('115 OAuth授权错误:', error)
    ElMessage.error('获取授权地址失败')
  }
}

const handleBaiduOAuth = async (accountId?: number) => {
  try {
    const response = await http?.get(`${SERVER_URL}/baidupan/oauth-url?account_id=${accountId}`)

    if (response?.data.code === 200 && response.data.data) {
      const oauthUrl = response.data.data
      currentAccountId.value = accountId
      window.open(oauthUrl, '_blank', 'width=600,height=700')
    } else {
      ElMessage.error(response?.data.message || '获取授权地址失败')
    }
  } catch (error) {
    console.error('百度网盘 OAuth授权错误:', error)
    ElMessage.error('获取授权地址失败')
  }
}

const resetForm = () => {
  newAccountForm.value = {
    type: '',
    name: '',
    base_url: '',
    username: '',
    password: '',
    token: '',
    auth_type: 'password',
    app_id_name: 'Q115-STRM',
    app_id: '',
  }
}

const handleAddAccount = async () => {
  try {
    const data: Record<string, string | number> = {
      source_type: newAccountForm.value.type,
      name: newAccountForm.value.name,
    }
    let url = `${SERVER_URL}/account/add`
    if (newAccountForm.value.type === '115') {
      Object.assign(data, {
        base_url: newAccountForm.value.base_url,
        username: newAccountForm.value.username,
        password: newAccountForm.value.password,
        app_id_name: newAccountForm.value.app_id_name,
        app_id: newAccountForm.value.app_id,
      })
    } else if (newAccountForm.value.type === 'openlist') {
      url = `${SERVER_URL}/account/openlist`
      Object.assign(data, {
        base_url: newAccountForm.value.base_url,
        auth_type: newAccountForm.value.auth_type,
      })
      if (newAccountForm.value.auth_type === 'token') {
        data.token = newAccountForm.value.token
      } else {
        data.username = newAccountForm.value.username
        data.password = newAccountForm.value.password
      }
    }

    const response = await http?.post(url, data)

    if (response?.data.code === 200) {
      ElMessage.success('添加账号成功')
      showAddAccountDialog.value = false
      loadAccounts()
      resetForm()
    } else {
      ElMessage.error(`添加账号失败: ${response?.data.message || '未知错误'}`)
    }
  } catch (error) {
    console.error('添加账号失败:', error)
    const err: AxiosError = error as AxiosError
    const errData = err.response?.data as { message?: string }
    ElMessage.error(`添加账号失败: Http ${err.status}，${errData.message || err.message}`)
  }
}

const handleOAuthMessage = async (event: MessageEvent) => {
  if (event.data.type === 'oauth_success') {
    try {
      const requestData = {
        account_id: currentAccountId.value,
        data: event.data.data,
      }
      let url = "";
      if (event.data.source === '' || event.data.source === '115') {
        url = `${SERVER_URL}/115/oauth-confirm`
      } else if (event.data.source === 'baidupan') {
        url = `${SERVER_URL}/baidupan/oauth-confirm`
      }
      const response = await http?.post(url, requestData, {
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (response?.data.code === 200) {
        ElMessage.success('授权成功')
        loadAccounts()
        currentAccountId.value = undefined
      } else {
        ElMessage.error(response?.data.message || '授权确认失败')
      }
    } catch (error) {
      console.error('115 OAuth确认错误:', error)
      ElMessage.error('授权确认失败')
    }
  } else if (event.data.type === 'oauth_error') {
    const errorMsg = event.data.error || '授权失败，请重试'
    const errorCode = event.data.errno || 0
    ElMessage.error(`授权失败: ${errorMsg}${errorCode ? ` (错误代码: ${errorCode})` : ''}`)
    currentAccountId.value = undefined
  }
}

const selectedAccountId = ref<number | undefined>(undefined)
const show123AuthDialog = ref(false)

let removeDeviceTypeListener: (() => void) | null = null

onMounted(() => {
  loadAccounts()
  window.addEventListener('message', handleOAuthMessage)
  removeDeviceTypeListener = onDeviceTypeChange((newIsMobile) => {
    isMobile.value = newIsMobile
  })
})

onUnmounted(() => {
  window.removeEventListener('message', handleOAuthMessage)
  if (removeDeviceTypeListener) {
    removeDeviceTypeListener()
  }
})
</script>

<style scoped>
.cloud-accounts-page {
  min-height: 100%;
  background: #f5f7fa;
  padding: 0;
}

.page-header {
  background: #fff;
  padding: 24px;
  border-bottom: 1px solid #ebeef5;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 20px;
}

.header-title-section {
  flex: 1;
}

.page-title {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.title-icon {
  font-size: 28px;
  color: #409eff;
}

.page-subtitle {
  margin: 0;
  font-size: 14px;
  color: #909399;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.add-btn {
  background: #409eff !important;
  border-color: #409eff !important;
  transition: all 0.3s ease;
}

.add-btn:hover {
  background: #66b1ff !important;
  border-color: #66b1ff !important;
}

.stats-bar {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #f5f7fa;
  padding: 12px 16px;
  border-radius: 8px;
  min-width: 140px;
}

.stat-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
}

.stat-icon.total {
  background: #ecf5ff;
  color: #409eff;
}

.stat-icon.authorized {
  background: #f0f9eb;
  color: #67c23a;
}

.stat-icon.unauthorized {
  background: #fdf6ec;
  color: #e6a23c;
}

.stat-icon.failed {
  background: #fef0f0;
  color: #f56c6c;
}

.stat-info {
  display: flex;
  flex-direction: column;
}

.stat-value {
  font-size: 20px;
  font-weight: 600;
  line-height: 1.2;
  color: #303133;
}

.stat-label {
  font-size: 12px;
  color: #909399;
}

.accounts-content {
  padding: 24px;
}

.accounts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
}

.account-card {
  background: #fff;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  transition: all 0.3s ease;
  position: relative;
}

.account-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
}

.account-card.is-authorized {
  box-shadow: 0 2px 12px rgba(103, 194, 58, 0.2);
}

.account-card.is-authorized:hover {
  box-shadow: 0 8px 24px rgba(103, 194, 58, 0.3);
}

.account-card.is-unauthorized {
  box-shadow: 0 2px 12px rgba(230, 162, 60, 0.2);
}

.account-card.is-unauthorized:hover {
  box-shadow: 0 8px 24px rgba(230, 162, 60, 0.3);
}

.account-card.is-failed {
  box-shadow: 0 2px 12px rgba(245, 108, 108, 0.2);
}

.account-card.is-failed:hover {
  box-shadow: 0 8px 24px rgba(245, 108, 108, 0.3);
}

.card-status-bar {
  height: 4px;
  background: #e4e7ed;
}

.card-status-bar.status-authorized {
  background: linear-gradient(90deg, #67c23a, #95d475);
}

.card-status-bar.status-unauthorized {
  background: linear-gradient(90deg, #e6a23c, #f0c78a);
}

.card-status-bar.status-failed {
  background: linear-gradient(90deg, #f56c6c, #fab6b6);
}

.card-main {
  padding: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f2f5;
}

.card-title-wrapper {
  flex: 1;
  min-width: 0;
}

.card-id {
  display: inline-block;
  font-size: 12px;
  color: #909399;
  background: #f5f7fa;
  padding: 2px 8px;
  border-radius: 4px;
  margin-right: 8px;
}

.card-name {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  word-break: break-all;
}

.source-tag {
  flex-shrink: 0;
  margin-left: 8px;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.info-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
  flex-shrink: 0;
}

.info-content {
  flex: 1;
  display: flex;
  justify-content: space-between;
  align-items: center;
  min-width: 0;
}

.info-label {
  font-size: 13px;
  color: #909399;
}

.info-value {
  font-size: 14px;
  color: #303133;
  text-align: right;
}

.path-value {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 12px;
  word-break: break-all;
  max-width: 200px;
}

.status-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  padding-top: 12px;
  border-top: 1px dashed #ebeef5;
}

.status-divider {
  height: 1px;
  background: #ebeef5;
  margin: 8px 0;
}

.disk-status-section {
  background: #fafbfc;
  border-radius: 8px;
  padding: 12px;
  margin-top: 4px;
}

.status-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.status-title {
  font-size: 13px;
  font-weight: 600;
  color: #606266;
}

.no-status {
  text-align: center;
  color: #909399;
  font-size: 13px;
  padding: 8px 0;
}

.space-info {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  flex: 1;
}

.space-text {
  font-size: 12px;
  color: #606266;
}

.expire-warning {
  color: #f56c6c;
  font-weight: 500;
}

.status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
}

.status-indicator.status-authorized {
  background: #f0f9eb;
  color: #67c23a;
}

.status-indicator.status-unauthorized {
  background: #fdf6ec;
  color: #e6a23c;
}

.status-indicator.status-failed {
  background: #fef0f0;
  color: #f56c6c;
}

.error-help-icon {
  font-size: 16px;
  color: #f56c6c;
  cursor: help;
}

.rotating {
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.card-footer {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 16px;
  margin-top: 12px;
  border-top: 1px solid #f0f2f5;
}

.card-footer .el-button {
  flex: 1;
  min-width: 70px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  margin-bottom: 24px;
}

.empty-illustration {
  position: relative;
  margin-bottom: 24px;
}

.empty-icon {
  font-size: 80px;
  color: #dcdfe6;
}

.empty-dots {
  display: flex;
  justify-content: center;
  gap: 8px;
  margin-top: 16px;
}

.empty-dots span {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #dcdfe6;
  animation: bounce 1.4s infinite ease-in-out both;
}

.empty-dots span:nth-child(1) { animation-delay: -0.32s; }
.empty-dots span:nth-child(2) { animation-delay: -0.16s; }

@keyframes bounce {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}

.empty-title {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.empty-description {
  margin: 0 0 24px 0;
  font-size: 14px;
  color: #909399;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  background: #fff;
  border-radius: 16px;
  color: #909399;
  gap: 12px;
}

.loading-icon {
  font-size: 32px;
  color: #409eff;
}

.page-footer-tips {
  border: none;
  border-radius: 16px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.tips-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
  font-size: 15px;
  font-weight: 600;
}

.tips-icon {
  font-size: 18px;
}

.tips-content {
  display: flex;
  flex-wrap: wrap;
  gap: 0;
}

.tip-group {
  flex: 1;
  min-width: 300px;
  padding: 20px;
  border-right: 1px solid #f0f2f5;
}

.tip-group:last-child {
  border-right: none;
}

.tip-group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding-bottom: 10px;
  border-bottom: 2px solid #f0f2f5;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.tip-group-title .el-icon {
  color: #e6a23c;
  font-size: 18px;
}

.tip-group-items {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tip-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 13px;
  color: #606266;
  line-height: 1.6;
}

.tip-bullet {
  flex-shrink: 0;
  width: 16px;
  color: #c0c4cc;
  text-align: center;
}

.tip-item strong {
  color: #409eff;
}

.tip-highlight {
  background: linear-gradient(135deg, #fdf6ec 0%, #fef8eb 100%);
  margin: 6px -12px;
  padding: 12px;
  border-radius: 8px;
  border-left: 3px solid #e6a23c;
}

.tip-highlight .tip-bullet {
  color: #e6a23c;
}

.tip-highlight span:last-child {
  color: #8b6b3d;
}

.dialog-footer {
  display: flex;
  justify-content: center;
  gap: 12px;
}

@media (max-width: 768px) {
  .page-header {
    padding: 12px;
    background: #fff;
  }

  .header-title-section {
    display: none;
  }

  .header-content {
    margin-bottom: 0;
  }

  .header-actions {
    justify-content: stretch;
  }

  .header-actions .add-btn {
    width: 100%;
    background: #409eff !important;
    border-color: #409eff !important;
    color: #fff !important;
  }

  .header-actions .add-btn:hover {
    background: #66b1ff !important;
    border-color: #66b1ff !important;
    transform: none;
  }

  .mobile-hidden {
    display: none !important;
  }

  .accounts-content {
    padding: 12px;
  }

  .accounts-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .account-card {
    border-radius: 12px;
  }

  .card-main {
    padding: 12px;
  }

  .card-header {
    margin-bottom: 12px;
    padding-bottom: 10px;
  }

  .card-id {
    font-size: 11px;
    padding: 2px 6px;
  }

  .card-name {
    font-size: 14px;
  }

  .source-tag {
    font-size: 11px;
  }

  .card-body {
    gap: 10px;
  }

  .info-row {
    gap: 10px;
  }

  .info-icon {
    width: 28px;
    height: 28px;
    font-size: 14px;
  }

  .info-label {
    font-size: 12px;
  }

  .info-value {
    font-size: 13px;
  }

  .path-value {
    font-size: 11px;
    max-width: 140px;
  }

  .status-row {
    margin-top: 6px;
    padding-top: 10px;
  }

  .status-indicator {
    padding: 5px 10px;
    font-size: 12px;
  }

  .disk-status-section {
    padding: 10px;
  }

  .status-header {
    margin-bottom: 10px;
  }

  .status-title {
    font-size: 12px;
  }

  .space-info {
    align-items: flex-start;
  }

  .space-text {
    font-size: 11px;
  }

  .card-footer {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
    padding-top: 12px;
    margin-top: 10px;
  }

  .card-footer .el-button {
    flex: none;
    min-width: 0;
    width: 100%;
    margin: 0;
  }

  .card-footer .el-button :deep(.el-icon) {
    margin-right: 4px;
  }

  .empty-state {
    padding: 40px 16px;
    border-radius: 12px;
  }

  .empty-icon {
    font-size: 60px;
  }

  .empty-title {
    font-size: 16px;
  }

  .empty-description {
    font-size: 13px;
    margin-bottom: 20px;
  }

  .page-footer-tips {
    border-radius: 12px;
  }

  .tips-header {
    padding: 12px 14px;
    font-size: 14px;
  }

  .tip-group {
    padding: 14px;
    border-right: none;
    border-bottom: 1px solid #f0f2f5;
  }

  .tip-group:last-child {
    border-bottom: none;
  }

  .tip-group-title {
    font-size: 14px;
    margin-bottom: 12px;
    padding-bottom: 8px;
  }

  .tip-group-items {
    gap: 8px;
  }

  .tip-item {
    font-size: 12px;
  }

  .tip-highlight {
    margin: 4px -8px;
    padding: 10px;
  }
}

@media (max-width: 480px) {
  .info-content {
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
  }

  .info-value {
    text-align: left;
  }

  .path-value {
    max-width: 100%;
  }

  .card-footer {
    grid-template-columns: 1fr;
    gap: 6px;
  }
}
</style>
