<template>
  <div class="scrape-records-container">
    <div class="header-section">
      <h2 class="page-title hidden-md-and-down">刮削记录</h2>

      <div class="top-actions">
        <div class="action-group">
          <div class="action-buttons">
            <el-tooltip content="将所选的识别错误的记录导出成一个文件，可以发送给作者" placement="top">
              <el-button type="primary" @click="handleExportErrors"
                :disabled="selectedRecords.length === 0">导出识别错误</el-button>
            </el-tooltip>
            <el-tooltip content="将刮削记录删除后，对应的文件在下次扫描时会再次识别和刮削。" placement="top">
              <el-button type="danger" @click="handleDeleteSelectedRecords"
                :disabled="selectedRecords.length === 0">删除所选记录</el-button>
            </el-tooltip>
            <el-tooltip content="请选择整理失败的记录，该操作会将所选记录标记为待整理，下次整理时重试。" placement="top">
              <el-button type="warning" @click="handleRename"
                :disabled="selectedRecords.length === 0">重新整理所选</el-button>
            </el-tooltip>
          </div>
          <div class="selected-count">已选择 {{ selectedRecords.length }} 条记录</div>
        </div>
        <div class="action-group">
          <div class="action-buttons">
            <el-tooltip content="会将列表中属于同一个电视剧的所有集合并成一条数据，方便查看" placement="top">
              <el-button type="primary" @click="toggleMergeEpisodes">
                {{ isMerged ? '显示电视剧集' : '合并电视剧集' }}
              </el-button>
            </el-tooltip>
            <el-button type="warning" @click="handleDeleteFailedRecords">清除失败记录</el-button>
            <el-tooltip content="删除所有刮削记录，慎用！" placement="top">
              <el-button type="danger" @click="handleTruncateAll">清空记录</el-button>
            </el-tooltip>
          </div>
        </div>
      </div>

      <div class="search-filter-section">
        <el-input v-model="nameFilter" placeholder="按文件名模糊搜索" class="filter-input" @keyup.enter="applyFilter" />
        <el-select v-model="statusFilter" placeholder="筛选状态" class="filter-select">
          <el-option label="全部" value=""></el-option>
          <el-option label="未刮削" value="scanned"></el-option>
          <el-option label="刮削中" value="scraping"></el-option>
          <el-option label="已刮削" value="scraped"></el-option>
          <el-option label="刮削失败" value="scrape_failed"></el-option>
          <el-option label="整理中" value="renaming"></el-option>
          <el-option label="已整理" value="renamed"></el-option>
          <el-option label="整理失败" value="rename_failed"></el-option>
        </el-select>
        <el-select v-model="typeFilter" placeholder="筛选类型" class="filter-select-small">
          <el-option label="全部" value=""></el-option>
          <el-option label="电影" value="movie"></el-option>
          <el-option label="电视剧" value="tvshow"></el-option>
          <el-option label="其他" value="other"></el-option>
        </el-select>
        <el-button type="primary" @click="applyFilter">筛选</el-button>
        <el-button @click="resetFilter">重置</el-button>
      </div>
    </div>

    <div class="table-section">
      <div class="table-container">
        <el-table v-loading="loading" :data="records" @selection-change="handleSelectionChange"
          :row-key="(row: ScrapeRecord) => row.id" style="width: 100%" class="mobile-table">
          <el-table-column type="selection" width="30" />
          <el-table-column type="expand" width="30">
            <template #default="{ row }">
              <el-descriptions class="margin-top" :column="2" border size="small" label-width="50px">
                <el-descriptions-item label="类型">
                  <el-tag :type="getTypeTagType(row.type)">
                    {{ getTypeName(row.type) }}
                  </el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="来源">
                  <el-tag :type="getSourceTypeTagType(row.source_type)">
                    {{ getSourceTypeName(row.source_type) }}
                  </el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="状态">
                  <el-tag :type="getStatusTagType(row.status)">
                    {{ getStatusName(row.status) }}
                  </el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="识别信息">
                  <div>
                    <p class="path-text">Tmdb ID: {{ row.tmdb_id }}</p>
                    <p class="path-text">识别名称: {{ row.media_name }} </p>
                    <p>年份：{{ row.year }}</p>
                    <p class="path-text">原始名称: {{ row.original_name }}</p>
                  </div>
                </el-descriptions-item>
                <el-descriptions-item label="时间" :span="2">
                  <div>
                    <p>创建：<br />{{ row.created_at ? formatTimestamp(row.created_at) : '-' }}</p>
                    <p>刮削：<br />{{ row.scraped_at ? formatTimestamp(row.scraped_at) : '-' }}</p>
                    <p>整理：<br />{{ row.renamed_at ? formatTimestamp(row.renamed_at) : '-' }}</p>
                  </div>
                </el-descriptions-item>
                <el-descriptions-item label="失败原因" v-if="row.failed_reason" :span="2">
                  {{ row.failed_reason ? row.failed_reason : '-' }}
                </el-descriptions-item>
                <el-descriptions-item label="操作" :span="2">
                  <div>
                    <el-button type="text" @click="handleDetail(row)">详情</el-button>
                    <el-button type="warning" size="small" @click="reScrape(row)"
                      v-if="(row.type == 'movie' && (row.status == 'scrape_failed' || row.status == 'scanned' || row.status == 'renamed')) || (row.type == 'tvshow' && row.status == 'scrape_failed')">重新识别</el-button>
                    <el-button type="success" size="small" @click="markAsFinished(row)"
                      v-if="row.status == 'renaming' || row.status == 'scraped'">标记为已整理</el-button>
                  </div>
                </el-descriptions-item>
              </el-descriptions>
            </template>
          </el-table-column>
          <el-table-column prop="path" label="文件路径">
            <template #default="{ row }">
              <div>
                <p class="path-text">{{ row.source_full_path }}</p>
                <p style="margin: 10px 0; display: flex; align-items: center; flex-wrap: wrap;">
                  <el-tag :type="getRenameTypeTagType(row.rename_type)">
                    {{ getRenameTypeName(row.rename_type) }}
                  </el-tag>
                  <span style="margin-left: 12px;">到</span>
                </p>
                <p class="path-text">{{ row.dest_full_path }}</p>
              </div>
            </template>
          </el-table-column>
        </el-table>
        <el-table v-loading="loading" :data="records" @selection-change="handleSelectionChange"
          :row-key="(row: ScrapeRecord) => row.id" style="width: 100%" class="desktop-table">
          <el-table-column type="selection" width="55" />
          <el-table-column prop="path_is_scraping" label="运行状态" width="80">
            <template #default="{ row }">
              <span class="info-value" v-if="row.path_is_scraping">
                <el-icon class="is-loading">
                  <Loading />
                </el-icon>
                <el-text class="mx-1" type="primary">刮削中...</el-text>
              </span>
              <span class="info-value" v-else>
                <el-text class="mx-1" type="info" size="small">未执行</el-text>
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="文件状态" width="200">
            <template #default="{ row }">
              <el-tooltip :content="getStatusTooltip(row.status)" placement="top">
                <el-tag :type="getStatusTagType(row.status)">
                  <el-icon>
                    <Warning />
                  </el-icon>
                  {{ getStatusName(row.status) }}
                </el-tag>
              </el-tooltip>
              <p v-if="row.failed_reason" style="margin-top: 4px;">{{ row.failed_reason }}</p>
            </template>
          </el-table-column>
          <el-table-column prop="type" label="类型" width="80">
            <template #default="{ row }">
              <el-tag :type="getTypeTagType(row.type)">
                {{ getTypeName(row.type) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="source_type" label="来源" width="80">
            <template #default="{ row }">
              <el-tag :type="getSourceTypeTagType(row.source_type)">
                {{ getSourceTypeName(row.source_type) }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column prop="path" label="文件路径" width="400">
            <template #default="{ row }">
              <div>
                <p class="path-text">{{ row.source_full_path }}</p>
                <p style="margin: 10px 0; display: flex; align-items: center; flex-wrap: wrap;">
                  <el-tag :type="getRenameTypeTagType(row.rename_type)">
                    {{ getRenameTypeName(row.rename_type) }}
                  </el-tag>
                  <span style="margin-left: 12px;">到</span>
                </p>
                <p class="path-text">{{ row.dest_full_path }}</p>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="识别信息" width="240">
            <template #default="{ row }">
              <div>
                <p class="path-text">Tmdb ID: {{ row.tmdb_id }}</p>
                <p class="path-text">识别名称: {{ row.media_name }} 年份：{{ row.year }}</p>
                <p class="path-text">原始名称: {{ row.original_name }}</p>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="rename_time" label="时间" width="250">
            <template #default="{ row }">
              <p>创建时间：{{ row.created_at ? formatTimestamp(row.created_at) : '-' }}</p>
              <p>刮削时间：{{ row.scraped_at ? formatTimestamp(row.scraped_at) : '-' }}</p>
              <p>整理时间：{{ row.renamed_at ? formatTimestamp(row.renamed_at) : '-' }}</p>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button type="text" @click="handleDetail(row)">详情</el-button>
              <el-button type="warning" size="small" @click="reScrape(row)"
                v-if="(row.type == 'movie' && (row.status == 'scrape_failed' || row.status == 'scanned' || row.status == 'renamed')) || (row.type == 'tvshow' && row.status == 'scrape_failed')">重新识别</el-button>
              <el-button type="success" size="small" @click="markAsFinished(row)"
                v-if="row.status == 'renaming' || row.status == 'scraped'">标记为已整理</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination-container mobile-pagination">
          <el-pagination v-model:current-page="pagination.currentPage" v-model:page-size="pagination.pageSize"
            :page-sizes="100" layout="total, prev, pager, next" :total="total"
             @current-change="handleCurrentChange" />
        </div>
        <div class="pagination-container desktop-pagination">
          <el-pagination v-model:current-page="pagination.currentPage" v-model:page-size="pagination.pageSize"
            :page-sizes="[100, 200, 500]" layout="total, sizes, prev, pager, next, jumper" :total="total"
            @size-change="handleSizeChange" @current-change="handleCurrentChange" />
        </div>
      </div>
    </div>

    <!-- 详情对话框 -->
    <el-dialog v-model="showDetailDialog" title="刮削详情" width="600px">
      <div v-if="selectedRecord" class="detail-content">
        <el-descriptions border direction="vertical">
          <el-descriptions-item label="原始路径">
            <el-tooltip :content="selectedRecord.path" placement="top">
              <pre style="margin: 0; white-space: pre-wrap; word-break: break-all; max-height: 100px; overflow: auto;">{{
                selectedRecord.path }}</pre>
            </el-tooltip>
          </el-descriptions-item>
          <el-descriptions-item label="原始文件名">
            <el-tooltip :content="selectedRecord.file_name" placement="top">
              <pre style="margin: 0; white-space: pre-wrap; word-break: break-all; max-height: 100px; overflow: auto;">{{
                selectedRecord.file_name }}</pre>
            </el-tooltip>
          </el-descriptions-item>
          <el-descriptions-item label="识别名称">{{ selectedRecord.media_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="识别年份">{{ selectedRecord.year || '-' }}</el-descriptions-item>
          <el-descriptions-item label="识别类型">{{ getTypeName(selectedRecord.type) }}</el-descriptions-item>
          <el-descriptions-item label="二级分类">{{ selectedRecord.category_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="新文件夹">{{ selectedRecord.new_dest_path || '-' }}</el-descriptions-item>
          <el-descriptions-item label="新文件名">{{ selectedRecord.new_dest_name || '-' }}</el-descriptions-item>
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
            <pre v-if="selectedRecord.failed_reason"
              style="margin: 0; white-space: pre-wrap; word-break: break-all; max-height: 100px; overflow: auto;">{{
                selectedRecord.failed_reason }}</pre>
            <span v-else>-</span>
          </el-descriptions-item>

        </el-descriptions>
      </div>
    </el-dialog>

    <!-- 重新识别对话框 -->
    <el-dialog v-model="showReScrapeDialog" title="重新识别" width="700px" :close-on-click-modal="false">
      <div class="re-scrape-dialog-content">
        <div class="search-section">
          <el-form label-width="80px" @submit.prevent>
            <el-form-item label="文件名">
              <el-text>{{ reScrapeForm.originalFileName }}</el-text>
            </el-form-item>
            <el-form-item label="搜索方式">
              <el-radio-group v-model="searchMode" @change="handleSearchModeChange">
                <el-radio value="name">名称+年份</el-radio>
                <el-radio value="tmdb">TMDB ID</el-radio>
              </el-radio-group>
            </el-form-item>
            <div v-if="searchMode === 'name'" class="search-row">
              <el-form-item label="名称" class="name-input">
                <el-input v-model="reScrapeForm.name" placeholder="请输入影视剧名称" />
              </el-form-item>
              <el-form-item label="年份" class="year-input">
                <el-input v-model="reScrapeForm.year" placeholder="年份" type="number" :min="1900"
                  :max="new Date().getFullYear() + 5" />
              </el-form-item>
            </div>
            <div v-if="searchMode === 'tmdb'" class="search-row">
              <el-form-item label="TMDB ID" class="tmdb-input">
                <el-input v-model="reScrapeForm.tmdb_id" placeholder="请输入TMDB ID" type="number" :min="1" />
              </el-form-item>
            </div>
            <el-form-item label="季" v-if="reScrapeForm.type == 'tvshow'" class="season-episode-row">
              <el-input v-model="reScrapeForm.season" placeholder="季数" type="number" :min="0" :max="100"
                style="width: 100px;" />
              <span class="episode-label">集</span>
              <el-input v-model="reScrapeForm.episode" placeholder="集数" type="number" :min="0" :max="10000"
                style="width: 100px;" />
            </el-form-item>
          </el-form>
        </div>

        <div class="search-results-section" v-if="searchResults.length > 0">
          <div class="results-header">
            <el-icon>
              <Film />
            </el-icon>
            <span>搜索结果 ({{ searchResults.length }})</span>
          </div>
          <div class="results-list">
            <div class="result-item" v-for="item in searchResults" :key="item.tmdb_id">
              <div class="result-poster">
                <el-image :src="item.poster_url" fit="cover" lazy>
                  <template #error>
                    <div class="poster-placeholder">
                      <el-icon>
                        <Picture />
                      </el-icon>
                    </div>
                  </template>
                </el-image>
              </div>
              <div class="result-info">
                <div class="result-title">{{ item.title }}</div>
                <div class="result-original-title" v-if="item.original_title !== item.title">
                  原名: {{ item.original_title }}
                </div>
                <div class="result-meta">
                  <el-tag size="small" type="info">{{ item.year || '未知年份' }}</el-tag>
                  <el-tag size="small" type="warning">TMDB ID: {{ item.tmdb_id }}</el-tag>
                </div>
                <div class="result-overview" v-if="item.overview">
                  {{ item.overview }}
                </div>
              </div>
              <div class="result-action">
                <el-button type="primary" size="small" @click="selectSearchResult(item)" :loading="item.selecting">
                  选择
                </el-button>
              </div>
            </div>
          </div>
        </div>

        <div class="empty-results" v-if="hasSearched && searchResults.length === 0">
          <el-empty description="未找到匹配的影片" :image-size="80" />
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="closeReScrapeDialog">取消</el-button>
          <el-button type="primary" @click="searchTmdb" :loading="searchLoading">
            <el-icon>
              <Search />
            </el-icon>
            搜索
          </el-button>
        </div>
      </template>
    </el-dialog>

    <div class="bottom-tips">
      <el-alert type="info" :closable="false">
        <template #title>
          <span class="tips-text">
            当前刮削产生的临时文件存放在 <strong>config/tmp/刮削临时文件/</strong> 目录下，可以观察是否存在异常情况，刮削完成的文件会删除临时文件
          </span>
        </template>
      </el-alert>
    </div>

    <!-- 回滚对话框 -->
    <el-dialog v-model="showRollbackDialog" title="注意" width="320px">
      <p>确认回滚该刮削记录吗？回滚后视频+字幕会放回原目录并且根据查询到的tmdb信息重命名，刮削记录会被删除，后续扫描时会重新刮削该影片。</p>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showRollbackDialog = false">取消</el-button>
          <el-button type="primary" @click="showReScrapeDialog = true">
            确认
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Film, Picture, Search } from '@element-plus/icons-vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject } from 'vue'
import { formatTimestamp } from '@/utils/timeUtils'
import 'element-plus/theme-chalk/display.css'

const http: AxiosStatic | undefined = inject('$http')

// 定义刮削记录接口
interface ScrapeRecord {
  id: number
  type: 'movie' | 'tvshow' | 'other'
  path: string
  file_name: string
  media_name: string
  original_name: string
  year: number
  tmdb_id: number
  season_number: number
  episode_number: number
  episode_name?: string
  status: 'scanned' | 'scraping' | 'scraped' | 'scrape_failed' | 'renaming' | 'renamed' | 'rename_failed'
  failed_reason: string
  created_at: number
  updated_at: number
  scanned_at: number
  scraped_at: number
  renamed_at: number
  audio_count: number
  subtitle_count: number
  resolution: string
  resolution_level: string
  is_hdr: boolean
  category_name: string
  new_dest_path: string
  new_dest_name: string
  path_is_scraping: boolean
  source_full_path: string
  dest_full_path: string
  source_type: string
  rename_type: string
  scrape_type: string
}

interface TmdbSearchResult {
  tmdb_id: number
  title: string
  original_title: string
  year: string
  poster_url: string
  overview: string
  selecting?: boolean
}

// 状态变量
const records = ref<ScrapeRecord[]>([])
const originalRecords = ref<ScrapeRecord[]>([])
const isMerged = ref(false)
const loading = ref(false)
const selectedRecords = ref<ScrapeRecord[]>([])
const statusFilter = ref('')
const typeFilter = ref('')
const nameFilter = ref('')
const showDetailDialog = ref(false)
const selectedRecord = ref<ScrapeRecord | null>(null)
const showRollbackDialog = ref(false)

// 分页相关
const pagination = ref({
  currentPage: 1,
  pageSize: 20
})
const total = ref(0)

// 加载刮削记录
const loadRecords = async () => {
  try {
    loading.value = true
    // 构建查询参数
    const params: Record<string, string | number> = {
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

    // 添加名称搜索参数
    if (nameFilter.value) {
      params.name = nameFilter.value
    }

    const response = await http?.get(`${SERVER_URL}/scrape/records`, { params })

    if (response?.data.code === 200) {
      records.value = response.data.data.list
      originalRecords.value = JSON.parse(JSON.stringify(response.data.data.list)) // 深拷贝保存原始记录
      total.value = response.data.data.total
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
  nameFilter.value = '' // 重置名称搜索过滤器
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

// 删除所选刮削记录
const handleDeleteSelectedRecords = async () => {
  try {
    if (selectedRecords.value.length === 0) {
      ElMessage.warning('请选择记录')
      return
    }

    // 确认删除操作
    if (!confirm(`确定要删除选中的 ${selectedRecords.value.length} 条记录吗？`)) {
      return
    }

    const ids = selectedRecords.value.map(record => record.id)
    // 发送DELETE请求，参数与导出识别错误文件接口一致
    // 构造URL，将ids作为GET参数传递
    const idsQuery = ids.join(',')
    const response = await http?.delete(`${SERVER_URL}/scrape/records?ids=${idsQuery}`)

    if (response?.data.code === 200) {
      ElMessage.success('删除成功')
      // 清空选择
      selectedRecords.value = []
      // 刷新记录列表
      loadRecords()
    } else {
      ElMessage.error(`删除失败: ${response?.data.message || '未知错误'}`)
    }
  } catch (error) {
    console.error('删除失败:', error)
    ElMessage.error('删除失败: 网络错误')
  }
}

const handleRename = async () => {
  try {
    if (selectedRecords.value.length === 0) {
      ElMessage.warning('请选择记录')
      return
    }
    // 确认删除操作
    if (!confirm(`确定要重新整理选中的 ${selectedRecords.value.length} 条记录吗？`)) {
      return
    }

    const ids = selectedRecords.value.map(record => record.id)
    // 发送DELETE请求，参数与导出识别错误文件接口一致
    // 构造URL，将ids作为GET参数传递
    const idsQuery = ids.join(',')
    const response = await http?.post(`${SERVER_URL}/scrape/rename-failed?ids=${idsQuery}`)

    if (response?.data.code === 200) {
      ElMessage.success('重新整理成功')
      // 清空选择
      selectedRecords.value = []
      // 刷新记录列表
      loadRecords()
    } else {
      ElMessage.error(`重新整理失败: ${response?.data.message || '未知错误'}`)
    }
  } catch (error) {
    console.error('重新整理失败:', error)
    ElMessage.error('重新整理失败: 网络错误')
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
  type: '',
  id: 0,
  name: '',
  year: '',
  tmdb_id: '',
  originalFileName: '',
  season: -1,
  episode: -1,
  status: ''
})
const searchResults = ref<TmdbSearchResult[]>([])
const searchLoading = ref(false)
const hasSearched = ref(false)
const searchMode = ref<'name' | 'tmdb'>('name')

const handleSearchModeChange = () => {
  searchResults.value = []
  hasSearched.value = false
  if (searchMode.value === 'name') {
    reScrapeForm.value.tmdb_id = ''
  } else {
    reScrapeForm.value.name = ''
    reScrapeForm.value.year = ''
  }
}

const closeReScrapeDialog = () => {
  showReScrapeDialog.value = false
  searchResults.value = []
  hasSearched.value = false
  searchMode.value = 'name'
}

const searchTmdb = async () => {
  if (searchMode.value === 'name') {
    if (!reScrapeForm.value.name) {
      ElMessage.warning('请输入影视剧名称')
      return
    }
  } else {
    if (!reScrapeForm.value.tmdb_id) {
      ElMessage.warning('请输入TMDB ID')
      return
    }
  }

  try {
    searchLoading.value = true
    hasSearched.value = false
    searchResults.value = []

    const params: Record<string, string | number> = {
      type: reScrapeForm.value.type
    }

    if (searchMode.value === 'name') {
      params.name = reScrapeForm.value.name
      if (reScrapeForm.value.year) {
        params.year = reScrapeForm.value.year
      }
    } else {
      params.tmdb_id = reScrapeForm.value.tmdb_id
    }

    const response = await http?.get(`${SERVER_URL}/scrape/tmdb-search`, { params, timeout: 30000 })

    if (response?.data.code === 200) {
      searchResults.value = (response.data.data || []).map((item: TmdbSearchResult) => ({
        ...item,
        selecting: false
      }))
      hasSearched.value = true
    } else {
      ElMessage.error(response?.data.message || '搜索失败')
    }
  } catch (error) {
    console.error('TMDB搜索失败:', error)
    ElMessage.error('搜索失败: 网络错误')
  } finally {
    searchLoading.value = false
  }
}

const selectSearchResult = async (item: TmdbSearchResult) => {
  try {
    item.selecting = true

    const params = {
      id: reScrapeForm.value.id,
      tmdb_id: item.tmdb_id,
      season: reScrapeForm.value.season >= 0 ? parseInt(reScrapeForm.value.season + "") : -1,
      episode: reScrapeForm.value.episode >= 0 ? parseInt(reScrapeForm.value.episode + "") : -1
    }

    const response = await http?.post(`${SERVER_URL}/scrape/re-scrape`, params, { timeout: 60000 })

    if (response?.data.code === 200) {
      ElMessage.success('重新识别请求已发送')
      showReScrapeDialog.value = false
      searchResults.value = []
      hasSearched.value = false
      loadRecords()
    } else {
      ElMessage.error(response?.data.message || '重新识别失败')
    }
  } catch (error) {
    console.error('重新识别失败:', error)
    ElMessage.error('重新识别失败: 网络错误')
  } finally {
    item.selecting = false
  }
}

// 处理重新识别
const reScrape = (record: ScrapeRecord) => {
  searchResults.value = []
  hasSearched.value = false
  searchMode.value = 'name'
  reScrapeForm.value = {
    type: record.type,
    id: record.id,
    name: record.media_name || '',
    year: record.year?.toString() || '',
    tmdb_id: '',
    originalFileName: record.file_name || '',
    season: record.season_number || -1,
    episode: parseInt(record.episode_number + "") || -1,
    status: record.status || ''
  }
  if (record.status == "renamed") {
    showRollbackDialog.value = true
  } else {
    showReScrapeDialog.value = true
  }
}

const handleDeleteFailedRecords = async () => {
  try {
    const response = await http?.post(`${SERVER_URL}/scrape/clear-failed`)

    if (response?.data.code === 200) {
      ElMessage.success('清除所有刮削失败的记录成功')
      loadRecords()
    } else {
      ElMessage.error(`清除所有刮削失败的记录失败: ${response?.data.message || '未知错误'}`)
    }
  } catch (error) {
    console.error('清除所有刮削失败的记录失败:', error)
    ElMessage.error('清除所有刮削失败的记录失败: 网络错误')
  }
}

const handleTruncateAll = async () => {
  try {
    // 第一次确认
    await ElMessageBox.confirm(
      '此操作将删除所有刮削记录，是否继续？',
      '警告',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    // 第二次确认
    await ElMessageBox.confirm(
      '确认要清空所有刮削记录吗？此操作不可恢复！',
      '二次确认',
      {
        confirmButtonText: '确认清空',
        cancelButtonText: '取消',
        type: 'error',
      }
    )

    // 发送请求
    const response = await http?.post(`${SERVER_URL}/scrape/truncate-all`)

    if (response?.data.code === 200) {
      ElMessage.success('清空记录成功')
      // 清空选择
      selectedRecords.value = []
      // 刷新记录列表
      loadRecords()
    } else {
      ElMessage.error(`清空记录失败: ${response?.data.message || '未知错误'}`)
    }
  } catch (error) {
    // 如果用户取消操作，不显示错误消息
    if (error !== 'cancel') {
      console.error('清空记录失败:', error)
      ElMessage.error('清空记录失败: 网络错误')
    }
  }
}

const markAsFinished = async (record: ScrapeRecord) => {
  try {
    // 显示确认对话框
    await ElMessageBox.confirm(
      '请确保文件已在目标位置存在，变为已整理的文件不会继续整理',
      '确认操作',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    // 发送POST请求到/scrape/finish接口
    const response = await http?.post(`${SERVER_URL}/scrape/finish`, { id: record.id })

    if (response?.data.code === 200) {
      ElMessage.success('标记为已整理成功')
      // 刷新记录列表
      loadRecords()
    } else {
      ElMessage.error(`标记为已整理失败: ${response?.data.message || '未知错误'}`)
    }
  } catch (error) {
    // 如果用户取消操作，不显示错误消息
    if (!(error as Error).message.includes('用户取消操作')) {
      console.error('标记为已整理失败:', error)
      ElMessage.error('标记为已整理失败: 网络错误')
    }
  }
}

const getStatusTooltip = (status: string): string => {
  switch (status) {
    case 'scanned':
      return '文件已扫描入库，等待刮削'
    case 'scraping':
      return '正在刮削中...'
    case 'scraped':
      return '刮削成功，等待整理。如果本次没有成功，下次任务启动时会继续处理'
    case 'scrape_failed':
      return '刮削失败，需要重新识别。'
    case 'renaming':
      return '正在整理...'
    case 'renamed':
      return '整理成功，无需额外处理'
    case 'rename_failed':
      return '整理失败，请删除刮削记录或者标记为待整理后等待下次任务启动时重新整理。'
    case 'rollbacking':
      return '等待回滚任务执行时会将视频+字幕放回原目录，然后删除该刮削记录'
    default:
      return '未知状态'
  }
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

const getSourceTypeName = (type: string): string => {
  switch (type) {
    case 'local':
      return '本地文件'
    case '115':
      return '115云盘'
    case 'openlist':
      return 'OpenList'
    case '123':
      return '123云盘'
    default:
      return '其他'
  }
}

// 获取类型标签类型
const getSourceTypeTagType = (type: string): string => {
  switch (type) {
    case 'local':
      return 'warning'
    case '115':
      return 'primary'
    case 'openlist':
      return 'success'
    case '123':
      return 'info'
    default:
      return 'info'
  }
}

// 获取类型标签类型
const getRenameTypeTagType = (type: string): string => {
  switch (type) {
    case 'move':
      return 'warning'
    case 'copy':
      return 'primary'
    case 'hard_symlink':
      return 'success'
    case 'soft_symlink':
      return 'info'
    default:
      return 'info'
  }
}

const getRenameTypeName = (type: string): string => {
  switch (type) {
    case 'move':
      return '移动'
    case 'copy':
      return '复制'
    case 'hard_symlink':
      return '硬链接'
    case 'soft_symlink':
      return '软链接'
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
      return 'info'
    case 'scraped':
      return 'primary'
    case 'scrape_failed':
      return 'danger'
    case 'renaming':
      return 'primary'
    case 'renamed':
      return 'success'
    case 'rename_failed':
      return 'danger'
    case 'rollbacking':
      return 'danger'
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
    case 'rename_failed':
      return '整理失败'
    case 'rollbacking':
      return '回滚中'
    default:
      return '未知'
  }
}

// 组件挂载时加载数据
onMounted(() => {
  loadRecords()
})

onUnmounted(() => {

})
</script>

<style scoped>
.scrape-records-container {
  padding: 20px;
  display: flex;
  flex-direction: column;
  height: calc(100vh - 40px);
}

.header-section {
  flex-shrink: 0;
}

.page-title {
  font-size: 20px;
  font-weight: bold;
  margin-bottom: 20px;
  color: #303133;
}

.top-actions {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
  padding: 16px;
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e7ed 100%);
  border-radius: 8px;
  flex-wrap: wrap;
}

.action-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.action-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.selected-count {
  color: #606266;
  font-size: 13px;
}

.search-filter-section {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.filter-input {
  width: 200px;
}

.filter-select {
  width: 150px;
}

.filter-select-small {
  width: 120px;
}

.table-section {
  display: block;
}

.table-container {
  background: #fff;
  border-radius: 4px;
  padding: 0;
  display: block;
}

.mobile-table, .mobile-pagination {
  display: none;
}

.desktop-table, .desktop-pagination {
  display: table;
}

.pagination-container {
  padding: 16px;
  display: flex;
  justify-content: flex-end;
  background: #fff;
  border-top: 1px solid #ebeef5;
  margin-top: -1px;
  border-radius: 0 0 4px 4px;
  flex-shrink: 0;
}

.bottom-tips {
  margin-top: 20px;
}

.tips-text {
  font-size: 13px;
  color: #606266;
}

.tips-text strong {
  color: #303133;
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

.detail-label {
  color: #969797;
}

.detail-value {
  font-size: 16px;
}

.re-scrape-dialog-content {
  max-height: 60vh;
  overflow-y: auto;
}

.search-section {
  margin-bottom: 20px;
}

.search-row {
  display: flex;
  gap: 12px;
}

.search-row .name-input {
  flex: 1;
}

.search-row .year-input {
  width: 150px;
}

.search-row .tmdb-input {
  flex: 1;
  max-width: 250px;
}

.season-episode-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.episode-label {
  color: #606266;
  font-size: 14px;
}

.search-results-section {
  margin-top: 16px;
  border-top: 1px solid #ebeef5;
  padding-top: 16px;
}

.results-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 300px;
  overflow-y: auto;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: #fafafa;
  transition: all 0.2s ease;
}

.result-item:hover {
  border-color: #409eff;
  background: #f5f7fa;
}

.result-poster {
  width: 60px;
  height: 90px;
  flex-shrink: 0;
  border-radius: 4px;
  overflow: hidden;
  background: #e4e7ed;
}

.result-poster .el-image {
  width: 100%;
  height: 100%;
}

.poster-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #c0c4cc;
  font-size: 24px;
}

.result-info {
  flex: 1;
  min-width: 0;
}

.result-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.result-original-title {
  font-size: 13px;
  color: #909399;
  margin-bottom: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.result-meta {
  display: flex;
  gap: 8px;
}

.result-overview {
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

.result-action {
  flex-shrink: 0;
}

.empty-results {
  padding: 20px 0;
}

@media (min-width: 769px) {
  .mobile-table, .mobile-pagination {
    display: none;
  }

  .desktop-table, .desktop-pagination {
    display: table;
  }
}

@media (max-width: 768px) {
  .scrape-records-container {
    padding: 12px;
  }

  .top-actions {
    flex-direction: column;
    padding: 12px;
  }

  .action-group {
    width: 100%;
  }

  .action-buttons {
    width: 100%;
  }

  .action-buttons .el-button {
    flex: 1;
    min-width: 0;
    font-size: 13px;
    padding: 8px 12px;
  }

  .search-filter-section {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-input,
  .filter-select,
  .filter-select-small {
    width: 100%;
  }

  .search-filter-section .el-button {
    width: 100%;
  }

  .page-title {
    font-size: 18px;
    margin-bottom: 12px;
  }

  .scrape-records-container {
    height: auto;
    display: block;
  }

  .header-section {
    margin-bottom: 0;
  }

  .mobile-table, .mobile-pagination {
    display: table;
  }

  .desktop-table, .desktop-pagination {
    display: none;
  }
}

@media (max-width: 600px) {
  .search-row {
    flex-direction: column;
  }

  .search-row .year-input {
    width: 100%;
  }

  .result-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .result-poster {
    width: 100%;
    height: 150px;
  }

  .result-action {
    width: 100%;
    margin-top: 8px;
  }

  .result-action .el-button {
    width: 100%;
  }
}
</style>
