<script setup lang="ts">
import { ref, inject } from 'vue'
import { ElMessage } from 'element-plus'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'

const http: AxiosStatic | undefined = inject('$http')
const loading = ref(false)

const repairDatabase = async () => {
  try {
    loading.value = true
    const response = await http?.post(`${SERVER_URL}/database/repair`)
    if (response?.data.code === 200) {
      ElMessage.success('数据库修复成功')
    } else {
      ElMessage.error(response?.data.message || '数据库修复失败')
    }
  } catch (error) {
    console.error('数据库修复失败:', error)
    ElMessage.error('数据库修复失败，请重试')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="database-repair-container">
    <div class="page-header">
      <div class="header-content">
        <p>修复数据库表缺失问题</p>
      </div>
    </div>

    <div class="section-card">
      <div class="section-header">
        <span class="section-icon">🔧</span>
        <span>数据库修复</span>
      </div>
      <div class="repair-content">
        <p class="repair-description">
          如果遇到错误提示: SQL logic error: no such table: 表名
        </p>
        <el-button
          type="primary"
          size="large"
          :loading="loading"
          @click="repairDatabase"
          round
        >
          {{ loading ? '修复中...' : '修复数据库' }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.database-repair-container {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px;
  color: white;
}

.header-content h1 {
  margin: 0 0 4px 0;
  font-size: 28px;
  font-weight: 700;
}

.header-content p {
  margin: 0;
  font-size: 14px;
  opacity: 0.9;
}

.section-card {
  background: white;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #f0f0f0;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.section-icon {
  font-size: 20px;
}

.repair-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  padding: 40px 20px;
}

.repair-description {
  font-size: 14px;
  color: #606266;
  text-align: center;
  margin: 0;
  line-height: 1.6;
}
</style>
