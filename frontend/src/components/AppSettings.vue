<template>
  <div class="core-settings-container">
    <!-- 115开放平台授权卡片 -->
    <el-card class="core-settings-card" shadow="hover">
      <template #header>
        <h2 class="card-title">115开放平台授权</h2>
        <p class="card-subtitle">115开放平台授权和账号管理</p>
      </template>

      <div class="core-content">
        <!-- 115开放平台授权部分 -->
        <div v-if="!accountInfo" class="login-section">
          <h3 class="section-title">
            <el-icon><User /></el-icon>
            115开放平台授权
          </h3>
          <p class="section-description">
            扫码授权通过使用115开放平台提供的文件筛选、上传、下载功能
          </p>
          <div class="login-actions">
            <el-button type="primary" size="large" @click="handle115Login" :loading="loginLoading">
              <el-icon><Key /></el-icon>
              点击授权
            </el-button>

            <el-button
              type="primary"
              plain
              size="large"
              @click="checkLoginStatus"
              :loading="checkingStatus"
            >
              <el-icon><Search /></el-icon>
              检查授权状态
            </el-button>
          </div>

          <!-- 登录状态显示 -->
          <el-alert
            v-if="loginStatus"
            :title="loginStatus.title"
            :type="loginStatus.type"
            :description="loginStatus.description"
            :closable="false"
            show-icon
            class="login-status"
          />
        </div>

        <!-- 115平台账号详细信息 -->
        <div v-if="accountInfo" class="account-info">
          <h4 class="info-title">账号信息</h4>
          <div class="info-grid">
            <div class="info-item">
              <span class="info-label">用户ID:</span>
              <span class="info-value">{{ accountInfo.user_id }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">用户名:</span>
              <span class="info-value">{{ accountInfo.username }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">存储空间:</span>
              <span class="info-value"
                >{{ formatStorage(accountInfo.used_space) }} /
                {{ formatStorage(accountInfo.total_space) }}</span
              >
            </div>
            <div class="info-item">
              <span class="info-label">使用率:</span>
              <span class="info-value"
                >{{ getStoragePercent(accountInfo.used_space, accountInfo.total_space) }}%</span
              >
            </div>
            <div class="info-item">
              <span class="info-label">会员等级:</span>
              <span class="info-value" :class="getMemberClass(accountInfo.member_level)">{{
                accountInfo.member_level
              }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">到期时间:</span>
              <span class="info-value" :class="getExpireClass(accountInfo.expire_time)">{{
                formatExpireTime(accountInfo.expire_time)
              }}</span>
            </div>
          </div>

          <!-- 存储空间进度条 -->
          <div class="storage-progress">
            <el-progress
              :percentage="getStoragePercent(accountInfo.used_space, accountInfo.total_space)"
              :color="getProgressColor(accountInfo.used_space, accountInfo.total_space)"
              :show-text="false"
            />
          </div>
        </div>

        <el-divider />
      </div>
    </el-card>
  </div>

  <!-- 二维码登录对话框 -->
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
          @click="refreshQRCode"
        >
          重新获取二维码
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import QRCode from 'qrcode'
import {
  User,
  Key,
  Search,
  Loading,
  Iphone,
  SuccessFilled,
  CircleCheckFilled,
  WarningFilled,
  CircleCloseFilled,
} from '@element-plus/icons-vue'
import { inject, onMounted, onUnmounted, reactive, ref } from 'vue'
import { formatStorage, formatExpireTime } from '@/utils/timeUtils'
import { isMobile as checkIsMobile, onDeviceTypeChange } from '@/utils/deviceUtils'

interface LoginStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}

interface AccountInfo {
  user_id: string
  username: string
  used_space: number
  total_space: number
  member_level: string
  expire_time: string
}

interface LoginData {
  device_type: string
}

interface QRCodeData {
  qrcode: string
  [key: string]: unknown // 允许其他未知字段
}

const http: AxiosStatic | undefined = inject('$http')
const isMobile = ref(false)
const loginLoading = ref(false)
const checkingStatus = ref(false)
const loginStatus = ref<LoginStatus | null>(null)
const accountInfo = ref<AccountInfo | null>(null)

// 二维码登录相关状态
const showQRDialog = ref(false)
const qrCodeUrl = ref('')
const qrCodeContent = ref('') // 保存二维码内容
const qrCodeData = ref<QRCodeData | null>(null) // 保存完整的扫码接口结果
// 轮询定时器
const pollingTimer = ref<NodeJS.Timeout | null>(null)
const qrStatus = ref<'waiting' | 'scanned' | 'confirmed' | 'expired' | 'error'>('waiting')

// 登录相关数据
const loginData = reactive<LoginData>({
  device_type: 'web', // 默认使用网页设备类型
})

// 检测是否为移动设备
const checkMobile = () => {
  isMobile.value = checkIsMobile()
}

// 计算存储使用百分比
const getStoragePercent = (used: number, total: number): number => {
  if (!total || total === 0) return 0
  return Math.round((used / total) * 100)
}

// 获取进度条颜色
const getProgressColor = (used: number, total: number): string => {
  const percent = getStoragePercent(used, total)
  if (percent >= 90) return '#f56c6c'
  if (percent >= 70) return '#e6a23c'
  return '#67c23a'
}

// 获取会员等级样式类
const getMemberClass = (level: string): string => {
  const lowerLevel = level.toLowerCase()
  if (lowerLevel.includes('vip') || lowerLevel.includes('会员')) {
    return 'member-vip'
  }
  return 'member-normal'
}

// 获取到期时间样式类
const getExpireClass = (expireTime: string): string => {
  if (!expireTime) return 'expire-unknown'

  const date = new Date(expireTime)
  if (isNaN(date.getTime())) return 'expire-unknown'

  const now = new Date()
  const diffTime = date.getTime() - now.getTime()
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24))

  if (diffDays < 0) return 'expire-expired'
  if (diffDays <= 7) return 'expire-warning'
  if (diffDays <= 30) return 'expire-caution'
  return 'expire-normal'
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
} // 处理115开放平台授权
const handle115Login = async () => {
  try {
    loginLoading.value = true
    loginStatus.value = null
    accountInfo.value = null

    // 获取二维码
    const requestData = {
      device_type: loginData.device_type,
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
      showQRDialog.value = true
      qrStatus.value = 'waiting'

      // 开始轮询二维码状态
      startPolling()
    } else {
      loginStatus.value = {
        title: '获取二维码失败',
        type: 'error',
        description: response?.data.msg || '无法获取登录二维码，请稍后重试',
      }
    }
  } catch (error) {
    console.error('115登录错误:', error)
    loginStatus.value = {
      title: '登录出错',
      type: 'error',
      description: '登录过程中发生错误，请检查网络连接',
    }
  } finally {
    loginLoading.value = false
  }
}

