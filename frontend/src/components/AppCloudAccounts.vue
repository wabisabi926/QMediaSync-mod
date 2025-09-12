<template>
  <div class="main-content-container accounts-content">
    <!-- 操作提示 -->
    <el-alert
      title="操作提示"
      type="info"
      description="先添加账号，然后点击列表中操作区域的授权按钮完成账号绑定"
      :closable="false"
      show-icon
      class="operation-tip"
    />

    <!-- 添加账号按钮 -->
    <div class="add-account-button">
      <el-button type="primary" @click="showAddAccountDialog = true">添加账号</el-button>
    </div>

    <!-- 账号卡片列表 -->
    <div v-loading="loading" element-loading-text="加载中..." class="accounts-loading-container">
      <el-row :gutter="20">
        <el-col
          style="margin-bottom: 10px"
          :xs="24"
          :sm="12"
          :md="6"
          :lg="4"
          v-for="account in accounts"
          :key="account.id"
        >
          <el-card class="account-card" shadow="hover">
            <template #header>
              <div class="account-card-header">
                <div class="account-info">
                  <h3 class="account-name">#{{ account.id }} {{ account.name }}</h3>
                  <el-tag :type="sourceTypeTagMap[account.type]" effect="dark">
                    {{ sourceTypeMap[account.type] }}
                  </el-tag>
                </div>
                <div>
                  <el-tag v-if="account.token" type="success" size="large">已授权</el-tag>
                  <el-tag v-else type="danger" size="large">未授权</el-tag>
                </div>
              </div>
            </template>
            <div class="account-card-body">
              <el-row justify="space-between" v-if="account.token">
                <el-col :span="12"> <strong>用户ID:</strong> {{ account.userId }} </el-col>
                <el-col :span="12"> <strong>用户名:</strong> {{ account.username }} </el-col>
              </el-row>
              <el-row>
                <el-col :span="24"> <strong>添加时间:</strong> {{ account.addTime }} </el-col>
              </el-row>
            </div>
            <template #footer>
              <div class="account-card-footer">
                <el-button type="danger" @click="handleDelete(account)"> 删除 </el-button>
                <el-button v-if="!account.token" type="warning" @click="handleAuthorize(account)">
                  授权
                </el-button>
              </div>
            </template>
          </el-card>
        </el-col>
      </el-row>
    </div>
    <!-- 添加账号对话框 -->
    <el-dialog v-model="showAddAccountDialog" title="添加账号" :width="isMobile ? '90%' : '500px'">
      <el-form :model="newAccountForm" label-width="80px">
        <el-form-item label="网盘类型">
          <el-select v-model="newAccountForm.type" placeholder="请选择网盘类型">
            <el-option
              v-for="typeItem in sourceTypeOptions"
              :key="typeItem.value"
              :label="typeItem.label"
              :value="typeItem.value"
            ></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="账号备注">
          <el-input v-model="newAccountForm.name" placeholder="请输入账号备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showAddAccountDialog = false">取消</el-button>
          <el-button type="primary" @click="handleAddAccount">确定</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 115网盘二维码授权对话框 -->
    <el-dialog
      v-model="showQRDialog"
      title="115开放平台扫码授权"
      width="400px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      @close="closeQRDialog"
    >
      <div class="qr-login-container">
        <div class="qr-code-section">
          <div v-if="qrCodeUrl" class="qr-code-wrapper">
            <img :src="qrCodeUrl" alt="登录二维码" class="qr-code-image" />
          </div>
          <div v-else class="qr-loading">
            <el-icon class="is-loading"><Loading /></el-icon>
            <p>正在生成二维码...</p>
          </div>
        </div>

        <div class="qr-status-section">
          <div v-if="qrStatus === 'waiting'" class="status-waiting">
            <el-icon><Iphone /></el-icon>
            <p>请使用115手机客户端扫描二维码</p>
          </div>
          <div v-else-if="qrStatus === 'scanned'" class="status-scanned">
            <el-icon><SuccessFilled /></el-icon>
            <p>扫描成功，请在手机上确认授权</p>
          </div>
          <div v-else-if="qrStatus === 'confirmed'" class="status-confirmed">
            <el-icon><CircleCheckFilled /></el-icon>
            <p>授权确认成功，正在获取账号信息...</p>
          </div>
          <div v-else-if="qrStatus === 'expired'" class="status-expired">
            <el-icon><WarningFilled /></el-icon>
            <p>二维码已过期，请重新获取</p>
          </div>
          <div v-else-if="qrStatus === 'error'" class="status-error">
            <el-icon><CircleCloseFilled /></el-icon>
            <p>授权过程中出现错误，请重试</p>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="closeQRDialog">取消</el-button>
          <el-button
            v-if="qrStatus === 'expired' || qrStatus === 'error'"
            type="primary"
            @click="refreshQRCode(currentAccountId)"
          >
            重新获取二维码
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 123云盘授权确认对话框 -->
    <el-dialog
      v-model="show123AuthDialog"
      title="123云盘授权确认"
      width="400px"
      :close-on-click-modal="true"
      :close-on-press-escape="true"
    >
      <div class="auth-123-container">
        <el-button type="primary" :icon="Link">打开123云盘授权页面</el-button>
        <p>授权成功后会返回本页面</p>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="show123AuthDialog = false">取消</el-button>
          <el-button type="primary" @click="proceed123Auth">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject, ref, onMounted, onUnmounted } from 'vue'
