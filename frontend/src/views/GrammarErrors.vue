<template>
  <div class="grammar-errors-container">
    <div class="header">
      <div class="header-left">
        <el-button :icon="ArrowLeft" @click="$router.back()">{{ $t('grammar.back') }}</el-button>
        <h1>{{ $t('grammar.title') }}</h1>
      </div>
      <div class="header-actions">
        <el-button
            v-if="currentTypeFilter !== 'all'"
            type="warning"
            :icon="Delete"
            @click="clearCurrentType"
            :disabled="filteredErrors.length === 0"
        >
          {{ $t('grammar.clearType') }}
        </el-button>
        <el-button
            type="danger"
            :icon="Delete"
            @click="clearAll"
            :disabled="errors.length === 0"
        >
          {{ $t('grammar.clearAll') }}
        </el-button>
      </div>
    </div>

    <div class="content">
      <!-- 统计卡片 -->
      <div class="stats">
        <el-statistic :title="$t('grammar.totalErrors')" :value="statistics.total" />
        <el-statistic :title="$t('grammar.type1Count')" :value="statistics.by_type['错误1'] || 0" />
        <el-statistic :title="$t('grammar.type2Count')" :value="statistics.by_type['错误2'] || 0" />
        <el-statistic :title="$t('grammar.weekErrors')" :value="statistics.week_count" />
        <el-statistic :title="$t('grammar.todayErrors')" :value="statistics.today_count" />
      </div>

      <!-- 筛选器 -->
      <div class="filters">
        <el-input
            v-model="searchKeyword"
            :placeholder="$t('grammar.search')"
            prefix-icon="Search"
            clearable
            style="width: 300px"
        />
        <el-select
            v-model="currentTypeFilter"
            :placeholder="$t('grammar.typeFilter')"
            style="width: 180px"
            @change="handleTypeChange"
        >
          <el-option :label="$t('grammar.allTypes')" value="all" />
          <el-option :label="$t('grammar.errorType1')" value="错误1" />
          <el-option :label="$t('grammar.errorType2')" value="错误2" />
        </el-select>
      </div>

      <!-- 按类型分组展示 -->
      <div class="error-groups">
        <template v-for="(groupErrors, errorType) in groupedErrors" :key="errorType">
          <div class="error-group" v-if="groupErrors.length > 0">
            <div class="group-header">
              <h2>
                <el-tag :type="errorType === '错误1' ? 'danger' : 'warning'" size="large">
                  {{ errorType === '错误1' ? $t('grammar.errorType1') : $t('grammar.errorType2') }}
                </el-tag>
                <span class="group-count">({{ groupErrors.length }})</span>
              </h2>
            </div>

            <div class="error-list">
              <el-card
                  v-for="error in groupErrors"
                  :key="error.id"
                  class="error-card"
                  shadow="hover"
              >
                <div class="error-header">
                  <span class="error-time">
                    <el-icon><Clock /></el-icon>
                    {{ formatDate(error.created_at) }}
                  </span>
                  <div class="error-actions">
                    <el-dropdown @command="(cmd) => handleCommand(cmd, error)">
                      <el-button size="small" :icon="MoreFilled" text />
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item command="changeType">
                            <el-icon><Edit /></el-icon>
                            {{ $t('grammar.changeType') }}
                          </el-dropdown-item>
                          <el-dropdown-item command="delete" divided>
                            <el-icon color="#f56c6c"><Delete /></el-icon>
                            {{ $t('grammar.delete') }}
                          </el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </div>
                </div>

                <div class="error-content">
                  <div class="section">
                    <div class="section-title">
                      <el-icon color="#f56c6c"><WarningFilled /></el-icon>
                      {{ $t('grammar.originalText') }}
                    </div>
                    <div class="original-text">{{ error.original_text }}</div>
                  </div>

                  <div class="section">
                    <div class="section-title">
                      <el-icon color="#67c23a"><Check /></el-icon>
                      {{ $t('grammar.suggestion') }}
                    </div>
                    <div class="suggestion-text">
                      <span>{{ error.llm_suggestion }}</span>
                      <el-button
                          :icon="CopyDocument"
                          size="small"
                          @click="copyText(error.llm_suggestion)"
                          text
                      >
                        {{ $t('grammar.copy') }}
                      </el-button>
                    </div>
                  </div>

                  <div class="section">
                    <div class="section-title">
                      <el-icon color="#409eff"><InfoFilled /></el-icon>
                      {{ $t('grammar.explanation') }}
                    </div>
                    <div class="explanation-text">{{ error.llm_explanation }}</div>
                  </div>

                  <div v-if="error.message_deleted" class="deleted-notice">
                    <el-icon color="#909399"><Warning /></el-icon>
                    <span>{{ $t('grammar.messageDeleted') }}</span>
                  </div>
                </div>
              </el-card>
            </div>
          </div>
        </template>

        <!-- 空状态 -->
        <el-empty
            v-if="Object.keys(groupedErrors).every(key => groupedErrors[key].length === 0)"
            :description="$t('grammar.noErrors')"
        />
      </div>
    </div>

    <!-- 更改错误类型对话框 -->
    <el-dialog
        v-model="changeTypeDialogVisible"
        :title="$t('grammar.changeType')"
        width="400px"
    >
      <el-form :model="changeTypeForm">
        <el-form-item :label="$t('grammar.errorTypeLabel')">
          <el-select v-model="changeTypeForm.newType" style="width: 100%">
            <el-option :label="$t('grammar.errorType1')" value="错误1" />
            <el-option :label="$t('grammar.errorType2')" value="错误2" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="changeTypeDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="confirmChangeType">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft,
  Delete,
  Clock,
  WarningFilled,
  Check,
  InfoFilled,
  Warning,
  CopyDocument,
  MoreFilled,
  Edit
} from '@element-plus/icons-vue'
import api from '@/api'

