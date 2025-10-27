<template>
  <div class="chat-container">
    <!-- 左侧用户列表 -->
    <div class="user-list">
      <div class="user-list-header">
        <div class="current-user">
          <el-avatar :size="40">{{ userStore.userInfo?.username?.[0] }}</el-avatar>
          <div class="user-info">
            <div class="username">{{ userStore.userInfo?.username }}</div>
            <div class="country">{{ userStore.userInfo?.country }}</div>
          </div>
        </div>
        <div class="header-actions">
          <el-button
              :icon="Document"
              circle
              size="small"
              @click="$router.push('/grammar-errors')"
              title="错误记录"
          />
          <el-button
              :icon="SwitchButton"
              circle
              size="small"
              @click="handleLogout"
              title="退出登录"
          />
        </div>
      </div>

      <div class="connection-status" :class="{ connected: chatStore.connected }">
        <el-icon><Connection /></el-icon>
        {{ chatStore.connected ? '已连接' : '未连接' }}
      </div>

      <el-input
          v-model="searchKeyword"
          placeholder="搜索用户"
          prefix-icon="Search"
          class="search-input"
      />

      <div class="users-container">
        <div
            v-for="user in filteredUsers"
            :key="user.id"
            class="user-item"
            :class="{ active: chatStore.currentChatUser?.id === user.id }"
            @click="selectUser(user)"
        >
          <el-avatar :size="45">{{ user.username[0] }}</el-avatar>
          <div class="user-detail">
            <div class="username">{{ user.username }}</div>
            <div class="country-tag">{{ user.country }}</div>
          </div>
        </div>

        <el-empty v-if="filteredUsers.length === 0" description="暂无用户" />
      </div>
    </div>

    <!-- 右侧聊天区域 -->
    <div class="chat-area">
      <div v-if="chatStore.currentChatUser" class="chat-content">
        <!-- 聊天头部 -->
        <div class="chat-header">
          <div class="chat-user-left">
            <el-avatar :size="40">{{ chatStore.currentChatUser.username[0] }}</el-avatar>
            <div class="chat-user-info">
              <div class="username">{{ chatStore.currentChatUser.username }}</div>
              <div class="country">{{ chatStore.currentChatUser.country }}</div>
            </div>
          </div>
          <div class="chat-actions">
            <el-dropdown @command="handleChatAction">
              <el-button :icon="MoreFilled" circle size="small" />
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="clear">清空聊天记录</el-dropdown-item>
                  <el-dropdown-item command="deleteMyMessages">删除我的消息</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>

        <!-- 消息列表 -->
        <div class="messages-container" ref="messagesContainer">
          <div
              v-for="msg in chatStore.currentMessages"
              :key="msg.id"
              class="message-item"
              :class="{ 'is-mine': msg.sender_id === userStore.userInfo.id }"
          >
            <el-avatar :size="35">{{ msg.sender.username[0] }}</el-avatar>
            <div class="message-content">
              <div class="message-header">
                <span class="sender-name">{{ msg.sender.username }}</span>
                <span class="message-time">{{ formatTime(msg.sent_at) }}</span>
                <el-button
                    v-if="msg.sender_id === userStore.userInfo.id"
                    :icon="Delete"
                    size="small"
                    type="danger"
                    text
                    @click="deleteMessage(msg.id)"
                    class="delete-btn"
                    title="删除消息"
                />
              </div>
              <div class="message-text">{{ msg.content }}</div>
            </div>
          </div>

          <div v-if="chatStore.currentMessages.length === 0" class="empty-messages">
            <el-empty description="暂无消息，开始聊天吧" />
          </div>
        </div>

        <!-- 输入框 -->
        <div class="input-area">
          <el-input
              v-model="messageInput"
              type="textarea"
              :rows="3"
              placeholder="输入消息... (Ctrl+Enter 发送)"
              @keydown.ctrl.enter="sendMessage"
          />
          <el-button
              type="primary"
              :icon="Promotion"
              @click="sendMessage"
              :disabled="!messageInput.trim()"
              class="send-btn"
          >
            发送
          </el-button>
        </div>
      </div>

      <!-- 未选择用户时的提示 -->
      <div v-else class="empty-chat">
        <el-empty description="请选择一个用户开始聊天" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { SwitchButton, Document, Promotion, Connection, Delete, MoreFilled } from '@element-plus/icons-vue'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { useChatStore } from '@/stores/chat'
import wsManager from '@/utils/websocket'

const router = useRouter()
const userStore = useUserStore()
const chatStore = useChatStore()

const searchKeyword = ref('')
const messageInput = ref('')
const messagesContainer = ref(null)

