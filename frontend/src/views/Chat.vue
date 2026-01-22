<template>
  <div class="chat-container">
    <!-- 侧边栏 -->
    <div class="sidebar">
      <div class="sidebar-header">
        <h2>{{ userStore.userInfo.username }}</h2>
        <div class="header-actions">
          <!-- 语言切换 -->
          <LocaleSwitcher />
          <el-button
              :icon="Document"
              circle
              @click="router.push('/grammar-errors')"
              :title="$t('chat.errorRecord')"
          />
          <el-button
              :icon="SwitchButton"
              circle
              @click="handleLogout"
              :title="$t('chat.logout')"
          />
        </div>
      </div>

      <div class="search-box">
        <el-input
            v-model="searchKeyword"
            :placeholder="$t('chat.searchUser')"
            prefix-icon="Search"
            clearable
        />
      </div>

      <div class="connection-status" :class="chatStore.isConnected ? 'connected' : 'disconnected'">
        <el-icon><Connection /></el-icon>
        <span>{{ chatStore.isConnected ? $t('chat.connected') : $t('chat.disconnected') }}</span>
      </div>

      <div class="user-list">
        <div
            v-for="user in filteredUsers"
            :key="user.id"
            class="user-item"
            :class="{ active: chatStore.currentUser?.id === user.id }"
            @click="chatStore.selectUser(user)"
        >
          <el-avatar :size="40">{{ user.username[0].toUpperCase() }}</el-avatar>
          <div class="user-info">
            <div class="username">{{ user.username }}</div>
            <div class="country">{{ user.country }}</div>
          </div>
        </div>

        <el-empty v-if="filteredUsers.length === 0" :description="$t('chat.noUsers')" />
      </div>
    </div>

    <!-- 聊天区域 -->
    <div class="chat-area">
      <div v-if="chatStore.currentUser" class="chat-content">
        <!-- 聊天头部 -->
        <div class="chat-header">
          <div class="chat-user-info">
            <el-avatar :size="40">{{ chatStore.currentUser.username[0].toUpperCase() }}</el-avatar>
            <div>
              <div class="username">{{ chatStore.currentUser.username }}</div>
              <div class="country">{{ chatStore.currentUser.country }}</div>
            </div>
          </div>
          <el-dropdown trigger="click">
            <el-button :icon="MoreFilled" circle />
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="clearChatHistory">
                  <el-icon><Delete /></el-icon>
                  {{ $t('chat.clearChat') }}
                </el-dropdown-item>
                <el-dropdown-item @click="deleteMyMessages">
                  <el-icon><Delete /></el-icon>
                  {{ $t('chat.deleteMyMessages') }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <!-- 消息列表 -->
        <div class="messages-container" ref="messagesContainer">
          <div
              v-for="msg in chatStore.currentMessages"
              :key="msg.id"
              class="message-wrapper"
              :class="{ 'message-sent': msg.sender_id === userStore.userInfo.id }"
          >
            <div class="message">
              <div class="message-header">
                <span class="message-time">{{ formatMessageTime(msg.created_at) }}</span>
                <el-button
                    v-if="msg.sender_id === userStore.userInfo.id || userStore.userInfo.role === 'admin'"
                    :icon="Delete"
                    size="small"
                    :type="msg.sender_id === userStore.userInfo.id ? 'warning' : 'danger'"
                    text
                    @click="deleteMessage(msg.id)"
                    class="delete-btn"
                    :title="msg.sender_id !== userStore.userInfo.id ? $t('chat.adminDelete') : $t('chat.deleteMessage')"
                />
              </div>
              <div class="message-text">{{ msg.content }}</div>
            </div>
          </div>

          <div v-if="chatStore.currentMessages.length === 0" class="empty-messages">
            <el-empty :description="$t('chat.noMessages')" />
          </div>
        </div>

        <!-- 输入框 -->
        <div class="input-area">
          <el-input
              v-model="messageInput"
              type="textarea"
              :rows="3"
              :placeholder="$t('chat.inputPlaceholder')"
              @keydown.ctrl.enter="sendMessage"
          />
          <el-button
              type="primary"
              :icon="Promotion"
              @click="sendMessage"
              :disabled="!messageInput.trim()"
              class="send-btn"
          >
            {{ $t('chat.send') }}
          </el-button>
        </div>
      </div>

      <!-- 未选择用户时的提示 -->
      <div v-else class="empty-chat">
        <el-empty :description="$t('chat.selectUser')" />
      </div>
    </div>
  </div>
  <!-- 语用失误检测对话框 -->
  <ErrorCheckDialog
      v-model="showErrorDialog"
      :error-data="errorCheckData"
      @send-original="handleSendOriginal"
      @send-edited="handleSendEdited"
      @cancel="handleCancelSend"
  />
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import ErrorCheckDialog from '@/components/ErrorCheckDialog.vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { debounce } from 'lodash-es'
import {
  SwitchButton,
  Document,
  Promotion,
  Connection,
  Delete,
  MoreFilled
} from '@element-plus/icons-vue'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { useChatStore } from '@/stores/chat'
import wsManager from '@/utils/websocket'
import LocaleSwitcher from '@/components/LocaleSwitcher.vue'

// 错误检测相关状态
const showErrorDialog = ref(false)
const errorCheckData = ref({
  original_content: '',
  suggestion: '',
  explanation: '',
  error_type: ''
})

const router = useRouter()
const userStore = useUserStore()
const chatStore = useChatStore()
const { t } = useI18n()

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
  wsManager.connect()

  // 加载用户列表
  try {
    const users = await api.getUsers()
    chatStore.setUsers(users.filter(u => u.id !== userStore.userInfo.id))
  } catch (error) {
    console.error('Load users error:', error)
  }

  // 监听新消息
  wsManager.onMessage((data) => {
    if (data.type === 'new_message') {
      chatStore.addMessage(data.message)
      scrollToBottom()
    } else if (data.type === 'grammar_error') {
      // 处理语法错误
      console.log('Grammar error:', data.error)
    }
  })
})

