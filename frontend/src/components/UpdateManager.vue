<script setup lang="ts">
import { ref } from 'vue'
import { useUpdate } from '@/composables/useUpdate'
import { formatFileSize } from '@/utils/fileSizeUtils'
import MarkdownIt from 'markdown-it'
import 'github-markdown-css'
import { CircleCheck } from '@element-plus/icons-vue'

const activeNames = ref<string[]>(['update-0'])

const {
  updateList,
  updateLoading,
  isUpdating,
  updatingVersion,
  updateProgress,
  showUpdateCompleteDialog,
  countdown,
  loadUpdateList,
  updateToVersion,
  cancelUpdate,
  handleDownloadClick,
  manuallyRefresh
} = useUpdate()

const md = new MarkdownIt({
  html: true,
  breaks: true,
  linkify: true
})

const renderMarkdown = (content: string): string => {
  return md.render(content || '')
}
</script>

<template>
  <div class="update-section">
    <div class="section-header">
      <div class="section-title">
        <span class="title-icon">🚀</span>
        <span>版本更新</span>
      </div>
      <el-button type="primary" size="small" @click="loadUpdateList(true)" :loading="updateLoading" round>
        刷新
      </el-button>
    </div>

    <div v-if="updateList.length > 0" class="update-list">
      <el-collapse v-model="activeNames" class="update-collapse">
        <el-collapse-item v-for="(update, index) in updateList" :key="index" :name="`update-${index}`">
          <template #title>
            <div class="update-title-row">
              <div class="update-version">
                <span class="version-number">{{ update.version }}</span>
                <span class="version-date">{{ update.date }}</span>
              </div>
              <div class="update-tags">
                <el-tag v-if="update.latest" type="success" size="small" effect="dark">最新</el-tag>
                <el-tag v-if="update.current" type="primary" size="small" effect="dark">当前</el-tag>
              </div>
            </div>
          </template>
          <div class="update-detail">
            <div class="update-note markdown-body" v-html="renderMarkdown(update.note)"></div>
            <div class="update-actions" v-if="!update.current">
              <el-button type="default" size="small" @click="handleDownloadClick(update)" round>
                手动下载
              </el-button>
              <el-button type="primary" size="small" @click="updateToVersion(update.version)" :disabled="isUpdating" round>
                在线更新
              </el-button>
            </div>

            <div v-if="isUpdating && update.version === updatingVersion" class="update-progress">
              <el-progress :percentage="updateProgress.progress" :stroke-width="8" :show-text="false" />
              <div class="progress-info">
                <span>{{ formatFileSize(updateProgress.downloaded) }} / {{ formatFileSize(updateProgress.total_size) }}</span>
                <span>{{ updateProgress.status === 'downloading' ? '下载中' : updateProgress.status === 'install' ? '安装中' : '' }}</span>
              </div>
              <el-button type="danger" size="small" @click="cancelUpdate" round>
                取消
              </el-button>
            </div>
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>

    <div v-else class="empty-state">
      <el-empty description="暂无版本信息" :image-size="80" />
    </div>
  </div>

  <!-- 更新完成弹窗 -->
  <el-dialog v-model="showUpdateCompleteDialog" title="正在安装更新" class="update-complete-dialog"
    :close-on-click-modal="false" :close-on-press-escape="false" show-close="false" :destroy-on-close="true">
    <div class="dialog-content">
      <el-icon>
        <CircleCheck />
      </el-icon>
      <h3>安装包已下载，正在更新中</h3>
      <p>系统将在 <strong>{{ countdown }}</strong> 秒后自动刷新页面</p>
      <div class="dialog-tips">
        <p>提示：刷新页面后，新版本将生效。如未生效，请手动刷新或手动下载最新版本，如果是docker可以更新镜像</p>
      </div>
    </div>
    <template #footer>
      <div class="dialog-footer">
        <el-button type="primary" @click="manuallyRefresh">
          立即刷新
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.update-section {
  background: white;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #f0f0f0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.title-icon {
  font-size: 20px;
}

.update-collapse {
  border: none;
}

.update-collapse :deep(.el-collapse-item__header) {
  background: #f8f9fa;
  border-radius: 12px;
  padding: 0 16px;
  margin-bottom: 8px;
  border: none;
  height: 56px;
}

.update-collapse :deep(.el-collapse-item__wrap) {
  border: none;
}

.update-collapse :deep(.el-collapse-item__content) {
  padding: 16px;
}

.update-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.update-version {
  display: flex;
  align-items: center;
  gap: 12px;
}

.version-number {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.version-date {
  font-size: 13px;
  color: #909399;
}

.update-tags {
  display: flex;
  gap: 8px;
}

.update-detail {
  background: #fafafa;
  border-radius: 12px;
  padding: 16px;
}

.update-note {
  background: white;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
  font-size: 14px;
  line-height: 1.6;
  color: #606266;
}

.update-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.update-progress {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.update-progress .el-progress {
  flex: 1;
}

.progress-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 12px;
  color: #909399;
  min-width: 120px;
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
}

.update-complete-dialog :deep(.el-dialog) {
  width: 500px;
  max-width: 90vw;
  border-radius: 16px;
}

.dialog-content {
  text-align: center;
  padding: 30px 20px;
}

.dialog-content .el-icon {
  font-size: 48px;
  color: #67c23a;
  margin-bottom: 20px;
}

.dialog-content h3 {
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 12px;
  color: #303133;
}

.dialog-content p {
  font-size: 15px;
  color: #606266;
  margin-bottom: 16px;
}

.dialog-tips {
  padding: 12px 16px;
  background: #f0f9ff;
  border-radius: 8px;
}

.dialog-tips p {
  font-size: 13px;
  color: #909399;
  margin: 0;
}

.dialog-footer {
  display: flex;
  justify-content: center;
  padding: 16px;
  border-top: 1px solid #ebeef5;
}

.update-note :deep(.markdown-body) {
  font-size: 14px;
  line-height: 1.6;
}

.update-note :deep(.markdown-body pre) {
  background-color: #f6f8fa;
  border-radius: 6px;
  padding: 12px;
  overflow: auto;
}

.update-note :deep(.markdown-body code) {
  background-color: #f1f1f1;
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 13px;
}

.update-note :deep(.markdown-body pre code) {
  background-color: transparent;
  padding: 0;
}

.update-note :deep(.markdown-body a) {
  color: #409eff;
  text-decoration: none;
}

.update-note :deep(.markdown-body a:hover) {
  text-decoration: underline;
}

.update-note :deep(.markdown-body ul),
.update-note :deep(.markdown-body ol) {
  padding-left: 1.5em;
  margin: 8px 0;
}

.update-note :deep(.markdown-body li) {
  margin-bottom: 4px;
}
</style>
