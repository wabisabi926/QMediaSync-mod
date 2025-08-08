<script setup lang="ts">
import MarkdownIt from 'markdown-it'
const md = new MarkdownIt()
const content = `
## 注意事项
1. 同一个账号同一时间只能有一个生成目录树的操作
1. 115网盘中的目录名不能包含媒体文件扩展名，否则会被识别为文件而不是目录
    > 比如战狼电影：Media/Movie/战狼.FLAC.MP4/战狼.FLAC.MP4，这个目录会被识别为两个MP4文件
    - Media/Movie/战狼.FLAC.MP4
    - Media/Movie/战狼.FLAC.MP4/战狼.FLAC.MP4
    > 这是由于115目录树不包含文件元数据，只能通过是否有媒体文件扩展名来确定到底是文件还是目录
1. 如果文件很多，建议添加多个同步目录，这样处理速度更快；否则请添加一个同步目录即可
1. 如果您已经部署了CookieCloud，建议使用CookieCloud来同步115cookie，这样不影响浏览器访问115。
1. 除了目录树以外，本项目其他操作都使用115开放平台接口并且做了访问频率限制，理论上不存在被115封禁的风险
1. 如果要在多台机器部署本项目且使用同一115账号，请只在其中一台机器上开启同步，其他机器请使用其他同步工具（如微力同步）同步strm目录的文件即可。

## 使用步骤：
1. 系统设置-核心设置-登录115账号 或者 系统设置-CookieCloud设置 - 开启CookieCloud
2. 如果开启CookieCloud，请在设置完成后回到系统设置-核心设置 查看115账号是否正常获取
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
