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
          <div class="mode-cards">
            <div
              class="mode-card"
              :class="{ active: form.mode === 'user_l2' }"
              @click="form.mode = 'user_l2'"
            >
              <el-icon :size="32" color="#409eff"><User /></el-icon>
              <div class="mode-card-title">{{ $t('setup.modeUserL2Title') }}</div>
              <div class="mode-card-desc">{{ $t('setup.modeUserL2Desc') }}</div>
            </div>
            <div
              class="mode-card"
              :class="{ active: form.mode === 'llm_l2' }"
              @click="form.mode = 'llm_l2'"
            >
              <el-icon :size="32" color="#67c23a"><Monitor /></el-icon>
              <div class="mode-card-title">{{ $t('setup.modeLLML2Title') }}</div>
              <div class="mode-card-desc">{{ $t('setup.modeLLML2Desc') }}</div>
            </div>
          </div>
        </el-form-item>

        <!-- LLM角色 -->
        <div class="section-title">{{ $t('setup.llmRoleTitle') }}</div>
        <el-form-item prop="llm_role_id">
          <div class="role-cards">
            <div
              v-for="role in availableRoles"
              :key="role.id"
              class="role-card"
              :class="{ active: form.llm_role_id === role.id }"
              @click="form.llm_role_id = role.id"
            >
              <div class="role-card-header">
                <div>
                  <div class="role-card-title">{{ role.name }}</div>
                  <div class="role-card-meta">{{ role.age }} · {{ role.gender }} · {{ role.culture }}</div>
                </div>
                <el-tag size="small" effect="plain">{{ role.nativeLanguage }}</el-tag>
              </div>
              <div class="role-card-desc">{{ role.personality }}</div>
            </div>
          </div>
        </el-form-item>

        <!-- 目标语言 -->
        <el-form-item :label="$t('setup.targetLanguage')" prop="target_language">
          <el-select v-model="form.target_language" :placeholder="$t('setup.selectLanguage')" size="large" class="full-width">
            <el-option v-for="lang in languages" :key="lang.value" :label="lang.label" :value="lang.value" />
          </el-select>
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
          <div class="feedback-cards">
            <div
              class="feedback-card"
              :class="{ active: form.feedback_mode === 'complete' }"
              @click="form.feedback_mode = 'complete'"
            >
              <el-icon :size="24" color="#409eff"><Document /></el-icon>
              <div class="feedback-card-title">{{ $t('setup.feedbackComplete') }}</div>
              <div class="feedback-card-desc">{{ $t('setup.feedbackCompleteDesc') }}</div>
            </div>
            <div
              class="feedback-card"
              :class="{ active: form.feedback_mode === 'rounds_5' }"
              @click="form.feedback_mode = 'rounds_5'"
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

        <div class="logout-link" @click="handleLogout">{{ $t('setup.logout') }}</div>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
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

const form = ref({
  mode: 'user_l2',
  target_language: 'EN',
  llm_role_id: 'aiko',
  relationship_type: '',
  topic: '',
  feedback_mode: 'complete',
  ai_suggestions_enabled: true
})

const isZh = computed(() => locale.value === 'zh-CN')

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

