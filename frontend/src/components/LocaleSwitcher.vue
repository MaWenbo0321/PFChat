<template>
  <el-dropdown @command="handleCommand" trigger="click">
    <el-button circle :icon="Location">
      <span v-if="showText" class="locale-text">{{ currentLocaleName }}</span>
    </el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item command="zh-CN" :class="{ active: locale === 'zh-CN' }">
          <el-icon v-if="locale === 'zh-CN'"><Check /></el-icon>
          简体中文
        </el-dropdown-item>
        <el-dropdown-item command="en-US" :class="{ active: locale === 'en-US' }">
          <el-icon v-if="locale === 'en-US'"><Check /></el-icon>
          English
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Location, Check } from '@element-plus/icons-vue'
import { switchLocale } from '@/utils/locale'

defineProps({
  showText: {
    type: Boolean,
    default: false
  }
})

const { locale } = useI18n()

const currentLocaleName = computed(() => {
  return locale.value === 'zh-CN' ? '中文' : 'EN'
})

const handleCommand = (command) => {
  switchLocale(command, { global: { locale: { value: locale.value } } })
  // 刷新页面以应用新语言
  window.location.reload()
}
</script>

<style scoped>
.locale-text {
  margin-left: 8px;
  font-size: 14px;
}

.active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

:deep(.el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
