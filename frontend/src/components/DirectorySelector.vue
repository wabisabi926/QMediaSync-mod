<template>
  <div class="directory-selector">
    <div v-loading="loading" class="tree-container">
      <div v-if="treeData.length === 0" class="empty-state">
        <el-empty description="暂无目录" />
      </div>
      <div v-else>
        <TreeNode
          v-for="node in treeData"
          :key="node.id"
          :node="node"
          :selected-id="selectedDir?.id"
          :source-type="sourceType"
          :account-id="accountId"
          @select="handleNodeSelect"
          @toggle="handleToggle"
        />
      </div>
    </div>
    <div class="footer-buttons">
      <el-button @click="$emit('create')">新建文件夹</el-button>
      <el-button @click="handleCancel">取消</el-button>
      <el-button type="primary" @click="handleButtonSelect" :disabled="!selectedDir">
        选择
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, inject, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { AxiosStatic } from 'axios'
import type { DirInfo } from '@/typing'
import TreeNode from './TreeNode.vue'
import { SERVER_URL } from '@/const'

interface Props {
  modelValue?: DirInfo | null
  rootPath?: string
  sourceType: string
  accountId?: number
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: null,
  rootPath: '',
  accountId: 0,
})

const emit = defineEmits<{
  'update:modelValue': [value: DirInfo | null]
  create: []
  cancel: []
  select: []
  reset: []
}>()

const http: AxiosStatic | undefined = inject('$http')

interface TreeNodeData extends DirInfo {
  expanded?: boolean
  loading?: boolean
  children?: TreeNodeData[]
  hasChildren?: boolean
}

const treeData = ref<TreeNodeData[]>([])
const selectedDir = ref<DirInfo | null>(props.modelValue)
const loading = ref(false)

const loadSubdirectories = async (parentNode: TreeNodeData): Promise<TreeNodeData[]> => {
  try {
    const response = await http?.get(`${SERVER_URL}/path/list`, {
      timeout: 60000,
      params: {
        parent_id: parentNode.id || '',
        parent_path: parentNode.path || '',
        source_type: props.sourceType,
        account_id: props.accountId || 0,
      },
    })

    if (response?.data.code === 200) {
      const subdirs = (response.data.data || []) as DirInfo[]
      return subdirs.map(dir => ({
        ...dir,
        expanded: false,
        loading: false,
        children: [],
        hasChildren: true,
      }))
    } else {
      ElMessage.error(response?.data.message || '加载子目录失败')
      return []
    }
  } catch {
    console.error('加载子目录错误')
    ElMessage.error('加载子目录失败')
    return []
  }
}

const handleToggle = async (node: TreeNodeData) => {
  if (node.expanded) {
    node.expanded = false
  } else {
    if (!node.children || node.children.length === 0) {
      node.loading = true
      const children = await loadSubdirectories(node)
      node.children = children
      node.loading = false
    }
    node.expanded = true
  }
}

const handleNodeSelect = (node: TreeNodeData) => {
  selectedDir.value = {
    id: node.id,
    name: node.name,
    path: node.path,
  }
  emit('update:modelValue', selectedDir.value)
}

const handleCancel = () => {
  resetState()
  emit('cancel')
}

const handleButtonSelect = () => {
  emit('select')
  resetState()
}

const resetState = () => {
  selectedDir.value = null
  emit('update:modelValue', null)
  loadRootDirectories()
  emit('reset')
}

const loadRootDirectories = async () => {
  loading.value = true
  try {
    const response = await http?.get(`${SERVER_URL}/path/list`, {
      timeout: 60000,
      params: {
        parent_id: '',
        parent_path: props.rootPath || '',
        source_type: props.sourceType,
        account_id: props.accountId || 0,
      },
    })

    if (response?.data.code === 200) {
      const dirs = (response.data.data || []) as DirInfo[]
      treeData.value = dirs.map(dir => ({
        ...dir,
        expanded: false,
        loading: false,
        children: [],
        hasChildren: true,
      }))
    } else {
      ElMessage.error(response?.data.message || '加载目录失败')
      treeData.value = []
    }
  } catch {
    console.error('加载目录树错误')
    ElMessage.error('加载目录失败')
    treeData.value = []
  } finally {
    loading.value = false
  }
}

watch(() => props.sourceType, () => {
  loadRootDirectories()
})

watch(() => props.accountId, () => {
  loadRootDirectories()
})

watch(() => props.rootPath, () => {
  loadRootDirectories()
})

watch(() => props.modelValue, (newValue) => {
  selectedDir.value = newValue
})

onMounted(() => {
  loadRootDirectories()
})

defineExpose({
  refresh: loadRootDirectories,
})
</script>

<style scoped>
.directory-selector {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 16px;
}

.tree-container {
  flex: 1;
  min-height: 250px;
  overflow-y: auto;
}

.empty-state {
  padding: 40px 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.footer-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 12px;
  border-top: 1px solid #ebeef5;
}
</style>
