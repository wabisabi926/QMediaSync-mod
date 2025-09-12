# 工具函数目录

## deviceUtils.ts

该文件包含设备相关的工具函数：

### isMobile()

检测当前设备是否为移动端。

```typescript
import { isMobile } from '@/utils/deviceUtils'

// 检测是否为移动端
cosnt mobile = isMobile()
```

### onDeviceTypeChange(callback)

监听设备类型变化，当设备类型从移动端切换到桌面端或反之亦然时触发回调。

```typescript
import { onDeviceTypeChange } from '@/utils/deviceUtils'

// 监听设备类型变化
const removeListener = onDeviceTypeChange((isMobile) => {
  console.log('设备类型变化:', isMobile ? '移动端' : '桌面端')
})

// 在组件卸载时移除监听器
// removeListener()
```