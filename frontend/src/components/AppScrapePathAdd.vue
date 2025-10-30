<template>
  <div class="scrape-paths-page">
    <el-button type="primary" @click="goBack" size="large" link>
      <el-icon>
        <ArrowLeft />
      </el-icon>
      返回刮削目录
    </el-button>
    <el-card class="scrape-paths-card">
      <template #header>
        <div class="card-header">

        </div>
      </template>

      <el-form ref="addFormRef" :model="addForm" :rules="addFormRules" label-width="160px"
        :label-position="checkIsMobile ? 'top' : 'left'">
        <el-form-item label="同步源类型" prop="source_type">
          <el-radio-group v-model="addForm.source_type" placeholder="请选择同步源类型">
            <el-radio-button v-for="typeItem in sourceTypeOptions" :key="typeItem.value" :value="typeItem.value">
              {{ typeItem.label }}
            </el-radio-button>
          </el-radio-group>
          <div class="form-tip">
            <div v-if="addForm.source_type === 'local'">本地目录路径</div>
            <div v-if="addForm.source_type === '115'">需要先添加用于同步的115账号并授权</div>
          </div>
        </el-form-item>
        <el-form-item label="网盘账号" prop="account_id" v-if="addForm.source_type !== 'local'">
          <el-select v-model="addForm.account_id" placeholder="请选择网盘账号" :loading="accountsLoading"
            :disabled="addLoading">
            <template v-for="account in accounts">
              <el-option v-if="account.source_type === addForm.source_type && account.token !== ''" :key="account.id"
                :label="account.name" :value="account.id"></el-option>
            </template>
          </el-select>
          <div class="form-tip">选择用于刮削的网盘账号</div>
        </el-form-item>
        <el-form-item label="媒体类型" prop="media_type">
          <el-radio-group v-model="addForm.media_type" placeholder="请选择媒体类型">
            <el-radio-button value="movie">电影</el-radio-button>
            <el-radio-button value="tvshow">电视剧</el-radio-button>
            <el-radio-button value="other">其他</el-radio-button>
          </el-radio-group>
          <div class="form-tip">其他：只能整理不能刮削</div>
        </el-form-item>
        <el-form-item label="操作方式" prop="scrape_type">
          <el-radio-group v-model="addForm.scrape_type">
            <el-radio-button value="only_scrape" :disabled="addForm.media_type === 'other'">仅刮削</el-radio-button>
            <el-radio-button value="scrape_and_rename"
              :disabled="addForm.media_type === 'other'">刮削和整理</el-radio-button>
            <el-radio-button value="only_rename" :disabled="addForm.media_type === 'tvshow'">仅整理</el-radio-button>
          </el-radio-group>
          <div class="form-tip">
            仅刮削：不改变文件路径和重命名，生成对应视频文件的nfo和下载封面等，不需要选择目标路径<br />
            刮削和整理：会根据刮削结果，改变文件路径和重命名，生成对应视频文件的nfo和下载封面等，需要选择目标路径<br />
            仅整理：默认来源路径内都是刮削好的，然后根据nfo中的内容将视频、封面、字幕等整理到目标路径（电视剧暂不支持）
          </div>
        </el-form-item>
        <el-form-item label="整理方式" prop="rename_type" v-if="addForm.scrape_type !== 'only_scrape'">
          <el-radio-group v-model="addForm.rename_type">
            <el-radio-button value="move">移动</el-radio-button>
            <el-radio-button value="copy">复制</el-radio-button>
            <el-radio-button value="soft_symlink" :disabled="addForm.source_type !== 'local'">软链接</el-radio-button>
            <el-radio-button value="hard_symlink" :disabled="addForm.source_type !== 'local'">硬链接</el-radio-button>
          </el-radio-group>
          <div class="form-tip">
            移动：将视频文件移动到目标路径，元数据（nfo、字幕等）也会直接生成或移动到目标路径<br />
            复制：将文件复制到目标路径，元数据（nfo、字幕等）也会直接生成或复制到目标路径<br />
            软链接：创建文件的软链接到目标路径，元数据（nfo、字幕等）也会直接生成或复制到目标路径<br />
            硬链接：创建文件的硬链接到目标路径，元数据（nfo、字幕等）也会直接生成或复制到目标路径
          </div>
        </el-form-item>
        <el-form-item label="来源路径" prop="source_path" v-if="
          (addForm.source_type !== 'local' && addForm.account_id) ||
          addForm.source_type === 'local'
        ">
          <div class="pan-dir-input">
            <el-input v-model="addForm.source_path" placeholder="点击选择按钮选择目录" :disabled="addLoading" readonly />
            <el-button type="primary" @click="openDirSelector(true)" :disabled="addLoading">
              选择目录
            </el-button>
          </div>
          <div v-if="addForm.source_path != ''" class="selected-path-inline">
            <span class="path-label">选中目录路径：</span>
            <code class="path-url">{{ addForm.source_path }}</code>
          </div>
          <div class="form-tip">选择要刮削的源目录, 会从该目录下找出所有视频文件进行刮削</div>
        </el-form-item>
        <el-form-item label="目标路径" prop="dest_path" v-if="
          ((addForm.source_type !== 'local' && addForm.account_id) || addForm.source_type === 'local') && addForm.scrape_type !== 'only_scrape'
        ">
          <div class="pan-dir-input">
            <el-input v-model="addForm.dest_path" placeholder="点击选择按钮选择目标目录" :disabled="addLoading" readonly />
            <el-button type="primary" @click="openDirSelector(false)" :disabled="addLoading">
              选择目录
            </el-button>
          </div>
          <div v-if="addForm.dest_path != ''" class="selected-path-inline">
            <span class="path-label">选中目录路径：</span>
            <code class="path-url">{{ addForm.dest_path }}</code>
          </div>
          <div class="form-tip">选择刮削后文件的存放位置</div>
        </el-form-item>
        <el-form-item label="开启二级分类" prop="enable_category" v-if="addForm.scrape_type !== 'only_scrape'">
          <el-switch v-model="addForm.enable_category" :active-value="true" :inactive-value="false"
            :disabled="addLoading" />
          <div class="form-tip">是否按照二级分类策略组织文件，开启后会在目标路径先创建二级分类目录</div>
        </el-form-item>
        <el-form-item label="文件夹重命名模板" prop="folder_name_template" v-if="addForm.scrape_type !== 'only_scrape'">
          <el-input v-model="addForm.folder_name_template" :disabled="addLoading" placeholder="留空保留原名称" />
          <div class="form-tip">详细请参考：<a
              href="https://github.com/qicfan/qmediasync/wiki/%E6%95%B4%E7%90%86%E6%96%87%E4%BB%B6%EF%BC%88%E5%A4%B9%EF%BC%89%E6%A8%A1%E6%9D%BF%E5%8F%94%E7%94%A8%E5%8F%98%E9%87%8F"
              target="_blank">文件夹重命名模板</a></div>
        </el-form-item>
        <el-form-item label="文件重命名模板" prop="file_name_template" v-if="addForm.scrape_type !== 'only_scrape'">
          <el-input v-model="addForm.file_name_template" :disabled="addLoading" placeholder="留空保留原名称" />
          <div class="form-tip">详细请参考：<a
              href="https://github.com/qicfan/qmediasync/wiki/%E6%95%B4%E7%90%86%E6%96%87%E4%BB%B6%EF%BC%88%E5%A4%B9%EF%BC%89%E6%A8%A1%E6%9D%BF%E5%8F%94%E7%94%A8%E5%8F%98%E9%87%8F"
              target="_blank">文件重命名模板</a></div>
        </el-form-item>
        <el-form-item label="要删除的关键词" prop="delete_keyword">
          <el-input-tag v-model="addForm.delete_keyword" placeholder="输入关键词后按回车添加" :disabled="addLoading" />
          <div class="form-tip">从视频文件名中提取影视剧标题时先删除这些关键词，添加的越多识别准确率越高</div>
        </el-form-item>
        <el-form-item label="最小视频文件大小" prop="min_video_file_size">
          <el-input-number v-model="addForm.min_video_file_size" :min="0" :step="1" style="width: 100%"
            placeholder="请输入最小视频文件大小" :disabled="addLoading"></el-input-number>
          <div class="form-tip">单位：MB，小于此值的视频文件将被忽略</div>
        </el-form-item>
        <el-form-item label="视频文件扩展名" prop="video_ext_list">
          <el-tag v-for="(tag, index) in addForm.video_ext_list" :key="index" closable
            @close="removeVideoExt(index, 'add')" class="mr-2 mb-2" :disabled="addLoading">
            {{ tag }}
          </el-tag>
          <el-input v-model="tempVideoExt" class="mt-2" placeholder="请输入视频文件扩展名，回车添加" @keyup.enter="addVideoExt('add')"
            clearable :disabled="addLoading" style="width: 100%"></el-input>
          <div class="form-tip">支持的视频文件扩展名，用于筛选视频文件</div>
        </el-form-item>
        <el-form-item label="过滤无头像演员" prop="exclude_no_image_actor">
          <el-switch v-model="addForm.exclude_no_image_actor" :active-value="true" :inactive-value="false"
            :disabled="addLoading" />
          <div class="form-tip">没有头像的演员不会加入到nfo文件中</div>
        </el-form-item>
        <el-form-item label="删除整理完的非空路径" prop="exclude_no_image_actor">
          <el-switch v-model="addForm.force_delete_source_path" :active-value="true" :inactive-value="false"
            :disabled="addLoading" />
          <div class="form-tip">整理完成是否强制删除源文件所在路径（一般会遗留广告垃圾文件），如果禁用只会删除空目录</div>
        </el-form-item>
        <el-form-item label="是否启用AI识别" prop="enable_ai">
          <el-radio-group v-model="addForm.enable_ai" placeholder="请选择AI识别模式" :disabled="addLoading" size="large">
            <el-radio-button label="off">禁用</el-radio-button>
            <el-radio-button label="assist">辅助识别</el-radio-button>
            <el-radio-button label="enforce">强制使用</el-radio-button>
          </el-radio-group>
          <div class="form-help">
            辅助识别：仅在无法通过其他方式识别时使用AI。每天会限额使用1000次，如果想要一直使用请申请自己的API Key。 <br />
            强制使用：只使用AI识别，必须使用自己的API Key。
          </div>
        </el-form-item>
        <el-form-item label="提示词" prop="ai_prompt">
          <el-input v-model="addForm.ai_prompt" type="textarea" placeholder="请输入AI提示词"
            :disabled="addLoading || addForm.enable_ai === 'off'" :rows="4" maxlength="1000" />
          <div class="form-help">
            用于指导AI进行媒体识别的提示词，如果不清楚如何设置请留空。<br />
            <span v-if="addForm.ai_prompt == ''">
              默认提示词：{{ defaultAiPrompt }}{{ addForm.ai_prompt }}{{ defaultAiPrompSuffix }}
            </span>
          </div>
        </el-form-item>
        <el-form-item label="定时同步" prop="enable_cron">
          <el-switch v-model="addForm.enable_cron" :active-value="true" :inactive-value="false"
            :disabled="addLoading" />
          <div class="form-tip">是否启用定时同步功能</div>
        </el-form-item>
        <el-form-item label="启用fanart.tv" prop="enable_fanart_tv" v-if="editForm.media_type == 'movie'">
          <el-switch v-model="addForm.enable_fanart_tv" :active-value="true" :inactive-value="false"
            :disabled="addLoading" />
          <div class="form-tip">是否启用fanart.tv的高清图下载，下载很慢会降低刮削效率。</div>
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
    </el-card>

    <!-- 目录选择对话框 -->
    <el-dialog v-model="showDirDialog" :title="isSelectSource ? '选择来源目录' : '选择目标目录'" width="600px"
      @close="handleCloseDirDialog">
      <div v-loading="dirTreeLoading" element-loading-text="加载中...">
        <div v-if="dirTreeData.length === 0" class="empty-state">
          <el-empty description="暂无目录数据" />
        </div>

        <div v-else class="dir-tree-container">
          <el-tree :data="dirTreeData" node-key="id" :props="dirTreeProps" :expand-on-click-node="false"
            :highlight-current="true" @node-click="selectTempDir">
            <template #default="{ node, data }">
              <span class="custom-tree-node">
                <span>
                  <el-icon v-if="data.is_dir">
                    <Folder />
                  </el-icon>
                  <el-icon v-else>
                    <Document />
                  </el-icon>
                  {{ node.label }}
                </span>
              </span>
            </template>
          </el-tree>
        </div>

        <div class="selected-dir-info" v-if="tempSelectedDir">
          <p>选中目录: {{ tempSelectedDir.path }}</p>
        </div>
      </div>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="handleCancelDirDialog">取消</el-button>
          <el-button type="primary" @click="confirmSelectDir" :disabled="!tempSelectedDir">
            确定选择
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { isMobile } from "@/utils/deviceUtils";