const { t, locale } = useI18n()

const errors = ref([])
const statistics = ref({
  total: 0,
  by_type: {},
  today_count: 0,
  week_count: 0
})
const searchKeyword = ref('')
const currentTypeFilter = ref('all')
const changeTypeDialogVisible = ref(false)
const changeTypeForm = ref({
  errorId: null,
  newType: '错误1'
})

// 按搜索关键词筛选
const filteredErrors = computed(() => {
  if (!searchKeyword.value) return errors.value

  const keyword = searchKeyword.value.toLowerCase()
  return errors.value.filter(e =>
      e.original_text.toLowerCase().includes(keyword) ||
      e.llm_suggestion.toLowerCase().includes(keyword) ||
      e.llm_explanation.toLowerCase().includes(keyword)
  )
})

// 按错误类型分组
const groupedErrors = computed(() => {
  const groups = {
    '错误1': [],
    '错误2': []
  }

  filteredErrors.value.forEach(error => {
    if (groups[error.error_type]) {
      groups[error.error_type].push(error)
    }
  })

  return groups
})

onMounted(async () => {
  await loadErrors()
})

const loadErrors = async (errorType = 'all') => {
  try {
    const response = await api.getGrammarErrors(errorType)
    errors.value = response.errors || []
    statistics.value = response.statistics || {
      total: 0,
      by_type: {},
      today_count: 0,
      week_count: 0
    }
  } catch (error) {
    console.error('Load errors:', error)
    ElMessage.error(t('common.error'))
  }
}

const handleTypeChange = async (value) => {
  await loadErrors(value)
}

const handleCommand = (command, error) => {
  if (command === 'delete') {
    deleteError(error.id)
  } else if (command === 'changeType') {
    openChangeTypeDialog(error)
  }
}

const openChangeTypeDialog = (error) => {
  changeTypeForm.value = {
    errorId: error.id,
    newType: error.error_type === '错误1' ? '错误2' : '错误1'
  }
  changeTypeDialogVisible.value = true
}

const confirmChangeType = async () => {
  try {
    await api.updateGrammarErrorType(changeTypeForm.value.errorId, changeTypeForm.value.newType)
    ElMessage.success(t('grammar.updateTypeSuccess'))
    changeTypeDialogVisible.value = false
    await loadErrors(currentTypeFilter.value)
  } catch (error) {
    console.error('Update type error:', error)
    ElMessage.error(t('grammar.updateTypeFailed'))
  }
}

