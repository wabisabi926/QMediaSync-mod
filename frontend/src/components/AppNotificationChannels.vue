<template>
  <div>
    <el-alert type="info" :show-icon="true" style="margin-top: 12px">
      管理系统的通知渠道，支持 Telegram、MeoW、Bark、Server酱、Webhook 等多种推送方式
    </el-alert>
    <div class="main-content-container notification-channels-container">
      <!-- 操作按钮区 -->
      <div class="action-bar">
        <el-button type="primary" :icon="Plus" @click="showCreateDialog">
          添加通知渠道
        </el-button>
        <el-button :icon="Refresh" @click="loadChannels" :loading="loading">
          刷新
        </el-button>
      </div>

      <!-- 渠道列表 -->
      <el-table :data="channels" v-loading="loading" stripe style="width: 100%; margin-top: 16px">
        <el-table-column prop="channel_type" label="类型" width="120">
          <template #default="{ row }">
            <el-tag :type="getChannelTypeColor(row.channel_type)">
              {{ getChannelTypeName(row.channel_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="channel_name" label="名称" />
        <el-table-column prop="is_enabled" label="状态" width="100">
          <template #default="{ row }">
            <el-switch
              v-model="row.is_enabled"
              @change="toggleChannelStatus(row)"
              :loading="row._switching"
            />
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="380" fixed="right">
          <template #default="{ row }">
            <el-button
              size="small"
              :icon="Edit"
              @click="showEditDialog(row)"
              :loading="editLoading"
            >
              编辑
            </el-button>
            <el-button
              size="small"
              :icon="Setting"
              @click="showRulesDialog(row)"
            >
              规则
            </el-button>
            <el-button
              size="small"
              type="success"
              :icon="Message"
              @click="testChannel(row)"
              :loading="row._testing"
            >
              测试
            </el-button>
            <el-button
              size="small"
              type="danger"
              :icon="Delete"
              @click="deleteChannel(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建渠道对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      title="添加通知渠道"
      width="600px"
      :close-on-click-modal="false"
    >
      <!-- 渠道类型选择 -->
      <el-form v-if="!selectedChannelType" label-width="0">
        <el-form-item label="">
          <!-- 没有可用渠道类型时的提示 -->
          <el-empty
            v-if="channelTypes.length === 0"
            description="所有渠道类型都已添加"
            :image-size="80"
          >
            <template #description>
              <span>所有渠道类型都已添加</span>
              <br>
              <span style="font-size: 12px; color: var(--el-text-color-secondary);">
                每种渠道类型只能添加一个，如需修改请先删除现有渠道
              </span>
            </template>
          </el-empty>

          <!-- 渠道类型卡片 -->
          <div v-else class="channel-type-selector">
            <el-card
              v-for="type in channelTypes"
              :key="type.value"
              class="channel-type-card"
              shadow="hover"
              @click="selectedChannelType = type.value"
            >
              <div class="channel-type-content">
                <div class="channel-type-icon">{{ type.icon }}</div>
                <div class="channel-type-name">{{ type.label }}</div>
                <div class="channel-type-desc">{{ type.description }}</div>
              </div>
            </el-card>
          </div>
        </el-form-item>
      </el-form>

      <!-- 渠道配置表单 -->
      <el-form
        v-else
        :model="channelForm"
        :label-position="checkIsMobile ? 'top' : 'left'"
        label-width="120px"
        ref="channelFormRef"
      >
        <el-form-item>
          <el-button size="small" @click="selectedChannelType = ''" :icon="Back">
            重新选择类型
          </el-button>
        </el-form-item>

        <el-form-item label="渠道名称" required>
          <el-input
            v-model="channelForm.channel_name"
            placeholder="请输入渠道显示名称"
          />
        </el-form-item>

        <!-- Telegram 配置 -->
        <template v-if="selectedChannelType === 'telegram'">
          <el-form-item label="Bot Token" required>
            <el-input
              v-model="channelForm.bot_token"
              placeholder="123456:ABC-DEF..."
            />
          </el-form-item>
          <el-form-item label="Chat ID" required>
            <el-input
              v-model="channelForm.chat_id"
              placeholder="123456789"
            />
          </el-form-item>
        </template>

        <!-- MeoW 配置 -->
        <template v-if="selectedChannelType === 'meow'">
          <el-form-item label="昵称" required>
            <el-input
              v-model="channelForm.nickname"
              placeholder="my_nickname"
            />
          </el-form-item>
          <el-form-item label="API地址">
            <el-input
              v-model="channelForm.endpoint"
              placeholder="http://api.chuckfang.com"
            />
          </el-form-item>
        </template>

        <!-- Bark 配置 -->
        <template v-if="selectedChannelType === 'bark'">
          <el-form-item label="设备密钥" required>
            <el-input
              v-model="channelForm.device_key"
              placeholder="your_device_key_here"
            />
          </el-form-item>
          <el-form-item label="服务器地址">
            <el-input
              v-model="channelForm.server_url"
              placeholder="https://api.day.app"
            />
          </el-form-item>
          <el-form-item label="通知声音">
            <el-input
              v-model="channelForm.sound"
              placeholder="alert"
            />
          </el-form-item>
          <el-form-item label="通知图标">
            <el-input
              v-model="channelForm.icon"
              placeholder="https://example.com/icon.png"
            />
          </el-form-item>
        </template>

        <!-- Server酱 配置 -->
        <template v-if="selectedChannelType === 'serverchan'">
          <el-form-item label="SCKEY" required>
            <el-input
              v-model="channelForm.sc_key"
              placeholder="SCU1234567890abcdef"
            />
          </el-form-item>
          <el-form-item label="API地址">
            <el-input
              v-model="channelForm.endpoint"
              placeholder="https://sc.ftqq.com"
            />
          </el-form-item>
        </template>

        <!-- Webhook 配置 -->
        <template v-if="selectedChannelType === 'webhook'">
          <el-form-item label="请求地址" required>
            <el-input
              v-model="channelForm.endpoint"
              placeholder="https://example.com/webhook"
            />
          </el-form-item>
          <el-form-item label="请求方法" required>
            <el-select v-model="channelForm.method" placeholder="选择请求方法" style="width: 100%">
              <el-option label="GET" value="GET" />
              <el-option label="POST" value="POST" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="channelForm.method === 'POST'" label="数据格式" required>
            <el-select v-model="channelForm.format" placeholder="选择数据格式" style="width: 100%">
              <el-option label="JSON" value="json" />
              <el-option label="Form" value="form" />
              <el-option label="Text" value="text" />
            </el-select>
          </el-form-item>
          <el-form-item label="消息模板" required>
            <el-input
              v-model="channelForm.template"
              type="textarea"
              :rows="6"
              placeholder='支持变量: &#123;&#123;title&#125;&#125;, &#123;&#123;content&#125;&#125;, &#123;&#123;timestamp&#125;&#125;, &#123;&#123;image&#125;&#125;'
            />
            <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 4px;">
              支持的变量:<br />
              &#123;&#123;title&#125;&#125; - 通知标题<br />
              &#123;&#123;content&#125;&#125; - 通知内容<br />
              &#123;&#123;timestamp&#125;&#125; - 时间戳<br />
              &#123;&#123;image&#125;&#125; - 图片URL（如果有） <br />
              POST JSON示例: {"title":"&#123;&#123;title&#125;&#125;","content":"&#123;&#123;content&#125;&#125;"}
              <br>
              POST Form示例: title=&#123;&#123;title&#125;&#125;&content=&#123;&#123;content&#125;&#125;
              <br>
              GET/Text示例: 【&#123;&#123;title&#125;&#125;】&#123;&#123;content&#125;&#125;
            </div>
          </el-form-item>
          <el-form-item v-if="channelForm.method === 'GET'" label="查询参数名">
            <el-input
              v-model="channelForm.query_param"
              placeholder="默认: q"
            />
          </el-form-item>
          <el-form-item label="鉴权类型">
            <el-select v-model="channelForm.auth_type" placeholder="选择鉴权方式" style="width: 100%">
              <el-option label="无鉴权" value="none" />
              <el-option label="Bearer Token" value="bearer" />
              <el-option label="Basic Auth" value="basic" />
              <el-option label="自定义Header" value="header" />
              <el-option label="Query参数" value="query" />
            </el-select>
          </el-form-item>
          <template v-if="channelForm.auth_type === 'bearer' || channelForm.auth_type === 'query'">
            <el-form-item :label="channelForm.auth_type === 'bearer' ? 'Token' : '参数值'">
              <el-input
                v-model="channelForm.auth_token"
                placeholder="输入token或参数值"
              />
            </el-form-item>
            <el-form-item v-if="channelForm.auth_type === 'query'" label="参数名">
              <el-input
                v-model="channelForm.auth_query_key"
                placeholder="例如: token"
              />
            </el-form-item>
          </template>
          <template v-if="channelForm.auth_type === 'basic'">
            <el-form-item label="用户名">
              <el-input
                v-model="channelForm.auth_user"
                placeholder="Basic Auth用户名"
              />
            </el-form-item>
            <el-form-item label="密码">
              <el-input
                v-model="channelForm.auth_pass"
                type="password"
                placeholder="Basic Auth密码"
                show-password
              />
            </el-form-item>
          </template>
          <template v-if="channelForm.auth_type === 'header'">
            <el-form-item label="Header名称">
              <el-input
                v-model="channelForm.auth_header_key"
                placeholder="例如: X-Api-Key"
              />
            </el-form-item>
            <el-form-item label="Header值">
              <el-input
                v-model="channelForm.auth_token"
                placeholder="输入Header值"
              />
            </el-form-item>
          </template>
          <el-form-item label="备注说明">
            <el-input
              v-model="channelForm.description"
              type="textarea"
              :rows="2"
              placeholder="可选的备注信息"
            />
          </el-form-item>
        </template>
      </el-form>

      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button
          v-if="selectedChannelType"
          type="primary"
          @click="createChannel"
          :loading="creating"
        >
          创建
        </el-button>
      </template>
    </el-dialog>

    <!-- 通知规则对话框 -->
    <el-dialog
      v-model="rulesDialogVisible"
      :title="`${currentChannel?.channel_name} - 通知规则`"
      width="600px"
    >
      <el-table :data="currentRules" v-loading="rulesLoading">
        <el-table-column prop="event_type" label="事件类型" width="180">
          <template #default="{ row }">
            {{ getEventTypeName(row.event_type) }}
          </template>
        </el-table-column>
        <el-table-column prop="is_enabled" label="启用状态">
          <template #default="{ row }">
            <el-switch
              v-model="row.is_enabled"
              @change="updateRule(row)"
              :loading="row._updating"
            />
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <el-button @click="rulesDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 编辑渠道对话框 -->
    <el-dialog
      v-model="editDialogVisible"
      :title="`编辑渠道 - ${editingChannel?.channel_name}`"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form
        :model="channelForm"
        :label-position="checkIsMobile ? 'top' : 'left'"
        label-width="120px"
        ref="channelFormRef"
      >
        <el-form-item label="渠道名称" required>
          <el-input
            v-model="channelForm.channel_name"
            placeholder="请输入渠道显示名称"
          />
        </el-form-item>

        <!-- Telegram 编辑 -->
        <template v-if="editingChannel?.channel_type === 'telegram'">
          <el-form-item label="Bot Token">
            <el-input
              v-model="channelForm.bot_token"
              placeholder="123456:ABC-DEF..."
            />
          </el-form-item>
          <el-form-item label="Chat ID">
            <el-input
              v-model="channelForm.chat_id"
              placeholder="123456789"
            />
          </el-form-item>
        </template>

        <!-- MeoW 编辑 -->
        <template v-if="editingChannel?.channel_type === 'meow'">
          <el-form-item label="昵称">
            <el-input
              v-model="channelForm.nickname"
              placeholder="my_nickname"
            />
          </el-form-item>
          <el-form-item label="API地址">
            <el-input
              v-model="channelForm.endpoint"
              placeholder="http://api.chuckfang.com"
            />
          </el-form-item>
        </template>

        <!-- Bark 编辑 -->
        <template v-if="editingChannel?.channel_type === 'bark'">
          <el-form-item label="设备密钥">
            <el-input
              v-model="channelForm.device_key"
              placeholder="your_device_key_here"
            />
          </el-form-item>
          <el-form-item label="服务器地址">
            <el-input
              v-model="channelForm.server_url"
              placeholder="https://api.day.app"
            />
          </el-form-item>
          <el-form-item label="通知声音">
            <el-input
              v-model="channelForm.sound"
              placeholder="alert"
            />
          </el-form-item>
          <el-form-item label="通知图标">
            <el-input
              v-model="channelForm.icon"
              placeholder="https://example.com/icon.png"
            />
          </el-form-item>
        </template>

        <!-- Server酱 编辑 -->
        <template v-if="editingChannel?.channel_type === 'serverchan'">
          <el-form-item label="SCKEY">
            <el-input
              v-model="channelForm.sc_key"
              placeholder="SCU1234567890abcdef"
            />
          </el-form-item>
          <el-form-item label="API地址">
            <el-input
              v-model="channelForm.endpoint"
              placeholder="https://sc.ftqq.com"
            />
          </el-form-item>
        </template>

        <!-- Webhook 编辑 -->
        <template v-if="editingChannel?.channel_type === 'webhook'">
          <el-form-item label="请求地址">
            <el-input
              v-model="channelForm.endpoint"
              placeholder="https://example.com/webhook"
            />
          </el-form-item>
          <el-form-item label="请求方法">
            <el-select v-model="channelForm.method" placeholder="选择请求方法" style="width: 100%">
              <el-option label="GET" value="GET" />
              <el-option label="POST" value="POST" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="channelForm.method === 'POST'" label="数据格式">
            <el-select v-model="channelForm.format" placeholder="选择数据格式" style="width: 100%">
              <el-option label="JSON" value="json" />
              <el-option label="Form" value="form" />
              <el-option label="Text" value="text" />
            </el-select>
          </el-form-item>
          <el-form-item label="消息模板">
            <el-input
              v-model="channelForm.template"
              type="textarea"
              :rows="6"
              placeholder='支持变量: &#123;&#123;title&#125;&#125;, &#123;&#123;content&#125;&#125;, &#123;&#123;timestamp&#125;&#125;, &#123;&#123;image&#125;&#125;'
            />
          </el-form-item>
          <el-form-item v-if="channelForm.method === 'GET'" label="查询参数名">
            <el-input
              v-model="channelForm.query_param"
              placeholder="默认: q"
            />
          </el-form-item>
          <el-form-item label="鉴权类型">
            <el-select v-model="channelForm.auth_type" placeholder="选择鉴权方式" style="width: 100%">
              <el-option label="无鉴权" value="none" />
              <el-option label="Bearer Token" value="bearer" />
              <el-option label="Basic Auth" value="basic" />
              <el-option label="自定义Header" value="header" />
              <el-option label="Query参数" value="query" />
            </el-select>
          </el-form-item>
          <template v-if="channelForm.auth_type === 'bearer' || channelForm.auth_type === 'query'">
            <el-form-item :label="channelForm.auth_type === 'bearer' ? 'Token' : '参数值'">
              <el-input
                v-model="channelForm.auth_token"
                placeholder="输入token或参数值"
              />
            </el-form-item>
            <el-form-item v-if="channelForm.auth_type === 'query'" label="参数名">
              <el-input
                v-model="channelForm.auth_query_key"
                placeholder="例如: token"
              />
            </el-form-item>
          </template>
          <template v-if="channelForm.auth_type === 'basic'">
            <el-form-item label="用户名">
              <el-input
                v-model="channelForm.auth_user"
                placeholder="Basic Auth用户名"
              />
            </el-form-item>
            <el-form-item label="密码">
              <el-input
                v-model="channelForm.auth_pass"
                type="password"
                placeholder="Basic Auth密码"
                show-password
              />
            </el-form-item>
          </template>
          <template v-if="channelForm.auth_type === 'header'">
            <el-form-item label="Header名称">
              <el-input
                v-model="channelForm.auth_header_key"
                placeholder="例如: X-Api-Key"
              />
            </el-form-item>
            <el-form-item label="Header值">
              <el-input
                v-model="channelForm.auth_token"
                placeholder="输入Header值"
              />
            </el-form-item>
          </template>
          <el-form-item label="备注说明">
            <el-input
              v-model="channelForm.description"
              type="textarea"
              :rows="2"
              placeholder="可选的备注信息"
            />
          </el-form-item>
        </template>
      </el-form>

      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          @click="updateChannel"
          :loading="updating"
        >
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted, inject, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Refresh,
  Setting,
  Message,
  Delete,
  Back,
  Edit
} from '@element-plus/icons-vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { isMobile } from '@/utils/deviceUtils'
import { formatDateTime } from '@/utils/timeUtils'
import {
  getChannelTypeName,
  getChannelTypeColor,
  getEventTypeName,
  type NotificationChannel,
  type NotificationRule,
  type ChannelType
} from '@/utils/notificationUtils'

// 渠道表单接口
interface ChannelFormData {
  channel_name: string
  // Telegram
  bot_token: string
  chat_id: string
  // MeoW
  nickname: string
  // 通用
  endpoint: string
  // Bark
  device_key: string
  server_url: string
  sound: string
  icon: string
  // Server酱
  sc_key: string
  // Webhook
  method: string
  format: string
  template: string
  query_param: string
  auth_type: string
  auth_token: string
  auth_user: string
  auth_pass: string
  auth_header_key: string
  auth_query_key: string
  description: string
}

// 渠道状态扩展接口
interface ChannelWithStatus extends NotificationChannel {
  _switching: boolean
  _testing: boolean
}

// 规则状态扩展接口
interface RuleWithStatus extends NotificationRule {
  _updating: boolean
}

const checkIsMobile = ref(isMobile())
const http: AxiosStatic | undefined = inject('$http')

const loading = ref(false)
const creating = ref(false)
const updating = ref(false)
const editLoading = ref(false)
const channels = ref<NotificationChannel[]>([])
const createDialogVisible = ref(false)
const editDialogVisible = ref(false)
const rulesDialogVisible = ref(false)
const selectedChannelType = ref<ChannelType | ''>('')
const editingChannel = ref<NotificationChannel | null>(null)
const currentChannel = ref<NotificationChannel | null>(null)
const currentRules = ref<NotificationRule[]>([])
const rulesLoading = ref(false)
const channelFormRef = ref()

// 所有渠道类型选项
const allChannelTypes = [
  {
    value: 'telegram' as ChannelType,
    label: 'Telegram',
    icon: '✈️',
    description: 'Telegram Bot 推送'
  },
  {
    value: 'meow' as ChannelType,
    label: 'MeoW',
    icon: '🐱',
    description: 'MeoW 推送服务'
  },
  {
    value: 'bark' as ChannelType,
    label: 'Bark',
    icon: '🍎',
    description: 'iOS Bark 推送'
  },
  {
    value: 'serverchan' as ChannelType,
    label: 'Server酱',
    icon: '💬',
    description: '微信推送服务'
  },
  {
    value: 'webhook' as ChannelType,
    label: 'Webhook',
    icon: '🔗',
    description: '自定义 Webhook 推送'
  }
]

// 可用的渠道类型选项（过滤掉已存在的）
const channelTypes = computed(() => {
  const existingTypes = channels.value.map(channel => channel.channel_type)
  return allChannelTypes.filter(type => !existingTypes.includes(type.value))
})

// 渠道表单
const channelForm = reactive<ChannelFormData>({
  channel_name: '',
  bot_token: '',
  chat_id: '',
  nickname: '',
  endpoint: '',
  device_key: '',
  server_url: '',
  sound: '',
  icon: '',
  sc_key: '',
  // Webhook字段
  method: 'POST',
  format: 'json',
  template: '',
  query_param: 'q',
  auth_type: 'none',
  auth_token: '',
  auth_user: '',
  auth_pass: '',
  auth_header_key: '',
  auth_query_key: '',
  description: ''
})

// 加载渠道列表
const loadChannels = async () => {
  loading.value = true
  try {
    const response = await http?.get(`${SERVER_URL}/setting/notification/channels`)
    if (response?.data.code === 0) {
      channels.value = response.data.data.map((channel: NotificationChannel): ChannelWithStatus => ({
        ...channel,
        _switching: false,
        _testing: false
      }))
    } else {
      ElMessage.error(response?.data.message || '加载失败')
    }
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : '加载渠道列表失败'
    ElMessage.error(errorMessage)
  } finally {
    loading.value = false
  }
}

// 显示创建对话框
const showCreateDialog = () => {
  // 检查是否还有可用的渠道类型
  const existingTypes = channels.value.map(channel => channel.channel_type)
  const availableTypes = allChannelTypes.filter(type => !existingTypes.includes(type.value))

  if (availableTypes.length === 0) {
    ElMessage.warning('所有渠道类型都已添加，每种类型只能添加一个渠道')
    return
  }

  selectedChannelType.value = ''
  resetChannelForm()
  createDialogVisible.value = true
}

// 重置表单
const resetChannelForm = () => {
  channelForm.channel_name = ''
  channelForm.bot_token = ''
  channelForm.chat_id = ''
  channelForm.nickname = ''
  channelForm.endpoint = ''
  channelForm.device_key = ''
  channelForm.server_url = ''
  channelForm.sound = ''
  channelForm.icon = ''
  channelForm.sc_key = ''
  channelForm.method = 'POST'
  channelForm.format = 'json'
  channelForm.template = ''
  channelForm.query_param = 'q'
  channelForm.auth_type = 'none'
  channelForm.auth_token = ''
  channelForm.auth_user = ''
  channelForm.auth_pass = ''
  channelForm.auth_header_key = ''
  channelForm.auth_query_key = ''
  channelForm.description = ''
}

// 显示编辑对话框
const showEditDialog = async (channel: NotificationChannel) => {
  editingChannel.value = channel
  editLoading.value = true

  try {
    // 根据渠道类型调用对应的查询接口获取详细配置
    const response = await http?.get(
      `${SERVER_URL}/setting/notification/channels/${channel.channel_type}/${channel.id}`
    )

    if (response?.data.code === 0) {
      const { channel: channelData, config } = response.data.data

      // 填充基本信息
      channelForm.channel_name = channelData.channel_name || ''
      channelForm.description = channelData.description || ''

      if (config) {
        // Telegram
        if (channel.channel_type === 'telegram') {
          channelForm.bot_token = config.bot_token || ''
          channelForm.chat_id = config.chat_id || ''
        }
        // MeoW
        else if (channel.channel_type === 'meow') {
          channelForm.nickname = config.nickname || ''
          channelForm.endpoint = config.endpoint || ''
        }
        // Bark
        else if (channel.channel_type === 'bark') {
          channelForm.device_key = config.device_key || ''
          channelForm.server_url = config.server_url || ''
          channelForm.sound = config.sound || ''
          channelForm.icon = config.icon || ''
        }
        // Server酱
        else if (channel.channel_type === 'serverchan') {
          channelForm.sc_key = config.sc_key || ''
          channelForm.endpoint = config.endpoint || ''
        }
        // Webhook
        else if (channel.channel_type === 'webhook') {
          channelForm.endpoint = config.endpoint || ''
          channelForm.method = config.method || 'POST'
          channelForm.format = config.format || 'json'
          channelForm.template = config.template || ''
          channelForm.query_param = config.query_param || 'q'
          channelForm.auth_type = config.auth_type || 'none'
          channelForm.auth_token = config.auth_token || ''
          channelForm.auth_user = config.auth_user || ''
          channelForm.auth_pass = config.auth_pass || ''
          channelForm.auth_header_key = config.auth_header_key || ''
          channelForm.auth_query_key = config.auth_query_key || ''
        }
      }

      editDialogVisible.value = true
    } else {
      ElMessage.error(response?.data.message || '获取渠道配置失败')
    }
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : '获取渠道配置失败'
    ElMessage.error(errorMessage)
  } finally {
    editLoading.value = false
  }
}

// 创建渠道
const createChannel = async () => {
  if (!channelForm.channel_name) {
    ElMessage.warning('请输入渠道名称')
    return
  }

  // 根据类型验证必填字段
  if (selectedChannelType.value === 'telegram') {
    if (!channelForm.bot_token || !channelForm.chat_id) {
      ElMessage.warning('请填写Bot Token和Chat ID')
      return
    }
  } else if (selectedChannelType.value === 'meow') {
    if (!channelForm.nickname) {
      ElMessage.warning('请填写昵称')
      return
    }
  } else if (selectedChannelType.value === 'bark') {
    if (!channelForm.device_key) {
      ElMessage.warning('请填写设备密钥')
      return
    }
  } else if (selectedChannelType.value === 'serverchan') {
    if (!channelForm.sc_key) {
      ElMessage.warning('请填写SCKEY')
      return
    }
  } else if (selectedChannelType.value === 'webhook') {
    if (!channelForm.endpoint || !channelForm.method || !channelForm.template) {
      ElMessage.warning('请填写请求地址、请求方法和消息模板')
      return
    }
    if (channelForm.method === 'POST' && !channelForm.format) {
      ElMessage.warning('请选择POST数据格式')
      return
    }
  }

  creating.value = true
  try {
    const requestData: Record<string, unknown> = {
      channel_name: channelForm.channel_name
    }

    // 根据类型添加配置字段
    if (selectedChannelType.value === 'telegram') {
      requestData.bot_token = channelForm.bot_token
      requestData.chat_id = channelForm.chat_id
    } else if (selectedChannelType.value === 'meow') {
      requestData.nickname = channelForm.nickname
      if (channelForm.endpoint) {
        requestData.endpoint = channelForm.endpoint
      }
    } else if (selectedChannelType.value === 'bark') {
      requestData.device_key = channelForm.device_key
      if (channelForm.server_url) {
        requestData.server_url = channelForm.server_url
      }
      if (channelForm.sound) {
        requestData.sound = channelForm.sound
      }
      if (channelForm.icon) {
        requestData.icon = channelForm.icon
      }
    } else if (selectedChannelType.value === 'serverchan') {
      requestData.sc_key = channelForm.sc_key
      if (channelForm.endpoint) {
        requestData.endpoint = channelForm.endpoint
      }
    } else if (selectedChannelType.value === 'webhook') {
      requestData.endpoint = channelForm.endpoint
      requestData.method = channelForm.method
      requestData.template = channelForm.template
      if (channelForm.method === 'POST') {
        requestData.format = channelForm.format
      }
      if (channelForm.method === 'GET' && channelForm.query_param) {
        requestData.query_param = channelForm.query_param
      }
      if (channelForm.auth_type && channelForm.auth_type !== 'none') {
        requestData.auth_type = channelForm.auth_type
        if (channelForm.auth_type === 'bearer' || channelForm.auth_type === 'query') {
          requestData.auth_token = channelForm.auth_token
          if (channelForm.auth_type === 'query' && channelForm.auth_query_key) {
            requestData.auth_query_key = channelForm.auth_query_key
          }
        } else if (channelForm.auth_type === 'basic') {
          requestData.auth_user = channelForm.auth_user
          requestData.auth_pass = channelForm.auth_pass
        } else if (channelForm.auth_type === 'header') {
          requestData.auth_header_key = channelForm.auth_header_key
          requestData.auth_token = channelForm.auth_token
        }
      }
      if (channelForm.description) {
        requestData.description = channelForm.description
      }
    }

    const response = await http?.post(
      `${SERVER_URL}/setting/notification/channels/${selectedChannelType.value}`,
      requestData
    )

    if (response?.data.code === 0) {
      ElMessage.success('创建成功')
      createDialogVisible.value = false
      loadChannels()
    } else {
      ElMessage.error(response?.data.message || '创建失败')
    }
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : '创建渠道失败'
    ElMessage.error(errorMessage)
  } finally {
    creating.value = false
  }
}

// 切换渠道状态
const toggleChannelStatus = async (channel: ChannelWithStatus) => {
  channel._switching = true
  try {
    const response = await http?.post(
      `${SERVER_URL}/setting/notification/channels/status`,
      {
        channel_id: channel.id,
        is_enabled: channel.is_enabled
      }
    )

    if (response?.data.code === 0) {
      ElMessage.success(channel.is_enabled ? '已启用' : '已禁用')
    } else {
      // 恢复原状态
      channel.is_enabled = !channel.is_enabled
      ElMessage.error(response?.data.message || '操作失败')
    }
  } catch (error: unknown) {
    // 恢复原状态
    channel.is_enabled = !channel.is_enabled
    const errorMessage = error instanceof Error ? error.message : '切换状态失败'
    ElMessage.error(errorMessage)
  } finally {
    channel._switching = false
  }
}

// 更新渠道
const updateChannel = async () => {
  if (!editingChannel.value || !channelForm.channel_name) {
    ElMessage.warning('请输入渠道名称')
    return
  }

  updating.value = true
  try {
    const requestData: Record<string, unknown> = {
      channel_id: editingChannel.value.id,
      channel_name: channelForm.channel_name
    }

    const channelType = editingChannel.value.channel_type

    // 根据类型添加配置字段
    if (channelType === 'telegram') {
      if (channelForm.bot_token) requestData.bot_token = channelForm.bot_token
      if (channelForm.chat_id) requestData.chat_id = channelForm.chat_id
    } else if (channelType === 'meow') {
      if (channelForm.nickname) requestData.nickname = channelForm.nickname
      if (channelForm.endpoint) requestData.endpoint = channelForm.endpoint
    } else if (channelType === 'bark') {
      if (channelForm.device_key) requestData.device_key = channelForm.device_key
      if (channelForm.server_url) requestData.server_url = channelForm.server_url
      if (channelForm.sound) requestData.sound = channelForm.sound
      if (channelForm.icon) requestData.icon = channelForm.icon
    } else if (channelType === 'serverchan') {
      if (channelForm.sc_key) requestData.sc_key = channelForm.sc_key
      if (channelForm.endpoint) requestData.endpoint = channelForm.endpoint
    } else if (channelType === 'webhook') {
      if (channelForm.endpoint) requestData.endpoint = channelForm.endpoint
      if (channelForm.method) requestData.method = channelForm.method
      if (channelForm.template) requestData.template = channelForm.template
      if (channelForm.method === 'POST' && channelForm.format) {
        requestData.format = channelForm.format
      }
      if (channelForm.method === 'GET' && channelForm.query_param) {
        requestData.query_param = channelForm.query_param
      }
      if (channelForm.auth_type) {
        requestData.auth_type = channelForm.auth_type
        if (channelForm.auth_type === 'bearer' || channelForm.auth_type === 'query') {
          if (channelForm.auth_token) requestData.auth_token = channelForm.auth_token
          if (channelForm.auth_type === 'query' && channelForm.auth_query_key) {
            requestData.auth_query_key = channelForm.auth_query_key
          }
        } else if (channelForm.auth_type === 'basic') {
          if (channelForm.auth_user) requestData.auth_user = channelForm.auth_user
          if (channelForm.auth_pass) requestData.auth_pass = channelForm.auth_pass
        } else if (channelForm.auth_type === 'header') {
          if (channelForm.auth_header_key) requestData.auth_header_key = channelForm.auth_header_key
          if (channelForm.auth_token) requestData.auth_token = channelForm.auth_token
        }
      }
      if (channelForm.description) requestData.description = channelForm.description
    }

    const response = await http?.put(
      `${SERVER_URL}/setting/notification/channels/${channelType}`,
      requestData
    )

    if (response?.data.code === 0) {
      ElMessage.success('更新成功')
      editDialogVisible.value = false
      loadChannels()
    } else {
      ElMessage.error(response?.data.message || '更新失败')
    }
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : '更新渠道失败'
    ElMessage.error(errorMessage)
  } finally {
    updating.value = false
  }
}

// 测试渠道
const testChannel = async (channel: ChannelWithStatus) => {
  channel._testing = true
  try {
    const response = await http?.post(
      `${SERVER_URL}/setting/notification/channels/test`,
      {
        channel_id: channel.id
      }
    )

    if (response?.data.code === 0) {
      ElMessage.success('测试消息已发送，请检查您的设备')
    } else {
      ElMessage.error(response?.data.message || '测试失败')
    }
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : '测试连接失败'
    ElMessage.error(errorMessage)
  } finally {
    channel._testing = false
  }
}

// 删除渠道
const deleteChannel = async (channel: NotificationChannel) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除渠道"${channel.channel_name}"吗？此操作将同时删除所有相关配置和规则。`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    const response = await http?.delete(
      `${SERVER_URL}/setting/notification/channels/${channel.id}`
    )

    if (response?.data.code === 0) {
      ElMessage.success('删除成功')
      loadChannels()
    } else {
      ElMessage.error(response?.data.message || '删除失败')
    }
  } catch (error: unknown) {
    if (error !== 'cancel') {
      const errorMessage = error instanceof Error ? error.message : '删除渠道失败'
      ElMessage.error(errorMessage)
    }
  }
}

// 显示规则对话框
const showRulesDialog = async (channel: NotificationChannel) => {
  currentChannel.value = channel
  rulesDialogVisible.value = true
  await loadRules(channel.id)
}

// 加载规则
const loadRules = async (channelId: number) => {
  rulesLoading.value = true
  try {
    const response = await http?.get(
      `${SERVER_URL}/setting/notification/rules?channel_id=${channelId}`
    )

    if (response?.data.code === 0) {
      currentRules.value = response.data.data.map((rule: NotificationRule): RuleWithStatus => ({
        ...rule,
        _updating: false
      }))
    } else {
      ElMessage.error(response?.data.message || '加载规则失败')
    }
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : '加载通知规则失败'
    ElMessage.error(errorMessage)
  } finally {
    rulesLoading.value = false
  }
}

// 更新规则
const updateRule = async (rule: RuleWithStatus) => {
  rule._updating = true
  try {
    const response = await http?.put(
      `${SERVER_URL}/setting/notification/rules`,
      {
        channel_id: rule.channel_id,
        event_type: rule.event_type,
        is_enabled: rule.is_enabled
      }
    )

    if (response?.data.code === 0) {
      ElMessage.success('更新成功')
    } else {
      // 恢复原状态
      rule.is_enabled = !rule.is_enabled
      ElMessage.error(response?.data.message || '更新失败')
    }
  } catch (error: unknown) {
    // 恢复原状态
    rule.is_enabled = !rule.is_enabled
    const errorMessage = error instanceof Error ? error.message : '更新规则失败'
    ElMessage.error(errorMessage)
  } finally {
    rule._updating = false
  }
}

onMounted(() => {
  loadChannels()
})
</script>

<style scoped>
.notification-channels-container {
  padding: 16px;
}

.action-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.channel-type-selector {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  max-width: 100%;
}

.channel-type-card {
  width: 100%;
  cursor: pointer;
  transition: all 0.3s;
  height: 140px;
  display: flex;
  align-items: center;
}

.channel-type-card:hover {
  transform: translateY(-4px);
  border-color: var(--el-color-primary);
}

.channel-type-content {
  text-align: center;
  padding: 16px;
  width: 100%;
}

.channel-type-icon {
  font-size: 36px;
  margin-bottom: 8px;
}

.channel-type-name {
  font-size: 16px;
  font-weight: bold;
  margin-bottom: 6px;
}

.channel-type-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}

/* 响应式适配 */
@media (max-width: 768px) {
  .channel-type-selector {
    grid-template-columns: 1fr;
  }

  .channel-type-card {
    height: 120px;
  }

  .channel-type-icon {
    font-size: 32px;
  }

  .channel-type-name {
    font-size: 15px;
  }

  .channel-type-desc {
    font-size: 11px;
  }
}
</style>
