<script setup lang="ts" generic="TRow">
import { computed } from 'vue'
import type { RecordAction, RecordActionPayload } from '@/types/recordTable'

const props = defineProps<{
  row: TRow
  actions: RecordAction<TRow>[]
}>()

const emit = defineEmits<{
  action: [payload: RecordActionPayload<TRow>]
}>()

const isVisible = (action: RecordAction<TRow>) => action.visible?.(props.row) ?? true
const isDisabled = (action: RecordAction<TRow>) => action.disabled?.(props.row) ?? false
const visibleActions = computed(() => props.actions.filter(isVisible))
</script>

<template>
  <div class="record-actions">
    <el-button
      v-for="action in visibleActions"
      :key="action.key"
      :type="action.type ?? 'primary'"
      :icon="action.icon"
      :disabled="isDisabled(action)"
      size="small"
      link
      @click="emit('action', { actionKey: action.key, row })"
    >
      <span>{{ action.label }}</span>
    </el-button>
  </div>
</template>

<style scoped>
.record-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-width: 0;
}

.record-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
</style>
