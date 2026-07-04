<script setup lang="ts">
import { Delete, Plus } from '@element-plus/icons-vue'

import type { WebhookHeaderRow } from '@/utils/notificationUtils'

const headers = defineModel<WebhookHeaderRow[]>({ default: () => [] })

const createHeaderRow = (): WebhookHeaderRow => {
  const maxID = headers.value.reduce((current, row) => Math.max(current, row.id), 0)
  return {
    id: maxID + 1,
    key: '',
    value: '',
  }
}

const addHeader = () => {
  headers.value = [...headers.value, createHeaderRow()]
}

const removeHeader = (id: number) => {
  headers.value = headers.value.filter((row) => row.id !== id)
}

const updateHeader = (id: number, field: 'key' | 'value', value: string) => {
  headers.value = headers.value.map((row) => (row.id === id ? { ...row, [field]: value } : row))
}
</script>

<template>
  <el-form-item label="额外 Headers">
    <div class="webhook-headers-editor">
      <div v-for="row in headers" :key="row.id" class="webhook-header-row">
        <el-input
          class="webhook-header-name"
          :model-value="row.key"
          placeholder="Header 名称"
          @update:model-value="updateHeader(row.id, 'key', $event)"
        />
        <el-input
          class="webhook-header-value"
          :model-value="row.value"
          placeholder="Header 值"
          @update:model-value="updateHeader(row.id, 'value', $event)"
        />
        <el-button :icon="Delete" circle text type="danger" @click="removeHeader(row.id)" />
      </div>
      <el-button :icon="Plus" text type="primary" @click="addHeader">添加 Header</el-button>
    </div>
  </el-form-item>
</template>

<style scoped>
.webhook-headers-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.webhook-header-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 32px;
  gap: 8px;
  align-items: center;
}

.webhook-header-name,
.webhook-header-value {
  min-width: 0;
}

@media (max-width: 768px) {
  .webhook-header-row {
    grid-template-columns: minmax(0, 1fr) 32px;
  }

  .webhook-header-value {
    grid-column: 1 / -1;
  }
}
</style>
