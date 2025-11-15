<template>
  <div class="main-content-container scrape-pathes-container full-width-container">
    <div class="card-header">
      <div class="header-left">
        <h2 class="card-title hidden-md-and-down">刮削目录管理</h2>
        <div class="card-subtitle">
          设置媒体文件的刮削和整理规则，支持电影、电视剧和其他媒体类型
        </div>
        <div class="card-subtitle">
          当前字幕等其他文件的处理方式都是移动到目标位置，不受整理方式的影响，请特别注意
        </div>
        <div class="card-subtitle">
          如果开启了定时任务则是每13分钟运行一次<br />
          刮削默认并发数为5，并发数越高越快，但是也会增加TMDB或者网盘限制的概率。<br />
        </div>
        <el-alert type="warning" :show-icon="true" style="margin-top: 12px">
          来源路径请按照电影和电视剧区分，不要设置成同一个文件夹。
        </el-alert>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="showAddDialog = true">
          <el-icon>
            <Plus />
          </el-icon>
          添加刮削目录
        </el-button>
      </div>
    </div>
    <div style="
        width: 100%;
        height: 100%;
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        justify-content: start;
        align-items: top;
      ">
      <el-card style="min-width: 320px" shadow="hover" v-for="(row, index) in pathes" :key="row.id || index">
        <template #header>
          <div class="card-header">
            <div class="card-title">
              <el-tooltip class="box-item" :content="'目录ID：' + row.id" placement="bottom">
                #{{ row.id }} {{ row.source_path }}
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
            <span class="info-value">{{ getAccountName(row.account_id) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">媒体类型:</span>
            <span class="info-value">{{ getMediaTypeText(row.media_type) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">目标路径:</span>
            <span class="info-value">{{ row.scrape_type === "only_scrape" ? "-" : row.dest_path }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">操作方式:</span>
            <span class="info-value">{{ getScrapeTypeText(row.scrape_type) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">整理方式:</span>
            <span class="info-value">{{ getRenameTypeText(row.rename_type) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">二级分类:</span>
            <span class="info-value">{{ row.enable_category ? '开启' : '关闭' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">定时同步:</span>
            <span class="info-value">
              <el-switch v-model="row.enable_cron" @change="toggleCron(row)" inline-prompt active-text="开启"
                inactive-text="关闭" style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949" />
            </span>
          </div>
          <div class="info-item">
            <span class="info-label">创建时间:</span>
            <span class="info-value">{{ formatTime(row.created_at || 0) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">运行状态:</span>
            <span class="info-value" v-if="row.is_running === 2">
              <el-icon class="is-loading">
                <Loading />
              </el-icon>
              <el-text class="mx-1" type="primary">运行中...</el-text>
            </span>
            <span class="info-value" v-if="row.is_running === 1">
              <el-icon class="is-loading">
                <Star />
              </el-icon>
              <el-text class="mx-1" type="primary">等待运行...</el-text>
            </span>
            <span class="info-value" v-if="row.is_running === 0">未执行</span>
          </div>
        </div>
        <template #footer>
          <div class="card-actions">
            <el-button type="success" size="small" @click="handleScan(row)" :loading="row.scanning" :icon="Refresh"
              v-if="row.is_running === 0">启动</el-button>
            <el-button type="warning" size="small" @click="handleStop(row)" :loading="row.scanning" :icon="VideoPause"
              v-if="row.is_running !== 0">停止</el-button>
            <el-button type="primary" size="small" @click="handleEdit(row)" :loading="row.editing"
              :icon="Edit">编辑</el-button>
            <el-button type="danger" size="small" @click="handleDelete(row, index)" :loading="row.deleting"
              :icon="Delete">删除</el-button>
          </div>
        </template>
      </el-card>

      <el-col v-if="pathes.length === 0 && !loading" :span="24" class="empty-card-col">
        <el-card shadow="never" class="empty-card">
          <div class="empty-content">
            <el-icon class="empty-icon">
              <Folder />
            </el-icon>
            <p class="empty-text">暂无刮削目录</p>
          </div>
        </el-card>
      </el-col>
    </div>

    <!-- 添加刮削目录对话框 -->
    <el-dialog v-model="showAddDialog" title="添加刮削目录" :width="checkIsMobile ? '90%' : '600px'"
      :close-on-click-modal="false">
      <el-form ref="addFormRef" :model="addForm" :rules="addFormRules" label-width="140px"
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
            仅整理：不刮削元数据，仅通过查询到的信息进行整理（重命名方式根据整理方式决定）；其他类型必须有nfo(因为没地方查询信息)
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
          <MetadataExtInput v-model="addForm.delete_keyword" placeholder="输入关键词后按回车添加"
            class="meta-ext-input limited-width-input" :autoAddDot="false" />
          <div class="form-tip">从视频文件名中提取影视剧标题时先删除这些关键词，添加的越多识别准确率越高</div>
        </el-form-item>
        <el-form-item label="最小视频文件大小" prop="min_video_file_size">
          <el-input-number v-model="addForm.min_video_file_size" :min="0" :step="1" style="width: 100%"
            placeholder="请输入最小视频文件大小" :disabled="addLoading"></el-input-number>
          <div class="form-tip">单位：MB，小于此值的视频文件将被忽略</div>
        </el-form-item>
        <el-form-item label="视频文件扩展名" prop="video_ext_list">
          <MetadataExtInput v-model="addForm.video_ext_list" placeholder="输入视频文件扩展名，回车添加"
            class="meta-ext-input limited-width-input" />
          <div class="form-tip">支持的视频文件扩展名，用于筛选视频文件</div>
        </el-form-item>
        <el-form-item label="过滤无头像演员" prop="exclude_no_image_actor">
          <el-switch v-model="addForm.exclude_no_image_actor" :active-value="true" :inactive-value="false"
            :disabled="addLoading" />
          <div class="form-tip">没有头像的演员不会加入到nfo文件中</div>
        </el-form-item>
        <el-form-item label="删除整理完的非空路径" prop="exclude_no_image_actor">
          <el-switch v-model="editForm.force_delete_source_path" :active-value="true" :inactive-value="false"
            :disabled="addLoading" />
          <div class="form-tip">整理完成是否强制删除源文件所在路径（一般会遗留广告垃圾文件），如果禁用只会删除空目录</div>
        </el-form-item>
        <el-form-item label="刮削线程数" prop="max_threads">
          <el-input-number v-model="addForm.max_threads" :disabled="addLoading" min="1"
            :max="addForm.source_type === 'local' ? 20 : 5" step="1" style="width: 100%" />
          <div class="form-help">刮削本地文件时的最大并发线程数，越高越快, 刮削网盘该值无效。默认值为5; 只有本地目录类型可以修改</div>
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
        <div class="dialog-footer">
          <el-button @click="showAddDialog = false">取消</el-button>
          <el-button type="primary" @click="handleAdd" :loading="addLoading"> 确定 </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑刮削目录对话框 -->
    <el-dialog v-model="showEditDialog" title="编辑刮削目录" :width="checkIsMobile ? '90%' : '600px'"
      :close-on-click-modal="false">
      <el-form ref="editFormRef" :model="editForm" :rules="editFormRules" label-width="140px"
        :label-position="checkIsMobile ? 'top' : 'left'">
        <el-form-item label="同步源类型" prop="source_type">
          <el-radio-group v-model="editForm.source_type" placeholder="请选择同步源类型" disabled>
            <el-radio-button v-for="option in sourceTypeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </el-radio-button>
          </el-radio-group>
          <div class="form-tip">选择用于刮削的同步源类型</div>
        </el-form-item>
        <el-form-item label="网盘账号" prop="account_id" v-if="editForm.source_type !== 'local'">
          <el-select v-model="editForm.account_id" placeholder="请选择网盘账号" :loading="accountsLoading" disabled>
            <el-option v-for="account in accounts" :key="account.id" :label="account.name"
              :value="account.id"></el-option>
          </el-select>
          <div class="form-tip">选择用于刮削的网盘账号</div>
        </el-form-item>
        <el-form-item label="媒体类型" prop="media_type">
          <el-select v-model="editForm.media_type" placeholder="请选择媒体类型" disabled>
            <el-option label="电影" value="movie"></el-option>
            <el-option label="电视剧" value="tvshow"></el-option>
            <el-option label="其他" value="other" disabled></el-option>
          </el-select>
          <div class="form-tip">其他类型只能整理不能刮削</div>
        </el-form-item>
        <el-form-item label="来源路径" prop="source_path_id" v-if="
          (editForm.source_type !== 'local' && editForm.account_id) ||
          editForm.source_type === 'local'
        ">
          <div class="pan-dir-input">
            <el-input v-model="editForm.source_path_id" placeholder="点击选择按钮选择目录" :disabled="editLoading" readonly />
            <el-button type="primary" @click="openEditDirSelector(true)" :disabled="editLoading">
              选择目录
            </el-button>
          </div>
          <div v-if="editForm.source_path != ''" class="selected-path-inline">
            <span class="path-label">选中目录路径：</span>
            <code class="path-url">{{ editForm.source_path }}</code>
          </div>
          <div class="form-tip">选择要刮削的源目录</div>
        </el-form-item>
        <el-form-item label="目标路径" prop="dest_path_id" v-if="
          ((editForm.source_type !== 'local' && editForm.account_id) || editForm.source_type === 'local') && editForm.scrape_type !== 'only_scrape'
        ">
          <div class="pan-dir-input">
            <el-input v-model="editForm.dest_path_id" placeholder="点击选择按钮选择目标目录" :disabled="editLoading" readonly />
            <el-button type="primary" @click="openEditDirSelector(false)" :disabled="editLoading">
              选择目录
            </el-button>
          </div>
          <div v-if="editForm.dest_path != ''" class="selected-path-inline">
            <span class="path-label">选中目录路径：</span>
            <code class="path-url">{{ editForm.dest_path }}</code>
          </div>
          <div class="form-tip">选择刮削后文件的存放位置</div>
        </el-form-item>
        <el-form-item label="操作方式" prop="scrape_type">
          <el-radio-group v-model="editForm.scrape_type">
            <el-radio-button label="only_scrape" :disabled="editForm.media_type === 'other'">仅刮削</el-radio-button>
            <el-radio-button label="scrape_and_rename"
              :disabled="editForm.media_type === 'other'">刮削和整理</el-radio-button>
            <el-radio-button label="only_rename">仅整理</el-radio-button>
          </el-radio-group>
          <div class="form-tip">选择要执行的操作类型</div>
        </el-form-item>
        <el-form-item label="整理方式" prop="rename_type" v-if="editForm.scrape_type !== 'only_scrape'">
          <el-radio-group v-model="editForm.rename_type">
            <el-radio-button label="move">移动</el-radio-button>
            <el-radio-button label="copy">复制</el-radio-button>
            <el-radio-button label="soft_symlink" :disabled="editForm.source_type !== 'local'">软链接</el-radio-button>
            <el-radio-button label="hard_symlink" :disabled="editForm.source_type !== 'local'">硬链接</el-radio-button>
          </el-radio-group>
          <div class="form-tip">选择文件整理的方式</div>
        </el-form-item>
        <el-form-item label="开启二级分类" prop="enable_category" v-if="editForm.scrape_type !== 'only_scrape'">
          <el-switch v-model="editForm.enable_category" :active-value="true" :inactive-value="false"
            :disabled="editLoading" />
          <div class="form-tip">是否按照二级分类策略组织文件</div>
        </el-form-item>
        <el-form-item label="文件夹重命名模板" prop="folder_name_template" v-if="editForm.scrape_type !== 'only_scrape'">
          <el-input v-model="editForm.folder_name_template" :disabled="editLoading" placeholder="留空保留原名称" />
          <div class="form-tip">文件夹重命名的模板格式</div>
        </el-form-item>
        <el-form-item label="文件重命名模板" prop="file_name_template" v-if="editForm.scrape_type !== 'only_scrape'">
          <el-input v-model="editForm.file_name_template" :disabled="editLoading" placeholder="留空保留原名称" />
          <div class="form-tip">文件重命名的模板格式</div>
        </el-form-item>
        <el-form-item label="要删除的关键词" prop="delete_keyword">
          <MetadataExtInput v-model="editForm.delete_keyword" placeholder="输入关键词后按回车添加"
            class="meta-ext-input limited-width-input" :autoAddDot="false" />
          <div class="form-tip">从视频文件名中提取影视剧标题时先删除这些关键词，添加的越多识别准确率越高</div>
        </el-form-item>
        <el-form-item label="最小视频文件大小" prop="min_video_file_size">
          <el-input-number v-model="editForm.min_video_file_size" :min="0" :step="1" style="width: 100%"
            placeholder="请输入最小视频文件大小" :disabled="editLoading"></el-input-number>
          <div class="form-tip">单位：MB，小于此值的视频文件将被忽略</div>
        </el-form-item>
        <el-form-item label="视频文件扩展名" prop="video_ext_list">
          <MetadataExtInput v-model="editForm.video_ext_list" placeholder="输入视频文件扩展名，回车添加"
            class="meta-ext-input limited-width-input" />
          <div class="form-tip">支持的视频文件扩展名，用于筛选视频文件</div>
        </el-form-item>
        <el-form-item label="排除无头像演员" prop="exclude_no_image_actor">
          <el-switch v-model="editForm.exclude_no_image_actor" :active-value="true" :inactive-value="false"
            :disabled="editLoading" />
          <div class="form-tip">是否排除无头像演员的影视剧</div>
        </el-form-item>
        <el-form-item label="删除整理完的非空路径" prop="exclude_no_image_actor">
          <el-switch v-model="editForm.force_delete_source_path" :active-value="true" :inactive-value="false"
            :disabled="editLoading" />
          <div class="form-tip">整理完成是否强制删除源文件所在路径（一般会遗留广告垃圾文件）</div>
        </el-form-item>
        <el-form-item label="刮削线程数" prop="max_threads">
          <el-input-number v-model="editForm.max_threads" :disabled="editLoading" min="1"
            :max="editForm.source_type === 'local' ? 20 : 5" step="1" style="width: 100%" />
          <div class="form-help">刮削本地文件时的最大并发线程数，越高越快, 刮削网盘该值无效。默认值为5; 只有本地目录类型可以修改</div>
        </el-form-item>
        <el-form-item label="开启AI识别" prop="enable_ai">
          <el-radio-group v-model="editForm.enable_ai" placeholder="请选择AI识别模式" :disabled="editLoading" size="large">
            <el-radio-button label="off">禁用</el-radio-button>
            <el-radio-button label="assist">辅助识别</el-radio-button>
            <el-radio-button label="enforce">强制使用</el-radio-button>
          </el-radio-group>
          <div class="form-tip">是否开启AI识别，开启后会根据AI提示识别影视剧标题</div>
        </el-form-item>
        <el-form-item label="提示词" prop="ai_prompt">
          <el-input v-model="editForm.ai_prompt" type="textarea" placeholder="请输入AI提示词"
            :disabled="editLoading || editForm.enable_ai === 'off'" :rows="4" maxlength="1000" />
          <div class="form-help">用于指导AI进行媒体识别的提示词，如果不清楚如何设置请留空。<br />文件名变量为{filename}</div>
        </el-form-item>
        <el-form-item label="定时同步" prop="enable_cron">
          <el-switch v-model="editForm.enable_cron" :active-value="true" :inactive-value="false"
            :disabled="editLoading" />
          <div class="form-tip">是否启用定时同步功能</div>
        </el-form-item>
        <el-form-item label="启用fanart.tv" prop="enable_fanart_tv" v-if="editForm.media_type == 'movie'">
          <el-switch v-model="editForm.enable_fanart_tv" :active-value="true" :inactive-value="false"
            :disabled="editLoading" />
          <div class="form-tip">是否启用fanart.tv的高清图下载，下载很慢会降低刮削效率。</div>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showEditDialog = false">取消</el-button>
          <el-button type="primary" @click="handleEditSave" :loading="editLoading"> 确定 </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 目录选择对话框 -->
    <el-dialog v-model="showDirDialog" :title="isSelectingLocalPath ? '选择目标目录' : '选择来源目录'"
      :width="checkIsMobile ? '90%' : '600px'" :close-on-click-modal="false" :before-close="handleCloseDirDialog">
      <div class="dir-selector">
        <el-scrollbar height="400px">
          <div v-if="dirTreeLoading" class="loading-container">
            <el-icon class="is-loading">
              <Loading />
            </el-icon>
            <p>加载中...</p>
          </div>
          <div v-else-if="dirTreeData.length === 0" class="empty-container">
            <p>暂无目录</p>
          </div>
          <div v-else class="dir-list">
            <div v-for="dir in dirTreeData" :key="dir.id" class="dir-item" @click="selectTempDir(dir)">
              <el-icon>
                <Folder />
              </el-icon>
              <span class="dir-name">{{ dir.name }}</span>
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
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject, onMounted, ref, reactive, watch, onUnmounted } from 'vue'
// import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Loading, Folder, Edit, Delete, Refresh, Star, VideoPause } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import { formatTime } from '@/utils/timeUtils'
import { isMobile } from '@/utils/deviceUtils'
import { sourceTypeOptions, sourceTypeTagMap, sourceTypeMap } from '@/utils/sourceTypeUtils'
import MetadataExtInput from './MetadataExtInput.vue'
import 'element-plus/theme-chalk/display.css'

// const router = useRouter()
const defaultAiPrompt = "从文件名中提取出电影名称、年份; 名称中不能有特殊字符如点、下划线、横杠、斜杠等; 如果文件中有tmdbid（格式{tmdbid-123455}）也返回tmdbid\n"
const defaultAiPrompSuffix = '\n输出格式：请严格按照以下JSON格式输出，不要添加任何其他内容：{"name": "提取的影视剧名称", "year": 年份或0}\n现在请处理文件名：{{filename}}'
interface ScrapePath {
  id?: number
  source_type: string
  account_id?: number
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
  created_at?: number
  updated_at?: number
  deleting: boolean
  editing: boolean
  scanning: boolean
  enable_ai: string
  ai_prompt: string
  exclude_no_image_actor: boolean
  force_delete_source_path: boolean
  enable_cron?: boolean
  enable_fanart_tv: boolean
  is_running: number
  max_threads: number
}

interface DirInfo {
  id: string
  name: string
  path: string
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
const pathes = ref<ScrapePath[]>([])
const loading = ref(false)

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
const selectedAccountId = ref(0)
const isSelectSource = ref(true)
const editSelectedDirPath = ref('')
const isDirDialogClosed = ref(false) // 标记目录选择对话框是否已关闭

// 添加对话框状态
const showAddDialog = ref(false)
const addLoading = ref(false)
const addFormRef = ref<FormInstance>();
const addForm = reactive({
  source_type: '115', // 默认选择115网盘
  account_id: '',
  media_type: 'movie', // 默认选择电影
  source_path: '',
  source_path_id: '',
  dest_path: '',
  dest_path_id: '',
  scrape_type: 'scrape_and_rename', // 默认选择刮削和整理
  rename_type: 'move', // 默认移动
  enable_category: false,
  folder_name_template: '{title} ({year})',
  file_name_template: '{title} ({year})',
  delete_keyword: [] as string[],
  min_video_file_size: 0, // 最小视频文件大小，单位MB
  video_ext_list: [".mp4", ".mkv", ".avi", ".mov", ".wmv", ".webm", ".flv", ".avi", ".ts", ".m4v", ".iso", ".rmvb", ".strm"], // 视频文件扩展名列表
  exclude_no_image_actor: false, // 默认不排除无头像演员
  enable_ai: 'off', // 默认禁用AI识别
  ai_prompt: '', // 默认AI提示
  force_delete_source_path: false, // 默认强制删除来源路径
  enable_cron: false, // 默认不启用定时同步
  enable_fanart_tv: false, // 默认不启用fanart.tv
  max_threads: 5 // 默认5个线程
})

// 编辑对话框状态
const showEditDialog = ref(false)
const editLoading = ref(false)
const editFormRef = ref<FormInstance>();
const editForm = reactive({
  id: 0,
  source_type: '',
  account_id: 0,
  media_type: 'movie', // 默认选择电影
  source_path: '',
  source_path_id: '',
  dest_path: '',
  dest_path_id: '',
  scrape_type: 'scrape_and_rename', // 默认选择刮削和整理
  rename_type: 'move', // 默认移动
  enable_category: false,
  folder_name_template: '',
  file_name_template: '',
  delete_keyword: [] as string[],
  min_video_file_size: 0, // 最小视频文件大小，单位MB
  video_ext_list: [".mp4", ".mkv", ".avi", ".mov", ".wmv", ".webm", ".flv", ".avi", ".ts", ".m4v", ".iso", ".rmvb", ".strm"], // 视频文件扩展名列表
  exclude_no_image_actor: false, // 默认不排除无头像演员
  enable_ai: 'off', // 默认禁用AI识别
  ai_prompt: '', // 默认AI提示
  force_delete_source_path: false, // 默认强制删除来源路径
  enable_cron: false, // 默认不启用定时同步
  enable_fanart_tv: false, // 默认不启用fanart.tv
  max_threads: 5 // 默认5个线程
})

// 表单验证规则
const addFormRules: FormRules = {
  source_type: [{ required: true, message: '请选择同步源类型', trigger: 'change' }],
  account_id: [{ required: true, message: '请选择网盘账号', trigger: 'change' }],
  media_type: [{ required: true, message: '请选择媒体类型', trigger: 'change' }],
  source_path: [
    { required: true, message: '请选择来源目录', trigger: 'blur' },
    { min: 1, max: 500, message: '长度在 1 到 500 个字符', trigger: 'blur' },
  ],
  source_path_id: [{ required: true, message: '请选择来源目录ID', trigger: 'blur' }],
  dest_path_id: [{ required: true, message: '请选择目标目录ID', trigger: 'blur' }],
  scrape_type: [{ required: true, message: '请选择操作方式', trigger: 'change' }],
  rename_type: [
    {
      required: addForm.scrape_type !== 'only_scrape',
      message: '请选择整理方式',
      trigger: 'change'
    }
  ],
  min_video_file_size: [{ type: 'number', min: 0, message: '最小视频文件大小必须大于等于0', trigger: 'change' }],
  video_ext_list: [{ type: 'array', required: true, message: '请至少添加一个视频文件扩展名', trigger: 'change' }]
}

// 编辑表单验证规则
const editFormRules: FormRules = {
  media_type: [{ required: true, message: '请选择媒体类型', trigger: 'change' }],
  source_path: [
    { required: true, message: '请选择来源目录', trigger: 'blur' },
    { min: 1, max: 500, message: '长度在 1 到 500 个字符', trigger: 'blur' },
  ],
  scrape_type: [{ required: true, message: '请选择操作方式', trigger: 'change' }],
  rename_type: [
    {
      required: editForm.scrape_type !== 'only_scrape',
      message: '请选择整理方式',
      trigger: 'change'
    }
  ],
  min_video_file_size: [{ type: 'number', min: 0, message: '最小视频文件大小必须大于等于0', trigger: 'change' }],
}

// 监听添加表单媒体类型变化
watch(() => addForm.media_type, (newType) => {
  if (newType === 'other') {
    addForm.scrape_type = 'only_rename' // 当媒体类型为'other'时，操作方式固定为'only_rename'
  }
  // 根据媒体类型设置默认的文件重命名模板
  if (newType === 'movie') {
    addForm.folder_name_template = '{title} ({year})'
    addForm.file_name_template = '{title} ({year})'
  } else if (newType === 'tvshow') {
    if (addForm.scrape_type === 'only_rename') {
      addForm.scrape_type = 'scrape_and_rename' // 当媒体类型为'tvshow'且操作方式为'only_rename'时，整理方式固定为'scrape_and_rename'
    }
    addForm.folder_name_template = '{title} ({year})'
    addForm.file_name_template = '{title} - {season_episode} - 第 {episode_number} 集'
  } else if (newType === "other") {
    addForm.folder_name_template = '{actors}/{num}'
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
  const account = accounts.value.find(a => a.id === accountId)
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
const tempVideoExt = ref('')

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
        path.is_running = p.is_running
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
      force_delete_source_path: addForm.force_delete_source_path,
      enable_cron: addForm.enable_cron,
      enable_fanart_tv: addForm.enable_fanart_tv,
      max_threads: addForm.max_threads,
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
  addForm.account_id = ''
  addForm.media_type = 'movie'
  addForm.source_path = ''
  addForm.source_path_id = ''
  addForm.dest_path = ''
  addForm.dest_path_id = ''
  addForm.scrape_type = 'scrape_and_rename'
  addForm.rename_type = 'same'
  addForm.enable_category = false
  // 根据默认媒体类型设置默认模板
  addForm.delete_keyword = []
  addForm.min_video_file_size = 0
  addForm.video_ext_list = [".mp4", ".mkv", ".avi", ".mov", ".wmv", ".webm", ".flv", ".avi", ".ts", ".m4v", ".iso", ".rmvb", ".strm"]
  tempVideoExt.value = ''
  selectedDirPath.value = ''
  addForm.exclude_no_image_actor = false
  addForm.enable_ai = 'off'
  addForm.ai_prompt = ''
  addForm.enable_cron = false
  addForm.enable_fanart_tv = false
  addForm.max_threads = 5
  addForm.force_delete_source_path = false



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
  editForm.video_ext_list = row.video_ext_list
  editSelectedDirPath.value = row.source_path
  tempVideoExt.value = ''
  editForm.exclude_no_image_actor = row.exclude_no_image_actor || false
  editForm.enable_ai = row.enable_ai || 'off'
  editForm.ai_prompt = row.ai_prompt || ''
  editForm.enable_cron = row.enable_cron || false
  editForm.enable_fanart_tv = row.enable_fanart_tv || false
  editForm.max_threads = row.max_threads || 5
  editForm.force_delete_source_path = row.force_delete_source_path || false
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
      force_delete_source_path: editForm.force_delete_source_path,
      enable_cron: editForm.enable_cron,
      enable_fanart_tv: editForm.enable_fanart_tv,
      max_threads: parseInt(editForm.max_threads + ""),
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



// 处理定时同步开关切换
const toggleCron = async (row: ScrapePath) => {
  try {
    const formData = {
      id: row.id || 0,
    }

    const response = await http?.post(`${SERVER_URL}/scrape/pathes/toggle-cron`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success(row.enable_cron ? '开启定时同步成功' : '关闭定时同步成功')
    } else {
      // 如果失败，恢复原来的状态
      row.enable_cron = !row.enable_cron
      ElMessage.error(response?.data.message || '切换定时同步状态失败')
    }
  } catch {
    console.error('切换定时同步状态错误')
    // 如果失败，恢复原来的状态
    row.enable_cron = !row.enable_cron
    ElMessage.error('切换定时同步状态失败')
  }
}

// 打开目录选择器
const openDirSelector = async (isSource: boolean = false) => {
  showDirDialog.value = true
  tempSelectedDir.value = null
  currentDir.value = null
  selectedSourceType.value = addForm.source_type
  selectedAccountId.value = parseInt(addForm.account_id as string) || 0
  isEditMode.value = false
  isSelectSource.value = isSource
  // 重置目录选择对话框关闭状态
  isDirDialogClosed.value = false;

  await loadDirTree(null)
}

// 打开编辑模式的目录选择器
const openEditDirSelector = async (isSource: boolean = false) => {
  showDirDialog.value = true
  tempSelectedDir.value = null
  currentDir.value = null
  selectedSourceType.value = editForm.source_type
  selectedAccountId.value = editForm.account_id
  isEditMode.value = true
  isSelectSource.value = isSource
  isSelectingLocalPath.value = !isSource
  // 重置目录选择对话框关闭状态
  isDirDialogClosed.value = false;

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
    // 检查目录选择对话框是否已关闭
    if (isDirDialogClosed.value) {
      return;
    }
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
    // 检查目录选择对话框是否已关闭
    if (isDirDialogClosed.value) {
      return;
    }
    dirTreeLoading.value = false
  }
}

// 选择临时目录
const selectTempDir = async (dir: DirInfo) => {
  // 检查目录选择对话框是否已关闭
  if (isDirDialogClosed.value) {
    return;
  }

  tempSelectedDir.value = dir
  // 如果选择了目录且不是本地路径，加载子目录
  await loadDirTree(dir)
}

// 确认选择目录 - 复用同步目录的来源路径逻辑
const confirmSelectDir = () => {
  // 标记目录选择对话框已关闭
  isDirDialogClosed.value = true;

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
  // 标记目录选择对话框已关闭
  isDirDialogClosed.value = true;
  showDirDialog.value = false
}

// 目录选择对话框取消事件处理
const handleCancelDirDialog = () => {
  // 标记目录选择对话框已关闭
  isDirDialogClosed.value = true;
  showDirDialog.value = false
}

// const startRenaming = async () => {
//   try {
//     await http?.post(`${SERVER_URL}/scrape/rename`)
//     ElMessage.success('整理已触发')
//   } catch (error) {
//     ElMessage.error('整理触发失败')
//     console.error('Rename error:', error)
//   }
// }

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
.full-width-container {
  padding: 0;
}

.full-width-card {
  border-radius: 0;
  box-shadow: none;
}

.scrape-pathes-container {
  border: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 0 !important;
  flex-wrap: wrap;
}

.header-left {
  margin-bottom: 12px;
}

.card-body {
  min-height: 279px;
}

.info-value {
  display: flex;
  align-items: center;
}

.card-subtitle {
  font-size: 14px;
  margin-bottom: 16px !important;
}
</style>
