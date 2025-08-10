<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import MarkdownIt from 'markdown-it'
import { inject, onMounted, ref } from 'vue'

interface AccountInfo {
  user_id: string
  username: string
  used_space: number
  total_space: number
  member_level: string
  expire_time: string
}

interface VersionInfo {
  version: string
  date: string
}

const http: AxiosStatic | undefined = inject('$http')
const accountInfo = ref<AccountInfo | null>(null)
const versionInfo = ref<VersionInfo | null>(null)
const accountLoading = ref(true)
const versionLoading = ref(true)

const md = new MarkdownIt()
const content = `
## 介绍

- 基于 115 开放平台接口来同步生成 STRM 和下载元数据，并且提供直链解析服务，不依赖其他项目。
- 原理：定时同步 115 的文件树根本地文件树对比：
- 1. 本地存在网盘不存在则删除本地文件（测试版本不会删除任何文件）
- 2. 本地不存在网盘存在则创建本地文件（STRM 或元数据下载）
- 3. 本地存在且网盘存在，则判断文件是否一致（文件 pick_code 是否相同），一致则不处理，不一致则更新
- 实测 3W 多文件大概 22T 的库需要 5 分钟左右完成目录树生成和对比，首次下载元数据需要的时间不定，可能在 1-2 个小时左右。
- 定时任务设定不能小于 1 小时间隔，建议设置成12点到23点每隔1小时执行一次：0 12-23 * * *

### 功能列表

- [x] 115 开放平台接入
- [x] STRM 生成
- [x] 元数据下载
- [x] 使用CookieCloud同步115网页Cookie（后续可以调用115 api）
- [x] 接入Telegram通知
- [x] 目录下新建隐藏文件.meta记录原始信息（供以后使用）
- [x] 元数据新增同名的隐藏文件.name.meta记录原始信息（供以后使用）
- [x] 同步时上传网盘不存在的元数据（STRM设置开启）
- [x] 同步时删除网盘不存在的STRM文件（STRM设置开启）
- [x] 同步时删除网盘不存在且本地为空的文件夹（STRM设置开启）
- [ ] 接入资源库(需要通过获取115 Cookie才能使用转存等服务)
- [ ] emby 302（待定，优先级低）
- [ ] 影片整理（待定， 优先级最低）

## 使用步骤：
1. 系统设置-核心设置-扫码授权115开放平台，页面显示出您的账号信息后表示授权成功
3. 系统设置-strm设置：输入strm直连地址，其他参数请根据需要修改
5. 同步记录 - 手动同步 进行首次全量同步（可能时间较长）
`
const result = md.render(content)

// 格式化存储空间
const formatStorage = (bytes: number): string => {
  if (!bytes || bytes === 0) return '0 B'

  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const size = bytes / Math.pow(1024, i)

  return `${size.toFixed(i === 0 ? 0 : 2)} ${sizes[i]}`
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

// 格式化到期时间
const formatExpireTime = (expireTime: string): string => {
  if (!expireTime) return '未知'

  const date = new Date(expireTime)
  if (isNaN(date.getTime())) return expireTime

  const now = new Date()
  const diffTime = date.getTime() - now.getTime()
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24))

  if (diffDays < 0) return '已过期'
  if (diffDays === 0) return '今天到期'
  if (diffDays <= 30) return `${diffDays}天后到期`

  return date.toLocaleDateString('zh-CN')
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
  if (diffDays <= 30) return 'expire-notice'

  return 'expire-normal'
}

// 加载115账号信息
const loadAccountInfo = async () => {
  try {
    accountLoading.value = true
    const response = await http?.get(`${SERVER_URL}/115/account`)

    if (response?.data.code === 200 && response.data.data) {
      accountInfo.value = response.data.data
    } else {
      accountInfo.value = null
    }
  } catch (error) {
    console.error('加载115账号信息错误:', error)
    accountInfo.value = null
  } finally {
    accountLoading.value = false
  }
}

// 加载系统版本信息
const loadVersionInfo = async () => {
  try {
    versionLoading.value = true
    const response = await http?.get(`${SERVER_URL}/version`)

    if (response?.data.code === 200 && response.data.data) {
      versionInfo.value = response.data.data
    } else {
      versionInfo.value = null
    }
  } catch (error) {
    console.error('加载系统版本信息错误:', error)
    versionInfo.value = null
  } finally {
    versionLoading.value = false
  }
}

