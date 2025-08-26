<template>
  <div class="strm-settings-container">
    <!-- STRM设置卡片 -->
    <el-card class="strm-settings-card" shadow="hover">
      <template #header>
        <h2 class="card-title">STRM配置</h2>
        <p class="card-subtitle">配置STRM文件生成和同步相关设置</p>
      </template>

      <div class="strm-content">
        <el-form
          :model="strmData"
          :rules="formRules"
          :label-position="'top'"
          class="strm-form"
          ref="formRef"
        >
          <!-- 视频文件扩展名 -->
          <el-form-item label="视频文件扩展名" prop="video_ext">
            <el-input-tag
              v-model="strmData.video_ext"
              placeholder="输入扩展名后按回车添加，如：.mp4"
              class="meta-ext-input limited-width-input"
            />
            <div class="form-help">
              <p>指定需要生成STRM文件的视频文件扩展名，如：.mp4, .mkv, .avi, .mov 等</p>
            </div>
          </el-form-item>

          <!-- 最小文件大小 -->
          <el-form-item label="最小文件大小 (MB)" prop="min_file_size">
            <el-input-number
              v-model="strmData.min_file_size"
              :min="0"
              :step="1"
              :precision="0"
              placeholder="输入最小文件大小"
              :disabled="strmLoading"
              class="limited-width-input"
            />
            <div class="form-help">
              <p>小于此大小的视频文件将不会生成STRM文件，单位为MB。设置为0表示不限制文件大小</p>
            </div>
          </el-form-item>

          <!-- 元数据扩展名 -->
          <el-form-item label="元数据扩展名" prop="meta_ext">
            <el-input-tag
              v-model="strmData.meta_ext"
              placeholder="输入扩展名后按回车添加，如：.jpg"
              class="meta-ext-input limited-width-input"
            />
            <div class="form-help">
              <p>指定需要处理的元数据文件扩展名，如：.jpg, .nfo, .srt, .ass 等</p>
            </div>
          </el-form-item>

          <!-- 定时同步表达式 -->
          <el-form-item label="定时同步表达式" prop="cron_expression">
            <el-input
              v-model="strmData.cron_expression"
              placeholder="输入Cron表达式，如：0 2 * * *"
              :disabled="strmLoading"
              class="limited-width-input"
              @blur="loadCronTimes"
            />
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
            <el-input
              v-model="strmData.direct_url"
              placeholder="输入HTTP地址，如：http://192.168.1.100:8080"
              :disabled="strmLoading"
              @input="updateStrmExample"
              class="limited-width-input"
            />
            <div v-if="strmExample" class="strm-example-inline">
              <span class="example-label">示例STRM文件内容：</span>
              <code class="example-url">{{ strmExample }}</code>
            </div>
            <div class="form-help">
              <p>STRM文件将使用此地址作为基础URL，请确保媒体服务器可以访问此地址</p>
              <p>一般使用部署本项目的机器的ip地址加上端口号，如：http://192.168.1.100:12333</p>
            </div>
          </el-form-item>

          <!-- 同步完是否上传网盘不存在的元数据 -->
          <el-form-item label="不存在的元数据" prop="upload_meta">
            <el-radio-group v-model="strmData.upload_meta">
              <el-radio-button :label="2">删除</el-radio-button>
              <el-radio-button :label="1">上传</el-radio-button>
              <el-radio-button :label="0">保留</el-radio-button>
            </el-radio-group>
            <div class="form-help">
              <p>删除: 本地存在且网盘不存在则删除本地文件</p>
              <p>
                上传: 本地存在且网盘不存在，分两种情况: <br />
                &nbsp;&nbsp;&nbsp;&nbsp;1. 父目录在网盘存在则上传<br />
                &nbsp;&nbsp;&nbsp;&nbsp;2. 父目录在网盘不存在（网盘已删除）责删除本地文件
              </p>
              <p>保留：不会删除本地文件，不管网盘有没有删除它</p>
            </div>
          </el-form-item>

          <!-- 同步完是否删除网盘不存在的STRM文件 -->
          <el-form-item label="删除网盘不存在的STRM文件" prop="delete_strm">
            <el-radio-group v-model="strmData.delete_strm">
              <el-radio-button :label="1">删除</el-radio-button>
              <el-radio-button :label="0">不删除</el-radio-button>
            </el-radio-group>
            <div class="form-help">
              <p>同步完成后是否删除本地存在但网盘不存在的STRM文件</p>
            </div>
          </el-form-item>

          <!-- 同步完是否删除网盘不存在的空目录 -->
          <el-form-item label="删除网盘不存在的空目录" prop="delete_dir">
            <el-radio-group v-model="strmData.delete_dir">
              <el-radio-button :label="1">删除</el-radio-button>
              <el-radio-button :label="0">不删除</el-radio-button>
            </el-radio-group>
            <div class="form-help">
              <p>同步完成后是否删除本地存在但网盘不存在的空目录</p>
            </div>
          </el-form-item>

          <!-- 保存和重置按钮 -->
          <el-form-item>
            <div class="strm-actions">
              <el-button type="success" @click="saveStrmConfig" :loading="strmLoading" size="large">
                <el-icon><Check /></el-icon>
                保存STRM配置
              </el-button>

              <el-button
                type="warning"
                plain
                @click="resetStrmConfig"
                :disabled="strmLoading"
                size="large"
              >
                <el-icon><Refresh /></el-icon>
                重置为默认值
              </el-button>
            </div>
          </el-form-item>
        </el-form>

        <!-- STRM配置状态显示 -->
        <el-alert
          v-if="strmStatus"
          :title="strmStatus.title"
          :type="strmStatus.type"
          :description="strmStatus.description"
          :closable="false"
          show-icon
          class="strm-status"
        />
      </div>
    </el-card>

    <!-- STRM配置说明卡片 -->
    <el-card class="strm-help-card" shadow="hover">
      <template #header>
        <h3 class="help-title">配置说明</h3>
      </template>

      <div class="help-content">
        <div class="help-section">
          <h4>什么是STRM文件？</h4>
          <p>
            STRM文件是一种特殊的播放列表文件，包含指向远程媒体文件的URL。媒体服务器（如Plex、Emby、Jellyfin）可以读取这些文件并直接播放远程内容，而无需本地存储完整的媒体文件。
          </p>
        </div>

        <div class="help-section">
          <h4>元数据文件</h4>
          <p>
            元数据文件包含媒体的附加信息，如封面图片(.jpg)、字幕文件(.srt)、NFO信息文件(.nfo)等。这些文件可以增强媒体服务器的显示效果和播放体验。
          </p>
        </div>

        <div class="help-section">
          <h4>定时同步</h4>
          <p>
            使用Cron表达式设置自动同步时间。系统将按照设定的时间自动检查115网盘的更新并生成相应的STRM文件。
          </p>
        </div>

        <div class="help-section">
          <h4>直连地址配置</h4>
          <p>直连地址是媒体服务器访问115网盘文件的入口地址。请确保：</p>
          <ul>
            <li>地址格式正确（http://或https://开头）</li>
            <li>媒体服务器可以正常访问此地址</li>
            <li>端口号与实际服务端口一致</li>
          </ul>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { Check, Refresh } from '@element-plus/icons-vue'
