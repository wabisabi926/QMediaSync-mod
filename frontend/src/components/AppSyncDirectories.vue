<template>
  <div class="main-content-container sync-directories-container full-width-container">
    <el-card shadow="none" class="full-width-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <h2 class="card-title hidden-md-and-down">同步目录管理</h2>
            <p class="card-subtitle">
              115同步无法感知网盘的文件夹重命名等操作，如果发现文件夹名字不对可以手动点击：重置&同步 <br />
              百度网盘同步会只查询上次同步时间之后修改的文件列表，不会查询所有文件、无法感知文件和文件夹删除。
              增量同步只能单线程，每分钟最多执行8次请求，每次请求1000个文件，如果单次变更文件数量大于8000，同步就会很慢。
            </p>
            <p>
              115的"全量同步"操作会删除所有缓存数据（不会删除本地文件），然后执行同步，可以处理所有网盘文件变更 <br />
              百度网盘的"全量同步"操作会删除所有缓存数据（不会删除本地文件），然后递归查询所有文件（不会附加其他查询条件），如果“同步”操作有无法同步的文件，可以执行"全量同步"。
              每天的第一次同步会执行“全量同步”，后续同步会执行“增量同步”。
            </p>
            <p class="card-subtitle">
              115请按照电影和电视剧分开添加同步目录，电影的同步速度非常快，电视剧的同步速度较慢
            </p>
          </div>
          <div class="header-right">
            <el-button type="primary" @click="showAddDialog = true">
              <el-icon>
                <Plus />
              </el-icon>
              添加同步目录
            </el-button>
          </div>
        </div>
      </template>
      <div style="
          width: 100%;
          height: 100%;
          display: flex;
          flex-wrap: wrap;
          gap: 6px;
          justify-content: start;
          align-items: top;
        ">
        <el-card style="min-width: 320px" shadow="hover" v-for="(row, index) in directories" :key="row.id || index">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-tooltip class="box-item" :content="'目录ID：' + row.base_cid" placement="bottom">
                  #{{ row.id }} {{ row.remote_path }}
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
              <span class="info-value">{{ GetFullPath(row) }}</span>
            </div>

            <div class="info-item">
              <span class="info-label">添加时间:</span>
              <span class="info-value">{{ formatTime(row.created_at) }}</span>
            </div>

            <div class="info-item">
              <span class="info-label">最后同步时间:</span>
              <span class="info-value">{{ formatTime(row.last_sync_at) }}</span>
            </div>

            <div class="info-item">
              <el-tooltip class="box-item" effect="dark"
                content="开启后会根据strm设置中的cron表达式定时同步数据，如果该同步目录内的资源变动概率较小，建议关闭定时同步，然后有变动时手动同步" placement="bottom">
                <span class="info-label">
                  <el-icon>
                    <Warning />
                  </el-icon> 定时同步:
                </span>
              </el-tooltip>
              <el-switch v-model="row.enable_cron" :active-value="true" :inactive-value="false"
                @change="toggleCron(row)" active-color="#13ce66" inactive-color="#dcdfe6" />
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
                <el-text class="mx-1" type="warning">等待运行...</el-text>
              </span>
              <span class="info-value" v-if="row.is_running === 0">未执行</span>
            </div>
          </div>
          <template #footer>
            <div class="card-actions">
              <el-tooltip content="改操作会删除所有缓存数据（不会删除本地文件），然后执行同步，可以处理所有网盘文件变更" placement="top">
                <el-button type="warning" size="small" @click="handleFullStart(row, index)" :loading="row.starting"
                  :icon="WarningFilled" v-if="(row.source_type === '115' || row.source_type === 'baidupan') && row.is_running === 0">全量同步</el-button>
              </el-tooltip>

              <el-button type="success" size="small" @click="handleStart(row, index)" :loading="row.starting"
                v-if="row.is_running === 0" :icon="VideoPlay">同步</el-button>
              <el-button type="info" size="small" @click="handleStop(row, index)" :loading="row.stopping"
                :icon="VideoPause" v-if="row.is_running != 0">停止</el-button>
              <el-button type="primary" size="small" @click="handleEdit(row)" :loading="row.editing"
                :icon="Edit">编辑</el-button>
              <el-button type="danger" size="small" @click="handleDelete(row, index)" :loading="row.deleting"
                :icon="Delete">删除</el-button>
            </div>
          </template>
        </el-card>

        <el-col v-if="directories.length === 0 && !loading" :span="24" class="empty-card-col">
          <el-card shadow="never" class="empty-card">
            <div class="empty-content">
              <el-icon class="empty-icon">
                <Folder />
              </el-icon>
              <p class="empty-text">暂无同步目录</p>
            </div>
          </el-card>
        </el-col>
      </div>
    </el-card>

    <!-- 添加同步目录对话框 -->
    <el-dialog v-model="showAddDialog" title="添加同步目录" :width="checkIsMobile ? '90%' : '600px'"
      :close-on-click-modal="false">
      <el-form ref="addFormRef" :model="addForm" :rules="addFormRules" label-width="160px"
        :label-position="checkIsMobile ? 'top' : 'left'">
        <el-form-item label="同步源类型" prop="source_type">
          <el-select v-model="addForm.source_type" placeholder="请选择同步源类型" @change="handleSourceTypeChange">
            <el-option v-for="typeItem in sourceTypeOptions" :key="typeItem.value" :label="typeItem.label"
              :value="typeItem.value"></el-option>
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
          <el-select v-model="addForm.account_id" placeholder="请选择网盘账号" :loading="accountsLoading"
            :disabled="addLoading">
            <el-option v-for="account in accounts" :key="account.id" :label="account.name"
              :value="account.id"></el-option>
          </el-select>
          <div class="form-tip">选择用于同步的网盘账号</div>
        </el-form-item>
        <el-form-item label="来源路径" prop="base_cid" v-if="
          (addForm.source_type !== 'local' && addForm.account_id) ||
          addForm.source_type === 'local'
        ">
          <div class="pan-dir-input">
            <el-input v-model="addForm.base_cid" placeholder="点击选择按钮选择网盘目录" :disabled="addLoading" readonly />
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
        <el-form-item label="目标路径" prop="local_path" v-if="
          (addForm.source_type !== 'local' && addForm.account_id) ||
          addForm.source_type === 'local'
        ">
          <div class="pan-dir-input">
            <el-input v-model="addForm.local_path" placeholder="点击选择按钮选择本地目录" :disabled="addLoading" readonly />
            <el-button type="primary" @click="openDirSelector(true)" :disabled="addLoading">
              选择目录
            </el-button>
          </div>
          <div class="form-tip">选择本地目录作为STRM文件的存放位置</div>
        </el-form-item>

        <el-form-item label="STRM存放目录" v-if="
          (addForm.source_type !== 'local' && addForm.account_id) ||
          addForm.source_type === 'local'
        ">
          <el-input v-model="addForm.strm_path" placeholder="自动计算：本地目录 + 选中目录路径" :disabled="true" readonly />
          <div class="form-tip">STRM和元数据实际存放目录（自动生成）</div>
        </el-form-item>
        <!-- <el-form-item label="同步方式" v-if="addForm.source_type === 'baidupan'">
          <el-radio-group v-model="addForm.baidu_sync_method">
            <el-radio label="1">递归文件夹</el-radio>
            <el-radio label="2">递归接口</el-radio>
          </el-radio-group>
          <div class="form-tip">递归文件夹: 适合8000以上文件及文件夹的目录同步</div>
          <div class="form-tip">递归接口: 适合8000以下文件及文件夹的目录同步，每分钟只能单线程请求8次接口，每次1000个，超过就要等待1分钟。</div>
        </el-form-item> -->
        <el-form-item label="是否自定义设置" prop="custom_config">
          <el-switch v-model="addForm.custom_config" :active-value="true" :inactive-value="false"
            :disabled="addLoading" />
          <div class="form-tip">
            开启后可自定义视频扩展名和元数据扩展名配置，否则使用strm设置中的值
          </div>
        </el-form-item>
        <!-- 最小视频文件大小 -->
        <el-form-item label="最小视频文件大小 (MB)" prop="min_video_size" v-if="addForm.custom_config">
          <el-slider v-model="addForm.min_video_size" :min="-1" :max="1000" :step="1" :precision="0"
            :format-tooltip="formatTooltip" show-input />
          <div class="form-help">
            <p>小于此大小的视频文件将不会生成STRM文件，单位为MB。设置为0表示不限制文件大小</p>
          </div>
        </el-form-item>
        <el-form-item label="视频扩展名" prop="video_ext" v-if="addForm.custom_config">
          <MetadataExtInput v-model="addForm.video_ext" placeholder="输入扩展名后按回车添加，逗号或者分行分隔"
            class="meta-ext-input limited-width-input" />
          <div class="form-tip">指定需要生成STRM文件的视频文件扩展名</div>
        </el-form-item>
        <el-form-item label="元数据扩展名" prop="meta_ext" v-if="addForm.custom_config">
          <MetadataExtInput v-model="addForm.meta_ext" placeholder="输入扩展名后按回车添加，逗号或者分行分隔"
            class="meta-ext-input limited-width-input" />
          <div class="form-tip">指定需要同步的元数据文件扩展名</div>
        </el-form-item>
        <el-form-item label="排除文件名" prop="exclude_name" v-if="addForm.custom_config">
          <MetadataExtInput v-model="addForm.exclude_name" :autoAddDot="false" placeholder="输入文件名后按回车添加，逗号或者分行分隔"
            class="meta-ext-input limited-width-input" />
          <div class="form-tip">指定需要排除同步的名称，必须输入完整，可以是文件夹名字或者文件名字</div>
        </el-form-item>
        <el-form-item label="是否下载元数据" prop="download_meta" v-if="addForm.custom_config">
          <el-radio-group v-model="addForm.download_meta">
            <el-radio-button :label="-1">使用STRM设置</el-radio-button>
            <el-radio-button :label="1">是</el-radio-button>
            <el-radio-button :label="0">否</el-radio-button>
          </el-radio-group>
          <div class="form-help">
            <p>如果选择是，同步时会将本地不存在的元数据文件下载回来</p>
            <p>
              如果选择否，同步时不会下载，<strong stylle="color: black;">但是也同时跳过处理元数据，已存在的会保留，新增的不会上传</strong>
            </p>
          </div>
        </el-form-item>
        <!-- 同步完是否上传网盘不存在的元数据 -->
        <el-form-item label="网盘不存在的元数据" prop="upload_meta" v-if="addForm.custom_config">
          <el-radio-group v-model="addForm.upload_meta">
            <el-radio-button :label="-1">使用STRM设置</el-radio-button>
            <el-radio-button :label="2" :disabled="addForm.download_meta === 0">删除</el-radio-button>
            <el-radio-button :label="1" :disabled="addForm.download_meta === 0">上传</el-radio-button>
            <el-radio-button :label="0">保留</el-radio-button>
          </el-radio-group>
          <div class="form-help">
            <p>删除: 本地存在且网盘不存在则删除本地文件</p>
            <p>
              上传: 本地存在且网盘不存在，分三种情况: <br />
              &nbsp;&nbsp;&nbsp;&nbsp;1. 父目录在网盘存在则上传<br />
              &nbsp;&nbsp;&nbsp;&nbsp;2. 父目录在网盘不存在（网盘已删除）则删除本地文件<br />
              &nbsp;&nbsp;&nbsp;&nbsp;3. 父目录是特定名字，则创建父目录并上传，特定名字包括："extrafanart",
              "exfanarts",
              "extrafanarts",
              "extras",
              "specials",
              "shorts",
              "scenes",
              "featurettes",
              "behind the scenes",
              "trailers",
              "interviews",
            </p>
            <p>保留：不会删除本地文件，不管网盘有没有删除它</p>
          </div>
        </el-form-item>
        <el-form-item label="是否检查元数据修改时间" prop="check_meta_mtime" v-if="addForm.custom_config">
          <el-radio-group v-model="addForm.check_meta_mtime">
            <el-radio-button :label="-1">使用STRM设置</el-radio-button>
            <el-radio-button :label="1">是</el-radio-button>
            <el-radio-button :label="0">否</el-radio-button>
          </el-radio-group>
          <div class="form-help">
            <p>如果选择是，会有两种情况：<br />
              &nbsp;&nbsp;&nbsp;&nbsp;1. 网盘文件修改时间比本地文件新，则下载网盘文件替换本地文件<br />
              &nbsp;&nbsp;&nbsp;&nbsp;2. 网盘文件修改时间比本地文件旧，则上传本地文件到网盘
            </p>
          </div>
        </el-form-item>
        <!-- 同步完是否删除网盘不存在的空目录 -->
        <el-form-item label="网盘不存在的空目录" prop="delete_dir" v-if="addForm.custom_config">
          <el-radio-group v-model="addForm.delete_dir">
            <el-radio-button :label="-1">使用STRM设置</el-radio-button>
            <el-radio-button :label="1">删除</el-radio-button>
            <el-radio-button :label="0">不删除</el-radio-button>
          </el-radio-group>
          <div class="form-help">
            <p>同步完成后是否删除本地存在但网盘不存在的目录，该本地目录必须是空目录</p>
          </div>
        </el-form-item>
        <!-- 是否给strm链接添加路径 -->
        <el-form-item label="给strm链接添加路径" prop="add_path" v-if="addForm.custom_config">
          <el-radio-group v-model="addForm.add_path">
            <el-radio-button :label="-1">使用STRM设置</el-radio-button>
            <el-radio-button :label="1">添加</el-radio-button>
            <el-radio-button :label="2">不添加</el-radio-button>
          </el-radio-group>
          <div class="form-help">
            <p>是否给strm链接添加路径</p>
          </div>
        </el-form-item>
        <!-- 是否检查元数据 -->
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showAddDialog = false">取消</el-button>
          <el-button type="primary" @click="handleAdd" :loading="addLoading"> 确定 </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑同步目录对话框 -->
    <el-dialog v-model="showEditDialog" title="编辑同步目录" :width="checkIsMobile ? '90%' : '600px'"
      :close-on-click-modal="false">
      <el-form ref="editFormRef" :model="editForm" :rules="editFormRules" label-width="160px"
        :label-position="checkIsMobile ? 'top' : 'left'">
        <el-form-item label="来源路径" prop="base_cid">
          <div class="pan-dir-input">
            <el-input v-model="editForm.base_cid" placeholder="点击选择按钮选择115网盘目录" :disabled="editLoading" readonly />
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
            <el-input v-model="editForm.local_path" placeholder="点击选择按钮选择本地目录" :disabled="editLoading" readonly />
            <el-button type="primary" @click="openEditDirSelector(true)" :disabled="editLoading">
              选择目录
            </el-button>
          </div>
          <div class="form-tip">选择本地目录作为STRM文件的存放位置</div>
        </el-form-item>
        <el-form-item label="STRM存放目录">
          <el-text type="danger" size="large" style="font-weight: bold">{{ editForm.strm_path }}</el-text>
          <div class="form-tip">STRM和元数据实际存放目录（自动生成）</div>
        </el-form-item>
        <!-- <el-form-item label="同步方式" v-if="editForm.source_type === 'baidupan'">
          <el-radio-group v-model="editForm.baidu_sync_method">
            <el-radio label="1">递归文件夹</el-radio>
            <el-radio label="2">递归接口</el-radio>
          </el-radio-group>
          <div class="form-tip">递归文件夹: 适合8000以上文件及文件夹的目录同步</div>
          <div class="form-tip">递归接口: 适合8000以下文件及文件夹的目录同步，每分钟只能单线程请求8次接口，每次1000个，超过就要等待1分钟。</div>
        </el-form-item> -->
        <el-form-item label="是否自定义设置" prop="custom_config">
          <el-switch v-model="editForm.custom_config" :active-value="true" :inactive-value="false"
            :disabled="editLoading" />
          <div class="form-tip">
            开启后可自定义视频扩展名和元数据扩展名配置，否则使用strm设置中的值
          </div>
        </el-form-item>
        <!-- 最小视频文件大小 -->
        <el-form-item label="最小视频文件大小 (MB)" prop="min_video_size" v-if="editForm.custom_config">
          <el-slider v-model="editForm.min_video_size" :min="-1" :max="1000" :step="1" :precision="0"
            :format-tooltip="formatTooltip" show-input />
          <div class="form-help">
            <p>小于此大小的视频文件将不会生成STRM文件，单位为MB。设置为0表示不限制文件大小</p>
          </div>
        </el-form-item>
        <el-form-item label="视频扩展名" prop="video_ext" v-if="editForm.custom_config">
          <MetadataExtInput v-model="editForm.video_ext" placeholder="输入扩展名后按回车添加，逗号或者分行分隔"
            class="meta-ext-input limited-width-input" />
          <div class="form-tip">指定需要生成STRM文件的视频文件扩展名</div>
        </el-form-item>
        <el-form-item label="元数据扩展名" prop="meta_ext" v-if="editForm.custom_config">
          <MetadataExtInput v-model="editForm.meta_ext" placeholder="输入扩展名后按回车添加，逗号或者分行分隔"
            class="meta-ext-input limited-width-input" />
          <div class="form-tip">指定需要同步的元数据文件扩展名</div>
        </el-form-item>
        <el-form-item label="排除文件名" prop="exclude_name" v-if="editForm.custom_config">
          <MetadataExtInput v-model="editForm.exclude_name" :autoAddDot="false" placeholder="输入文件名后按回车添加，逗号或者分行分隔"
            class="meta-ext-input limited-width-input" />
          <div class="form-tip">指定需要排除同步的名称，必须输入完整，可以是文件夹名字或者文件名字</div>
        </el-form-item>
        <el-form-item label="是否下载元数据" prop="download_meta" v-if="editForm.custom_config">
          <el-radio-group v-model="editForm.download_meta">
            <el-radio-button :label="-1">使用STRM设置</el-radio-button>
            <el-radio-button :label="1">是</el-radio-button>
            <el-radio-button :label="0">否</el-radio-button>
          </el-radio-group>
          <div class="form-help">
            <p>如果选择是，同步时会将本地不存在的元数据文件下载回来</p>
            <p>
              如果选择否，同步时不会下载，<strong stylle="color: black;">但是也同时跳过处理元数据，已存在的会保留，新增的不会上传</strong>
            </p>
          </div>
        </el-form-item>
        <!-- 同步完是否上传网盘不存在的元数据 -->
        <el-form-item label="网盘不存在的元数据" prop="upload_meta" v-if="editForm.custom_config">
          <el-radio-group v-model="editForm.upload_meta">
            <el-radio-button :label="-1">使用STRM设置</el-radio-button>
            <el-radio-button :label="2" :disabled="editForm.download_meta === 0">删除</el-radio-button>
            <el-radio-button :label="1" :disabled="editForm.download_meta === 0">上传</el-radio-button>
            <el-radio-button :label="0">保留</el-radio-button>
          </el-radio-group>
          <div class="form-help">
            <p>删除: 本地存在且网盘不存在则删除本地文件</p>
            <p>
              上传: 本地存在且网盘不存在，分三种情况: <br />
              &nbsp;&nbsp;&nbsp;&nbsp;1. 父目录在网盘存在则上传<br />
              &nbsp;&nbsp;&nbsp;&nbsp;2. 父目录在网盘不存在（网盘已删除）则删除本地文件<br />
              &nbsp;&nbsp;&nbsp;&nbsp;3. 父目录是特定名字，则创建父目录并上传，特定名字包括："extrafanart",
              "exfanarts",
              "extrafanarts",
              "extras",
              "specials",
              "shorts",
              "scenes",
              "featurettes",
              "behind the scenes",
              "trailers",
              "interviews",
            </p>
            <p>保留：不会删除本地文件，不管网盘有没有删除它</p>
          </div>
        </el-form-item>
        <el-form-item label="是否检查元数据修改时间" prop="check_meta_mtime" v-if="editForm.custom_config">
          <el-radio-group v-model="editForm.check_meta_mtime">
            <el-radio-button :label="-1">使用STRM设置</el-radio-button>
            <el-radio-button :label="1">是</el-radio-button>
            <el-radio-button :label="0">否</el-radio-button>
          </el-radio-group>
          <div class="form-help">
            <p>如果选择是，会有两种情况：<br />
              &nbsp;&nbsp;&nbsp;&nbsp;1. 网盘文件修改时间比本地文件新，则下载网盘文件替换本地文件<br />
              &nbsp;&nbsp;&nbsp;&nbsp;2. 网盘文件修改时间比本地文件旧，则上传本地文件到网盘
            </p>
          </div>
        </el-form-item>
        <!-- 同步完是否删除网盘不存在的空目录 -->
        <el-form-item label="网盘不存在的空目录" prop="delete_dir" v-if="editForm.custom_config">
          <el-radio-group v-model="editForm.delete_dir">
            <el-radio-button :label="-1">使用STRM设置</el-radio-button>
            <el-radio-button :label="1">删除</el-radio-button>
            <el-radio-button :label="0">不删除</el-radio-button>
          </el-radio-group>
          <div class="form-help">
            <p>同步完成后是否删除本地存在但网盘不存在的目录，该本地目录必须是空目录</p>
          </div>
        </el-form-item>
        <!-- 是否给strm链接添加路径 -->
        <el-form-item label="给strm链接添加路径" prop="add_path" v-if="editForm.custom_config">
          <el-radio-group v-model="editForm.add_path">
            <el-radio-button :label="-1">使用STRM设置</el-radio-button>
            <el-radio-button :label="1">添加</el-radio-button>
            <el-radio-button :label="2">不添加</el-radio-button>
          </el-radio-group>
          <div class="form-help">
            <p>是否给strm链接添加路径</p>
          </div>
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
    <el-dialog v-model="showDirDialog" :title="isSelectingLocalPath ? '选择目标目录' : '选择来源目录'"
      :width="checkIsMobile ? '90%' : '600px'" :close-on-click-modal="false">
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
import { Plus, Loading, Folder, VideoPlay, Edit, Delete, Warning, Star, VideoPause, WarningFilled } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import { formatTime } from '@/utils/timeUtils'
import { isMobile, onDeviceTypeChange } from '@/utils/deviceUtils'
import { sourceTypeOptions, sourceTypeTagMap, sourceTypeMap } from '@/utils/sourceTypeUtils'
import MetadataExtInput from './MetadataExtInput.vue'
import 'element-plus/theme-chalk/display.css'

