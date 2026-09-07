<template>
  <div class="setup-container">
    <div class="locale-switcher-wrapper">
      <LocaleSwitcher />
    </div>

    <div class="setup-box">
      <div class="setup-header">
        <h1>{{ $t('setup.title') }}</h1>
        <p>{{ $t('setup.subtitle') }}</p>
      </div>

      <el-form :model="form" :rules="rules" ref="formRef" label-position="top" class="setup-form">
        <!-- 对话模式 -->
        <div class="section-title">{{ $t('setup.modeTitle') }}</div>
        <el-form-item prop="mode">
          <div class="mode-cards" role="radiogroup" :aria-label="$t('setup.modeTitle')">
            <div
              ref="modeUserL2Ref"
              class="mode-card"
              :class="{ active: form.mode === 'user_l2' }"
              role="radio"
              :tabindex="form.mode === 'user_l2' ? 0 : -1"
              :aria-checked="form.mode === 'user_l2'"
              @click="form.mode = 'user_l2'"
              @keydown.enter.space.prevent="form.mode = 'user_l2'"
              @keydown.left.up.prevent="selectModeWithFocus('llm_l2')"
              @keydown.right.down.prevent="selectModeWithFocus('llm_l2')"
            >
              <el-icon :size="32" color="#409eff"><User /></el-icon>
              <div class="mode-card-title">{{ $t('setup.modeUserL2Title') }}</div>
              <div class="mode-card-desc">{{ $t('setup.modeUserL2Desc') }}</div>
            </div>
            <div
              ref="modeLLML2Ref"
              class="mode-card"
              :class="{ active: form.mode === 'llm_l2' }"
              role="radio"
              :tabindex="form.mode === 'llm_l2' ? 0 : -1"
              :aria-checked="form.mode === 'llm_l2'"
              @click="form.mode = 'llm_l2'"
              @keydown.enter.space.prevent="form.mode = 'llm_l2'"
              @keydown.left.up.prevent="selectModeWithFocus('user_l2')"
              @keydown.right.down.prevent="selectModeWithFocus('user_l2')"
            >
              <el-icon :size="32" color="#67c23a"><Monitor /></el-icon>
              <div class="mode-card-title">{{ $t('setup.modeLLML2Title') }}</div>
              <div class="mode-card-desc">{{ $t('setup.modeLLML2Desc') }}</div>
            </div>
          </div>
        </el-form-item>

        <!-- 目标语言 -->
        <el-form-item :label="$t('setup.targetLanguage')" prop="target_language">
          <el-select v-model="form.target_language" :placeholder="$t('setup.selectLanguage')" size="large" class="full-width">
            <el-option v-for="lang in languages" :key="lang.value" :label="lang.label" :value="lang.value" />
          </el-select>
        </el-form-item>

        <!-- 对话对象国家/地区；人物档案在创建会话时由后端生成并固化。 -->
        <el-form-item :label="$t('setup.personaCountry')" prop="llm_country">
          <el-select
            v-model="form.llm_country"
            :placeholder="$t('setup.selectPersonaCountry')"
            :loading="personaCountriesLoading"
            size="large"
            class="full-width"
          >
            <el-option
              v-for="country in personaCountries"
              :key="country.code"
              :label="`${isZh ? country.name_zh : country.name_en} · ${country.native_language}`"
              :value="country.code"
            />
          </el-select>
          <div class="persona-hint">
            {{ form.mode === 'user_l2' ? $t('setup.personaCountryUserL2Hint') : $t('setup.personaCountryLLML2Hint') }}
          </div>
          <div class="persona-hint">{{ $t('setup.randomPersonaHint') }}</div>
        </el-form-item>

        <!-- 关系类型 -->
        <el-form-item :label="$t('setup.relationship')" prop="relationship_type">
          <el-select v-model="form.relationship_type" :placeholder="$t('setup.selectRelationship')" size="large" class="full-width">
            <el-option v-for="r in relationships" :key="r.value" :label="r.label" :value="r.value" />
          </el-select>
        </el-form-item>

        <!-- 对话主题 -->
        <el-form-item :label="$t('setup.topic')" prop="topic">
          <el-select v-model="form.topic" :placeholder="$t('setup.selectTopic')" size="large" class="full-width">
            <el-option v-for="t in topics" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>

        <!-- AI建议开关 -->
        <div class="section-title">{{ $t('setup.aiSuggestionsTitle') }}</div>
        <div class="ai-suggestions-switch">
          <div>
            <div class="ai-suggestions-label">{{ form.ai_suggestions_enabled ? $t('setup.aiSuggestionsOn') : $t('setup.aiSuggestionsOff') }}</div>
            <div class="ai-suggestions-desc">{{ form.ai_suggestions_enabled ? $t('setup.aiSuggestionsOnDesc') : $t('setup.aiSuggestionsOffDesc') }}</div>
          </div>
          <el-switch v-model="form.ai_suggestions_enabled" size="large" :aria-label="$t('setup.aiSuggestionsTitle')" />
        </div>

        <!-- AI建议的展示方式 -->
        <template v-if="form.ai_suggestions_enabled">
        <div class="section-title feedback-method-title">{{ $t('setup.feedbackTitle') }}</div>
        <el-form-item prop="feedback_mode">
          <div class="feedback-cards" role="radiogroup" :aria-label="$t('setup.feedbackTitle')">
            <div
              ref="feedbackCompleteRef"
              class="feedback-card"
              :class="{ active: form.feedback_mode === 'complete' }"
              role="radio"
              :tabindex="form.feedback_mode === 'complete' ? 0 : -1"
              :aria-checked="form.feedback_mode === 'complete'"
              @click="form.feedback_mode = 'complete'"
              @keydown.enter.space.prevent="form.feedback_mode = 'complete'"
              @keydown.left.up.prevent="selectFeedbackWithFocus('rounds_5')"
              @keydown.right.down.prevent="selectFeedbackWithFocus('rounds_5')"
            >
              <el-icon :size="24" color="#409eff"><Document /></el-icon>
              <div class="feedback-card-title">{{ $t('setup.feedbackComplete') }}</div>
              <div class="feedback-card-desc">{{ $t('setup.feedbackCompleteDesc') }}</div>
            </div>
            <div
              ref="feedbackRoundsRef"
              class="feedback-card"
              :class="{ active: form.feedback_mode === 'rounds_5' }"
              role="radio"
              :tabindex="form.feedback_mode === 'rounds_5' ? 0 : -1"
              :aria-checked="form.feedback_mode === 'rounds_5'"
              @click="form.feedback_mode = 'rounds_5'"
              @keydown.enter.space.prevent="form.feedback_mode = 'rounds_5'"
              @keydown.left.up.prevent="selectFeedbackWithFocus('complete')"
              @keydown.right.down.prevent="selectFeedbackWithFocus('complete')"
            >
              <el-icon :size="24" color="#67c23a"><Timer /></el-icon>
              <div class="feedback-card-title">{{ $t('setup.feedbackRounds5') }}</div>
              <div class="feedback-card-desc">{{ $t('setup.feedbackRounds5Desc') }}</div>
            </div>
          </div>
        </el-form-item>
        </template>

        <el-button
          type="primary"
          size="large"
          :loading="loading"
          @click="startConversation"
          class="start-btn"
        >
          {{ $t('setup.startBtn') }}
        </el-button>

        <div class="logout-link" role="button" tabindex="0" @click="handleLogout" @keydown.enter.space.prevent="handleLogout">{{ $t('setup.logout') }}</div>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { User, Monitor, Timer, Document } from '@element-plus/icons-vue'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { useSessionStore } from '@/stores/session'