const filteredUsers = computed(() => {
  if (!searchKeyword.value) return chatStore.users
  return chatStore.users.filter(user =>
      user.username.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

onMounted(async () => {
  // 连接 WebSocket
  wsManager.connect(userStore.token)

  // 加载用户列表
  await loadUsers()
})

onUnmounted(() => {
  wsManager.disconnect()
})

const loadUsers = async () => {
  try {
    const users = await api.getUsers()
    chatStore.setUsers(users)
  } catch (error) {
    console.error('Load users error:', error)
  }
}

const selectUser = async (user) => {
  chatStore.setCurrentChatUser(user)

  // 加载聊天记录
  try {
    const messages = await api.getMessages(user.id)
    chatStore.setMessages(user.id, messages)
    await nextTick()
    scrollToBottom()
  } catch (error) {
    console.error('Load messages error:', error)
  }
}

const sendMessage = async () => {
  if (!messageInput.value.trim()) return

  const content = messageInput.value.trim()
  messageInput.value = ''

  try {
    // 通过 WebSocket 发送
    wsManager.sendMessage({
      receiver_id: chatStore.currentChatUser.id,
      content: content
    })

    // 也通过 HTTP 发送（确保消息保存）
    const msg = await api.sendMessage({
      receiver_id: chatStore.currentChatUser.id,
      content: content
    })

    // 添加到本地消息列表
    chatStore.addMessage(chatStore.currentChatUser.id, msg)

    await nextTick()
    scrollToBottom()
  } catch (error) {
    console.error('Send message error:', error)
    ElMessage.error('发送失败')
  }
}

const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const formatTime = (time) => {
  const date = new Date(time)
  const now = new Date()
  const diff = now - date

  // 小于1分钟
  if (diff < 60000) {
    return '刚刚'
  }

  // 小于1小时
  if (diff < 3600000) {
    return `${Math.floor(diff / 60000)}分钟前`
  }

  // 今天
  if (date.toDateString() === now.toDateString()) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }

  // 其他
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const handleLogout = () => {
  wsManager.disconnect()
  userStore.logout()
  router.push('/login')
  ElMessage.success('已退出登录')
}

// 删除单条消息
const deleteMessage = async (messageId) => {
  try {
    await ElMessageBox.confirm('确定要删除这条消息吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })

    await api.deleteMessage(messageId)

    // 从本地消息列表中移除
    const messages = chatStore.messages[chatStore.currentChatUser.id]
    const index = messages.findIndex(m => m.id === messageId)
    if (index > -1) {
      messages.splice(index, 1)
    }

    ElMessage.success('消息已删除')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Delete message error:', error)
    }
  }
}

// 处理聊天操作（清空/删除）
const handleChatAction = async (command) => {
  if (!chatStore.currentChatUser) return

  try {
    if (command === 'clear') {
      await ElMessageBox.confirm(
          `确定要清空与 ${chatStore.currentChatUser.username} 的所有聊天记录吗？此操作不可恢复！`,
          '警告',
          {
            confirmButtonText: '确定清空',
            cancelButtonText: '取消',
            type: 'warning',
            confirmButtonClass: 'el-button--danger'
          }
      )

      await api.clearChatHistory(chatStore.currentChatUser.id)
      chatStore.setMessages(chatStore.currentChatUser.id, [])
      ElMessage.success('聊天记录已清空')

    } else if (command === 'deleteMyMessages') {
      await ElMessageBox.confirm(
          `确定要删除我发送给 ${chatStore.currentChatUser.username} 的所有消息吗？`,
          '提示',
          {
            confirmButtonText: '确定删除',
            cancelButtonText: '取消',
            type: 'warning'
          }
      )

      const result = await api.deleteMyMessages(chatStore.currentChatUser.id)

      // 从本地消息列表中移除我发送的消息
      const messages = chatStore.messages[chatStore.currentChatUser.id]
      const filtered = messages.filter(m => m.sender_id !== userStore.userInfo.id)
      chatStore.setMessages(chatStore.currentChatUser.id, filtered)

      ElMessage.success(`已删除 ${result.deleted_count} 条消息`)
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Chat action error:', error)
    }
  }
}

// 监听当前消息列表变化，自动滚动到底部
watch(() => chatStore.currentMessages.length, () => {
  nextTick(() => scrollToBottom())
})
</script>

<style scoped>
.chat-container {
  display: flex;
  height: 100vh;
  background: #f5f5f5;
}

.user-list {
  width: 300px;
  background: white;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
}

.user-list-header {
  padding: 20px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.current-user {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-info {
  display: flex;
  flex-direction: column;
}

.username {
  font-weight: 600;
  font-size: 16px;
  color: #303133;
}

.country {
  font-size: 12px;
  color: #909399;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.connection-status {
  padding: 8px 20px;
  background: #f56c6c;
  color: white;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 5px;
}

.connection-status.connected {
  background: #67c23a;
}

.search-input {
  margin: 15px 20px;
}

.users-container {
  flex: 1;
  overflow-y: auto;
}

.user-item {
  padding: 15px 20px;
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  transition: background 0.2s;
}

.user-item:hover {
  background: #f5f7fa;
}

.user-item.active {
  background: #ecf5ff;
}

.user-detail {
  flex: 1;
}

.country-tag {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.chat-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: white;
}

.chat-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.chat-header {
  padding: 20px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chat-user-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.chat-user-info {
  display: flex;
  flex-direction: column;
}

.messages-container {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  background: #f5f5f5;
}

.message-item {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

.message-item.is-mine {
  flex-direction: row-reverse;
}

.message-content {
  max-width: 60%;
}

.message-item.is-mine .message-content {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.message-header {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 5px;
  font-size: 12px;
  color: #909399;
}

.sender-name {
  font-weight: 600;
}

.delete-btn {
  margin-left: auto;
  opacity: 0;
  transition: opacity 0.2s;
}

.message-item:hover .delete-btn {
  opacity: 1;
}

.message-text {
  padding: 10px 15px;
  background: white;
  border-radius: 8px;
  word-break: break-word;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.message-item.is-mine .message-text {
  background: #409eff;
  color: white;
}

.empty-messages {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
}

.input-area {
  padding: 20px;
  border-top: 1px solid #e4e7ed;
  display: flex;
  gap: 10px;
  background: white;
}

.send-btn {
  align-self: flex-end;
}

.empty-chat {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
}
</style>