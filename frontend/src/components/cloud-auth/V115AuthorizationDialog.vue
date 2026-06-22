<script setup lang="ts">
import { useV115DeviceAuthorization } from '@/composables/useV115DeviceAuthorization'
import V115QrCode from './V115QrCode.vue'
import { Refresh } from '@element-plus/icons-vue'
import type { AxiosStatic } from 'axios'
import { computed, inject, watch } from 'vue'

const visible = defineModel<boolean>('visible', { required: true })

const props = defineProps<{
  accountId: number | null
  accountName: string
}>()

const emit = defineEmits<{
  confirmed: []
}>()

const http: AxiosStatic | undefined = inject('$http')
const {
  qrCode,
  status,
  tip,
  loading,
  isPolling,
  startAuthorization,
  stopPolling,
  resetAuthorization,
} = useV115DeviceAuthorization(http)

const statusType = computed(() => {
  if (status.value === 'confirmed') return 'success' as const
  if (status.value === 'expired' || status.value === 'failed') return 'danger' as const
  if (status.value === 'scanned') return 'warning' as const
  return 'info' as const
})

watch(
  () => visible.value,
  (isVisible) => {
    if (isVisible && props.accountId) {
      void startAuthorization(props.accountId)
      return
    }
    stopPolling()
  },
)

watch(status, (value) => {
  if (value === 'confirmed') {
    visible.value = false
    emit('confirmed')
  }
})

const refreshQrCode = () => {
  if (props.accountId) void startAuthorization(props.accountId)
}

const handleClosed = () => {
  resetAuthorization()
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="115 二维码授权"
    width="360px"
    destroy-on-close
    @closed="handleClosed"
  >
    <div class="v115-auth-dialog">
      <div class="v115-auth-dialog__name">{{ accountName }}</div>
      <V115QrCode v-if="qrCode" :content="qrCode.qrcode" />
      <el-skeleton v-else :rows="4" animated />
      <el-tag :type="statusType" class="v115-auth-dialog__status">
        {{ tip || '等待授权' }}
      </el-tag>
      <el-button
        :icon="Refresh"
        :loading="loading"
        :disabled="isPolling && status !== 'expired'"
        @click="refreshQrCode"
      >
        刷新二维码
      </el-button>
    </div>
  </el-dialog>
</template>

<style scoped>
.v115-auth-dialog {
  display: grid;
  justify-items: center;
  gap: 14px;
  min-width: 0;
}

.v115-auth-dialog__name {
  max-width: 100%;
  overflow-wrap: anywhere;
  color: #606266;
}

.v115-auth-dialog__status {
  max-width: 100%;
}
</style>