const roleProfiles = computed(() => [
  {
    id: 'aiko',
    age: 24,
    gender: t('setup.roleGenderFemale'),
    country: 'JP',
    nativeLanguage: getLanguageNameByCountry('JP'),
    name: t('setup.roleAikoName'),
    personality: t('setup.roleAikoDesc')
  },
  {
    id: 'minji',
    age: 20,
    gender: t('setup.roleGenderFemale'),
    country: 'KR',
    nativeLanguage: getLanguageNameByCountry('KR'),
    name: t('setup.roleMinjiName'),
    personality: t('setup.roleMinjiDesc')
  },
  {
    id: 'haruto',
    age: 19,
    gender: t('setup.roleGenderMale'),
    country: 'JP',
    nativeLanguage: getLanguageNameByCountry('JP'),
    name: t('setup.roleHarutoName'),
    personality: t('setup.roleHarutoDesc')
  },
  {
    id: 'enkhjin',
    age: 23,
    gender: t('setup.roleGenderFemale'),
    country: 'MN',
    nativeLanguage: getLanguageNameByCountry('MN'),
    name: t('setup.roleEnkhjinName'),
    personality: t('setup.roleEnkhjinDesc')
  },
  {
    id: 'xiayu',
    age: 21,
    gender: t('setup.roleGenderFemale'),
    country: 'CN',
    nativeLanguage: getLanguageNameByCountry('CN'),
    name: t('setup.roleXiayuName'),
    personality: t('setup.roleXiayuDesc')
  },
  {
    id: 'nurul',
    age: 25,
    gender: t('setup.roleGenderFemale'),
    country: 'MY',
    nativeLanguage: getLanguageNameByCountry('MY'),
    name: t('setup.roleNurulName'),
    personality: t('setup.roleNurulDesc')
  },
  {
    id: 'cheryl',
    age: 28,
    gender: t('setup.roleGenderFemale'),
    country: 'SG',
    nativeLanguage: getLanguageNameByCountry('SG'),
    name: t('setup.roleCherylName'),
    personality: t('setup.roleCherylDesc')
  },
  {
    id: 'marcus',
    age: 34,
    gender: t('setup.roleGenderMale'),
    country: 'DE',
    nativeLanguage: getLanguageNameByCountry('DE'),
    name: t('setup.roleMarcusName'),
    personality: t('setup.roleMarcusDesc')
  },
  {
    id: 'sofia',
    age: 29,
    gender: t('setup.roleGenderFemale'),
    country: 'FR',
    nativeLanguage: getLanguageNameByCountry('FR'),
    name: t('setup.roleSofiaName'),
    personality: t('setup.roleSofiaDesc')
  },
  {
    id: 'daniel',
    age: 42,
    gender: t('setup.roleGenderMale'),
    country: 'US',
    nativeLanguage: getLanguageNameByCountry('US'),
    name: t('setup.roleDanielName'),
    personality: t('setup.roleDanielDesc')
  },
  {
    id: 'amara',
    age: 31,
    gender: t('setup.roleGenderFemale'),
    country: 'NG',
    nativeLanguage: getLanguageNameByCountry('NG'),
    name: t('setup.roleAmaraName'),
    personality: t('setup.roleAmaraDesc')
  },
  {
    id: 'joao',
    age: 27,
    gender: t('setup.roleGenderMale'),
    country: 'BR',
    nativeLanguage: getLanguageNameByCountry('BR'),
    name: t('setup.roleJoaoName'),
    personality: t('setup.roleJoaoDesc')
  },
  {
    id: 'mia',
    age: 38,
    gender: t('setup.roleGenderFemale'),
    country: 'AU',
    nativeLanguage: getLanguageNameByCountry('AU'),
    name: t('setup.roleMiaName'),
    personality: t('setup.roleMiaDesc')
  },
  {
    id: 'thabo',
    age: 45,
    gender: t('setup.roleGenderMale'),
    country: 'ZA',
    nativeLanguage: getLanguageNameByCountry('ZA'),
    name: t('setup.roleThaboName'),
    personality: t('setup.roleThaboDesc')
  },
  {
    id: 'priya',
    age: 33,
    gender: t('setup.roleGenderFemale'),
    country: 'IN',
    nativeLanguage: getLanguageNameByCountry('IN'),
    name: t('setup.rolePriyaName'),
    personality: t('setup.rolePriyaDesc')
  },
  {
    id: 'lucia',
    age: 22,
    gender: t('setup.roleGenderFemale'),
    country: 'MX',
    nativeLanguage: getLanguageNameByCountry('MX'),
    name: t('setup.roleLuciaName'),
    personality: t('setup.roleLuciaDesc')
  }
])

const availableRoles = computed(() => {
  const roles = roleProfiles.value.filter(role => {
    if (form.value.mode !== 'llm_l2') return true
    return role.country !== userStore.userInfo?.country &&
      role.country !== targetLanguageToCountry(form.value.target_language)
  })
  return roles.map(role => ({
    ...role,
    culture: getCountryLabel(role.country)
  }))
})

const rules = {
  mode: [{ required: true, message: t('setup.modeRequired') }],
  target_language: [{ required: true, message: t('setup.languageRequired'), trigger: 'change' }],
  llm_role_id: [{ required: true, message: t('setup.llmRoleRequired'), trigger: 'change' }],
  relationship_type: [{ required: true, message: t('setup.relationshipRequired'), trigger: 'change' }],
  topic: [{ required: true, message: t('setup.topicRequired'), trigger: 'change' }],
  feedback_mode: [{ required: true, message: t('setup.feedbackRequired') }],
}

const getLanguageNameByCountry = (country) => {
  const map = { CN: '中文', TW: '中文', HK: '中文', SG: 'English/Mandarin', MY: 'Malay/English/Chinese', JP: '日本語', KR: '한국어', MN: 'Mongolian', FR: 'Français', DE: 'Deutsch', US: 'English', GB: 'English', CA: 'English', AU: 'English', NG: 'English + local languages', BR: 'Português', ZA: 'English + local languages', IN: 'Hindi/English', MX: 'Español' }
  return map[country] || 'English'
}

const getCountryLabel = (country) => {
  return t(`countries.${country}`)
}

const targetLanguageToCountry = (lang) => {
  const map = { EN: 'US', ZH: 'CN', JP: 'JP', KR: 'KR', FR: 'FR', DE: 'DE' }
  return map[lang] || 'US'
}

watch(availableRoles, (roles) => {
  if (roles.length === 0) return
  if (!roles.some(role => role.id === form.value.llm_role_id)) {
    form.value.llm_role_id = roles[0].id
  }
}, { immediate: true })

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
  color: #909399;
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
  color: #909399;
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

@media (max-width: 560px) {
  .role-cards {
    grid-template-columns: 1fr;
  }
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
  color: #909399;
  line-height: 1.5;
}

.full-width {
  width: 100%;
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
</style>
