<template>
  <div class="main-content-container sync-directories-container full-width-container">
    <el-card shadow="none" class="full-width-card">
      <template #header>
        <div class="card-header">
          <div class="header-content">
            <div class="header-info">
              <h2 class="card-title">同步目录管理</h2>
              <div class="header-actions">
                <el-button type="primary" @click="showAddDialog = true">
                  <el-icon><Plus /></el-icon>
                  添加同步目录
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </template>
      <div
        style="
          width: 100%;
          height: 100%;
          display: flex;
          flex-wrap: wrap;
          gap: 6px;
          justify-content: start;
          align-items: top;
        "
      >
        <el-card
          style="min-width: 320px"
          shadow="hover"
          v-for="(row, index) in directories"
          :key="row.id || index"
        >
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-tooltip
                  class="box-item"
                  :content="'目录ID：' + row.base_cid"
                  placement="bottom"
                >
                  #{{ index + 1 }} {{ row.remote_path }}
                </el-tooltip>
              </div>
              <div>
                <el-tag :type="sourceTypeTagMap[row.source_type]">
                  {{ sourceTypeMap[row.source_type] }}
                </el-tag>
              </div>
            </div>
          </template>

          <div class="card-body">
            <div class="info-item" v-if="row.source_type !== 'local'">
              <span class="info-label">账号:</span>
              <span class="info-value">{{ row.account_name }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">目标路径:</span>
              <span class="info-value">{{ row.local_path }}</span>
            </div>

            <div class="info-item" v-if="row.source_type === '115'">
              <span class="info-label">缓存目录层级:</span>
              <span class="info-value">{{ row.dir_depth || '-' }}层</span>
            </div>

            <div class="info-item">
              <span class="info-label">添加时间:</span>
              <span class="info-value">{{ formatTime(row.created_at) }}</span>
            </div>

            <div class="info-item">
              <span class="info-label">修改时间:</span>
              <span class="info-value">{{ formatTime(row.updated_at) }}</span>
            </div>

            <div class="info-item">
              <el-tooltip
                class="box-item"
                effect="dark"
                content="开启后会根据strm设置中的cron表达式定时同步数据，如果该同步目录内的资源变动概率较小，建议关闭定时同步，然后有变动时手动同步"
                placement="bottom"
              >
                <span class="info-label">
                  <el-icon><Warning /></el-icon> 定时同步:
                </span>
              </el-tooltip>
              <el-switch
                v-model="row.enable_cron"
                :active-value="true"
                :inactive-value="false"
                @change="toggleCron(row)"
                active-color="#13ce66"
                inactive-color="#dcdfe6"
              />
            </div>
          </div>
          <template #footer>
            <div class="card-actions">
              <el-button
                type="success"
                size="small"
                @click="handleStart(row, index)"
                :loading="row.starting"
                :icon="VideoPlay"
                >启动同步</el-button
              >
              <el-button
                type="primary"
                size="small"
                @click="handleEdit(row)"
                :loading="row.editing"
                :icon="Edit"
                >编辑</el-button
              >
              <el-button
                type="danger"
                size="small"
                @click="handleDelete(row, index)"
                :loading="row.deleting"
                :icon="Delete"
                >删除</el-button
              >
            </div>
          </template>
        </el-card>

        <el-col v-if="directories.length === 0 && !loading" :span="24" class="empty-card-col">
          <el-card shadow="never" class="empty-card">
            <div class="empty-content">
              <el-icon class="empty-icon"><Folder /></el-icon>
              <p class="empty-text">暂无同步目录</p>
            </div>
          </el-card>
        </el-col>
      </div>
    </el-card>

    <!-- 添加同步目录对话框 -->
    <el-dialog
      v-model="showAddDialog"
      title="添加同步目录"
      :width="checkIsMobile ? '90%' : '600px'"
      :close-on-click-modal="false"
    >
      <el-form
        ref="addFormRef"
        :model="addForm"
        :rules="addFormRules"
        label-width="120px"
        :label-position="checkIsMobile ? 'top' : 'left'"
      >
        <el-form-item label="同步源类型" prop="source_type">
          <el-select
            v-model="addForm.source_type"
            placeholder="请选择同步源类型"
            @change="handleSourceTypeChange"
          >
            <el-option
              v-for="typeItem in sourceTypeOptions"
              :key="typeItem.value"
              :label="typeItem.label"
              :value="typeItem.value"
            ></el-option>
          </el-select>
          <div class="form-tip">
            <div v-if="addForm.source_type === 'local'">
              本地目录可以通过CD2间接支持更多网盘，请将CD2的本地挂载目录映射到容器中（如果使用docker）,然后选择该目录
            </div>
            <div v-if="addForm.source_type === '115'">需要先添加用于同步的115账号并授权</div>
            <div v-if="addForm.source_type === '123'">需要先添加用于同步的123账号并授权</div>
          </div>
        </el-form-item>
        <el-form-item label="网盘账号" prop="account_id" v-if="addForm.source_type !== 'local'">
          <el-select
            v-model="addForm.account_id"
            placeholder="请选择网盘账号"
            :loading="accountsLoading"
            :disabled="addLoading"
          >
            <el-option
              v-for="account in accounts"
              :key="account.id"
              :label="account.name"
              :value="account.id"
            ></el-option>
          </el-select>
          <div class="form-tip">选择用于同步的网盘账号</div>
        </el-form-item>
        <el-form-item
          label="来源路径"
          prop="base_cid"
          v-if="
            (addForm.source_type !== 'local' && addForm.account_id) ||
            addForm.source_type === 'local'
          "
        >
          <div class="pan-dir-input">
            <el-input
              v-model="addForm.base_cid"
              placeholder="点击选择按钮选择网盘目录"
              :disabled="addLoading"
              readonly
            />
            <el-button type="primary" @click="openDirSelector(false)" :disabled="addLoading">
              选择目录
            </el-button>
          </div>
          <div v-if="selectedDirPath" class="selected-path-inline">
            <span class="path-label">选中目录路径：</span>
            <code class="path-url">{{ selectedDirPath }}</code>
          </div>
          <div class="form-tip">选择网盘中要同步的目录</div>
        </el-form-item>
        <el-form-item
          label="目标路径"
          prop="local_path"
          v-if="
            (addForm.source_type !== 'local' && addForm.account_id) ||
            addForm.source_type === 'local'
          "
        >
          <div class="pan-dir-input">
            <el-input
              v-model="addForm.local_path"
              placeholder="点击选择按钮选择本地目录"
              :disabled="addLoading"
              readonly
            />
            <el-button type="primary" @click="openDirSelector(true)" :disabled="addLoading">
              选择目录
            </el-button>
          </div>
          <div class="form-tip">选择本地目录作为STRM文件的存放位置</div>
        </el-form-item>

        <el-form-item
          label="STRM存放目录"
          v-if="
            (addForm.source_type !== 'local' && addForm.account_id) ||
            addForm.source_type === 'local'
          "
        >
          <el-input
            v-model="addForm.strm_path"
            placeholder="自动计算：本地目录 + 选中目录路径"
            :disabled="true"
            readonly
          />
          <div class="form-tip">STRM和元数据实际存放目录（自动生成）</div>
        </el-form-item>

        <el-form-item label="缓存目录层级" prop="dir_depth" v-if="addForm.source_type === '115'">
          <el-input-number
            v-model="addForm.dir_depth"
            :min="1"
            :max="10"
            :step="1"
            :disabled="addLoading"
            style="width: 100%"
          />
          <p>该设置会严重影响全量同步所需时间，建议认真设置；如果不是很清楚，建议设置成2</p>
          <div class="form-tip">
            <p>
              如果所选网盘目录是AV类型的根目录，下面的目录结构如：小姐姐名/番号/番号.mkv，那就输入2。
            </p>
            <p>
              如果所选网盘目录是影视剧的根目录如Media，下面的子目录结构如：电影/动画电影/哪吒/哪吒.mkv，那就输入3；
            </p>
          </div>
        </el-form-item>

        <el-form-item label="是否自定义设置" prop="custom_config">
          <el-switch
            v-model="addForm.custom_config"
            :active-value="true"
            :inactive-value="false"
            :disabled="addLoading"
          />
          <div class="form-tip">
            开启后可自定义视频扩展名和元数据扩展名配置，否则使用strm设置中的值
          </div>
        </el-form-item>

        <el-form-item label="视频扩展名" prop="video_ext" v-if="addForm.custom_config">
          <el-input-tag
            v-model="addForm.video_ext"
            placeholder="输入扩展名后按回车添加，如：.mp4"
            :disabled="addLoading"
          />
          <div class="form-tip">指定需要生成STRM文件的视频文件扩展名</div>
        </el-form-item>

        <el-form-item label="元数据扩展名" prop="meta_ext" v-if="addForm.custom_config">
          <el-input-tag
            v-model="addForm.meta_ext"
            placeholder="输入扩展名后按回车添加，如：.nfo"
            :disabled="addLoading"
          />
          <div class="form-tip">指定需要同步的元数据文件扩展名</div>
        </el-form-item>
        <el-form-item label="排除文件名" prop="exclude_name" v-if="addForm.custom_config">
          <el-input-tag
            v-model="addForm.exclude_name"
            placeholder="输入文件名后按回车添加，如：.nfo"
            :disabled="addLoading"
          />
          <div class="form-tip">指定需要同步的元数据文件扩展名</div>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showAddDialog = false">取消</el-button>
          <el-button type="primary" @click="handleAdd" :loading="addLoading"> 确定 </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑同步目录对话框 -->
    <el-dialog
      v-model="showEditDialog"
      title="编辑同步目录"
      :width="checkIsMobile ? '90%' : '600px'"
      :close-on-click-modal="false"
    >
      <el-form
        ref="editFormRef"
        :model="editForm"
        :rules="editFormRules"
        label-width="120px"
        :label-position="checkIsMobile ? 'top' : 'left'"
      >
        <el-form-item label="来源路径" prop="base_cid">
          <div class="pan-dir-input">
            <el-input
              v-model="editForm.base_cid"
              placeholder="点击选择按钮选择115网盘目录"
              :disabled="editLoading"
              readonly
            />
            <el-button type="primary" @click="openEditDirSelector(false)" :disabled="editLoading">
              选择目录
            </el-button>
          </div>
          <div v-if="editSelectedDirPath" class="selected-path-inline">
            <span class="path-label">选中目录路径：</span>
            <code class="path-url">{{ editSelectedDirPath }}</code>
          </div>
          <div class="form-tip">选择115网盘中要同步的目录</div>
        </el-form-item>
        <el-form-item label="目标路径" prop="local_path">
          <div class="pan-dir-input">
            <el-input
              v-model="editForm.local_path"
              placeholder="点击选择按钮选择本地目录"
              :disabled="editLoading"
              readonly
            />
            <el-button type="primary" @click="openEditDirSelector(true)" :disabled="editLoading">
              选择目录
            </el-button>
          </div>
          <div class="form-tip">选择本地目录作为STRM文件的存放位置</div>
        </el-form-item>
        <el-form-item label="缓存目录层级" prop="dir_depth" v-if="editForm.source_type === '115'">
          <el-input-number
            v-model="editForm.dir_depth"
            :min="1"
            :max="10"
            :step="1"
            :disabled="editLoading"
            style="width: 100%"
          />
          <div class="form-tip">缓存目录层级，建议设置为 1-3 层（默认 2层）</div>
        </el-form-item>
        <el-form-item label="STRM存放目录">
          <el-input
            v-model="editForm.strm_path"
            placeholder="自动计算：本地目录 + 选中目录路径"
            :disabled="true"
            readonly
          />
          <div class="form-tip">STRM和元数据实际存放目录（自动生成）</div>
        </el-form-item>
        <el-form-item label="是否自定义设置" prop="custom_config">
          <el-switch
            v-model="editForm.custom_config"
            :active-value="true"
            :inactive-value="false"
            :disabled="editLoading"
          />
          <div class="form-tip">
            开启后可自定义视频扩展名和元数据扩展名配置，否则使用strm设置中的值
          </div>
        </el-form-item>

        <el-form-item label="视频扩展名" prop="video_ext" v-if="editForm.custom_config">
          <el-input-tag
            v-model="editForm.video_ext"
            placeholder="输入扩展名后按回车添加，如：.mp4"
            :disabled="editLoading"
          />
          <div class="form-tip">指定需要生成STRM文件的视频文件扩展名</div>
        </el-form-item>

        <el-form-item label="元数据扩展名" prop="meta_ext" v-if="editForm.custom_config">
          <el-input-tag
            v-model="editForm.meta_ext"
            placeholder="输入扩展名后按回车添加，如：.nfo"
            :disabled="editLoading"
          />
          <div class="form-tip">指定需要同步的元数据文件扩展名</div>
        </el-form-item>
        <el-form-item label="排除文件名" prop="exclude_name" v-if="editForm.custom_config">
          <el-input-tag
            v-model="editForm.exclude_name"
            placeholder="输入文件名后按回车添加，如：.nfo"
            :disabled="editLoading"
          />
          <div class="form-tip">指定需要同步的元数据文件扩展名</div>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showEditDialog = false">取消</el-button>
          <el-button type="primary" @click="handleEditSave" :loading="editLoading">
            确定
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 目录选择对话框 -->
    <el-dialog
      v-model="showDirDialog"
      :title="isSelectingLocalPath ? '选择目标目录' : '选择来源目录'"
      :width="checkIsMobile ? '90%' : '600px'"
      :close-on-click-modal="false"
    >
      <div class="dir-selector">
        <el-scrollbar height="400px">
          <div v-if="dirTreeLoading" class="loading-container">
            <el-icon class="is-loading"><Loading /></el-icon>
            <p>加载中...</p>
          </div>
          <div v-else-if="dirTreeData.length === 0" class="empty-container">
            <p>暂无目录</p>
          </div>
          <div v-else class="dir-list">
            <div
              v-for="dir in dirTreeData"
              :key="dir.id"
              class="dir-item"
              @click="selectTempDir(isSelectingLocalPath, dir)"
            >
              <el-icon><Folder /></el-icon>
              <span class="dir-name">{{ dir.name }}</span>
              <el-icon class="enter-icon"><ArrowRight /></el-icon>
            </div>
          </div>
        </el-scrollbar>

        <!-- 选中目录显示和确认区域 -->
        <div v-if="tempSelectedDir" class="selected-dir-section">
          <div class="selected-dir-info">
            <div class="selected-dir-label">当前选中目录：</div>
            <div class="selected-dir-path">{{ tempSelectedDir.path || tempSelectedDir.name }}</div>
          </div>
        </div>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showDirDialog = false">取消</el-button>
          <el-button type="primary" @click="confirmSelectDir" :disabled="!tempSelectedDir">
            确定选择
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject, onMounted, onUnmounted, ref, reactive, watch, type Ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Loading,
  Folder,
  ArrowRight,
  VideoPlay,
  Edit,
  Delete,
  Warning,
} from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import { formatTime } from '@/utils/timeUtils'
import { isMobile, onDeviceTypeChange } from '@/utils/deviceUtils'
import { sourceTypeOptions, sourceTypeTagMap, sourceTypeMap } from '@/utils/sourceTypeUtils'

