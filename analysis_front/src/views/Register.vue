<template>
  <div class="auth-container">
    <!-- 背景装饰 -->
    <div class="auth-bg">
      <div class="bg-shape shape-1"></div>
      <div class="bg-shape shape-2"></div>
      <div class="bg-shape shape-3"></div>
    </div>

    <div class="auth-content">
      <!-- 左侧介绍区域 -->
      <div class="auth-intro">
        <div class="intro-content">
          <h1 class="brand-title">加入我们的社区</h1>
          <p class="brand-subtitle">开启您的加密资产投资之旅</p>
          <div class="benefits-list">
            <div class="benefit-item">
              <div class="benefit-icon">🚀</div>
              <span>免费注册，立即开始</span>
            </div>
            <div class="benefit-item">
              <div class="benefit-icon">🔒</div>
              <span>安全可靠的数据保护</span>
            </div>
<!--            <div class="benefit-item">-->
<!--              <div class="benefit-icon">📱</div>-->
<!--              <span>支持多设备同步</span>-->
<!--            </div>-->
<!--            <div class="benefit-item">-->
<!--              <div class="benefit-icon">🎯</div>-->
<!--              <span>个性化投资建议</span>-->
<!--            </div>-->
          </div>
        </div>
      </div>

      <!-- 右侧注册表单 -->
      <div class="auth-form-container">
        <div class="auth-form-card">
          <!-- 表单头部 -->
          <div class="form-header">
            <h2 class="form-title">创建账户</h2>
            <p class="form-subtitle">填写信息完成注册</p>
          </div>

          <!-- 表单主体 -->
          <form class="auth-form" @submit.prevent="submit">
            <!-- 用户名输入框 -->
            <div class="form-group">
              <label class="form-label">
                <svg class="input-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"></path>
                </svg>
                用户名
              </label>
              <input
                v-model="username"
                type="text"
                class="form-input"
                :class="{ 'input-error': fieldErrors.username }"
                placeholder="请输入用户名（2-20字符）"
                autocomplete="username"
                @blur="validateField('username')"
                @input="clearFieldError('username')"
              />
              <div v-if="fieldErrors.username" class="field-error">
                {{ fieldErrors.username }}
              </div>
            </div>

            <!-- 邮箱输入框（可选） -->
            <div class="form-group">
              <label class="form-label">
                <svg class="input-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path>
                </svg>
                邮箱地址 <span class="optional">(可选)</span>
              </label>
              <input
                v-model="email"
                type="email"
                class="form-input"
                :class="{ 'input-error': fieldErrors.email }"
                placeholder="请输入邮箱地址"
                autocomplete="email"
                @blur="validateField('email')"
                @input="clearFieldError('email')"
              />
              <div v-if="fieldErrors.email" class="field-error">
                {{ fieldErrors.email }}
              </div>
            </div>

            <!-- 密码输入框 -->
            <div class="form-group">
              <label class="form-label">
                <svg class="input-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path>
                </svg>
                密码
              </label>
              <div class="password-input-container">
                <input
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  class="form-input password-input"
                  :class="{ 'input-error': fieldErrors.password }"
                  placeholder="请输入密码（至少6位）"
                  autocomplete="new-password"
                  @blur="validateField('password')"
                  @input="handlePasswordInput"
                />
                <button
                  type="button"
                  class="password-toggle"
                  @click="showPassword = !showPassword"
                  :aria-label="showPassword ? '隐藏密码' : '显示密码'"
                >
                  <svg v-if="showPassword" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.878 9.878L3 3m6.878 6.878L21 21"></path>
                  </svg>
                  <svg v-else fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"></path>
                  </svg>
                </button>
              </div>

              <!-- 密码强度指示器 -->
              <div v-if="password" class="password-strength">
                <div class="strength-meter">
                  <div
                    class="strength-bar"
                    :class="passwordStrengthClass"
                    :style="{ width: passwordStrengthPercent + '%' }"
                  ></div>
                </div>
                <span class="strength-text">{{ passwordStrengthText }}</span>
              </div>

              <div v-if="fieldErrors.password" class="field-error">
                {{ fieldErrors.password }}
              </div>
            </div>

            <!-- 确认密码输入框 -->
            <div class="form-group">
              <label class="form-label">
                <svg class="input-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                </svg>
                确认密码
              </label>
              <div class="password-input-container">
                <input
                  v-model="confirm"
                  :type="showConfirmPassword ? 'text' : 'password'"
                  class="form-input password-input"
                  :class="{ 'input-error': fieldErrors.confirm }"
                  placeholder="请再次输入密码"
                  autocomplete="new-password"
                  @blur="validateField('confirm')"
                  @input="clearFieldError('confirm')"
                />
                <button
                  type="button"
                  class="password-toggle"
                  @click="showConfirmPassword = !showConfirmPassword"
                  :aria-label="showConfirmPassword ? '隐藏密码' : '显示密码'"
                >
                  <svg v-if="showConfirmPassword" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.878 9.878L3 3m6.878 6.878L21 21"></path>
                  </svg>
                  <svg v-else fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"></path>
                  </svg>
                </button>
              </div>
              <div v-if="fieldErrors.confirm" class="field-error">
                {{ fieldErrors.confirm }}
              </div>
            </div>

            <!-- 服务条款同意 -->
            <div class="form-group terms-group">
              <label class="checkbox-label">
                <input
                  v-model="agreeToTerms"
                  type="checkbox"
                  class="checkbox-input"
                  @change="validateField('terms')"
                />
                <span class="checkbox-mark"></span>
                <span class="checkbox-text">
                  我已阅读并同意
                  <button type="button" class="link-button" @click="showTerms">《服务条款》</button>
                  和
                  <button type="button" class="link-button" @click="showPrivacy">《隐私政策》</button>
                </span>
              </label>
              <div v-if="fieldErrors.terms" class="field-error">
                {{ fieldErrors.terms }}
              </div>
            </div>

            <!-- 全局错误提示 -->
            <div v-if="err" class="global-error">
              <svg class="error-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
              {{ err }}
            </div>

            <!-- 注册按钮 -->
            <button
              type="submit"
              class="auth-button"
              :disabled="loading || !isFormValid"
              :class="{ 'loading': loading }"
            >
              <span v-if="loading" class="loading-text">
                <svg class="loading-spinner" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
                </svg>
                注册中...
              </span>
              <span v-else>创建账户</span>
            </button>

            <!-- 分割线 -->
            <div class="divider">
              <span class="divider-text">已有账号？</span>
            </div>

            <!-- 登录链接 -->
            <RouterLink to="/login" class="login-link">
              <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"></path>
              </svg>
              <span>返回登录</span>
            </RouterLink>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, computed } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api/api.js'
