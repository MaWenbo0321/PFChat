<template>
  <div class="login-container">
    <!-- 语言切换按钮 - 右上角 -->
    <div class="locale-switcher-wrapper">
      <LocaleSwitcher />
    </div>

    <div class="login-box">
      <h1 class="title">{{ $t('login.title') }}</h1>

      <el-tabs v-model="activeTab" class="login-tabs">
        <!-- 登录标签页 -->
        <el-tab-pane :label="$t('login.loginTab')" name="login">
          <el-form :model="loginForm" :rules="loginRules" ref="loginFormRef">
            <el-form-item prop="username">
              <el-input
                  v-model="loginForm.username"
                  :placeholder="$t('login.username')"
                  :aria-label="$t('login.username')"
                  :prefix-icon="User"
                  size="large"
              />
            </el-form-item>
            <el-form-item prop="password">
              <el-input
                  v-model="loginForm.password"
                  type="password"
                  :placeholder="$t('login.password')"
                  :aria-label="$t('login.password')"
                  :prefix-icon="Lock"
                  size="large"
                  show-password
                  @keyup.enter="handleLogin"
              />
            </el-form-item>
            <el-button
                type="primary"
                size="large"
                :loading="loading"
                @click="handleLogin"
                class="submit-btn"
            >
              {{ $t('login.loginBtn') }}
            </el-button>
          </el-form>
        </el-tab-pane>

        <!-- 注册标签页 -->
        <el-tab-pane :label="$t('login.registerTab')" name="register">
          <el-form :model="registerForm" :rules="registerRules" ref="registerFormRef">
            <el-form-item prop="username">
              <el-input
                  v-model="registerForm.username"
                  :placeholder="$t('login.usernamePlaceholder')"
                  :aria-label="$t('login.username')"
                  :prefix-icon="User"
                  size="large"
              />
            </el-form-item>
            <el-form-item prop="password">
              <el-input
                  v-model="registerForm.password"
                  type="password"
                  :placeholder="$t('login.passwordPlaceholder')"
                  :aria-label="$t('login.password')"
                  :prefix-icon="Lock"
                  size="large"
                  show-password
              />
            </el-form-item>
            <el-form-item prop="country">
              <el-select
                  v-model="registerForm.country"
                  :placeholder="$t('login.selectCountry')"
                  :aria-label="$t('login.selectCountry')"
                  size="large"
                  class="country-select"
              >
                <el-option :label="$t('countries.CN')" value="CN" />
                <el-option :label="$t('countries.US')" value="US" />
                <el-option :label="$t('countries.GB')" value="GB" />
                <el-option :label="$t('countries.JP')" value="JP" />
                <el-option :label="$t('countries.KR')" value="KR" />
                <el-option :label="$t('countries.FR')" value="FR" />
                <el-option :label="$t('countries.DE')" value="DE" />
                <el-option :label="$t('countries.CA')" value="CA" />
                <el-option :label="$t('countries.AU')" value="AU" />
                <el-option :label="$t('countries.OTHER')" value="OTHER" />
              </el-select>
            </el-form-item>

            <!-- 隐私提示 -->
            <div class="privacy-notice">
              <el-icon color="#e6a23c" :size="16"><Warning /></el-icon>
              <span>{{ $t('login.privacyNotice') }}</span>
            </div>

            <el-button
                type="primary"
                size="large"
                :loading="loading"
                @click="handleRegister"
                class="submit-btn"
            >
              {{ $t('login.registerBtn') }}
            </el-button>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Warning, User, Lock } from '@element-plus/icons-vue'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { useSessionStore } from '@/stores/session'
import { getLocaleByCountry } from '@/utils/locale'
import LocaleSwitcher from '@/components/LocaleSwitcher.vue'

const router = useRouter()
const userStore = useUserStore()
const sessionStore = useSessionStore()
const { t, locale } = useI18n()

const activeTab = ref('login')
const loading = ref(false)

const loginFormRef = ref(null)
const registerFormRef = ref(null)

const loginForm = reactive({
  username: '',
  password: ''
})

const registerForm = reactive({
  username: '',
  password: '',
  country: ''
})

const loginRules = {
  username: [{ required: true, message: t('login.usernameRequired'), trigger: 'blur' }],
  password: [{ required: true, message: t('login.passwordRequired'), trigger: 'blur' }]
}