// 检查115登录状态
const checkLoginStatus = async () => {
  try {
    checkingStatus.value = true
    loginStatus.value = null
    accountInfo.value = null

    const response = await http?.get(`${SERVER_URL}/auth/115-status`)

    if (response?.data.code === 200) {
      const isLoggedIn = response.data.data?.logged_in || false

      loginStatus.value = {
        title: isLoggedIn ? '115平台已授权' : '115平台未授权',
        type: isLoggedIn ? 'success' : 'warning',
        description: isLoggedIn ? `已成功授权115开放平台` : '请点击授权按钮完成115平台授权',
      }

      // 如果已登录，保存账号详细信息
      if (isLoggedIn && response.data.data) {
        accountInfo.value = {
          user_id: response.data.data.user_id || '未知',
          username: response.data.data.username || '未知用户',
          used_space: response.data.data.used_space || 0,
          total_space: response.data.data.total_space || 0,
          member_level: response.data.data.member_level || '普通会员',
          expire_time: response.data.data.expire_time || '',
        }
      }
    } else {
      loginStatus.value = {
        title: '无法获取授权状态',
        type: 'error',
        description: response?.data.msg || '检查授权状态失败，请稍后重试',
      }
    }
  } catch (error) {
    console.error('检查登录状态错误:', error)
    loginStatus.value = {
      title: '状态检查出错',
      type: 'error',
      description: '检查过程中发生错误，请检查网络连接',
    }
  } finally {
    checkingStatus.value = false
  }
}

// 开始轮询二维码状态
const startPolling = () => {
  if (pollingTimer.value) {
    clearInterval(pollingTimer.value)
  }

  pollingTimer.value = setInterval(async () => {
    await checkQRStatus()
  }, 1000) // 每10秒检查一次
}

// 停止轮询
const stopPolling = () => {
  if (pollingTimer.value) {
    clearInterval(pollingTimer.value)
    pollingTimer.value = null
  }
}