import { useAuth } from '../stores/auth.js'

const router = useRouter()
const { setAuth } = useAuth()

// 表单数据
const username = ref('')
const email = ref('')
const password = ref('')
const confirm = ref('')
const showPassword = ref(false)
const showConfirmPassword = ref(false)
const agreeToTerms = ref(false)
const loading = ref(false)
const err = ref('')

// 字段错误
const fieldErrors = ref({
  username: '',
  email: '',
  password: '',
  confirm: '',
  terms: ''
})

// 密码强度计算
const passwordStrength = computed(() => {
  const pwd = password.value
  if (!pwd) return 0

  let score = 0

  // 长度检查
  if (pwd.length >= 8) score += 25
  else if (pwd.length >= 6) score += 15

  // 字符类型检查
  if (/[a-z]/.test(pwd)) score += 20  // 小写字母
  if (/[A-Z]/.test(pwd)) score += 20  // 大写字母
  if (/[0-9]/.test(pwd)) score += 15  // 数字
  if (/[^A-Za-z0-9]/.test(pwd)) score += 20  // 特殊字符

  return Math.min(score, 100)
})

const passwordStrengthPercent = computed(() => passwordStrength.value)

const passwordStrengthClass = computed(() => {
  const strength = passwordStrength.value
  if (strength < 30) return 'weak'
  if (strength < 60) return 'fair'
  if (strength < 80) return 'good'
  return 'strong'
})

const passwordStrengthText = computed(() => {
  const strength = passwordStrength.value
  if (strength < 30) return '弱'
  if (strength < 60) return '一般'
  if (strength < 80) return '良好'
  return '强'
})

