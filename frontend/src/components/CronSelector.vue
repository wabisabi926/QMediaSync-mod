<template>
  <div class="cron-selector">
    <el-select
      v-model="selectedPreset"
      placeholder="选择定时策略"
      @change="handlePresetChange"
      style="width: 100%"
    >
      <el-option label="每天凌晨2点" value="0 2 * * *" />
      <el-option label="每天凌晨3点" value="0 3 * * *" />
      <el-option label="每4小时" value="0 */4 * * *" />
      <el-option label="每12小时" value="0 */12 * * *" />
      <el-option label="每周日凌晨" value="0 0 * * 0" />
      <el-option label="自定义" value="custom" />
    </el-select>

    <!-- 自定义输入框 -->
    <el-input
      v-if="selectedPreset === 'custom'"
      v-model="customCron"
      placeholder="请输入 Cron 表达式，如: 0 2 * * *"
      @input="handleCustomCronChange"
      style="margin-top: 12px"
    >
      <template #prepend>Cron</template>
    </el-input>

    <!-- 验证错误提示 -->
    <el-alert
      v-if="validationError"
      :title="validationError"
      type="error"
      :closable="false"
      style="margin-top: 8px"
    />

    <!-- 下次执行时间预览 -->
    <div v-if="nextExecutionTime && !validationError" class="next-execution">
      <el-icon><Clock /></el-icon>
      <span>下次执行时间：{{ nextExecutionTime }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { Clock } from '@element-plus/icons-vue'

type CronParserClass = {
  new (expr: string): {
    reset(): void
    next(): { toDate(): Date }
  }
}

let CronParser: CronParserClass | null = null

// 懒罗加载 cron-parser
const loadCronParser = async () => {
  if (!CronParser) {
    const cronParser = await import('cron-parser')
    CronParser = cronParser.default as unknown as CronParserClass
  }
}

onMounted(() => {
  loadCronParser()
})

// Props 和 Emits
const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

// 状态
const selectedPreset = ref<string>('0 2 * * *')
const customCron = ref('')
const validationError = ref('')
const nextExecutionTime = ref('')

// 预设值映射
const presetValues = [
  '0 2 * * *',
  '0 3 * * *',
  '0 */4 * * *',
  '0 */12 * * *',
  '0 0 * * 0',
]

// 验证 Cron 表达式
const validateCron = (cronExpression: string): { valid: boolean; error?: string } => {
  try {
    if (!CronParser) {
      return { valid: false, error: 'Cron解析器未加载' }
    }

    // 使用 cron-parser 验证表达式
    const expression = new CronParser(cronExpression)

    // 计算下两次执行时间，用于验证最小间隔
    expression.reset()
    const next1 = expression.next().toDate()
    const next2 = expression.next().toDate()
    const intervalMinutes = (next2.getTime() - next1.getTime()) / (1000 * 60)

    // 验证最小间隔是否 >= 1小时
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

// 计算下次执行时间
const calculateNextExecution = (cronExpression: string) => {
  try {
    if (!CronParser) {
      nextExecutionTime.value = ''
      return
    }

    const expression = new CronParser(cronExpression)
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

// 处理预设值变更
const handlePresetChange = (value: string) => {
  validationError.value = ''

  if (value === 'custom') {
    // 切换到自定义模式，保留当前值或使用默认值
    customCron.value = props.modelValue || '0 2 * * *'
    validateAndEmit(customCron.value)
  } else {
    // 使用预设值
    validateAndEmit(value)
  }
}

// 处理自定义 Cron 变更
const handleCustomCronChange = () => {
  validateAndEmit(customCron.value)
}

// 验证并发出更新事件
const validateAndEmit = (cronExpression: string) => {
  const result = validateCron(cronExpression)

  if (result.valid) {
    validationError.value = ''
    calculateNextExecution(cronExpression)
    emit('update:modelValue', cronExpression)
  } else {
    validationError.value = result.error || '验证失败'
    nextExecutionTime.value = ''
  }
}

// 初始化
onMounted(() => {
  if (props.modelValue) {
    // 检查是否是预设值
    if (presetValues.includes(props.modelValue)) {
      selectedPreset.value = props.modelValue
    } else {
      selectedPreset.value = 'custom'
      customCron.value = props.modelValue
    }

    // 验证并计算下次执行时间
    const result = validateCron(props.modelValue)
    if (result.valid) {
      calculateNextExecution(props.modelValue)
    } else {
      validationError.value = result.error || ''
    }
  }
})

// 监听外部值变化
watch(() => props.modelValue, (newValue) => {
  if (newValue && newValue !== customCron.value && selectedPreset.value === 'custom') {
    customCron.value = newValue
    const result = validateCron(newValue)
    if (result.valid) {
      validationError.value = ''
      calculateNextExecution(newValue)
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