interface SyncDirectory {
  id?: number
  base_cid: string
  local_path: string
  remote_path: string
  strm_path: string
  dir_depth?: number
  created_at: number
  updated_at: number
  deleting?: boolean
  editing?: boolean
  starting?: boolean
  source_type: string
  account_id?: number
  account_name: string
  custom_config?: boolean
  video_ext_arr?: string[]
  meta_ext_arr?: string[]
  exclude_name_arr?: string[]
  enable_cron?: boolean
}

interface DirInfo {
  id: string
  name: string
  path?: string
}

// 账户信息接口
interface CloudAccount {
  id: number
  name: string
  source_type: string
  user_id: string
  username: string
  created_at: number
  token: string
}

const http: AxiosStatic | undefined = inject('$http')

// 数据状态
const directories = ref<SyncDirectory[]>([])
const loading = ref(false)
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(9999)

// 账户列表状态
const accounts = ref<CloudAccount[]>([])
const accountsLoading = ref(false)

// 移动端检测
const checkIsMobile = ref(isMobile())

// 目录选择相关状态
const showDirDialog = ref(false)
const dirTreeData = ref<DirInfo[]>([])
const dirTreeLoading = ref(false)
const selectedDirPath = ref('')
const currentDir = ref<DirInfo | null>(null)
const tempSelectedDir = ref<DirInfo | null>(null)
const isEditMode = ref(false) // 标记是否为编辑模式
const isSelectingLocalPath = ref(false) // 标记是否为选择本地路径
const selectedSourceType = ref('')
const selectedAccountId: Ref<number | string> = ref(0)

