<template>
  <div class="main-content-container emby-content">
    <el-form :model="embyData" :rules="formRules" :label-position="isMobile ? 'top' : 'left'" :label-width="220"
      class="emby-form" ref="formRef">
      <!-- Emby服务器地址 -->
      <el-form-item label="Emby服务器地址" prop="emby_url">
        <el-input v-model="embyData.emby_url" placeholder="请输入Emby服务器地址，格式：http://ip:port" :disabled="embyLoading"
          class="limited-width-input" @input="updateEmbyExample" />
        <div v-if="embyExample" class="emby-example-inline">
          <span class="example-label">示例格式：</span>
          <code class="example-url">{{ embyExample }}</code>
        </div>
        <div>
          <p>想使用Emby外网302必须输入Emby服务器地址，不要以/结尾</p>
        </div>
      </el-form-item>
      <el-form-item label="Emby API密钥" prop="emby_api_key">
        <el-input v-model="embyData.emby_api_key" placeholder="请输入Emby API密钥" :disabled="embyLoading"
          class="limited-width-input" @input="updateEmbyExample" />
        <div>
          <p>API密钥用来提取strm的视频、音频、内封字幕信息，如果不需要该功能，可以不填 </p>
          <p>Strm信息提取功能由<a href="https://github.com/truewhile" target="_blank">@truewhile</a> 提供，感谢其无私的分享。</p>
        </div>
      </el-form-item>

      <!-- 同步和功能配置 -->
      <el-divider />
      <h4 style="margin: 10px 0">同步和功能配置</h4>

      <el-form-item label="Emby通知链接">
        <el-input v-model="webhookUrl" readonly class="limited-width-input">
          <template #append>
            <el-button @click="copyWebhookUrl" :icon="Check">复制</el-button>
          </template>
        </el-input>
        <div>将此链接配置到Emby的通知设置中，
          <a href="https://github.com/qicfan/qmediasync/wiki/Emby-%E9%80%9A%E7%9F%A5%E9%85%8D%E7%BD%AE" target="_blank">配置教程</a>
          <a :href="embyData.emby_url + '/web/index.html#!/settings/notifications.html'" target="_blank">去配置</a>
          如果下方开启了鉴权，请确保在Emby的通知链接中添加Api Key参数，示例：<code>{{webhookUrl}}?api_key=你的ApiKey</code>
        </div>
      </el-form-item>

      <el-form-item label="Emby通知链接是否启用鉴权" prop="enable_auth">
        <el-switch v-model="embyData.enable_auth" :active-value="1" :inactive-value="0" :disabled="embyLoading" />
        <span style="margin-left: 10px; color: #666">{{ embyData.enable_auth ? '启用' : '禁用' }}</span>
        <div>启用后，Emby的Webhook请求需要携带Api Key才能生效。如果要在外网使用Ebmy通知链接建议启用以提高安全性. 请到<router-link to="/settings/api-keys" class="api-key-link">Api Key模块</router-link>生成</div>
      </el-form-item>

      <el-form-item label="STRM同步完成后刷新媒体库" prop="enable_refresh_library">
        <el-switch v-model="embyData.enable_refresh_library" :active-value="1" :inactive-value="0" :disabled="embyLoading" />
        <span style="margin-left: 10px; color: #666">{{ embyData.enable_refresh_library ? '启用' : '禁用' }}</span>

        <div>该功能需要至少同步完一次Emby媒体库才能生效，如果下方同步管理卡片中的总项目数为0，请点击下方：启动同步 按钮 触发一次同步。</div>
        <div>功能解释：某个STRM同步目录同步完成后会自动触发相关联的Emby媒体库刷新，这样可以及时的将新增加的STRM文件入库</div>
      </el-form-item>

      <el-form-item label="Emby入库后自动提取媒体信息" prop="enable_extract_media_info">
        <el-switch v-model="embyData.enable_extract_media_info" :active-value="1" :inactive-value="0" :disabled="embyLoading" />
        <span style="margin-left: 10px; color: #666">{{ embyData.enable_extract_media_info ? '启用' : '禁用' }}</span>
        <div>该功能需要在Emby中配置通知才能生效，<a href="https://github.com/qicfan/qmediasync/wiki/Emby-%E9%80%9A%E7%9F%A5%E9%85%8D%E7%BD%AE" target="_blank">配置教程</a> <a :href="embyData.emby_url + '/web/index.html#!/settings/notifications.html'" target="_blank">去配置</a></div>
        <div>功能解释：QMediaSync在收到Emby的通知某个资源入库后，自动触发提取该资源的媒体信息，加快起播速度。媒体信息指：视频、音频、内封字幕等详细信息</div>
      </el-form-item>

      <el-form-item label="Emby删除联动删除网盘文件" prop="enable_delete_netdisk">
        <el-switch v-model="embyData.enable_delete_netdisk" :active-value="1" :inactive-value="0" :disabled="embyLoading" />
        <span style="margin-left: 10px; color: #d32f2f">{{ embyData.enable_delete_netdisk ? '启用' : '禁用' }}</span>
        <div style="margin-top: 8px; font-size: 12px; color: #d32f2f">
          <strong>⚠ 谨慎启用：</strong> 启用后，删除Emby中的项目时，对应的网盘文件也会被删除
        </div>
        <div>该功能需要在Emby中配置通知才能生效，<a href="https://github.com/qicfan/qmediasync/wiki/Emby-%E9%80%9A%E7%9F%A5%E9%85%8D%E7%BD%AE" target="_blank">配置教程</a> <a :href="embyData.emby_url + '/web/index.html#!/settings/notifications.html'" target="_blank">去配置</a></div>
        <div>如果在Emby中删除了电影，会在网盘中将视频文件的父目录一起删除</div>
        <div>如果在Emby中删除了剧，会在网盘中将tvshow.nfo的父目录删除</div>
        <div>如果在Emby中删除了季，会先检查视频文件的父目录，如果父目录是季文件夹则删除该文件夹；如果父目录是有tvshow的目录则仅删除季下所有集对应的视频文件+元数据（nfo、封面)</div>
        <div>如果在Emby中删了集，会删除视频文件+元数据（nfo、封面)</div>
      </el-form-item>

      <el-form-item>
        <!-- 保存和重置按钮 -->
        <div class="form-actions">
          <div>
            <el-button type="success" @click="saveEmbyConfig" :loading="embyLoading" :icon="Check">
              保存设置
            </el-button>
          </div>
          <div>
            <el-button type="warning" @click="praseEmby" :loading="embyLoading" :icon="Refresh"
              :disabled="!embyData.emby_url || !embyData.emby_api_key">
              提取媒体信息
            </el-button>
            <p>该功能会将Emby没有提取媒体信息的项目全部触发提取，如果是重建媒体库或者新Emby可以执行一次。进度或者详情请在<router-link to="/download-queue" class="api-key-link">下载队列页</router-link>面查看</p>
          </div>
        </div>
      </el-form-item>
    </el-form>

    <!-- 同步管理区域 -->
    <div class="sync-management-section">
      <h3>同步管理</h3>
      <div class="sync-controls">
        <el-button type="primary" @click="startSync" :loading="syncStartLoading" :icon="Refresh"
          :disabled="!embyData.emby_url || syncPolling">
          {{ syncPolling ? '同步进行中...' : '启动同步' }}
        </el-button>
      </div>

      <!-- 同步状态信息 -->
      <div v-if="syncInfo" class="sync-info-container">
        <el-card style="width: 580px">
          <template #header>
            <div class="card-header">
              <span>同步状态信息</span>
              <span v-if="syncPolling" class="sync-polling-indicator">
                <el-icon class="is-loading">
                  <Loading />
                </el-icon>
                同步进行中...
              </span>
            </div>
          </template>
          <el-row :gutter="20">
            <el-col :xs="24" :sm="12" :md="8">
              <div class="sync-info-item">
                <span class="label">是否启用自动同步：</span>
                <span class="value">{{ syncInfo.sync_enabled ? '是' : '否' }}</span>
              </div>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8">
              <div class="sync-info-item">
                <span class="label">同步周期：</span>
                <span class="value">{{ syncInfo.sync_cron || '1小时' }}</span>
              </div>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8">
              <div class="sync-info-item">
                <span class="label">关联的Emby Item数：</span>
                <span class="value">{{ syncInfo.total_items || 0 }}</span>
              </div>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8">
              <div class="sync-info-item">
                <span class="label">最后同步时间：</span>
                <span class="value">{{ formatLastSyncTime(syncInfo.last_sync_time) }}</span>
              </div>
            </el-col>
          </el-row>
        </el-card>
      </div>
    </div>

    <!-- 设置状态显示 -->
    <el-alert v-if="embyStatus" :title="embyStatus.title" :type="embyStatus.type" :description="embyStatus.description"
      :closable="false" show-icon class="emby-status" />
    <div class="security-content">
      <div class="warning-section">
        <el-alert title="使用提示" type="warning" :closable="false" show-icon>
          <template #default>
            只要填写了Emby服务器地址和API密钥，就可以触发提取媒体信息，提取完成后Emby可以显示出来音视频和内封字幕信息，可以切换字幕<br />
            如果需要同步，可以点击上方的 "全量提取媒体信息" 按钮
          </template>
        </el-alert>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { Check, Refresh, Loading } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { inject, onMounted, ref, reactive, onBeforeUnmount } from 'vue'
