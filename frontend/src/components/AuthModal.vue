<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'

const { t } = useI18n()
const emit = defineEmits(['login-success', 'close'])

// 状态
const mode = ref('login') // 'login', 'register', 'redeem', 'reset'
const step = ref(1) // 1: 邮箱输入, 2: 验证码/密码 (仅验证码登录使用)
const loading = ref(false)
const sendingCode = ref(false) // 发送验证码加载状态
const error = ref('')
const successMessage = ref('')
const countdown = ref(0)
const devCode = ref('') // 开发模式下显示验证码

// 表单数据
const email = ref('')
const code = ref('')
const password = ref('')
const confirmPassword = ref('')
const nickname = ref('')
const redeemKey = ref('')
const showPassword = ref(false)
const loginMethod = ref('password') // 'password' or 'code'
const inviteCode = ref(localStorage.getItem('pendingInviteCode') || 'VIP666DB')
const acceptedPolicies = ref(false)
const termsPath = '/terms-of-service'
const privacyPath = '/privacy-policy'

// 计时器
let countdownTimer = null

// 检查是否有待使用的邀请码
onMounted(() => {
  if (localStorage.getItem('pendingInviteCode')) {
    mode.value = 'register'
  }
})

// 计算属性
const canSendCode = computed(() => {
  return countdown.value === 0 && isValidEmail(email.value) && !sendingCode.value
})

const submitButtonText = computed(() => {
  if (loading.value) return t('auth.pleaseWait')
  if (mode.value === 'redeem') return t('auth.redeem')
  if (mode.value === 'register') return t('auth.register')
  if (mode.value === 'reset') return t('auth.resetPassword')
  return t('auth.login')
})

// 方法
const isValidEmail = (email) => {
  const pattern = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
  return pattern.test(email)
}

const startCountdown = () => {
  countdown.value = 60
  countdownTimer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      clearInterval(countdownTimer)
    }
  }, 1000)
}

const sendCode = async () => {
  if (!isValidEmail(email.value)) {
    error.value = t('auth.invalidEmail')
    return
  }

  error.value = ''
  sendingCode.value = true

  try {
    const codeType = mode.value === 'register' ? 'register' : (mode.value === 'reset' ? 'reset' : 'login')
    const response = await axios.post('/api/auth/send-code', {
      email: email.value,
      type: codeType
    })

    successMessage.value = response.data.message
    startCountdown()

    // 开发模式下显示验证码
    if (response.data.dev_code) {
      devCode.value = response.data.dev_code
    }
  } catch (e) {
    error.value = e.response?.data?.error || t('auth.sendCodeFailed')
  } finally {
    sendingCode.value = false
  }
}

const handleSubmit = async () => {
  error.value = ''
  successMessage.value = ''

  // 兑换模式
  if (mode.value === 'redeem') {
    await handleRedeem()
    return
  }

  if (mode.value === 'login' && !acceptedPolicies.value) {
    error.value = t('auth.agreementRequired')
    return
  }

  // 密码登录
  if (mode.value === 'login' && loginMethod.value === 'password') {
    await handlePasswordLogin()
    return
  }

  // 验证码登录
  if (mode.value === 'login' && loginMethod.value === 'code') {
    await handleCodeLogin()
    return
  }

  // 注册 - 直接提交
  if (mode.value === 'register') {
    await handleRegister()
    return
  }

  // 重置密码
  if (mode.value === 'reset') {
    await handleResetPassword()
    return
  }
}

const handlePasswordLogin = async () => {
  if (!isValidEmail(email.value)) {
    error.value = t('auth.invalidEmail')
    return
  }
  if (!password.value) {
    error.value = t('auth.enterPassword')
    return
  }

  loading.value = true
  try {
    const response = await axios.post('/api/auth/login', {
      email: email.value,
      password: password.value
    })

    emit('login-success', response.data.user, response.data.token)
  } catch (e) {
    error.value = e.response?.data?.error || t('auth.loginFailed')
  } finally {
    loading.value = false
  }
}

const handleCodeLogin = async () => {
  if (!code.value || code.value.length !== 6) {
    error.value = t('auth.enterCode')
    return
  }

  loading.value = true
  try {
    const response = await axios.post('/api/auth/login', {
      email: email.value,
      code: code.value
    })

    emit('login-success', response.data.user, response.data.token)
  } catch (e) {
    error.value = e.response?.data?.error || t('auth.loginFailed')
  } finally {
    loading.value = false
  }
}

