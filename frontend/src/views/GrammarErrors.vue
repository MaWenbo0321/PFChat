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
            :disabled="statistics.total === 0"
        >
          {{ $t('grammar.clearAll') }}
        </el-button>
      </div>
    </div>

    <div class="content">
      <!-- 统计卡片 -->
      <div class="stats">
        <el-statistic :title="$t('grammar.totalErrors')" :value="statistics.total" />
        <el-statistic :title="$t('grammar.todayErrors')" :value="statistics.today_count" />
        <el-statistic :title="$t('grammar.weekErrors')" :value="statistics.week_count" />
        <el-statistic :title="$t('grammar.improvableCount')" :value="improvableTotal" />
        <el-statistic :title="$t('grammar.problematicCount')" :value="problematicTotal" />
      </div>

      <!-- 筛选器 -->
      <div class="filters">
        <el-input
            v-model="searchKeyword"
            :placeholder="$t('grammar.search')"
            :prefix-icon="Search"
            clearable
            style="width: 300px"
        />
        <el-select
            v-model="currentTypeFilter"
            :placeholder="$t('grammar.typeFilter')"
            style="width: 300px"
            @change="handleTypeChange"
        >
          <el-option :label="$t('grammar.allTypes')" value="all" />
          <el-option :label="$t('grammar.errorType1')" value="语用语言失误" />
          <el-option :label="$t('grammar.errorType2')" value="社会语用失误" />
          <el-option :label="$t('grammar.errorType3')" value="严重语用语言失误" />
          <el-option :label="$t('grammar.errorType4')" value="严重社会语用失误" />
          <el-option :label="$t('grammar.errorType5')" value="语用语言失误和社会语用失误" />
          <el-option :label="$t('grammar.errorType6')" value="无明显语用失误" />
        </el-select>
        <el-select
            v-model="currentModeFilter"
            :placeholder="$t('grammar.modeFilter')"
            style="width: 260px"
            @change="handleModeChange"
        >
          <el-option :label="$t('grammar.allModes')" value="all" />
          <el-option :label="$t('grammar.userL2Mode')" value="user_l2" />
          <el-option :label="$t('grammar.llmL2Mode')" value="llm_l2" />
        </el-select>
      </div>

      <!-- 按类型分组展示 -->
      <div class="error-groups">
        <template v-for="(groupErrors, errorType) in groupedErrors" :key="errorType">
          <div class="error-group" v-if="groupErrors.length > 0">
            <div class="group-header">
              <h2>
                <el-tag :type="getErrorTagType(errorType)" size="large">
                  {{ getErrorTypeLabel(errorType) }}
                </el-tag>
                <el-tag
                    :type="getSeverityTagType(errorType)"
                    size="small"
                    effect="plain"
                    style="margin-left: 8px;"
                >
                  {{ getSeverityLabel(errorType) }}
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
                  <div class="error-meta">
                    <span class="error-time">
                      <el-icon><Clock /></el-icon>
                      {{ formatDate(error.created_at) }}
                    </span>
                    <el-tag size="small" :type="error.source_role === 'llm' ? 'success' : 'primary'" effect="plain">
                      {{ getSourceRoleLabel(error.source_role) }}
                    </el-tag>
                    <el-tag size="small" type="info" effect="plain">
                      {{ getPracticeModeLabel(error) }}
                    </el-tag>
                  </div>
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

                  <div v-if="error.conversation_summary" class="section conversation-summary-section">
                    <div class="section-title">
                      <el-icon color="#8b5cf6"><ChatLineRound /></el-icon>
                      {{ $t('grammar.conversationSummary') }}
                    </div>
                    <div class="conversation-summary-text">{{ error.conversation_summary }}</div>
                  </div>

                  <div v-if="shouldShowIntendedMeaning(error) && error.llm_intended_meaning" class="section">
                    <div class="section-title">
                      <el-icon color="#409eff"><InfoFilled /></el-icon>
                      {{ $t('grammar.intendedMeaning') }}
                    </div>
                    <div class="intended-meaning-text">{{ error.llm_intended_meaning }}</div>
                  </div>

                  <div v-if="shouldShowSuggestion(error) && error.llm_suggestion" class="section">
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
    <el-dialog v-model="changeTypeDialogVisible" :title="$t('grammar.changeType')" width="400px">
      <el-form :model="changeTypeForm">
        <el-form-item :label="$t('grammar.errorTypeLabel')">
          <el-select v-model="changeTypeForm.newType" style="width: 100%">
            <el-option :label="$t('grammar.errorType1')" value="语用语言失误" />
            <el-option :label="$t('grammar.errorType2')" value="社会语用失误" />
            <el-option :label="$t('grammar.errorType3')" value="严重语用语言失误" />
            <el-option :label="$t('grammar.errorType4')" value="严重社会语用失误" />
            <el-option :label="$t('grammar.errorType5')" value="语用语言失误和社会语用失误" />
            <el-option :label="$t('grammar.errorType6')" value="无明显语用失误" />
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
  Edit,
  Search,
  ChatLineRound
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
const currentModeFilter = ref('all')
const changeTypeDialogVisible = ref(false)
const changeTypeForm = ref({
  errorId: null,
  newType: '语用语言失误'
})

