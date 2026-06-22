<script setup lang="ts" generic="TRow">
import type { RecordDetailField } from '@/types/recordTable'

const props = defineProps<{
  row: TRow
  fields: RecordDetailField<TRow>[]
}>()

const getValue = (field: RecordDetailField<TRow>) => {
  const value = field.value(props.row)
  return value === null || value === undefined || value === '' ? '-' : String(value)
}
</script>

<template>
  <el-descriptions class="record-detail" :column="2" border size="small">
    <el-descriptions-item
      v-for="field in fields"
      :key="field.key"
      :label="field.label"
      :span="field.span ?? 1"
    >
      <span :class="['record-detail__value', { 'record-detail__value--long': field.isLongText }]">
        {{ getValue(field) }}
      </span>
    </el-descriptions-item>
  </el-descriptions>
</template>

<style scoped>
.record-detail__value {
  font-variant-numeric: tabular-nums;
}

.record-detail__value--long {
  overflow-wrap: anywhere;
}
</style>