import { inject } from 'vue'
import type { AxiosStatic } from 'axios';
import type { CloudAccount, DirInfo } from '@/typing';
import { useRouter } from 'vue-router';
import { sourceTypeOptions } from '@/utils/sourceTypeUtils'
const http: AxiosStatic | undefined = inject('$http')
// 依赖注入
const SERVER_URL = inject('SERVER_URL')

const router = useRouter()
// const router = useRouter()
const defaultAiPrompt = "从文件名中提取出电影名称、年份; 名称中不能有特殊字符如点、下划线、横杠、斜杠等; 如果文件中有tmdbid（格式{tmdbid-123455}）也返回tmdbid\n"
const defaultAiPrompSuffix = '\n输出格式：请严格按照以下JSON格式输出，不要添加任何其他内容：{"name": "提取的影视剧名称", "year": 年份或0}\n现在请处理文件名：{{filename}}'

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
  force_delete_source_path: boolean
}

// 响应式变量
const pathes = ref<ScrapePath[]>([])
const loading = ref(false)
const accounts = ref<CloudAccount[]>([])
const accountsLoading = ref(false)
const checkIsMobile = ref(isMobile())
const showAddDialog = ref(false)
const addLoading = ref(false)
const showDirDialog = ref(false)
const dirTreeData = ref<DirInfo[]>([])
const dirTreeLoading = ref(false)
const tempSelectedDir = ref<DirInfo | null>(null)
const currentDir = ref<DirInfo | null>(null)
const selectedSourceType = ref('115')
const selectedAccountId = ref(0)
const isEditMode = ref(false)
const isSelectSource = ref(false)
const selectedDirPath = ref('')
const tempVideoExt = ref('')