// 检测是否为移动设备
const checkMobile = () => {
  checkIsMobile.value = isMobile()
}

// 添加对话框状态
const showAddDialog = ref(false)
const addLoading = ref(false)
const addFormRef = ref<FormInstance>()
const addForm = reactive({
  local_path: '',
  base_cid: '',
  strm_path: '',
  dir_depth: 2,
  source_type: '',
  account_id: '',
  custom_config: false,
  video_ext: [] as string[],
  meta_ext: [] as string[],
  exclude_name: [] as string[],
  remote_path: '',
})

// 编辑对话框状态
const showEditDialog = ref(false)
const editLoading = ref(false)
const editFormRef = ref<FormInstance>()
const editForm = reactive({
  id: 0,
  local_path: '',
  base_cid: '',
  strm_path: '',
  dir_depth: 2,
  source_type: '',
  account_id: 0,
  custom_config: false,
  video_ext: [] as string[],
  meta_ext: [] as string[],
  exclude_name: [] as string[],
  remote_path: '',
})
const editSelectedDirPath = ref('')

// 表单验证规则
const addFormRules: FormRules = {
  local_path: [
    { required: true, message: '请输入本地目录路径', trigger: 'blur' },
    { min: 1, max: 500, message: '长度在 1 到 500 个字符', trigger: 'blur' },
  ],
  base_cid: [
    { required: true, message: '请输入CID', trigger: 'blur' },
    { min: 1, max: 100, message: '长度在 1 到 100 个字符', trigger: 'blur' },
  ],
  dir_depth: [
    { required: true, message: '请输入目录深度', trigger: 'blur' },
    { type: 'number', min: 1, max: 10, message: '目录深度必须在 1 到 10 之间', trigger: 'blur' },
  ],
  source_type: [{ required: true, message: '请选择同步源类型', trigger: 'change' }],
  account_id: [{ required: true, message: '请选择网盘账号', trigger: 'change' }],
}