const handleRegister = async () => {
  if (!code.value || code.value.length !== 6) {
    error.value = t('auth.enterCode')
    return
  }
  if (password.value.length < 6) {
    error.value = t('auth.passwordMinLength')
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = t('auth.passwordMismatch')
    return
  }

  loading.value = true
  try {
    const response = await axios.post('/api/auth/register', {
      email: email.value,
      code: code.value,
      password: password.value,
      nickname: nickname.value || undefined,
      invite_code: inviteCode.value.trim() || undefined
    })

    // 清除待使用的邀请码
    localStorage.removeItem('pendingInviteCode')
    emit('login-success', response.data.user, response.data.token)
  } catch (e) {
    error.value = e.response?.data?.error || t('auth.registerFailed')
  } finally {
    loading.value = false
  }
}

const handleResetPassword = async () => {
  if (!isValidEmail(email.value)) {
    error.value = t('auth.invalidEmail')
    return
  }
  if (!code.value || code.value.length !== 6) {
    error.value = t('auth.enterCode')
    return
  }
  if (password.value.length < 6) {
    error.value = t('auth.passwordMinLength')
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = t('auth.passwordMismatch')
    return
  }

  loading.value = true
  try {
    await axios.post('/api/auth/reset-password', {
      email: email.value,
      code: code.value,
      password: password.value
    })

    successMessage.value = t('auth.resetSuccess')
    // 2秒后切换到登录模式
    setTimeout(() => {
      switchMode('login')
    }, 2000)
  } catch (e) {
    error.value = e.response?.data?.error || t('auth.resetFailed')
  } finally {
    loading.value = false
  }
}

const handleRedeem = async () => {
  if (!redeemKey.value.trim()) {
    error.value = t('auth.enterKey')
    return
  }

  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const response = await axios.post('/api/user/redeem', {
      key: redeemKey.value
    }, {
      headers: { Authorization: `Bearer ${token}` }
    })

    successMessage.value = t('auth.redeemSuccess', { credits: response.data.credits_added, total: response.data.current_credits })
    redeemKey.value = ''

    // 更新本地用户信息
    const user = JSON.parse(localStorage.getItem('user') || '{}')
    user.credits = response.data.current_credits
    localStorage.setItem('user', JSON.stringify(user))

    emit('login-success', user)
  } catch (e) {
    error.value = e.response?.data?.error || t('auth.redeemFailed')
  } finally {
    loading.value = false
  }
}

const switchMode = (newMode) => {
  mode.value = newMode
  step.value = 1
  error.value = ''
  successMessage.value = ''
  code.value = ''
  password.value = ''
  confirmPassword.value = ''
  devCode.value = ''
}

const goBack = () => {
  step.value = 1
  code.value = ''
  error.value = ''
  devCode.value = ''
}

const handleKeyDown = (e) => {
  if (e.key === 'Enter' && !loading.value) {
    handleSubmit()
  }
}

// OAuth login
const oauthLoading = ref(false)

const handleLinuxDoLogin = async () => {
  oauthLoading.value = true
  error.value = ''

  try {
    const response = await axios.get('/api/auth/oauth/linuxdo')
    const authURL = response.data.url

    // Open popup window (centered, 600x700)
    const width = 600
    const height = 700
    const left = window.screenX + (window.outerWidth - width) / 2
    const top = window.screenY + (window.outerHeight - height) / 2
    const popup = window.open(
      authURL,
      'linuxdo-oauth',
      `width=${width},height=${height},left=${left},top=${top},toolbar=no,menubar=no,scrollbars=yes`
    )

    // Popup blocked by browser
    if (!popup || popup.closed) {
      oauthLoading.value = false
      error.value = t('auth.oauthFailed')
      return
    }

    // Listen for postMessage from popup
    let timeoutId = null
    const messageHandler = (event) => {
      if (event.origin !== window.location.origin) return
      if (event.data?.type === 'oauth-login-success') {
        window.removeEventListener('message', messageHandler)
        if (timeoutId) clearTimeout(timeoutId)
        oauthLoading.value = false
        emit('login-success', event.data.user, event.data.token)
      }
    }
    window.addEventListener('message', messageHandler)

    // Timeout: stop loading after 5 minutes if no response
    timeoutId = setTimeout(() => {
      window.removeEventListener('message', messageHandler)
      oauthLoading.value = false
    }, 5 * 60 * 1000)
  } catch (e) {
    error.value = e.response?.data?.error || t('auth.oauthFailed')
    oauthLoading.value = false
  }
}
</script>

