<template>
  <div class="cron-selector">
    <el-select v-model="selectedPreset" placeholder="选择定时策略" @change="handlePresetChange" style="width: 100%">
      <el-option label="每天凌晨2点" value="0 2 * * *" />
      <el-option label="每天凌晨3点" value="0 3 * * *" />
      <el-option label="每4小时" value="0 */4 * * *" />
      <el-option label="每12小时" value="0 */12 * * *" />
      <el-option label="每周日凌晨" value="0 0 * * 0" />
      <el-option label="自定义" value="custom" />
    </el-select>

    <!-- 自定义输入框 -->
    <el-input v-if="selectedPreset === 'custom'" v-model="customCron" placeholder="请输入 Cron 表达式，如: 0 2 * * *"
      @input="handleCustomCronChange" style="margin-top: 12px">
      <template #prepend>Cron</template>
    </el-input>

    <!-- 验证错误提示 -->
    <el-alert v-if="validationError" :title="validationError" type="error" :closable="false" style="margin-top: 8px" />

    <!-- 下次执行时间预览 -->
    <div v-if="nextExecutionTime && !validationError" class="next-execution">
      <el-icon>
        <Clock />
      </el-icon>
      <span>下次执行时间：{{ nextExecutionTime }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { Clock } from '@element-plus/icons-vue'

type CronParserClass = {
  new(expr: string): {
    reset(): void
    next(): { toDate(): Date }
  }
}

let CronParser: CronParserClass | null = null

const loadCronParser = async () => {
  if (!CronParser) {
    const cronParser = await import('cron-parser')
    CronParser = cronParser.default as unknown as CronParserClass
  }
  return CronParser
}

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const selectedPreset = ref<string>('0 2 * * *')
const customCron = ref('')
const validationError = ref('')
const nextExecutionTime = ref('')

const presetValues = [
  '0 2 * * *',
  '0 3 * * *',
  '0 */4 * * *',
  '0 */12 * * *',
  '0 0 * * 0',
]

const validateCron = async (cronExpression: string): Promise<{ valid: boolean; error?: string }> => {
  try {
    const parser = await loadCronParser()
    if (!parser) {
      return { valid: false, error: 'Cron解析器未加载' }
    }

    const expression = new parser(cronExpression)

    expression.reset()
    const next1 = expression.next().toDate()
    const next2 = expression.next().toDate()
    const intervalMinutes = (next2.getTime() - next1.getTime()) / (1000 * 60)

    if (intervalMinutes < 60) {
      return {
        valid: false,
        error: '备份任务定时最小间隔为1小时',
      }
    }

    return { valid: true }
  } catch {
    return {
      valid: false,
      error: 'Cron表达式格式无效',
    }
  }
}

const calculateNextExecution = async (cronExpression: string) => {
  try {
    const parser = await loadCronParser()
    if (!parser) {
      nextExecutionTime.value = ''
      return
    }

    const expression = new parser(cronExpression)
    const next = expression.next().toDate()
    nextExecutionTime.value = next.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    })
  } catch {
    nextExecutionTime.value = ''
  }
}

const handlePresetChange = async (value: string) => {
  validationError.value = ''
  console.log(value)
  if (value === 'custom') {
    customCron.value = props.modelValue || '0 2 * * *'
    await validateAndEmit(customCron.value)
  } else {
    await validateAndEmit(value)
  }
}

const handleCustomCronChange = async () => {
  await validateAndEmit(customCron.value)
}

const validateAndEmit = async (cronExpression: string) => {
  // const result = await validateCron(cronExpression)

  // if (result.valid) {
  //   validationError.value = ''
  await calculateNextExecution(cronExpression)
  emit('update:modelValue', cronExpression)
  // } else {
  //   validationError.value = result.error || '验证失败'
  //   nextExecutionTime.value = ''
  // }
}

onMounted(async () => {
  if (props.modelValue) {
    console.log('加载初始值:', props.modelValue)
    if (presetValues.includes(props.modelValue)) {
      selectedPreset.value = props.modelValue
    } else {
      selectedPreset.value = 'custom'
      customCron.value = props.modelValue
    }

    const result = await validateCron(props.modelValue)
    if (result.valid) {
      await calculateNextExecution(props.modelValue)
    } else {
      validationError.value = result.error || ''
    }
  }
})

watch(() => props.modelValue, async (newValue) => {
  if (!newValue) return

  // 检查新值是否是预设值
  if (presetValues.includes(newValue)) {
    if (selectedPreset.value !== newValue) {
      selectedPreset.value = newValue
      validationError.value = ''
      await calculateNextExecution(newValue)
    }
  } else {
    // 自定义值
    if (selectedPreset.value !== 'custom' || customCron.value !== newValue) {
      selectedPreset.value = 'custom'
      customCron.value = newValue
      const result = await validateCron(newValue)
      if (result.valid) {
        validationError.value = ''
        await calculateNextExecution(newValue)
      } else {
        validationError.value = result.error || ''
      }
    }
  }
})
</script>

<style scoped>
.cron-selector {
  width: 100%;
}

.next-execution {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 8px 12px;
  background-color: #f0f9ff;
  border: 1px solid #91d5ff;
  border-radius: 4px;
  color: #1890ff;
  font-size: 13px;
}
</style>