// 编辑表单验证规则
const editFormRules: FormRules = {
  local_path: [
    { required: true, message: '请输入本地目录路径', trigger: 'blur' },
    { min: 1, max: 500, message: '长度在 1 到 500 个字符', trigger: 'blur' },
  ],
  base_cid: [
    { required: true, message: '请输入CID', trigger: 'blur' },
    { min: 1, max: 100, message: '长度在 1 到 100 个字符', trigger: 'blur' },
  ],
  dir_depth: [
    { required: true, message: '请输入目录深度', trigger: 'blur' },
    { type: 'number', min: 1, max: 10, message: '目录深度必须在 1 到 10 之间', trigger: 'blur' },
  ],
  account_id: [{ required: true, message: '请选择网盘账号', trigger: 'change' }],
}

// 加载同步目录列表
const loadDirectories = async () => {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/path-list`, {
      params: {
        page: currentPage.value,
        page_size: pageSize.value,
      },
    })

    if (response?.data.code === 200) {
      directories.value = response.data.data.list || []
      total.value = response.data.data.total || 0
    } else {
      ElMessage.error(response?.data.msg || '加载同步目录失败')
      directories.value = []
      total.value = 0
    }
  } catch {
    console.error('加载同步目录错误')
    ElMessage.error('加载同步目录失败')
    directories.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

// 处理添加同步目录
const handleAdd = async () => {
  if (!addFormRef.value) return

  try {
    await addFormRef.value.validate()
    addLoading.value = true

    const formData = {
      local_path: addForm.local_path.trim(),
      base_cid: addForm.base_cid.trim(),
      remote_path: selectedDirPath.value,
      dir_depth: addForm.dir_depth,
      source_type: addForm.source_type.trim(),
      account_id: addForm.account_id ? addForm.account_id : 0,
      custom_config: addForm.custom_config,
      video_ext: addForm.video_ext,
      meta_ext: addForm.meta_ext,
      exclude_name: addForm.exclude_name,
    }

    const response = await http?.post(`${SERVER_URL}/sync/path-add`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success('添加同步目录成功')
      showAddDialog.value = false
      addForm.local_path = ''
      addForm.base_cid = ''
      addForm.strm_path = ''
      addForm.dir_depth = 2
      addForm.custom_config = false
      addForm.video_ext = []
      addForm.meta_ext = []
      selectedDirPath.value = ''
      loadDirectories()
    } else {
      ElMessage.error(response?.data.msg || '添加同步目录失败')
    }
  } catch {
    console.error('添加同步目录错误')
    ElMessage.error('添加同步目录失败')
  } finally {
    addLoading.value = false
  }
}

// 处理编辑同步目录
const handleEdit = async (row: SyncDirectory) => {
  editForm.id = row.id || 0
  editForm.local_path = row.local_path
  editForm.base_cid = row.base_cid
  editForm.dir_depth = row.dir_depth || 2
  editForm.source_type = row.source_type || ''
  editForm.account_id = row.account_id || 0
  editForm.custom_config = row.custom_config || false
  editForm.video_ext = row.video_ext_arr || []
  editForm.meta_ext = row.meta_ext_arr || []
  editForm.exclude_name = row.exclude_name_arr || []
  editSelectedDirPath.value = row.remote_path || ''

  // 初始化STRM路径
  updateEditStrmPath()

  showEditDialog.value = true
}

// 处理编辑保存
const handleEditSave = async () => {
  if (!editFormRef.value) return

  try {
    await editFormRef.value.validate()
    editLoading.value = true

    const formData = {
      id: editForm.id,
      local_path: editForm.local_path.trim(),
      base_cid: editForm.base_cid.trim(),
      strm_path: editForm.strm_path.trim(),
      dir_depth: editForm.dir_depth,
      custom_config: editForm.custom_config,
      video_ext: editForm.video_ext,
      meta_ext: editForm.meta_ext,
      exclude_name: editForm.exclude_name,
      source_type: editForm.source_type.trim(),
      remote_path: editSelectedDirPath.value,
    }

    const response = await http?.post(`${SERVER_URL}/sync/path-update`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success('编辑同步目录成功')
      showEditDialog.value = false
      editForm.id = 0
      editForm.local_path = ''
      editForm.base_cid = ''
      editForm.strm_path = ''
      editForm.dir_depth = 2
      editForm.custom_config = false
      editForm.video_ext = []
      editForm.meta_ext = []
      editSelectedDirPath.value = ''
      loadDirectories()
    } else {
      ElMessage.error(response?.data.msg || '编辑同步目录失败')
    }
  } catch {
    console.error('编辑同步目录错误')
    ElMessage.error('编辑同步目录失败')
  } finally {
    editLoading.value = false
  }
}

// 处理删除同步目录
const handleDelete = async (row: SyncDirectory, index: number) => {
  try {
    await ElMessageBox.confirm(
      `该操作将删除所有strm和元数据文件，确定要删除同步目录 "${row.local_path}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )

    directories.value[index].deleting = true

    const formData = {
      id: row.id || '',
    }

    const response = await http?.post(`${SERVER_URL}/sync/path-delete`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success('删除同步目录成功')
      loadDirectories()
    } else {
      ElMessage.error(response?.data.message || '删除同步目录失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除同步目录错误')
      ElMessage.error('删除同步目录失败')
    }
  } finally {
    if (directories.value[index]) {
      directories.value[index].deleting = false
    }
  }
}

