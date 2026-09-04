<template>
  <div class="admin-container">
    <div class="admin-header">
      <div class="header-left">
        <el-button :icon="ArrowLeft" @click="$router.back()">返回</el-button>
        <h1>管理员控制台</h1>
      </div>
      <div class="header-right">
        <el-tag type="danger">管理员</el-tag>
        <span class="admin-name">{{ userStore.userInfo?.username }}</span>
      </div>
    </div>

    <div class="admin-content">
      <!-- 统计卡片 -->
      <div class="stats-grid">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon users">
              <el-icon><User /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">{{ stats.totalUsers }}</div>
              <div class="stat-label">总用户数</div>
            </div>
          </div>
        </el-card>

        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon admins">
              <el-icon><Avatar /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">{{ stats.adminUsers }}</div>
              <div class="stat-label">管理员</div>
            </div>
          </div>
        </el-card>

        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon messages">
              <el-icon><ChatDotRound /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">{{ stats.totalMessages }}</div>
              <div class="stat-label">总消息数</div>
            </div>
          </div>
        </el-card>

        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon errors">
              <el-icon><Warning /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">{{ stats.totalGrammarErrors }}</div>
              <div class="stat-label">语法错误</div>
            </div>
          </div>
        </el-card>
      </div>

      <!-- 用户管理 -->
      <el-card class="user-management">
        <template #header>
          <div class="card-header">
            <span>用户管理</span>
            <div class="header-actions">
              <el-input
                  v-model="searchKeyword"
                  placeholder="搜索用户名或国家"
                  :prefix-icon="Search"
                  clearable
                  style="width: 250px"
                  @input="searchUsers"
              />
              <el-button :icon="Refresh" @click="loadUsers">刷新</el-button>
            </div>
          </div>
        </template>

        <el-table
            :data="users"
            v-loading="loading"
            style="width: 100%"
            empty-text="暂无用户数据"
        >
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="username" label="用户名" width="150" />
          <el-table-column prop="country" label="国家" width="120" />
          <el-table-column prop="role" label="角色" width="100">
            <template #default="scope">
              <el-tag
                  :type="scope.row.role === 'admin' ? 'danger' : 'primary'"
                  size="small"
              >
                {{ scope.row.role === 'admin' ? '管理员' : '用户' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="注册时间" width="180">
            <template #default="scope">
              {{ formatDate(scope.row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200">
            <template #default="scope">
              <div class="action-buttons">
                <el-button
                    v-if="scope.row.role === 'user'"
                    type="success"
                    size="small"
                    @click="promoteToAdmin(scope.row)"
                    :disabled="scope.row.id === userStore.userInfo?.id"
                >
                  提升为管理员
                </el-button>
                <el-button
                    v-if="scope.row.role === 'admin'"
                    type="warning"
                    size="small"
                    @click="demoteToUser(scope.row)"
                    :disabled="scope.row.id === userStore.userInfo?.id"
                >
                  降级为用户
                </el-button>
                <el-button
                    type="danger"
                    size="small"
                    @click="deleteUser(scope.row)"
                    :disabled="scope.row.id === userStore.userInfo?.id"
                >
                  删除
                </el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <!-- 分页 -->
        <div class="pagination-container">
          <el-pagination
              v-model:current-page="currentPage"
              v-model:page-size="pageSize"
              :page-sizes="[10, 20, 50, 100]"
              :total="total"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="handleSizeChange"
              @current-change="handleCurrentChange"
          />
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft,
  User,
  Avatar,
  ChatDotRound,
  Warning,
  Refresh,
  Search
} from '@element-plus/icons-vue'
import api from '@/api'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()

const loading = ref(false)
const searchKeyword = ref('')
const allUsers = ref([])
const currentPage = ref(1)
const pageSize = ref(20)

const stats = ref({
  totalUsers: 0,
  adminUsers: 0,
  regularUsers: 0,
  totalMessages: 0,
  totalGrammarErrors: 0
})

const filteredUsers = computed(() => {
  const kw = searchKeyword.value.toLowerCase()
  if (!kw) return allUsers.value
  return allUsers.value.filter(u =>
    u.username.toLowerCase().includes(kw) ||
    (u.country && u.country.toLowerCase().includes(kw))
  )
})

const total = computed(() => filteredUsers.value.length)

const users = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredUsers.value.slice(start, start + pageSize.value)
})

onMounted(async () => {
  await loadStats()
  await loadUsers()
})

const loadStats = async () => {
  try {
    const result = await api.getUserStats()
    stats.value = {
      totalUsers: result.total_users ?? 0,
      adminUsers: result.admin_users ?? 0,
      regularUsers: result.regular_users ?? 0,
      totalMessages: result.total_messages ?? 0,
      totalGrammarErrors: result.total_grammar_errors ?? 0
    }
  } catch (error) {
    console.error('Load stats error:', error)
  }
}

const loadUsers = async () => {
  loading.value = true
  try {
    const result = await api.getAllUsersForAdmin()
    allUsers.value = result.users
  } catch (error) {
    console.error('Load users error:', error)
    if (!error?.pfchatNotified) ElMessage.error('加载用户列表失败')
  } finally {
    loading.value = false
  }
}

const searchUsers = () => {
  currentPage.value = 1
}

const handleSizeChange = (newSize) => {
  pageSize.value = newSize
  currentPage.value = 1
}

const handleCurrentChange = (newPage) => {
  currentPage.value = newPage
}

const promoteToAdmin = async (user) => {
  try {
    await ElMessageBox.confirm(
        `确定要将用户 "${user.username}" 提升为管理员吗？`,
        '提升管理员',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
    )

    await api.updateUserRole(user.id, 'admin')
    ElMessage.success('用户已提升为管理员')
    await loadUsers()
    await loadStats()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Promote user error:', error)
    }
  }
}

const demoteToUser = async (user) => {
  try {
    await ElMessageBox.confirm(
        `确定要将管理员 "${user.username}" 降级为普通用户吗？`,
        '降级用户',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
    )

    await api.updateUserRole(user.id, 'user')
    ElMessage.success('管理员已降级为普通用户')
    await loadUsers()
    await loadStats()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Demote user error:', error)
    }
  }
}

