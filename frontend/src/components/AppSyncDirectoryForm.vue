<template>
  <div class="sync-directory-form-page" :class="{ 'is-mobile': checkIsMobile }">
    <el-button type="primary" @click="goBack" size="large" link>
      <el-icon>
        <ArrowLeft />
      </el-icon>
      返回STRM同步目录
    </el-button>

    <template v-if="checkIsMobile">
      <div class="mobile-form-header">
        <h3>{{ isEditMode ? '编辑同步目录' : '添加同步目录' }}</h3>
      </div>

      <el-form ref="formRef" :model="form" :rules="formRules" label-width="160px"
        label-position="top">
        <el-form-item label="同步源类型" prop="source_type" v-if="!isEditMode">
          <el-select v-model="form.source_type" placeholder="请选择同步源类型" @change="handleSourceTypeChange">
            <el-option v-for="typeItem in sourceTypeOptions" :key="typeItem.value" :label="typeItem.label"
              :value="typeItem.value"></el-option>
          </el-select>
          <div class="form-tip">
            <div v-if="form.source_type === 'local'">
              本地目录可以通过CD2间接支持更多网盘，请将CD2的本地挂载目录映射到容器中（如果使用docker）,然后选择该目录
            </div>
            <div v-if="form.source_type === '115'">需要先添加用于同步的115账号并授权</div>
            <div v-if="form.source_type === '123'">需要先添加用于同步的123账号并授权</div>
          </div>
        </el-form-item>
        <el-form-item label="网盘账号" prop="account_id" v-if="form.source_type !== 'local' && !isEditMode">
          <el-select v-model="form.account_id" placeholder="请选择网盘账号" :loading="accountsLoading"
            :disabled="loading">
            <el-option v-for="account in accounts" :key="account.id" :label="account.name"
              :value="account.id"></el-option>
          </el-select>
          <div class="form-tip">选择用于同步的网盘账号</div>
        </el-form-item>
        <el-form-item label="来源路径" prop="base_cid" v-if="
          (form.source_type !== 'local' && form.account_id) ||
          form.source_type === 'local' ||
          isEditMode
        ">
          <div class="pan-dir-input">
            <el-input v-model="form.base_cid" placeholder="点击选择按钮选择网盘目录" :disabled="loading" readonly />
            <el-button type="primary" @click="openDirSelector(false)" :disabled="loading">
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
          (form.source_type !== 'local' && form.account_id) ||
          form.source_type === 'local' ||
          isEditMode
        ">
          <div class="pan-dir-input">
            <el-input v-model="form.local_path" placeholder="点击选择按钮选择本地目录" :disabled="loading" readonly />
            <el-button type="primary" @click="openDirSelector(true)" :disabled="loading">
              选择目录
            </el-button>
          </div>
          <div class="form-tip">选择本地目录作为STRM文件的存放位置</div>
        </el-form-item>

        <el-form-item label="STRM存放目录" v-if="
          (form.source_type !== 'local' && form.account_id) ||
          form.source_type === 'local' ||
          isEditMode
        ">
          <el-input v-model="form.strm_path" placeholder="自动计算：本地目录 + 选中目录路径" :disabled="true" readonly />
          <div class="form-tip">STRM和元数据实际存放目录（自动生成）</div>
        </el-form-item>

        <el-form-item label="是否自定义设置" prop="custom_config">
          <el-switch v-model="form.custom_config" :active-value="true" :inactive-value="false"
            :disabled="loading" />
          <div class="form-tip">
            开启后可自定义视频扩展名和元数据扩展名配置，否则使用strm设置中的值
          </div>
        </el-form-item>

        <template v-if="form.custom_config">
          <el-form-item label="定时同步表达式" prop="cron">
            <el-input v-model="form.cron" placeholder="留空则使用STRM设置中的表达式" :disabled="loading"
              @blur="loadCronTimes" />
            <div class="form-help">
              <p><strong>常用示例：</strong></p>
              <ul class="cron-examples">
                <li><code>0 0 * * *</code> - 每天0点执行</li>
                <li><code>0 */6 * * *</code> - 每6小时执行一次</li>
                <li><code>0 2 * * *</code> - 每天凌晨2点执行</li>
              </ul>
              <div v-if="cronTimes.length > 0" class="cron-next-times">
                <p><strong>下5次执行时间：</strong></p>
                <div v-loading="cronTimesLoading" class="cron-times-list">
                  <div v-for="(time, index) in cronTimes" :key="index" class="cron-time-item">
                    <el-tag type="info" size="small">{{ time }}</el-tag>
                  </div>
                </div>
              </div>
            </div>
          </el-form-item>
          <el-form-item label="STRM直连地址" prop="strm_base_url" v-if="form.source_type !== 'local' && form.source_type !== 'openlist'">
            <el-input v-model="form.strm_base_url" placeholder="留空则使用STRM设置中的地址"
              :disabled="loading" @input="updateStrmExample" />
            <div v-if="strmExample" class="strm-example-inline">
              <span class="example-label">示例：</span>
              <code class="example-url">{{ strmExample }}</code>
            </div>
            <div class="form-tip">STRM文件将使用此地址作为基础URL，留空则使用STRM设置中的值</div>
          </el-form-item>
          <el-form-item label="最小视频文件大小 (MB)" prop="min_video_size">
            <el-slider v-model="form.min_video_size" :min="-1" :max="1000" :step="1" :precision="0"
              :format-tooltip="formatTooltip" show-input />
            <div class="form-help">
              <p>小于此大小的视频文件将不会生成STRM文件，单位为MB。设置为0表示不限制文件大小</p>
            </div>
          </el-form-item>
          <el-form-item label="视频扩展名" prop="video_ext">
            <MetadataExtInput v-model="form.video_ext" placeholder="输入扩展名后按回车添加，逗号或者分行分隔"
              class="meta-ext-input limited-width-input" />
            <div class="form-tip">指定需要生成STRM文件的视频文件扩展名</div>
          </el-form-item>
          <el-form-item label="元数据扩展名" prop="meta_ext">
            <MetadataExtInput v-model="form.meta_ext" placeholder="输入扩展名后按回车添加，逗号或者分行分隔"
              class="meta-ext-input limited-width-input" />
            <div class="form-tip">指定需要同步的元数据文件扩展名</div>
          </el-form-item>
          <el-form-item label="排除文件名" prop="exclude_name">
            <MetadataExtInput v-model="form.exclude_name" :autoAddDot="false" placeholder="输入文件名后按回车添加，逗号或者分行分隔"
              class="meta-ext-input limited-width-input" />
            <div class="form-tip">指定需要排除同步的名称，必须输入完整，可以是文件夹名字或者文件名字</div>
          </el-form-item>
          <el-form-item label="是否下载元数据" prop="download_meta">
            <el-radio-group v-model="form.download_meta">
              <el-radio-button :label="-1">使用STRM设置</el-radio-button>
              <el-radio-button :label="1">是</el-radio-button>
              <el-radio-button :label="0">否</el-radio-button>
            </el-radio-group>
            <div class="form-help">
              <p>如果选择是，同步时会将本地不存在的元数据文件下载回来</p>
              <p>
                如果选择否，同步时不会下载，<strong style="color: black;">但是也同时跳过处理元数据，已存在的会保留，新增的不会上传</strong>
              </p>
            </div>
          </el-form-item>
          <el-form-item label="网盘不存在的元数据" prop="upload_meta">
            <el-radio-group v-model="form.upload_meta">
              <el-radio-button :label="-1">使用STRM设置</el-radio-button>
              <el-radio-button :label="2" :disabled="form.download_meta === 0">删除</el-radio-button>
              <el-radio-button :label="1" :disabled="form.download_meta === 0">上传</el-radio-button>
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
          <el-form-item label="是否检查元数据修改时间" prop="check_meta_mtime">
            <el-radio-group v-model="form.check_meta_mtime">
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
          <el-form-item label="网盘不存在的空目录" prop="delete_dir">
            <el-radio-group v-model="form.delete_dir">
              <el-radio-button :label="-1">使用STRM设置</el-radio-button>
              <el-radio-button :label="1">删除</el-radio-button>
              <el-radio-button :label="0">不删除</el-radio-button>
            </el-radio-group>
            <div class="form-help">
              <p>同步完成后是否删除本地存在但网盘不存在的目录，该本地目录必须是空目录</p>
            </div>
          </el-form-item>
          <el-form-item label="给strm链接添加路径" prop="add_path">
            <el-radio-group v-model="form.add_path">
              <el-radio-button :label="-1">使用STRM设置</el-radio-button>
              <el-radio-button :label="1">添加</el-radio-button>
              <el-radio-button :label="2">不添加</el-radio-button>
            </el-radio-group>
            <div class="form-help">
              <p>是否给strm链接添加路径</p>
            </div>
          </el-form-item>
        </template>
      </el-form>

      <div class="mobile-form-footer">
        <el-button @click="goBack">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="loading">
          {{ isEditMode ? '保存修改' : '确定添加' }}
        </el-button>
      </div>
    </template>

    <el-card v-else class="form-card">
      <template #header>
        <div class="card-header">
          <h3>{{ isEditMode ? '编辑同步目录' : '添加同步目录' }}</h3>
        </div>
      </template>

      <el-form ref="formRef" :model="form" :rules="formRules" label-width="160px"
        :label-position="checkIsMobile ? 'top' : 'left'">
        <el-form-item label="同步源类型" prop="source_type" v-if="!isEditMode">
          <el-select v-model="form.source_type" placeholder="请选择同步源类型" @change="handleSourceTypeChange">
            <el-option v-for="typeItem in sourceTypeOptions" :key="typeItem.value" :label="typeItem.label"
              :value="typeItem.value"></el-option>
          </el-select>
          <div class="form-tip">
            <div v-if="form.source_type === 'local'">
              本地目录可以通过CD2间接支持更多网盘，请将CD2的本地挂载目录映射到容器中（如果使用docker）,然后选择该目录
            </div>
            <div v-if="form.source_type === '115'">需要先添加用于同步的115账号并授权</div>
            <div v-if="form.source_type === '123'">需要先添加用于同步的123账号并授权</div>
          </div>
        </el-form-item>
        <el-form-item label="网盘账号" prop="account_id" v-if="form.source_type !== 'local' && !isEditMode">
          <el-select v-model="form.account_id" placeholder="请选择网盘账号" :loading="accountsLoading"
            :disabled="loading">
            <el-option v-for="account in accounts" :key="account.id" :label="account.name"
              :value="account.id"></el-option>
          </el-select>
          <div class="form-tip">选择用于同步的网盘账号</div>
        </el-form-item>
        <el-form-item label="来源路径" prop="base_cid" v-if="
          (form.source_type !== 'local' && form.account_id) ||
          form.source_type === 'local' ||
          isEditMode
        ">
          <div class="pan-dir-input">
            <el-input v-model="form.base_cid" placeholder="点击选择按钮选择网盘目录" :disabled="loading" readonly />
            <el-button type="primary" @click="openDirSelector(false)" :disabled="loading">
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
          (form.source_type !== 'local' && form.account_id) ||
          form.source_type === 'local' ||
          isEditMode
        ">
          <div class="pan-dir-input">
            <el-input v-model="form.local_path" placeholder="点击选择按钮选择本地目录" :disabled="loading" readonly />
            <el-button type="primary" @click="openDirSelector(true)" :disabled="loading">
              选择目录
            </el-button>
          </div>
          <div class="form-tip">选择本地目录作为STRM文件的存放位置</div>
        </el-form-item>

        <el-form-item label="STRM存放目录" v-if="
          (form.source_type !== 'local' && form.account_id) ||
          form.source_type === 'local' ||
          isEditMode
        ">
          <el-input v-model="form.strm_path" placeholder="自动计算：本地目录 + 选中目录路径" :disabled="true" readonly />
          <div class="form-tip">STRM和元数据实际存放目录（自动生成）</div>
        </el-form-item>

        <el-form-item label="是否自定义设置" prop="custom_config">
          <el-switch v-model="form.custom_config" :active-value="true" :inactive-value="false"
            :disabled="loading" />
          <div class="form-tip">
            开启后可自定义视频扩展名和元数据扩展名配置，否则使用strm设置中的值
          </div>
        </el-form-item>

        <template v-if="form.custom_config">
          <el-form-item label="定时同步表达式" prop="cron">
            <el-input v-model="form.cron" placeholder="留空则使用STRM设置中的表达式" :disabled="loading"
              @blur="loadCronTimes" />
            <div class="form-help">
              <p><strong>常用示例：</strong></p>
              <ul class="cron-examples">
                <li><code>0 0 * * *</code> - 每天0点执行</li>
                <li><code>0 */6 * * *</code> - 每6小时执行一次</li>
                <li><code>0 2 * * *</code> - 每天凌晨2点执行</li>
              </ul>
              <div v-if="cronTimes.length > 0" class="cron-next-times">
                <p><strong>下5次执行时间：</strong></p>
                <div v-loading="cronTimesLoading" class="cron-times-list">
                  <div v-for="(time, index) in cronTimes" :key="index" class="cron-time-item">
                    <el-tag type="info" size="small">{{ time }}</el-tag>
                  </div>
                </div>
              </div>
            </div>
          </el-form-item>
          <el-form-item label="STRM直连地址" prop="strm_base_url" v-if="form.source_type !== 'local' && form.source_type !== 'openlist'">
            <el-input v-model="form.strm_base_url" placeholder="留空则使用STRM设置中的地址"
              :disabled="loading" @input="updateStrmExample" />
            <div v-if="strmExample" class="strm-example-inline">
              <span class="example-label">示例：</span>
              <code class="example-url">{{ strmExample }}</code>
            </div>
            <div class="form-tip">STRM文件将使用此地址作为基础URL，留空则使用STRM设置中的值</div>
          </el-form-item>
          <el-form-item label="最小视频文件大小 (MB)" prop="min_video_size">
            <el-slider v-model="form.min_video_size" :min="-1" :max="1000" :step="1" :precision="0"
              :format-tooltip="formatTooltip" show-input />
            <div class="form-help">
              <p>小于此大小的视频文件将不会生成STRM文件，单位为MB。设置为0表示不限制文件大小</p>
            </div>
          </el-form-item>
          <el-form-item label="视频扩展名" prop="video_ext">
            <MetadataExtInput v-model="form.video_ext" placeholder="输入扩展名后按回车添加，逗号或者分行分隔"
              class="meta-ext-input limited-width-input" />
            <div class="form-tip">指定需要生成STRM文件的视频文件扩展名</div>
          </el-form-item>
          <el-form-item label="元数据扩展名" prop="meta_ext">
            <MetadataExtInput v-model="form.meta_ext" placeholder="输入扩展名后按回车添加，逗号或者分行分隔"
              class="meta-ext-input limited-width-input" />
            <div class="form-tip">指定需要同步的元数据文件扩展名</div>
          </el-form-item>
          <el-form-item label="排除文件名" prop="exclude_name">
            <MetadataExtInput v-model="form.exclude_name" :autoAddDot="false" placeholder="输入文件名后按回车添加，逗号或者分行分隔"
              class="meta-ext-input limited-width-input" />
            <div class="form-tip">指定需要排除同步的名称，必须输入完整，可以是文件夹名字或者文件名字</div>
          </el-form-item>
          <el-form-item label="是否下载元数据" prop="download_meta">
            <el-radio-group v-model="form.download_meta">
              <el-radio-button :label="-1">使用STRM设置</el-radio-button>
              <el-radio-button :label="1">是</el-radio-button>
              <el-radio-button :label="0">否</el-radio-button>
            </el-radio-group>
            <div class="form-help">
              <p>如果选择是，同步时会将本地不存在的元数据文件下载回来</p>
              <p>
                如果选择否，同步时不会下载，<strong style="color: black;">但是也同时跳过处理元数据，已存在的会保留，新增的不会上传</strong>
              </p>
            </div>
          </el-form-item>
          <el-form-item label="网盘不存在的元数据" prop="upload_meta">
            <el-radio-group v-model="form.upload_meta">
              <el-radio-button :label="-1">使用STRM设置</el-radio-button>
              <el-radio-button :label="2" :disabled="form.download_meta === 0">删除</el-radio-button>
              <el-radio-button :label="1" :disabled="form.download_meta === 0">上传</el-radio-button>
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
          <el-form-item label="是否检查元数据修改时间" prop="check_meta_mtime">
            <el-radio-group v-model="form.check_meta_mtime">
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
          <el-form-item label="网盘不存在的空目录" prop="delete_dir">
            <el-radio-group v-model="form.delete_dir">
              <el-radio-button :label="-1">使用STRM设置</el-radio-button>
              <el-radio-button :label="1">删除</el-radio-button>
              <el-radio-button :label="0">不删除</el-radio-button>
            </el-radio-group>
            <div class="form-help">
              <p>同步完成后是否删除本地存在但网盘不存在的目录，该本地目录必须是空目录</p>
            </div>
          </el-form-item>
          <el-form-item label="给strm链接添加路径" prop="add_path">
            <el-radio-group v-model="form.add_path">
              <el-radio-button :label="-1">使用STRM设置</el-radio-button>
              <el-radio-button :label="1">添加</el-radio-button>
              <el-radio-button :label="2">不添加</el-radio-button>
            </el-radio-group>
            <div class="form-help">
              <p>是否给strm链接添加路径</p>
            </div>
          </el-form-item>
        </template>
      </el-form>

      <template #footer>
        <div class="form-footer">
          <el-button @click="goBack">取消</el-button>
          <el-button type="primary" @click="handleSubmit" :loading="loading">
            {{ isEditMode ? '保存修改' : '确定添加' }}
          </el-button>
        </div>
      </template>
    </el-card>

    <el-dialog v-model="showDirDialog" title="选择文件夹"
      :width="checkIsMobile ? '90%' : '600px'" :close-on-click-modal="false" body-class="directory-selector">
      <div class="dir-selector">
        <DirectorySelector
          v-model="tempSelectedDir"
          :root-path="initialRootPath"
          :source-type="selectedSourceType"
          :account-id="Number(selectedAccountId)"
          @cancel="showDirDialog = false"
          @select="confirmSelectDir"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { inject, onMounted, onUnmounted, ref, reactive, watch, type Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { isMobile, onDeviceTypeChange } from '@/utils/deviceUtils'
import { sourceTypeOptions } from '@/utils/sourceTypeUtils'
import MetadataExtInput from './MetadataExtInput.vue'
import DirectorySelector from './DirectorySelector.vue'
import type { DirInfo } from '@/typing'

interface CloudAccount {
  id: number
  name: string
  source_type: string
  user_id: string
  username: string
  created_at: number
  token: string
}

interface VersionInfo {
  version: string
  date: string
  isWindows: boolean
  isRelease: boolean
}

const http: AxiosStatic | undefined = inject('$http')
const route = useRoute()
const router = useRouter()

const checkIsMobile = ref(isMobile())
const isEditMode = ref(false)
const loading = ref(false)

const formRef = ref<FormInstance>()
const form = reactive({
  id: 0,
  local_path: '',
  base_cid: '',
  strm_path: '',
  source_type: '',
  baidu_sync_method: 1 as 1 | 2,
  account_id: '' as string | number,
  custom_config: false,
  video_ext: [] as string[],
  meta_ext: [] as string[],
  exclude_name: [] as string[],
  remote_path: '',
  min_video_size: -1,
  upload_meta: -1 as -1 | 0 | 1 | 2,
  download_meta: -1 as -1 | 0 | 1,
  delete_dir: -1 as -1 | 0 | 1,
  add_path: -1 as -1 | 1 | 2,
  check_meta_mtime: -1 as -1 | 0 | 1,
  cron: '',
  strm_base_url: '',
})

const formRules: FormRules = {
  local_path: [
    { required: true, message: '请选择目标目录', trigger: 'blur' },
    { min: 1, max: 500, message: '长度在 1 到 500 个字符', trigger: 'blur' },
  ],
  base_cid: [
    { required: true, message: '请选择来源目录', trigger: 'blur' },
    { min: 1, max: 100, message: '长度在 1 到 100 个字符', trigger: 'blur' },
  ],
  source_type: [{ required: true, message: '请选择同步源类型', trigger: 'change' }],
  account_id: [{ required: true, message: '请选择网盘账号', trigger: 'change' }],
}

const accounts = ref<CloudAccount[]>([])
const accountsLoading = ref(false)

const showDirDialog = ref(false)
const selectedDirPath = ref('')
const tempSelectedDir = ref<DirInfo | null>(null)
const initialRootPath = ref('')
const isSelectingLocalPath = ref(false)
const selectedSourceType = ref('')
const selectedAccountId: Ref<number | string> = ref(0)

const versionInfo = ref<VersionInfo | null>(null)

const cronTimes = ref<string[]>([])
const cronTimesLoading = ref(false)
const strmExample = ref('')

const formatTooltip = (value: number) => {
  if (value === -1) {
    return '使用STRM设置'
  }
  return `${value} MB`
}

const loadCronTimes = async () => {
  if (!form.cron) {
    cronTimes.value = []
    return
  }

  try {
    cronTimesLoading.value = true
    const response = await http?.get(`${SERVER_URL}/setting/cron`, {
      params: { cron: form.cron },
    })

    if (response?.data.code === 200) {
      cronTimes.value = response.data.data || []
    } else {
      cronTimes.value = []
    }
  } catch {
    cronTimes.value = []
  } finally {
    cronTimesLoading.value = false
  }
}

const updateStrmExample = () => {
  if (form.strm_base_url) {
    const baseUrl = form.strm_base_url.replace(/\/$/, '')
    strmExample.value = `${baseUrl}/path/to/video.strm`
  } else {
    strmExample.value = ''
  }
}

const goBack = () => {
  router.push({ name: 'sync-directories' })
}

const handleSourceTypeChange = () => {
  if (form.source_type !== 'local') {
    loadAccounts()
  }
}

const loadAccounts = async () => {
  accounts.value = []
  try {
    accountsLoading.value = true
    const response = await http?.get(`${SERVER_URL}/account/list`)
    if (response?.data.code === 200) {
      const data = response.data.data || []
      for (const account of data) {
        if (account.source_type !== form.source_type) continue
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

const loadDirectoryData = async (id: number) => {
  try {
    loading.value = true
    const response = await http?.get(`${SERVER_URL}/sync/path-list`, {
      params: { page: 1, page_size: 9999 }
    })

    if (response?.data.code === 200) {
      const directory = response.data.data.list?.find((d: { id: number }) => d.id === id)
      if (directory) {
        form.id = directory.id
        form.account_id = directory.account_id
        form.local_path = directory.local_path
        form.base_cid = directory.base_cid
        form.source_type = directory.source_type
        form.custom_config = directory.custom_config
        form.video_ext = directory.video_ext_arr || []
        form.meta_ext = directory.meta_ext_arr || []
        form.exclude_name = directory.exclude_name_arr || []
        form.remote_path = directory.remote_path
        selectedDirPath.value = directory.remote_path
        form.min_video_size = directory.min_video_size
        form.upload_meta = directory.upload_meta
        form.download_meta = directory.download_meta
        form.delete_dir = directory.delete_dir
        form.add_path = directory.add_path
        form.check_meta_mtime = directory.check_meta_mtime
        form.baidu_sync_method = directory.baidu_sync_method
        form.cron = directory.cron || ''
        form.strm_base_url = directory.strm_base_url || ''
        updateStrmPath()
        if (form.strm_base_url) {
          updateStrmExample()
        }
      } else {
        ElMessage.error('未找到该同步目录')
        goBack()
      }
    } else {
      ElMessage.error(response?.data.message || '加载同步目录失败')
      goBack()
    }
  } catch {
    console.error('加载同步目录错误')
    ElMessage.error('加载同步目录失败')
    goBack()
  } finally {
    loading.value = false
  }
}

const calculateStrmPath = (localPath: string, dirPath: string): string => {
  if (!localPath || !dirPath) return ''

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

const updateStrmPath = () => {
  if (form.source_type !== 'local') {
    form.strm_path = calculateStrmPath(form.local_path, selectedDirPath.value)
  } else {
    form.strm_path = form.local_path
  }
}

const openDirSelector = async (isLocalPath: boolean = false) => {
  showDirDialog.value = true
  tempSelectedDir.value = null
  selectedSourceType.value = isLocalPath ? 'local' : form.source_type
  isSelectingLocalPath.value = isLocalPath
  selectedAccountId.value = form.account_id
  initialRootPath.value = ''
}

const confirmSelectDir = async () => {
  if (!tempSelectedDir.value) return

  const selectedDir = tempSelectedDir.value

  if (isSelectingLocalPath.value) {
    form.local_path = selectedDir.path ? selectedDir.path : selectedDir.name
  } else {
    form.base_cid = selectedDir.id
    selectedDirPath.value = selectedDir.path
    updateStrmPath()
  }

  showDirDialog.value = false
  tempSelectedDir.value = null
  isSelectingLocalPath.value = false
}

const handleSubmit = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    loading.value = true

    if (isEditMode.value) {
      const formData = {
        id: form.id,
        account_id: form.account_id,
        local_path: form.local_path.trim(),
        base_cid: form.base_cid.trim(),
        strm_path: form.strm_path.trim(),
        custom_config: form.custom_config,
        video_ext_arr: form.video_ext,
        meta_ext_arr: form.meta_ext,
        exclude_name_arr: form.exclude_name,
        source_type: form.source_type.trim(),
        remote_path: selectedDirPath.value,
        min_video_size: form.min_video_size,
        upload_meta: form.upload_meta,
        download_meta: form.download_meta,
        delete_dir: form.delete_dir,
        add_path: form.add_path,
        check_meta_mtime: form.check_meta_mtime,
        baidu_sync_method: form.baidu_sync_method,
        cron: form.cron.trim(),
        strm_base_url: form.strm_base_url.trim(),
      }

      const response = await http?.post(`${SERVER_URL}/sync/path-update`, formData, {
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (response?.data.code === 200) {
        ElMessage.success('编辑同步目录成功')
        goBack()
      } else {
        ElMessage.error(response?.data.message || '编辑同步目录失败')
      }
    } else {
      const formData = {
        local_path: form.local_path.trim(),
        base_cid: form.base_cid.trim(),
        remote_path: selectedDirPath.value,
        source_type: form.source_type.trim(),
        account_id: form.account_id ? form.account_id : 0,
        custom_config: form.custom_config,
        video_ext_arr: form.video_ext,
        meta_ext_arr: form.meta_ext,
        exclude_name_arr: form.exclude_name,
        min_video_size: form.min_video_size,
        upload_meta: form.upload_meta,
        download_meta: form.download_meta,
        delete_dir: form.delete_dir,
        add_path: form.add_path,
        check_meta_mtime: form.check_meta_mtime,
        baidu_sync_method: form.baidu_sync_method,
        cron: form.cron.trim(),
        strm_base_url: form.strm_base_url.trim(),
      }

      const response = await http?.post(`${SERVER_URL}/sync/path-add`, formData, {
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (response?.data.code === 200) {
        ElMessage.success('添加同步目录成功')
        goBack()
      } else {
        ElMessage.error(response?.data.message || '添加同步目录失败')
      }
    }
  } catch {
    console.error('提交同步目录错误')
    ElMessage.error(isEditMode.value ? '编辑同步目录失败' : '添加同步目录失败')
  } finally {
    loading.value = false
  }
}

watch(
  () => form.local_path,
  () => {
    updateStrmPath()
  },
)

watch(
  () => form.cron,
  (newVal, oldVal) => {
    if (newVal !== oldVal) {
      cronTimes.value = []
    }
  },
)

let removeDeviceTypeListener: (() => void) | null = null

onMounted(async () => {
  removeDeviceTypeListener = onDeviceTypeChange((newIsMobile) => {
    checkIsMobile.value = newIsMobile
  })

  await loadVersionInfo()

  const id = route.params.id as string
  if (id) {
    isEditMode.value = true
    await loadDirectoryData(Number(id))
  }
})

onUnmounted(() => {
  if (removeDeviceTypeListener) {
    removeDeviceTypeListener()
  }
})
</script>

<style scoped>
.sync-directory-form-page {
  padding: 20px;
}

.form-card {
  width: 100%;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  margin-top: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.form-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.form-help {
  font-size: 12px;
  color: #606266;
  margin-top: 8px;
  line-height: 1.6;
}

.form-help p {
  margin: 4px 0;
}

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

.dir-selector {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 500px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.cron-examples {
  margin: 8px 0;
  padding-left: 20px;
}

.cron-examples li {
  margin: 4px 0;
  font-size: 12px;
  color: #606266;
}

.cron-examples code {
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  color: #409eff;
}

.cron-next-times {
  margin-top: 12px;
  padding-top: 8px;
  border-top: 1px dashed #e4e7ed;
}

.cron-times-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.cron-time-item {
  display: inline-flex;
}

.strm-example-inline {
  margin-top: 8px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
}

.example-label {
  color: #909399;
  font-weight: 500;
}

.example-url {
  color: #409eff;
  background: #fff;
  padding: 2px 6px;
  border-radius: 2px;
  border: 1px solid #dcdfe6;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  word-break: break-all;
}

.sync-directory-form-page.is-mobile {
  padding: 12px;
}

.mobile-form-header {
  margin: 12px 0;
  padding-bottom: 12px;
  border-bottom: 1px solid #ebeef5;
}

.mobile-form-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

@media (max-width: 768px) {
  .mobile-form-header {
    display: none;
  }
}

.mobile-form-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #ebeef5;
}

.is-mobile .pan-dir-input {
  flex-direction: column;
}

.is-mobile .pan-dir-input .el-button {
  width: 100%;
}
</style>
