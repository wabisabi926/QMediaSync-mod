<template>
  <div class="main-content-container emby-content">
    <div class="emby-settings-wrapper">
      <el-form :model="embyData" :rules="formRules" :label-position="isMobile ? 'top' : 'left'" :label-width="200"
        class="emby-form" ref="formRef">
        <el-card class="settings-card emby-server-card" shadow="hover">
          <template #header>
            <div class="card-header-wrapper">
              <div class="card-header-icon server-icon">
                <el-icon :size="24"><Monitor /></el-icon>
              </div>
              <div class="card-header-content">
                <h3 class="card-title">Emby服务器配置</h3>
                <p class="card-subtitle">配置Emby服务器连接信息</p>
              </div>
            </div>
          </template>

          <el-form-item label="Emby服务器地址" prop="emby_url">
            <el-input v-model="embyData.emby_url" placeholder="请输入Emby服务器地址，格式：http://ip:port" :disabled="embyLoading"
              class="limited-width-input" @input="updateEmbyExample" :prefix-icon="Link" clearable />
            <p style="color: red; font-weight: bold; margin-left: 16px; font-size: 28px;">输入Emby地址后，页面往下滚，点击"保存设置"按钮后，重启QMediaSync才能生效</p>
            <div v-if="embyExample" class="emby-example-inline">
              <span class="example-label">示例格式：</span>
              <code class="example-url">{{ embyExample }}</code>
            </div>
            <div class="form-help">
              <el-icon><InfoFilled /></el-icon>
              <span>想使用Emby外网302必须输入Emby服务器地址，不要以/结尾，输入emby的内网地址即可，比如：http://192.168.1.100:8096 或者docker容器的地址</span>

            </div>
          </el-form-item>

          <el-form-item label="Emby API密钥" prop="emby_api_key">
            <el-input v-model="embyData.emby_api_key" placeholder="请输入Emby API密钥" :disabled="embyLoading"
              class="limited-width-input" @input="updateEmbyExample" :prefix-icon="Key" />
            <div class="form-help">
              <el-icon><InfoFilled /></el-icon>
              <span>API密钥用来提取strm的视频、音频、内封字幕信息，如果不需要该功能，可以不填</span>
            </div>
            <div class="form-help author-credit">
              <span>Strm信息提取功能由<a href="https://github.com/truewhile" target="_blank">@truewhile</a> 提供，感谢其无私的分享。</span>
            </div>
          </el-form-item>
        </el-card>

        <el-card class="settings-card webhook-card" shadow="hover">
          <template #header>
            <div class="card-header-wrapper">
              <div class="card-header-icon webhook-icon">
                <el-icon :size="24"><Connection /></el-icon>
              </div>
              <div class="card-header-content">
                <h3 class="card-title">通知链接配置</h3>
                <p class="card-subtitle">配置Emby与QMediaSync的通知连接</p>
              </div>
            </div>
          </template>

          <el-form-item label="Emby通知链接">
            <el-input v-model="webhookUrl" readonly class="limited-width-input webhook-input" :prefix-icon="Link">
              <template #append>
                <el-button @click="copyWebhookUrl" :icon="DocumentCopy">复制</el-button>
              </template>
            </el-input>
            <div class="form-help">
              <el-icon><InfoFilled /></el-icon>
              <span>将此链接配置到Emby的通知设置中，</span>
              <a href="https://github.com/qicfan/qmediasync/wiki/Emby-%E9%80%9A%E7%9F%A5%E9%85%8D%E7%BD%AE" target="_blank" class="help-link">配置教程</a>
              <a :href="embyData.emby_url + '/web/index.html#!/settings/notifications.html'" target="_blank" class="help-link action-link">去配置</a>
            </div>
            <div class="form-help" v-if="embyData.enable_auth">
              <el-icon><WarningFilled /></el-icon>
              <span class="warning-text">已开启鉴权，请确保在Emby的通知链接中添加Api Key参数，示例：<code class="inline-code">{{ webhookUrl }}?api_key=你的ApiKey</code></span>
            </div>
          </el-form-item>

          <el-form-item label="通知链接鉴权" prop="enable_auth">
            <div class="switch-wrapper">
              <el-switch v-model="embyData.enable_auth" :active-value="1" :inactive-value="0" :disabled="embyLoading"
                active-color="#67c23a" inactive-color="#dcdfe6" />
              <span class="switch-label" :class="{ 'is-active': embyData.enable_auth }">
                {{ embyData.enable_auth ? '已启用鉴权' : '已禁用鉴权' }}
              </span>
            </div>
            <div class="form-help">
              <el-icon><InfoFilled /></el-icon>
              <span>启用后，Emby的Webhook请求需要携带Api Key才能生效。如果要在外网使用Emby通知链接建议启用以提高安全性。请到<router-link to="/settings/api-keys" class="help-link">Api Key模块</router-link>生成</span>
            </div>
          </el-form-item>
        </el-card>

        <el-card class="settings-card sync-features-card" shadow="hover">
          <template #header>
            <div class="card-header-wrapper">
              <div class="card-header-icon features-icon">
                <el-icon :size="24"><Setting /></el-icon>
              </div>
              <div class="card-header-content">
                <h3 class="card-title">同步和功能配置</h3>
                <p class="card-subtitle">配置STRM同步与媒体库联动功能</p>
              </div>
            </div>
          </template>

           <div class="feature-item">
            <el-form-item label="入库后提取媒体信息" prop="enable_extract_media_info">
              <div class="switch-wrapper">
                <el-switch v-model="embyData.enable_extract_media_info" :active-value="1" :inactive-value="0"
                  :disabled="embyLoading" active-color="#67c23a" inactive-color="#dcdfe6" />
                <span class="switch-label" :class="{ 'is-active': embyData.enable_extract_media_info }">
                  {{ embyData.enable_extract_media_info ? '启用' : '禁用' }}
                </span>
              </div>
            </el-form-item>
            <div class="feature-description">
              <div class="config-links">
                <span>该功能需要在Emby中配置通知才能生效，</span>
                <a href="https://github.com/qicfan/qmediasync/wiki/Emby-%E9%80%9A%E7%9F%A5%E9%85%8D%E7%BD%AE" target="_blank" class="help-link">配置教程</a>
                <a :href="embyData.emby_url + '/web/index.html#!/settings/notifications.html'" target="_blank" class="help-link action-link">去配置</a>
              </div>
              <p class="feature-note">功能解释：QMediaSync在收到Emby的通知某个资源入库后，自动触发提取该资源的媒体信息，加快起播速度。媒体信息指：视频、音频、内封字幕等详细信息</p>
            </div>
          </div>

          <el-divider class="feature-divider" />

          <div class="feature-item">
            <el-form-item label="启用同步" prop="sync_enabled">
              <div class="switch-wrapper">
                <el-switch v-model="embyData.sync_enabled" :active-value="1" :inactive-value="0"
                  :disabled="embyLoading" active-color="#67c23a" inactive-color="#dcdfe6" />
                <span class="switch-label" :class="{ 'is-active': embyData.sync_enabled }">
                  {{ embyData.sync_enabled ? '已启用' : '已禁用' }}
                </span>
              </div>
            </el-form-item>
            <div class="feature-description">
              <p class="feature-note">启用后可以将Emby中的资源同步到QMediaSync中和网盘文件建立联系，来实现同步后刷新媒体库和联动删除网盘文件功能</p>
            </div>
          </div>

          <el-divider class="feature-divider" />

          <div class="feature-item" :class="{ 'is-disabled': !embyData.sync_enabled }">
            <el-form-item label="同步时间" prop="sync_cron">
              <el-input v-model="embyData.sync_cron" placeholder="请输入Cron表达式，如：0 2 * * *"
                :disabled="embyLoading || !embyData.sync_enabled"
                class="limited-width-input" @blur="fetchCronNextTimes" clearable />
              <div class="form-help">
                <el-icon><InfoFilled /></el-icon>
                <span>Cron表达式，格式：秒 分 时 日 月 周（如：0 2 * * * 表示每天凌晨2点执行）</span>
              </div>
            </el-form-item>
            <div v-if="cronNextTimes.length > 0" class="cron-next-times">
              <div class="cron-times-header">
                <el-icon><Clock /></el-icon>
                <span>接下来5次执行时间：</span>
              </div>
              <ul class="cron-times-list">
                <li v-for="(time, index) in cronNextTimes" :key="index">{{ time }}</li>
              </ul>
            </div>
          </div>

          <el-divider class="feature-divider" />

          <div class="feature-item" :class="{ 'is-disabled': !embyData.sync_enabled }">
            <el-form-item label="同步后刷新媒体库" prop="enable_refresh_library">
              <div class="switch-wrapper">
                <el-switch v-model="embyData.enable_refresh_library" :active-value="1" :inactive-value="0"
                  :disabled="embyLoading || !embyData.sync_enabled" active-color="#67c23a" inactive-color="#dcdfe6" />
                <span class="switch-label" :class="{ 'is-active': embyData.enable_refresh_library }">
                  {{ embyData.enable_refresh_library ? '启用' : '禁用' }}
                </span>
              </div>
            </el-form-item>
            <div class="feature-description">
              <p>该功能需要至少同步完一次Emby媒体库才能生效，如果下方同步管理卡片中的总项目数为0，请点击下方：启动同步 按钮触发一次同步。</p>
              <p>STRM同步完成后会延迟30s执行刷新动作，以供元数据下载（如果开启了下载），但是可能下载不完就触发了刷新，做为备份手段：请开启Emby的实时监控</p>
              <p class="feature-note">功能解释：某个STRM同步目录同步完成后会自动触发相关联的Emby媒体库刷新，这样可以及时的将新增加的STRM文件入库</p>
            </div>
          </div>

          <el-divider class="feature-divider" />

          <div class="feature-item danger-item" :class="{ 'is-disabled': !embyData.sync_enabled }">
            <el-form-item label="删除联动删除网盘文件" prop="enable_delete_netdisk">
              <div class="switch-wrapper">
                <el-switch v-model="embyData.enable_delete_netdisk" :active-value="1" :inactive-value="0"
                  :disabled="embyLoading || !embyData.sync_enabled" active-color="#f56c6c" inactive-color="#dcdfe6" />
                <span class="switch-label" :class="{ 'is-danger': embyData.enable_delete_netdisk }">
                  {{ embyData.enable_delete_netdisk ? '启用' : '禁用' }}
                </span>
              </div>
            </el-form-item>
            <div class="feature-description">
              <el-alert type="warning" :closable="false" class="danger-alert">
                <template #default>
                  <strong>⚠ 谨慎启用：</strong> 启用后，删除Emby中的项目时，对应的网盘文件也会被删除<br />
                  <strong>由于Emby的特性如果strm文件内容变更，Emby会先删除再新增，这时有概率导致：STRM变更→Emby通知删除→QMS联动删除网盘→Emby新增项目→播放失败</strong>，这个问题暂时无解<br />
                  <strong>如果打开了Emby的实时监控，在文件系统内删除Strm或者文件夹也会导致Emby触发删除通知→QMS联动删除网盘，所有删除文件一定要谨慎。</strong>
                </template>
              </el-alert>
              <div class="config-links">
                <span>该功能需要在Emby中配置通知才能生效，</span>
                <a href="https://github.com/qicfan/qmediasync/wiki/Emby-%E9%80%9A%E7%9F%A5%E9%85%8D%E7%BD%AE" target="_blank" class="help-link">配置教程</a>
                <a :href="embyData.emby_url + '/web/index.html#!/settings/notifications.html'" target="_blank" class="help-link action-link">去配置</a>
              </div>
              <ul class="delete-rules">
                <li>如果在Emby中删除了电影，会在网盘中将视频文件的父目录一起删除</li>
                <li>如果在Emby中删除了剧，会在网盘中将tvshow.nfo的父目录删除</li>
                <li>如果在Emby中删除了季，会先检查视频文件的父目录，如果父目录是季文件夹则删除该文件夹；如果父目录是有tvshow的目录则仅删除季下所有集对应的视频文件+元数据（nfo、封面）</li>
                <li>如果在Emby中删了集，会删除视频文件+元数据（nfo、封面）</li>
              </ul>
            </div>
          </div>
        </el-card>

        <div class="form-actions-wrapper">
          <el-button type="success" @click="saveEmbyConfig" :loading="embyLoading" :icon="Check" size="large" class="save-btn">
            保存设置
          </el-button>
          <el-button type="primary" @click="praseEmby" :loading="embyLoading" :icon="Refresh"
            :disabled="!embyData.emby_url || !embyData.emby_api_key" size="large" class="extract-btn">
            提取媒体信息
          </el-button>
          <div class="extract-help">
            <p>该功能会将Emby没有提取媒体信息的项目全部触发提取，如果是重建媒体库或者新Emby可以执行一次。进度或者详情请在<router-link to="/download-queue" class="help-link">下载队列页</router-link>面查看</p>
          </div>
        </div>
      </el-form>

      <el-card class="sync-management-card" shadow="hover">
        <template #header>
          <div class="card-header-wrapper">
            <div class="card-header-icon sync-icon">
              <el-icon :size="24"><Refresh /></el-icon>
            </div>
            <div class="card-header-content">
              <h3 class="card-title">同步管理</h3>
              <p class="card-subtitle">管理Emby媒体库同步状态</p>
            </div>
            <div class="card-header-action">
              <el-button type="primary" @click="startSync" :loading="syncStartLoading" :icon="Refresh"
                :disabled="!embyData.emby_url || !embyData.sync_enabled || syncPolling" size="default">
                {{ syncPolling ? '同步进行中...' : '启动同步' }}
              </el-button>
            </div>
          </div>
        </template>

        <div v-if="syncInfo" class="sync-info-grid">
          <div class="sync-stat-card">
            <div class="stat-icon auto-sync-icon">
              <el-icon :size="28"><Timer /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-label">自动同步</div>
              <div class="stat-value" :class="{ 'is-enabled': syncInfo.sync_enabled }">
                {{ syncInfo.sync_enabled ? '已启用' : '已禁用' }}
              </div>
            </div>
          </div>

          <div class="sync-stat-card">
            <div class="stat-icon cycle-icon">
              <el-icon :size="28"><Clock /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-label">同步周期</div>
              <div class="stat-value">{{ syncInfo.sync_cron }}</div>
            </div>
          </div>

          <div class="sync-stat-card">
            <div class="stat-icon items-icon">
              <el-icon :size="28"><FolderOpened /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-label">关联Item数</div>
              <div class="stat-value highlight">{{ syncInfo.total_items || 0 }}</div>
            </div>
          </div>

          <div class="sync-stat-card">
            <div class="stat-icon time-icon">
              <el-icon :size="28"><Calendar /></el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-label">最后同步时间</div>
              <div class="stat-value">{{ formatLastSyncTime(syncInfo.last_sync_time) }}</div>
            </div>
          </div>
        </div>

        <div v-if="syncPolling" class="sync-progress">
          <div class="progress-indicator">
            <el-icon class="is-loading" :size="20"><Loading /></el-icon>
            <span>同步进行中，请稍候...</span>
          </div>
        </div>

        <div v-if="!syncInfo" class="sync-empty">
          <el-empty description="暂无同步数据，请点击上方按钮启动同步" :image-size="80" />
        </div>
      </el-card>

      <el-alert v-if="embyStatus" :title="embyStatus.title" :type="embyStatus.type" :description="embyStatus.description"
        :closable="false" show-icon class="emby-status-alert" />

      <el-alert title="使用提示" type="info" :closable="false" show-icon class="tips-alert">
        <template #default>
          只要填写了Emby服务器地址和API密钥，就可以触发提取媒体信息，提取完成后Emby可以显示出来音视频和内封字幕信息，可以切换字幕<br />
          如果需要同步，可以点击上方的 "提取媒体信息" 按钮
        </template>
      </el-alert>
    </div>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { Check, Refresh, Loading, Monitor, Link, Key, Connection, Setting, DocumentCopy, InfoFilled, WarningFilled, Timer, Clock, FolderOpened, Calendar } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { inject, onMounted, ref, reactive, onBeforeUnmount } from 'vue'