onUnmounted(() => {
  wsManager.disconnect()
})

// 监听当前用户变化,加载消息
watch(() => chatStore.currentUser, async (newUser) => {
  if (newUser) {
    try {
      const messages = await api.getMessages(newUser.id)
      chatStore.setCurrentMessages(messages)
      await nextTick()
      scrollToBottom()
    } catch (error) {
      console.error('Load messages error:', error)
    }
  }
})

const sendMessage = debounce(async () => {
  if (!messageInput.value.trim() || !chatStore.currentUser) return

  const content = messageInput.value.trim()
  const receiverId = chatStore.currentUser.id

  try {
    // 调用发送前检测 API
    const checkResult = await api.checkMessageBeforeSend({
      receiver_id: receiverId,
      content: content
    })

    // 如果检测到语用失误，显示编辑对话框
    if (checkResult.has_error) {
      errorCheckData.value = {
        original_content: content,
        suggestion: checkResult.suggestion || '',
        explanation: checkResult.explanation || '',
        error_type: checkResult.error_type || '语言语用失误',
        error_record_id: checkResult.error_record_id || 0  // 🔧 保存记录ID
      }
      showErrorDialog.value = true
      // 不清空输入框，保留原文
    } else {
      // 没有错误，直接发送
      await sendMessageDirectly(content, receiverId, 0)
      messageInput.value = '' // 发送成功后清空
    }
  } catch (error) {
    console.error('Check message error:', error)
    ElMessage.error(t('chat.checkFailed'))
  }
}, 1000, { leading: true, trailing: false })

// 直接发送消息（带错误记录ID）
const sendMessageDirectly = async (content, receiverId, errorRecordId = 0) => {
  try {
    const message = {
      receiver_id: receiverId,
      content: content,
      error_record_id: errorRecordId  // 🔧 传递错误记录ID
    }
    await api.sendMessage(message)
    ElMessage.success(t('chat.sendSuccess'))
  } catch (error) {
    console.error('Send message error:', error)
    ElMessage.error(t('chat.sendFailed'))
  }
}

// 处理发送原始消息（带错误记录ID）
const handleSendOriginal = async (content) => {
  if (!chatStore.currentUser) return
  // 🔧 传递错误记录ID
  await sendMessageDirectly(content, chatStore.currentUser.id, errorCheckData.value.error_record_id)
  messageInput.value = '' // 发送成功后清空
}

// 处理发送编辑后的消息
const handleSendEdited = async (content) => {
  if (!chatStore.currentUser) return
  // 🔧 用户修改了内容，不传递错误记录ID（因为内容已改变）
  await sendMessageDirectly(content, chatStore.currentUser.id, 0)
  messageInput.value = '' // 发送成功后清空
}

// 处理取消发送
const handleCancelSend = () => {
  ElMessage.info(t('chat.sendCancelled'))
  // 错误已经保存在数据库中（message_id 为 0）
  // 用户可以在错误记录页面查看所有检测到的错误
}

// 保存错误记录并发送消息
const saveErrorAndSend = async (content, receiverId, checkResult) => {
  // 先发送消息
  const message = {
    receiver_id: receiverId,
    content: content
  }
  const sentMessage = await api.sendMessage(message)

  // 保存语用错误记录到数据库（通过后端自动完成）
  // 注意：我们需要在后端添加逻辑来保存用户确认发送的错误消息

  ElMessage.success(t('chat.sendSuccess'))
}

