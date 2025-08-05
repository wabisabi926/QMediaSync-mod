<template>
  <div class="sync-task-detail-container">
    <!-- 任务详情卡片 -->
    <el-card class="task-detail-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-button type="primary" @click="goBack" size="small" link>
              <el-icon><ArrowLeft /></el-icon>
              返回同步记录
            </el-button>
            <h2 class="card-title">同步任务详情 #{{ taskId }}</h2>
            <p class="card-subtitle">查看同步任务的执行情况和详细信息</p>
          </div>
        </div>
      </template>

      <div class="task-content">
        <!-- 任务基本信息 -->
        <div class="task-info" v-loading="infoLoading">
          <h3>任务信息</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="任务ID">{{ taskId }}</el-descriptions-item>
            <el-descriptions-item label="任务状态">
              <el-tag v-if="taskInfo" :type="getStatusType(taskInfo.status)">
                {{ getStatusText(taskInfo.status) }}
              </el-tag>
              <span v-else>-</span>
            </el-descriptions-item>
            <el-descriptions-item label="开始时间">
              {{ taskInfo?.start_time ? formatDateTime(taskInfo.start_time) : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="结束时间">
              {{ taskInfo?.end_time ? formatDateTime(taskInfo.end_time) : '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="处理文件数">
              {{ taskInfo?.processed_files || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="生成STRM">
              {{ taskInfo?.created_strm || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="下载元数据数">
              {{ taskInfo?.downloaded_meta || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="执行时长">
              {{ getExecutionDuration() }}
            </el-descriptions-item>
          </el-descriptions>
        </div>

        <!-- 文件对比树 -->
        <div class="file-compare-tree">
          <h3>文件对比</h3>
          <div class="tree-container" v-loading="logsLoading">
            <el-tree
              :data="compareData"
              :props="treeProps"
              :render-content="renderContent"
              class="compare-tree"
              default-expand-all
              empty-text="暂无对比数据"
              max-height="500"
            />
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { ArrowLeft } from '@element-plus/icons-vue'
import { inject, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

// 与同步任务列表接口保持一致的数据结构
interface ApiSyncRecord {
  id: number
  created_at: number
  finish_at: number | null
  status: number
  total: number
  new_strm: number
  downloaded_meta: number
}

// 任务详情数据结构
interface TaskInfo {
  id: number
  start_time: number
  end_time: number | null
  status: 0 | 1 | 2 | 3 // 0-待开始，1-运行中，2-完成，3-失败
  processed_files: number
  created_strm: number
  downloaded_meta: number
}

// 定义文件对比项接口
interface CompareItem {
  id: string
  name: string
  type: 'file' | 'directory'
  cloudFile?: { path: string; exists: boolean }
  localFile?: { path: string; exists: boolean }
  status: 'add' | 'delete' | 'nochange'
  children?: CompareItem[]
}

const http: AxiosStatic | undefined = inject('$http')
const route = useRoute()
const router = useRouter()

// 获取任务ID
const taskId = ref(route.params.id as string)

// 数据状态
const taskInfo = ref<TaskInfo | null>(null)
const compareData = ref<CompareItem[]>([])
const infoLoading = ref(false)
const logsLoading = ref(false)

// 返回上一页
const goBack = () => {
  router.push({ name: 'sync-records' })
}

// 获取状态标签类型
const getStatusType = (status: number) => {
  switch (status) {
    case 0:
      return 'info' // 待开始
    case 1:
      return 'primary' // 运行中
    case 2:
      return 'success' // 完成
    case 3:
      return 'danger' // 失败
    default:
      return 'info'
  }
}

// 获取状态文本
const getStatusText = (status: number) => {
  switch (status) {
    case 0:
      return '待开始'
    case 1:
      return '运行中'
    case 2:
      return '已完成'
    case 3:
      return '失败'
    default:
      return '未知'
  }
}

// 格式化时间戳为 YYYY-MM-DD HH:mm:ss 格式
const formatDateTime = (timestamp: number) => {
  if (!timestamp) return '-'
  const date = new Date(timestamp * 1000) // 时间戳转毫秒
  return date
    .toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    })
    .replace(/\//g, '-')
}

// 计算执行时长
const getExecutionDuration = () => {
  if (!taskInfo.value?.start_time) return '-'

  const startTime = taskInfo.value.start_time
  const endTime = taskInfo.value.end_time || Math.floor(Date.now() / 1000)
  const duration = endTime - startTime

  if (duration < 60) {
    return `${duration}秒`
  } else if (duration < 3600) {
    const minutes = Math.floor(duration / 60)
    const seconds = duration % 60
    return `${minutes}分${seconds}秒`
  } else {
    const hours = Math.floor(duration / 3600)
    const minutes = Math.floor((duration % 3600) / 60)
    return `${hours}小时${minutes}分`
  }
}

// 树状图配置
const treeProps = {
  children: 'children',
  label: 'name',
}

// 渲染树节点内容
const renderContent = (h: any, { node, data }: any) => {
  return h('div', { class: 'tree-node-content' }, [
    h('div', { class: 'node-name' }, [h('span', data.name)]),
    h('div', { class: 'node-cloud' }, data.cloudFile?.exists ? data.cloudFile.path : '-'),
    h('div', { class: 'node-local' }, data.localFile?.exists ? data.localFile.path : '-'),
    h('div', { class: 'node-status' }, [
      h(
        'el-tag',
        {
          props: {
            type: getCompareStatusType(data.status),
          },
          attrs: { size: 'small' },
        },
        getCompareStatusText(data.status),
      ),
    ]),
  ])
}

// 获取对比状态标签类型
const getCompareStatusType = (status: string) => {
  switch (status) {
    case 'add':
      return 'success'
    case 'delete':
      return 'danger'
    case 'nochange':
      return 'info'
    default:
      return 'info'
  }
}

// 获取对比状态文本
const getCompareStatusText = (status: string) => {
  switch (status) {
    case 'add':
      return '新增'
    case 'delete':
      return '删除'
    case 'nochange':
      return '不处理'
    default:
      return '未知'
  }
}

// 加载任务信息
const loadTaskInfo = async () => {
  try {
    infoLoading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/task/${taskId.value}`)

    if (response?.data.code === 200) {
      const data = response.data.data
      taskInfo.value = {
        id: data.id,
        start_time: data.created_at,
        end_time: data.finish_at,
        status: data.status as 0 | 1 | 2 | 3,
        processed_files: data.total,
        created_strm: data.new_strm,
        downloaded_meta: data.downloaded_meta || 0,
      }
    }
  } catch (error) {
    console.error('加载任务信息错误:', error)
  } finally {
    infoLoading.value = false
  }
}

// 加载文件对比数据
const loadCompareData = async () => {
  try {
    logsLoading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/task/${taskId.value}/compare`)

    if (response?.data.code === 200) {
      // 处理API返回的数据，确保符合CompareItem结构
      compareData.value = response.data.data.map((item: any) => ({
        id: item.id,
        name: item.name,
        type: item.type,
        cloudFile: item.cloud_file
          ? {
              path: item.cloud_file.path,
              exists: item.cloud_file.exists,
            }
          : undefined,
        localFile: item.local_file
          ? {
              path: item.local_file.path,
              exists: item.local_file.exists,
            }
          : undefined,
        status: item.status,
        children: item.children
          ? item.children.map((child: any) => ({
              id: child.id,
              name: child.name,
              type: child.type,
              cloudFile: child.cloud_file
                ? {
                    path: child.cloud_file.path,
                    exists: child.cloud_file.exists,
                  }
                : undefined,
              localFile: child.local_file
                ? {
                    path: child.local_file.path,
                    exists: child.local_file.exists,
                  }
                : undefined,
              status: child.status,
              children: child.children
                ? child.children.map((subchild: any) => ({
                    id: subchild.id,
                    name: subchild.name,
                    type: subchild.type,
                    cloudFile: subchild.cloud_file
                      ? {
                          path: subchild.cloud_file.path,
                          exists: subchild.cloud_file.exists,
                        }
                      : undefined,
                    localFile: subchild.local_file
                      ? {
                          path: subchild.local_file.path,
                          exists: subchild.local_file.exists,
                        }
                      : undefined,
                    status: subchild.status,
                  }))
                : undefined,
            }))
          : undefined,
      }))
    } else {
      compareData.value = []
    }
  } catch (error) {
    console.error('加载文件对比数据错误:', error)
    compareData.value = []
  } finally {
    logsLoading.value = false
  }
}

// 页面挂载时加载数据
onMounted(() => {
  loadTaskInfo()
  loadCompareData()
})
</script>

<style scoped>
.sync-task-detail-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.task-detail-card {
  width: 100%;
  max-width: none;
  margin: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
}

.header-left {
  flex: 1;
}

.card-title {
  margin: 8px 0 8px 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.card-subtitle {
  margin: 0;
  font-size: 14px;
  color: #909399;
}

.task-content {
  margin-top: 20px;
}

.task-info {
  margin-bottom: 30px;
}

.task-info h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.file-compare-tree h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.tree-container {
  width: 100%;
  overflow-x: auto;
}

.compare-tree {
  width: 100%;
  min-width: 800px;
}

.tree-node-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 4px 0;
}

.node-name {
  flex: 2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding-right: 10px;
}

.node-cloud,
.node-local {
  flex: 3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding-right: 10px;
  color: #606266;
}

.node-status {
  flex: 1;
  text-align: center;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .card-title {
    font-size: 20px;
  }

  .card-subtitle {
    font-size: 13px;
  }

  .task-info h3,
  .task-logs h3 {
    font-size: 16px;
  }

  .logs-table {
    font-size: 12px;
  }

  .logs-pagination {
    flex-wrap: wrap;
    gap: 8px;
  }
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .card-title {
    font-size: 18px;
  }
}
</style>