import { isMobile as checkIsMobile } from '@/utils/deviceUtils'

const http: AxiosStatic | undefined = inject('$http')

const formRef = ref<FormInstance>()

const isMobile = ref(checkIsMobile())

const embyLoading = ref(false)

const syncStartLoading = ref(false)
const syncPolling = ref(false)
const syncInfo = ref<{
  sync_enabled: boolean
  sync_cron: string
  total_items: number
  last_sync_time: number | null
} | null>(null)
let syncPollTimer: number | null = null

const cronNextTimes = ref<string[]>([])

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

const embyExample = ref('http://192.168.1.100:8096')

const webhookUrl = ref('')
const updateWebhookUrl = () => {
  let baseUrl: string
  if (SERVER_URL === '/api') {
    baseUrl = window.location.origin
  } else {
    baseUrl = SERVER_URL.replace(/\/api$/, '')
  }
  console.log(baseUrl)
  webhookUrl.value = `${baseUrl}/emby/webhook`
}

const embyStatus = ref<{
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
} | null>(null)

const formRules: FormRules = {
  emby_url: [
    {
      message: '请输入Emby服务器地址',
      trigger: 'blur',
    },
    {
      pattern: /^(http|https):\/\/[^\s/$.?#].[^\s]*$/,
      message: '请输入有效的URL格式，如：http://ip:port',
      trigger: 'blur',
    },
  ],
}

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

const loadEmbyConfig = async () => {
  try {
    embyLoading.value = true
    const response = await http?.get(`${SERVER_URL}/setting/emby-config`)

    if (response?.data.code === 200) {
      if (response.data.data?.exists && response.data.data?.config) {
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
        Object.assign(embyData, defaultConfig)
      }
    } else {
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
        sync_enabled: embyData.sync_enabled,
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
      await loadEmbyConfig()
    } else {
      embyStatus.value = {
        title: '保存失败',
        type: 'error',
        description: response?.data.message || '保存Emby配置失败',
      }
      ElMessage.error(response?.data.message || '保存失败')
    }

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

const updateEmbyExample = () => {
}

const copyWebhookUrl = async () => {
  try {
    await navigator.clipboard.writeText(webhookUrl.value)
    ElMessage.success('Webhook链接已复制到剪贴板')
  } catch (error) {
    console.error('复制失败:', error)
    ElMessage.error('复制失败，请手动复制')
  }
}

const fetchCronNextTimes = async () => {
  if (!embyData.sync_cron || !embyData.sync_cron.trim()) {
    cronNextTimes.value = []
    return
  }

  try {
    const response = await http?.get(`${SERVER_URL}/setting/cron`, {
      params: { cron: embyData.sync_cron.trim() }
    })

    if (response?.data.code === 200 && response.data.data) {
      cronNextTimes.value = response.data.data
    } else {
      cronNextTimes.value = []
      if (response?.data.message) {
        ElMessage.warning(response.data.message)
      }
    }
  } catch (error) {
    console.error('获取Cron执行时间错误:', error)
    cronNextTimes.value = []
    ElMessage.error('获取Cron执行时间失败，请检查表达式格式')
  }
}

const startSync = async () => {
  try {
    syncStartLoading.value = true
    const response = await http?.post(`${SERVER_URL}/emby/sync/start`)

    if (response?.data.code === 200) {
      ElMessage.success('同步已启动')
      syncPolling.value = true
      await querySyncStatus()
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

const querySyncStatus = async () => {
  try {
    const response = await http?.get(`${SERVER_URL}/emby/sync/status`)

    if (response?.data.code === 200) {
      syncInfo.value = response.data.data
      syncPolling.value = response.data.data?.is_running
      if (syncPolling.value) {
      } else {
        stopSyncPolling()
      }
    }
  } catch (error) {
    console.error('查询同步状态错误:', error)
  }
}

const startSyncPolling = () => {
  syncPollTimer = window.setInterval(async () => {
    try {
      const response = await http?.get(`${SERVER_URL}/emby/sync/status`)

      if (response?.data.code === 200) {
        syncInfo.value = response.data.data
         syncPolling.value = response.data.data?.is_running
        console.log(syncPolling.value)
        if (!syncPolling.value) {
          stopSyncPolling()
        }
      }
    } catch (error) {
      console.error('轮询同步状态错误:', error)
    }
  }, 3000)
}

const stopSyncPolling = () => {
  syncPolling.value = false
  if (syncPollTimer !== null) {
    clearInterval(syncPollTimer)
    syncPollTimer = null
  }
}

const formatLastSyncTime = (timestamp: number | null | undefined) => {
  if (!timestamp) return '未同步'

  try {
    const timestampMs = timestamp < 10000000000 ? timestamp * 1000 : timestamp

    const date = new Date(timestampMs)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()

    if (diffMs < 0) return date.toLocaleString('zh-CN')

    const diffSecs = Math.floor(diffMs / 1000)
    const diffMins = Math.floor(diffSecs / 60)
    const diffHours = Math.floor(diffSecs / 3600)
    const diffDays = Math.floor(diffSecs / 86400)

    if (diffSecs < 60) return '刚刚'
    if (diffMins < 60) return `${diffMins}分钟前`
    if (diffHours < 24) return `${diffHours}小时前`
    if (diffDays < 30) return `${diffDays}天前`

    return date.toLocaleString('zh-CN')
  } catch {
    return String(timestamp)
  }
}

onMounted(() => {
  loadEmbyConfig()
  querySyncStatus()
  updateWebhookUrl()
})

onBeforeUnmount(() => {
  stopSyncPolling()
})
</script>

<style scoped lang="css">
.emby-content {
  /* padding: 20px; */
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e7ed 100%);
}

.emby-settings-wrapper {
  /* max-width: 1400px; */
  margin: 0 auto;
}

.emby-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
  margin-bottom: 24px;
}

.settings-card {
  border-radius: 12px;
  border: none;
  overflow: hidden;
  transition: all 0.3s ease;
  width: 100%;
}

.settings-card:hover {
  transform: translateY(-2px);
}

.card-header-wrapper {
  display: flex;
  align-items: center;
  gap: 16px;
}

.card-header-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.server-icon {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.webhook-icon {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.features-icon {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.sync-icon {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
}

.card-header-content {
  flex: 1;
}

.card-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.card-subtitle {
  margin: 4px 0 0;
  font-size: 13px;
  color: #909399;
}

.card-header-action {
  margin-left: auto;
}

.limited-width-input {
  max-width: 600px;
}

.webhook-input :deep(.el-input__wrapper) {
  background-color: #f5f7fa;
}

.emby-example-inline {
  margin-top: 10px;
  padding: 12px 16px;
  background: linear-gradient(135deg, #f5f7fa 0%, #e8eef5 100%);
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 12px;
  border: 1px solid #e4e7ed;
}

.example-label {
  font-weight: 500;
  color: #606266;
  font-size: 13px;
}

.example-url {
  color: #409eff;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  background: white;
  padding: 4px 8px;
  border-radius: 4px;
  border: 1px solid #d9ecff;
}

.form-help {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  font-size: 14px;
  color: #3c3d40;
  margin-top: 8px;
  line-height: 1.5;
}

.form-help .el-icon {
  margin-top: 2px;
  flex-shrink: 0;
}

.author-credit {
  margin-top: 4px;
}

.help-link {
  color: #409eff;
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s;
}

.help-link:hover {
  color: #66b1ff;
  text-decoration: underline;
}

.action-link {
  margin-left: 8px;
  padding: 2px 8px;
  background: #ecf5ff;
  border-radius: 4px;
  font-size: 12px;
}

.action-link:hover {
  background: #d9ecff;
}

.inline-code {
  background: #fef0f0;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 11px;
  color: #f56c6c;
}

.switch-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
}

.switch-label {
  font-size: 14px;
  color: #909399;
  font-weight: 500;
  transition: color 0.3s;
}

.switch-label.is-active {
  color: #67c23a;
}

.switch-label.is-danger {
  color: #f56c6c;
}

.warning-text {
  color: #e6a23c;
}

.feature-item {
  padding: 16px 0;
}

.feature-item:first-child {
  padding-top: 0;
}

.feature-item.is-disabled {
  opacity: 0.5;
  pointer-events: none;
}

.danger-item {
  background: linear-gradient(135deg, #fef6f6 0%, #fff 100%);
  margin: 0 -20px;
  padding: 20px;
  border-radius: 8px;
  border: 1px solid #fde2e2;
}

.danger-alert {
  margin-bottom: 12px;
  border-radius: 8px;
}

.feature-description {
  font-size: 13px;
  color: #606266;
  line-height: 1.8;
  margin-top: 8px;
}

.feature-description p {
  margin: 6px 0;
}

.feature-note {
  color: #909399;
  font-style: italic;
}

.cron-next-times {
  margin-top: 12px;
  padding: 12px 16px;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border-radius: 8px;
  border: 1px solid #bae6fd;
}

.cron-times-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 500;
  color: #0284c7;
  margin-bottom: 8px;
}

.cron-times-list {
  margin: 0;
  padding-left: 20px;
  list-style: decimal;
}

.cron-times-list li {
  font-size: 13px;
  color: #0369a1;
  line-height: 1.8;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.feature-divider {
  margin: 16px 0;
}

.config-links {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 8px;
}

.delete-rules {
  margin: 12px 0 0;
  padding-left: 20px;
  color: #909399;
}

.delete-rules li {
  margin: 6px 0;
  line-height: 1.6;
}

.form-actions-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding: 20px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  align-items: center;
}

.save-btn,
.extract-btn {
  min-width: 140px;
}

.extract-help {
  flex: 1;
  min-width: 200px;
}

.extract-help p {
  margin: 0;
  font-size: 12px;
  color: #909399;
  line-height: 1.6;
}

.sync-management-card {
  border-radius: 12px;
  border: none;
  margin-bottom: 20px;
  width: 100%;
}

.sync-info-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
}

.sync-stat-card {
  display: flex;
  align-items: center;
  width: 100%;
  gap: 16px;
  padding: 20px;
  background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
  border-radius: 12px;
  border: 1px solid #ebeef5;
  transition: all 0.3s ease;
}

.sync-stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.auto-sync-icon {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.cycle-icon {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.items-icon {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.time-icon {
  background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  margin-bottom: 4px;
}

.stat-value {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.stat-value.is-enabled {
  color: #67c23a;
}

.stat-value.highlight {
  color: #409eff;
}

.sync-progress {
  margin-top: 20px;
  padding: 16px;
  background: linear-gradient(135deg, #ecf5ff 0%, #f0f9ff 100%);
  border-radius: 8px;
  border: 1px solid #d9ecff;
}

.progress-indicator {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #409eff;
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

.sync-empty {
  padding: 20px;
}

.emby-status-alert {
  margin-bottom: 20px;
  border-radius: 8px;
}

.tips-alert {
  border-radius: 8px;
}

@media (max-width: 1200px) {
  .sync-info-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .emby-content {
    padding: 12px;
  }

  .emby-settings-wrapper {
    max-width: 100%;
  }

  .emby-form {
    gap: 16px;
  }

  .settings-card {
    border-radius: 8px;
  }

  .settings-card:hover {
    transform: none;
  }

  .card-header-wrapper {
    flex-wrap: wrap;
  }

  .card-header-icon {
    width: 40px;
    height: 40px;
  }

  .card-header-icon .el-icon {
    font-size: 20px;
  }

  .card-title {
    font-size: 16px;
  }

  .card-subtitle {
    font-size: 12px;
  }

  .card-header-action {
    width: 100%;
    margin-top: 12px;
  }

  .card-header-action .el-button {
    width: 100%;
  }

  .limited-width-input {
    max-width: 100%;
  }

  .emby-example-inline {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .form-help {
    flex-wrap: wrap;
  }

  .config-links {
    flex-direction: column;
    align-items: flex-start;
  }

  .config-links .help-link {
    margin-top: 4px;
  }

  .action-link {
    margin-left: 0;
    margin-top: 4px;
  }

  .form-actions-wrapper {
    flex-direction: column;
    align-items: stretch;
    padding: 16px;
    gap: 12px;
  }

  .save-btn,
  .extract-btn {
    width: 100%;
    min-width: auto;
  }

  .extract-help {
    min-width: auto;
  }

  .sync-management-card {
    border-radius: 8px;
  }

  .sync-info-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .sync-stat-card {
    padding: 14px;
    gap: 12px;
  }

  .sync-stat-card:hover {
    transform: none;
  }

  .stat-icon {
    width: 44px;
    height: 44px;
  }

  .stat-icon .el-icon {
    font-size: 22px;
  }

  .stat-label {
    font-size: 12px;
  }

  .stat-value {
    font-size: 15px;
  }

  .sync-progress {
    padding: 12px;
  }

  .feature-item {
    padding: 12px 0;
  }

  .danger-item {
    margin: 0 -12px;
    padding: 12px;
    border-radius: 0;
  }

  .danger-alert {
    font-size: 12px;
  }

  .delete-rules {
    font-size: 12px;
  }

  .delete-rules li {
    margin: 4px 0;
  }

  .emby-status-alert,
  .tips-alert {
    border-radius: 8px;
  }
}

@media (max-width: 480px) {
  .emby-content {
    padding: 8px;
  }

  .card-header-icon {
    width: 36px;
    height: 36px;
  }

  .card-header-icon .el-icon {
    font-size: 18px;
  }

  .card-title {
    font-size: 15px;
  }

  .switch-wrapper {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .switch-label {
    font-size: 13px;
  }

  .feature-description {
    font-size: 12px;
  }

  .sync-stat-card {
    padding: 12px;
  }

  .stat-icon {
    width: 40px;
    height: 40px;
  }

  .stat-icon .el-icon {
    font-size: 20px;
  }

  .stat-value {
    font-size: 14px;
  }
}
</style>
