<script setup lang="ts">
import MarkdownIt from 'markdown-it'
const md = new MarkdownIt()
const content = `
## 注意事项
1. 如果本地没有元数据，那么首次同步根据网盘库的大小，可能需要N个小时
1. 如果要在多台机器部署本项目且使用同一115账号，请只在其中一台机器上开启同步，其他机器请使用其他同步工具（如微力同步）同步strm目录的文件即可。

## 使用步骤：
1. 系统设置-核心设置-扫码授权115开放平台，页面显示出您的账号信息后表示授权成功
3. 系统设置-strm设置：输入strm直连地址，其他参数请根据需要修改
5. 同步记录 - 手动同步 进行首次全量同步（可能时间较长）
`
const result = md.render(content)
</script>
<template>
  <div class="home-container">
    <div class="greetings" v-html="result"></div>
  </div>
</template>

<style scoped>
.home-container {
  max-width: 100%;
  padding: 0;
}

.greetings {
  padding: 20px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  line-height: 1.6;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .greetings {
    padding: 15px;
    margin: 0;
    border-radius: 4px;
    font-size: 14px;
  }

  .greetings h2 {
    font-size: 18px;
    margin-top: 20px;
    margin-bottom: 12px;
  }

  .greetings h2:first-child {
    margin-top: 0;
  }

  .greetings ol,
  .greetings ul {
    padding-left: 20px;
  }

  .greetings li {
    margin-bottom: 8px;
  }

  .greetings blockquote {
    margin: 10px 0;
    padding: 8px 12px;
    border-left: 3px solid #409eff;
    background-color: #f4f4f5;
    font-size: 13px;
  }

  .greetings code {
    padding: 2px 4px;
    background-color: #f1f1f1;
    border-radius: 3px;
    font-size: 12px;
  }
}

/* 小屏移动设备 */
@media (max-width: 480px) {
  .greetings {
    padding: 12px;
    font-size: 13px;
  }

  .greetings h2 {
    font-size: 16px;
  }

  .greetings ol,
  .greetings ul {
    padding-left: 16px;
  }
}
</style>
