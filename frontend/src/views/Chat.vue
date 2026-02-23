<template>
  <div class="chat-container">
    <!-- 左侧: 用户列表 -->
    <div class="sidebar">
      <div class="sidebar-header">
        <h2>{{ userStore.userInfo.username }}</h2>
        <div class="header-actions">
          <LocaleSwitcher />
          <el-button :icon="Document" circle @click="router.push('/grammar-errors')" :title="$t('chat.errorRecord')" />
          <el-button :icon="SwitchButton" circle @click="handleLogout" :title="$t('chat.logout')" />
        </div>
      </div>

      <div class="search-box">
        <el-input v-model="searchKeyword" :placeholder="$t('chat.searchUser')" prefix-icon="Search" clearable />
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

    <!-- 中间: 聊天区域 -->
    <div class="chat-area">
      <div v-if="chatStore.currentUser" class="chat-content">
        <!-- 聊天头部 -->
        <div class="chat-header">
          <div class="chat-user-info">
            <el-avatar :size="36">{{ chatStore.currentUser.username[0].toUpperCase() }}</el-avatar>
            <div>
              <div class="username">{{ chatStore.currentUser.username }}</div>
              <div class="country">{{ chatStore.currentUser.country }}</div>
            </div>
          </div>
          <el-dropdown trigger="click">
            <el-button :icon="MoreFilled" circle size="small" />
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
                <div class="message-actions">
                  <!-- 接收方看到的语用失误小标识 -->
                  <el-tooltip
                      v-if="msg.sender_id !== userStore.userInfo.id && messageErrors[msg.id]"
                      effect="light"
                      placement="top"
                      :width="320"
                      trigger="click"
                      popper-class="error-tooltip-popper"
                  >
                    <template #content>
                      <div class="error-tooltip-content">
                        <div class="error-tooltip-header">
                          <el-tag :type="messageErrors[msg.id].error_type === '语言语用失误' ? 'warning' : 'danger'" size="small">
                            {{ messageErrors[msg.id].error_type === '语言语用失误' ? $t('chat.pragmalinguisticError') : $t('chat.sociopragmaticError') }}
                          </el-tag>
                        </div>
                        <div class="error-tooltip-section" v-if="messageErrors[msg.id].suggestion">
                          <div class="error-tooltip-label">{{ $t('chat.suggestion') }}</div>
                          <div class="error-tooltip-text suggestion">{{ messageErrors[msg.id].suggestion }}</div>
                        </div>
                        <div class="error-tooltip-section" v-if="messageErrors[msg.id].explanation">
                          <div class="error-tooltip-label">{{ $t('chat.explanation') }}</div>
                          <div class="error-tooltip-text explanation">{{ messageErrors[msg.id].explanation }}</div>
                        </div>
                      </div>
                    </template>
                    <span class="error-indicator" :class="messageErrors[msg.id].error_type === '语言语用失误' ? 'pragma' : 'socio'">
                      <el-icon :size="14"><WarningFilled /></el-icon>
                    </span>
                  </el-tooltip>

                  <el-button
                      v-if="msg.sender_id === userStore.userInfo.id || userStore.userInfo.role === 'admin'"
                      :icon="Delete"
                      size="small"
                      :type="msg.sender_id === userStore.userInfo.id ? 'warning' : 'danger'"
                      text
                      @click="deleteMessage(msg.id)"
                      class="delete-btn"
                  />
                </div>
              </div>
              <div class="message-text">{{ msg.content }}</div>
            </div>
          </div>

          <div v-if="chatStore.currentMessages.length === 0" class="empty-messages">
            <el-empty :description="$t('chat.noMessages')" />
          </div>
        </div>

        <!-- Grammarly 风格输入区域 -->
        <div class="input-area">
          <div class="input-wrapper">
            <div class="input-editor-container">
              <!-- 下划线渲染层 -->
              <div class="underline-layer" ref="underlineLayer" v-html="renderedUnderlineHtml"></div>
              <!-- 文本输入 -->
              <textarea
                  ref="textareaRef"
                  v-model="messageInput"
                  class="input-editor"
                  :placeholder="$t('chat.inputPlaceholder')"
                  @keydown.ctrl.enter="sendMessage"
                  @input="handleInputChange"
                  @scroll="syncScroll"
              ></textarea>
            </div>
            <div class="input-status-bar">
              <div class="status-left">
                <span v-if="isChecking" class="checking-status">
                  <el-icon class="is-loading"><Loading /></el-icon>
                  {{ $t('chat.checking') }}
                </span>
                <span v-else-if="inputErrors.length > 0" class="error-count">
                  <el-icon><WarningFilled /></el-icon>
                  {{ $t('chat.errorsFound', { count: inputErrors.length }) }}
                </span>
                <span v-else-if="messageInput.trim() && lastCheckTime" class="no-error">
                  <el-icon><CircleCheckFilled /></el-icon>
                  {{ $t('chat.noErrors') }}
                </span>
              </div>
              <div class="status-right">
                <el-button type="primary" :icon="Promotion" @click="sendMessage" :disabled="!messageInput.trim()" size="small">
                  {{ $t('chat.send') }}
                </el-button>
              </div>
            </div>
          </div>

          <!-- 输入框下方的错误详情卡片 -->
          <div v-if="inputErrors.length > 0" class="error-cards">
            <div v-for="(error, index) in inputErrors" :key="index" class="error-card" :class="error.error_type === '语言语用失误' ? 'pragma-card' : 'socio-card'">
              <div class="error-card-header">
                <el-tag :type="error.error_type === '语言语用失误' ? 'warning' : 'danger'" size="small" effect="dark">
                  {{ error.error_type === '语言语用失误' ? $t('chat.pragmalinguisticError') : $t('chat.sociopragmaticError') }}
                </el-tag>
                <el-button type="primary" size="small" text @click="applySuggestion(error)">
                  {{ $t('chat.applySuggestion') }}
                </el-button>
              </div>
              <div class="error-card-body">
                <div class="error-original">
                  <span class="label">{{ $t('chat.originalMessage') }}:</span>
                  <span class="text error-text">{{ error.original_text }}</span>
                </div>
                <div class="error-suggestion" v-if="error.suggestion">
                  <span class="label">{{ $t('chat.suggestion') }}:</span>
                  <span class="text suggestion-text">{{ error.suggestion }}</span>
                </div>
                <div class="error-explanation" v-if="error.explanation">
                  <span class="label">{{ $t('chat.explanation') }}:</span>
                  <span class="text">{{ error.explanation }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 未选择用户时的提示 -->
      <div v-else class="empty-chat">
        <el-empty :description="$t('chat.selectUser')" />
      </div>
    </div>

    <!-- 右侧: AI Chat 面板 -->
    <div class="ai-panel" :class="{ collapsed: aiPanelCollapsed }">
      <div class="ai-panel-header">
        <div class="ai-panel-title" @click="aiPanelCollapsed = !aiPanelCollapsed">
          <el-icon><ChatDotRound /></el-icon>
          <span>{{ $t('chat.aiAssistant') }}</span>
        </div>
        <el-button :icon="aiPanelCollapsed ? ArrowLeft : ArrowRight" circle size="small" @click="aiPanelCollapsed = !aiPanelCollapsed" />
      </div>

      <div v-if="!aiPanelCollapsed" class="ai-panel-content">
        <div class="ai-messages" ref="aiMessagesContainer">
          <div v-if="aiMessages.length === 0" class="ai-welcome">
            <el-icon :size="48" color="#409eff"><ChatDotRound /></el-icon>
            <p>{{ $t('chat.aiWelcome') }}</p>
            <div class="ai-suggestions">
              <el-button size="small" round @click="sendAiMessage($t('chat.aiSuggest1'))">{{ $t('chat.aiSuggest1') }}</el-button>
              <el-button size="small" round @click="sendAiMessage($t('chat.aiSuggest2'))">{{ $t('chat.aiSuggest2') }}</el-button>
              <el-button size="small" round @click="sendAiMessage($t('chat.aiSuggest3'))">{{ $t('chat.aiSuggest3') }}</el-button>
            </div>
          </div>

          <div v-for="(msg, index) in aiMessages" :key="index" class="ai-message-wrapper" :class="{ 'ai-user-msg': msg.role === 'user', 'ai-bot-msg': msg.role === 'assistant' }">
            <div class="ai-message">
              <div class="ai-message-content" v-html="msg.role === 'assistant' ? renderMarkdown(msg.content) : escapeHtml(msg.content)"></div>
            </div>
          </div>

          <div v-if="aiLoading" class="ai-message-wrapper ai-bot-msg">
            <div class="ai-message">
              <div class="ai-message-content">
                <el-icon class="is-loading"><Loading /></el-icon>
                {{ $t('chat.aiThinking') }}
              </div>
            </div>
          </div>
        </div>

        <div class="ai-input-area">
          <el-input
              v-model="aiInput"
              :placeholder="$t('chat.aiInputPlaceholder')"
              @keydown.enter.prevent="sendAiMessage(aiInput)"
              :disabled="aiLoading"
              size="default"
          >
            <template #append>
              <el-button :icon="Promotion" @click="sendAiMessage(aiInput)" :disabled="!aiInput.trim() || aiLoading" />
            </template>
          </el-input>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { debounce } from 'lodash-es'
import {
  SwitchButton, Document, Promotion, Connection, Delete, MoreFilled,
  WarningFilled, CircleCheckFilled, Loading, ChatDotRound, ArrowLeft, ArrowRight
} from '@element-plus/icons-vue'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { useChatStore } from '@/stores/chat'
import wsManager from '@/utils/websocket'
import LocaleSwitcher from '@/components/LocaleSwitcher.vue'

const router = useRouter()
const userStore = useUserStore()
const chatStore = useChatStore()
const { t } = useI18n()

// ===================== 基础状态 =====================
const searchKeyword = ref('')
const messageInput = ref('')
const messagesContainer = ref(null)
const textareaRef = ref(null)
const underlineLayer = ref(null)

// ===================== 实时检测状态 =====================
const isChecking = ref(false)
const inputErrors = ref([])
const lastCheckTime = ref(null)
let checkTimer = null
let lastCheckedContent = ''

// ===================== 接收方消息错误标识 =====================
const messageErrors = ref({}) // { messageId: { error_type, suggestion, explanation } }

// ===================== AI Chat 状态 =====================
const aiPanelCollapsed = ref(false)
const aiMessages = ref([])
const aiInput = ref('')
const aiLoading = ref(false)
const aiMessagesContainer = ref(null)

// ===================== 计算属性 =====================
const filteredUsers = computed(() => {
  if (!searchKeyword.value) return chatStore.users
  return chatStore.users.filter(user =>
      user.username.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

// ===================== 下划线渲染 =====================
const renderedUnderlineHtml = computed(() => {
  if (!messageInput.value || inputErrors.value.length === 0) {
    return escapeHtml(messageInput.value) + '\n'
  }

  let text = messageInput.value
  let html = ''
  let lastIndex = 0

  // 收集所有错误的位置信息, 按 start 排序
  const markers = []
  for (const error of inputErrors.value) {
    if (error.start_index !== undefined && error.end_index !== undefined) {
      markers.push({
        start: error.start_index,
        end: error.end_index,
        type: error.error_type
      })
    }
  }
  markers.sort((a, b) => a.start - b.start)

  for (const marker of markers) {
    // 正文部分
    if (marker.start > lastIndex) {
      html += escapeHtml(text.slice(lastIndex, marker.start))
    }
    // 错误部分
    const errorClass = marker.type === '语言语用失误' ? 'underline-pragma' : 'underline-socio'
    html += `<span class="${errorClass}">${escapeHtml(text.slice(marker.start, marker.end))}</span>`
    lastIndex = marker.end
  }

  // 剩余部分
  if (lastIndex < text.length) {
    html += escapeHtml(text.slice(lastIndex))
  }

  return html + '\n'
})

// ===================== 生命周期 =====================
onMounted(async () => {
  wsManager.connect()

  try {
    const users = await api.getUsers()
    chatStore.setUsers(users.filter(u => u.id !== userStore.userInfo.id))
  } catch (error) {
    console.error('Load users error:', error)
  }

  // 启动5秒定时检测
  checkTimer = setInterval(() => {
    if (messageInput.value.trim() && messageInput.value.trim() !== lastCheckedContent && chatStore.currentUser) {
      performCheck()
    }
  }, 5000)
})

onUnmounted(() => {
  wsManager.disconnect()
  if (checkTimer) clearInterval(checkTimer)
})

// ===================== 监听当前用户变化 =====================
watch(() => chatStore.currentUser, async (newUser) => {
  if (newUser) {
    try {
      const messages = await api.getMessages(newUser.id)
      chatStore.setCurrentMessages(messages)
      await nextTick()
      scrollToBottom()
      // 加载接收方消息的错误标识
      loadMessageErrors()
    } catch (error) {
      console.error('Load messages error:', error)
    }
  }
  // 清除输入框和错误
  inputErrors.value = []
  lastCheckedContent = ''
  lastCheckTime.value = null
})

// 监听新消息加入后检查错误
watch(() => chatStore.currentMessages, async () => {
  await nextTick()
  scrollToBottom()
  loadMessageErrors()
}, { deep: true })

// ===================== 加载接收方消息错误 =====================
const loadMessageErrors = async () => {
  if (!chatStore.currentUser) return
  const msgs = chatStore.currentMessages
  // 只对收到的消息查询错误
  const receivedMsgIds = msgs
      .filter(m => m.sender_id !== userStore.userInfo.id && m.id)
      .map(m => m.id)

  if (receivedMsgIds.length === 0) return

  try {
    const errors = await api.getMessageErrors(receivedMsgIds)
    const errorMap = {}
    for (const err of errors) {
      errorMap[err.message_id] = err
    }
    messageErrors.value = errorMap
  } catch (e) {
    // 静默处理
    console.error('Load message errors:', e)
  }
}

// ===================== 实时检测逻辑 =====================
const handleInputChange = () => {
  // 输入变化时同步滚动
  syncScroll()
}

const syncScroll = () => {
  if (textareaRef.value && underlineLayer.value) {
    underlineLayer.value.scrollTop = textareaRef.value.scrollTop
    underlineLayer.value.scrollLeft = textareaRef.value.scrollLeft
  }
}

const performCheck = async () => {
  if (!chatStore.currentUser || !messageInput.value.trim()) return

  const content = messageInput.value.trim()
  isChecking.value = true

  try {
    const result = await api.realtimeCheck({
      receiver_id: chatStore.currentUser.id,
      content: content
    })

    lastCheckedContent = content
    lastCheckTime.value = Date.now()

    if (result.has_error && result.errors && result.errors.length > 0) {
      inputErrors.value = result.errors
    } else {
      inputErrors.value = []
    }
  } catch (error) {
    console.error('Realtime check error:', error)
  } finally {
    isChecking.value = false
  }
}

// ===================== 发送消息 =====================
const sendMessage = debounce(async () => {
  if (!messageInput.value.trim() || !chatStore.currentUser) return

  const content = messageInput.value.trim()
  const receiverId = chatStore.currentUser.id

  // 如果有检测到错误, 弹窗确认
  if (inputErrors.value.length > 0) {
    try {
      await ElMessageBox.confirm(
          t('chat.sendWithErrorsConfirm'),
          t('chat.warning'),
          {
            confirmButtonText: t('chat.sendAnyway'),
            cancelButtonText: t('common.cancel'),
            type: 'warning'
          }
      )
    } catch {
      return // 用户取消
    }
  }

  try {
    // 收集错误信息用于关联
    const errorRecordId = inputErrors.value.length > 0 && inputErrors.value[0].error_record_id
        ? inputErrors.value[0].error_record_id
        : 0

    await api.sendMessage({
      receiver_id: receiverId,
      content: content,
      error_record_id: errorRecordId
    })

    messageInput.value = ''
    inputErrors.value = []
    lastCheckedContent = ''
    lastCheckTime.value = null
    ElMessage.success(t('chat.sendSuccess'))
  } catch (error) {
    console.error('Send message error:', error)
    ElMessage.error(t('chat.sendFailed'))
  }
}, 500, { leading: true, trailing: false })

// ===================== 应用建议 =====================
const applySuggestion = (error) => {
  if (error.suggestion) {
    messageInput.value = error.suggestion
    inputErrors.value = []
    lastCheckedContent = ''
  }
}

// ===================== AI Chat 逻辑 =====================
const sendAiMessage = async (content) => {
  if (typeof content !== 'string') return
  const text = content.trim()
  if (!text || aiLoading.value) return

  aiMessages.value.push({ role: 'user', content: text })
  aiInput.value = ''
  aiLoading.value = true

  await nextTick()
  scrollAiToBottom()

  try {
    const result = await api.aiChat({
      message: text,
      chat_user_id: chatStore.currentUser?.id || 0
    })
    aiMessages.value.push({ role: 'assistant', content: result.reply })
  } catch (error) {
    aiMessages.value.push({ role: 'assistant', content: t('chat.aiError') })
    console.error('AI chat error:', error)
  } finally {
    aiLoading.value = false
    await nextTick()
    scrollAiToBottom()
  }
}

const scrollAiToBottom = () => {
  if (aiMessagesContainer.value) {
    aiMessagesContainer.value.scrollTop = aiMessagesContainer.value.scrollHeight
  }
}

// ===================== 工具函数 =====================
const escapeHtml = (text) => {
  if (!text) return ''
  return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/ /g, '&nbsp;')
}

const renderMarkdown = (text) => {
  if (!text) return ''
  // 简单 markdown 渲染
  let html = escapeHtml(text)
  html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
  html = html.replace(/\*(.*?)\*/g, '<em>$1</em>')
  html = html.replace(/\n/g, '<br>')
  return html
}

// ===================== 消息操作 =====================
const deleteMessage = async (messageId) => {
  try {
    await ElMessageBox.confirm(t('chat.deleteConfirm'), t('chat.warning'), {
      confirmButtonText: t('chat.confirmDelete'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    await api.deleteMessage(messageId)
    chatStore.removeMessage(messageId)
    ElMessage.success(t('chat.messageDeleted'))
  } catch (error) {
    if (error !== 'cancel') console.error('Delete message error:', error)
  }
}

const clearChatHistory = async () => {
  if (!chatStore.currentUser) return
  try {
    await ElMessageBox.confirm(
        t('chat.clearConfirm', { username: chatStore.currentUser.username }),
        t('chat.warning'),
        { confirmButtonText: t('chat.confirmClear'), cancelButtonText: t('common.cancel'), type: 'warning', confirmButtonClass: 'el-button--danger' }
    )
    await api.clearChatHistory(chatStore.currentUser.id)
    chatStore.setCurrentMessages([])
    ElMessage.success(t('chat.chatCleared'))
  } catch (error) {
    if (error !== 'cancel') console.error('Clear chat error:', error)
  }
}

const deleteMyMessages = async () => {
  if (!chatStore.currentUser) return
  try {
    await ElMessageBox.confirm(
        t('chat.deleteMyConfirm', { username: chatStore.currentUser.username }),
        t('chat.warning'),
        { confirmButtonText: t('chat.confirmDelete'), cancelButtonText: t('common.cancel'), type: 'warning' }
    )
    const result = await api.deleteMyMessages(chatStore.currentUser.id)
    const messages = await api.getMessages(chatStore.currentUser.id)
    chatStore.setCurrentMessages(messages)
    ElMessage.success(t('chat.messagesDeleted', { count: result.deleted_count }))
  } catch (error) {
    if (error !== 'cancel') console.error('Delete my messages error:', error)
  }
}

const handleLogout = async () => {
  try {
    await ElMessageBox.confirm(t('chat.logoutConfirm'), t('chat.hint'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    userStore.logout()
    wsManager.disconnect()
    ElMessage.success(t('chat.logoutSuccess'))
    router.push('/login')
  } catch (error) { /* cancelled */ }
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
  if (diff < 60000) return t('chat.justNow')
  if (diff < 3600000) return t('chat.minutesAgo', { n: Math.floor(diff / 60000) })
  if (date.toDateString() === now.toDateString()) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
/* ===================== 整体布局 ===================== */
.chat-container {
  display: flex;
  height: 100vh;
  background: #f0f2f5;
  overflow: hidden;
}

/* ===================== 左侧用户列表 ===================== */
.sidebar {
  width: 280px;
  min-width: 280px;
  background: white;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
}
.sidebar-header {
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.sidebar-header h2 {
  margin: 0;
  font-size: 16px;
  color: #303133;
}
.header-actions {
  display: flex;
  gap: 6px;
}
.search-box {
  padding: 12px;
}
.connection-status {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  font-size: 12px;
  background: #f5f7fa;
  border-top: 1px solid #e4e7ed;
  border-bottom: 1px solid #e4e7ed;
}
.connection-status.connected { color: #67c23a; }
.connection-status.disconnected { color: #f56c6c; }
.user-list {
  flex: 1;
  overflow-y: auto;
}
.user-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 0.2s;
}
.user-item:hover { background: #f5f7fa; }
.user-item.active { background: #ecf5ff; }
.user-info { flex: 1; }
.username { font-size: 14px; color: #303133; font-weight: 500; }
.country { font-size: 12px; color: #909399; margin-top: 2px; }

/* ===================== 中间聊天区域 ===================== */
.chat-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.chat-content {
  display: flex;
  flex-direction: column;
  height: 100vh;
}
.chat-header {
  background: white;
  padding: 12px 20px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.chat-user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}
.messages-container {
  flex: 1;
  padding: 16px 20px;
  overflow-y: auto;
  background: #f5f5f5;
}
.message-wrapper {
  margin-bottom: 16px;
  display: flex;
}
.message-wrapper.message-sent {
  justify-content: flex-end;
}
.message {
  max-width: 65%;
  background: white;
  border-radius: 12px;
  padding: 10px 14px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}
.message-sent .message {
  background: #409eff;
  color: white;
}
.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
  font-size: 11px;
  opacity: 0.7;
}
.message-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}
.message-text {
  word-break: break-word;
  line-height: 1.6;
  font-size: 14px;
}
.delete-btn { padding: 2px; }
.empty-messages, .empty-chat {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

/* ===================== 接收方错误标识 ===================== */
.error-indicator {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  cursor: pointer;
  transition: transform 0.2s;
}
.error-indicator:hover {
  transform: scale(1.2);
}
.error-indicator.pragma {
  color: #e6a23c;
  background: #fdf6ec;
}
.error-indicator.socio {
  color: #f56c6c;
  background: #fef0f0;
}

.error-tooltip-content {
  max-width: 300px;
}
.error-tooltip-header {
  margin-bottom: 8px;
}
.error-tooltip-section {
  margin-bottom: 8px;
}
.error-tooltip-label {
  font-size: 12px;
  color: #909399;
  margin-bottom: 4px;
}
.error-tooltip-text {
  font-size: 13px;
  line-height: 1.5;
  color: #303133;
}
.error-tooltip-text.suggestion {
  color: #67c23a;
  padding: 6px 8px;
  background: #f0f9eb;
  border-radius: 4px;
}
.error-tooltip-text.explanation {
  color: #606266;
  padding: 6px 8px;
  background: #f4f4f5;
  border-radius: 4px;
}

/* ===================== Grammarly 风格输入区域 ===================== */
.input-area {
  background: white;
  border-top: 1px solid #e4e7ed;
  padding: 12px 20px 8px;
}
.input-wrapper {
  border: 2px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
  transition: border-color 0.3s;
}
.input-wrapper:focus-within {
  border-color: #409eff;
}
.input-editor-container {
  position: relative;
  min-height: 80px;
  max-height: 150px;
}
.underline-layer {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 10px 14px;
  font-size: 14px;
  line-height: 1.6;
  font-family: inherit;
  white-space: pre-wrap;
  word-wrap: break-word;
  overflow: hidden;
  color: transparent;
  pointer-events: none;
  z-index: 0;
}
.underline-layer :deep(.underline-pragma) {
  background: transparent;
  border-bottom: 3px solid #e6a23c;
  color: transparent;
}
.underline-layer :deep(.underline-socio) {
  background: transparent;
  border-bottom: 3px solid #f56c6c;
  color: transparent;
}
.input-editor {
  position: relative;
  width: 100%;
  min-height: 80px;
  max-height: 150px;
  padding: 10px 14px;
  border: none;
  outline: none;
  font-size: 14px;
  line-height: 1.6;
  font-family: inherit;
  resize: none;
  background: transparent;
  z-index: 1;
  overflow-y: auto;
}
.input-status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 14px;
  background: #fafafa;
  border-top: 1px solid #f0f0f0;
}
.status-left {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}
.checking-status { color: #909399; }
.error-count { color: #e6a23c; }
.no-error { color: #67c23a; }

/* ===================== 错误详情卡片 ===================== */
.error-cards {
  margin-top: 8px;
  max-height: 180px;
  overflow-y: auto;
}
.error-card {
  padding: 10px 14px;
  border-radius: 8px;
  margin-bottom: 6px;
  border-left: 4px solid;
}
.error-card.pragma-card {
  background: #fdf6ec;
  border-left-color: #e6a23c;
}
.error-card.socio-card {
  background: #fef0f0;
  border-left-color: #f56c6c;
}
.error-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.error-card-body {
  font-size: 13px;
  line-height: 1.5;
}
.error-card-body .label {
  font-weight: 600;
  color: #606266;
  margin-right: 4px;
}
.error-card-body .error-text {
  text-decoration: line-through;
  color: #909399;
}
.error-card-body .suggestion-text {
  color: #67c23a;
  font-weight: 500;
}
.error-card-body > div {
  margin-bottom: 4px;
}
.error-original, .error-suggestion, .error-explanation {
  margin-bottom: 4px;
}

/* ===================== 右侧 AI Chat ===================== */
.ai-panel {
  width: 340px;
  min-width: 340px;
  background: white;
  border-left: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  transition: width 0.3s, min-width 0.3s;
}
.ai-panel.collapsed {
  width: 48px;
  min-width: 48px;
}
.ai-panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 14px;
  border-bottom: 1px solid #e4e7ed;
  cursor: pointer;
}
.ai-panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}
.ai-panel.collapsed .ai-panel-title span {
  display: none;
}
.ai-panel-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.ai-messages {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}
.ai-welcome {
  text-align: center;
  padding: 40px 16px;
  color: #909399;
}
.ai-welcome p {
  margin: 12px 0;
  font-size: 13px;
}
.ai-suggestions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 16px;
  width: 100%;
}

/* 让 suggestion 按钮文字自动换行、边框随内容伸缩 */
.ai-suggestions .el-button {
  white-space: normal !important;
  word-break: break-word;
  height: auto !important;
  line-height: 1.5 !important;
  padding: 8px 14px !important;
  width: 100%;
  text-align: center;
}
.ai-message-wrapper {
  margin-bottom: 12px;
  display: flex;
}
.ai-user-msg {
  justify-content: flex-end;
}
.ai-message {
  max-width: 90%;
  border-radius: 12px;
  padding: 8px 12px;
  font-size: 13px;
  line-height: 1.6;
}
.ai-user-msg .ai-message {
  background: #409eff;
  color: white;
}
.ai-bot-msg .ai-message {
  background: #f4f4f5;
  color: #303133;
}
.ai-message-content :deep(strong) { font-weight: 600; }
.ai-input-area {
  padding: 12px;
  border-top: 1px solid #e4e7ed;
}
</style>

<style>
/* 全局 tooltip 样式 */
.error-tooltip-popper {
  max-width: 360px !important;
}
</style>