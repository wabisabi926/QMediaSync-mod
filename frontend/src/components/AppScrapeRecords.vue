<template>
  <div class="scrape-records-container">
    <h2 class="page-title">刮削记录</h2>
    <p  class="card-subtitle" style="margin-bottom:16px;">当前刮削产生的临时文件存放在 <span style="color:#000; font-weight:bold;">config/tmp/刮削临时文件/</span> 目录下,可以观察是否存在异常情况，刮削完成的文件会删除临时文件</p>
    <!-- 多选操作栏 -->
    <div v-if="selectedRecords.length > 0" class="batch-operations">
      <el-button type="primary" @click="handleExportErrors">导出识别错误文件</el-button>
      <span class="selected-count">已选择 {{ selectedRecords.length }} 条记录</span>
    </div>

    <!-- 搜索和过滤区域 -->
    <div style="margin-bottom: 16px;">
      <el-button type="primary" @click="toggleMergeEpisodes">
        {{ isMerged ? '显示电视剧集' : '合并电视剧集' }}
      </el-button>
    </div>
    <div class="search-filter-section">
      <el-select v-model="statusFilter" placeholder="筛选状态" style="margin-left: 12px; width: 150px;">
        <el-option label="全部" value=""></el-option>
        <el-option label="未刮削" value="scanned"></el-option>
        <el-option label="刮削中" value="scraping"></el-option>
        <el-option label="已刮削" value="scraped"></el-option>
        <el-option label="刮削失败" value="scrape_failed"></el-option>
        <el-option label="整理中" value="renaming"></el-option>
        <el-option label="已整理" value="renamed"></el-option>
      </el-select>
      <el-select v-model="typeFilter" placeholder="筛选类型" style="margin-left: 12px; width: 120px;">
        <el-option label="全部" value=""></el-option>
        <el-option label="电影" value="movie"></el-option>
        <el-option label="电视剧" value="tvshow"></el-option>
        <el-option label="其他" value="other"></el-option>
      </el-select>
      <el-button type="primary" @click="applyFilter" style="margin-left: 12px;">
        筛选
      </el-button>
      <el-button @click="resetFilter" style="margin-left: 8px;">
        重置
      </el-button>
    </div>

    <!-- 表格 -->
    <div class="table-container">
      <el-table
        v-loading="loading"
        :data="records"
        style="width: 100%"
        @selection-change="handleSelectionChange"
        :row-key="(row: ScrapeRecord) => row.id"
        height="calc(100vh - 300px)"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="type" label="类型" width="80">
          <template #default="{ row }">
            <el-tag :type="getTypeTagType(row.type)">
              {{ getTypeName(row.type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="文件路径" min-width="300" show-overflow-tooltip />
        <el-table-column prop="file_name" label="文件名" min-width="200" show-overflow-tooltip />
        <el-table-column prop="media_name" label="识别名" width="180" />
        <el-table-column prop="year" label="识别年" width="80" />
        <el-table-column prop="tmdb_id" label="TMDB" width="80" />
        <el-table-column prop="season_number" label="季集" width="80">
          <template #default="{ row }">
            <span v-if="row.type == 'tvshow'">
              S{{ row.season_number }}E{{ row.episode_number }}
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)">
              {{ getStatusName(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="error_reason" label="失败原因" min-width="200">
          <template #default="{ row }">
            <el-popover placement="top" :width="400" trigger="hover" v-if="row.error_reason">
              <template #reference>
                <span class="error-reason-text">{{ truncateText(row.error_reason, 20) }}</span>
              </template>
              <pre style="margin: 0; white-space: pre-wrap; word-break: break-all;">{{ row.error_reason }}</pre>
            </el-popover>
            <span v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button type="text" @click="handleDetail(row)">详情</el-button>
            <el-button type="warning" size="small" @click="reScrape(row)" v-if="row.status == 'scrape_failed' || row.status == 'scanned'">重新识别</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.currentPage"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[100, 200, 500]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>

    <!-- 详情对话框 -->
    <el-dialog v-model="showDetailDialog" title="刮削详情" width="600px">
      <div v-if="selectedRecord" class="detail-content">
        <el-descriptions border direction="vertical">
          <el-descriptions-item
            :rowspan="2"
            label="封面"
          >
            <el-image
              style="width: 100px;"
              :src="SERVER_URL + selectedRecord.poster"
            />
          </el-descriptions-item>
          <el-descriptions-item label="文件路径">
            <el-tooltip :content="selectedRecord.path" placement="top">
              <pre style="margin: 0; white-space: pre-wrap; word-break: break-all; max-height: 100px; overflow: auto;">{{ selectedRecord.path }}</pre>
            </el-tooltip>
          </el-descriptions-item>
          <el-descriptions-item label="文件名">
            <el-tooltip :content="selectedRecord.file_name" placement="top">
              <pre style="margin: 0; white-space: pre-wrap; word-break: break-all; max-height: 100px; overflow: auto;">{{ selectedRecord.file_name }}</pre>
            </el-tooltip>
          </el-descriptions-item>
          <el-descriptions-item label="影视剧名称">{{ selectedRecord.media_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="年份">{{ selectedRecord.year || '-' }}</el-descriptions-item>
          <el-descriptions-item label="类型">{{ getTypeName(selectedRecord.type) }}</el-descriptions-item>
          <el-descriptions-item label="TMDBID">{{ selectedRecord.tmdb_id || '-' }}</el-descriptions-item>
          <el-descriptions-item label="季集">
            <span v-if="selectedRecord.type == 'tvshow'">
              S{{ selectedRecord.season_number }}E{{ selectedRecord.episode_number }}
            </span>
            <span v-else>-</span>
          </el-descriptions-item>
          <el-descriptions-item label="集名称">{{ selectedRecord.episode_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="音轨数量">{{ selectedRecord.audio_count || '-' }}</el-descriptions-item>
          <el-descriptions-item label="字幕数量">{{ selectedRecord.subtitle_count || '-' }}</el-descriptions-item>
          <el-descriptions-item label="分辨率">{{ selectedRecord.resolution || '-' }}</el-descriptions-item>
          <el-descriptions-item label="分辨率等级">{{ selectedRecord.resolution_level || '-' }}</el-descriptions-item>
          <el-descriptions-item label="是否HDR">{{ selectedRecord.is_hdr ? '是' : '否' }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ getStatusName(selectedRecord.status) }}</el-descriptions-item>
          <el-descriptions-item label="识别时间">{{ formatTimestamp(selectedRecord.scanned_at) }}</el-descriptions-item>
          <el-descriptions-item label="刮削时间">{{ formatTimestamp(selectedRecord.scraped_at) }}</el-descriptions-item>
          <el-descriptions-item label="失败原因">
            <pre v-if="selectedRecord.error_reason" style="margin: 0; white-space: pre-wrap; word-break: break-all; max-height: 100px; overflow: auto;">{{ selectedRecord.error_reason }}</pre>
            <span v-else>-</span>
          </el-descriptions-item>

        </el-descriptions>
      </div>
    </el-dialog>

    <!-- 重新识别对话框 -->
    <el-dialog v-model="showReScrapeDialog" title="重新识别" width="500px">
      <el-form label-width="80px">
        <el-form-item label="原始文件名">
          <el-input v-model="reScrapeForm.originalFileName" placeholder="原始文件名" readonly />
        </el-form-item>
        <el-form-item label="文件名称">
          <el-input
            v-model="reScrapeForm.name"
            placeholder="请输入影视剧名称"
          />
        </el-form-item>
        <el-form-item label="年份">
          <el-input
            v-model="reScrapeForm.year"
            placeholder="请输入年份"
            type="number"
            :min="1900"
            :max="new Date().getFullYear() + 5"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showReScrapeDialog = false">取消</el-button>
          <el-button type="primary" @click="submitReScrape" :loading="reScrapeLoading">确认重新识别</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject } from 'vue'
import { formatTimestamp } from '@/utils/timeUtils'

// 获取HTTP客户端
const http: AxiosStatic | undefined = inject('$http')

// 定义刮削记录接口
interface ScrapeRecord {
  id: number
  type: 'movie' | 'tvshow' | 'other'
  path: string
  file_name: string
  media_name?: string
  year?: number
  tmdb_id?: string
  season_number?: string
  episode_number?: string
  episode_name?: string
  status: 'scanned' | 'scraping' | 'scraped' | 'scrape_failed' | 'renaming' | 'renamed'
  error_reason?: string
  created_at: number
  updated_at: number
  scanned_at: number
  scraped_at: number
  audio_count?: number
  subtitle_count?: number
  resolution: string
  resolution_level: string
  is_hdr: boolean
  poster?: string
  fanart?: string
  logo?: string
  episode_poster?: string
  season_poster?: string
}

// 状态变量
const records = ref<ScrapeRecord[]>([])
const originalRecords = ref<ScrapeRecord[]>([])
const isMerged = ref(false)
const loading = ref(false)
const selectedRecords = ref<ScrapeRecord[]>([])
const statusFilter = ref('')
const typeFilter = ref('')
const showDetailDialog = ref(false)
const selectedRecord = ref<ScrapeRecord | null>(null)
// 添加自动刷新相关变量
const autoRefreshTimer = ref<number | null>(null)

// 分页相关
const pagination = ref({
  currentPage: 1,
  pageSize: 100
})
const total = ref(0)

// 加载刮削记录
const loadRecords = async () => {
  try {
    loading.value = true
    // 构建查询参数
    const params: Record<string, string|number> = {
      page: pagination.value.currentPage,
      pageSize: pagination.value.pageSize
    }

    // 根据需求，将statusFilter映射到media_type参数
    if (statusFilter.value) {
      params.status = statusFilter.value
    }

    // 添加类型筛选参数
    if (typeFilter.value) {
      params.type = typeFilter.value
    }

    const response = await http?.get(`${SERVER_URL}/scrape/records`, { params })

    if (response?.data.code === 200) {
      records.value = response.data.data.list
      originalRecords.value = JSON.parse(JSON.stringify(response.data.data.list)) // 深拷贝保存原始记录
      total.value = response.data.data.total
      console.log(total, response.data.data.total)

      // 检查是否有刮削中的记录，如果有则启动自动刷新
      checkAndSetAutoRefresh()
    } else {
      ElMessage.error(`加载刮削记录失败: ${response?.data.message || '未知错误'}`)
    }
  } catch (error) {
    console.error('加载刮削记录失败:', error)
    ElMessage.error('加载刮削记录失败: 网络错误')
  } finally {
    loading.value = false
  }
}

// 合并/显示电视剧集
const toggleMergeEpisodes = () => {
  if (!isMerged.value) {
    // 合并电视剧集：根据tmdb_id和season_number分组，相同值只保留一份
    const mergedMap = new Map<string, ScrapeRecord>()

    records.value.forEach(record => {
      if (record.type === 'tvshow' && record.tmdb_id && record.season_number) {
        const key = `${record.tmdb_id}-${record.season_number}`
        if (!mergedMap.has(key)) {
          mergedMap.set(key, record)
        }
      } else {
        // 非电视剧或信息不完整的记录直接保留
        mergedMap.set(`unique-${record.id}`, record)
      }
    })

    records.value = Array.from(mergedMap.values())
    isMerged.value = true
  } else {
    // 还原回原始列表
    records.value = JSON.parse(JSON.stringify(originalRecords.value))
    isMerged.value = false
  }
}

// 检查并设置自动刷新
const checkAndSetAutoRefresh = () => {
  // 清除已存在的定时器
  if (autoRefreshTimer.value) {
    clearInterval(autoRefreshTimer.value)
    autoRefreshTimer.value = null
  }

  // 检查是否有刮削中的记录
  const hasScrapingRecords = records.value.some(record => record.status === 'scraping')

  if (hasScrapingRecords) {
    // 设置定时器，每隔1秒刷新一次
    autoRefreshTimer.value = window.setInterval(() => {
      loadRecords()
    },5000)
  }
}

// 组件卸载时清除定时器
onUnmounted(() => {
  if (autoRefreshTimer.value) {
    clearInterval(autoRefreshTimer.value)
  }
})

// 应用筛选
const applyFilter = () => {
  pagination.value.currentPage = 1 // 重置为第一页
  loadRecords() // 重新加载数据
  // 重置合并状态
  if (isMerged.value) {
    isMerged.value = false
  }
}

// // 处理搜索
// const handleSearch = () => {
//   // 搜索逻辑
//   if (searchKeyword.value) {
//     // 如果有搜索关键词，使用本地过滤
//     if (isMerged.value) {
//       // 如果处于合并状态，先还原再过滤
//       records.value = JSON.parse(JSON.stringify(originalRecords.value))
//     }

//     const keyword = searchKeyword.value.toLowerCase()
//     records.value = originalRecords.value.filter(record =>
//       (record.file_name && record.file_name.toLowerCase().includes(keyword)) ||
//       (record.path && record.path.toLowerCase().includes(keyword))
//     )
//   } else {
//     // 如果没有搜索关键词，重新加载
//     if (isMerged.value) {
//       // 如果处于合并状态，保持合并状态
//       toggleMergeEpisodes()
//       toggleMergeEpisodes()
//     } else {
//       loadRecords()
//     }
//   }
// }

// 重置筛选
const resetFilter = () => {
  statusFilter.value = ''
  typeFilter.value = ''
  pagination.value.currentPage = 1
  loadRecords() // 重新加载数据
}

// 分页处理
const handleSizeChange = (size: number) => {
  pagination.value.pageSize = size
  loadRecords() // 重新加载数据
}

const handleCurrentChange = (current: number) => {
  pagination.value.currentPage = current
  loadRecords() // 重新加载数据
}

// 处理选择变化
const handleSelectionChange = (selection: ScrapeRecord[]) => {
  selectedRecords.value = selection
}

// 导出识别错误文件
const handleExportErrors = async () => {
  try {
    // 移除筛选条件，导出所有已选择的记录
    if (selectedRecords.value.length === 0) {
      ElMessage.warning('请选择记录')
      return
    }

    const ids = selectedRecords.value.map(record => record.id)
    // 构造URL，将ids作为GET参数传递
    const idsQuery = ids.join(',')
    const downloadUrl = `${SERVER_URL}/scrape/records/export?ids=${idsQuery}`

    // 在新窗口打开下载
    window.open(downloadUrl, '_blank')
    ElMessage.success('导出请求已发送')
  } catch (error) {
    console.error('导出失败:', error)
    ElMessage.error('导出失败: 网络错误')
  }
}

// 查看详情
const handleDetail = (record: ScrapeRecord) => {
  selectedRecord.value = record
  showDetailDialog.value = true
}

// 重识别相关变量
const showReScrapeDialog = ref(false)
const reScrapeForm = ref({
  id: 0,
  name: '',
  year: '',
  originalFileName: ''
})
const reScrapeLoading = ref(false)

// 处理重新识别
const reScrape = (record: ScrapeRecord) => {
  // 初始化表单数据
  reScrapeForm.value = {
    id: record.id,
    name: record.media_name || '',
    year: record.year?.toString() || '',
    originalFileName: record.file_name || ''
  }
  // 打开对话框
  showReScrapeDialog.value = true
}

// 提交重新识别请求
const submitReScrape = async () => {
  try {
    reScrapeLoading.value = true

    // 准备请求参数
    const params = {
      id: reScrapeForm.value.id,
      name: reScrapeForm.value.name,
      year: reScrapeForm.value.year ? parseInt(reScrapeForm.value.year) : undefined
    }

    // 发送请求
    const response = await http?.post(`${SERVER_URL}/scrape/re-scrape`, params)

    if (response?.data.code === 200) {
      ElMessage.success('重新识别请求已发送')
      showReScrapeDialog.value = false
      // 刷新记录列表
      loadRecords()
    } else {
      ElMessage.error(`重新识别失败: ${response?.data.message || '未知错误'}`)
    }
  } catch (error) {
    console.error('重新识别失败:', error)
    ElMessage.error('重新识别失败: 网络错误')
  } finally {
    reScrapeLoading.value = false
  }
}

const truncateText = (text: string, maxLength: number): string => {
  return text.length > maxLength ? text.substring(0, maxLength) + '...' : text
}

// 获取类型标签类型
const getTypeTagType = (type: string): string => {
  switch (type) {
    case 'movie':
      return 'primary'
    case 'tvshow':
      return 'success'
    default:
      return 'info'
  }
}

// 获取类型名称
const getTypeName = (type: string): string => {
  switch (type) {
    case 'movie':
      return '电影'
    case 'tvshow':
      return '电视剧'
    default:
      return '其他'
  }
}

// 获取状态标签类型
const getStatusTagType = (status: string): string => {
  switch (status) {
    case 'scanned':
      return 'info'
    case 'scraping':
      return 'warning'
    case 'scraped':
      return 'success'
    case 'scrape_failed':
      return 'danger'
    case 'renaming':
      return 'warning'
    case 'renamed':
      return 'success'
    default:
      return 'info'
  }
}

// 获取状态名称
const getStatusName = (status: string): string => {
  switch (status) {
    case 'scanned':
      return '未刮削'
    case 'scraping':
      return '刮削中'
    case 'scraped':
      return '已刮削'
    case 'scrape_failed':
      return '刮削失败'
    case 'renaming':
      return '整理中'
    case 'renamed':
      return '已整理'
    default:
      return '未知'
  }
}

// 组件挂载时加载数据
onMounted(() => {
  loadRecords()
})
</script>

<style scoped>
.scrape-records-container {
  padding: 20px;
}

.page-title {
  font-size: 20px;
  font-weight: bold;
  margin-bottom: 20px;
  color: #303133;
}

.batch-operations {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px;
  background-color: #f0f9ff;
  border: 1px solid #91d5ff;
  border-radius: 4px;
}

.selected-count {
  margin-left: 16px;
  color: #606266;
}

.search-filter-section {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

.search-filter-section .el-input {
  width: 300px;
}

.table-container {
  background: #fff;
  border-radius: 4px;
  padding: 0;
}

.pagination-container {
  padding: 16px;
  display: flex;
  justify-content: flex-end;
  background: #fff;
  border-top: 1px solid #ebeef5;
  margin-top: -1px;
  border-radius: 0 0 4px 4px;
}

.error-reason-text {
  color: #f56c6c;
  cursor: pointer;
}

.detail-content {
  max-height: 400px;
  overflow-y: auto;
}

.detail-item {
  display: flex;
  gap: 10px;
  margin-bottom: 10px;
}
.detail-label{
  color: #969797;
}
.detail-value {
  font-size: 16px;
}
</style>