// 检查二维码状态
const checkQRStatus = async () => {
  if (!qrCodeData.value) return

  try {
    // 构造JSON请求数据
    const requestData = {
      ...qrCodeData.value,
      device_type: loginData.device_type,
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
          // 延迟1秒后关闭对话框并刷新登录状态
          setTimeout(() => {
            closeQRDialog()
            checkLoginStatus()
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
}

// 刷新二维码
const refreshQRCode = async () => {
  qrStatus.value = 'waiting'
  qrCodeUrl.value = ''

  try {
    const requestData = {
      device_type: loginData.device_type,
    }

    const response = await http?.post(`${SERVER_URL}/auth/115-qrcode`, requestData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200 && response.data.data) {
      qrCodeData.value = response.data.data // 保存完整的扫码结果
      qrCodeContent.value = response.data.data.qrcode // 保存二维码内容
      qrCodeUrl.value = await generateQRCode(response.data.data.qrcode) // 生成二维码图片
      startPolling()
    } else {
      qrStatus.value = 'error'
    }
  } catch (error) {
    console.error('刷新二维码错误:', error)
    qrStatus.value = 'error'
  }
}

// 组件挂载时加载数据
let removeDeviceTypeListener: (() => void) | null = null

onMounted(() => {
  checkMobile()
  removeDeviceTypeListener = onDeviceTypeChange((newIsMobile) => {
    isMobile.value = newIsMobile
  })
  checkLoginStatus()
})

onUnmounted(() => {
  if (removeDeviceTypeListener) {
    removeDeviceTypeListener()
  }
  stopPolling()
})
</script>

<style scoped>
.core-settings-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.core-settings-card,
.settings-links-card {
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

.card-subtitle {
  margin: 0;
  font-size: 14px;
  color: #909399;
}

.core-content {
  margin-top: 20px;
}

.login-section,
.proxy-section {
  margin-bottom: 24px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 12px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.section-description {
  margin: 0 0 16px 0;
  font-size: 14px;
  color: #606266;
  line-height: 1.5;
}

.login-actions {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.login-status {
  margin-top: 16px;
}

.account-info {
  margin-top: 20px;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.info-title {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.info-item {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 1px solid #ebeef5;
}

.info-item:last-child {
  border-bottom: none;
}

.info-label {
  font-size: 13px;
  color: #606266;
  font-weight: 500;
  flex-shrink: 0;
}

.info-value {
  font-size: 14px;
  color: #303133;
  font-weight: 600;
  flex: 1;
}

.member-vip {
  color: #e6a23c;
}

.member-normal {
  color: #909399;
}

.expire-normal {
  color: #67c23a;
}

.expire-caution {
  color: #e6a23c;
}

.expire-warning {
  color: #f56c6c;
}

.expire-expired {
  color: #f56c6c;
  font-weight: bold;
}

.expire-unknown {
  color: #909399;
}

.storage-progress {
  margin-top: 16px;
}

.login-form {
  margin-top: 16px;
  margin-bottom: 20px;
}

.login-form .el-form-item {
  margin-bottom: 20px;
}

.login-form .form-help {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

.proxy-form {
  margin-top: 16px;
}

.proxy-form .el-form-item {
  margin-bottom: 20px;
}

.proxy-form .form-help {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

.proxy-actions {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.proxy-status {
  margin-top: 16px;
}

.settings-links {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  margin-top: 20px;
}

.settings-link-btn {
  height: 60px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
}

.settings-link-btn .el-icon {
  font-size: 20px;
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

/* 移动端适配 */
@media (max-width: 768px) {
  .core-settings-card,
  .settings-links-card {
    margin: 0;
  }

  .card-title {
    font-size: 20px;
  }

  .card-subtitle {
    font-size: 13px;
  }

  .login-actions {
    flex-direction: column;
    gap: 8px;
  }

  .login-actions .el-button {
    width: 100%;
  }

  .proxy-actions {
    flex-direction: column;
    gap: 8px;
  }

  .proxy-actions .el-button {
    width: 100%;
  }

  .info-grid {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .info-item {
    flex-direction: row;
    align-items: center;
    gap: 8px;
    padding: 12px 0;
  }

  .status-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .settings-links {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .settings-link-btn {
    width: 100%;
    height: 50px;
  }

  .section-title {
    font-size: 16px;
  }
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .card-title {
    font-size: 18px;
  }

  .section-title {
    font-size: 15px;
  }

  .section-description {
    font-size: 13px;
  }
}
</style>
