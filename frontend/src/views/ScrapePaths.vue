<template>
  <div class="scrape-paths-page">
    <el-card class="scrape-paths-card">
      <template #header>
        <div class="card-header">
          <span>刮削目录管理</span>
          <el-button type="primary" @click="showAddDialog = true">添加刮削目录</el-button>
        </div>
      </template>

      <el-table :data="pathes" v-loading="loading" element-loading-text="加载中..." class="scrape-paths-table">
        <el-table-column prop="account_name" label="账号" min-width="120">
          <template #default="scope">
            <span class="info-value">{{ getAccountName(scope.row.account_id) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="media_type" label="媒体类型" min-width="100">
          <template #default="scope">
            <span class="info-value">{{ getMediaTypeText(scope.row.media_type) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="source_path" label="来源路径" min-width="200">
          <template #default="scope">
            <span class="info-value">{{ scope.row.source_path }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="dest_path" label="目标路径" min-width="200">
          <template #default="scope">
            <span class="info-value">{{ scope.row.dest_path }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="scrape_type" label="操作方式" min-width="120">
          <template #default="scope">
            <span class="info-value">{{ getScrapeTypeText(scope.row.scrape_type) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="rename_type" label="整理方式" min-width="100">
          <template #default="scope">
            <span class="info-value">{{ getRenameTypeText(scope.row.rename_type) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="120">
          <template #default="scope">
            <el-tag v-if="scope.row.is_scraping" type="warning">刮削中</el-tag>
            <el-tag v-else-if="scope.row.is_renaming" type="warning">整理中</el-tag>
            <el-tag v-else type="success">空闲</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" min-width="200">
          <template #default="scope">
            <el-button size="small" type="primary" :loading="scope.row.scanning" @click="handleScan(scope.row)">
              {{ scope.row.is_scraping ? '刮削中...' : '启动' }}
            </el-button>
            <el-button size="small" type="warning" :loading="scope.row.scanning" @click="handleStop(scope.row)"
              :disabled="!scope.row.is_scraping && !scope.row.is_renaming">
              停止
            </el-button>
            <el-button size="small" @click="handleEdit(scope.row)">编辑</el-button>
            <el-button size="small" type="danger" :loading="scope.row.deleting"
              @click="handleDelete(scope.row, scope.$index)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="pathes.length === 0 && !loading" class="empty-state">
        <el-empty description="暂无刮削目录" />
      </div>
    </el-card>

    <!-- 添加刮削目录对话框 -->
    <el-dialog v-model="showAddDialog" title="添加刮削目录" width="600px" :close-on-click-modal="false">
      <el-form ref="addFormRef" :model="addForm" :rules="addFormRules" label-width="120px" class="scrape-form">
        <el-form-item label="同步源类型" prop="source_type">
          <el-select v-model="addForm.source_type" placeholder="请选择同步源类型" @change="loadAccounts">
            <el-option label="115网盘" value="115" />
            <el-option label="阿里云盘" value="ali" />
            <el-option label="夸克网盘" value="quark" />
            <el-option label="本地目录" value="local" />
          </el-select>
        </el-form-item>

        <el-form-item v-if="addForm.source_type !== 'local'" label="网盘账号" prop="account_id">
          <el-select v-model="addForm.account_id" placeholder="请选择网盘账号" :loading="accountsLoading">
            <el-option v-for="account in accounts" :key="account.id" :label="account.name" :value="account.id" />
          </el-select>
        </el-form-item>

        <el-form-item label="媒体类型" prop="media_type">
          <el-select v-model="addForm.media_type" placeholder="请选择媒体类型">
            <el-option label="电影" value="movie" />
            <el-option label="电视剧" value="tvshow" />
            <el-option label="其他" value="other" />
          </el-select>
        </el-form-item>

        <el-form-item label="来源路径" prop="source_path_id">
          <el-input v-model="addForm.source_path" readonly placeholder="请选择来源路径" @click="openDirSelector(true)">
            <template #append>
              <el-button @click="openDirSelector(true)">选择</el-button>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="操作方式" prop="scrape_type">
          <el-select v-model="addForm.scrape_type" placeholder="请选择操作方式">
            <el-option label="仅刮削" value="only_scrape" />
            <el-option label="刮削和整理" value="scrape_and_rename" />
            <el-option label="仅整理" value="only_rename" />
          </el-select>
        </el-form-item>

        <el-form-item v-if="addForm.scrape_type !== 'only_scrape'" label="整理方式" prop="rename_type">
          <el-select v-model="addForm.rename_type" placeholder="请选择整理方式"
            :disabled="addForm.scrape_type === 'only_scrape'">
            <el-option label="移动" value="move" />
            <el-option label="复制" value="copy" />
            <el-option label="软链接" value="soft_symlink" />
            <el-option label="硬链接" value="hard_symlink" />
          </el-select>
        </el-form-item>

        <el-form-item v-if="addForm.scrape_type !== 'only_scrape'" label="二级分类" prop="enable_category">
          <el-switch v-model="addForm.enable_category" />
        </el-form-item>

        <el-form-item v-if="addForm.scrape_type !== 'only_scrape'" label="目标路径" prop="dest_path_id">
          <el-input v-model="addForm.dest_path" readonly placeholder="请选择目标路径" @click="openDirSelector(false)">
            <template #append>
              <el-button @click="openDirSelector(false)">选择</el-button>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item v-if="addForm.scrape_type !== 'only_scrape' && addForm.enable_category" label="文件夹重命名模板"
          prop="folder_name_template">
          <el-input v-model="addForm.folder_name_template" placeholder="请输入文件夹重命名模板" />
        </el-form-item>

        <el-form-item v-if="addForm.scrape_type !== 'only_scrape'" label="文件重命名模板" prop="file_name_template">
          <el-input v-model="addForm.file_name_template" placeholder="请输入文件重命名模板" />
        </el-form-item>

        <el-form-item label="删除关键词" prop="delete_keyword">
          <el-select v-model="addForm.delete_keyword" multiple filterable allow-create default-first-option
            placeholder="请输入删除关键词">
            <el-option v-for="item in deleteKeywords" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>

        <el-form-item label="最小视频文件大小(MB)" prop="min_video_file_size">
          <el-input-number v-model="addForm.min_video_file_size" :min="0" controls-position="right"
            style="width: 100%" />
        </el-form-item>

        <el-form-item label="视频文件扩展名" prop="video_ext_list">
          <div class="video-ext-container">
            <el-tag v-for="(ext, index) in addForm.video_ext_list" :key="index" closable
              @close="removeVideoExt(index, 'add')">
              {{ ext }}
            </el-tag>
            <el-input v-model="tempVideoExt" placeholder="添加扩展名" size="small" style="width: 120px; margin-top: 5px;"
              @keyup.enter="addVideoExt('add')">
              <template #append>
                <el-button @click="addVideoExt('add')">添加</el-button>
              </template>
            </el-input>
          </div>
        </el-form-item>

        <el-form-item label="排除无头像演员" prop="exclude_no_image_actor">
          <el-switch v-model="addForm.exclude_no_image_actor" />
        </el-form-item>

        <el-form-item label="AI识别" prop="enable_ai">
          <el-select v-model="addForm.enable_ai" placeholder="请选择AI识别模式">
            <el-option label="禁用" value="off" />
            <el-option label="辅助" value="assist" />
            <el-option label="强制" value="force" />
          </el-select>
        </el-form-item>

        <el-form-item v-if="addForm.enable_ai !== 'off'" label="AI提示词" prop="ai_prompt">
          <el-input v-model="addForm.ai_prompt" type="textarea" placeholder="请输入AI提示词"
            :autosize="{ minRows: 2, maxRows: 4 }" />
        </el-form-item>

        <el-form-item label="定时同步" prop="enable_cron">
          <el-switch v-model="addForm.enable_cron" />
        </el-form-item>

        <el-form-item label="启用fanart.tv" prop="enable_fanart_tv">
          <el-switch v-model="addForm.enable_fanart_tv" />
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showAddDialog = false">取消</el-button>
          <el-button type="primary" @click="handleAdd" :loading="addLoading">
            确定
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 编辑刮削目录对话框 -->
    <el-dialog v-model="showEditDialog" title="编辑刮削目录" width="600px" :close-on-click-modal="false">
      <el-form ref="editFormRef" :model="editForm" :rules="editFormRules" label-width="120px" class="scrape-form">
        <el-form-item label="同步源类型" prop="source_type">
          <el-select v-model="editForm.source_type" placeholder="请选择同步源类型" disabled>
            <el-option label="115网盘" value="115" />
            <el-option label="阿里云盘" value="ali" />
            <el-option label="夸克网盘" value="quark" />
            <el-option label="本地目录" value="local" />
          </el-select>
        </el-form-item>

        <el-form-item v-if="editForm.source_type !== 'local'" label="网盘账号" prop="account_id">
          <el-select v-model="editForm.account_id" placeholder="请选择网盘账号" :loading="accountsLoading" disabled>
            <el-option v-for="account in accounts" :key="account.id" :label="account.name" :value="account.id" />
          </el-select>
        </el-form-item>

        <el-form-item label="媒体类型" prop="media_type">
          <el-select v-model="editForm.media_type" placeholder="请选择媒体类型">
            <el-option label="电影" value="movie" />
            <el-option label="电视剧" value="tvshow" />
            <el-option label="其他" value="other" />
          </el-select>
        </el-form-item>

        <el-form-item label="来源路径" prop="source_path_id">
          <el-input v-model="editForm.source_path" readonly placeholder="请选择来源路径" @click="openEditDirSelector(true)">
            <template #append>
              <el-button @click="openEditDirSelector(true)">选择</el-button>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="操作方式" prop="scrape_type">
          <el-select v-model="editForm.scrape_type" placeholder="请选择操作方式">
            <el-option label="仅刮削" value="only_scrape" />
            <el-option label="刮削和整理" value="scrape_and_rename" />
            <el-option label="仅整理" value="only_rename" />
          </el-select>
        </el-form-item>

        <el-form-item v-if="editForm.scrape_type !== 'only_scrape'" label="整理方式" prop="rename_type">
          <el-select v-model="editForm.rename_type" placeholder="请选择整理方式"
            :disabled="editForm.scrape_type === 'only_scrape'">
            <el-option label="移动" value="move" />
            <el-option label="复制" value="copy" />
            <el-option label="软链接" value="soft_symlink" />
            <el-option label="硬链接" value="hard_symlink" />
          </el-select>
        </el-form-item>

        <el-form-item v-if="editForm.scrape_type !== 'only_scrape'" label="二级分类" prop="enable_category">
          <el-switch v-model="editForm.enable_category" />
        </el-form-item>

        <el-form-item v-if="editForm.scrape_type !== 'only_scrape'" label="目标路径" prop="dest_path_id">
          <el-input v-model="editForm.dest_path" readonly placeholder="请选择目标路径" @click="openEditDirSelector(false)">
            <template #append>
              <el-button @click="openEditDirSelector(false)">选择</el-button>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item v-if="editForm.scrape_type !== 'only_scrape' && editForm.enable_category" label="文件夹重命名模板"
          prop="folder_name_template">
          <el-input v-model="editForm.folder_name_template" placeholder="请输入文件夹重命名模板" />
        </el-form-item>

        <el-form-item v-if="editForm.scrape_type !== 'only_scrape'" label="文件重命名模板" prop="file_name_template">
          <el-input v-model="editForm.file_name_template" placeholder="请输入文件重命名模板" />
        </el-form-item>

        <el-form-item label="删除关键词" prop="delete_keyword">
          <el-select v-model="editForm.delete_keyword" multiple filterable allow-create default-first-option
            placeholder="请输入删除关键词">
            <el-option v-for="item in deleteKeywords" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>

        <el-form-item label="最小视频文件大小(MB)" prop="min_video_file_size">
          <el-input-number v-model="editForm.min_video_file_size" :min="0" controls-position="right"
            style="width: 100%" />
        </el-form-item>

        <el-form-item label="视频文件扩展名" prop="video_ext_list">
          <div class="video-ext-container">
            <el-tag v-for="(ext, index) in editForm.video_ext_list" :key="index" closable
              @close="removeVideoExt(index, 'edit')">
              {{ ext }}
            </el-tag>
            <el-input v-model="tempVideoExt" placeholder="添加扩展名" size="small" style="width: 120px; margin-top: 5px;"
              @keyup.enter="addVideoExt('edit')">
              <template #append>
                <el-button @click="addVideoExt('edit')">添加</el-button>
              </template>
            </el-input>
          </div>
        </el-form-item>

        <el-form-item label="排除无头像演员" prop="exclude_no_image_actor">
          <el-switch v-model="editForm.exclude_no_image_actor" />
        </el-form-item>

        <el-form-item label="AI识别" prop="enable_ai">
          <el-select v-model="editForm.enable_ai" placeholder="请选择AI识别模式">
            <el-option label="禁用" value="off" />
            <el-option label="辅助" value="assist" />
            <el-option label="强制" value="force" />
          </el-select>
        </el-form-item>

        <el-form-item v-if="editForm.enable_ai !== 'off'" label="AI提示词" prop="ai_prompt">
          <el-input v-model="editForm.ai_prompt" type="textarea" placeholder="请输入AI提示词"
            :autosize="{ minRows: 2, maxRows: 4 }" />
        </el-form-item>

        <el-form-item label="定时同步" prop="enable_cron">
          <el-switch v-model="editForm.enable_cron" />
        </el-form-item>

        <el-form-item label="启用fanart.tv" prop="enable_fanart_tv">
          <el-switch v-model="editForm.enable_fanart_tv" />
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showEditDialog = false">取消</el-button>
          <el-button type="primary" @click="handleEditSave" :loading="editLoading">
            确定
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 目录选择对话框 -->
    <el-dialog v-model="showDirDialog" :title="isSelectSource ? '选择来源目录' : '选择目标目录'"
      :width="checkIsMobile ? '90%' : '600px'" :close-on-click-modal="false" body-class="directory-selector">
      <div class="dir-selector">
        <DirectorySelector
          v-model="tempSelectedDir"
          :source-type="selectedSourceType"
          :account-id="selectedAccountId"
          @cancel="showDirDialog = false"
          @select="confirmSelectDir"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { isMobile } from "@/utils/deviceUtils"
import DirectorySelector from '@/components/DirectorySelector.vue'

import { inject } from 'vue'
import type { AxiosStatic } from 'axios';
import type { CloudAccount, DirInfo } from '@/typing';
const http: AxiosStatic | undefined = inject('$http')
// 依赖注入
const SERVER_URL = inject('SERVER_URL')

// 接口定义
interface ScrapePath {
  id?: number
  source_type: string
  account_id: number
  media_type: string
  source_path: string
  source_path_id: string
  dest_path: string
  dest_path_id: string
  scrape_type: string
  rename_type: string
  enable_category: boolean
  folder_name_template: string
  file_name_template: string
  delete_keyword: string[]
  min_video_file_size: number
  video_ext_list: string[]
  exclude_no_image_actor: boolean
  enable_ai: string
  ai_prompt: string
  enable_cron: boolean
  enable_fanart_tv: boolean
  is_scraping?: boolean
  is_renaming?: boolean
  deleting?: boolean
  scanning?: boolean
}

// 响应式变量
const pathes = ref<ScrapePath[]>([])
const loading = ref(false)
const accounts = ref<CloudAccount[]>([])
const accountsLoading = ref(false)
const checkIsMobile = ref(isMobile())
const showAddDialog = ref(false)
const showEditDialog = ref(false)
const addLoading = ref(false)
const editLoading = ref(false)
const showDirDialog = ref(false)
const tempSelectedDir = ref<DirInfo | null>(null)
const selectedSourceType = ref('115')
const selectedAccountId = ref(0)
const isEditMode = ref(false)
const isSelectSource = ref(false)
const tempVideoExt = ref('')

// 表单引用
const addFormRef = ref<FormInstance>()
const editFormRef = ref<FormInstance>()

// 表单数据
const addForm = reactive<ScrapePath>({
  source_type: '115',
  account_id: 0,
  media_type: 'movie',
  source_path: '',
  source_path_id: '',
  dest_path: '',
  dest_path_id: '',
  scrape_type: 'scrape_and_rename',
  rename_type: 'same',
  enable_category: false,
  folder_name_template: '{title}({year})',
  file_name_template: '{title}({year})',
  delete_keyword: [],
  min_video_file_size: 0,
  video_ext_list: ['.mp4', '.mkv', '.avi', '.mov', '.wmv', '.webm', '.flv', '.ts', '.m4v', '.iso', '.rmvb', '.strm'],
  exclude_no_image_actor: false,
  enable_ai: 'off',
  ai_prompt: '',
  enable_cron: false,
  enable_fanart_tv: false,
})

const editForm = reactive<ScrapePath>({
  id: 0,
  source_type: '115',
  account_id: 0,
  media_type: 'movie',
  source_path: '',
  source_path_id: '',
  dest_path: '',
  dest_path_id: '',
  scrape_type: 'scrape_and_rename',
  rename_type: 'same',
  enable_category: false,
  folder_name_template: '{title}({year})',
  file_name_template: '{title}({year})',
  delete_keyword: [],
  min_video_file_size: 0,
  video_ext_list: [],
  exclude_no_image_actor: false,
  enable_ai: 'off',
  ai_prompt: '',
  enable_cron: false,
  enable_fanart_tv: false,
})

// 删除关键词选项
const deleteKeywords = [
  { label: '1080p', value: '1080p' },
  { label: '720p', value: '720p' },
  { label: '4K', value: '4K' },
  { label: 'HDR', value: 'HDR' },
  { label: '杜比', value: '杜比' },
  { label: 'HDR10', value: 'HDR10' },
  { label: 'HDR10+', value: 'HDR10+' },
  { label: 'SDR', value: 'SDR' },
  { label: 'HLG', value: 'HLG' },
  { label: 'Dolby Vision', value: 'Dolby Vision' },
  { label: 'DV', value: 'DV' },
  { label: 'WEBRip', value: 'WEBRip' },
  { label: 'WEB-DL', value: 'WEB-DL' },
  { label: 'BluRay', value: 'BluRay' },
  { label: 'BDRip', value: 'BDRip' },
  { label: 'REMUX', value: 'REMUX' },
  { label: 'HDTV', value: 'HDTV' },
  { label: 'HDDVD', value: 'HDDVD' },
  { label: 'DVD', value: 'DVD' },
  { label: 'CAM', value: 'CAM' },
  { label: 'TS', value: 'TS' },
  { label: 'TC', value: 'TC' },
  { label: 'R5', value: 'R5' },
  { label: 'R6', value: 'R6' },
  { label: 'SCR', value: 'SCR' },
  { label: 'HQ', value: 'HQ' },
  { label: 'HD', value: 'HD' },
  { label: 'BD', value: 'BD' },
  { label: 'X264', value: 'X264' },
  { label: 'X265', value: 'X265' },
  { label: 'H264', value: 'H264' },
  { label: 'H265', value: 'H265' },
  { label: 'AAC', value: 'AAC' },
  { label: 'AC3', value: 'AC3' },
  { label: 'DTS', value: 'DTS' },
  { label: 'DDP', value: 'DDP' },
  { label: 'DD', value: 'DD' },
  { label: 'TrueHD', value: 'TrueHD' },
  { label: 'Atmos', value: 'Atmos' },
  { label: 'DTS-X', value: 'DTS-X' },
  { label: 'IMAX', value: 'IMAX' },
  { label: 'EXTENDED', value: 'EXTENDED' },
  { label: 'THEATRICAL', value: 'THEATRICAL' },
  { label: 'UNRATED', value: 'UNRATED' },
  { label: 'REPACK', value: 'REPACK' },
  { label: 'PROPER', value: 'PROPER' },
  { label: 'LIMITED', value: 'LIMITED' },
  { label: 'INTERNAL', value: 'INTERNAL' },
  { label: 'SUBBED', value: 'SUBBED' },
  { label: 'DUBBED', value: 'DUBBED' },
  { label: 'CHS', value: 'CHS' },
  { label: 'CHT', value: 'CHT' },
  { label: 'JPN', value: 'JPN' },
  { label: 'ENG', value: 'ENG' },
]

// 表单验证规则
const addFormRules = reactive<FormRules>({
  source_type: [{ required: true, message: '请选择同步源类型', trigger: 'change' }],
  account_id: [{ required: true, message: '请选择网盘账号', trigger: 'change' }],
  media_type: [{ required: true, message: '请选择媒体类型', trigger: 'change' }],
  source_path_id: [{ required: true, message: '请选择来源路径', trigger: 'change' }],
  scrape_type: [{ required: true, message: '请选择操作方式', trigger: 'change' }],
  rename_type: [{ required: true, message: '请选择整理方式', trigger: 'change' }],
  dest_path_id: [{ required: true, message: '请选择目标路径', trigger: 'change' }],
  folder_name_template: [{ required: true, message: '请输入文件夹重命名模板', trigger: 'blur' }],
  file_name_template: [{ required: true, message: '请输入文件重命名模板', trigger: 'blur' }],
})

const editFormRules = reactive<FormRules>({
  source_type: [{ required: true, message: '请选择同步源类型', trigger: 'change' }],
  account_id: [{ required: true, message: '请选择网盘账号', trigger: 'change' }],
  media_type: [{ required: true, message: '请选择媒体类型', trigger: 'change' }],
  source_path_id: [{ required: true, message: '请选择来源路径', trigger: 'change' }],
  scrape_type: [{ required: true, message: '请选择操作方式', trigger: 'change' }],
  rename_type: [{ required: true, message: '请选择整理方式', trigger: 'change' }],
  dest_path_id: [{ required: true, message: '请选择目标路径', trigger: 'change' }],
  folder_name_template: [{ required: true, message: '请输入文件夹重命名模板', trigger: 'blur' }],
  file_name_template: [{ required: true, message: '请输入文件重命名模板', trigger: 'blur' }],
})

// 监听添加表单媒体类型变化
watch(() => addForm.media_type, (newType) => {
  if (newType === 'other') {
    addForm.scrape_type = 'only_rename' // 当媒体类型为'other'时，操作方式固定为'only_rename'
    addForm.folder_name_template = '{num}'
    addForm.file_name_template = '{num}'
  }
})

// 监听添加表单操作方式变化
watch(() => addForm.scrape_type, (newType) => {
  if (newType === 'only_scrape') {
    addForm.rename_type = 'same' // 当操作方式为'only_scrape'时，整理方式固定为'same'
    addForm.enable_category = false // 当操作方式为'only_scrape'时，二级分类固定为false
  }
})

// 监听编辑表单媒体类型变化
watch(() => editForm.media_type, (newType) => {
  if (newType === 'other') {
    editForm.scrape_type = 'only_rename' // 当媒体类型为'other'时，操作方式固定为'only_rename'
  }
})

// 监听编辑表单操作方式变化
watch(() => editForm.scrape_type, (newType) => {
  if (newType === 'only_scrape') {
    editForm.rename_type = 'same' // 当操作方式为'only_scrape'时，整理方式固定为'same'
    editForm.enable_category = false // 当操作方式为'only_scrape'时，二级分类固定为false
  }
})

// 获取账号名称
const getAccountName = (accountId?: number): string => {
  if (!accountId) return ''
  const account = accounts.value.find((a: { id: number; }) => a.id === accountId)
  return account ? account.name : ''
}

// 获取媒体类型文本
const getMediaTypeText = (mediaType: string): string => {
  const typeMap: Record<string, string> = {
    movie: '电影',
    tvshow: '电视剧',
    other: '其他',
  }
  return typeMap[mediaType] || mediaType
}

// 获取操作方式文本
const getScrapeTypeText = (scrapeType: string): string => {
  const typeMap: Record<string, string> = {
    only_scrape: '仅刮削',
    scrape_and_rename: '刮削和整理',
    only_rename: '仅整理',
  }
  return typeMap[scrapeType] || scrapeType
}

// 获取整理方式文本
const getRenameTypeText = (renameType: string): string => {
  const typeMap: Record<string, string> = {
    move: '移动',
    copy: '复制',
    soft_symlink: '软链接',
    hard_symlink: '硬链接',
    same: "-"
  }
  return typeMap[renameType] || renameType
}

// 视频扩展名相关
const addVideoExt = (formType: string) => {
  if (!tempVideoExt.value.trim()) return

  const ext = tempVideoExt.value.trim()
  if (formType === 'add' && !addForm.video_ext_list.includes(ext)) {
    addForm.video_ext_list.push(ext)
  } else if (formType === 'edit' && !editForm.video_ext_list.includes(ext)) {
    editForm.video_ext_list.push(ext)
  }
  tempVideoExt.value = ''
}

const removeVideoExt = (index: number, formType: string) => {
  if (formType === 'add') {
    addForm.video_ext_list.splice(index, 1)
  } else {
    editForm.video_ext_list.splice(index, 1)
  }
}

// 加载刮削目录列表
const loadPathes = async () => {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/scrape/pathes`)

    if (response?.data.code === 200) {
      pathes.value = response.data.data || []
    } else {
      ElMessage.error(response?.data.message || '加载刮削目录失败')
      pathes.value = []
    }
  } catch {
    console.error('加载刮削目录错误')
    ElMessage.error('加载刮削目录失败')
    pathes.value = []
  } finally {
    loading.value = false
  }
}

const updatePathesStatus = async () => {
  loading.value = true
  const response = await http?.get(`${SERVER_URL}/scrape/pathes`)

  if (response?.data.code === 200) {
    for (const p of response?.data?.data || []) {
      const path = pathes.value.find(pa => pa.id === p.id)
      if (path) {
        path.is_renaming = p.is_renaming
        path.is_scraping = p.is_scraping
      }
    }
  }
}

// 加载账号列表
const loadAccounts = async (sourceType?: string) => {
  try {
    accountsLoading.value = true
    const response = await http?.get(`${SERVER_URL}/account/list`, {
      params: { source_type: sourceType },
    })

    if (response?.data.code === 200) {
      accounts.value = response.data.data || []
    } else {
      ElMessage.error(response?.data.message || '加载账号列表失败')
      accounts.value = []
    }
  } catch {
    console.error('加载账号列表错误')
    ElMessage.error('加载账号列表失败')
    accounts.value = []
  } finally {
    accountsLoading.value = false
  }
}

// 处理添加刮削目录
const handleAdd = async () => {
  if (!addFormRef.value) return
  if (addForm.scrape_type !== 'only_scrape' && (addForm.dest_path_id == '')) {
    ElMessage.error('请选择目标路径且填写文件夹重命名模板和文件重命名模板')
    return
  }
  try {
    await addFormRef.value.validate()
    addLoading.value = true

    const response = await http?.post(`${SERVER_URL}/scrape/pathes`, {
      id: 0,
      source_type: addForm.source_type,
      account_id: addForm.source_type !== 'local' ? addForm.account_id : undefined,
      media_type: addForm.media_type,
      source_path: addForm.source_path,
      source_path_id: addForm.source_path_id,
      dest_path: addForm.dest_path,
      dest_path_id: addForm.dest_path_id,
      scrape_type: addForm.scrape_type,
      rename_type: addForm.rename_type,
      enable_category: addForm.enable_category,
      folder_name_template: addForm.folder_name_template,
      file_name_template: addForm.file_name_template,
      delete_keyword: addForm.delete_keyword,
      min_video_file_size: addForm.min_video_file_size,
      video_ext_list: addForm.video_ext_list,
      exclude_no_image_actor: addForm.exclude_no_image_actor,
      enable_ai: addForm.enable_ai,
      ai_prompt: addForm.ai_prompt,
      force_delete_source_path: false,
      enable_cron: addForm.enable_cron,
      enable_fanart_tv: addForm.enable_fanart_tv,
    })

    if (response?.data.code === 200) {
      ElMessage.success('添加刮削目录成功')
      showAddDialog.value = false
      loadPathes()
      resetAddForm()
    } else {
      ElMessage.error(response?.data.message || '添加刮削目录失败')
    }
  } catch (error) {
    console.error('添加刮削目录错误', error)
    ElMessage.error('添加刮削目录失败')
  } finally {
    addLoading.value = false
  }
}

// 重置添加表单
const resetAddForm = () => {
  addForm.source_type = '115'
  addForm.account_id = 0
  addForm.media_type = 'movie'
  addForm.source_path = ''
  addForm.source_path_id = ''
  addForm.dest_path = ''
  addForm.dest_path_id = ''
  addForm.scrape_type = 'scrape_and_rename'
  addForm.rename_type = 'same'
  addForm.enable_category = false
  addForm.folder_name_template = '{title}({year})'
  addForm.file_name_template = '{title}({year})'
  addForm.delete_keyword = []
  addForm.min_video_file_size = 0
  addForm.video_ext_list = [".mp4", ".mkv", ".avi", ".mov", ".wmv", ".webm", ".flv", ".avi", ".ts", ".m4v", ".iso", ".rmvb", ".strm"]
  tempVideoExt.value = ''
  addForm.exclude_no_image_actor = false
  addForm.enable_ai = 'off'
  addForm.ai_prompt = ''

  if (addFormRef.value) {
    addFormRef.value.clearValidate()
  }
}

// 处理编辑刮削目录
const handleEdit = (row: ScrapePath) => {
  // 设置编辑表单的值
  editForm.id = row.id || 0
  editForm.source_type = row.source_type
  editForm.account_id = row.account_id || 0
  editForm.media_type = row.media_type
  editForm.source_path = row.source_path
  editForm.source_path_id = row.source_path_id
  editForm.dest_path = row.dest_path
  editForm.dest_path_id = row.dest_path_id
  editForm.scrape_type = row.scrape_type
  editForm.rename_type = row.rename_type
  editForm.enable_category = row.enable_category
  editForm.folder_name_template = row.folder_name_template
  editForm.file_name_template = row.file_name_template
  editForm.delete_keyword = [...row.delete_keyword]
  editForm.min_video_file_size = row.min_video_file_size || 0
  editForm.video_ext_list = [...(row.video_ext_list || ['mp4', 'mkv', 'avi', 'wmv', 'flv', 'mov', 'webm'])]
  tempVideoExt.value = ''
  editForm.exclude_no_image_actor = row.exclude_no_image_actor || false
  editForm.enable_ai = row.enable_ai || 'off'
  editForm.ai_prompt = row.ai_prompt || ''
  editForm.enable_cron = row.enable_cron || false
  editForm.enable_fanart_tv = row.enable_fanart_tv || false
  showEditDialog.value = true
}

// 处理保存编辑
const handleEditSave = async () => {
  if (!editFormRef.value) return
  if (editForm.scrape_type !== 'only_scrape' && (editForm.dest_path_id == '')) {
    ElMessage.error('请选择目标路径')
    return
  }
  try {
    await editFormRef.value.validate()
    editLoading.value = true

    const response = await http?.post(`${SERVER_URL}/scrape/pathes`, {
      id: editForm.id,
      source_path: editForm.source_path,
      source_path_id: editForm.source_path_id,
      dest_path: editForm.dest_path,
      dest_path_id: editForm.dest_path_id,
      scrape_type: editForm.scrape_type,
      rename_type: editForm.rename_type,
      enable_category: editForm.enable_category,
      folder_name_template: editForm.folder_name_template,
      file_name_template: editForm.file_name_template,
      delete_keyword: editForm.delete_keyword,
      min_video_file_size: editForm.min_video_file_size,
      video_ext_list: editForm.video_ext_list,
      exclude_no_image_actor: editForm.exclude_no_image_actor,
      enable_ai: editForm.enable_ai,
      ai_prompt: editForm.ai_prompt,
      force_delete_source_path: false,
      enable_cron: editForm.enable_cron,
      enable_fanart_tv: editForm.enable_fanart_tv,
    })

    if (response?.data.code === 200) {
      ElMessage.success('编辑刮削目录成功')
      showEditDialog.value = false
      loadPathes()
    } else {
      ElMessage.error(response?.data.message || '编辑刮削目录失败')
    }
  } catch (error) {
    console.error('编辑刮削目录错误', error)
    ElMessage.error('编辑刮削目录失败')
  } finally {
    editLoading.value = false
  }
}

// 处理删除刮削目录
const handleDelete = async (row: ScrapePath, index: number) => {
  try {
    await ElMessageBox.confirm('确定要删除这个刮削目录吗？', '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    if (pathes.value[index]) {
      pathes.value[index].deleting = true
    }

    const response = await http?.delete(`${SERVER_URL}/scrape/pathes/${row.id}`)

    if (response?.data.code === 200) {
      ElMessage.success('删除刮削目录成功')
      loadPathes()
    } else {
      ElMessage.error(response?.data.message || '删除刮削目录失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除刮削目录错误')
      ElMessage.error('删除刮削目录失败')
    }
  } finally {
    if (pathes.value[index]) {
      pathes.value[index].deleting = false
    }
  }
}

// 处理扫描操作
const handleScan = async (row: ScrapePath) => {
  if (!http) return

  try {
    row.scanning = true
    await http.post(`${SERVER_URL}/scrape/pathes/start`, { id: row.id })
    ElMessage.success('任务已开始')
  } catch (error) {
    ElMessage.error('任务启动失败')
    console.error('Scan error:', error)
  } finally {
    row.scanning = false
  }
}

// 处理停止操作
const handleStop = async (row: ScrapePath) => {
  if (!http) return

  try {
    row.scanning = true
    await http.post(`${SERVER_URL}/scrape/pathes/stop`, { id: row.id })
    ElMessage.success('任务已停止')
  } catch (error) {
    ElMessage.error('任务停止失败')
    console.error('Stop error:', error)
  } finally {
    row.scanning = false
  }
}

// 打开目录选择器
const openDirSelector = (isSource: boolean = false) => {
  showDirDialog.value = true
  tempSelectedDir.value = null
  selectedSourceType.value = addForm.source_type
  selectedAccountId.value = Number(addForm.account_id) || 0
  isEditMode.value = false
  isSelectSource.value = isSource
}

// 打开编辑模式的目录选择器
const openEditDirSelector = (isSource: boolean = false) => {
  showDirDialog.value = true
  tempSelectedDir.value = null
  selectedSourceType.value = editForm.source_type
  selectedAccountId.value = editForm.account_id
  isEditMode.value = true
  isSelectSource.value = isSource
}

// 确认选择目录
const confirmSelectDir = () => {
  if (!tempSelectedDir.value) return

  if (isEditMode.value) {
    if (isSelectSource.value) {
      editForm.source_path = tempSelectedDir.value.path
      editForm.source_path_id = tempSelectedDir.value.id
    } else {
      editForm.dest_path = tempSelectedDir.value.path
      editForm.dest_path_id = tempSelectedDir.value.id
    }
  } else {
    if (isSelectSource.value) {
      addForm.source_path = tempSelectedDir.value.path
      addForm.source_path_id = tempSelectedDir.value.id
    } else {
      addForm.dest_path = tempSelectedDir.value.path
      addForm.dest_path_id = tempSelectedDir.value.id
    }
  }

  showDirDialog.value = false
}

// 添加自动刷新相关变量
const autoRefreshTimer = ref<number | null>(null)
// 检查并设置自动刷新
const checkAndSetAutoRefresh = () => {
  // 清除已存在的定时器
  if (autoRefreshTimer.value) {
    clearInterval(autoRefreshTimer.value)
    autoRefreshTimer.value = null
  }

  // 设置定时器，每隔2秒刷新一次
  autoRefreshTimer.value = window.setInterval(() => {
    // 只改状态
    updatePathesStatus()
  }, 2000)
}

// 组件挂载时加载数据
onMounted(async () => {
  await loadPathes()
  checkAndSetAutoRefresh()
  // 加载默认同步源类型的账号，确保网盘账号的source_type和所选的同步源类型一致
  await loadAccounts(addForm.source_type !== 'local' ? addForm.source_type : undefined)

  // 监听窗口大小变化更新移动端状态
  const handleResize = () => {
    checkIsMobile.value = isMobile()
  }

  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  // 组件卸载时清除定时器
  if (autoRefreshTimer.value) {
    clearInterval(autoRefreshTimer.value)
    autoRefreshTimer.value = null
  }
})
</script>

<style scoped>
.scrape-paths-page {
  padding: 20px;
}

.scrape-paths-card {
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.scrape-paths-table {
  width: 100%;
}

.empty-state {
  text-align: center;
  padding: 40px 0;
}

.scrape-form .el-form-item {
  margin-bottom: 18px;
}

.video-ext-container .el-tag {
  margin-right: 10px;
  margin-bottom: 10px;
}

.dir-selector {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 500px;
}

.info-value {
  display: flex;
  align-items: center;
}
</style>