// 表单引用
const addFormRef = ref<FormInstance>()

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
  force_delete_source_path: false,
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
  force_delete_source_path: false,
})

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

// 返回上一页
const goBack = () => {
  router.push({ name: 'scrape-pathes' })
}

// 目录树属性
const dirTreeProps = {
  label: 'name',
  children: 'children',
  isLeaf: 'is_dir',
}

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
  selectedDirPath.value = ''
  addForm.exclude_no_image_actor = false
  addForm.enable_ai = 'off'
  addForm.ai_prompt = ''

  if (addFormRef.value) {
    addFormRef.value.clearValidate()
  }
}
// 打开目录选择器
const openDirSelector = async (isSource: boolean = false) => {
  showDirDialog.value = true
  tempSelectedDir.value = null
  currentDir.value = null
  selectedSourceType.value = addForm.source_type
  selectedAccountId.value = parseInt(addForm.account_id + "") || 0
  isEditMode.value = false
  isSelectSource.value = isSource

  await loadDirTree(null)
}

// 加载目录树 - 复用同步目录的接口逻辑
const loadDirTree = async (dir: DirInfo | null) => {
  try {
    dirTreeLoading.value = true
    const response = await http?.get(`${SERVER_URL}/path/list`, {
      timeout: 30000,
      params: {
        source_type: selectedSourceType.value,
        account_id: selectedAccountId.value,
        parent_id: dir?.id || "",
        parent_path: dir?.path || "",
      },
    })
    if (response?.data.code === 200) {
      dirTreeData.value = response.data.data || []
    } else {
      ElMessage.error(response?.data.message || '加载目录失败')
      dirTreeData.value = []
    }
  } catch {
    console.error('加载目录树错误')
    ElMessage.error('加载目录失败')
    dirTreeData.value = []
  } finally {
    dirTreeLoading.value = false
  }
}

// 选择临时目录
const selectTempDir = async (dir: DirInfo) => {
  tempSelectedDir.value = dir
  // 如果选择了目录且不是本地路径，加载子目录
  await loadDirTree(dir)
}

// 确认选择目录 - 复用同步目录的来源路径逻辑
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

// 目录选择对话框关闭事件处理
const handleCloseDirDialog = () => {
  showDirDialog.value = false
}

// 目录选择对话框取消事件处理
const handleCancelDirDialog = () => {
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
  width: 100%;
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

.dir-tree-container {
  max-height: 300px;
  overflow-y: auto;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 10px;
}

.selected-dir-info {
  margin-top: 15px;
  padding: 10px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.custom-tree-node {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 14px;
  padding-right: 8px;
}

.info-value {
  display: flex;
  align-items: center;
}
</style>