const deleteError = async (id) => {
  try {
    await ElMessageBox.confirm(
        t('grammar.deleteConfirm'),
        t('chat.hint'),
        {
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel'),
          type: 'warning'
        }
    )

    await api.deleteGrammarError(id)
    ElMessage.success(t('grammar.deleteSuccess'))
    await loadErrors(currentTypeFilter.value)
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Delete error:', error)
    }
  }
}

const clearCurrentType = async () => {
  const currentErrors = groupedErrors.value[currentTypeFilter.value] || []

  try {
    await ElMessageBox.confirm(
        t('grammar.clearTypeConfirm', { count: currentErrors.length }),
        t('chat.warning'),
        {
          confirmButtonText: t('chat.confirmClear'),
          cancelButtonText: t('common.cancel'),
          type: 'warning',
          confirmButtonClass: 'el-button--danger'
        }
    )

    await api.clearGrammarErrorsByType(currentTypeFilter.value)
    ElMessage.success(t('grammar.clearSuccess'))
    await loadErrors(currentTypeFilter.value)
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Clear type error:', error)
    }
  }
}

const clearAll = async () => {
  try {
    await ElMessageBox.confirm(
        t('grammar.clearAllConfirm', { count: errors.value.length }),
        t('chat.warning'),
        {
          confirmButtonText: t('chat.confirmClear'),
          cancelButtonText: t('common.cancel'),
          type: 'warning',
          confirmButtonClass: 'el-button--danger'
        }
    )

    await api.clearGrammarErrorsByType('all')
    ElMessage.success(t('grammar.clearSuccess'))
    await loadErrors(currentTypeFilter.value)
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Clear all error:', error)
    }
  }
}

const copyText = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('grammar.copySuccess'))
  } catch (error) {
    ElMessage.error(t('grammar.copyFailed'))
  }
}

const formatDate = (dateString) => {
  const date = new Date(dateString)
  const localeString = locale.value === 'zh-CN' ? 'zh-CN' : 'en-US'

  return date.toLocaleString(localeString, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<style scoped>
.grammar-errors-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f5f5f5;
}

.header {
  background: white;
  padding: 20px 30px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.header h1 {
  margin: 0;
  font-size: 24px;
  color: #303133;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.content {
  flex: 1;
  padding: 30px;
  overflow-y: auto;
}

.stats {
  display: flex;
  gap: 30px;
  margin-bottom: 30px;
  flex-wrap: wrap;
}

.filters {
  display: flex;
  gap: 15px;
  margin-bottom: 30px;
}

.error-groups {
  display: flex;
  flex-direction: column;
  gap: 30px;
}

.error-group {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.group-header {
  margin-bottom: 20px;
  padding-bottom: 15px;
  border-bottom: 2px solid #f0f0f0;
}

.group-header h2 {
  margin: 0;
  font-size: 20px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.group-count {
  color: #909399;
  font-size: 16px;
  font-weight: normal;
}

.error-list {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.error-card {
  transition: all 0.3s;
}

.error-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.error-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
  padding-bottom: 10px;
  border-bottom: 1px solid #ebeef5;
}

.error-time {
  display: flex;
  align-items: center;
  gap: 5px;
  color: #909399;
  font-size: 14px;
}

.error-actions {
  display: flex;
  gap: 5px;
}

.error-content {
  font-size: 14px;
}

.section {
  margin: 15px 0;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  margin-bottom: 10px;
  color: #606266;
}

.original-text {
  padding: 12px;
  background: #fef0f0;
  border-left: 3px solid #f56c6c;
  border-radius: 4px;
  color: #606266;
  line-height: 1.6;
}

.suggestion-text {
  padding: 12px;
  background: #f0f9ff;
  border-left: 3px solid #67c23a;
  border-radius: 4px;
  color: #606266;
  line-height: 1.6;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.explanation-text {
  padding: 12px;
  background: #f4f4f5;
  border-left: 3px solid #409eff;
  border-radius: 4px;
  color: #606266;
  line-height: 1.6;
}

.deleted-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px;
  background: #f4f4f5;
  border-radius: 4px;
  color: #909399;
  font-size: 13px;
  margin-top: 10px;
}
</style>