<script setup lang="ts">
import { computed } from 'vue'

const currentPage = defineModel<number>('currentPage', { required: true })
const pageSize = defineModel<number>('pageSize', { required: true })

const props = withDefaults(
  defineProps<{
    total: number
    pageSizes?: number[]
    isMobile?: boolean
    mobileLayout?: string
    desktopLayout?: string
    background?: boolean
  }>(),
  {
    pageSizes: () => [10, 20, 50, 100],
    isMobile: false,
    mobileLayout: 'total, prev, pager, next',
    desktopLayout: 'total, sizes, prev, pager, next, jumper',
    background: true,
  },
)

const emit = defineEmits<{
  sizeChange: [size: number]
  currentChange: [page: number]
}>()

const layout = computed(() => (props.isMobile ? props.mobileLayout : props.desktopLayout))
const pagerCount = computed(() => (props.isMobile ? 5 : 7))
</script>

<template>
  <div class="responsive-pagination">
    <el-pagination
      v-model:current-page="currentPage"
      v-model:page-size="pageSize"
      :page-sizes="pageSizes"
      :pager-count="pagerCount"
      :small="isMobile"
      :background="background"
      :layout="layout"
      :total="total"
      @size-change="emit('sizeChange', $event)"
      @current-change="emit('currentChange', $event)"
    />
  </div>
</template>

<style scoped>
.responsive-pagination {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px 12px;
  margin-top: 20px;
  overflow: visible;
}

.responsive-pagination :deep(.el-pagination__total),
.responsive-pagination :deep(.el-pagination__sizes),
.responsive-pagination :deep(.el-pagination__jump) {
  margin-right: 0;
}

@media (max-width: 768px) {
  .responsive-pagination {
    justify-content: center;
    margin-top: 12px;
  }

  .responsive-pagination :deep(.el-pagination) {
    justify-content: center;
  }
}
</style>
