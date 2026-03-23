<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '../stores/user'
import axios from 'axios'

const { t } = useI18n()
const userStore = useUserStore()

const loading = ref(false)
const sendingCode = ref(false)
const error = ref('')
const successMessage = ref('')
const countdown = ref(0)
const devCode = ref('')

const email = ref('')
const code = ref('')

let countdownTimer = null

const isValidEmail = (val) => {
  return /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(val)
}

const canSendCode = computed(() => {
  return countdown.value === 0 && isValidEmail(email.value) && !sendingCode.value
})

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
    const response = await axios.post('/api/auth/send-code', {
      email: email.value,
      type: 'bind'
    })

    successMessage.value = response.data.message
    startCountdown()

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

  if (!isValidEmail(email.value)) {
    error.value = t('auth.invalidEmail')
    return
  }
  if (!code.value || code.value.length !== 6) {
    error.value = t('auth.enterCode')
    return
  }

  loading.value = true
  try {
    await userStore.bindEmail(email.value, code.value)
    successMessage.value = t('bind.success')
    // Brief delay to show success message, then close
    setTimeout(() => {
      userStore.closeBindEmail()
    }, 1000)
  } catch (e) {
    error.value = e.response?.data?.error || t('bind.failed')
  } finally {
    loading.value = false
  }
}

const handleKeyDown = (e) => {
  if (e.key === 'Enter' && !loading.value) {
    handleSubmit()
  }
}
</script>

<template>
  <div class="auth-modal-overlay">
    <div class="auth-modal">
      <!-- No close button - mandatory binding -->

      <!-- Header -->
      <div class="auth-header">
        <div class="auth-brand">
          <img src="/images/icon.png" alt="小野 AI" class="logo-icon" />
          <h1>小野 AI</h1>
        </div>
        <p class="subtitle">{{ $t('bind.title') }}</p>
      </div>

      <!-- Form -->
      <form @submit.prevent="handleSubmit" class="auth-form">
        <!-- Email -->
        <div class="form-group">
          <label class="form-label">{{ $t('bind.email') }}</label>
          <input
            v-model="email"
            type="email"
            :placeholder="$t('bind.emailPlaceholder')"
            class="form-input"
            @keydown="handleKeyDown"
            :disabled="loading"
          />
        </div>

        <!-- Verification code -->
        <div class="form-group">
          <label class="form-label">{{ $t('bind.code') }}</label>
          <div class="code-input-wrapper">
            <input
              v-model="code"
              type="text"
              maxlength="6"
              :placeholder="$t('bind.codePlaceholder')"
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
              {{ sendingCode ? '' : (countdown > 0 ? `${countdown}s` : $t('bind.getCode')) }}
            </button>
          </div>
        </div>

        <!-- Dev mode code hint -->
        <div v-if="devCode" class="dev-code-hint">
          {{ $t('auth.devCode') }} <strong>{{ devCode }}</strong>
        </div>

        <!-- Success message -->
        <div v-if="successMessage" class="success-alert">
          <span class="success-icon">✓</span>
          <span>{{ successMessage }}</span>
        </div>

        <!-- Error message -->
        <div v-if="error" class="error-alert">
          <span class="error-icon">⚠️</span>
          <span>{{ error }}</span>
        </div>

        <!-- Submit button -->
        <button
          type="submit"
          :disabled="loading"
          class="submit-btn"
        >
          <span v-if="loading" class="spinner"></span>
          {{ loading ? $t('bind.submitting') : $t('bind.submit') }}
        </button>
      </form>

      <!-- Footer hint + logout -->
      <div class="auth-footer">
        <p class="footer-text hint-text">{{ $t('bind.hint') }}</p>
        <button class="logout-btn" @click="userStore.logout()">{{ $t('sidebar.logout') }}</button>
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

.hint-text {
  font-size: 12px;
}

.logout-btn {
  margin-top: 12px;
  background: transparent;
  border: none;
  color: var(--color-text-muted);
  font-size: 12px;
  cursor: pointer;
  transition: color 0.2s ease;
}

.logout-btn:hover {
  color: var(--color-error-light);
}

/* Responsive */
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
}

@media (max-width: 360px) {
  .auth-modal {
    padding: 20px 14px;
  }

  .auth-header h1 {
    font-size: 20px;
  }
}
</style>