// 5种错误类型
const ALL_ERROR_TYPES = [
  '语用语言失误',
  '社会语用失误',
  '严重语用语言失误',
  '严重社会语用失误',
  '语用语言失误和社会语用失误',
  '无明显语用失误'
]

// 判断是否为严重(problematic)类型
const isProblematicType = (errorType) => {
  return errorType === '严重语用语言失误' ||
      errorType === '严重社会语用失误' ||
      errorType === '语用语言失误和社会语用失误'
}

const getSourceRoleLabel = (sourceRole) => {
  if (sourceRole === 'session') {
    return locale.value === 'zh-CN' ? '会话反馈' : 'Session feedback'
  }
  if (sourceRole === 'llm') {
    return locale.value === 'zh-CN' ? 'LLM模拟学习者' : 'LLM learner'
  }
  return locale.value === 'zh-CN' ? '用户' : 'User'
}
// 获取错误类型的 tag 颜色
const getErrorTagType = (errorType) => {
  if (errorType === '无明显语用失误') {
    return 'success'
  }
  if (isProblematicType(errorType)) {
    return 'danger' // 红色
  }
  return 'warning' // 橙色
}

const getPracticeMode = (error) => {
  if (error?.session_mode === 'user_l2' || error?.session_mode === 'llm_l2') return error.session_mode
  if (error?.source_role === 'llm') return 'llm_l2'
  if (error?.source_role === 'user') return 'user_l2'
  return ''
}

const getPracticeModeLabel = (error) => {
  const mode = getPracticeMode(error)
  if (mode === 'llm_l2') return t('grammar.llmL2Mode')
  if (mode === 'user_l2') return t('grammar.userL2Mode')
  return t('grammar.unknownMode')
}

const shouldShowIntendedMeaning = (error) => getPracticeMode(error) === 'llm_l2'
const shouldShowSuggestion = (error) => getPracticeMode(error) !== 'llm_l2'

const getSeverityTagType = (errorType) => {
  if (errorType === '无明显语用失误') return 'success'
  return isProblematicType(errorType) ? 'danger' : 'warning'
}

const getSeverityLabel = (errorType) => {
  if (errorType === '无明显语用失误') return locale.value === 'zh-CN' ? '良好' : 'Good'
  return isProblematicType(errorType) ? t('grammar.severityProblematic') : t('grammar.severityImprovable')
}

// 获取错误类型的本地化标签
const getErrorTypeLabel = (errorType) => {
  const labelMap = {
    '语用语言失误': t('grammar.errorType1'),
    '社会语用失误': t('grammar.errorType2'),
    '严重语用语言失误': t('grammar.errorType3'),
    '严重社会语用失误': t('grammar.errorType4'),
    '语用语言失误和社会语用失误': t('grammar.errorType5'),
    '无明显语用失误': t('grammar.errorType6'),
  }
  return labelMap[errorType] || errorType
}

// 计算可改进总数
const improvableTotal = computed(() => {
  const byType = statistics.value.by_type || {}
  return (byType['语用语言失误'] || 0) + (byType['社会语用失误'] || 0)
})

// 计算严重问题总数
const problematicTotal = computed(() => {
  const byType = statistics.value.by_type || {}
  return (byType['严重语用语言失误'] || 0) +
      (byType['严重社会语用失误'] || 0) +
      (byType['语用语言失误和社会语用失误'] || 0)
})

