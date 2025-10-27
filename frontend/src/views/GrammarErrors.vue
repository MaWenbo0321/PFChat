<template>
  <div class="grammar-errors-container">
    <div class="header">
      <div class="header-left">
        <el-button :icon="ArrowLeft" @click="$router.back()">返回</el-button>
        <h1>语法错误记录</h1>
      </div>
      <el-button
          type="danger"
          :icon="Delete"
          @click="clearAll"
          :disabled="errors.length === 0"
      >
        清空全部
      </el-button>
    </div>

    <div class="content">
      <div class="stats">
        <el-statistic title="总错误数" :value="errors.length" />
        <el-statistic title="本周错误" :value="weekErrors" />
        <el-statistic title="今日错误" :value="todayErrors" />
      </div>

      <div class="filters">
        <el-input
            v-model="searchKeyword"
            placeholder="搜索错误内容"
            prefix-icon="Search"
            clearable
            style="width: 300px"
        />
        <el-select v-model="timeFilter" placeholder="时间筛选" style="width: 150px">
          <el-option label="全部" value="all" />
          <el-option label="今天" value="today" />
          <el-option label="本周" value="week" />
          <el-option label="本月" value="month" />
        </el-select>
      </div>

      <div class="error-list">
        <el-empty v-if="filteredErrors.length === 0" description="暂无错误记录" />

        <el-card
            v-for="error in filteredErrors"
            :key="error.id"
            class="error-card"
            shadow="hover"
        >
          <div class="error-header">
            <span class="error-time">
              <el-icon><Clock /></el-icon>
              {{ formatDate(error.created_at) }}
            </span>
            <el-button
                type="danger"
                size="small"
                :icon="Delete"
                @click="deleteError(error.id)"
                text
            >
              删除
            </el-button>
          </div>

          <div class="error-content">
            <div class="section">
              <div class="section-title">
                <el-icon color="#f56c6c"><WarningFilled /></el-icon>
                原始文本
              </div>
              <div class="original-text">{{ error.original_text }}</div>
            </div>

            <el-divider />

            <div class="section">
              <div class="section-title">
                <el-icon color="#67c23a"><CircleCheckFilled /></el-icon>
                建议修改
              </div>
              <div class="suggestion-text">
                {{ error.llm_suggestion }}
                <el-button
                    :icon="CopyDocument"
                    size="small"
                    @click="copyText(error.llm_suggestion)"
                    text
                >
                  复制
                </el-button>
              </div>
            </div>

            <el-divider />

            <div class="section">
              <div class="section-title">
                <el-icon color="#409eff"><InfoFilled /></el-icon>
                错误说明
              </div>
              <div class="explanation-text">{{ error.llm_explanation }}</div>
            </div>
          </div>
        </el-card>
      </div>

      <!-- 分页 -->
      <div class="pagination" v-if="filteredErrors.length > pageSize">
        <el-pagination
            v-model:current-page="currentPage"
            :page-size="pageSize"
            :total="filteredErrors.length"
            layout="prev, pager, next, jumper, total"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft,
  Delete,
  Clock,
  WarningFilled,
  CircleCheckFilled,
  InfoFilled,
  CopyDocument
} from '@element-plus/icons-vue'
import api from '@/api'

const errors = ref([])
const searchKeyword = ref('')
const timeFilter = ref('all')
const currentPage = ref(1)
const pageSize = ref(10)

const todayErrors = computed(() => {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return errors.value.filter(e => new Date(e.created_at) >= today).length
})

const weekErrors = computed(() => {
  const weekAgo = new Date()
  weekAgo.setDate(weekAgo.getDate() - 7)
  return errors.value.filter(e => new Date(e.created_at) >= weekAgo).length
})

const filteredErrors = computed(() => {
  let filtered = errors.value

  // 时间筛选
  if (timeFilter.value !== 'all') {
    const now = new Date()
    if (timeFilter.value === 'today') {
      now.setHours(0, 0, 0, 0)
      filtered = filtered.filter(e => new Date(e.created_at) >= now)
    } else if (timeFilter.value === 'week') {
      now.setDate(now.getDate() - 7)
      filtered = filtered.filter(e => new Date(e.created_at) >= now)
    } else if (timeFilter.value === 'month') {
      now.setMonth(now.getMonth() - 1)
      filtered = filtered.filter(e => new Date(e.created_at) >= now)
    }
  }

  // 关键词搜索
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    filtered = filtered.filter(e =>
        e.original_text.toLowerCase().includes(keyword) ||
        e.llm_suggestion.toLowerCase().includes(keyword) ||
        e.llm_explanation.toLowerCase().includes(keyword)
    )
  }

  return filtered
})

onMounted(async () => {
  await loadErrors()
})

const loadErrors = async () => {
  try {
    errors.value = await api.getGrammarErrors()
  } catch (error) {
    console.error('Load errors:', error)
  }
}

const deleteError = async (id) => {
  try {
    await ElMessageBox.confirm('确定要删除这条记录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })

    await api.deleteGrammarError(id)
    errors.value = errors.value.filter(e => e.id !== id)
    ElMessage.success('删除成功')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Delete error:', error)
    }
  }
}

const clearAll = async () => {
  try {
    await ElMessageBox.confirm(
        `确定要清空全部 ${errors.value.length} 条记录吗？此操作不可恢复！`,
        '警告',
        {
          confirmButtonText: '确定清空',
          cancelButtonText: '取消',
          type: 'warning',
          confirmButtonClass: 'el-button--danger'
        }
    )

    // 批量删除
    const deletePromises = errors.value.map(e => api.deleteGrammarError(e.id))
    await Promise.all(deletePromises)

    errors.value = []
    ElMessage.success('已清空所有记录')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Clear all error:', error)
    }
  }
}

const copyText = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败')
  }
}

const formatDate = (dateString) => {
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
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

.content {
  flex: 1;
  padding: 30px;
  overflow-y: auto;
}

.stats {
  display: flex;
  gap: 30px;
  margin-bottom: 30px;
}

.filters {
  display: flex;
  gap: 15px;
  margin-bottom: 20px;
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

.pagination {
  display: flex;
  justify-content: center;
  margin-top: 30px;
}
</style>