// 表单验证规则
const validateField = (field) => {
  fieldErrors.value[field] = ''

  switch (field) {
    case 'username':
      if (!username.value.trim()) {
        fieldErrors.value.username = '请输入用户名'
      } else if (username.value.trim().length < 2) {
        fieldErrors.value.username = '用户名至少需要2个字符'
      } else if (username.value.trim().length > 20) {
        fieldErrors.value.username = '用户名不能超过20个字符'
      } else if (!/^[a-zA-Z0-9_]+$/.test(username.value.trim())) {
        fieldErrors.value.username = '用户名只能包含字母、数字和下划线'
      }
      break

    case 'email':
      if (email.value && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value)) {
        fieldErrors.value.email = '请输入有效的邮箱地址'
      }
      break

    case 'password':
      if (!password.value) {
        fieldErrors.value.password = '请输入密码'
      } else if (password.value.length < 6) {
        fieldErrors.value.password = '密码至少需要6个字符'
      } else if (passwordStrength.value < 40) {
        fieldErrors.value.password = '密码强度太弱，请使用更复杂的密码'
      }
      break

    case 'confirm':
      if (!confirm.value) {
        fieldErrors.value.confirm = '请确认密码'
      } else if (confirm.value !== password.value) {
        fieldErrors.value.confirm = '两次输入的密码不一致'
      }
      break

    case 'terms':
      if (!agreeToTerms.value) {
        fieldErrors.value.terms = '请同意服务条款和隐私政策'
      }
      break
  }
}

const clearFieldError = (field) => {
  if (fieldErrors.value[field]) {
    fieldErrors.value[field] = ''
  }
}

const handlePasswordInput = () => {
  clearFieldError('password')
  // 实时验证确认密码
  if (confirm.value && confirm.value !== password.value) {
    validateField('confirm')
  } else {
    clearFieldError('confirm')
  }
}

// 表单有效性检查
const isFormValid = computed(() => {
  return username.value.trim() &&
         password.value &&
         confirm.value &&
         agreeToTerms.value &&
         username.value.trim().length >= 2 &&
         username.value.trim().length <= 20 &&
         password.value.length >= 6 &&
         passwordStrength.value >= 40 &&
         password.value === confirm.value &&
         /^[a-zA-Z0-9_]+$/.test(username.value.trim()) &&
         (!email.value || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value)) &&
         !Object.values(fieldErrors.value).some(error => error)
})

async function submit() {
  if (loading.value || !isFormValid.value) return

  // 验证所有字段
  validateField('username')
  validateField('email')
  validateField('password')
  validateField('confirm')
  validateField('terms')

  if (!isFormValid.value) return

  loading.value = true
  err.value = ''

  // 添加重试机制
  let retryCount = 0
  const maxRetries = 2

  while (retryCount <= maxRetries) {
    try {
      const registerData = {
        username: username.value.trim(),
        password: password.value
      }

      if (email.value) {
        registerData.email = email.value
      }

      const r = await api.register(registerData)
      const token = r?.token
      const user = r?.user

      if (!token) throw new Error('注册失败，请重试')

      setAuth(token, user?.username || username.value.trim())
      await nextTick()
      router.replace('/dashboard')
      return // 成功则退出

    } catch (e) {
      const error = e
      const isNetworkError = !navigator.onLine || error.message?.includes('fetch')
      const isServerError = error.status >= 500
      const isRateLimited = error.status === 429

      // 如果是网络错误且还有重试次数，自动重试
      if (isNetworkError && retryCount < maxRetries) {
        retryCount++
        err.value = `网络连接失败，正在重试 (${retryCount}/${maxRetries})...`
        await new Promise(resolve => setTimeout(resolve, 1000 * retryCount)) // 递增延迟
        continue
      }

      // 根据错误类型提供不同的提示信息
      if (isRateLimited) {
        err.value = '请求过于频繁，请稍后再试'
      } else if (isServerError) {
        err.value = '服务器暂时不可用，请稍后重试'
      } else if (isNetworkError) {
        err.value = '网络连接失败，请检查网络后重试'
      } else if (error.status === 409) {
        err.value = '用户名已存在，请选择其他用户名'
      } else if (error.status === 422) {
        err.value = '输入信息格式不正确，请检查后重试'
      } else {
        err.value = error.message || '注册失败，请稍后重试'
      }
      break
    }
  }

  loading.value = false
}

function showTerms() {
  alert('服务条款：\n\n1. 本平台提供加密资产分析服务\n2. 用户需遵守相关法律法规\n3. 平台保留服务解释权\n\n详细条款请访问官方网站。')
}

function showPrivacy() {
  alert('隐私政策：\n\n1. 我们重视您的隐私保护\n2. 用户数据仅用于提供服务\n3. 我们不会泄露用户个人信息\n\n详细政策请访问官方网站。')
}
</script>