// 按搜索关键词筛选
const filteredErrors = computed(() => {
  if (!searchKeyword.value) return errors.value

  const keyword = searchKeyword.value.toLowerCase()
  return errors.value.filter(e =>
      String(e.original_text || '').toLowerCase().includes(keyword) ||
      String(e.conversation_summary || '').toLowerCase().includes(keyword) ||
      String(e.llm_intended_meaning || '').toLowerCase().includes(keyword) ||
      String(e.llm_suggestion || '').toLowerCase().includes(keyword) ||
      String(e.llm_explanation || '').toLowerCase().includes(keyword) ||
      getSourceRoleLabel(e.source_role).toLowerCase().includes(keyword) ||
      getPracticeModeLabel(e).toLowerCase().includes(keyword)
  )
})

// 按错误类型分组
const groupedErrors = computed(() => {
  const groups = {}
  ALL_ERROR_TYPES.forEach(type => {
    groups[type] = []
  })

  filteredErrors.value.forEach(error => {
    if (groups[error.error_type]) {
      groups[error.error_type].push(error)
    } else {
      // 兜底: 放到语用语言失误分组
      groups['语用语言失误'].push(error)
    }
  })

  return groups
})

onMounted(async () => {
  await loadErrors()
})

const loadErrors = async (errorType = currentTypeFilter.value, sessionMode = currentModeFilter.value) => {
  try {
    const response = await api.getGrammarErrors(errorType, sessionMode)
    // 每条记录代表一次真实会话/消息事件；文本相同不等于重复记录。
    errors.value = response.errors || []
    statistics.value = response.statistics || {
      total: 0,
      by_type: {},
      today_count: 0,
      week_count: 0
    }
  } catch (error) {
    console.error('Load errors:', error)
    if (!error?.pfchatNotified) ElMessage.error(t('common.error'))
  }
}

const handleTypeChange = async (value) => {
  await loadErrors(value, currentModeFilter.value)
}

const handleModeChange = async (value) => {
  await loadErrors(currentTypeFilter.value, value)
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
    newType: error.error_type
  }
  changeTypeDialogVisible.value = true
}

const confirmChangeType = async () => {
  try {
    await api.updateGrammarErrorType(changeTypeForm.value.errorId, changeTypeForm.value.newType)
    ElMessage.success(t('grammar.updateTypeSuccess'))
    changeTypeDialogVisible.value = false
    await loadErrors(currentTypeFilter.value, currentModeFilter.value)
  } catch (error) {
    console.error('Update type error:', error)
    if (!error?.pfchatNotified) ElMessage.error(t('grammar.updateTypeFailed'))
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
    await loadErrors(currentTypeFilter.value, currentModeFilter.value)
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

    await api.clearGrammarErrors(currentTypeFilter.value, currentModeFilter.value)
    ElMessage.success(t('grammar.clearSuccess'))
    await loadErrors(currentTypeFilter.value, currentModeFilter.value)
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Clear type error:', error)
    }
  }
}

const clearAll = async () => {
  try {
    await ElMessageBox.confirm(
        t('grammar.clearAllConfirm', { count: statistics.value.total }),
        t('chat.warning'),
        {
          confirmButtonText: t('chat.confirmClear'),
          cancelButtonText: t('common.cancel'),
          type: 'warning',
          confirmButtonClass: 'el-button--danger'
        }
    )

    await api.clearGrammarErrors('all', currentModeFilter.value)
    ElMessage.success(t('grammar.clearSuccess'))
    await loadErrors(currentTypeFilter.value, currentModeFilter.value)
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
  flex-wrap: wrap;
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

.error-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
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
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section {
  padding: 10px;
  border-radius: 6px;
  background: #fafafa;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: #606266;
  margin-bottom: 8px;
}

.original-text {
  font-size: 14px;
  color: #303133;
  line-height: 1.6;
}

.suggestion-text {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
  font-size: 14px;
  color: #67c23a;
  line-height: 1.6;
}

.explanation-text {
  font-size: 14px;
  color: #606266;
  line-height: 1.6;
}

.intended-meaning-text {
  padding: 12px 15px;
  background: #ecf5ff;
  border-left: 4px solid #409eff;
  border-radius: 4px;
  color: #337ecc;
  line-height: 1.6;
  white-space: pre-wrap;
}

.conversation-summary-section {
  background: #f7f5ff;
}

.conversation-summary-text {
  padding: 12px 15px;
  border-left: 4px solid #8b5cf6;
  border-radius: 4px;
  color: #5b4b8a;
  line-height: 1.7;
  white-space: pre-wrap;
}

.deleted-notice {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #909399;
  font-size: 13px;
  padding: 8px;
  background: #f5f5f5;
  border-radius: 4px;
}
</style>