import { inject, onMounted, reactive, ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'

interface StrmData {
  video_ext: string[]
  min_file_size: number
  meta_ext: string[]
  cron_expression: string
  direct_url: string
  upload_meta: 0 | 1 | 2
  delete_strm: 0 | 1
  delete_dir: 0 | 1
}

interface StrmStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}

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
  delete_strm: 1, // 默认删除
  delete_dir: 0, // 默认不删除
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
    strmExample.value = `${baseUrl}/115/url?pick_code=d6tkyd62bmngxx5bg&sha1=xxxxxxx&name=哪吒.mkv&size=1024000`
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

    const saveData = new URLSearchParams()
    saveData.append('video_ext', JSON.stringify(strmData.video_ext))
    saveData.append('min_video_size', strmData.min_file_size.toString())
    saveData.append('meta_ext', JSON.stringify(strmData.meta_ext))
    saveData.append('cron', strmData.cron_expression)
    saveData.append('strm_base_url', strmData.direct_url)
    saveData.append('upload_meta', strmData.upload_meta.toString())
    saveData.append('delete_strm', strmData.delete_strm.toString())
    saveData.append('delete_dir', strmData.delete_dir.toString())

    const response = await http?.post(`${SERVER_URL}/setting/update-strm-config`, saveData, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
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
        description: response?.data.msg || '保存设置失败，请重试',
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
      strmData.delete_strm = config.delete_strm !== undefined ? config.delete_strm : 1
      strmData.delete_dir = config.delete_dir !== undefined ? config.delete_dir : 0

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

// 重置STRM配置为默认值
const resetStrmConfig = () => {
  Object.assign(strmData, defaultStrmData)
  updateStrmExample()

  // 清除表单验证
  if (formRef.value) {
    formRef.value.clearValidate()
  }

  strmStatus.value = {
    title: '已重置为默认配置',
    type: 'info',
    description: '配置已重置为默认值，请点击保存按钮应用更改',
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

<style scoped>
.strm-settings-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.strm-settings-card,
.strm-help-card {
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

.strm-content {
  margin-top: 20px;
}

.strm-form {
  margin-top: 16px;
}

.strm-form .el-form-item {
  margin-bottom: 24px;
  position: relative;
}

.strm-form .el-form-item .el-form-item__content {
  position: relative;
}

.form-help {
  font-size: 12px;
  color: #909399;
  margin-top: 8px;
  line-height: 1.5;
  display: block;
  width: 100%;
  clear: both;
}

.form-help p {
  margin: 8px 0 12px 0;
  font-weight: 600;
  display: block;
  width: 100%;
  clear: both;
}

.form-help ul {
  margin: 8px 0 0 0;
  padding-left: 16px;
}

.form-help li {
  margin-bottom: 4px;
  line-height: 1.4;
}

.form-help code {
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 11px;
  color: #e6a23c;
}

.cron-examples {
  list-style: none;
  padding-left: 0;
}

.cron-examples li {
  padding: 6px 0;
  border-bottom: 1px solid #f0f0f0;
}

.cron-examples li:last-child {
  border-bottom: none;
}

.cron-next-times {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e4e7ed;
}

.cron-next-times p {
  margin: 0 0 8px 0;
  font-weight: 600;
  color: #303133;
}

.cron-times-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0;
}

.cron-time-item {
  margin-bottom: 4px;
}

.meta-ext-input {
  max-width: 500px;
  width: 100%;
}

.limited-width-input {
  max-width: 500px;
  width: 100%;
}

.limited-width-input.el-input-number {
  max-width: 200px;
}

.strm-example {
  margin-top: 12px;
  padding: 12px;
  background: #f8f9fa;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
}

.strm-example-inline {
  margin-top: 6px;
  margin-bottom: 0;
  padding: 6px 0;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  position: relative;
  width: 100%;
  clear: both;
}

.example-label {
  font-size: 13px;
  font-weight: 600;
  color: #606266;
  white-space: nowrap;
}

.example-url {
  background: #f5f7fa;
  padding: 4px 8px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  color: #e6a23c;
  word-break: break-all;
  flex: 1;
  min-width: 0;
}

.pan-dir-input {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.pan-dir-input .el-input {
  flex: 1;
}

.selected-path {
  margin-top: 12px;
  padding: 12px;
  background: #f0f9ff;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
}

.selected-path-inline {
  margin-top: 6px;
  margin-bottom: 0;
  padding: 6px 0;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  position: relative;
  width: 100%;
  clear: both;
}

.path-label {
  font-size: 13px;
  font-weight: 600;
  color: #1e40af;
  white-space: nowrap;
}

.path-url {
  background: #e0f2fe;
  padding: 4px 8px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  color: #0c4a6e;
  word-break: break-all;
  flex: 1;
  min-width: 0;
}

.dir-selector {
  padding: 16px 0;
}

.loading-container,
.empty-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: #909399;
}

.loading-container .el-icon {
  font-size: 24px;
  margin-bottom: 8px;
}

.dir-list {
  padding: 8px;
}

.dir-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  border-radius: 6px;
  transition: background-color 0.2s;
  justify-content: space-between;
}

.dir-item:hover {
  background-color: #f5f7fa;
}

.dir-item .el-icon {
  color: #409eff;
}

.dir-item .el-icon:first-child {
  margin-right: 8px;
}

.enter-icon {
  margin-left: 8px;
  font-size: 14px;
  color: #909399;
}

.dir-name {
  font-size: 14px;
  color: #303133;
  flex: 1;
}

.selected-dir-section {
  margin-top: 16px;
  padding: 16px;
  background: #f0f9ff;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
}

.selected-dir-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.selected-dir-label {
  font-size: 14px;
  font-weight: 600;
  color: #1e40af;
}

.selected-dir-path {
  font-size: 13px;
  color: #0c4a6e;
  background: #e0f2fe;
  padding: 8px 12px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  word-break: break-all;
}

.strm-actions {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.strm-status {
  margin-top: 16px;
}

.help-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.help-content {
  margin-top: 16px;
}

.help-section {
  margin-bottom: 24px;
}

.help-section:last-child {
  margin-bottom: 0;
}

.help-section h4 {
  margin: 0 0 12px 0;
  font-size: 16px;
  font-weight: 600;
  color: #409eff;
}

.help-section p {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #606266;
  line-height: 1.6;
}

.help-section p:last-child {
  margin-bottom: 0;
}

.help-section ul {
  margin: 8px 0 0 16px;
  padding: 0;
}

.help-section li {
  margin-bottom: 6px;
  font-size: 14px;
  color: #606266;
  line-height: 1.5;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .strm-settings-card,
  .strm-help-card {
    margin: 0;
  }

  .card-title {
    font-size: 20px;
  }

  .card-subtitle {
    font-size: 13px;
  }

  .limited-width-input,
  .meta-ext-input {
    max-width: none;
    width: 100%;
  }

  .strm-actions {
    flex-direction: column;
    gap: 8px;
  }

  .strm-actions .el-button {
    width: 100%;
  }

  .pan-dir-input {
    flex-direction: column;
    gap: 8px;
  }

  .pan-dir-input .el-button {
    width: 100%;
  }

  .strm-example-inline,
  .selected-path-inline {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .example-label,
  .path-label {
    font-size: 12px;
  }

  .example-url,
  .path-url {
    width: 100%;
    font-size: 11px;
  }

  .help-title {
    font-size: 16px;
  }

  .help-section h4 {
    font-size: 15px;
  }

  .help-section p,
  .help-section li {
    font-size: 13px;
  }
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .card-title {
    font-size: 18px;
  }

  .help-title {
    font-size: 15px;
  }

  .help-section h4 {
    font-size: 14px;
  }
}
</style>