import { isMobile as checkIsMobile } from '@/utils/deviceUtils'

// HTTP客户端
const http: AxiosStatic | undefined = inject('$http')

// 表单引用
const formRef = ref<FormInstance>()

// 移动端检测
const isMobile = ref(checkIsMobile())

// 加载状态
const embyLoading = ref(false)

// 同步相关状态
const syncStartLoading = ref(false)
const syncPolling = ref(false)
const syncInfo = ref<{
  sync_enabled: boolean
  sync_cron: string
  total_items: number
  last_sync_time: number | null
} | null>(null)
let syncPollTimer: number | null = null

// Emby设置数据
const embyData = reactive({
  emby_url: '',
  emby_api_key: '',
  sync_enabled: 1,
  sync_cron: '0 2 * * *',
  enable_refresh_library: 1,
  enable_extract_media_info: 1,
  enable_delete_netdisk: 0,
  enable_auth: 1,
})

// 示例显示
const embyExample = ref('http://192.168.1.100:8096')

// Webhook URL 计算
const webhookUrl = ref('')
const updateWebhookUrl = () => {
  let baseUrl: string
  if (SERVER_URL === '/api') {
    // 如果 SERVER_URL 为空或为 "/"，使用当前访问的 HOST:PORT
    baseUrl = window.location.origin
  } else {
    baseUrl = SERVER_URL.replace(/\/api$/, '')
  }
  console.log(baseUrl)
  webhookUrl.value = `${baseUrl}/emby/webhook`
}