const deleteMessage = async (messageId) => {
  try {
    await ElMessageBox.confirm(
        t('chat.deleteConfirm'),
        t('chat.warning'),
        {
          confirmButtonText: t('chat.confirmDelete'),
          cancelButtonText: t('common.cancel'),
          type: 'warning'
        }
    )

    await api.deleteMessage(messageId)
    chatStore.removeMessage(messageId)
    ElMessage.success(t('chat.messageDeleted'))
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Delete message error:', error)
    }
  }
}

const clearChatHistory = async () => {
  if (!chatStore.currentUser) return

  try {
    await ElMessageBox.confirm(
        t('chat.clearConfirm', { username: chatStore.currentUser.username }),
        t('chat.warning'),
        {
          confirmButtonText: t('chat.confirmClear'),
          cancelButtonText: t('common.cancel'),
          type: 'warning',
          confirmButtonClass: 'el-button--danger'
        }
    )

    await api.clearChatHistory(chatStore.currentUser.id)
    chatStore.setCurrentMessages([])
    ElMessage.success(t('chat.chatCleared'))
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Clear chat error:', error)
    }
  }
}

const deleteMyMessages = async () => {
  if (!chatStore.currentUser) return

  try {
    await ElMessageBox.confirm(
        t('chat.deleteMyConfirm', { username: chatStore.currentUser.username }),
        t('chat.warning'),
        {
          confirmButtonText: t('chat.confirmDelete'),
          cancelButtonText: t('common.cancel'),
          type: 'warning'
        }
    )

    const result = await api.deleteMyMessages(chatStore.currentUser.id)

    // 重新加载消息列表
    const messages = await api.getMessages(chatStore.currentUser.id)
    chatStore.setCurrentMessages(messages)

    ElMessage.success(t('chat.messagesDeleted', { count: result.deleted_count }))
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Delete my messages error:', error)
    }
  }
}

const handleLogout = async () => {
  try {
    await ElMessageBox.confirm(
        t('chat.logoutConfirm'),
        t('chat.hint'),
        {
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel'),
          type: 'warning'
        }
    )

    userStore.logout()
    wsManager.disconnect()
    ElMessage.success(t('chat.logoutSuccess'))
    router.push('/login')
  } catch (error) {
    // 用户取消
  }
}

const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const formatMessageTime = (timestamp) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now - date

  // 1分钟内
  if (diff < 60000) {
    return t('chat.justNow')
  }

  // 1小时内
  if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000)
    return t('chat.minutesAgo', { n: minutes })
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
</script>

<style scoped>
.chat-container {
  display: flex;
  height: 100vh;
  background: #f5f5f5;
}

.sidebar {
  width: 300px;
  background: white;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 20px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.sidebar-header h2 {
  margin: 0;
  font-size: 18px;
  color: #303133;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.search-box {
  padding: 15px;
}

.connection-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 15px;
  font-size: 14px;
  background: #f5f7fa;
  border-top: 1px solid #e4e7ed;
  border-bottom: 1px solid #e4e7ed;
}

.connection-status.connected {
  color: #67c23a;
}

.connection-status.disconnected {
  color: #f56c6c;
}

.user-list {
  flex: 1;
  overflow-y: auto;
}

.user-item {
  display: flex;
  align-items: center;
  gap: 15px;
  padding: 15px;
  cursor: pointer;
  transition: background 0.2s;
}

.user-item:hover {
  background: #f5f7fa;
}

.user-item.active {
  background: #ecf5ff;
}

.user-info {
  flex: 1;
}

.username {
  font-size: 15px;
  color: #303133;
  font-weight: 500;
}

.country {
  font-size: 13px;
  color: #909399;
  margin-top: 4px;
}

.chat-area {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.chat-content {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.chat-header {
  background: white;
  padding: 15px 20px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chat-user-info {
  display: flex;
  align-items: center;
  gap: 15px;
}

.messages-container {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  background: #f5f5f5;
}

.message-wrapper {
  margin-bottom: 20px;
  display: flex;
}

.message-wrapper.message-sent {
  justify-content: flex-end;
}

.message {
  max-width: 60%;
  background: white;
  border-radius: 8px;
  padding: 12px 16px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.message-sent .message {
  background: #409eff;
  color: white;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 12px;
  opacity: 0.7;
}

.message-text {
  word-break: break-word;
  line-height: 1.6;
}

.delete-btn {
  padding: 4px;
  margin-left: 8px;
}

.empty-messages,
.empty-chat {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.input-area {
  background: white;
  padding: 20px;
  border-top: 1px solid #e4e7ed;
  display: flex;
  gap: 15px;
  align-items: flex-end;
}

.input-area :deep(.el-textarea) {
  flex: 1;
}

.send-btn {
  height: 90px;
}
</style>