<style scoped>
/* ===== 认证页面主容器 ===== */
.auth-container {
  min-height: 100vh;
  display: flex;
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.auth-bg {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  overflow: hidden;
}

.bg-shape {
  position: absolute;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  animation: float 6s ease-in-out infinite;
}

.shape-1 {
  width: 300px;
  height: 300px;
  top: -150px;
  right: -150px;
  animation-delay: 0s;
}

.shape-2 {
  width: 200px;
  height: 200px;
  top: 50%;
  left: -100px;
  animation-delay: 2s;
}

.shape-3 {
  width: 150px;
  height: 150px;
  bottom: -75px;
  right: 20%;
  animation-delay: 4s;
}

@keyframes float {
  0%, 100% { transform: translateY(0px) rotate(0deg); }
  50% { transform: translateY(-20px) rotate(180deg); }
}

.auth-content {
  display: flex;
  width: 100%;
  min-height: 100vh;
  position: relative;
  z-index: 1;
}

/* ===== 左侧介绍区域 ===== */
.auth-intro {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-8);
  color: white;
}

.intro-content {
  max-width: 500px;
  text-align: center;
}

.brand-title {
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: var(--space-4);
  background: linear-gradient(135deg, #ffffff 0%, #e0e7ff 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.brand-subtitle {
  font-size: 1.125rem;
  opacity: 0.9;
  margin-bottom: var(--space-8);
  line-height: 1.6;
}

.benefits-list {
  display: grid;
  gap: var(--space-4);
  text-align: left;
}

.benefit-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  background: rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-lg);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.benefit-icon {
  font-size: 1.5rem;
  flex-shrink: 0;
}

/* ===== 右侧表单区域 ===== */
.auth-form-container {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-8);
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
}

.auth-form-card {
  width: 100%;
  max-width: 400px;
  background: white;
  border-radius: var(--radius-2xl);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  padding: var(--space-8);
  border: 1px solid rgba(255, 255, 255, 0.8);
}

/* ===== 表单头部 ===== */
.form-header {
  text-align: center;
  margin-bottom: var(--space-8);
}

.form-title {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: var(--space-2);
}

.form-subtitle {
  color: var(--text-muted);
  font-size: var(--text-base);
}

/* ===== 表单样式 ===== */
.auth-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.form-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--text-primary);
}

.input-icon {
  width: 1rem;
  height: 1rem;
  color: var(--primary-500);
  flex-shrink: 0;
}

.optional {
  color: var(--text-muted);
  font-weight: 400;
}

.form-input {
  width: 100%;
  padding: var(--space-3) var(--space-4);
  border: 2px solid var(--border-light);
  border-radius: var(--radius-lg);
  font-size: var(--text-base);
  background: white;
  color: var(--text-primary);
  transition: all var(--transition-fast);
  outline: none;
}

.form-input:focus {
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px var(--primary-100);
}

.form-input.input-error {
  border-color: var(--error-500);
}

.form-input.input-error:focus {
  box-shadow: 0 0 0 3px var(--error-100);
}

.field-error {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--text-sm);
  color: var(--error-600);
  margin-top: var(--space-1);
  animation: slideIn 0.3s ease-out;
}

/* ===== 密码输入框 ===== */
.password-input-container {
  position: relative;
}

.password-input {
  padding-right: 3rem;
}

.password-toggle {
  position: absolute;
  right: var(--space-3);
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: var(--space-1);
  border-radius: var(--radius-md);
  transition: color var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
}

.password-toggle:hover {
  color: var(--primary-600);
}

.password-toggle svg {
  width: 1.25rem;
  height: 1.25rem;
}

/* ===== 密码强度指示器 ===== */
.password-strength {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-top: var(--space-2);
}

.strength-meter {
  flex: 1;
  height: 4px;
  background: var(--border-light);
  border-radius: 2px;
  overflow: hidden;
}

.strength-bar {
  height: 100%;
  border-radius: 2px;
  transition: all var(--transition-normal);
}

.strength-bar.weak {
  background: var(--error-500);
}

.strength-bar.fair {
  background: var(--warning-500);
}

.strength-bar.good {
  background: var(--primary-500);
}

.strength-bar.strong {
  background: var(--success-500);
}

.strength-text {
  font-size: var(--text-xs);
  font-weight: 500;
  min-width: 2rem;
  text-align: right;
}

.strength-text {
  color: var(--text-muted);
}

/* ===== 服务条款复选框 ===== */
.terms-group {
  margin-top: var(--space-2);
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  font-size: var(--text-sm);
  color: var(--text-secondary);
  user-select: none;
  line-height: 1.5;
}