const registerRules = {
  username: [
    { required: true, message: t('login.usernameRequired'), trigger: 'blur' },
    { min: 2, max: 20, message: t('login.usernameLength'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('login.passwordRequired'), trigger: 'blur' },
    { min: 6, message: t('login.passwordLength'), trigger: 'blur' }
  ],
  country: [{ required: true, message: t('login.countryRequired'), trigger: 'change' }]
}

// 监听国家选择,自动切换语言
watch(() => registerForm.country, (newCountry) => {
  if (newCountry) {
    const newLocale = getLocaleByCountry(newCountry)
    locale.value = newLocale
    localStorage.setItem('locale', newLocale)
  }
})

const handleLogin = async () => {
  try {
    await loginFormRef.value.validate()
    loading.value = true

    const response = await api.login(loginForm)
    sessionStore.clearSession()
    userStore.setToken(response.token)
    userStore.setUserInfo(response.user)

    ElMessage.success(t('login.loginSuccess'))
    await router.push('/setup')
  } catch (error) {
    console.error('Login error:', error)
  } finally {
    loading.value = false
  }
}

const handleRegister = async () => {
  try {
    await registerFormRef.value.validate()

    // 显示隐私声明
    await ElMessageBox.confirm(
        '',
        t('privacy.title'),
        {
          confirmButtonText: t('privacy.agree'),
          cancelButtonText: t('privacy.cancel'),
          type: 'warning',
          customClass: 'privacy-dialog',
          dangerouslyUseHTMLString: true,
          message: `
            <div class="privacy-content">
              <p class="privacy-notice">${t('privacy.notice')}</p>
              <div class="privacy-details">
                <h4>${t('privacy.dataUsage')}</h4>
                <ul>
                  <li>${t('privacy.point1')}</li>
                  <li>${t('privacy.point2')}</li>
                  <li>${t('privacy.point3')}</li>
                </ul>
                <p class="privacy-agreement">${t('privacy.agreement')}</p>
              </div>
            </div>
          `
        }
    )

    loading.value = true
    const response = await api.register(registerForm)

    sessionStore.clearSession()
    userStore.setToken(response.token)
    userStore.setUserInfo(response.user)

    ElMessage.success(t('login.registerSuccess'))
    await router.push('/setup')
  } catch (error) {
    if (error === 'cancel') {
      ElMessage.info(t('login.registerCanceled'))
    } else {
      console.error('Register error:', error)
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  position: relative;
}

/* 语言切换按钮容器 - 固定在右上角 */
.locale-switcher-wrapper {
  position: absolute;
  top: 30px;
  right: 30px;
  z-index: 100;
}

.login-box {
  background: white;
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  padding: 40px;
  width: 420px;
}

.title {
  text-align: center;
  margin-bottom: 30px;
  font-size: 28px;
  color: #303133;
  font-weight: 600;
}

.login-tabs {
  margin-bottom: 20px;
}

.country-select {
  width: 100%;
}

.privacy-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: #fef0f0;
  border-radius: 4px;
  margin-bottom: 20px;
  font-size: 13px;
  color: #e6a23c;
}

.submit-btn {
  width: 100%;
  height: 44px;
  font-size: 16px;
  font-weight: 600;
}

:deep(.el-tabs__item) {
  font-size: 16px;
  font-weight: 500;
}

:deep(.el-form-item) {
  margin-bottom: 20px;
}

:deep(.privacy-dialog) {
  width: 500px;
}

:deep(.privacy-content) {
  text-align: left;
  padding: 0 10px;
}

:deep(.privacy-content .privacy-notice) {
  color: #e6a23c;
  font-weight: 600;
  margin-bottom: 15px;
  padding: 10px;
  background: #fdf6ec;
  border-radius: 4px;
}

:deep(.privacy-content .privacy-details) {
  margin-top: 15px;
}

:deep(.privacy-content h4) {
  color: #303133;
  margin-bottom: 10px;
}

:deep(.privacy-content ul) {
  margin: 10px 0;
  padding-left: 25px;
}

:deep(.privacy-content li) {
  margin: 8px 0;
  color: #606266;
  line-height: 1.5;
}

:deep(.privacy-content .privacy-agreement) {
  margin-top: 15px;
  padding: 10px;
  background: #f4f4f5;
  border-radius: 4px;
  color: #606266;
  font-size: 14px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .locale-switcher-wrapper {
    top: 20px;
    right: 20px;
  }

  .login-box {
    width: 90%;
    max-width: 420px;
    padding: 30px 20px;
  }
}
</style>
