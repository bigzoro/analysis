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
          <h1 class="brand-title">加密资产分析平台</h1>
          <p class="brand-subtitle">智能投资决策，数据驱动未来</p>
          <div class="features-list">
            <div class="feature-item">
              <div class="feature-icon">📊</div>
              <span>实时市场数据分析</span>
            </div>
<!--            <div class="feature-item">-->
<!--              <div class="feature-icon">🤖</div>-->
<!--              <span>AI智能推荐系统</span>-->
<!--            </div>-->
<!--            <div class="feature-item">-->
<!--              <div class="feature-icon">📈</div>-->
<!--              <span>专业回测引擎</span>-->
<!--            </div>-->
<!--            <div class="feature-item">-->
<!--              <div class="feature-icon">🛡️</div>-->
<!--              <span>风险管理工具</span>-->
<!--            </div>-->
          </div>
        </div>
      </div>

      <!-- 右侧登录表单 -->
      <div class="auth-form-container">
        <div class="auth-form-card">
          <!-- 表单头部 -->
          <div class="form-header">
            <h2 class="form-title">欢迎回来</h2>
            <p class="form-subtitle">请登录您的账户</p>
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
                placeholder="请输入用户名"
                autocomplete="username"
                @blur="validateField('username')"
                @input="clearFieldError('username')"
              />
              <div v-if="fieldErrors.username" class="field-error">
                {{ fieldErrors.username }}
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
                  placeholder="请输入密码"
                  autocomplete="current-password"
                  @blur="validateField('password')"
                  @input="clearFieldError('password')"
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
              <div v-if="fieldErrors.password" class="field-error">
                {{ fieldErrors.password }}
              </div>
            </div>

            <!-- 记住我 -->
            <div class="form-group remember-group">
              <label class="checkbox-label">
                <input
                  v-model="rememberMe"
                  type="checkbox"
                  class="checkbox-input"
                />
                <span class="checkbox-mark"></span>
                <span class="checkbox-text">记住我</span>
              </label>
            </div>

            <!-- 全局错误提示 -->
            <div v-if="err" class="global-error">
              <svg class="error-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
              {{ err }}
            </div>

            <!-- 登录按钮 -->
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
                登录中...
              </span>
              <span v-else>登录</span>
            </button>

            <!-- 忘记密码 -->
            <div class="auth-links">
              <button type="button" class="link-button" @click="forgotPassword">
                忘记密码？
              </button>
            </div>

            <!-- 分割线 -->
            <div class="divider">
              <span class="divider-text">还没有账号？</span>
            </div>

            <!-- 注册链接 -->
<!--            <RouterLink to="/register" class="register-link">-->
<!--              <span>立即注册</span>-->
<!--              <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">-->
<!--                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>-->
<!--              </svg>-->
<!--            </RouterLink>-->
          </form>
        </div>
      </div>
    </div>

    <!-- 忘记密码对话框 -->
    <div v-if="showForgotPassword" class="modal-overlay" @click="showForgotPassword = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3 class="modal-title">忘记密码</h3>
          <button class="modal-close" @click="showForgotPassword = false">
            <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>

        <div class="modal-body">
          <p class="modal-description">
            请输入您注册时使用的邮箱地址，我们将发送密码重置链接到您的邮箱。
          </p>

          <form class="reset-form" @submit.prevent="submitPasswordReset">
            <div class="form-group">
              <label class="form-label">
                <svg class="input-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path>
                </svg>
                邮箱地址
              </label>
              <input
                v-model="resetEmail"
                type="email"
                class="form-input"
                placeholder="请输入邮箱地址"
                autocomplete="email"
                required
              />
            </div>

            <div v-if="resetMessage" class="reset-message" :class="{ 'success': resetMessage.includes('发送'), 'error': !resetMessage.includes('发送') }">
              {{ resetMessage }}
            </div>

            <div class="modal-actions">
              <button
                type="button"
                class="btn btn-secondary"
                @click="showForgotPassword = false"
                :disabled="resetLoading"
              >
                取消
              </button>
              <button
                type="submit"
                class="btn btn-primary"
                :disabled="resetLoading"
                :class="{ 'loading': resetLoading }"
              >
                <span v-if="resetLoading">发送中...</span>
                <span v-else>发送重置链接</span>
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { api } from '../api/api.js'
import { useAuth } from '../stores/auth.js'

const router = useRouter()
const route = useRoute()
const { setAuth } = useAuth()

// 表单数据
const username = ref('')
const password = ref('')
const showPassword = ref(false)
const rememberMe = ref(false)
const loading = ref(false)
const err = ref('')

// 字段错误
const fieldErrors = ref({
  username: '',
  password: ''
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
      }
      break

    case 'password':
      if (!password.value) {
        fieldErrors.value.password = '请输入密码'
      } else if (password.value.length < 6) {
        fieldErrors.value.password = '密码至少需要6个字符'
      }
      break
  }
}

const clearFieldError = (field) => {
  if (fieldErrors.value[field]) {
    fieldErrors.value[field] = ''
  }
}

// 表单有效性检查
const isFormValid = computed(() => {
  return username.value.trim() &&
         password.value &&
         username.value.trim().length >= 2 &&
         password.value.length >= 6 &&
         !fieldErrors.value.username &&
         !fieldErrors.value.password
})

