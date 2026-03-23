<script setup>
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import AdminReviewPage from './components/AdminReviewPage.vue'
</script>

<template>
  <a-config-provider
    :locale="zhCN"
    :theme="{
      token: {
        colorPrimary: '#1677ff',
        borderRadius: 8,
        fontFamily: '\'Manrope\', \'PingFang SC\', \'Microsoft YaHei\', sans-serif'
      }
    }"
  >
    <a-app>
      <AdminReviewPage />
    </a-app>
  </a-config-provider>
</template>
