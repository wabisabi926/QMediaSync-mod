import { ref, computed, onMounted } from 'vue'
import { inject } from 'vue'
import { SERVER_URL } from '@/const'
import type { AxiosStatic } from 'axios'
import { formatDateTime } from '@/utils/timeUtils'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, LineChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
  DataZoomComponent
} from 'echarts/components'

use([
  CanvasRenderer,
  BarChart,
  LineChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
  DataZoomComponent
])

export interface HourlyStat {
  hour_ts: number
  total_requests: number
  throttled_requests: number
  avg_duration: string
}

export interface HourlyStatsData {
  start_date: string
  end_date: string
  total_requests: number
  total_throttled: number
  hourly_stats: HourlyStat[]
  query_time_range_days: number
}

export function useHourlyStats() {
  const http = inject<AxiosStatic>('$http')
  const hourlyStats = ref<HourlyStatsData | null>(null)
  const hourlyStatsLoading = ref(false)

  const loadHourlyStats = async () => {
    try {
      hourlyStatsLoading.value = true
      const response = await http?.get(`${SERVER_URL}/115/stats/hourly`)
      if (response && response.data && response.data.code === 200) {
        hourlyStats.value = response.data.data
      } else {
        hourlyStats.value = null
      }
    } catch (error) {
      console.error('加载每小时请求统计错误:', error)
      hourlyStats.value = null
    } finally {
      hourlyStatsLoading.value = false
    }
  }

  const chartOption = computed(() => {
    if (!hourlyStats.value || !hourlyStats.value.hourly_stats) {
      return {}
    }

    const hours = hourlyStats.value.hourly_stats.map(item => formatDateTime(item.hour_ts))
    const requestCounts = hourlyStats.value.hourly_stats.map(item => item.total_requests)
    const throttledCounts = hourlyStats.value.hourly_stats.map(item => item.throttled_requests)
    const avgDurations = hourlyStats.value.hourly_stats.map(item => {
      const value = parseFloat(item.avg_duration)
      return isNaN(value) ? 0 : Math.round(value)
    })

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'shadow'
        }
      },
      legend: {
        data: ['请求数', '限流次数', '平均响应时间(ms)'],
        top: 10
      },
      grid: {
        left: '50px',
        right: '4%',
        bottom: '60px',
        top: '60px'
      },
      xAxis: {
        type: 'category',
        data: hours,
        axisLabel: {
          rotate: 45,
          interval: 0,
          fontSize: 10
        }
      },
      yAxis: [
        {
          type: 'value',
          name: '次数',
          position: 'left'
        },
        {
          type: 'value',
          name: '响应时间(ms)',
          position: 'right'
        }
      ],
      series: [
        {
          name: '请求数',
          type: 'bar',
          yAxisIndex: 0,
          data: requestCounts,
          itemStyle: {
            color: '#409eff'
          }
        },
        {
          name: '限流次数',
          type: 'bar',
          yAxisIndex: 0,
          data: throttledCounts,
          itemStyle: {
            color: '#f56c6c'
          }
        },
        {
          name: '平均响应时间(ms)',
          type: 'line',
          yAxisIndex: 1,
          data: avgDurations,
          itemStyle: {
            color: '#67c23a'
          },
          lineStyle: {
            width: 2
          },
          smooth: true
        }
      ],
      dataZoom: [
        {
          type: 'inside',
          start: 0,
          end: 100
        },
        {
          start: 0,
          end: 100
        }
      ]
    }
  })

  onMounted(() => {
    loadHourlyStats()
  })

  return {
    hourlyStats,
    hourlyStatsLoading,
    chartOption,
    loadHourlyStats
  }
}
