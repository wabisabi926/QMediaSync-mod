# 工具函数目录

`frontend/src/utils` 存放前端跨组件复用的小工具。当前目录按功能拆分为以下文件。

## cloudAccountUtils.ts

云盘账号和 115 开放平台应用信息展示辅助：

- `CloudAccountAppInfo`
- `isCustomV115App(account)`
- `isBuiltInV115App(appName)`
- `getV115AppInfoRows(account)`

## deviceUtils.ts

设备类型判断和窗口尺寸变化监听：

- `isMobile()`
- `onDeviceTypeChange(callback)`

在组件中按需导入这两个函数即可；`onDeviceTypeChange` 会返回一个取消监听的函数。

```typescript
const removeListener = onDeviceTypeChange((nextIsMobile) => {
  console.log(nextIsMobile)
})
```

## fileIconUtils.ts

文件类型识别和 Element Plus 图标名称映射：

- `getFileType(filename)`
- `getFileIcon(type, isDirectory)`
- `getFileIconByName(filename, isDirectory)`

```typescript
const icon = getFileIconByName('movie.mp4')
```

支持的视频扩展名：`mp4`、`mkv`、`avi`、`mov`、`wmv`、`flv`、`m4v`、`webm`、`ts`、`rmvb`、`rm`、`3gp`、`mpg`、`mpeg`。

支持的图片扩展名：`jpg`、`jpeg`、`png`、`gif`、`bmp`、`webp`、`svg`、`ico`、`tiff`、`tga`。

## fileSizeUtils.ts

文件大小格式化：

- `formatFileSize(bytes)`

## notificationUtils.ts

通知渠道、事件类型和展示文本辅助：

- `ChannelType`
- `EventType`
- `NotificationChannel`
- `NotificationConfig`
- `NotificationRule`
- `getChannelTypeName(type)`
- `getChannelTypeColor(type)`
- `getEventTypeName(type)`
- `getEventTypeDescription(type)`

## oauthCallback.ts

OAuth 回调参数收集：

- `collectOAuthCallbackParams(search, hash)`

该函数会合并普通 query string 和 hash 中的 query 参数。

## sourceTypeUtils.ts

同步来源类型展示配置：

- `sourceTypeOptions`
- `sourceTypeTagMap`
- `sourceTypeMap`

当前启用的来源类型包括 `115`、`baidupan`、`openlist`、`local`。

## syncRefreshDecision.ts

同步任务完成后是否提交 Emby 媒体库刷新的展示决策：

- `getEmbyRefreshDecision(input)`

该函数根据同步结果中的新增 STRM 数、下载元数据数和任务状态派生展示状态；已完成任务二者皆为 `0` 时展示为未提交媒体库刷新，运行中任务会展示为暂无媒体库刷新变更。

## taskSourceUtils.ts

任务来源、同步队列任务类型和来源类型展示映射：

- `getDownloadSourceName(source)`
- `getUploadSourceName(source)`
- `getSyncTaskTypeName(taskType)`
- `getTaskSourceTypeName(type)`
- `getTaskSourceTypeTagType(type)`

## timeUtils.ts

时间、存储空间和状态样式辅助：

- `formatTimestamp(timestamp)`
- `formatDateTime(timestamp)`
- `formatTime(timestamp)`
- `formatStorage(bytes)`
- `getStoragePercent(used, total)`
- `getProgressColor(used, total)`
- `getMemberClass(level)`
- `formatExpireTime(expireTime)`
- `getExpireClass(expireTime)`
- `formatDuration(seconds)`

## mockAPI.ts

当前为空文件，保留为本地开发或后续模拟接口辅助的占位文件。
