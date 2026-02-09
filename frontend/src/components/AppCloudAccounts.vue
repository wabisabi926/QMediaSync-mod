<template>
  <div class="main-content-container accounts-content">
    <!-- 操作提示 -->
    <el-alert title="操作提示" type="info" description="先添加账号，然后点击列表中操作区域的授权按钮完成账号绑定" :closable="false" show-icon
      class="operation-tip" />

    <!-- 添加账号按钮 -->
    <div class="add-account-button">
      <el-button type="primary" @click="showAddAccountDialog = true">添加账号</el-button>
    </div>

    <!-- 账号卡片列表 -->
    <div v-loading="loading" element-loading-text="加载中..." class="accounts-loading-container">
      <div style="
          width: 100%;
          height: 100%;
          display: flex;
          flex-wrap: wrap;
          gap: 6px;
          justify-content: start;
          align-items: top;
        ">
        <el-card class="account-card" shadow="hover" v-for="account in accounts" :key="account.id">
          <template #header>
            <div class="account-card-header">
              <div class="account-info">
                <h3 class="account-name">#{{ account.id }} {{ account.name }}</h3>
                <el-tag :type="sourceTypeTagMap[account.source_type]" effect="dark">
                  {{ sourceTypeMap[account.source_type] }}
                </el-tag>
              </div>
              <div>
                <el-tag v-if="account.token" type="success" size="large">已授权</el-tag>
                <template v-if="account.token_failed_reason && !account.token">
                  <el-tooltip :content="account.token_failed_reason" placement="top">
                    <el-tag type="danger" size="large">
                      <el-icon>
                        <WarningFilled />
                      </el-icon>
                      凭证刷新失败
                    </el-tag>
                  </el-tooltip>
                </template>
                <el-tag v-if="!account.token_failed_reason && !account.token" type="danger" size="large">未授权</el-tag>
              </div>
            </div>
          </template>
          <div class="card-body">
            <div class="info-item" v-if="account.source_type === '115'">
              <span class="info-label">115账号:</span>
              <span class="info-value">{{ account.username }}</span>
            </div>
            <div class="info-item" v-if="account.source_type === '115'">
              <span class="info-label">115开放平台应用:</span>
              <span class="info-value">{{ account.app_id_name }}</span>
            </div>
            <div class="info-item" v-if="account.source_type === 'baidupan'">
              <span class="info-label">APP ID:</span>
              <span class="info-value">{{ account.app_id || '-' }}</span>
            </div>
            <template v-if="account.source_type === 'openlist'">
              <div class="info-item">
                <span class="info-label">OpenList地址:</span>
                <span class="info-value">{{ account.base_url }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">用户名:</span>
                <span class="info-value">{{ account.name }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">用户ID:</span>
                <span class="info-value">{{ account.user_id }}</span>
              </div>
            </template>
            <div class="info-item">
              <span class="info-label">添加时间:</span>
              <span class="info-value">{{ formatTimestamp(account.created_at) }}</span>
            </div>
          </div>
          <template #footer>
            <div class="account-card-footer">
              <el-button type="danger" @click="handleDelete(account)"> 删除 </el-button>
              <el-button type="warning" @click="handleAuthorize(account)" size="small"
                v-if="account.source_type !== 'openlist'">
                授权
              </el-button>
              <el-button type="primary" @click="handleEdit(account)" size="small"
                v-if="account.source_type === 'openlist'">
                编辑
              </el-button>
            </div>
          </template>
        </el-card>
      </div>
    </div>
  </div>
  <!-- 添加账号对话框 -->
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

  <!-- 编辑账号对话框 -->
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
import { inject, ref, onMounted, onUnmounted } from 'vue'

import {
  WarningFilled,
} from '@element-plus/icons-vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { formatTimestamp } from '@/utils/timeUtils'
import { sourceTypeMap, sourceTypeOptions, sourceTypeTagMap } from '@/utils/sourceTypeUtils'
import { isMobile as checkIsMobile } from '@/utils/deviceUtils'

const isMobile = ref(checkIsMobile())

// 定义页面显示的账号数据结构
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
}

// 获取HTTP客户端
const http: AxiosStatic | undefined = inject('$http')

// 账号列表数据
const accounts = ref<CloudAccount[]>([])

// 加载状态
const loading = ref(false)

// 添加账号对话框显示状态
const showAddAccountDialog = ref(false)

// 添加账号loading状态
const addAccountLoading = ref(false)

// 新账号表单数据
const newAccountForm = ref({
  type: '',
  name: '',
  base_url: '',
  username: '',
  password: '',
  token: '',
  auth_type: 'password',
  app_id_name: 'Q115-STRM', // 默认值
  app_id: '',
})

// 编辑账号相关状态
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

const currentAccountId = ref<number | undefined>(undefined) // 当前账号ID

// 123云盘授权相关状态
const show123AuthDialog = ref(false)
const selectedAccountId = ref<number | undefined>(undefined)

// 加载账号列表
const loadAccounts = async () => {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/account/list`)

    if (response?.data.code === 200) {
      const data = response.data.data
      console.log(data)
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
      }))
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

// 删除账号
const handleDelete = async (row: CloudAccount) => {
  try {
    await ElMessageBox.confirm(`确定要删除账号 "${row.name}" 吗？此操作不可恢复。`, '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    // 调用API删除账号
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
      loadAccounts() // 刷新账号列表
    } else {
      ElMessage.error(response?.data.message || '删除账号失败')
    }
  } catch (error) {
    // 用户取消删除或请求失败
    if (error !== 'cancel' && error !== 'close') {
      console.error('删除账号失败:', error)
      ElMessage.error('删除账号失败')
    }
  }
}

// 处理编辑账号
const handleEdit = (account: CloudAccount) => {
  currentEditAccount.value = account
  console.log(account.base_url, account.password)

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
  console.log(editAccountForm.value)
  showEditAccountDialog.value = true
}

// 处理更新账号
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

// 授权账号
const handleAuthorize = (row: CloudAccount) => {
  console.log('授权账号:', row)
  // 实现授权逻辑
  // 如果是115网盘，使用OAuth授权
  if (row.source_type === '115') {
    handle115OAuth(row.id)
  }
  // 如果是123云盘，显示确认对话框
  else if (row.source_type === '123') {
    selectedAccountId.value = row.id
    show123AuthDialog.value = true
  }
  // 如果是百度网盘，显示确认对话框
  else if (row.source_type === 'baidupan') {
    handleBaiduOAuth(row.id)
  }
}

// 处理115 OAuth授权
const handle115OAuth = async (accountId?: number) => {
  try {

    const response = await http?.get(`${SERVER_URL}/115/oauth-url?account_id=${accountId}`)

    if (response?.data.code === 200 && response.data.data) {
      const oauthUrl = response.data.data
      // 保存当前授权的账号ID
      currentAccountId.value = accountId
      // 在新窗口打开OAuth授权页面
      window.open(oauthUrl, '_blank', 'width=600,height=700')
    } else {
      ElMessage.error(response?.data.message || '获取授权地址失败')
    }
  } catch (error) {
    console.error('115 OAuth授权错误:', error)
    ElMessage.error('获取授权地址失败')
  }
}

// 处理百度网盘OAuth授权
const handleBaiduOAuth = async (accountId?: number) => {
  try {
    const response = await http?.get(`${SERVER_URL}/baidupan/oauth-url?account_id=${accountId}`)

    if (response?.data.code === 200 && response.data.data) {
      const oauthUrl = response.data.data
      // 保存当前授权的账号ID
      currentAccountId.value = accountId
      // 在新窗口打开OAuth授权页面
      window.open(oauthUrl, '_blank', 'width=600,height=700')
    } else {
      ElMessage.error(response?.data.message || '获取授权地址失败')
    }
  } catch (error) {
    console.error('百度网盘 OAuth授权错误:', error)
    ElMessage.error('获取授权地址失败')
  }
}

// 处理添加账号
// 重置表单
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

// 添加账号
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

// 处理115 OAuth授权完成后的消息
const handleOAuthMessage = async (event: MessageEvent) => {
  if (event.data.type === 'oauth_success') {
    // 115 OAuth授权成功，发送数据到服务端处理
    console.log('OAuth授权成功，准备确认授权:', event.data)

    try {
      const requestData = {
        account_id: currentAccountId.value,
        data: event.data.data,
      }
      let url = "";
      if (event.data.source === '' || event.data.source === '115') {
        url = `${SERVER_URL}/115/oauth-confirm`
        // requestData.data = JSON.stringify(requestData.data)
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
        // 刷新账号列表
        loadAccounts()
        // 清空当前账号ID
        currentAccountId.value = undefined
      } else {
        ElMessage.error(response?.data.message || '授权确认失败')
      }
    } catch (error) {
      console.error('115 OAuth确认错误:', error)
      ElMessage.error('授权确认失败')
    }
  } else if (event.data.type === 'oauth_error') {
    // OAuth授权失败
    console.error('OAuth授权失败:', event.data)

    // 显示错误提示
    const errorMsg = event.data.error || '授权失败，请重试'
    const errorCode = event.data.errno || 0
    ElMessage.error(`授权失败: ${errorMsg}${errorCode ? ` (错误代码: ${errorCode})` : ''}`)

    // 清空当前账号ID
    currentAccountId.value = undefined
  }
}

// 组件挂载时加载数据并添加事件监听器
onMounted(() => {
  loadAccounts()
  // 添加事件监听器以处理115 OAuth授权完成后的消息
  window.addEventListener('message', handleOAuthMessage)
})

// 组件卸载时移除事件监听器
onUnmounted(() => {
  window.removeEventListener('message', handleOAuthMessage)
})
</script>

<style scoped>
.cloud-accounts-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.cloud-accounts-card {
  width: 100%;
  max-width: none;
  margin: 0;
}

.card-title {
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.add-account-button {
  margin-bottom: 20px;
}

.card-subtitle {
  margin: 0;
  font-size: 14px;
  color: #909399;
}

.accounts-content {
  margin-top: 16px;
}

.operation-tip {
  margin-bottom: 20px;
}

/* 二维码登录对话框样式 */
.qr-login-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0;
}

.qr-code-section {
  margin-bottom: 20px;
}

.qr-code-wrapper {
  display: flex;
  justify-content: center;
  padding: 20px;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
}

.qr-code-image {
  width: 200px;
  height: 200px;
  border-radius: 4px;
}

.qr-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 20px;
  color: #909399;
}

.qr-loading .el-icon {
  font-size: 32px;
  margin-bottom: 16px;
}

.qr-status-section {
  text-align: center;
  min-height: 60px;
}

.status-waiting,
.status-scanned,
.status-confirmed,
.status-expired,
.status-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.status-waiting .el-icon {
  font-size: 24px;
  color: #909399;
}

.status-scanned .el-icon {
  font-size: 24px;
  color: #67c23a;
}

.status-confirmed .el-icon {
  font-size: 24px;
  color: #67c23a;
}

.status-expired .el-icon {
  font-size: 24px;
  color: #e6a23c;
}

.status-error .el-icon {
  font-size: 24px;
  color: #f56c6c;
}

.status-waiting p,
.status-scanned p,
.status-confirmed p,
.status-expired p,
.status-error p {
  margin: 0;
  font-size: 14px;
  color: #606266;
}

.dialog-footer {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.action-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  justify-content: flex-start;
}

.action-buttons .el-button {
  margin: 0;
}

/* 123云盘授权对话框样式 */
.auth-123-container {
  padding: 20px 0;
  text-align: center;
}

.auth-123-container p {
  margin: 10px 0;
  font-size: 14px;
  color: #606266;
  line-height: 1.5;
}

/* 账号卡片列表样式 */
.accounts-loading-container {
  min-height: 300px;
  padding: 10px 0;
}

.account-card {
  /* width: 100%; */
  display: flex;
  flex-direction: column;
  transition: all 0.3s ease;
  min-width: 360px;
}

.account-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 10px 20px rgba(0, 0, 0, 0.1);
}

.account-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: nowrap;
}

.account-name {
  font-size: 16px;
  font-weight: bold;
  margin-bottom: 10px;
  color: #303133;
}

.account-type {
  font-size: 12px;
  color: #606266;
  background-color: #f0f2f5;
  padding: 2px 8px;
  border-radius: 12px;
  display: inline-block;
  width: fit-content;
  margin-top: 4px;
}

.account-id {
  font-size: 12px;
  color: #909399;
  background-color: #f5f7fa;
  padding: 2px 8px;
  border-radius: 12px;
}

.account-card-body {
  flex: 1;
  margin-bottom: 16px;
}

.account-card-body p {
  margin: 8px 0;
  font-size: 14px;
  color: #606266;
}

.account-card-body strong {
  color: #303133;
}

.account-card-footer {
  display: flex;
  justify-content: end;
  align-items: center;
}
</style>