<template>
  <div class="auth-modal-overlay">
    <div class="auth-modal">
      <!-- 关闭按钮 -->
      <button class="close-btn" @click="$emit('close')">✕</button>

      <!-- Header -->
      <div class="auth-header">
        <div class="auth-brand">
          <img src="/images/icon.png" alt="小野 AI" class="logo-icon" />
          <h1>小野 AI</h1>
        </div>
        <p class="subtitle">
          {{ mode === 'redeem' ? $t('auth.redeemDiamonds') : mode === 'register' ? $t('auth.createAccount') : mode === 'reset' ? $t('auth.resetPassword') : $t('auth.welcomeBack') }}
        </p>
      </div>

      <!-- Tab 切换 -->
      <div class="auth-tabs" v-if="mode !== 'redeem' && mode !== 'reset'">
        <button
          @click="switchMode('login')"
          :class="{ active: mode === 'login' }"
          class="tab-btn"
        >
          {{ $t('auth.login') }}
        </button>
        <button
          @click="switchMode('register')"
          :class="{ active: mode === 'register' }"
          class="tab-btn"
        >
          {{ $t('auth.register') }}
        </button>
      </div>

      <!-- 表单 -->
      <form @submit.prevent="handleSubmit" class="auth-form">
        <!-- 兑换模式 -->
        <template v-if="mode === 'redeem'">
          <div class="form-group">
            <label class="form-label">{{ $t('auth.key') }}</label>
            <textarea
              v-model="redeemKey"
              :placeholder="$t('auth.keyPlaceholder')"
              class="form-textarea"
              rows="8"
              :disabled="loading"
            ></textarea>
          </div>
          <p class="form-hint">{{ $t('auth.keyHint') }}</p>
        </template>

        <!-- 登录/注册模式 -->
        <template v-else>
          <!-- ===== 登录模式 ===== -->
          <template v-if="mode === 'login'">
            <!-- 邮箱输入 -->
            <div class="form-group">
              <label class="form-label">{{ $t('auth.email') }}</label>
              <input
                v-model="email"
                type="email"
                :placeholder="$t('auth.emailPlaceholder')"
                class="form-input"
                @keydown="handleKeyDown"
                :disabled="loading"
              />
            </div>

            <!-- 登录方式切换 -->
            <div class="login-method">
              <button
                type="button"
                @click="loginMethod = 'password'"
                :class="{ active: loginMethod === 'password' }"
                class="method-btn"
              >
                {{ $t('auth.passwordLogin') }}
              </button>
              <button
                type="button"
                @click="loginMethod = 'code'"
                :class="{ active: loginMethod === 'code' }"
                class="method-btn"
              >
                {{ $t('auth.codeLogin') }}
              </button>
            </div>

            <!-- 密码输入 (密码登录) -->
            <template v-if="loginMethod === 'password'">
              <div class="form-group">
                <label class="form-label">{{ $t('auth.password') }}</label>
                <div class="input-wrapper">
                  <input
                    v-model="password"
                    :type="showPassword ? 'text' : 'password'"
                    :placeholder="$t('auth.passwordPlaceholder')"
                    class="form-input"
                    @keydown="handleKeyDown"
                    :disabled="loading"
                  />
                  <button
                    type="button"
                    @click="showPassword = !showPassword"
                    class="toggle-password-btn"
                  >
                    {{ showPassword ? '👁️' : '👁️‍🗨️' }}
                  </button>
                </div>
                <div class="forgot-password">
                  <button type="button" @click="switchMode('reset')" class="link-btn">
                    {{ $t('auth.forgotPassword') }}
                  </button>
                </div>
              </div>
            </template>

            <!-- 验证码输入 (验证码登录) -->
            <template v-if="loginMethod === 'code'">
              <div class="form-group">
                <label class="form-label">{{ $t('auth.verificationCode') }}</label>
                <div class="code-input-wrapper">
                  <input
                    v-model="code"
                    type="text"
                    maxlength="6"
                    :placeholder="$t('auth.codePlaceholder')"
                    class="form-input"
                    @keydown="handleKeyDown"
                    :disabled="loading"
                  />
                  <button
                    type="button"
                    @click="sendCode"
                    :disabled="!canSendCode"
                    class="resend-btn"
                  >
                    <span v-if="sendingCode" class="spinner-small"></span>
                    {{ sendingCode ? '' : (countdown > 0 ? `${countdown}s` : $t('auth.getCode')) }}
                  </button>
                </div>
              </div>

              <!-- 开发模式验证码提示 -->
              <div v-if="devCode" class="dev-code-hint">
                {{ $t('auth.devCode') }} <strong>{{ devCode }}</strong>
              </div>
            </template>

            <div class="agreement-check">
              <div class="agreement-label">
                <input
                  v-model="acceptedPolicies"
                  type="checkbox"
                  class="agreement-checkbox"
                  :disabled="loading"
                />
                <span class="agreement-text">
                  {{ $t('auth.agreePrefix') }}
                  <a
                    :href="termsPath"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="agreement-link"
                    @click.stop
                  >
                    {{ $t('auth.userAgreement') }}
                  </a>
                  {{ $t('auth.and') }}
                  <a
                    :href="privacyPath"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="agreement-link"
                    @click.stop
                  >
                    {{ $t('auth.privacyPolicy') }}
                  </a>
                </span>
              </div>
            </div>
          </template>

          <!-- ===== 注册模式 ===== -->
          <template v-if="mode === 'register'">
            <!-- 邮箱 -->
            <div class="form-group">
              <label class="form-label">{{ $t('auth.email') }}</label>
              <input
                v-model="email"
                type="email"
                :placeholder="$t('auth.emailPlaceholder')"
                class="form-input"
                @keydown="handleKeyDown"
                :disabled="loading"
              />
            </div>

            <!-- 验证码 -->
            <div class="form-group">
              <label class="form-label">{{ $t('auth.verificationCode') }}</label>
              <div class="code-input-wrapper">
                <input
                  v-model="code"
                  type="text"
                  maxlength="6"
                  :placeholder="$t('auth.codePlaceholder')"
                  class="form-input"
                  @keydown="handleKeyDown"
                  :disabled="loading"
                />
                <button
                  type="button"
                  @click="sendCode"
                  :disabled="!canSendCode"
                  class="resend-btn"
                >
                  <span v-if="sendingCode" class="spinner-small"></span>
                  {{ sendingCode ? '' : (countdown > 0 ? `${countdown}s` : $t('auth.getCode')) }}
                </button>
              </div>
            </div>

            <!-- 开发模式验证码提示 -->
            <div v-if="devCode" class="dev-code-hint">
              {{ $t('auth.devCode') }} <strong>{{ devCode }}</strong>
            </div>

            <!-- 昵称 -->
            <div class="form-group">
              <label class="form-label">{{ $t('auth.nickname') }} <span class="optional">{{ $t('auth.nicknameOptional') }}</span></label>
              <input
                v-model="nickname"
                type="text"
                :placeholder="$t('auth.nicknamePlaceholder')"
                class="form-input"
                :disabled="loading"
              />
            </div>

            <!-- 密码 -->
            <div class="form-group">
              <label class="form-label">{{ $t('auth.password') }}</label>
              <div class="input-wrapper">
                <input
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  :placeholder="$t('auth.passwordHint')"
                  class="form-input"
                  @keydown="handleKeyDown"
                  :disabled="loading"
                />
                <button
                  type="button"
                  @click="showPassword = !showPassword"
                  class="toggle-password-btn"
                >
                  {{ showPassword ? '👁️' : '👁️‍🗨️' }}
                </button>
              </div>
            </div>

            <!-- 确认密码 -->
            <div class="form-group">
              <label class="form-label">{{ $t('auth.confirmPassword') }}</label>
              <input
                v-model="confirmPassword"
                :type="showPassword ? 'text' : 'password'"
                :placeholder="$t('auth.confirmPasswordPlaceholder')"
                class="form-input"
                @keydown="handleKeyDown"
                :disabled="loading"
              />
            </div>

          </template>

          <!-- ===== 重置密码模式 ===== -->
          <template v-if="mode === 'reset'">
            <!-- 邮箱 -->
            <div class="form-group">
              <label class="form-label">{{ $t('auth.email') }}</label>
              <input
                v-model="email"
                type="email"
                :placeholder="$t('auth.registerEmail')"
                class="form-input"
                @keydown="handleKeyDown"
                :disabled="loading"
              />
            </div>

            <!-- 验证码 -->
            <div class="form-group">
              <label class="form-label">{{ $t('auth.verificationCode') }}</label>
              <div class="code-input-wrapper">
                <input
                  v-model="code"
                  type="text"
                  maxlength="6"
                  :placeholder="$t('auth.codePlaceholder')"
                  class="form-input"
                  @keydown="handleKeyDown"
                  :disabled="loading"
                />
                <button
                  type="button"
                  @click="sendCode"
                  :disabled="!canSendCode"
                  class="resend-btn"
                >
                  <span v-if="sendingCode" class="spinner-small"></span>
                  {{ sendingCode ? '' : (countdown > 0 ? `${countdown}s` : $t('auth.getCode')) }}
                </button>
              </div>
            </div>

            <!-- 开发模式验证码提示 -->
            <div v-if="devCode" class="dev-code-hint">
              {{ $t('auth.devCode') }} <strong>{{ devCode }}</strong>
            </div>

            <!-- 新密码 -->
            <div class="form-group">
              <label class="form-label">{{ $t('auth.newPassword') }}</label>
              <div class="input-wrapper">
                <input
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  :placeholder="$t('auth.newPasswordHint')"
                  class="form-input"
                  @keydown="handleKeyDown"
                  :disabled="loading"
                />
                <button
                  type="button"
                  @click="showPassword = !showPassword"
                  class="toggle-password-btn"
                >
                  {{ showPassword ? '👁️' : '👁️‍🗨️' }}
                </button>
              </div>
            </div>

            <!-- 确认新密码 -->
            <div class="form-group">
              <label class="form-label">{{ $t('auth.confirmNewPassword') }}</label>
              <input
                v-model="confirmPassword"
                :type="showPassword ? 'text' : 'password'"
                :placeholder="$t('auth.confirmNewPasswordPlaceholder')"
                class="form-input"
                @keydown="handleKeyDown"
                :disabled="loading"
              />
            </div>
          </template>
        </template>

        <!-- 成功消息 -->
        <div v-if="successMessage" class="success-alert">
          <span class="success-icon">✓</span>
          <span>{{ successMessage }}</span>
        </div>

        <!-- 错误消息 -->
        <div v-if="error" class="error-alert">
          <span class="error-icon">⚠️</span>
          <span>{{ error }}</span>
        </div>

        <!-- 提交按钮 -->
        <button
          type="submit"
          :disabled="loading"
          class="submit-btn"
        >
          <span v-if="loading" class="spinner"></span>
          {{ submitButtonText }}
        </button>
      </form>

      <!-- OAuth 分割线 + 按钮 (仅登录模式显示) -->
      <template v-if="mode === 'login'">
        <div class="oauth-divider">
          <span class="divider-line"></span>
          <span class="divider-text">{{ $t('auth.orDivider') }}</span>
          <span class="divider-line"></span>
        </div>
        <button
          class="oauth-btn linuxdo-btn"
          @click="handleLinuxDoLogin"
          :disabled="oauthLoading"
        >
          <span v-if="oauthLoading" class="spinner-small"></span>
          <svg v-else class="oauth-icon" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
          </svg>
          {{ $t('auth.loginWithLinuxDo') }}
        </button>
      </template>

      <!-- 底部切换 -->
      <div class="auth-footer">
        <template v-if="mode === 'redeem' || mode === 'reset'">
          <button @click="switchMode('login')" class="switch-mode-btn">
            ← {{ $t('auth.backToLogin') }}
          </button>
        </template>
        <template v-else>
          <p class="footer-text" v-if="mode === 'login'">
            {{ $t('auth.noAccount') }}
            <button @click="switchMode('register')" class="link-btn">{{ $t('auth.registerNow') }}</button>
          </p>
          <p class="footer-text" v-else>
            {{ $t('auth.hasAccount') }}
            <button @click="switchMode('login')" class="link-btn">{{ $t('auth.loginNow') }}</button>
          </p>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.auth-modal-overlay {
  position: fixed;
  inset: 0;
  background: var(--color-overlay-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(8px);
  padding: 20px;
}

.auth-modal {
  width: 100%;
  max-width: 440px;
  max-height: 90vh;
  overflow-y: auto;
  padding: 28px 24px;
  background: var(--color-modal-bg);
  border: 1px solid var(--color-border-medium);
  border-radius: 18px;
  box-shadow: 0 25px 80px var(--color-tint-black-50);
  position: relative;
  animation: modalIn 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

@keyframes modalIn {
  from {
    opacity: 0;
    transform: scale(0.9) translateY(-30px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.close-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 36px;
  height: 36px;
  border: none;
  background: var(--color-tint-white-06);
  border-radius: 10px;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s ease;
  z-index: 10;
}

.close-btn:hover {
  background: var(--color-tint-white-12);
  color: var(--color-text-primary);
  transform: rotate(90deg);
}

.auth-header {
  text-align: center;
  margin-bottom: 32px;
}

.auth-brand {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-bottom: 12px;
}

.logo-icon {
  width: 40px;
  height: 40px;
  object-fit: contain;
}

.auth-header h1 {
  font-family: 'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  font-size: 28px;
  font-weight: 900;
  margin: 0;
  color: white;
  letter-spacing: 0.02em;
  transform: skewX(-3deg);
}

.subtitle {
  color: var(--color-text-secondary);
  font-size: 14px;
  margin: 0;
  letter-spacing: 0.02em;
}

.auth-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 28px;
  background: var(--color-tint-white-03);
  border-radius: 12px;
  padding: 6px;
  border: 1px solid var(--color-tint-white-06);
}

.tab-btn {
  flex: 1;
  padding: 12px 16px;
  border: none;
  background: transparent;
  color: var(--color-text-muted);
  font-size: 14px;
  font-weight: 600;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.tab-btn.active {
  background: #00cae0;
  color: white;
  box-shadow: 0 4px 16px rgba(0, 202, 224, 0.35);
}

.tab-btn:hover:not(.active) {
  color: var(--color-text-secondary);
  background: rgba(0, 202, 224, 0.08);
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  letter-spacing: 0.3px;
  margin-bottom: 2px;
}

.optional {
  font-weight: 400;
  color: var(--color-text-muted);
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.form-input {
  width: 100%;
  padding: 12px 14px;
  font-size: 14px;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-medium);
  border-radius: 12px;
  color: var(--color-text-primary);
  transition: all 0.3s ease;
}

.form-input:focus {
  background: var(--color-input-bg-focus);
  border-color: rgba(0, 202, 224, 0.5);
  outline: none;
  box-shadow: 0 0 0 3px rgba(0, 202, 224, 0.15);
}

.form-input::placeholder {
  color: var(--color-text-muted);
}

.form-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.form-textarea {
  width: 100%;
  padding: 14px 14px;
  font-size: 13px;
  background: var(--color-tint-white-08);
  border: 1.5px solid var(--glass-border);
  border-radius: 10px;
  color: var(--color-text-primary);
  transition: all 0.3s ease;
  font-family: 'Courier New', monospace;
  resize: vertical;
  line-height: 1.6;
  letter-spacing: 0.3px;
  word-break: break-all;
  min-height: 160px;
}

.form-textarea:focus {
  background: var(--color-tint-white-12);
  border-color: rgba(0, 202, 224, 0.6);
  outline: none;
  box-shadow: 0 0 0 3px rgba(0, 202, 224, 0.2);
}

.form-textarea::placeholder {
  color: var(--color-text-muted);
}

.form-textarea:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.toggle-password-btn {
  position: absolute;
  right: 12px;
  background: transparent;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: 16px;
  padding: 4px;
  transition: all 0.2s ease;
}

.toggle-password-btn:hover {
  color: var(--color-accent);
}

.login-method {
  display: flex;
  gap: 8px;
}

.method-btn {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--glass-border);
  background: transparent;
  color: var(--color-text-muted);
  font-size: 12px;
  font-weight: 600;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.method-btn.active {
  border-color: rgba(0, 202, 224, 0.5);
  color: #00cae0;
  background: rgba(0, 202, 224, 0.1);
}

.method-btn:hover:not(.active) {
  border-color: var(--color-tint-white-20);
  color: var(--color-text-secondary);
}

.code-input-wrapper {
  display: flex;
  gap: 10px;
}

.code-input-wrapper .form-input {
  flex: 1;
}

.resend-btn {
  padding: 12px 14px;
  background: rgba(0, 202, 224, 0.1);
  border: 1px solid rgba(0, 202, 224, 0.3);
  color: #00cae0;
  font-size: 12px;
  font-weight: 600;
  border-radius: 10px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.3s ease;
}

.resend-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.resend-btn:hover:not(:disabled) {
  background: rgba(0, 202, 224, 0.2);
}

.back-btn {
  background: transparent;
  border: none;
  color: var(--color-text-muted);
  font-size: 13px;
  cursor: pointer;
  text-align: left;
  padding: 0;
  transition: color 0.2s ease;
}

.back-btn:hover {
  color: var(--color-accent);
}

.dev-code-hint {
  padding: 10px 12px;
  background: var(--color-success-light);
  border: 1px solid rgba(76, 175, 80, 0.3);
  border-radius: 8px;
  color: var(--color-success);
  font-size: 12px;
}

.dev-code-hint strong {
  font-family: 'SF Mono', 'Fira Code', monospace;
  letter-spacing: 2px;
  font-size: 14px;
}

.form-hint {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-muted);
  line-height: 1.5;
}

.agreement-check {
  margin-top: 4px;
}

.agreement-label {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-muted);
}

.agreement-checkbox {
  margin-top: 2px;
  width: 14px;
  height: 14px;
  cursor: pointer;
  accent-color: #00cae0;
  flex-shrink: 0;
}

.agreement-text {
  display: inline;
}

.agreement-link {
  color: var(--color-accent);
  text-decoration: none;
}

.agreement-link:hover {
  color: #00cae0;
  text-decoration: underline;
}

.success-alert {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  background: var(--color-success-light);
  border: 1px solid rgba(76, 175, 80, 0.3);
  border-radius: 10px;
  color: var(--color-success);
  font-size: 13px;
}

.success-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.error-alert {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  background: rgba(255, 107, 107, 0.1);
  border: 1px solid rgba(255, 107, 107, 0.3);
  border-radius: 10px;
  color: var(--color-error-light);
  font-size: 13px;
}

.error-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.submit-btn {
  padding: 12px;
  font-size: 14px;
  font-weight: 700;
  color: white;
  background: #00cae0;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 8px;
  letter-spacing: 0.5px;
}

.submit-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 8px 28px rgba(0, 202, 224, 0.4);
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.spinner-small {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(0, 202, 224, 0.3);
  border-top-color: #00cae0;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.auth-footer {
  text-align: center;
  margin-top: 28px;
  padding-top: 24px;
  border-top: 1px solid var(--color-tint-white-06);
}

.footer-text {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0;
}

.link-btn, .switch-mode-btn {
  background: transparent;
  border: none;
  color: var(--color-accent);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.link-btn:hover, .switch-mode-btn:hover {
  text-decoration: underline;
  color: #00cae0;
}

.forgot-password {
  text-align: right;
  margin-top: 8px;
}

.forgot-password .link-btn {
  font-size: 12px;
  color: var(--color-text-muted);
}

.forgot-password .link-btn:hover {
  color: var(--color-accent);
}

/* OAuth */
.oauth-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 20px 0 16px;
}

.divider-line {
  flex: 1;
  height: 1px;
  background: var(--color-tint-white-08);
}

.divider-text {
  font-size: 12px;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.oauth-btn {
  width: 100%;
  padding: 11px 16px;
  border: 1px solid var(--color-border-medium);
  background: var(--color-tint-white-04);
  color: var(--color-text-secondary);
  font-size: 14px;
  font-weight: 600;
  border-radius: 10px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: all 0.3s ease;
}

.oauth-btn:hover:not(:disabled) {
  background: var(--color-tint-white-08);
  border-color: var(--color-tint-white-20);
  color: var(--color-text-primary);
}

.oauth-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.oauth-icon {
  flex-shrink: 0;
}

/* 响应式 */
/* 平板设备 */
@media (max-width: 768px) {
  .auth-modal {
    max-width: calc(100% - 32px);
    margin: 16px;
    padding: 28px 20px;
  }

  .auth-header h1 {
    font-size: 24px;
  }

  .logo-icon {
    width: 34px;
    height: 34px;
  }
}

/* 移动设备 */
@media (max-width: 480px) {
  .auth-modal {
    max-width: calc(100% - 24px);
    margin: 12px;
    padding: 24px 16px;
  }

  .auth-header h1 {
    font-size: 22px;
  }

  .logo-icon {
    width: 30px;
    height: 30px;
  }

  .form-group label {
    font-size: 12px;
  }

  .form-input {
    padding: 10px 12px;
    font-size: 13px;
  }

  .form-hint {
    font-size: 11px;
  }
}

/* 超小屏幕 */
@media (max-width: 360px) {
  .auth-modal {
    padding: 20px 14px;
  }

  .auth-header h1 {
    font-size: 20px;
  }
}
</style>