import QRCode from 'qrcode'
import {
  Loading,
  Iphone,
  SuccessFilled,
  CircleCheckFilled,
  WarningFilled,
  CircleCloseFilled,
  Link,
} from '@element-plus/icons-vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { formatTimestamp } from '@/utils/timeUtils'
import { sourceTypeMap, sourceTypeOptions, sourceTypeTagMap } from '@/utils/sourceTypeUtils'
import { isMobile as checkIsMobile } from '@/utils/deviceUtils'

const isMobile = ref(checkIsMobile())

// 定义API返回的账号数据结构
interface ApiCloudAccount {
  id: number
  source_type: string
  name: string
  user_id: string
  username: string
  created_at: number
  token: string
}

// 定义二维码数据结构
interface QRCodeData {
  qrcode: string
  [key: string]: unknown // 允许其他未知字段
}

// 定义页面显示的账号数据结构
interface CloudAccount {
  id: number
  type: string
  name: string
  userId: string
  username: string
  addTime: string
  token: string
}

// 获取HTTP客户端
const http: AxiosStatic | undefined = inject('$http')

// 账号列表数据
const accounts = ref<CloudAccount[]>([])

// 加载状态
const loading = ref(false)

// 添加账号对话框显示状态
const showAddAccountDialog = ref(false)

// 新账号表单数据
const newAccountForm = ref({
  type: '',
  name: '',
})

// 二维码登录相关状态
const showQRDialog = ref(false)
const qrCodeUrl = ref('')
const qrCodeContent = ref('') // 保存二维码内容
const qrCodeData = ref<QRCodeData | null>(null) // 保存完整的扫码接口结果
const currentAccountId = ref<number | undefined>(undefined) // 当前账号ID
// 轮询定时器
const pollingTimer = ref<NodeJS.Timeout | null>(null)
const qrStatus = ref<'waiting' | 'scanned' | 'confirmed' | 'expired' | 'error'>('waiting')

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
      accounts.value = data.map((item: ApiCloudAccount) => ({
        id: item.id,
        type: item.source_type,
        name: item.name,
        userId: item.user_id,
        username: item.username,
        addTime: formatTimestamp(item.created_at),
        token: item.token,
      }))
    } else {
      console.error('加载账号列表失败:', response?.data.msg || '未知错误')
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
      ElMessage.error(response?.data.msg || '删除账号失败')
    }
  } catch (error) {
    // 用户取消删除或请求失败
    if (error !== 'cancel' && error !== 'close') {
      console.error('删除账号失败:', error)
      ElMessage.error('删除账号失败')
    }
  }
}

// 授权账号
const handleAuthorize = (row: CloudAccount) => {
  console.log('授权账号:', row)
  // 实现授权逻辑
  // 如果是115网盘，显示二维码对话框
  if (row.type === '115') {
    handle115Login(row.id)
  }
  // 如果是123云盘，显示确认对话框
  else if (row.type === '123') {
    selectedAccountId.value = row.id
    show123AuthDialog.value = true
  }
}

// 处理123云盘授权确认
const proceed123Auth = () => {
  show123AuthDialog.value = false
  // 打开新页面进行123云盘授权
  // 由于是演示，我们使用本地的123auth.html页面
  const authUrl = '/123auth.html'
  window.open(authUrl, '_blank')
}

// 处理添加账号
const handleAddAccount = async () => {
  try {
    // 准备请求数据
    const requestData = {
      source_type: newAccountForm.value.type,
      name: newAccountForm.value.name,
    }

    // 调用API添加账号
    const response = await http?.post(`${SERVER_URL}/account/add`, requestData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      // 添加成功，关闭对话框，重置表单，刷新账号列表
      showAddAccountDialog.value = false
      newAccountForm.value.type = ''
      newAccountForm.value.name = ''
      loadAccounts() // 刷新账号列表
      console.log('账号添加成功')
    } else {
      console.error('添加账号失败:', response?.data.msg || '未知错误')
    }
  } catch (error) {
    console.error('添加账号错误:', error)
  }
}

