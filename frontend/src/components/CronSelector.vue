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
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const selectedPreset = ref<string>('0 2 * * *')
const customCron = ref('')

const presetValues = [
  '0 2 * * *',
  '0 3 * * *',
  '0 */4 * * *',
  '0 */12 * * *',
  '0 0 * * 0',
]

const handlePresetChange = (value: string) => {
  if (value === 'custom') {
    customCron.value = props.modelValue || '0 2 * * *'
    emit('update:modelValue', customCron.value)
  } else {
    emit('update:modelValue', value)
  }
}

const handleCustomCronChange = () => {
  emit('update:modelValue', customCron.value)
}

onMounted(() => {
  if (props.modelValue) {
    if (presetValues.includes(props.modelValue)) {
      selectedPreset.value = props.modelValue
    } else {
      selectedPreset.value = 'custom'
      customCron.value = props.modelValue
    }
  }
})

watch(() => props.modelValue, (newValue) => {
  if (!newValue) return

  if (presetValues.includes(newValue)) {
    if (selectedPreset.value !== newValue) {
      selectedPreset.value = newValue
    }
  } else {
    if (selectedPreset.value !== 'custom' || customCron.value !== newValue) {
      selectedPreset.value = 'custom'
      customCron.value = newValue
    }
  }
})
</script>

<style scoped>
.cron-selector {
  width: 100%;
}
</style>