// 状态提示
const embyStatus = ref<{
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
} | null>(null)

// 表单验证规则
const formRules: FormRules = {
  emby_url: [
    {
      message: '请输入Emby服务器地址',
      trigger: 'blur',
    },
    {
      pattern: /^(http|https):\/\/[^\s/$.?#].[^\s]*$/, // 简单的URL验证正则
      message: '请输入有效的URL格式，如：http://ip:port',
      trigger: 'blur',
    },
  ],
}

// 默认配置
const defaultConfig = {
  emby_url: '',
  emby_api_key: '',
  sync_enabled: 1,
  sync_cron: '0 2 * * *',
  enable_refresh_library: 1,
  enable_extract_media_info: 1,
  enable_delete_netdisk: 0,
  enable_auth: 0,
}

// 加载Emby配置
const loadEmbyConfig = async () => {
  try {
    embyLoading.value = true
    const response = await http?.get(`${SERVER_URL}/setting/emby-config`)

    if (response?.data.code === 200) {
      if (response.data.data?.exists && response.data.data?.config) {
        // 填充数据到表单
        const config = response.data.data.config
        embyData.emby_url = config.emby_url || ''
        embyData.emby_api_key = config.emby_api_key || ''
        embyData.sync_enabled = config.sync_enabled ?? 1
        embyData.sync_cron = config.sync_cron || '0 2 * * *'
        embyData.enable_refresh_library = config.enable_refresh_library ?? 1
        embyData.enable_extract_media_info = config.enable_extract_media_info ?? 1
        embyData.enable_delete_netdisk = config.enable_delete_netdisk ?? 0
        embyData.enable_auth = config.enable_auth ?? 1
      } else {
        // 配置不存在，使用默认配置
        Object.assign(embyData, defaultConfig)
      }
    } else {
      // 使用默认配置
      Object.assign(embyData, defaultConfig)
      ElMessage.warning('加载Emby配置失败，使用默认配置')
    }
  } catch (error) {
    console.error('加载Emby配置错误:', error)
    Object.assign(embyData, defaultConfig)
    ElMessage.error('加载Emby配置失败')
  } finally {
    embyLoading.value = false
  }
}

// 保存Emby配置
const saveEmbyConfig = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    embyLoading.value = true

    const response = await http?.post(
      `${SERVER_URL}/setting/emby-config`,
      {
        emby_url: embyData.emby_url.trim(),
        emby_api_key: embyData.emby_api_key.trim(),
        sync_enabled: 1,
        sync_cron: embyData.sync_cron,
        enable_refresh_library: embyData.enable_refresh_library,
        enable_extract_media_info: embyData.enable_extract_media_info,
        enable_delete_netdisk: embyData.enable_delete_netdisk,
        enable_auth: embyData.enable_auth,
      },
      {
        headers: {
          'Content-Type': 'application/json',
        },
      },
    )

    if (response?.data.code === 200) {
      embyStatus.value = {
        title: '保存成功',
        type: 'success',
        description: 'Emby配置已成功保存',
      }
      ElMessage.success('Emby配置已成功保存')
      // 重新加载配置
      await loadEmbyConfig()
    } else {
      embyStatus.value = {
        title: '保存失败',
        type: 'error',
        description: response?.data.message || '保存Emby配置失败',
      }
      ElMessage.error(response?.data.message || '保存失败')
    }

    // 3秒后隐藏状态提示
    setTimeout(() => {
      embyStatus.value = null
    }, 3000)
  } catch (error) {
    console.error('保存Emby配置错误:', error)
    embyStatus.value = {
      title: '保存失败',
      type: 'error',
      description: '保存Emby配置时出现错误',
    }
    ElMessage.error('保存失败')
  } finally {
    embyLoading.value = false
  }
}