import LocaleSwitcher from '@/components/LocaleSwitcher.vue'

const router = useRouter()
const userStore = useUserStore()
const sessionStore = useSessionStore()
const { t, locale } = useI18n()
const loading = ref(false)
const formRef = ref(null)
const modeUserL2Ref = ref(null)
const modeLLML2Ref = ref(null)
const feedbackCompleteRef = ref(null)
const feedbackRoundsRef = ref(null)
const personaCountries = ref([])
const personaCountriesLoading = ref(false)
let personaCountriesRequest = 0

const form = ref({
  mode: 'user_l2',
  target_language: 'EN',
  llm_country: '',
  relationship_type: '',
  topic: '',
  feedback_mode: 'complete',
  ai_suggestions_enabled: true
})

const isZh = computed(() => locale.value === 'zh-CN')

const selectModeWithFocus = async (mode) => {
  form.value.mode = mode
  await nextTick()
  const target = mode === 'user_l2' ? modeUserL2Ref.value : modeLLML2Ref.value
  target?.focus()
}

const selectFeedbackWithFocus = async (feedbackMode) => {
  form.value.feedback_mode = feedbackMode
  await nextTick()
  const target = feedbackMode === 'complete' ? feedbackCompleteRef.value : feedbackRoundsRef.value
  target?.focus()
}

