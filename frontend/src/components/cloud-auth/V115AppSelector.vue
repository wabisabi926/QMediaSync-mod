<script setup lang="ts">
const appName = defineModel<string>('appName', { required: true })
const appId = defineModel<string>('appId', { required: true })

const appOptions = [
  { label: 'QMediaSync', value: 'QMediaSync' },
  { label: '自定义 APPID', value: '自定义' },
]
</script>

<template>
  <div class="v115-app-selector">
    <el-form-item label="115开放平台应用">
      <el-segmented v-model="appName" :options="appOptions" />
    </el-form-item>
    <el-form-item v-if="appName === '自定义'" label="APPID">
      <el-input
        v-model="appId"
        name="v115-app-id"
        autocomplete="off"
        placeholder="请输入 115 开放平台 APPID…"
        clearable
      />
    </el-form-item>
  </div>
</template>
