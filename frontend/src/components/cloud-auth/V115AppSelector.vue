<script setup lang="ts">
const appName = defineModel<string>('appName', { required: true })
const appId = defineModel<string>('appId', { required: true })
const customAppName = defineModel<string>('customAppName', { required: true })

const appOptions = [
  { label: 'QMediaSync', value: 'QMediaSync' },
  { label: 'Q115-STRM', value: 'Q115-STRM' },
  { label: 'MQ的媒体库', value: 'MQ的媒体库' },
  { label: '自定义 APPID', value: '自定义' },
]
</script>

<template>
  <div class="v115-app-selector">
    <el-form-item label="115开放平台应用">
      <el-segmented v-model="appName" :options="appOptions" />
    </el-form-item>
    <template v-if="appName === '自定义'">
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
          v-model="appId"
          name="v115-app-id"
          autocomplete="off"
          placeholder="请输入 115 开放平台 APPID…"
          clearable
        />
      </el-form-item>
    </template>
  </div>
</template>