const languages = computed(() => [
  { value: 'EN', label: isZh.value ? '英语 (English)' : 'English' },
  { value: 'ZH', label: isZh.value ? '中文 (Chinese)' : 'Chinese' },
  { value: 'JP', label: isZh.value ? '日语 (Japanese)' : 'Japanese' },
  { value: 'KR', label: isZh.value ? '韩语 (Korean)' : 'Korean' },
  { value: 'FR', label: isZh.value ? '法语 (French)' : 'French' },
  { value: 'DE', label: isZh.value ? '德语 (German)' : 'German' },
])

const relationships = computed(() => [
  { value: t('setup.relStranger'), label: t('setup.relStranger') },
  { value: t('setup.relClassmate'), label: t('setup.relClassmate') },
  { value: t('setup.relColleague'), label: t('setup.relColleague') },
  { value: t('setup.relFriend'), label: t('setup.relFriend') },
  { value: t('setup.relTeacherStudent'), label: t('setup.relTeacherStudent') },
])

const topics = computed(() => [
  { value: t('setup.topicDaily'), label: t('setup.topicDaily') },
  { value: t('setup.topicAcademic'), label: t('setup.topicAcademic') },
  { value: t('setup.topicBusiness'), label: t('setup.topicBusiness') },
  { value: t('setup.topicTravel'), label: t('setup.topicTravel') },
  { value: t('setup.topicCulture'), label: t('setup.topicCulture') },
])

const rules = {
  mode: [{ required: true, message: t('setup.modeRequired') }],
  target_language: [{ required: true, message: t('setup.languageRequired'), trigger: 'change' }],
  llm_country: [{ required: true, message: t('setup.personaCountryRequired'), trigger: 'change' }],
  relationship_type: [{ required: true, message: t('setup.relationshipRequired'), trigger: 'change' }],
  topic: [{ required: true, message: t('setup.topicRequired'), trigger: 'change' }],
  feedback_mode: [{ required: true, message: t('setup.feedbackRequired') }],
}

const refreshPersonaCountries = async () => {
  const requestId = ++personaCountriesRequest
  personaCountriesLoading.value = true
  try {
    const data = await api.getPersonaCountries(form.value.mode, form.value.target_language)
    if (requestId !== personaCountriesRequest) return
    const countries = Array.isArray(data.countries) ? data.countries : []
    personaCountries.value = countries
    if (!countries.some(country => country.code === form.value.llm_country)) {
      form.value.llm_country = countries[0]?.code || ''
    }
  } catch (error) {
    if (requestId !== personaCountriesRequest) return
    personaCountries.value = []
    form.value.llm_country = ''
    if (!error?.pfchatNotified) ElMessage.error(t('setup.personaCountriesLoadFailed'))
  } finally {
    if (requestId === personaCountriesRequest) personaCountriesLoading.value = false
  }
}

watch(
  () => [form.value.mode, form.value.target_language],
  refreshPersonaCountries,
  { immediate: true }
)

onMounted(async () => {
  loading.value = true
  try {
    const data = await api.getActiveSession()
    if (data.session) {
      sessionStore.setSession(data.session)
      sessionStore.setMessages([])
      await router.replace('/chat')
    }
  } catch (error) {
    console.error('Restore active session from setup error:', error)
  } finally {
    loading.value = false
  }
})

const startConversation = async () => {
  if (loading.value) return
  try {
    await formRef.value.validate()
    loading.value = true

    // 获取bot信息
    const botUser = await api.getLLMBotInfo()
    sessionStore.setBotUser(botUser)

    // 创建会话
    const session = await api.createSession(form.value)
    sessionStore.setSession(session)
    sessionStore.setMessages([])

    router.push('/chat')
  } catch (error) {
    if (error?.message) {
      console.error('Create session error:', error)
    }
  } finally {
    loading.value = false
  }
}

const handleLogout = () => {
  userStore.logout()
  sessionStore.clearSession()
  router.push('/login')
}
</script>

<style scoped>
.setup-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
  position: relative;
}

.locale-switcher-wrapper {
  position: absolute;
  top: 30px;
  right: 30px;
  z-index: 100;
}