onMounted(() => {
  loadAccountInfo()
  loadVersionInfo()
})
</script>
<template>
  <div class="home-container">
    <!-- 115账号信息卡片 -->
    <el-card class="account-card" shadow="hover" v-loading="accountLoading">
      <template #header>
        <h2 class="card-title">115账号信息</h2>
        <p class="card-subtitle">当前登录的115开放平台账号</p>
      </template>

      <div v-if="accountInfo" class="account-info">
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

      <div v-else class="no-account">
        <el-empty description="暂未获取到115账号信息">
          <el-button type="primary" @click="$router.push('/settings')">前往授权</el-button>
        </el-empty>
      </div>
    </el-card>

    <!-- 系统版本信息卡片 -->
    <el-card class="version-card" shadow="hover" v-loading="versionLoading">
      <template #header>
        <h2 class="card-title">系统信息</h2>
        <p class="card-subtitle">当前系统版本和编译信息</p>
      </template>

      <div v-if="versionInfo" class="version-info">
        <div class="version-item">
          <span class="version-label">系统版本:</span>
          <span class="version-value">{{ versionInfo.version }}</span>
        </div>
        <div class="version-item">
          <span class="version-label">编译时间:</span>
          <span class="version-value">{{ versionInfo.date }}</span>
        </div>
      </div>

      <div v-else class="no-version">
        <el-empty description="暂未获取到系统版本信息" />
      </div>
    </el-card>

    <!-- 项目介绍 -->
    <el-card class="intro-card" shadow="hover">
      <template #header>
        <h2 class="card-title">项目介绍</h2>
        <p class="card-subtitle">115 STRM 服务使用说明</p>
      </template>

      <div class="intro-content" v-html="result"></div>
    </el-card>
  </div>
</template>

<style scoped>
.home-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.account-card,
.version-card,
.intro-card {
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

/* 115账号信息样式 */
.account-info {
  margin-top: 16px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.info-label {
  font-size: 14px;
  font-weight: 600;
  color: #606266;
}

.info-value {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
}

.member-vip {
  color: #f56c6c !important;
  font-weight: 600;
}

.member-normal {
  color: #909399;
}

.expire-normal {
  color: #67c23a;
}

.expire-notice {
  color: #e6a23c;
}

.expire-warning {
  color: #f56c6c;
}

.expire-expired {
  color: #f56c6c;
  font-weight: 600;
}

.expire-unknown {
  color: #909399;
}

.storage-progress {
  margin-top: 16px;
}

.no-account {
  padding: 40px 20px;
  text-align: center;
}

/* 系统版本信息样式 */
.version-info {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.version-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.version-label {
  font-size: 14px;
  font-weight: 600;
  color: #606266;
  margin-right: 16px;
  min-width: 80px;
}

.version-value {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.no-version {
  padding: 40px 20px;
  text-align: center;
}

/* 项目介绍样式 */
.intro-content {
  margin-top: 16px;
  line-height: 1.6;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .account-card,
  .version-card,
  .intro-card {
    margin: 0;
  }

  .card-title {
    font-size: 20px;
  }

  .card-subtitle {
    font-size: 13px;
  }

  .info-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .info-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .info-label {
    font-size: 13px;
  }

  .info-value {
    font-size: 13px;
  }

  .version-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .version-label {
    font-size: 13px;
    margin-right: 0;
    min-width: auto;
  }

  .version-value {
    font-size: 13px;
  }

  .intro-content {
    font-size: 14px;
  }

  .intro-content h2 {
    font-size: 18px;
    margin-top: 20px;
    margin-bottom: 12px;
  }

  .intro-content h2:first-child {
    margin-top: 0;
  }

  .intro-content ol,
  .intro-content ul {
    padding-left: 20px;
  }

  .intro-content li {
    margin-bottom: 8px;
  }

  .intro-content blockquote {
    margin: 10px 0;
    padding: 8px 12px;
    border-left: 3px solid #409eff;
    background-color: #f4f4f5;
    font-size: 13px;
  }

  .intro-content code {
    padding: 2px 4px;
    background-color: #f1f1f1;
    border-radius: 3px;
    font-size: 12px;
  }
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .card-title {
    font-size: 18px;
  }

  .intro-content {
    font-size: 13px;
  }

  .intro-content h2 {
    font-size: 16px;
  }

  .info-item,
  .version-item {
    padding: 10px 12px;
  }
}
</style>
