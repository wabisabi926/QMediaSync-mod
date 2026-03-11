<template>
  <div class="welcome-banner">
    <div class="banner-content">
      欢迎使用 QMediaSync
    </div>
  </div>
</template>

<script setup lang="ts">
// Banner 组件 - 显示欢迎消息
</script>

<style scoped>
.welcome-banner {
  width: 100%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 16px 0;
  text-align: center;
  font-size: 18px;
  font-weight: 500;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.banner-content {
  display: inline-block;
  letter-spacing: 1px;
}

@media (max-width: 768px) {
  .welcome-banner {
    font-size: 16px;
    padding: 12px 0;
  }
}
</style>
