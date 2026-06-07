<template>
  <div class="chat-container">
    <!-- 左侧：会话信息面板 -->
    <div class="sidebar">
      <div class="sidebar-header">
        <h2>{{ userStore.userInfo.username }}</h2>
        <div class="header-actions">
          <LocaleSwitcher />
          <el-button :icon="Document" circle @click="router.push('/grammar-errors')" :title="$t('chat.errorRecord')" />
          <el-button :icon="SwitchButton" circle @click="handleLogout" :title="$t('chat.logout')" />
        </div>
      </div>

      <!-- 会话信息 -->
      <div v-if="sessionStore.currentSession" class="session-info">
        <div class="session-info-title">{{ $t('chat.sessionInfo') }}</div>
        <div class="session-info-item">
          <span class="info-label">{{ $t('setup.relationship') }}:</span>
          <span class="info-value">{{ sessionStore.currentSession.relationship_type }}</span>
        </div>
        <div class="session-info-item">
          <span class="info-label">{{ $t('setup.topic') }}:</span>
          <span class="info-value">{{ sessionStore.currentSession.topic }}</span>
        </div>
        <div class="session-info-item">
          <span class="info-label">{{ $t('chat.mode') }}:</span>
          <el-tag size="small" :type="sessionStore.currentSession.mode === 'user_l2' ? 'primary' : 'success'">
            {{ sessionStore.currentSession.mode === 'user_l2' ? $t('setup.modeUserL2Short') : $t('setup.modeLLML2Short') }}
          </el-tag>
        </div>
        <div class="session-info-item">
          <span class="info-label">{{ $t('chat.targetLanguage') }}:</span>
          <span class="info-value">{{ getLanguageName(sessionStore.currentSession.target_language) }}</span>
        </div>
        <div class="session-info-item">
          <span class="info-label">{{ $t('chat.rounds') }}:</span>
          <span class="info-value round-count">{{ sessionStore.currentSession.round_count || 0 }}</span>
          <span v-if="sessionStore.currentSession.feedback_mode === 'rounds_5'" class="rounds-limit"> / 5</span>
        </div>

        <!-- 倒计时（完整对话模式） -->
        <div v-if="sessionStore.currentSession.feedback_mode === 'complete' && sessionActive && timeRemaining > 0" class="timer-info" :class="{ 'timer-warning': timeRemaining <= 60 }">
          <el-icon><Timer /></el-icon>
          <span>{{ formatTimeRemaining() }}</span>
        </div>

        <el-divider />
        <el-button type="danger" plain size="small" @click="endSession" class="end-session-btn" :disabled="!sessionActive">
          {{ $t('chat.endSession') }}
        </el-button>
        <el-button plain size="small" @click="goToSetup" class="new-session-btn">
          {{ $t('chat.newSession') }}
        </el-button>
      </div>

      <!-- 语用错误统计 -->
      <div v-if="errorCount > 0" class="error-stats">
        <div class="error-stats-title">{{ $t('chat.pragmaticErrors') }}</div>
        <div class="error-stats-count">
          <el-icon color="#e6a23c"><WarningFilled /></el-icon>
          <span>{{ $t('chat.errorsDetected', { count: errorCount }) }}</span>
        </div>
      </div>
    </div>

    <!-- 中间：聊天区域 -->
    <div class="chat-area">
      <div class="chat-content">
        <!-- 聊天头部 -->
        <div class="chat-header">
          <div class="chat-user-info">
            <div class="bot-avatar">
              <el-icon :size="24" color="white"><ChatDotRound /></el-icon>
            </div>
            <div>
              <div class="username">{{ $t('chat.llmBot') }}</div>
              <div class="country">
                <el-tag size="small" effect="plain" :type="sessionStore.currentSession?.mode === 'user_l2' ? 'primary' : 'success'">
                  {{ sessionStore.currentSession?.mode === 'user_l2' ? $t('setup.modeUserL2Short') : $t('setup.modeLLML2Short') }}
                </el-tag>
                <span class="target-lang">{{ getLanguageName(sessionStore.currentSession?.target_language) }}</span>
              </div>
            </div>
          </div>
          <div class="header-right">
            <el-tag v-if="sessionStore.currentSession?.feedback_mode === 'complete'" size="small" type="info">
              {{ $t('setup.feedbackPeriodic') }}
            </el-tag>
            <el-tag v-else size="small" type="success">
              {{ $t('setup.feedbackSummary') }}
            </el-tag>
          </div>
        </div>

        <!-- 消息列表 -->
        <div class="messages-container" ref="messagesContainer">
          <!-- 开场白 -->
          <div class="welcome-message" v-if="sessionStore.messages.length === 0">
            <div class="welcome-content">
              <el-icon :size="40" color="#409eff"><ChatDotRound /></el-icon>
              <h3>{{ $t('chat.welcomeTitle') }}</h3>
              <p>{{ getWelcomeMessage() }}</p>
              <div class="welcome-tips">
                <p v-if="sessionStore.currentSession?.mode === 'user_l2'">{{ $t('chat.tipUserL2') }}</p>
                <p v-else>{{ $t('chat.tipLLML2') }}</p>
              </div>
            </div>
          </div>

          <div
            v-for="msg in sessionStore.messages"
            :key="msg.id"
            class="message-wrapper"
            :class="{ 'message-sent': msg.role === 'user' || msg.sender_id === userStore.userInfo.id }"
          >
            <!-- Bot头像（左侧消息） -->
            <div v-if="msg.role === 'llm' || msg.sender_id !== userStore.userInfo.id" class="message-avatar bot-msg-avatar">
              <el-icon :size="16" color="white"><ChatDotRound /></el-icon>
            </div>

            <div class="message">
              <div class="message-header">
                <span class="message-sender">
                  {{ msg.role === 'llm' || msg.sender_id !== userStore.userInfo.id ? $t('chat.llmBot') : userStore.userInfo.username }}
                </span>
                <span class="message-time">{{ formatMessageTime(msg.created_at) }}</span>
              </div>
              <div class="message-text">{{ msg.content }}</div>
              <!-- 语用错误标识（用户消息） -->
              <div v-if="(msg.role === 'user' || msg.sender_id === userStore.userInfo.id) && messageErrors[msg.id]" class="message-error-badge">
                <el-popover
                  placement="top"
                  :width="300"
                  trigger="click"
                >
                  <template #reference>
                    <span class="error-indicator" :class="getErrorClass(messageErrors[msg.id].error_type)">
                      <el-icon :size="12"><WarningFilled /></el-icon>
                      {{ getErrorTypeShort(messageErrors[msg.id].error_type) }}
                    </span>
                  </template>
                  <div class="error-popover">
                    <div class="error-popover-type">
                      <el-tag size="small" :type="isProblematicType(messageErrors[msg.id].error_type) ? 'danger' : 'warning'">
                        {{ messageErrors[msg.id].error_type }}
                      </el-tag>
                    </div>
                    <div v-if="messageErrors[msg.id].suggestion" class="error-popover-section">
                      <div class="error-popover-label">{{ $t('chat.suggestion') }}</div>
                      <div class="error-popover-text suggestion-text">{{ messageErrors[msg.id].suggestion }}</div>
                    </div>
                    <div v-if="messageErrors[msg.id].explanation" class="error-popover-section">
                      <div class="error-popover-label">{{ $t('chat.explanation') }}</div>
                      <div class="error-popover-text">{{ messageErrors[msg.id].explanation }}</div>
                    </div>
                  </div>
                </el-popover>
              </div>
            </div>
          </div>

          <!-- LLM 正在输入指示 -->
          <div v-if="isLLMTyping" class="message-wrapper">
            <div class="message-avatar bot-msg-avatar">
              <el-icon :size="16" color="white"><ChatDotRound /></el-icon>
            </div>
            <div class="message typing-indicator">
              <span></span><span></span><span></span>
            </div>
          </div>
        </div>

        <!-- 输入区域 -->
        <div class="input-area">
          <div class="input-wrapper">
            <textarea
              ref="textareaRef"
              v-model="messageInput"
              class="input-editor"
              :placeholder="getInputPlaceholder()"
              @keydown.ctrl.enter="sendMessage"
              :disabled="isSending || !sessionActive"
            ></textarea>
            <div class="input-status-bar">
              <div class="status-left">
                <span v-if="!sessionActive" class="session-ended-hint">{{ $t('feedback.summaryTitle') }}</span>
              </div>
              <div class="status-right">
                <el-button
                  type="primary"
                  :icon="Promotion"
                  @click="sendMessage"
                  :disabled="!messageInput.trim() || isSending || !sessionStore.currentSession || !sessionActive"
                  size="small"
                >
                  {{ $t('chat.send') }}
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 会话汇总对话框 -->
    <el-dialog
      v-model="showSummary"
      :title="$t('feedback.summaryTitle')"
      width="600px"
      :close-on-click-modal="false"
    >
      <div class="feedback-content">
        <div v-if="autoEndReason" class="auto-end-notice">
          <el-icon color="#409eff"><InfoFilled /></el-icon>
          <span>{{ autoEndReason }}</span>
        </div>
        <div class="feedback-stats">
          <div class="stat-item">
            <div class="stat-num">{{ sessionStore.currentSession?.round_count || 0 }}</div>
            <div class="stat-label">{{ $t('feedback.totalRounds') }}</div>
          </div>
          <div class="stat-item">
            <div class="stat-num">{{ errorCount }}</div>
            <div class="stat-label">{{ $t('feedback.totalErrors') }}</div>
          </div>
        </div>
        <el-divider />
        <div v-if="summaryLoading" class="summary-loading">
          <el-icon class="is-loading"><Loading /></el-icon>
          {{ $t('feedback.generating') }}
        </div>
        <div v-else class="feedback-text" v-html="renderMarkdown(summaryText)"></div>
      </div>
      <template #footer>
        <el-button @click="showSummary = false; goToSetup()">{{ $t('feedback.newSession') }}</el-button>
        <el-button type="primary" @click="showSummary = false">{{ $t('feedback.close') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'

import {
  SwitchButton, Document, Promotion, WarningFilled,
  Loading, ChatDotRound, Timer, InfoFilled
} from '@element-plus/icons-vue'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { useSessionStore } from '@/stores/session'
import LocaleSwitcher from '@/components/LocaleSwitcher.vue'

const router = useRouter()
const userStore = useUserStore()
const sessionStore = useSessionStore()
const { t } = useI18n()

// ===================== 基础状态 =====================
const messageInput = ref('')
const messagesContainer = ref(null)
const textareaRef = ref(null)
const isSending = ref(false)
const isLLMTyping = ref(false)
const sessionActive = ref(true)

// ===================== 消息错误标识 =====================
const messageErrors = ref({})
const errorCount = ref(0)

// ===================== 汇总对话框状态 =====================
const showSummary = ref(false)
const summaryText = ref('')
const summaryLoading = ref(false)
const autoEndReason = ref('')

// ===================== 10分钟倒计时 =====================
const SESSION_DURATION = 10 * 60 // 10分钟（秒）
const timeRemaining = ref(SESSION_DURATION)
let countdownTimer = null

const formatTimeRemaining = () => {
  const min = Math.floor(timeRemaining.value / 60)
  const sec = timeRemaining.value % 60
  return t('feedback.timeRemaining', { min, sec: String(sec).padStart(2, '0') })
}

const startCountdown = () => {
  if (!sessionStore.currentSession || sessionStore.currentSession.feedback_mode !== 'complete') return
  countdownTimer = setInterval(() => {
    if (!sessionActive.value) {
      clearInterval(countdownTimer)
      return
    }
    timeRemaining.value--
    if (timeRemaining.value === 60) {
      ElMessage.warning(t('feedback.timeoutWarning'))
    }
    if (timeRemaining.value <= 0) {
      clearInterval(countdownTimer)
      autoEndSession('timeout')
    }
  }, 1000)
}

const autoEndSession = async (reason) => {
  if (!sessionActive.value || !sessionStore.currentSession) return
  sessionActive.value = false
  summaryLoading.value = true
  showSummary.value = true
  autoEndReason.value = reason === 'timeout'
    ? t('feedback.autoEndTimeout')
    : t('feedback.autoEndRounds')

  try {
    const data = await api.endSession(sessionStore.currentSession.id)
    summaryText.value = data.summary_feedback || t('feedback.noSummary')
    if (sessionStore.currentSession) {
      sessionStore.currentSession.is_active = false
    }
  } catch (error) {
    console.error('Auto end session error:', error)
    summaryText.value = t('feedback.noSummary')
  } finally {
    summaryLoading.value = false
  }
}

// ===================== 生命周期 =====================
onMounted(async () => {
  if (!sessionStore.currentSession) {
    router.push('/setup')
    return
  }

  if (sessionStore.currentSession.is_active === false) {
    sessionActive.value = false
  }

  if (sessionStore.messages.length === 0) {
    await loadSessionMessages()
  }
  await nextTick()
  scrollToBottom()

  if (sessionActive.value) {
    startCountdown()
  }
})

onUnmounted(() => {
  if (countdownTimer) clearInterval(countdownTimer)
})

// ===================== 加载会话消息 =====================
const loadSessionMessages = async () => {
  if (!sessionStore.currentSession) return
  try {
    const data = await api.getSessionMessages(sessionStore.currentSession.id)
    sessionStore.setMessages(data.messages || [])
    sessionStore.setSession(data.session)
    await loadMessageErrors()
  } catch (e) {
    console.error('Load session messages error:', e)
  }
}

// ===================== 加载消息语用错误 =====================
const loadMessageErrors = async () => {
  const msgs = sessionStore.messages.filter(m => m.role === 'user' || m.sender_id === userStore.userInfo.id)
  if (msgs.length === 0) return
  const msgIds = msgs.map(m => m.id).filter(Boolean)
  if (msgIds.length === 0) return
  try {
    const errors = await api.getMessageErrors(msgIds)
    const errorMap = {}
    for (const err of errors) {
      errorMap[err.message_id] = err
    }
    messageErrors.value = errorMap
    errorCount.value = errors.length
  } catch (e) {
    console.error('Load message errors:', e)
  }
}

// ===================== 发送消息给LLM =====================
const sendMessage = async () => {
  if (isSending.value || !messageInput.value.trim() || !sessionStore.currentSession || !sessionActive.value) return

  isSending.value = true
  const content = messageInput.value.trim()
  messageInput.value = ''
  isLLMTyping.value = true

  try {
    const result = await api.sendLLMMessage({
      session_id: sessionStore.currentSession.id,
      content: content
    })

    if (result.user_message) {
      sessionStore.addMessage(result.user_message)
    }
    if (result.llm_response) {
      sessionStore.addMessage(result.llm_response)
    }

    sessionStore.incrementRound()

    if (result.pragmatic_check?.has_error && result.user_message?.id) {
      messageErrors.value[result.user_message.id] = {
        message_id: result.user_message.id,
        error_type: result.pragmatic_check.error_type,
        suggestion: result.pragmatic_check.suggestion,
        explanation: result.pragmatic_check.explanation,
      }
      errorCount.value++
    }

    await nextTick()
    scrollToBottom()

    // 五轮模式自动结束
    if (result.session_ended) {
      sessionActive.value = false
      if (countdownTimer) clearInterval(countdownTimer)
      summaryText.value = result.session_summary || t('feedback.noSummary')
      autoEndReason.value = t('feedback.autoEndRounds')
      showSummary.value = true
      if (sessionStore.currentSession) {
        sessionStore.currentSession.is_active = false
      }
    }

  } catch (error) {
    console.error('Send message error:', error)
    ElMessage.error(t('chat.sendFailed'))
    messageInput.value = content
  } finally {
    isSending.value = false
    isLLMTyping.value = false
  }
}

// ===================== 结束会话 =====================
const endSession = async () => {
  try {
    await ElMessageBox.confirm(t('chat.endSessionConfirm'), t('chat.hint'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })

    if (countdownTimer) clearInterval(countdownTimer)
    sessionActive.value = false
    summaryLoading.value = true
    showSummary.value = true
    autoEndReason.value = ''

    const data = await api.endSession(sessionStore.currentSession.id)
    summaryText.value = data.summary_feedback || t('feedback.noSummary')

    if (sessionStore.currentSession) {
      sessionStore.currentSession.is_active = false
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('End session error:', error)
      ElMessage.error(t('chat.endSessionFailed'))
    }
    showSummary.value = false
    sessionActive.value = true
  } finally {
    summaryLoading.value = false
  }
}

const goToSetup = () => {
  sessionStore.clearSession()
  router.push('/setup')
}

// ===================== 工具函数 =====================
const isProblematicType = (errorType) => {
  return errorType === '严重语用语言失误' ||
    errorType === '严重社会语用失误' ||
    errorType === '语用语言失误和社会语用失误'
}

const getErrorClass = (errorType) => {
  return isProblematicType(errorType) ? 'error-problematic' : 'error-improvable'
}

const getErrorTypeShort = (errorType) => {
  if (!errorType) return ''
  const map = {
    '语用语言失误': t('chat.errorShort1'),
    '社会语用失误': t('chat.errorShort2'),
    '严重语用语言失误': t('chat.errorShort3'),
    '严重社会语用失误': t('chat.errorShort4'),
    '语用语言失误和社会语用失误': t('chat.errorShort5'),
  }
  return map[errorType] || errorType
}

const getLanguageName = (lang) => {
  const map = { EN: 'English', ZH: '中文', JP: '日本語', KR: '한국어', FR: 'Français', DE: 'Deutsch' }
  return map[lang] || lang || ''
}

const getInputPlaceholder = () => {
  if (!sessionActive.value) return ''
  if (!sessionStore.currentSession) return t('chat.inputPlaceholder')
  const lang = getLanguageName(sessionStore.currentSession.target_language)
  if (sessionStore.currentSession.mode === 'user_l2') {
    return t('chat.inputPlaceholderL2', { lang })
  }
  return t('chat.inputPlaceholderNative')
}

const getWelcomeMessage = () => {
  if (!sessionStore.currentSession) return ''
  const lang = getLanguageName(sessionStore.currentSession.target_language)
  const rel = sessionStore.currentSession.relationship_type
  const topic = sessionStore.currentSession.topic
  if (sessionStore.currentSession.mode === 'user_l2') {
    return t('chat.welcomeUserL2', { lang, rel, topic })
  }
  return t('chat.welcomeLLML2', { lang, rel, topic })
}

const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const formatMessageTime = (timestamp) => {
  if (!timestamp) return ''
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

const escapeHtml = (text) => {
  if (!text) return ''
  return text
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

const renderMarkdown = (text) => {
  if (!text) return ''
  let html = escapeHtml(text)
  html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
  html = html.replace(/\*(.*?)\*/g, '<em>$1</em>')
  html = html.replace(/\n/g, '<br>')
  return html
}

const handleLogout = async () => {
  try {
    await ElMessageBox.confirm(t('chat.logoutConfirm'), t('chat.hint'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    if (countdownTimer) clearInterval(countdownTimer)
    userStore.logout()
    sessionStore.clearSession()
    router.push('/login')
  } catch { /* cancelled */ }
}

watch(() => sessionStore.messages.length, async () => {
  await nextTick()
  scrollToBottom()
})
</script>

<style scoped>
.chat-container {
  display: flex;
  height: 100vh;
  background: #f0f2f5;
  overflow: hidden;
}

/* ===================== 左侧会话信息面板 ===================== */
.sidebar {
  width: 260px;
  min-width: 260px;
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

.session-info {
  padding: 16px;
  flex: 1;
}

.session-info-title {
  font-size: 13px;
  font-weight: 600;
  color: #606266;
  margin-bottom: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.session-info-item {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin-bottom: 10px;
  font-size: 13px;
}

.info-label {
  color: #909399;
  white-space: nowrap;
  flex-shrink: 0;
}

.info-value {
  color: #303133;
  font-weight: 500;
}

.round-count {
  font-size: 18px;
  font-weight: 700;
  color: #409eff;
}

.rounds-limit {
  font-size: 14px;
  color: #909399;
}

.timer-info {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #909399;
  padding: 6px 10px;
  background: #f4f4f5;
  border-radius: 6px;
  margin-bottom: 8px;
}

.timer-info.timer-warning {
  color: #e6a23c;
  background: #fdf6ec;
}

.end-session-btn,
.new-session-btn {
  width: 100%;
  margin-bottom: 8px;
}

.error-stats {
  padding: 12px 16px;
  border-top: 1px solid #e4e7ed;
  background: #fdf6ec;
}

.error-stats-title {
  font-size: 12px;
  color: #909399;
  margin-bottom: 6px;
}

.error-stats-count {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #e6a23c;
  font-weight: 500;
}

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

.bot-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea, #764ba2);
  display: flex;
  align-items: center;
  justify-content: center;
}

.username {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.country {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 2px;
}

.target-lang {
  font-size: 12px;
  color: #909399;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.messages-container {
  flex: 1;
  padding: 16px 20px;
  overflow-y: auto;
  background: #f5f5f5;
}

/* ===================== 欢迎页 ===================== */
.welcome-message {
  display: flex;
  justify-content: center;
  padding: 40px 20px;
}

.welcome-content {
  text-align: center;
  max-width: 400px;
}

.welcome-content h3 {
  font-size: 18px;
  color: #303133;
  margin: 12px 0 8px;
}

.welcome-content p {
  color: #606266;
  font-size: 14px;
  line-height: 1.6;
}

.welcome-tips {
  background: #ecf5ff;
  border-radius: 8px;
  padding: 12px 16px;
  margin-top: 12px;
  text-align: left;
}

.welcome-tips p {
  margin: 0;
  font-size: 13px;
  color: #409eff;
}

/* ===================== 消息气泡 ===================== */
.message-wrapper {
  margin-bottom: 16px;
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

.message-wrapper.message-sent {
  justify-content: flex-end;
}

.message-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea, #764ba2);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.bot-msg-avatar {
  align-self: flex-end;
}

.message {
  max-width: 65%;
  background: white;
  border-radius: 12px 12px 12px 4px;
  padding: 10px 14px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.message-sent .message {
  background: #409eff;
  color: white;
  border-radius: 12px 12px 4px 12px;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
  font-size: 11px;
  opacity: 0.7;
}

.message-sender {
  font-weight: 600;
}

.message-text {
  word-break: break-word;
  line-height: 1.6;
  font-size: 14px;
}

.message-error-badge {
  margin-top: 6px;
}

.error-indicator {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 11px;
  cursor: pointer;
}

.error-improvable {
  background: #fdf6ec;
  color: #e6a23c;
  border: 1px solid #f5dab1;
}

.error-problematic {
  background: #fef0f0;
  color: #f56c6c;
  border: 1px solid #fbc4c4;
}

/* 打字动画 */
.typing-indicator {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 14px 16px;
  background: white;
  border-radius: 12px 12px 12px 4px;
}

.typing-indicator span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #909399;
  animation: typing 1.2s infinite;
}

.typing-indicator span:nth-child(2) { animation-delay: 0.2s; }
.typing-indicator span:nth-child(3) { animation-delay: 0.4s; }

@keyframes typing {
  0%, 60%, 100% { transform: translateY(0); opacity: 0.4; }
  30% { transform: translateY(-6px); opacity: 1; }
}

/* ===================== 错误详情弹出框 ===================== */
.error-popover {
  font-size: 13px;
}

.error-popover-type {
  margin-bottom: 10px;
}

.error-popover-section {
  margin-bottom: 8px;
}

.error-popover-label {
  font-size: 11px;
  color: #909399;
  margin-bottom: 3px;
}

.error-popover-text {
  color: #303133;
  line-height: 1.5;
}

.suggestion-text {
  color: #67c23a;
  padding: 4px 8px;
  background: #f0f9eb;
  border-radius: 4px;
}

/* ===================== 输入区域 ===================== */
.input-area {
  background: white;
  border-top: 1px solid #e4e7ed;
  padding: 12px 20px 12px;
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

.input-editor {
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
  background: white;
  overflow-y: auto;
  box-sizing: border-box;
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

.session-ended-hint {
  color: #909399;
  font-style: italic;
}

/* ===================== 反馈对话框 ===================== */
.feedback-content {
  padding: 4px 0;
}

.auto-end-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #ecf5ff;
  border-radius: 6px;
  margin-bottom: 12px;
  font-size: 13px;
  color: #409eff;
}

.feedback-text {
  font-size: 14px;
  line-height: 1.8;
  color: #303133;
  white-space: pre-wrap;
}

.feedback-text :deep(strong) {
  color: #409eff;
}

.feedback-stats {
  display: flex;
  justify-content: space-around;
  padding: 16px 0;
}

.stat-item {
  text-align: center;
}

.stat-num {
  font-size: 32px;
  font-weight: 700;
  color: #409eff;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  margin-top: 4px;
}

.summary-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  color: #909399;
}
</style>
