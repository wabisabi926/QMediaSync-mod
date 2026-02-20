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
      <el-button @click="openCreateDialog">新建文件夹</el-button>
      <el-button @click="handleCancel">取消</el-button>
      <el-button type="primary" @click="handleButtonSelect" :disabled="!selectedDir">
        选择
      </el-button>
    </div>

    <el-dialog v-model="showCreateDialog" title="新建文件夹" width="400px" :close-on-click-modal="false" append-to-body>
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="80px">
        <el-form-item label="文件夹名称" prop="name">
          <el-input v-model="createForm.name" placeholder="请输入文件夹名称" clearable />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showCreateDialog = false">取消</el-button>
          <el-button type="primary" @click="handleCreateDirectory" :loading="createLoading">
            确定
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, inject, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { AxiosStatic } from 'axios'
import type { FormInstance, FormRules } from 'element-plus'
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
  cancel: []
  select: []
  reset: []
}>()

const http: AxiosStatic | undefined = inject('$http')

const showCreateDialog = ref(false)
const createLoading = ref(false)
const createFormRef = ref<FormInstance>()
const createForm = ref({ name: '' })
const createRules = ref<FormRules>({
  name: [
    { required: true, message: '请输入文件夹名称', trigger: 'blur' },
    { min: 1, max: 255, message: '文件夹名称长度在 1 到 255 个字符', trigger: 'blur' }
  ]
})

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

const openCreateDialog = () => {
  if (!selectedDir.value) {
    ElMessage.warning('请先选择一个父目录')
    return
  }
  createForm.value.name = ''
  showCreateDialog.value = true
}

const handleCreateDirectory = async () => {
  if (!createFormRef.value) return
  if (!selectedDir.value) {
    ElMessage.warning('请先选择一个父目录')
    return
  }

  try {
    await createFormRef.value.validate()
    createLoading.value = true

    const response = await http?.post(`${SERVER_URL}/path/create`, {
      parent_id: selectedDir.value.id,
      parent_path: selectedDir.value.path,
      name: createForm.value.name.trim(),
      source_type: props.sourceType,
      account_id: props.accountId,
    })

    if (response?.data.code === 200) {
      ElMessage.success('创建文件夹成功')
      showCreateDialog.value = false
      createForm.value.name = ''

      const newDir = response.data.data as DirInfo
      selectedDir.value = newDir
      emit('update:modelValue', newDir)

      const parentPath = selectedDir.value.path || ''
      const parentPathParts = parentPath.split('/').filter(Boolean)
      if (parentPathParts.length === 0) {
        treeData.value.push({
          ...newDir,
          expanded: false,
          loading: false,
          children: [],
          hasChildren: true,
        })
      } else {
        const findAndAddToParent = (nodes: TreeNodeData[]): boolean => {
          for (const node of nodes) {
            if (node.id === selectedDir.value?.id) {
              if (!node.children) {
                node.children = []
              }
              node.children.push({
                ...newDir,
                expanded: false,
                loading: false,
                children: [],
                hasChildren: true,
              })
              node.expanded = true
              return true
            }
            if (node.children && findAndAddToParent(node.children)) {
              return true
            }
          }
          return false
        }
        findAndAddToParent(treeData.value)
      }
    } else {
      ElMessage.error(response?.data.message || '创建文件夹失败')
    }
  } catch {
    ElMessage.error('创建文件夹失败')
  } finally {
    createLoading.value = false
  }
}

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