const praseEmby = async () => {
  try {
    embyLoading.value = true
    const response = await http?.post(
      `${SERVER_URL}/setting/emby/parse`,
      {
        emby_url: embyData.emby_url.trim(),
        emby_api_key: embyData.emby_api_key.trim(),
      },
      {
        headers: {
          'Content-Type': 'application/json',
        },
      },
    )

    if (response?.data.code === 200) {
      embyStatus.value = {
        title: '触发提取媒体信息成功',
        type: 'success',
        description: '已成功触发提取媒体信息',
      }
      ElMessage.success('已成功触发提取媒体信息')
    } else {
      embyStatus.value = {
        title: '触发提取媒体信息失败',
        type: 'error',
        description: response?.data.message || '触发提取媒体信息失败',
      }
      ElMessage.error(response?.data.message || '触发提取媒体信息失败')
    }
  } catch (error) {
    console.error('触发提取媒体信息错误:', error)
    embyStatus.value = {
      title: '触发提取媒体信息失败',
      type: 'error',
      description: '触发提取媒体信息时出现错误',
    }
    ElMessage.error('触发提取媒体信息失败')
  } finally {
    embyLoading.value = false
  }
}

// 更新示例
const updateEmbyExample = () => {
  // 根据用户输入动态更新示例（如果需要）
}

// 复制 Webhook URL
const copyWebhookUrl = async () => {
  try {
    await navigator.clipboard.writeText(webhookUrl.value)
    ElMessage.success('Webhook链接已复制到剪贴板')
  } catch (error) {
    console.error('复制失败:', error)
    ElMessage.error('复制失败，请手动复制')
  }
}

// 启动同步
const startSync = async () => {
  try {
    syncStartLoading.value = true
    const response = await http?.post(`${SERVER_URL}/emby/sync/start`)

    if (response?.data.code === 200) {
      ElMessage.success('同步已启动')
      syncPolling.value = true
      // 立即查询一次同步状态
      await querySyncStatus()
      // 然后开始轮询
      startSyncPolling()
    } else {
      ElMessage.error(response?.data.message || '启动同步失败')
    }
  } catch (error) {
    console.error('启动同步错误:', error)
    ElMessage.error('启动同步失败')
  } finally {
    syncStartLoading.value = false
  }
}

