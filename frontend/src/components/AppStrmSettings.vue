<template>
  <!-- STRM设置卡片 -->
  <div class="main-content-container strm-content">
    <el-form :model="strmData" :rules="formRules" :label-position="checkIsMobile ? 'top' : 'left'" :label-width="180"
      class="strm-form" ref="formRef">
      <!-- 排除的名称 -->
      <el-form-item label="排除的名称" prop="exclude_names">
        <MetadataExtInput v-model="strmData.exclude_name" placeholder="输入名称后按回车添加"
          class="meta-ext-input limited-width-input" :autoAddDot="false" />
        <div class="form-help">
          <p>指定需要排除的文件名或目录名，完整匹配不支持正则表达式。</p>
          <p>被排除的文件或目录将不会同步，其下的所有内容也都不会同步</p>
        </div>
      </el-form-item>
      <!-- 视频文件扩展名 -->
      <el-form-item label="视频文件扩展名" prop="video_ext">
        <MetadataExtInput v-model="strmData.video_ext" placeholder="输入扩展名后按回车添加,逗号或者分行分隔"
          class="meta-ext-input limited-width-input" />
        <div class="form-help">
          <p>指定需要生成STRM文件的视频文件扩展名，如：.mp4, .mkv, .avi, .mov 等</p>
        </div>
      </el-form-item>

      <!-- 最小文件大小 -->
      <el-form-item label="最小文件大小 (MB)" prop="min_file_size">
        <el-input-number v-model="strmData.min_file_size" :min="0" :step="1" :precision="0" placeholder="输入最小文件大小"
          :disabled="strmLoading" class="limited-width-input" />
        <div class="form-help">
          <p>小于此大小的视频文件将不会生成STRM文件，单位为MB。设置为0表示不限制文件大小</p>
        </div>
      </el-form-item>

      <!-- 元数据扩展名 -->
      <el-form-item label="元数据扩展名" prop="meta_ext">
        <MetadataExtInput v-model="strmData.meta_ext" placeholder="输入扩展名后按回车添加，逗号或者分行分隔"
          class="meta-ext-input limited-width-input" />
        <div class="form-help">
          <p>指定需要处理的元数据文件扩展名，如：.jpg, .nfo, .srt, .ass 等</p>
        </div>
      </el-form-item>

      <!-- 定时同步表达式 -->
      <el-form-item label="定时同步表达式" prop="cron_expression">
        <el-input v-model="strmData.cron_expression" placeholder="输入Cron表达式，如：0 2 * * *" :disabled="strmLoading"
          class="limited-width-input" @blur="loadCronTimes" />
        <div class="form-help">
          <p><strong>常用示例：</strong></p>
          <ul class="cron-examples">
            <li><code>0 0 * * *</code> - 每天0点执行</li>
            <li><code>0 */6 * * *</code> - 每6小时执行一次</li>
            <li><code>0 2 * * *</code> - 每天凌晨2点执行</li>
            <li><code>0 0 * * 0</code> - 每周日0点执行</li>
          </ul>

          <!-- 下次执行时间显示 -->
          <div v-if="cronTimes.length > 0" class="cron-next-times">
            <p><strong>下5次执行时间：</strong></p>
            <div v-loading="cronTimesLoading" class="cron-times-list">
              <div v-for="(time, index) in cronTimes" :key="index" class="cron-time-item">
                <el-tag type="info" size="small">{{ time }}</el-tag>
              </div>
            </div>
          </div>
        </div>
      </el-form-item>

      <!-- STRM直连地址 -->
      <el-form-item label="STRM直连地址" prop="direct_url">
        <el-input v-model="strmData.direct_url" placeholder="输入HTTP地址，如：http://192.168.1.100:8080"
          :disabled="strmLoading" @input="updateStrmExample" class="limited-width-input" />
        <div v-if="strmExample" class="strm-example-inline">
          <span class="example-label">示例：</span>
          <code class="example-url">{{ strmExample }}</code>
        </div>
        <div class="form-help">
          <p>STRM文件将使用此地址作为基础URL，请确保媒体服务器可以访问此地址</p>
          <p>一般使用部署本项目的机器的ip地址加上端口号，如：http://192.168.1.100:12333</p>
        </div>
      </el-form-item>

      <el-form-item label="是否下载元数据" prop="download_meta">
        <el-radio-group v-model="strmData.download_meta" @change="changeDownloadMeta">
          <el-radio-button :label="1">是</el-radio-button>
          <el-radio-button :label="0">否</el-radio-button>
        </el-radio-group>
        <div class="form-help">
          <p>如果选择是，同步时会将本地不存在的元数据文件下载回来</p>
          <p>
            如果选择否，同步时不会下载，<strong stylle="color: black;">但是也同时跳过处理元数据，已存在的会保留，新增的不会上传</strong>
          </p>
        </div>
      </el-form-item>

      <!-- 同步完是否上传网盘不存在的元数据 -->
      <el-form-item label="网盘不存在的元数据" prop="upload_meta">
        <el-radio-group v-model="strmData.upload_meta">
          <el-radio-button :label="2" :disabled="strmData.download_meta === 0">删除</el-radio-button>
          <el-radio-button :label="1" :disabled="strmData.download_meta === 0">上传</el-radio-button>
          <el-radio-button :label="0">保留</el-radio-button>
        </el-radio-group>
        <div class="form-help">
          <p>删除: 本地存在且网盘不存在则删除本地文件</p>
          <p>
            上传: 本地存在且网盘不存在，分三种情况: <br />
            &nbsp;&nbsp;&nbsp;&nbsp;1. 父目录在网盘存在则上传<br />
            &nbsp;&nbsp;&nbsp;&nbsp;2. 父目录在网盘不存在（网盘已删除）则删除本地文件<br />
            &nbsp;&nbsp;&nbsp;&nbsp;3. 父目录是特定名字，则创建父目录并上传，特定名字包括："extrafanart",
            "exfanarts",
            "extrafanarts",
            "extras",
            "specials",
            "shorts",
            "scenes",
            "featurettes",
            "behind the scenes",
            "trailers",
            "interviews",
          </p>
          <p>保留：不会删除本地文件，不管网盘有没有删除它</p>
        </div>
      </el-form-item>


      <!-- 同步完是否删除网盘不存在的空目录 -->
      <el-form-item label="网盘不存在的空目录" prop="delete_dir">
        <el-radio-group v-model="strmData.delete_dir">
          <el-radio-button :label="1">删除</el-radio-button>
          <el-radio-button :label="0">不删除</el-radio-button>
        </el-radio-group>
        <div class="form-help">
          <p>同步完成后是否删除本地存在但网盘不存在的目录，该本地目录必须是空目录</p>
        </div>
      </el-form-item>

      <el-form-item label="启用本地代理播放" prop="local_proxy">
        <el-radio-group v-model="strmData.local_proxy">
          <el-radio-button :label="1">启用</el-radio-button>
          <el-radio-button :label="0">关闭</el-radio-button>
        </el-radio-group>
        <div class="form-help">
          <p>
            开启后将使用本地代理访问网盘，可以解决局域网其他设备因为UA不同无法播放的问题。
          </p>
          <p>
            启用和关闭都不影响Emby 外网302的使用，外网302强制302跳转到直链。
          </p>
        </div>
      </el-form-item>
      <!-- 保存和重置按钮 -->
      <div class="strm-actions">
        <el-button type="success" @click="saveStrmConfig" :loading="strmLoading" size="large" :icon="Check">
          保存STRM配置
        </el-button>
      </div>
    </el-form>

    <!-- STRM配置状态显示 -->
    <el-alert v-if="strmStatus" :title="strmStatus.title" :type="strmStatus.type" :description="strmStatus.description"
      :closable="false" show-icon class="strm-status" />
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { Check } from '@element-plus/icons-vue'
import { inject, onMounted, reactive, ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { isMobile } from '@/utils/deviceUtils'
import MetadataExtInput from './MetadataExtInput.vue'
interface StrmData {
  video_ext: string[]
  min_file_size: number
  meta_ext: string[]
  cron_expression: string
  direct_url: string
  upload_meta: 0 | 1 | 2
  download_meta: 0 | 1
  delete_strm: 0 | 1
  delete_dir: 0 | 1
  local_proxy: 0 | 1
  exclude_name: string[]
}

interface StrmStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}
const checkIsMobile = ref(isMobile())
const http: AxiosStatic | undefined = inject('$http')

