<template>
  <div class="tree-node">
    <div
      class="node-content"
      :class="{ 'is-selected': nodeId === selectedId }"
      @click="handleClick"
    >
      <span
        class="expand-icon"
        :class="{ 'is-expanded': node.expanded }"
        @click.stop="handleToggle"
      >
        <el-icon>
          <ArrowRight v-if="!node.expanded" />
          <ArrowDown v-else />
        </el-icon>
      </span>
      <el-icon class="folder-icon">
        <Folder />
      </el-icon>
      <span class="node-label">{{ node.name }}</span>
      <el-icon v-if="node.loading" class="is-loading loading-icon">
        <Loading />
      </el-icon>
    </div>
    <div v-if="node.expanded && node.children && node.children.length > 0" class="node-children">
      <TreeNode
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selected-id="selectedId"
        :source-type="sourceType"
        :account-id="accountId"
        @select="$emit('select', $event)"
        @toggle="$emit('toggle', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowRight, ArrowDown, Folder, Loading } from '@element-plus/icons-vue'

interface TreeNodeData {
  id: string
  name: string
  path: string
  expanded?: boolean
  loading?: boolean
  children?: TreeNodeData[]
  hasChildren?: boolean
}

interface Props {
  node: TreeNodeData
  selectedId?: string
  sourceType: string
  accountId?: number
}

const props = defineProps<Props>()

const emit = defineEmits<{
  select: [node: TreeNodeData]
  toggle: [node: TreeNodeData]
}>()

const nodeId = computed(() => props.node.id)

const handleClick = () => {
  emit('select', props.node)
}

const handleToggle = () => {
  emit('toggle', props.node)
}
</script>

<style scoped>
.tree-node {
  user-select: none;
}

.node-content {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  cursor: pointer;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.node-content:hover {
  background-color: #f5f7fa;
}

.node-content.is-selected {
  background-color: #ecf5ff;
}

.expand-icon {
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: #909399;
  transition: transform 0.2s;
}

.expand-icon:hover {
  color: #409eff;
}

.folder-icon {
  color: #6c56bb;
}

.node-label {
  flex: 1;
  font-size: 14px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.loading-icon {
  font-size: 14px;
  color: #409eff;
}

.node-children {
  padding-left: 24px;
}
</style>
