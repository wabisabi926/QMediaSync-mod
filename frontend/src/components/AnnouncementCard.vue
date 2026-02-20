<script setup lang="ts">
import { ref } from 'vue'
import { useAnnouncement } from '@/composables/useAnnouncement'

const { announcementList, announcementLoading } = useAnnouncement()

const expandedItems = ref<Set<number>>(new Set())

const isContentLong = (content: string): boolean => {
  const lines = content.split('\n')
  return lines.length > 3 || content.length > 150
}

const toggleExpand = (index: number) => {
  if (expandedItems.value.has(index)) {
    expandedItems.value.delete(index)
  } else {
    expandedItems.value.add(index)
  }
}

const isExpanded = (index: number): boolean => {
  return expandedItems.value.has(index)
}

const getDisplayContent = (content: string, index: number): string => {
  if (isExpanded(index)) {
    return content
  }
  const lines = content.split('\n')
  if (lines.length > 3) {
    return lines.slice(0, 3).join('\n') + '...'
  }
  if (content.length > 150) {
    return content.substring(0, 150) + '...'
  }
  return content
}
</script>

<template>
  <div class="announcement-section" v-loading="announcementLoading">
    <div class="section-header">
      <div class="section-title">
        <span class="title-icon">📢</span>
        <span>公告</span>
      </div>
    </div>

    <div v-if="announcementList.length > 0" class="announcement-list">
      <div 
        v-for="(announcement, index) in announcementList" 
        :key="announcement.id || index" 
        class="announcement-item"
      >
        <div class="announcement-header">
          <div class="announcement-title">{{ announcement.title }}</div>
          <div class="announcement-time">{{ announcement.time }}</div>
        </div>
        <div class="announcement-content">
          <div 
            class="content-text" 
            :class="{ 'is-expanded': isExpanded(index) }"
          >
            {{ getDisplayContent(announcement.content, index) }}
          </div>
          <el-button 
            v-if="isContentLong(announcement.content)" 
            type="primary" 
            link 
            size="small"
            @click="toggleExpand(index)"
          >
            {{ isExpanded(index) ? '收起' : '展开' }}
          </el-button>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      <el-empty description="暂无公告" :image-size="60" />
    </div>
  </div>
</template>

<style scoped>
.announcement-section {
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

.announcement-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.announcement-item {
  padding: 16px;
  background: #f8f9fa;
  border-radius: 12px;
  transition: all 0.3s ease;
}

.announcement-item:hover {
  background: #f0f2f5;
}

.announcement-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.announcement-time {
  font-size: 12px;
  color: #909399;
  white-space: nowrap;
}

.announcement-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  flex: 1;
}

.announcement-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.announcement-content .content-text {
  font-size: 14px;
  color: #606266;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
}
</style>
