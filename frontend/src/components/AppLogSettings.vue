<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { LogLevel } from '@/types/log'
import { isLogLevel, LOG_LEVEL_OPTIONS } from '@/utils/logLevel'
import { isMobile } from '@/utils/deviceUtils'
import type { AxiosStatic } from 'axios'
import { ElMessage } from 'element-plus'
import { Check } from '@element-plus/icons-vue'
import { inject, onMounted, reactive, ref } from 'vue'

interface APIResponse<T> {
  code: number
  message: string
  data: T
}

interface LogSettingResponse {
  level: string
  levels: string[]
}

interface SaveStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}

const http: AxiosStatic | undefined = inject('$http')
const checkIsMobile = ref(isMobile())
const loading = ref(false)
const saveStatus = ref<SaveStatus | null>(null)
const formData = reactive<{ level: LogLevel }>({
  level: 'info',
})

function normalizeLevel(value: unknown): LogLevel {
  return isLogLevel(value) ? value : 'info'
}

async function fetchLogSetting() {
  try {
    loading.value = true
    const response = await http?.get<APIResponse<LogSettingResponse>>(`${SERVER_URL}/setting/log`)
    if (!response || response.data.code !== 200) {
      throw new Error(response?.data.message || '获取日志设置失败')
    }
    formData.level = normalizeLevel(response.data.data.level)
  } catch (error) {
    console.error('获取日志设置失败：', error)
    ElMessage.error(error instanceof Error ? error.message : '获取日志设置失败')
  } finally {
    loading.value = false
  }
}

async function saveLogSetting() {
  try {
    loading.value = true
    saveStatus.value = null
    const response = await http?.post<APIResponse<LogSettingResponse>>(
      `${SERVER_URL}/setting/log`,
      {
        level: formData.level,
      },
    )
    if (!response || response.data.code !== 200) {
      throw new Error(response?.data.message || '保存日志设置失败')
    }
    formData.level = normalizeLevel(response.data.data.level)
    saveStatus.value = {
      title: '保存成功',
      type: 'success',
      description: '日志等级已保存',
    }
    setTimeout(() => {
      saveStatus.value = null
    }, 3000)
  } catch (error) {
    console.error('保存日志设置失败：', error)
    saveStatus.value = {
      title: '保存失败',
      type: 'error',
      description: error instanceof Error ? error.message : '保存日志设置失败',
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void fetchLogSetting()
})
</script>

<template>
  <div class="main-content-container log-settings-container">
    <el-form
      :model="formData"
      :label-position="checkIsMobile ? 'top' : 'left'"
      :label-width="120"
      class="log-form"
    >
      <el-form-item label="日志等级" prop="level">
        <el-radio-group v-model="formData.level" :disabled="loading" size="large">
          <el-radio-button
            v-for="option in LOG_LEVEL_OPTIONS"
            :key="option.value"
            :value="option.value"
          >
            {{ option.label }}
          </el-radio-button>
        </el-radio-group>
      </el-form-item>

      <el-form-item>
        <div class="form-actions">
          <el-button
            type="success"
            size="large"
            :icon="Check"
            :loading="loading"
            @click="saveLogSetting"
          >
            保存设置
          </el-button>
        </div>
      </el-form-item>
    </el-form>

    <el-alert
      v-if="saveStatus"
      :title="saveStatus.title"
      :type="saveStatus.type"
      :description="saveStatus.description"
      :closable="false"
      show-icon
      class="save-status"
    />
  </div>
</template>

<style scoped>
.log-settings-container {
  max-width: 800px;
  padding: 20px;
}

.log-form {
  margin-top: 16px;
}

.form-actions {
  margin-top: 20px;
}

.save-status {
  margin-top: 20px;
}

@media (max-width: 768px) {
  .log-settings-container {
    padding: 16px;
  }

  .log-form :deep(.el-radio-group) {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%;
  }

  .log-form :deep(.el-radio-button__inner) {
    width: 100%;
  }
}
</style>