.checkbox-input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
  margin: 0;
}

.checkbox-mark {
  width: 1.125rem;
  height: 1.125rem;
  border: 2px solid var(--border-medium);
  background: white;
  position: relative;
  transition: all var(--transition-fast);
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}

.checkbox-input:checked + .checkbox-mark {
  background: var(--primary-600);
  border-color: var(--primary-600);
}

.checkbox-input:checked + .checkbox-mark::after {
  content: '';
  position: absolute;
  left: 4px;
  width: 6px;
  height: 10px;
  border: solid white;
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}

.checkbox-text {
  font-weight: 500;
  flex: 1;
}

.link-button {
  background: none;
  border: none;
  color: var(--primary-600);
  cursor: pointer;
  text-decoration: underline;
  font-size: inherit;
  padding: 0;
  font-family: inherit;
}

.link-button:hover {
  color: var(--primary-700);
}

/* ===== 全局错误提示 ===== */
.global-error {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3);
  background: var(--error-50);
  border: 1px solid var(--error-200);
  border-radius: var(--radius-lg);
  color: var(--error-700);
  font-size: var(--text-sm);
  animation: slideIn 0.3s ease-out;
}

.error-icon {
  width: 1.25rem;
  height: 1.25rem;
  color: var(--error-500);
  flex-shrink: 0;
}

/* ===== 按钮样式 ===== */
.auth-button {
  width: 100%;
  padding: var(--space-4);
  background: linear-gradient(135deg, var(--primary-600) 0%, var(--primary-700) 100%);
  color: white;
  border: none;
  border-radius: var(--radius-lg);
  font-size: var(--text-base);
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
  position: relative;
  overflow: hidden;
  box-shadow: 0 4px 14px 0 rgba(99, 102, 241, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
}

.auth-button:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px 0 rgba(99, 102, 241, 0.4);
}

.auth-button:active:not(:disabled) {
  transform: translateY(0);
}

.auth-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
  box-shadow: 0 2px 8px 0 rgba(99, 102, 241, 0.2);
}

.auth-button.loading {
  pointer-events: none;
}

.loading-text {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
}

.loading-spinner {
  width: 1.25rem;
  height: 1.25rem;
  animation: spin 1s linear infinite;
}

/* ===== 分割线 ===== */
.divider {
  position: relative;
  text-align: center;
  margin: var(--space-6) 0;
}

.divider::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 0;
  right: 0;
  height: 1px;
  background: var(--border-light);
}

.divider-text {
  background: white;
  padding: 0 var(--space-4);
  color: var(--text-muted);
  font-size: var(--text-sm);
  position: relative;
  z-index: 1;
}

/* ===== 登录链接 ===== */
.login-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  width: 100%;
  padding: var(--space-3) var(--space-4);
  background: white;
  color: var(--primary-600);
  border: 2px solid var(--primary-200);
  border-radius: var(--radius-lg);
  font-size: var(--text-base);
  font-weight: 600;
  text-decoration: none;
  transition: all var(--transition-fast);
  cursor: pointer;
}

.login-link:hover {
  background: var(--primary-50);
  border-color: var(--primary-300);
  color: var(--primary-700);
  transform: translateY(-1px);
}

.login-link svg {
  width: 1rem;
  height: 1rem;
  transition: transform var(--transition-fast);
}

.login-link:hover svg {
  transform: translateX(-2px);
}

/* ===== 动画 ===== */
@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* ===== 响应式设计 ===== */
@media (max-width: 1024px) {
  .auth-content {
    flex-direction: column;
  }

  .auth-intro {
    flex: none;
    padding: var(--space-6) var(--space-4);
    min-height: 300px;
  }

  .brand-title {
    font-size: 2rem;
  }

  .benefits-list {
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: var(--space-3);
  }
}

@media (max-width: 640px) {
  .auth-container {
    padding: 0;
  }

  .auth-intro {
    padding: var(--space-4);
    min-height: 250px;
  }

  .brand-title {
    font-size: 1.75rem;
  }

  .auth-form-card {
    margin: var(--space-2);
    padding: var(--space-6);
    border-radius: var(--radius-xl);
  }

  .form-title {
    font-size: 1.75rem;
  }

  .benefits-list {
    grid-template-columns: 1fr;
  }

  .benefit-item {
    padding: var(--space-2);
  }
}
</style>
