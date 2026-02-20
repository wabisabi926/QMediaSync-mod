<script setup lang="ts">
import { ref } from 'vue'
import { Document } from '@element-plus/icons-vue'
import VersionManager from './VersionManager.vue'
import QueueStatsCard from './QueueStatsCard.vue'
import HourlyStatsChart from './HourlyStatsChart.vue'
import AnnouncementCard from './AnnouncementCard.vue'
import AppLogViewer from './AppLogViewer.vue'

const showLogDialog = ref(false)
const logViewerRef = ref<InstanceType<typeof AppLogViewer> | null>(null)

const handleLogDialogClose = () => {
  if (logViewerRef.value) {
    logViewerRef.value.disconnect()
  }
}
</script>

<template>
  <div class="home-container">
    <div class="header-section">
      <div class="header-title">
        <h1>控制台</h1>
        <p>系统运行状态监控与管理</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="showLogDialog = true" :icon="Document" round>
          运行日志
        </el-button>
      </div>
    </div>

    <div class="stats-section">
      <div class="stats-row">
        <QueueStatsCard />
        <HourlyStatsChart />
      </div>
    </div>

    <AnnouncementCard />

    <div class="info-section">
      <div class="info-grid">
        <VersionManager />

        <div class="info-card sponsor-card">
          <div class="info-card-header">
            <span class="info-icon">☕</span>
            <span>支持作者</span>
          </div>
          <div class="sponsor-content">
            <img src="https://s.mqfamily.top/alipay_wechat.jpg" alt="请作者喝杯咖啡" />
          </div>
        </div>

        <div class="info-card notice-card">
          <div class="info-card-header">
            <span class="info-icon">📝</span>
            <span>使用须知</span>
          </div>
          <div class="notice-list">
            <div class="notice-item notice-important">
              <span class="notice-number">1</span>
              <span>本项目使用115开放平台，QPS受限，介意勿用</span>
            </div>
            <div class="notice-item">
              <span class="notice-number">2</span>
              <span>播放、下载、媒体提取等操作并发总和勿超5</span>
            </div>
            <div class="notice-item">
              <span class="notice-number">3</span>
              <span>神医助手线程数建议调整为1或2</span>
            </div>
            <div class="notice-item">
              <span class="notice-number">4</span>
              <span>刮削和STRM同步为独立功能</span>
            </div>
            <div class="notice-item">
              <span class="notice-number">5</span>
              <span>问题请在
                <a href="https://github.com/qicfan/qmediasync" target="_blank">GitHub</a> 提交issue
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- 日志查看弹窗 -->
  <el-dialog v-model="showLogDialog" title="运行日志" class="log-dialog" :fullscreen="true" :close-on-click-modal="true"
    :close-on-press-escape="true" show-close="true" :destroy-on-close="true" @close="handleLogDialogClose">
    <div class="log-dialog-content">
      <AppLogViewer ref="logViewerRef" log-path="app.log" :is-real-time="true" />
    </div>
  </el-dialog>
</template>

<style scoped>
.home-container {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 0;
}

.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px;
  color: white;
}

.header-title h1 {
  margin: 0 0 4px 0;
  font-size: 28px;
  font-weight: 700;
}

.header-title p {
  margin: 0;
  font-size: 14px;
  opacity: 0.9;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.stats-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.stats-row {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: 20px;
}

.info-section {
  display: flex;
  flex-direction: column;
}

.info-grid {
  display: grid;
  grid-template-columns: 280px 280px 1fr;
  gap: 20px;
}

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

.sponsor-content {
  display: flex;
  justify-content: center;
}

.sponsor-content img {
  max-width: 100%;
  border-radius: 8px;
}

.notice-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.notice-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}

.notice-item.notice-important {
  color: #c62828;
}

.notice-number {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  background: #f0f0f0;
  border-radius: 50%;
  font-size: 11px;
  font-weight: 600;
  color: #606266;
  flex-shrink: 0;
}

.notice-item.notice-important .notice-number {
  background: #ffebee;
  color: #c62828;
}

.notice-item a {
  color: #409eff;
  text-decoration: none;
}

.notice-item a:hover {
  text-decoration: underline;
}

.log-dialog {
  display: flex;
  align-items: center;
  justify-content: center;
}

.log-dialog-content {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.log-dialog-content :deep(.el-dialog__body) {
  padding: 0;
  overflow: hidden;
  height: calc(100% - 60px);
}

.log-dialog-content :deep(.el-dialog__header) {
  padding: 10px 20px;
  border-bottom: 1px solid #ebeef5;
}

@media (max-width: 1200px) {
  .stats-row {
    grid-template-columns: 1fr;
  }

  .info-grid {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 768px) {
  .header-section {
    flex-direction: column;
    gap: 16px;
    text-align: center;
    padding: 16px;
  }

  .header-title h1 {
    font-size: 24px;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }
}
</style>
