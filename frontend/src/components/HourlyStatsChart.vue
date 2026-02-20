<script setup lang="ts">
import { useHourlyStats } from '@/composables/useHourlyStats'
import VChart from 'vue-echarts'

const { hourlyStats, hourlyStatsLoading, chartOption, loadHourlyStats } = useHourlyStats()
</script>

<template>
  <div class="chart-card" v-loading="hourlyStatsLoading">
    <div class="chart-header">
      <div class="chart-title">
        <span class="title-icon">📈</span>
        <span>请求趋势</span>
        <span class="chart-period">{{ hourlyStats?.start_date }} ~ {{ hourlyStats?.end_date }}</span>
      </div>
      <el-button type="primary" size="small" @click="loadHourlyStats" :loading="hourlyStatsLoading" round>
        刷新
      </el-button>
    </div>

    <div v-if="hourlyStats" class="chart-content">
      <div class="chart-summary">
        <div class="summary-item">
          <div class="summary-value">{{ hourlyStats.total_requests }}</div>
          <div class="summary-label">总请求</div>
        </div>
        <div class="summary-item" :class="{ 'summary-danger': hourlyStats.total_throttled > 0 }">
          <div class="summary-value">{{ hourlyStats.total_throttled }}</div>
          <div class="summary-label">总限流</div>
        </div>
      </div>
      <div class="chart-wrapper">
        <v-chart class="chart" :option="chartOption" autoresize />
      </div>
    </div>

    <div v-else class="empty-state">
      <el-empty description="暂无统计数据" :image-size="60" />
    </div>
  </div>
</template>

<style scoped>
.chart-card {
  background: white;
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #f0f0f0;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.chart-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.title-icon {
  font-size: 20px;
}

.chart-period {
  font-size: 12px;
  color: #909399;
  font-weight: 400;
  margin-left: 8px;
}

.chart-summary {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.summary-item {
  flex: 1;
  padding: 16px;
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e7ed 100%);
  border-radius: 12px;
  text-align: center;
}

.summary-item.summary-danger {
  background: linear-gradient(135deg, #ffebee 0%, #ffcdd2 100%);
}

.summary-value {
  font-size: 28px;
  font-weight: 700;
  color: #303133;
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
}

.summary-item.summary-danger .summary-value {
  color: #c62828;
}

.summary-label {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.chart-wrapper {
  width: 100%;
  height: 350px;
}

.chart {
  width: 100%;
  height: 100%;
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
}

@media (max-width: 768px) {
  .chart-wrapper {
    height: 280px;
  }

  .summary-value {
    font-size: 24px;
  }
}
</style>