interface SyncDirectory {
  id: number
  base_cid: string
  local_path: string
  remote_path: string
  strm_path: string
  created_at: number
  updated_at: number
  last_sync_at: number
  deleting?: boolean
  editing?: boolean
  starting?: boolean
  source_type: string
  account_id: number
  account_name: string
  custom_config: boolean
  video_ext_arr: string[]
  meta_ext_arr: string[]
  exclude_name_arr: string[]
  min_video_size: number
  upload_meta: -1 | 0 | 1 | 2
  download_meta: -1 | 0 | 1
  delete_dir: -1 | 0 | 1
  enable_cron: boolean
  is_running: number
  stopping?: boolean
  add_path: -1 | 1 | 2
  check_meta_mtime: -1 | 0 | 1
  baidu_sync_method: 1 | 2
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

const formatTooltip = (value: number) => {
  if (value === -1) {
    return '使用STRM设置'
  }
  return `${value} MB`
}

// 添加对话框状态
const showAddDialog = ref(false)
const addLoading = ref(false)
const addFormRef = ref<FormInstance>()
const addForm = reactive({
  local_path: '',
  base_cid: '',
  strm_path: '',
  source_type: '',
  baidu_sync_method: 1,
  account_id: '',
  custom_config: false,
  video_ext: [] as string[],
  meta_ext: [] as string[],
  exclude_name: [] as string[],
  remote_path: '',
  min_video_size: -1,
  upload_meta: -1,
  download_meta: -1,
  delete_dir: -1,
  add_path: -1,
  check_meta_mtime: -1,
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
  source_type: '',
  account_id: 0,
  baidu_sync_method: 1,
  custom_config: false,
  video_ext: [] as string[],
  meta_ext: [] as string[],
  exclude_name: [] as string[],
  remote_path: '',
  min_video_size: -1,
  upload_meta: -1,
  download_meta: -1,
  delete_dir: -1,
  add_path: -1,
  check_meta_mtime: -1,
})
const editSelectedDirPath = ref('')

// 表单验证规则
const addFormRules: FormRules = {
  local_path: [
    { required: true, message: '请选择目标目录', trigger: 'blur' },
    { min: 1, max: 500, message: '长度在 1 到 500 个字符', trigger: 'blur' },
  ],
  base_cid: [
    { required: true, message: '请选择来源目录', trigger: 'blur' },
    { min: 1, max: 100, message: '长度在 1 到 100 个字符', trigger: 'blur' },
  ],
  dir_depth: [
    { required: true, message: '请输入缓存的目录深度', trigger: 'blur' },
    {
      type: 'number',
      min: 1,
      max: 3,
      message: '缓存的目录深度必须在 1 到 3 之间',
      trigger: 'blur',
    },
  ],
  source_type: [{ required: true, message: '请选择同步源类型', trigger: 'change' }],
  account_id: [{ required: true, message: '请选择网盘账号', trigger: 'change' }],
}

// 编辑表单验证规则
const editFormRules: FormRules = {
  local_path: [
    { required: true, message: '请选择目标目录', trigger: 'blur' },
    { min: 1, max: 500, message: '长度在 1 到 500 个字符', trigger: 'blur' },
  ],
  base_cid: [
    { required: true, message: '请选择来源目录', trigger: 'blur' },
    { min: 1, max: 100, message: '长度在 1 到 100 个字符', trigger: 'blur' },
  ],
  account_id: [{ required: true, message: '请选择网盘账号', trigger: 'change' }],
}

const GetFullPath = (row: SyncDirectory) => {
  // 如果cleanLocalPath以字母开头则用\分隔，如果以/开头则用/分隔
  const pathSeparator = row.local_path.startsWith('/') ? '/' : '\\'
  if (row.source_type == 'local') {
    return row.local_path
  }
  let remotePath = row.remote_path
  if (pathSeparator === '/') {
    remotePath = remotePath.replace(/\\/g, pathSeparator)
  } else {
    remotePath = remotePath.replace(/\//g, pathSeparator)
  }
  return `${row.local_path}${pathSeparator}${remotePath}`
}

// 加载同步目录列表
const loadDirectories = async () => {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/path-list`, {
      timeout: 5000,
      params: {
        page: currentPage.value,
        page_size: pageSize.value,
      },
    })

    if (response?.data.code === 200) {
      directories.value = response.data.data.list || []
      total.value = response.data.data.total || 0
    } else {
      ElMessage.error(response?.data.message || '加载同步目录失败')
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

const updatePathesStatus = async () => {
  const response = await http?.get(`${SERVER_URL}/sync/path-list`)

  if (response?.data.code === 200) {
    for (const p of response.data.data.list || []) {
      const path = directories.value.find(pa => pa.id === p.id)
      if (path) {
        path.is_running = p.is_running
        // console.log(`更新路径状态: ${path.id}, 运行状态: ${path.is_running}`)
      }
    }
  }
  autoRefreshEnabled = true
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
      source_type: addForm.source_type.trim(),
      account_id: addForm.account_id ? addForm.account_id : 0,
      custom_config: addForm.custom_config,
      video_ext_arr: addForm.video_ext,
      meta_ext_arr: addForm.meta_ext,
      exclude_name_arr: addForm.exclude_name,
      min_video_size: addForm.min_video_size,
      upload_meta: addForm.upload_meta,
      download_meta: addForm.download_meta,
      delete_dir: addForm.delete_dir,
      add_path: addForm.add_path,
      check_meta_mtime: addForm.check_meta_mtime,
      baidu_sync_method: addForm.baidu_sync_method,
    }
    console.log(formData)

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
      addForm.custom_config = false
      addForm.video_ext = []
      addForm.meta_ext = []
      addForm.exclude_name = []
      selectedDirPath.value = ''
      addForm.min_video_size = -1
      addForm.upload_meta = -1
      addForm.download_meta = -1
      addForm.delete_dir = -1
      addForm.add_path = -1
      addForm.check_meta_mtime = -1
      addForm.baidu_sync_method = 1
      loadDirectories()
    } else {
      ElMessage.error(response?.data.message || '添加同步目录失败')
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
  editForm.id = row.id
  editForm.account_id = row.account_id
  editForm.local_path = row.local_path
  editForm.base_cid = row.base_cid
  editForm.source_type = row.source_type
  editForm.account_id = row.account_id
  editForm.custom_config = row.custom_config
  editForm.video_ext = row.video_ext_arr || []
  editForm.meta_ext = row.meta_ext_arr || []
  editForm.exclude_name = row.exclude_name_arr || []
  editForm.remote_path = row.remote_path
  editSelectedDirPath.value = row.remote_path
  editForm.min_video_size = row.min_video_size
  editForm.upload_meta = row.upload_meta
  editForm.download_meta = row.download_meta
  editForm.delete_dir = row.delete_dir
  editForm.add_path = row.add_path
  editForm.check_meta_mtime = row.check_meta_mtime
  editForm.baidu_sync_method = row.baidu_sync_method

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
      account_id: editForm.account_id,
      local_path: editForm.local_path.trim(),
      base_cid: editForm.base_cid.trim(),
      strm_path: editForm.strm_path.trim(),
      custom_config: editForm.custom_config,
      video_ext_arr: editForm.video_ext,
      meta_ext_arr: editForm.meta_ext,
      exclude_name_arr: editForm.exclude_name,
      source_type: editForm.source_type.trim(),
      remote_path: editSelectedDirPath.value,
      min_video_size: editForm.min_video_size,
      upload_meta: editForm.upload_meta,
      download_meta: editForm.download_meta,
      delete_dir: editForm.delete_dir,
      add_path: editForm.add_path,
      check_meta_mtime: editForm.check_meta_mtime,
      baidu_sync_method: editForm.baidu_sync_method,
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
      editForm.custom_config = false
      editForm.video_ext = []
      editForm.meta_ext = []
      editForm.exclude_name = []
      editSelectedDirPath.value = ''
      editForm.min_video_size = -1
      editForm.upload_meta = -1
      editForm.download_meta = -1
      editForm.delete_dir = -1
      editForm.add_path = -1
      editForm.check_meta_mtime = -1
      editForm.baidu_sync_method = 1
      loadDirectories()
    } else {
      ElMessage.error(response?.data.message || '编辑同步目录失败')
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
      `不会删除已经同步的元数据和STRM文件，确定要删除同步目录 "${row.local_path}" 吗？`,
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
const handleFullStart = async (row: SyncDirectory, index: number) => {
  try {
    directories.value[index].starting = true

    const formData = {
      id: row.id || '',
    }

    const response = await http?.post(`${SERVER_URL}/sync/path/full-start`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success(`同步目录 "${row.local_path}" 启动成功`)
    } else {
      ElMessage.error(response?.data.message || '启动同步目录失败')
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
      ElMessage.error(response?.data.message || '启动同步目录失败')
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

// 处理停止同步
const handleStop = async (row: SyncDirectory, index: number) => {
  try {
    directories.value[index].starting = true

    const formData = {
      id: row.id || '',
    }

    const response = await http?.post(`${SERVER_URL}/sync/path/stop`, formData, {
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (response?.data.code === 200) {
      ElMessage.success(`同步目录 "${row.local_path}" 停止成功`)
    } else {
      ElMessage.error(response?.data.message || '停止同步目录失败')
    }
  } catch {
    console.error('停止同步目录错误')
    ElMessage.error('停止同步目录失败')
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
const openDirSelector = async (isLocalPath: boolean = false) => {
  showDirDialog.value = true
  tempSelectedDir.value = null
  currentDir.value = null
  selectedSourceType.value = isLocalPath ? 'local' : addForm.source_type
  isSelectingLocalPath.value = isLocalPath
  selectedAccountId.value = addForm.account_id

  await loadDirTree(isLocalPath ? 'local' : addForm.source_type, null)
}

// 加载目录树
const loadDirTree = async (sourceType: string, dir: DirInfo | null) => {
  try {
    dirTreeLoading.value = true
    // 加载网盘目录树
    const accountIdToUse = selectedAccountId.value
    const response = await http?.get(`${SERVER_URL}/path/list`, {
      params: {
        parent_id: dir?.id || "",
        parent_path: dir?.path || "",
        source_type: sourceType,
        account_id: accountIdToUse,
      },
      timeout: 60000, // 设置超时时间为1分钟
    })

    if (response?.data.code === 200) {
      dirTreeData.value = response.data.data || []
      return true
    } else {
      ElMessage.error(response?.data.message || '加载目录树失败')
      return false
    }
  } catch (error) {
    console.error('加载目录树错误:', error)
    return false
  } finally {
    dirTreeLoading.value = false
  }
}

// 临时选择目录（点击目录时）
const selectTempDir = async (dir: DirInfo) => {
  // 加载该目录的子目录
  if (await loadDirTree(selectedSourceType.value, dir)) {
    tempSelectedDir.value = dir
    currentDir.value = dir
    return
  }
}

// 计算STRM存放目录
const calculateStrmPath = (localPath: string, dirPath: string): string => {
  if (!localPath || !dirPath) return ''

  // 移除目录路径开头的斜杠并规范化路径分隔符
  let cleanDirPath = dirPath
  if (versionInfo.value?.isWindows) {
    cleanDirPath = dirPath.replace(/^[/\\]+/, '').replace(/\//g, '\\')
  }
  let pathSeparator = '/'
  if (versionInfo.value?.isWindows) {
    pathSeparator = '\\'
  }
  return dirPath ? `${localPath}${pathSeparator}${cleanDirPath}` : localPath
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
      editSelectedDirPath.value = selectedDir.path
      // 更新编辑表单的STRM路径
      updateEditStrmPath()
    } else {
      // 添加模式：设置添加表单的CID值和显示路径
      addForm.base_cid = selectedDir.id
      selectedDirPath.value = selectedDir.path
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
  // 构造一个DirInfo对象
  const dir = {
    id: editForm.local_path,
    path: editForm.local_path,
    name: editForm.local_path
  }
  if (!isLocalPath) {
    dir.path = editForm.remote_path
    dir.id = editForm.base_cid
  }
  await loadDirTree(
    isLocalPath ? 'local' : editForm.source_type,
    dir,
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
        if (account.source_type !== addForm.source_type) continue
        accounts.value.push(account)
      }
    } else {
      console.error('加载账号列表失败:', response?.data.message || '未知错误')
      accounts.value = []
    }
  } catch (error) {
    console.error('加载账号列表失败:', error)
    accounts.value = []
  } finally {
    accountsLoading.value = false
  }
}
interface VersionInfo {
  version: string
  date: string
  isWindows: boolean
  isRelease: boolean
}
const versionInfo = ref<VersionInfo | null>(null)
// 加载系统版本信息
const loadVersionInfo = async () => {
  try {
    const response = await http?.get(`${SERVER_URL}/version`)
    if (response && response.data) {
      versionInfo.value = response.data
    } else {
      versionInfo.value = null
    }
  } catch (error) {
    console.error('加载系统版本信息错误:', error)
    versionInfo.value = null
  }
}
// 添加自动刷新相关变量
const autoRefreshTimer = ref<number | null>(null)
let autoRefreshEnabled = true
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
    if (!autoRefreshEnabled) {
      return
    }
    autoRefreshEnabled = false
    updatePathesStatus()
  }, 2000)
}
const clearAutoRefreshTimer = () => {
  if (autoRefreshTimer.value) {
    clearInterval(autoRefreshTimer.value)
    autoRefreshTimer.value = null
  }
}
// 组件挂载时加载数据
let removeDeviceTypeListener: (() => void) | null = null

onMounted(() => {
  loadVersionInfo()
  checkMobile()
  removeDeviceTypeListener = onDeviceTypeChange((newIsMobile) => {
    checkIsMobile.value = newIsMobile
  })
  loadDirectories()
  checkAndSetAutoRefresh()
})

onUnmounted(() => {
  if (removeDeviceTypeListener) {
    removeDeviceTypeListener()
  }
  clearAutoRefreshTimer()
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
  flex-wrap: wrap;
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
  color: #000;
  margin-bottom: 16px;
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
  justify-content: end !important;
  flex-wrap: wrap;
  /* gap: 8px; */
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
</style>