// 表单引用
const formRef = ref<FormInstance>()

// STRM配置相关状态
const strmLoading = ref(false)
const strmStatus = ref<StrmStatus | null>(null)
const strmExample = ref('')

// Cron下次执行时间相关状态
const cronTimes = ref<string[]>([])
const cronTimesLoading = ref(false)

// 默认STRM配置
const defaultStrmData: StrmData = {
  video_ext: ['.mp4', '.mkv', '.avi', '.mov', '.wmv', '.flv', '.webm', '.m4v', '.3gp', '.ts'],
  min_file_size: 50, // 默认50MB
  meta_ext: ['.jpg', '.jpeg', '.png', '.webp', '.nfo', '.srt', '.ass', '.svg', '.sup', '.lrc'],
  cron_expression: '30 * * * *',
  direct_url: '',
  upload_meta: 0, // 默认保留
  download_meta: 1, // 默认不下载元数据
  delete_strm: 1, // 默认删除
  delete_dir: 0, // 默认不删除
  local_proxy: 0, // 是否启用本地代理
  exclude_name: [], // 排除的名称列表，默认为空
}

const strmData = reactive<StrmData>({ ...defaultStrmData })

// 表单验证规则
const formRules: FormRules = {
  video_ext: [
    {
      required: true,
      validator: (rule, value, callback) => {
        if (!value || value.length === 0) {
          callback(new Error('请至少添加一个视频文件扩展名'))
        } else {
          callback()
        }
      },
      trigger: 'change',
    },
  ],
  min_file_size: [
    { required: true, message: '请输入最小文件大小', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value < 0) {
          callback(new Error('文件大小不能小于0'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
  meta_ext: [
    {
      required: true,
      validator: (rule, value, callback) => {
        if (!value || value.length === 0) {
          callback(new Error('请至少添加一个元数据扩展名'))
        } else {
          callback()
        }
      },
      trigger: 'change',
    },
  ],
  cron_expression: [{ required: true, message: '请输入定时同步表达式', trigger: 'blur' }],
  direct_url: [
    { required: true, message: '请输入STRM直连地址', trigger: 'blur' },
    {
      pattern: /^https?:\/\/.+/,
      message: '请输入有效的HTTP或HTTPS地址',
      trigger: 'blur',
    },
  ],
}

// 更新STRM示例
const updateStrmExample = () => {
  if (strmData.direct_url) {
    // 生成示例STRM文件内容
    const baseUrl = strmData.direct_url.replace(/\/$/, '') // 移除末尾斜杠
    strmExample.value = `${baseUrl}/115/newurl?pick_code=d6tkyd62bmngxx5bg`
  } else {
    strmExample.value = ''
  }
}

// 保存STRM配置
const saveStrmConfig = async () => {
  // 验证表单
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch (error) {
    console.log('表单验证失败:', error)
    return
  }

  try {
    strmLoading.value = true
    strmStatus.value = null

    const requestData = {
      video_ext: strmData.video_ext,
      min_video_size: strmData.min_file_size,
      meta_ext: strmData.meta_ext,
      cron: strmData.cron_expression,
      strm_base_url: strmData.direct_url,
      upload_meta: strmData.upload_meta,
      download_meta: strmData.download_meta,
      delete_strm: strmData.delete_strm,
      delete_dir: strmData.delete_dir,
      local_proxy: strmData.local_proxy,
      exclude_name: strmData.exclude_name,
    }

    const response = await http?.post(`${SERVER_URL}/setting/strm-config`, requestData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      strmStatus.value = {
        title: 'STRM配置已保存',
        type: 'success',
        description: '所有STRM相关设置已成功保存，将在下次同步时生效',
      }
    } else {
      strmStatus.value = {
        title: '保存STRM配置失败',
        type: 'error',
        description: response?.data.message || '保存设置失败，请重试',
      }
    }
  } catch (error) {
    console.error('保存STRM配置错误:', error)
    strmStatus.value = {
      title: '保存设置出错',
      type: 'error',
      description: '保存过程中发生错误，请检查网络连接',
    }
  } finally {
    strmLoading.value = false
  }
}

// 加载STRM配置
const loadStrmConfig = async () => {
  try {
    const response = await http?.get(`${SERVER_URL}/setting/strm-config`)

    if (response?.data.code === 200 && response.data.data) {
      const config = response.data.data
      strmData.video_ext = config.video_ext || defaultStrmData.video_ext
      strmData.min_file_size =
        config.min_video_size !== undefined ? config.min_video_size : defaultStrmData.min_file_size
      strmData.meta_ext = config.meta_ext || defaultStrmData.meta_ext
      strmData.cron_expression = config.cron || defaultStrmData.cron_expression
      strmData.direct_url = config.strm_base_url || ''
      strmData.upload_meta = config.upload_meta !== undefined ? config.upload_meta : 0
      strmData.download_meta = config.download_meta !== undefined ? config.download_meta : 1
      strmData.delete_strm = config.delete_strm !== undefined ? config.delete_strm : 1
      strmData.delete_dir = config.delete_dir !== undefined ? config.delete_dir : 0
      strmData.local_proxy = config.local_proxy !== undefined ? config.local_proxy : 0
      strmData.exclude_name = config.exclude_name || []

      // 更新示例
      updateStrmExample()

      // 加载Cron执行时间
      await loadCronTimes()
    }
  } catch (error) {
    console.error('加载STRM配置错误:', error)
  }
}

// 查询Cron下次执行时间
const loadCronTimes = async () => {
  if (!strmData.cron_expression) {
    cronTimes.value = []
    return
  }

  try {
    cronTimesLoading.value = true
    const response = await http?.get(`${SERVER_URL}/setting/cron`, {
      params: { cron: strmData.cron_expression },
    })

    if (response?.data.code === 200 && response.data.data) {
      cronTimes.value = response.data.data || []
    } else {
      cronTimes.value = []
    }
  } catch (error) {
    console.error('查询Cron执行时间错误:', error)
    cronTimes.value = []
  } finally {
    cronTimesLoading.value = false
  }
}

const changeDownloadMeta = () => {
  console.log('改变是否下载元数据')
  if (strmData.download_meta === 0) {
    strmData.upload_meta = 0
  }
}

// 监听cron表达式变化
watch(
  () => strmData.cron_expression,
  (newCron) => {
    if (newCron && newCron.trim()) {
      loadCronTimes()
    } else {
      cronTimes.value = []
    }
  },
  { immediate: false },
)

onMounted(() => {
  loadStrmConfig()
})
</script>
