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

该函数根据同步结果中的新增 STRM 数、下载元数据数和任务状态派生展示状态。前端只展示是否存在媒体库刷新相关变更，不声明后端已提交刷新任务；实际刷新还需要后端满足 Emby 已启用和同步目录已关联媒体库等条件。

## taskSourceUtils.ts

任务来源、同步队列任务类型和来源类型展示映射：

- `getDownloadSourceName(source)`
- `getUploadSourceName(source)`
- `getSyncTaskTypeName(taskType)`
- `getTaskSourceTypeName(type)`
- `getTaskSourceTypeTagType(type)`

## timeUtils.ts

时间、存储空间和状态样式辅助。业务时间统一使用后端返回的 Unix 秒，并在前端按浏览器本地环境格式化；日志字符串保持原始日志格式，不强制转换。新接口如需毫秒时间或耗时，字段名必须使用 `_ms` 后缀，例如 `duration_ms`、`event_time_ms`。

- `formatUnixDateTime(timestamp)`
- `formatMaybeUnixDateTime(value)`
- `formatUnixDate(timestamp)`
- `formatRelativeTime(value)`
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

兼容规则：

- Unix 秒数字按秒处理，展示为本地日期时间。
- RFC3339 UTC 字符串按浏览器本地时区展示。
- 旧无时区 `date` 字符串只作为兼容回退；新接口不要继续新增这种字段。
- 空值、`0` 和非法值返回 `-`，业务组件可按场景转换为“未同步”等文案。

## mockAPI.ts

当前为空文件，保留为本地开发或后续模拟接口辅助的占位文件。
