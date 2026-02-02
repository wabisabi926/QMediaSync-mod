<template>
  <div class="backup-settings-container">
    <div class="page-header">
      <el-icon>
        <Setting />
      </el-icon>
      <span>备份配置</span>
    </div>

    <el-form ref="configFormRef" :model="configForm" label-width="120px" :label-position="isMobile ? 'top' : 'right'">
      <el-form-item label="启用自动备份">
        <el-switch v-model="configForm.backup_enabled" :active-value="1" :inactive-value="0" />
      </el-form-item>

      <el-form-item label="定时策略" required>
        <cron-selector v-model="configForm.backup_cron" />
      </el-form-item>

      <el-form-item label="备份路径" required>
        <el-input v-model="configForm.backup_path" placeholder="例如: config/backups/" />
      </el-form-item>

      <el-form-item label="保留天数" required>
        <el-input-number v-model="configForm.backup_retention" :min="1" :max="365" controls-position="right" />
        <span style="margin-left: 8px; color: #909399">天</span>
      </el-form-item>

      <el-form-item label="最大备份数" required>
        <el-input-number v-model="configForm.backup_max_count" :min="1" :max="100" controls-position="right" />
        <span style="margin-left: 8px; color: #909399">个</span>
      </el-form-item>

      <el-form-item label="压缩备份">
        <el-switch v-model="configForm.backup_compress" :active-value="1" :inactive-value="0" />
      </el-form-item>

      <el-form-item>
        <el-button type="primary" :loading="configSaving" @click="saveConfig">
          保存配置
        </el-button>
      </el-form-item>
    </el-form>

    <!-- 下次执行时间提示 -->
    <el-alert v-if="nextBackupTime" :title="`下次自动备份时间：${nextBackupTime}`" type="info" :closable="false" />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, inject } from 'vue'
import { Setting } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { AxiosStatic } from 'axios'
import { SERVER_URL } from '@/const'
import type { BackupConfigResponse } from '@/typing'
import { isMobile as checkIsMobile } from '@/utils/deviceUtils'
import CronSelector from './CronSelector.vue'

type CronParserClass = {
  new(expr: string): {
    reset(): void
    next(): { toDate(): Date }
  }
}

let CronParser: CronParserClass | null = null

// 懒加载 cron-parser
const loadCronParser = async () => {
  if (!CronParser) {
    const cronParser = await import('cron-parser')
    CronParser = cronParser.default as unknown as CronParserClass
  }
  return CronParser
}

const http = inject<AxiosStatic>('$http')
const isMobile = checkIsMobile()

// 配置表单
const configForm = reactive({
  backup_enabled: 1 as 0 | 1,
  backup_cron: '0 2 * * *',
  backup_path: 'config/backups/',
  backup_retention: 7,
  backup_max_count: 10,
  backup_compress: 1 as 0 | 1,
})

const configSaving = ref(false)
const nextBackupTime = ref('')

// 加载备份配置
const loadBackupConfig = async () => {
  if (!http) return

  try {
    const res = await http.get<{ code: number; data: BackupConfigResponse }>(
      `${SERVER_URL}/database/backup-config`
    )

    if (res.data.code === 200 && res.data.data.exists && res.data.data.config) {
      const config = res.data.data.config

      const cronExpression = config.backup_cron || '0 2 * * *'

      Object.assign(configForm, {
        backup_enabled: config.backup_enabled,
        backup_cron: cronExpression,
        backup_path: config.backup_path,
        backup_retention: config.backup_retention,
        backup_max_count: config.backup_max_count,
        backup_compress: config.backup_compress,
      })
      await calculateNextBackupTime()
    }
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : '加载备份配置失败'
    ElMessage.error(errorMsg)
  }
}

// 保存备份配置
const saveConfig = async () => {
  if (!http) return

  configSaving.value = true
  try {
    const res = await http.post(`${SERVER_URL}/database/backup-config`, configForm)

    if (res.data.code === 200) {
      ElMessage.success('备份配置保存成功')
      await calculateNextBackupTime()
    } else {
      ElMessage.error('保存配置失败')
    }
  } catch (error: unknown) {
    const errorMsg = error instanceof Error ? error.message : '保存配置失败'
    ElMessage.error(errorMsg)
  } finally {
    configSaving.value = false
  }
}

// 计算下次备份时间
const calculateNextBackupTime = async () => {
  try {
    if (!configForm.backup_enabled || !configForm.backup_cron) {
      nextBackupTime.value = ''
      return
    }

    const parser = await loadCronParser()
    if (!parser) {
      nextBackupTime.value = ''
      return
    }

    const expression = new parser(configForm.backup_cron)
    const next = expression.next().toDate()
    nextBackupTime.value = next.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    })
  } catch {
    nextBackupTime.value = ''
  }
}

// 组件挂载时加载数据
onMounted(async () => {
  await loadCronParser()
  loadBackupConfig()
})
</script>

<style scoped>
.backup-settings-container {
  padding: 20px;
  max-width: 1200px;
}

.page-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 18px;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e4e7ed;
}

@media (max-width: 768px) {
  .backup-settings-container {
    padding: 10px;
  }
}
</style>
