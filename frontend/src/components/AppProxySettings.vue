<template>
  <div class="proxy-settings-container">
    <!-- 网络代理设置卡片 -->
    <el-card class="proxy-settings-card" shadow="hover">
      <template #header>
        <h2 class="card-title">网络代理设置</h2>
        <p class="card-subtitle">配置网络代理以访问被限制的网络服务</p>
      </template>

      <div class="proxy-content">
        <!-- 网络代理设置部分 -->
        <div class="proxy-section">
          <h3 class="section-title">
            <el-icon><Link /></el-icon>
            HTTP代理配置
          </h3>
          <p class="section-description">配置HTTP代理以访问被限制的网络服务（如Telegram API）</p>

          <el-form :model="proxyData" :label-position="'top'" class="proxy-form">
            <el-form-item label="HTTP代理地址" prop="proxy_url">
              <el-input
                v-model="proxyData.proxy_url"
                placeholder="例如: http://127.0.0.1:7890 或 http://proxy.example.com:8080"
                :disabled="proxyLoading"
                clearable
              />
              <div class="form-help">
                支持HTTP代理，格式：http://[用户名:密码@]主机:端口，留空表示不使用代理
              </div>
            </el-form-item>

            <el-form-item>
              <div class="proxy-actions">
                <el-button
                  type="primary"
                  @click="testProxy"
                  :loading="testingProxy"
                  :disabled="proxyLoading"
                  size="large"
                >
                  <el-icon><Connection /></el-icon>
                  测试代理连接
                </el-button>

                <el-button
                  type="success"
                  @click="saveProxy"
                  :loading="proxyLoading"
                  :disabled="testingProxy"
                  size="large"
                >
                  <el-icon><Check /></el-icon>
                  保存代理设置
                </el-button>

                <el-button
                  type="warning"
                  @click="resetProxy"
                  :disabled="proxyLoading || testingProxy"
                  size="large"
                >
                  <el-icon><RefreshLeft /></el-icon>
                  重置设置
                </el-button>
              </div>
            </el-form-item>
          </el-form>

          <!-- 代理状态显示 -->
          <el-alert
            v-if="proxyStatus"
            :title="proxyStatus.title"
            :type="proxyStatus.type"
            :description="proxyStatus.description"
            :closable="false"
            show-icon
            class="proxy-status"
          />
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { Link, Connection, Check, RefreshLeft } from '@element-plus/icons-vue'
import { inject, onMounted, reactive, ref } from 'vue'

interface ProxyData {
  proxy_url: string
}

interface ProxyStatus {
  title: string
  type: 'success' | 'warning' | 'error' | 'info'
  description: string
}

const http: AxiosStatic | undefined = inject('$http')

// 代理相关状态
const proxyLoading = ref(false)
const testingProxy = ref(false)
const proxyStatus = ref<ProxyStatus | null>(null)

const proxyData = reactive<ProxyData>({
  proxy_url: '',
})

// 测试代理连接
const testProxy = async () => {
  try {
    testingProxy.value = true
    proxyStatus.value = null

    const testData = new URLSearchParams()
    testData.append('http_proxy', proxyData.proxy_url)

    const response = await http?.post(`${SERVER_URL}/setting/test-http-proxy`, testData, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
    })

    if (response?.data.code === 200) {
      proxyStatus.value = {
        title: '代理连接测试成功',
        type: 'success',
        description: proxyData.proxy_url
          ? `代理服务器 ${proxyData.proxy_url} 连接正常`
          : '直连网络连接正常',
      }
    } else {
      proxyStatus.value = {
        title: '代理连接测试失败',
        type: 'error',
        description: response?.data.msg || '无法连接到代理服务器，请检查地址和端口',
      }
    }
  } catch (error) {
    console.error('代理测试错误:', error)
    proxyStatus.value = {
      title: '代理测试出错',
      type: 'error',
      description: '测试过程中发生错误，请检查网络连接和代理设置',
    }
  } finally {
    testingProxy.value = false
  }
}

// 保存代理设置
const saveProxy = async () => {
  try {
    proxyLoading.value = true
    proxyStatus.value = null

    const saveData = new URLSearchParams()
    saveData.append('http_proxy', proxyData.proxy_url)

    const response = await http?.post(`${SERVER_URL}/setting/update-http-proxy`, saveData, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
    })

    if (response?.data.code === 200) {
      proxyStatus.value = {
        title: '代理设置已保存',
        type: 'success',
        description: proxyData.proxy_url
          ? `已设置代理服务器：${proxyData.proxy_url}`
          : '已清除代理设置，使用直连网络',
      }
    } else {
      proxyStatus.value = {
        title: '保存代理设置失败',
        type: 'error',
        description: response?.data.msg || '保存设置失败，请重试',
      }
    }
  } catch (error) {
    console.error('保存代理设置错误:', error)
    proxyStatus.value = {
      title: '保存设置出错',
      type: 'error',
      description: '保存过程中发生错误，请检查网络连接',
    }
  } finally {
    proxyLoading.value = false
  }
}

// 重置代理设置
const resetProxy = () => {
  proxyData.proxy_url = ''
  proxyStatus.value = null
}

// 加载代理设置
const loadProxy = async () => {
  try {
    const response = await http?.get(`${SERVER_URL}/setting/http-proxy`)

    if (response?.data.code === 200 && response.data.data) {
      proxyData.proxy_url = response.data.data.http_proxy || ''
    }
  } catch (error) {
    console.error('加载代理设置错误:', error)
  }
}

onMounted(() => {
  loadProxy()
})
</script>

<style scoped>
.proxy-settings-container {
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.proxy-settings-card {
  width: 100%;
  max-width: none;
  margin: 0;
}

.card-title {
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.card-subtitle {
  margin: 0;
  font-size: 14px;
  color: #909399;
}

.proxy-content {
  margin-top: 20px;
}

.proxy-section {
  margin-bottom: 24px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 12px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.section-description {
  margin: 0 0 16px 0;
  font-size: 14px;
  color: #606266;
  line-height: 1.5;
}

.proxy-form {
  margin-top: 16px;
}

.proxy-form .el-form-item {
  margin-bottom: 20px;
}

.form-help {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

.proxy-actions {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.proxy-status {
  margin-top: 16px;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .proxy-settings-card {
    margin: 0;
  }

  .card-title {
    font-size: 20px;
  }

  .card-subtitle {
    font-size: 13px;
  }

  .proxy-actions {
    flex-direction: column;
    gap: 8px;
  }

  .proxy-actions .el-button {
    width: 100%;
  }

  .section-title {
    font-size: 16px;
  }
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .card-title {
    font-size: 18px;
  }

  .section-title {
    font-size: 15px;
  }

  .section-description {
    font-size: 13px;
  }
}
</style>
