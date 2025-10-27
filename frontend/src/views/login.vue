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
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '@/api'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

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
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const registerRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名长度应为 3-20 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 个字符', trigger: 'blur' }
  ],
  country: [{ required: true, message: '请选择国家', trigger: 'change' }]
}

const handleLogin = async () => {
  const valid = await loginFormRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    const res = await api.login(loginForm)
    userStore.setToken(res.token)
    userStore.setUserInfo(res.user)
    ElMessage.success('登录成功')
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

  loading.value = true
  try {
    const res = await api.register(registerForm)
    userStore.setToken(res.token)
    userStore.setUserInfo(res.user)
    ElMessage.success('注册成功')
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
</style>