.setup-box {
  background: white;
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  padding: 40px;
  width: 520px;
  max-width: 100%;
}

.setup-header {
  text-align: center;
  margin-bottom: 32px;
}

.setup-header h1 {
  font-size: 26px;
  color: #303133;
  margin: 0 0 8px;
}

.setup-header p {
  color: #909399;
  font-size: 14px;
  margin: 0;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12px;
  margin-top: 4px;
}

.mode-cards {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  width: 100%;
}

.mode-card {
  border: 2px solid #dcdfe6;
  border-radius: 12px;
  padding: 20px 16px;
  cursor: pointer;
  text-align: center;
  transition: all 0.2s;
  background: white;
}

.mode-card:hover {
  border-color: #409eff;
  background: #f0f7ff;
}

.mode-card.active {
  border-color: #409eff;
  background: #ecf5ff;
}

.mode-card-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin: 10px 0 6px;
}

.mode-card-desc {
  font-size: 12px;
  color: #606266;
  line-height: 1.5;
}

.feedback-cards {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  width: 100%;
}

.ai-suggestions-switch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 16px;
  margin-bottom: 20px;
  border: 1px solid #dcdfe6;
  border-radius: 10px;
  background: #fafafa;
}

.ai-suggestions-label {
  color: #303133;
  font-size: 14px;
  font-weight: 600;
}

.ai-suggestions-desc {
  margin-top: 5px;
  color: #606266;
  font-size: 12px;
  line-height: 1.5;
}

.feedback-method-title {
  margin-top: 0;
}

.role-cards {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  width: 100%;
  max-height: 360px;
  overflow-y: auto;
  padding-right: 4px;
}

.role-card {
  border: 2px solid #dcdfe6;
  border-radius: 10px;
  padding: 12px 14px;
  cursor: pointer;
  transition: all 0.2s;
  background: white;
}

.role-card:hover {
  border-color: #409eff;
  background: #f0f7ff;
}

.role-card.active {
  border-color: #409eff;
  background: #ecf5ff;
}

.role-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.role-card-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.role-card-meta {
  margin-top: 3px;
  font-size: 12px;
  color: #909399;
}

.role-card-desc {
  margin-top: 8px;
  font-size: 12px;
  color: #606266;
  line-height: 1.5;
}

.feedback-card {
  border: 2px solid #dcdfe6;
  border-radius: 10px;
  padding: 14px 12px;
  cursor: pointer;
  text-align: center;
  transition: all 0.2s;
}

.feedback-card:hover {
  border-color: #409eff;
  background: #f0f7ff;
}

.feedback-card.active {
  border-color: #409eff;
  background: #ecf5ff;
}

.feedback-card-title {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
  margin: 8px 0 4px;
}

.feedback-card-desc {
  font-size: 12px;
  color: #606266;
  line-height: 1.5;
}

.full-width {
  width: 100%;
}

.persona-hint {
  width: 100%;
  margin-top: 6px;
  color: #606266;
  font-size: 12px;
  line-height: 1.5;
}

.start-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
  margin-top: 8px;
}

.logout-link {
  text-align: center;
  margin-top: 16px;
  color: #909399;
  font-size: 13px;
  cursor: pointer;
}

.logout-link:hover {
  color: #409eff;
}

:deep(.el-form-item) {
  margin-bottom: 20px;
}

:deep(.el-form-item__label) {
  font-weight: 500;
}

@media (max-width: 560px) {
  .setup-container {
    align-items: flex-start;
    padding: 12px;
  }

  .locale-switcher-wrapper {
    top: 16px;
    right: 16px;
  }

  .setup-box {
    padding: 24px 18px 18px;
    border-radius: 12px;
  }

  .setup-header {
    margin-bottom: 24px;
    padding-right: 44px;
  }

  .mode-cards,
  .feedback-cards,
  .role-cards {
    grid-template-columns: 1fr;
  }

  .start-btn {
    position: sticky;
    bottom: 8px;
    z-index: 5;
    box-shadow: 0 8px 20px rgba(64, 158, 255, 0.3);
  }
}

@media (max-height: 760px) and (min-width: 561px) {
  .setup-container {
    align-items: flex-start;
  }

  .setup-box {
    padding: 28px 36px;
  }

  .setup-header {
    margin-bottom: 20px;
  }

  .mode-card {
    padding: 14px 12px;
  }

  :deep(.el-form-item) {
    margin-bottom: 14px;
  }
}
</style>