// 生成二维码
const generateQRCode = async (content: string): Promise<string> => {
  try {
    // 使用本地 qrcode 库生成二维码
    const qrDataURL = await QRCode.toDataURL(content, {
      width: 200,
      margin: 2,
      color: {
        dark: '#000000',
        light: '#FFFFFF',
      },
    })
    return qrDataURL
  } catch (error) {
    console.error('生成二维码失败:', error)
    // 如果本地生成失败，回退到在线服务
    const encodedContent = encodeURIComponent(content)
    return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodedContent}`
  }
}

// 处理115开放平台授权
const handle115Login = async (accountId?: number) => {
  try {
    // 获取二维码
    const requestData = {
      account_id: accountId, // 添加account_id参数
    }

    const response = await http?.post(`${SERVER_URL}/auth/115-qrcode-open`, requestData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200 && response.data.data) {
      qrCodeData.value = response.data.data // 保存完整的扫码结果
      qrCodeContent.value = response.data.data.qrcode // 保存二维码内容
      qrCodeUrl.value = await generateQRCode(response.data.data.qrcode) // 生成二维码图片
      showAddAccountDialog.value = false // 关闭添加账号对话框
      showQRDialog.value = true // 显示二维码对话框
      qrStatus.value = 'waiting'
      currentAccountId.value = accountId || undefined // 设置当前账号ID

      // 开始轮询二维码状态，传递account_id
      startPolling(accountId)
    } else {
      console.error('获取二维码失败:', response?.data.msg || '未知错误')
    }
  } catch (error) {
    console.error('115登录错误:', error)
  }
}

// 开始轮询二维码状态
const startPolling = (accountId?: number) => {
  if (pollingTimer.value) {
    clearInterval(pollingTimer.value)
  }

  pollingTimer.value = setInterval(async () => {
    await checkQRStatus(accountId)
  }, 500) // 每500毫秒检查一次
}

// 停止轮询
const stopPolling = () => {
  if (pollingTimer.value) {
    clearInterval(pollingTimer.value)
    pollingTimer.value = null
  }
}

// 检查二维码状态
const checkQRStatus = async (accountId?: number) => {
  if (!qrCodeData.value) return

  try {
    // 将扫码数据转换为JSON格式
    const requestData = {
      ...qrCodeData.value,
      account_id: accountId, // 添加account_id参数
    }

    const response = await http?.post(
      `${SERVER_URL}/auth/115-qrcode-status`,
      requestData, // 传递JSON格式的数据
      {
        headers: {
          'Content-Type': 'application/json',
        },
      },
    )

    if (response?.data.code === 200 && response.data.data) {
      const status = response.data.data.status

      switch (status) {
        case 2: // 未扫码
          qrStatus.value = 'waiting'
          break
        case 3: // 已扫码
          qrStatus.value = 'scanned'
          break
        case 4: // 已确认（扫码成功）
          qrStatus.value = 'confirmed'
          stopPolling()
          // 延迟1秒后关闭对话框并刷新账号列表
          setTimeout(() => {
            closeQRDialog()
            loadAccounts() // 刷新账号列表
          }, 1000)
          break
        case -5: // 已过期
          qrStatus.value = 'expired'
          stopPolling()
          break
        default:
          // 其他未知状态
          qrStatus.value = 'error'
          break
      }
    }
  } catch (error) {
    console.error('检查二维码状态错误:', error)
    qrStatus.value = 'error'
  }
}

// 关闭二维码对话框
const closeQRDialog = () => {
  showQRDialog.value = false
  stopPolling()
  qrCodeUrl.value = ''
  qrCodeContent.value = ''
  qrCodeData.value = null
  qrStatus.value = 'waiting'
  currentAccountId.value = undefined // 重置当前账号ID
}

// 刷新二维码
const refreshQRCode = async (accountId?: number) => {
  qrStatus.value = 'waiting'
  qrCodeUrl.value = ''

  try {
    const formData = new URLSearchParams()
    formData.append('device_type', 'web')
    if (accountId) {
      formData.append('account_id', accountId.toString())
    }

    const response = await http?.post(`${SERVER_URL}/auth/115-qrcode-open`, formData, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
    })

    if (response?.data.code === 200 && response.data.data) {
      qrCodeData.value = response.data.data // 保存完整的扫码结果
      qrCodeContent.value = response.data.data.qrcode // 保存二维码内容
      qrCodeUrl.value = await generateQRCode(response.data.data.qrcode) // 生成二维码图片
      startPolling(accountId)
    } else {
      qrStatus.value = 'error'
    }
  } catch (error) {
    console.error('刷新二维码错误:', error)
    qrStatus.value = 'error'
  }
}

// 处理123云盘授权完成后的消息
const handle123AuthMessage = (event: MessageEvent) => {
  if (event.data.type === '123_auth_success') {
    // 123云盘授权成功，刷新账号列表
    loadAccounts()
  } else if (event.data.type === '123_auth_cancel') {
    // 123云盘授权取消，不需要特殊处理
    console.log('123云盘授权已取消')
  }
}

// 组件挂载时加载数据并添加事件监听器
onMounted(() => {
  loadAccounts()
  // 添加事件监听器以处理123云盘授权完成后的消息
  window.addEventListener('message', handle123AuthMessage)
})

// 组件卸载时移除事件监听器
onUnmounted(() => {
  window.removeEventListener('message', handle123AuthMessage)
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
  width: 100%;
  display: flex;
  flex-direction: column;
  transition: all 0.3s ease;
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

.account-info {
  display: flex;
  flex-direction: column;
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
