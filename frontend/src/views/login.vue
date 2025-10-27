<template>
  <div class="login-container">
    <div class="login-box">
      <h1 class="title">即时通讯系统</h1>

      <el-tabs v-model="activeTab" class="login-tabs">
        <el-tab-pane label="登录" name="login">
          <el-form :model="loginForm" :rules="loginRules" ref="loginFormRef">
            <el-form-item prop="username">
              <el-input
                  v-model="loginForm.username"
                  placeholder="用户名"
                  prefix-icon="User"
                  size="large"
              />
            </el-form-item>
            <el-form-item prop="password">
              <el-input
                  v-model="loginForm.password"
                  type="password"
                  placeholder="密码"
                  prefix-icon="Lock"
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
              登录
            </el-button>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="注册" name="register">
          <el-form :model="registerForm" :rules="registerRules" ref="registerFormRef">
            <el-form-item prop="username">
              <el-input
                  v-model="registerForm.username"
                  placeholder="用户名 (3-20个字符)"
                  prefix-icon="User"
                  size="large"
              />
            </el-form-item>
            <el-form-item prop="password">
              <el-input
                  v-model="registerForm.password"
                  type="password"
                  placeholder="密码 (至少6个字符)"
                  prefix-icon="Lock"
                  size="large"
                  show-password
              />
            </el-form-item>
            <el-form-item prop="country">
              <el-select
                  v-model="registerForm.country"
                  placeholder="请选择国家"
                  size="large"
                  class="country-select"
              >
                <el-option label="中国 (China)" value="CN" />
                <el-option label="美国 (USA)" value="US" />
                <el-option label="英国 (UK)" value="GB" />
                <el-option label="日本 (Japan)" value="JP" />
                <el-option label="韩国 (Korea)" value="KR" />
                <el-option label="法国 (France)" value="FR" />
                <el-option label="德国 (Germany)" value="DE" />
                <el-option label="加拿大 (Canada)" value="CA" />
                <el-option label="澳大利亚 (Australia)" value="AU" />
                <el-option label="其他 (Other)" value="OTHER" />
              </el-select>
            </el-form-item>

            <!-- 隐私提示 -->
            <div class="privacy-notice">
              <el-icon color="#e6a23c" :size="16"><Warning /></el-icon>
              <span>注册即表示同意数据用于实验目的</span>
            </div>

            <el-button
                type="primary"
                size="large"
                :loading="loading"
                @click="handleRegister"
                class="submit-btn"
            >
              注册
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
import { Warning } from '@element-plus/icons-vue'
import api from '@/api'
import { useUserStore } from '@/stores/user'
import { autoSwitchLocaleByCountry } from '@/utils/locale'

const router = useRouter()
const userStore = useUserStore()
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
  username: [{ required: true, message: () => t('login.usernameRequired'), trigger: 'blur' }],
  password: [{ required: true, message: () => t('login.passwordRequired'), trigger: 'blur' }]
}

const registerRules = {
  username: [
    { required: true, message: () => t('login.usernameRequired'), trigger: 'blur' },
    { min: 3, max: 20, message: () => t('login.usernameLength'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: () => t('login.passwordRequired'), trigger: 'blur' },
    { min: 6, message: () => t('login.passwordLength'), trigger: 'blur' }
  ],
  country: [{ required: true, message: () => t('login.countryRequired'), trigger: 'change' }]
}

// 监听国家选择，自动切换语言
watch(() => registerForm.country, (newCountry) => {
  if (newCountry) {
    autoSwitchLocaleByCountry(newCountry, { global: { locale } })
  }
})

const handleLogin = async () => {
  const valid = await loginFormRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    const res = await api.login(loginForm)
    userStore.setToken(res.token)
    userStore.setUserInfo(res.user)

    // 根据用户国家设置语言
    if (res.user.country) {
      autoSwitchLocaleByCountry(res.user.country, { global: { locale } })
    }

    ElMessage.success(t('login.loginSuccess'))
    router.push('/')
  } catch (error) {
    console.error('Login error:', error)
  } finally {
    loading.value = false
  }
}

