<script setup lang="ts">
import type { AxiosStatic } from 'axios'
import { computed, inject, watch } from 'vue'

import { useV115AppIdSearch } from '@/composables/useV115AppIdSearch'
import {
  featuredBuiltInAppIDs,
  pinnedBuiltInAppIDs,
  type V115SelectedQrApp,
} from './v115AuthSources'

const selectedQrApp = defineModel<V115SelectedQrApp>('selectedQrApp', { required: true })
const customAppId = defineModel<string>('customAppId', { required: true })
const customAppName = defineModel<string>('customAppName', { required: true })

const http = inject<AxiosStatic | undefined>('$http')
const { keyword, items, loading, hasMore, search, loadMore, reset } = useV115AppIdSearch({ http })

const defaultOptions = computed(() => [
  ...pinnedBuiltInAppIDs,
  ...featuredBuiltInAppIDs,
  { label: '自定义 APPID', value: 'custom', appName: '自定义 APPID' },
])
const remoteOptions = computed(() =>
  items.value.map((item) => ({
    label: item.display_name || item.app_name,
    value: item.app_id,
    appName: item.app_name,
  })),
)
const selectOptions = computed(() => {
  if (keyword.value.trim()) {
    return remoteOptions.value
  }
  return defaultOptions.value
})
const selectedValue = computed({
  get: () => selectedQrApp.value.appId,
  set: (value) => {
    const option = [...defaultOptions.value, ...remoteOptions.value].find((item) => item.value === value)
    selectedQrApp.value = {
      appId: value,
      appName: option?.appName || option?.label || value,
    }
  },
})
const showCustomFields = computed(() => selectedQrApp.value.appId === 'custom')

const handleSearch = (value: string) => {
  keyword.value = value
  if (!value.trim()) {
    reset()
    return
  }
  void search()
}

watch(showCustomFields, (visible) => {
  if (!visible) return
  if (!customAppName.value) {
    customAppName.value = ''
  }
})
</script>

<template>
  <el-form-item label="APPID">
    <el-select
      v-model="selectedValue"
      class="v115-app-select"
      filterable
      remote
      :remote-method="handleSearch"
      :loading="loading"
    >
      <el-option
        v-for="option in selectOptions"
        :key="option.value"
        :label="option.label"
        :value="option.value"
      />
      <el-option v-if="hasMore" class="v115-load-more-option" label="加载更多" value="__load_more">
        <el-button text type="primary" @click.stop="loadMore">加载更多</el-button>
      </el-option>
    </el-select>
  </el-form-item>
  <template v-if="showCustomFields">
    <el-form-item label="应用名">
      <el-input
        v-model="customAppName"
        name="v115-custom-app-name"
        autocomplete="off"
        placeholder="请输入应用名，可留空"
        clearable
      />
    </el-form-item>
    <el-form-item label="APPID">
      <el-input
        v-model="customAppId"
        name="v115-app-id"
        autocomplete="off"
        placeholder="请输入 115 开放平台 APPID"
        clearable
      />
    </el-form-item>
  </template>
</template>

<style scoped>
.v115-app-select {
  width: 100%;
}

.v115-load-more-option {
  text-align: center;
}
</style>
