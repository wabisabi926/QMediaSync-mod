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

## fileIconUtils.ts

该文件包含文件类型识别和图标映射功能：

### getFileType(filename)

根据文件名识别文件类型。

```typescript
import { getFileType } from '@/utils/fileIconUtils'

const type = getFileType('movie.mp4') // returns 'video'
const type2 = getFileType('image.jpg') // returns 'image'
const type3 = getFileType('info.nfo') // returns 'nfo'
```

### getFileIcon(type, isDirectory)

根据文件类型获取Element Plus图标名称。

```typescript
import { getFileIcon } from '@/utils/fileIconUtils'

const icon = getFileIcon('video') // returns 'VideoPlay'
const icon2 = getFileIcon('directory', true) // returns 'Folder'
```

### getFileIconByName(filename, isDirectory)

根据文件名直接获取图标名称（组合了getFileType和getFileIcon）。

```typescript
import { getFileIconByName } from '@/utils/fileIconUtils'

const icon = getFileIconByName('movie.mp4') // returns 'VideoPlay'
```

**支持的文件类型：**
- **视频文件**：mp4, mkv, avi, mov, wmv, flv, m4v, webm, ts, rmvb, rm, 3gp, mpg, mpeg
- **图片文件**：jpg, jpeg, png, gif, bmp, webp, svg, ico, tiff, tga  
- **NFO文件**：nfo