// 处理启动同步目录
const handleStart = async (row: SyncDirectory, index: number) => {
  try {
    directories.value[index].starting = true

    const formData = {
      id: row.id || '',
    }

    const response = await http?.post(`${SERVER_URL}/sync/path/start`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success(`同步目录 "${row.local_path}" 启动成功`)
    } else {
      ElMessage.error(response?.data.msg || '启动同步目录失败')
    }
  } catch {
    console.error('启动同步目录错误')
    ElMessage.error('启动同步目录失败')
  } finally {
    if (directories.value[index]) {
      directories.value[index].starting = false
    }
  }
}

// 处理定时同步开关切换
const toggleCron = async (row: SyncDirectory) => {
  try {
    const formData = {
      id: row.id || '',
    }

    const response = await http?.post(`${SERVER_URL}/sync/path/toggle-cron`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success(row.enable_cron ? '开启定时同步成功' : '关闭定时同步成功')
    } else {
      // 如果失败，恢复原来的状态
      row.enable_cron = !row.enable_cron
      ElMessage.error(response?.data.msg || '切换定时同步状态失败')
    }
  } catch {
    console.error('切换定时同步状态错误')
    // 如果失败，恢复原来的状态
    row.enable_cron = !row.enable_cron
    ElMessage.error('切换定时同步状态失败')
  }
}