const handleRegister = async () => {
  const valid = await registerFormRef.value.validate().catch(() => false)
  if (!valid) return

  // 显示隐私提示弹窗
  try {
    await ElMessageBox.confirm(
        '',
        t('privacy.title'),
        {
          confirmButtonText: t('privacy.agree'),
          cancelButtonText: t('privacy.cancel'),
          type: 'warning',
          center: false,
          customClass: 'privacy-notice-box',
          dangerouslyUseHTMLString: true,
          message: `
          <div class="privacy-content">
            <div class="privacy-icon">
              <svg viewBox="0 0 1024 1024" width="48" height="48">
                <path d="M512 64C264.6 64 64 264.6 64 512s200.6 448 448 448 448-200.6 448-448S759.4 64 512 64zm-32 232c0-4.4 3.6-8 8-8h48c4.4 0 8 3.6 8 8v272c0 4.4-3.6 8-8 8h-48c-4.4 0-8-3.6-8-8V296zm32 440a48.01 48.01 0 0 1 0-96 48.01 48.01 0 0 1 0 96z" fill="#e6a23c"/>
              </svg>
            </div>

            <div class="privacy-title">${t('privacy.dataUsage')}</div>

            <div class="privacy-text">
              <p>${t('privacy.notice')}</p>

              <div class="privacy-points">
                <div class="privacy-point">
                  <span class="point-icon">📝</span>
                  <span class="point-text">${t('privacy.point1')}</span>
                </div>
                <div class="privacy-point">
                  <span class="point-icon">🔬</span>
                  <span class="point-text">${t('privacy.point2')}</span>
                </div>
                <div class="privacy-point">
                  <span class="point-icon">🔒</span>
                  <span class="point-text">${t('privacy.point3')}</span>
                </div>
              </div>

              <p style="margin-top: 15px; font-size: 14px; color: #909399;">
                ${t('privacy.agreement')}
              </p>
            </div>
          </div>
        `
        }
    )
  } catch (error) {
    ElMessage.info(t('login.registerCanceled'))
    return
  }

  loading.value = true
  try {
    const res = await api.register(registerForm)
    userStore.setToken(res.token)
    userStore.setUserInfo(res.user)
    ElMessage.success(t('login.registerSuccess'))
    router.push('/')
  } catch (error) {
    console.error('Register error:', error)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  width: 100vw;
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-box {
  width: 420px;
  padding: 40px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
}

.title {
  text-align: center;
  margin-bottom: 30px;
  color: #303133;
  font-size: 28px;
}

.login-tabs {
  margin-top: 20px;
}

.submit-btn {
  width: 100%;
  margin-top: 10px;
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
  border: 1px solid #fde2e2;
  border-radius: 6px;
  margin-bottom: 15px;
  font-size: 13px;
  color: #e6a23c;
}

/* 隐私提示弹窗样式 */
:deep(.privacy-notice-box) {
  width: 550px;
  border-radius: 12px;
}

:deep(.privacy-notice-box .el-message-box__header) {
  padding: 25px 30px 10px;
  border-bottom: 1px solid #ebeef5;
}

:deep(.privacy-notice-box .el-message-box__title) {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

:deep(.privacy-notice-box .el-message-box__content) {
  padding: 0;
}

:deep(.privacy-notice-box .el-message-box__btns) {
  padding: 20px 30px 25px;
  border-top: 1px solid #ebeef5;
}

:deep(.privacy-notice-box .el-button) {
  padding: 12px 35px;
  font-size: 15px;
}

:deep(.privacy-notice-box .el-button--primary) {
  background: #e6a23c;
  border-color: #e6a23c;
}

:deep(.privacy-notice-box .el-button--primary:hover) {
  background: #ebb563;
  border-color: #ebb563;
}

:deep(.privacy-content) {
  padding: 25px 30px;
}

:deep(.privacy-icon) {
  text-align: center;
  margin-bottom: 20px;
}

:deep(.privacy-title) {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
  text-align: center;
  margin-bottom: 20px;
}

:deep(.privacy-text) {
  font-size: 15px;
  line-height: 1.8;
  color: #606266;
}

:deep(.privacy-text p) {
  margin: 0 0 15px 0;
}

:deep(.privacy-points) {
  background: #f5f7fa;
  padding: 15px;
  border-radius: 8px;
  margin: 15px 0;
}

:deep(.privacy-point) {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 12px;
  font-size: 14px;
  line-height: 1.6;
}

:deep(.privacy-point:last-child) {
  margin-bottom: 0;
}

:deep(.point-icon) {
  font-size: 18px;
  flex-shrink: 0;
}

:deep(.point-text) {
  flex: 1;
  color: #606266;
}

:deep(.privacy-text strong) {
  color: #e6a23c;
  font-weight: 600;
}
</style>