async function submit() {
  if (loading.value || !isFormValid.value) return

  // 验证所有字段
  validateField('username')
  validateField('password')

  if (!isFormValid.value) return

  loading.value = true
  err.value = ''

  // 添加重试机制
  let retryCount = 0
  const maxRetries = 2

  while (retryCount <= maxRetries) {
    try {
      const r = await api.login({
        username: username.value.trim(),
        password: password.value
      })

      const token = r?.token
      const user = r?.user

      if (!token) throw new Error('登录失败，请重试')

      // 保存记住我状态
      if (rememberMe.value) {
        localStorage.setItem('remember_login', 'true')
      }

      // 更新认证状态
      setAuth(token, user?.username || username.value.trim())

      // 等待组件更新
      await nextTick()

      const redirect = route.query.redirect ? String(route.query.redirect) : '/dashboard'
      router.replace(redirect)
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
      } else if (error.status === 401) {
        err.value = '用户名或密码错误'
      } else if (error.status === 403) {
        err.value = '账户已被禁用，请联系管理员'
      } else {
        err.value = error.message || '登录失败，请检查用户名和密码'
      }
      break
    }
  }

  loading.value = false
}

// 忘记密码功能
const showForgotPassword = ref(false)
const resetEmail = ref('')
const resetLoading = ref(false)
const resetMessage = ref('')

function forgotPassword() {
  showForgotPassword.value = true
  resetEmail.value = ''
  resetMessage.value = ''
}

async function submitPasswordReset() {
  if (!resetEmail.value.trim()) {
    resetMessage.value = '请输入邮箱地址'
    return
  }

  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(resetEmail.value)) {
    resetMessage.value = '请输入有效的邮箱地址'
    return
  }

  resetLoading.value = true
  resetMessage.value = ''

  try {
    // 这里可以调用重置密码API
    // await api.forgotPassword({ email: resetEmail.value.trim() })

    // 暂时显示成功消息
    resetMessage.value = '密码重置链接已发送到您的邮箱，请查收'
    setTimeout(() => {
      showForgotPassword.value = false
    }, 3000)
  } catch (e) {
    resetMessage.value = e?.message || '发送失败，请稍后重试'
  } finally {
    resetLoading.value = false
  }
}

// 初始化时检查记住我状态
const initRememberMe = () => {
  const remembered = localStorage.getItem('remember_login') === 'true'
  rememberMe.value = remembered
}

// 组件挂载时初始化
initRememberMe()
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

.features-list {
  display: grid;
  gap: var(--space-4);
  text-align: left;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  background: rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-lg);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.feature-icon {
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

/* ===== 记住我复选框 ===== */
.remember-group {
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
}

.checkbox-input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}

.checkbox-mark {
  width: 1.125rem;
  height: 1.125rem;
  border: 2px solid var(--border-medium);
  border-radius: var(--radius-sm);
  background: white;
  position: relative;
  transition: all var(--transition-fast);
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

/* ===== 链接样式 ===== */
.auth-links {
  text-align: center;
}

.link-button {
  background: none;
  border: none;
  color: var(--primary-600);
  font-size: var(--text-sm);
  cursor: pointer;
  text-decoration: underline;
  transition: color var(--transition-fast);
}

.link-button:hover {
  color: var(--primary-700);
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

/* ===== 注册链接 ===== */
.register-link {
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

.register-link:hover {
  background: var(--primary-50);
  border-color: var(--primary-300);
  color: var(--primary-700);
  transform: translateY(-1px);
}

.register-link svg {
  width: 1rem;
  height: 1rem;
  transition: transform var(--transition-fast);
}

.register-link:hover svg {
  transform: translateX(2px);
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

/* ===== 模态框样式 ===== */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.3s ease-out;
}

.modal-content {
  background: white;
  border-radius: var(--radius-2xl);
  box-shadow: var(--shadow-xl);
  width: 100%;
  max-width: 400px;
  margin: var(--space-4);
  animation: slideIn 0.3s ease-out;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-6);
  border-bottom: 1px solid var(--border-light);
}

.modal-title {
  font-size: var(--text-xl);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin: 0;
}

.modal-close {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: var(--space-1);
  border-radius: var(--radius-md);
  transition: color var(--transition-fast);
}

.modal-close:hover {
  color: var(--text-primary);
  background: var(--bg-secondary);
}

.modal-close svg {
  width: 1.25rem;
  height: 1.25rem;
}

.modal-body {
  padding: var(--space-6);
}

.modal-description {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  line-height: 1.6;
  margin-bottom: var(--space-4);
}

.reset-form .form-group {
  margin-bottom: var(--space-4);
}

.reset-form .form-input {
  width: 100%;
}

.reset-message {
  padding: var(--space-3);
  border-radius: var(--radius-lg);
  font-size: var(--text-sm);
  margin-bottom: var(--space-4);
  animation: slideIn 0.3s ease-out;
}

.reset-message.success {
  background: var(--success-50);
  color: var(--success-700);
  border: 1px solid var(--success-200);
}

.reset-message.error {
  background: var(--error-50);
  color: var(--error-700);
  border: 1px solid var(--error-200);
}

.modal-actions {
  display: flex;
  gap: var(--space-3);
  justify-content: flex-end;
  margin-top: var(--space-6);
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-20px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
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

  .features-list {
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

  .features-list {
    grid-template-columns: 1fr;
  }

  .feature-item {
    padding: var(--space-2);
  }
}
</style>
