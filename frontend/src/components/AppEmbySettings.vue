<template>
  <div class="main-content-container emby-content">
    <div class="card-header">
      <h2 class="card-title">Emby 外网302 配置</h2>
    </div>

    <el-form
      :model="embyData"
      :rules="formRules"
      :label-position="isMobile ? 'top' : 'left'"
      :label-width="180"
      class="emby-form"
      ref="formRef"
    >
      <!-- Emby服务器地址 -->
      <el-form-item label="Emby服务器地址" prop="emby_url">
        <el-input
          v-model="embyData.emby_url"
          placeholder="请输入Emby服务器地址，格式：http://ip:port"
          :disabled="embyLoading"
          class="limited-width-input"
          @input="updateEmbyExample"
        />
        <div v-if="embyExample" class="emby-example-inline">
          <span class="example-label">示例格式：</span>
          <code class="example-url">{{ embyExample }}</code>
        </div>
        <div class="form-help">
          <p>请输入Emby服务器的完整地址，包含协议、IP地址和端口号</p>
          <p>例如：http://192.168.1.100:8096 或 http://emby-server.local:8096</p>
        </div>
      </el-form-item>
      <el-form-item>
        <!-- 保存和重置按钮 -->
        <div class="emby-actions">
          <el-button
            type="success"
            @click="saveEmbyConfig"
            :loading="embyLoading"
            size="large"
            :icon="Check"
          >
            保存设置
          </el-button>
        </div>
      </el-form-item>

      <el-form-item>
        <div>如果要启用外网302，需在 STRM设置 中将 启用本地代理直链 设置为 关闭</div>
      </el-form-item>
    </el-form>

    <!-- 设置状态显示 -->
    <el-alert
      v-if="embyStatus"
      :title="embyStatus.title"
      :type="embyStatus.type"
      :description="embyStatus.description"
      :closable="false"
      show-icon
      class="emby-status"
    />
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { Check } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { inject, onMounted, ref, reactive } from 'vue'
import { isMobile as checkIsMobile } from '@/utils/deviceUtils'

// HTTP客户端
const http: AxiosStatic | undefined = inject('$http')

// 表单引用
const formRef = ref<FormInstance>()

// 移动端检测
const isMobile = ref(checkIsMobile())

// 加载状态
const embyLoading = ref(false)

// Emby设置数据
const embyData = reactive({
  emby_url: '',
})

// 示例显示
const embyExample = ref('http://192.168.1.100:8096')

// 状态提示
const embyStatus = ref<{
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
} | null>(null)

// 表单验证规则
const formRules: FormRules = {
  emby_url: [
    {
      required: true,
      message: '请输入Emby服务器地址',
      trigger: 'blur',
    },
    {
      pattern: /^(http|https):\/\/[^\s/$.?#].[^\s]*$/, // 简单的URL验证正则
      message: '请输入有效的URL格式，如：http://ip:port',
      trigger: 'blur',
    },
  ],
}

// 默认配置
const defaultConfig = {
  emby_url: '',
}

// 加载Emby配置
const loadEmbyConfig = async () => {
  try {
    embyLoading.value = true
    const response = await http?.get(`${SERVER_URL}/setting/emby`)

    if (response?.data.code === 200) {
      // 填充数据到表单
      embyData.emby_url = response.data.data?.emby_url || ''
    } else {
      // 使用默认配置
      Object.assign(embyData, defaultConfig)
      ElMessage.warning('加载Emby配置失败，使用默认配置')
    }
  } catch (error) {
    console.error('加载Emby配置错误:', error)
    Object.assign(embyData, defaultConfig)
    ElMessage.error('加载Emby配置失败')
  } finally {
    embyLoading.value = false
  }
}

// 保存Emby配置
const saveEmbyConfig = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    embyLoading.value = true

    const response = await http?.post(
      `${SERVER_URL}/setting/emby`,
      {
        emby_url: embyData.emby_url.trim(),
      },
      {
        headers: {
          'Content-Type': 'application/json',
        },
      },
    )

    if (response?.data.code === 200) {
      embyStatus.value = {
        title: '保存成功',
        type: 'success',
        description: 'Emby服务器设置已成功保存',
      }
      ElMessage.success('Emby服务器设置已成功保存')
    } else {
      embyStatus.value = {
        title: '保存失败',
        type: 'error',
        description: response?.data.msg || '保存Emby服务器设置失败',
      }
      ElMessage.error(response?.data.msg || '保存失败')
    }

    // 3秒后隐藏状态提示
    setTimeout(() => {
      embyStatus.value = null
    }, 3000)
  } catch (error) {
    console.error('保存Emby配置错误:', error)
    embyStatus.value = {
      title: '保存失败',
      type: 'error',
      description: '保存Emby服务器设置时出现错误',
    }
    ElMessage.error('保存失败')
  } finally {
    embyLoading.value = false
  }
}

// 更新示例
const updateEmbyExample = () => {
  // 根据用户输入动态更新示例（如果需要）
}

onMounted(() => {
  loadEmbyConfig()
})
</script>

<style scoped>
.main-content-container {
  width: 100%;
  max-width: 100%;
  /* margin: 0 auto; */
  padding: 20px;
}

.emby-content {
  max-width: 800px;
}

.settings-card {
  width: 100%;
  border-radius: 8px;
}

.card-header {
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
  margin-bottom: 20px;
}

.card-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.card-subtitle {
  margin: 4px 0 0 0;
  font-size: 14px;
  color: #909399;
}

.emby-form {
  width: 100%;
}

.limited-width-input {
  max-width: 500px;
  width: 100%;
}

.form-help {
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
}

.form-help p {
  margin: 4px 0;
}

.emby-example-inline {
  margin-top: 8px;
  padding: 8px 12px;
  background-color: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
}

.example-label {
  color: #606266;
  margin-right: 8px;
}

.example-url {
  color: #409eff;
  background-color: #ecf5ff;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
}

.emby-actions {
  margin-top: 30px;
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.emby-status {
  margin-top: 20px;
}
</style>
