<template>
  <el-config-provider :locale="elementLocale">
    <div class="app-root">
      <router-view />
    </div>
  </el-config-provider>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import en from 'element-plus/dist/locale/en.mjs'

const { locale } = useI18n()
const elementLocale = computed(() => locale.value === 'zh-CN' ? zhCn : en)
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

.app-root {
  width: 100%;
  min-height: 100vh;
  overflow-x: hidden;
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', Arial, sans-serif;
}
</style>
