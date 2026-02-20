<script setup lang="ts">
import { useVersion } from '@/composables/useVersion'

const { versionInfo, versionLoading } = useVersion()
</script>

<template>
  <div class="info-card system-info" v-loading="versionLoading">
    <div class="info-card-header">
      <span class="info-icon">⚙️</span>
      <span>系统信息</span>
    </div>
    <div v-if="versionInfo" class="info-content">
      <div class="info-row">
        <span class="info-label">版本</span>
        <span class="info-value version-tag">{{ versionInfo.version }}</span>
      </div>
      <div class="info-row">
        <span class="info-label">编译时间</span>
        <span class="info-value">{{ versionInfo.date }}</span>
      </div>
    </div>
    <div v-else class="empty-state-small">
      <el-empty description="暂无信息" :image-size="40" />
    </div>
  </div>
</template>

<style scoped>
.info-card {
  background: white;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #f0f0f0;
}

.info-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.info-icon {
  font-size: 18px;
}

.info-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.info-label {
  font-size: 13px;
  color: #909399;
}

.info-value {
  font-size: 13px;
  color: #303133;
  font-weight: 500;
}

.version-tag {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.empty-state-small {
  padding: 20px;
  text-align: center;
}
</style>