// 打开目录选择器
const openDirSelector = async (isLocalPath: boolean = false) => {
  showDirDialog.value = true
  tempSelectedDir.value = null
  currentDir.value = null
  selectedSourceType.value = isLocalPath ? 'local' : addForm.source_type
  isSelectingLocalPath.value = isLocalPath
  selectedAccountId.value = addForm.account_id

  await loadDirTree(isLocalPath ? 'local' : addForm.source_type, '')
}

// 加载目录树
const loadDirTree = async (sourceType: string, dirId: string) => {
  try {
    dirTreeLoading.value = true
    // 加载网盘目录树
    const accountIdToUse = selectedAccountId.value
    const response = await http?.get(`${SERVER_URL}/path/list`, {
      params: {
        parent_id: dirId,
        source_type: sourceType,
        account_id: accountIdToUse,
      },
    })

    if (response?.data.code === 200) {
      dirTreeData.value = response.data.data || []
    }
  } catch (error) {
    console.error('加载目录树错误:', error)
  } finally {
    dirTreeLoading.value = false
  }
}

// 临时选择目录（点击目录时）
const selectTempDir = async (isLocal: boolean, dir: DirInfo) => {
  tempSelectedDir.value = dir
  currentDir.value = dir

  // 加载该目录的子目
  await loadDirTree(selectedSourceType.value, dir.id)
}