const deleteUser = async (user) => {
  try {
    await ElMessageBox.confirm(
        `确定要删除用户 "${user.username}" 吗？\n\n⚠️ 此操作将同时删除该用户的所有消息和数据，且不可恢复！`,
        '删除用户',
        {
          confirmButtonText: '确定删除',
          cancelButtonText: '取消',
          type: 'error',
          confirmButtonClass: 'el-button--danger'
        }
    )

    await api.deleteUser(user.id)
    ElMessage.success(`用户 "${user.username}" 已删除`)
    await loadUsers()
    await loadStats()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Delete user error:', error)
    }
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
.admin-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f5f5f5;
}

.admin-header {
  background: white;
  padding: 20px 30px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.header-left h1 {
  margin: 0;
  font-size: 24px;
  color: #303133;
  font-weight: 600;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.admin-name {
  font-weight: 500;
  color: #606266;
}

.admin-content {
  flex: 1;
  padding: 30px;
  overflow-y: auto;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
  margin-bottom: 30px;
}

.stat-card {
  border-radius: 12px;
  overflow: hidden;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 15px;
  padding: 10px;
}

.stat-icon {
  width: 50px;
  height: 50px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: white;
}

.stat-icon.users {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.stat-icon.admins {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.stat-icon.messages {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.stat-icon.errors {
  background: linear-gradient(135deg, #ffecd2 0%, #fcb69f 100%);
  color: #e6a23c;
}

.stat-info {
  flex: 1;
}

.stat-number {
  font-size: 28px;
  font-weight: 700;
  color: #303133;
  line-height: 1;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 5px;
}

.user-management {
  border-radius: 12px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
  font-size: 16px;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.action-buttons {
  display: flex;
  gap: 5px;
  flex-wrap: wrap;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #ebeef5;
}

:deep(.el-table) {
  font-size: 14px;
}

:deep(.el-table th) {
  background: #fafafa;
  font-weight: 600;
}

:deep(.el-button--small) {
  padding: 5px 8px;
  font-size: 12px;
}
</style>