// 查询同步状态
const querySyncStatus = async () => {
  try {
    const response = await http?.get(`${SERVER_URL}/emby/sync/status`)

    if (response?.data.code === 200) {
      syncInfo.value = response.data.data
      // 如果同步还在进行中，继续轮询
      if (response.data.data?.sync_enabled || syncPolling.value) {
        // 继续轮询
      } else {
        // 同步已完成
        stopSyncPolling()
      }
    }
  } catch (error) {
    console.error('查询同步状态错误:', error)
  }
}

// 开始轮询同步状态
const startSyncPolling = () => {
  // 每3秒轮询一次
  syncPollTimer = window.setInterval(async () => {
    try {
      const response = await http?.get(`${SERVER_URL}/emby/sync/status`)

      if (response?.data.code === 200) {
        syncInfo.value = response.data.data
        // 如果同步完成或禁用，停止轮询
        if (!syncPolling.value) {
          stopSyncPolling()
        }
      }
    } catch (error) {
      console.error('轮询同步状态错误:', error)
    }
  }, 3000)
}

// 停止轮询同步状态
const stopSyncPolling = () => {
  syncPolling.value = false
  if (syncPollTimer !== null) {
    clearInterval(syncPollTimer)
    syncPollTimer = null
  }
}

// 格式化最后同步时间
const formatLastSyncTime = (timestamp: number | null | undefined) => {
  if (!timestamp) return '未同步'

  try {
    const date = new Date(timestamp * 1000)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMs / 3600000)
    const diffDays = Math.floor(diffMs / 86400000)

    if (diffMins < 1) return '刚刚'
    if (diffMins < 60) return `${diffMins}分钟前`
    if (diffHours < 24) return `${diffHours}小时前`
    if (diffDays < 30) return `${diffDays}天前`

    return date.toLocaleString('zh-CN')
  } catch {
    return timestamp
  }
}

onMounted(() => {
  loadEmbyConfig()
  // 初始化时查询一次同步状态
  querySyncStatus()
  // 初始化 webhook URL
  updateWebhookUrl()
})

// 组件卸载时清理定时器
onBeforeUnmount(() => {
  stopSyncPolling()
})
</script>

<style scoped lang="css">
.emby-content {
  padding: 20px;
}

.emby-form {
  margin-bottom: 30px;
}

.emby-example-inline {
  margin-top: 8px;
  padding: 10px;
  background-color: #f5f7fa;
  border-radius: 4px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.example-label {
  font-weight: 500;
  color: #666;
}

.example-url {
  color: #409eff;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.form-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.form-actions div {
  display: flex;
  gap: 10px;
}

.emby-status {
  margin-bottom: 20px;
}

.security-content {
  margin-top: 20px;
}

.warning-section {
  margin-top: 20px;
}

.sync-management-section {
  background-color: #fff;
  padding: 20px;
  border-radius: 4px;
  margin-top: 20px;
  border: 1px solid #ebeef5;
}

.sync-management-section h3 {
  margin: 0 0 20px 0;
  color: #303133;
  font-size: 16px;
}

.sync-controls {
  margin-bottom: 20px;
}

.sync-info-container {
  margin-top: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.sync-polling-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #409eff;
  font-size: 14px;
  font-weight: 500;
}

.is-loading {
  animation: rotating 2s linear infinite;
}

@keyframes rotating {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}

.sync-info-item {
  padding: 12px;
  background-color: #f5f7fa;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sync-info-item .label {
  color: #606266;
  font-size: 14px;
  font-weight: 500;
}

.sync-info-item .value {
  color: #303133;
  font-size: 16px;
  font-weight: 600;
  word-break: break-all;
}

.api-key-link {
  color: #409eff;
  text-decoration: none;
  font-weight: 500;
}

.api-key-link:hover {
  color: #66b1ff;
  text-decoration: underline;
}

@media (max-width: 768px) {
  .emby-content {
    padding: 10px;
  }

  .form-actions {
    flex-direction: column;
  }

  .form-actions div {
    width: 100%;
  }

  .form-actions div button {
    width: 100%;
  }

  .sync-management-section {
    padding: 15px;
  }

  .sync-info-item {
    margin-bottom: 10px;
  }
}
</style>