// 计算STRM存放目录
const calculateStrmPath = (localPath: string, dirPath: string): string => {
  if (!localPath || !dirPath) return ''

  // 移除本地路径末尾的斜杠
  const cleanLocalPath = localPath.replace(/[/\\]+$/, '')
  // 移除目录路径开头的斜杠并规范化路径分隔符
  const cleanDirPath = dirPath.replace(/^[/\\]+/, '').replace(/\//g, '\\')

  return cleanDirPath ? `${cleanLocalPath}/${cleanDirPath}` : cleanLocalPath
}

// 更新添加表单的STRM路径
const updateAddStrmPath = () => {
  if (addForm.source_type !== 'local') {
    addForm.strm_path = calculateStrmPath(addForm.local_path, selectedDirPath.value)
  } else {
    addForm.strm_path = addForm.local_path
  }
}

// 更新编辑表单的STRM路径
const updateEditStrmPath = () => {
  if (editForm.source_type !== 'local') {
    editForm.strm_path = calculateStrmPath(editForm.local_path, editSelectedDirPath.value)
  } else {
    editForm.strm_path = editForm.local_path
  }
}

// 确认选择目录
const confirmSelectDir = async () => {
  if (!tempSelectedDir.value) return

  const selectedDir = tempSelectedDir.value

  if (isSelectingLocalPath.value) {
    // 选择本地路径：更新local_path字段
    if (isEditMode.value) {
      // 编辑模式
      editForm.local_path = selectedDir.path ? selectedDir.path : selectedDir.name
    } else {
      // 添加模式
      addForm.local_path = selectedDir.path ? selectedDir.path : selectedDir.name
    }
  } else {
    // 选择网盘路径：更新base_cid字段
    if (isEditMode.value) {
      // 编辑模式：设置编辑表单的CID值和显示路径
      editForm.base_cid = selectedDir.id
      editSelectedDirPath.value = selectedDir.path ? selectedDir.path : selectedDir.name
      // 更新编辑表单的STRM路径
      updateEditStrmPath()
    } else {
      // 添加模式：设置添加表单的CID值和显示路径
      addForm.base_cid = selectedDir.id
      selectedDirPath.value = selectedDir.path ? selectedDir.path : selectedDir.name
      // 更新添加表单的STRM路径
      updateAddStrmPath()
    }
  }

  showDirDialog.value = false
  tempSelectedDir.value = null
  currentDir.value = null
  isSelectingLocalPath.value = false
}

// 编辑时打开目录选择器
const openEditDirSelector = async (isLocalPath: boolean = false) => {
  isEditMode.value = true
  isSelectingLocalPath.value = isLocalPath
  showDirDialog.value = true
  tempSelectedDir.value = null
  currentDir.value = null
  selectedSourceType.value = isLocalPath ? 'local' : editForm.source_type
  selectedAccountId.value = editForm.account_id

  await loadDirTree(
    isLocalPath ? 'local' : editForm.source_type,
    isLocalPath ? editForm.local_path : editForm.base_cid,
  )
}

// 监听添加表单本地路径变化
watch(
  () => addForm.local_path,
  () => {
    updateAddStrmPath()
  },
)

// 监听编辑表单本地路径变化
watch(
  () => editForm.local_path,
  () => {
    updateEditStrmPath()
  },
)

const handleSourceTypeChange = () => {
  if (addForm.source_type !== 'local') {
    loadAccounts()
  }
}

// 加载账户列表
const loadAccounts = async () => {
  accounts.value = []
  try {
    accountsLoading.value = true
    const response = await http?.get(`${SERVER_URL}/account/list`)
    if (response?.data.code === 200) {
      const data = response.data.data || []
      for (const account of data) {
        if (account.username === '' || account.source_type !== addForm.source_type) continue
        accounts.value.push(account)
      }
    } else {
      console.error('加载账号列表失败:', response?.data.msg || '未知错误')
      accounts.value = []
    }
  } catch (error) {
    console.error('加载账号列表失败:', error)
    accounts.value = []
  } finally {
    accountsLoading.value = false
  }
}

// 组件挂载时加载数据
let removeDeviceTypeListener: (() => void) | null = null

onMounted(() => {
  checkMobile()
  removeDeviceTypeListener = onDeviceTypeChange((newIsMobile) => {
    checkIsMobile.value = newIsMobile
  })
  loadDirectories()
})

onUnmounted(() => {
  if (removeDeviceTypeListener) {
    removeDeviceTypeListener()
  }
})
</script>

<style scoped>
.sync-directories-container {
  width: 100% !important;
  max-width: 100% !important;
  margin: 0;
  padding: 0;
}

/* 全宽度容器，突破父容器的padding限制 */
.full-width-container {
  margin: -20px !important;
  padding: 20px !important;
  width: calc(100% + 40px) !important;
  max-width: calc(100% + 40px) !important;
}

.full-width-card {
  width: 100%;
  max-width: 100%;
  border: 0;
}
.full-width-card .el-card__header {
  padding: 0 !important;
}

.card-header {
  margin: 0;
  padding: 0;
  display: flex;
  justify-content: space-between;
}

.header-content {
  display: flex;
  align-items: flex-start;
}

.header-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.header-actions {
  margin-top: 16px;
  display: flex;
  align-items: center;
}

.card-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.card-subtitle {
  margin: 0;
  font-size: 14px;
  color: #909399;
}

.directories-table {
  width: 100% !important;
  margin-bottom: 20px;
  overflow-x: auto;
}

/* 确保表格容器也占满宽度 */
.directories-table :deep(.el-table) {
  width: 100% !important;
}

.directories-table :deep(.el-table__inner-wrapper) {
  width: 100% !important;
}

.table-cell-wrapper {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.cell-label {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
}

.cid-text {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  color: #606266;
  word-break: break-all;
}

.path-text {
  color: #303133;
  word-break: break-all;
  font-size: 13px;
  line-height: 1.4;
}

/* 卡片列表基础样式 */
.directories-card-list {
  margin-bottom: 20px;
}

.directory-card-col {
  margin-bottom: 20px;
}

.directory-card {
  height: 100%;
  transition: all 0.3s;
}

.directory-card:hover {
  border-color: #409eff;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.card-actions {
  display: flex;
  justify-content: end;
  gap: 8px;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  /* height: 200px; */
}

.info-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: nowrap;
  gap: 4px;
}

.info-label {
  font-size: 12px;
  color: #606266;
}

.info-value {
  font-size: 16px;
  color: #303133;
  word-break: break-all;
  line-height: 1.5;
}

.info-value.cid-text {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  display: inline-block;
}

.info-value.path-text {
  font-size: 13px;
}

.empty-card-col {
  margin-bottom: 20px;
}

.empty-card {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  color: #909399;
  background: #fafafa;
}

.empty-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.empty-icon {
  font-size: 48px;
}

.empty-text {
  font-size: 16px;
}

.mobile-time-info {
  display: flex;
  gap: 12px;
  margin-top: 4px;
  font-size: 12px;
}

.time-item {
  display: flex;
  gap: 2px;
}

.time-label {
  color: #909399;
  font-weight: 500;
}

.time-value {
  color: #606266;
}

.pagination-container {
  display: flex;
  justify-content: center;
  padding: 20px 0;
  overflow-x: auto;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

/* 115网盘目录选择相关样式 */
.pan-dir-input {
  display: flex;
  gap: 8px;
  align-items: flex-start;
}

.pan-dir-input .el-input {
  flex: 1;
}

.selected-path-inline {
  margin-top: 8px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
}

.path-label {
  color: #909399;
  font-weight: 500;
}

.path-url {
  color: #606266;
  background: #fff;
  padding: 2px 6px;
  border-radius: 2px;
  border: 1px solid #dcdfe6;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

/* 目录选择对话框样式 */
.dir-selector {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.loading-container,
.empty-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #909399;
}

.loading-container .el-icon {
  font-size: 32px;
  margin-bottom: 8px;
}

.dir-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.dir-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
}

.dir-item:hover {
  background: #f5f7fa;
  border-color: #e4e7ed;
}

.dir-item .el-icon:first-child {
  color: #409eff;
  font-size: 16px;
  margin-right: 8px;
}

.dir-name {
  flex: 1;
  font-size: 14px;
  color: #303133;
  word-break: break-all;
}

.enter-icon {
  color: #c0c4cc;
  font-size: 14px;
  margin-left: 8px;
}

.selected-dir-section {
  padding: 16px;
  background: #f5f7fa;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
}

.selected-dir-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.selected-dir-label {
  font-size: 14px;
  font-weight: 500;
  color: #606266;
}

.selected-dir-path {
  font-size: 13px;
  color: #303133;
  padding: 8px 12px;
  background: #fff;
  border-radius: 4px;
  border: 1px solid #dcdfe6;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  word-break: break-all;
  line-height: 1.4;
}
</style>
