<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'

const { t } = useI18n()
const emit = defineEmits(['success'])

const redeemKey = ref('')
const loading = ref(false)
const error = ref('')
const successMessage = ref('')

const handleRedeem = async () => {
  if (!redeemKey.value.trim()) {
    error.value = t('redeem.enterKey')
    return
  }

  loading.value = true
  error.value = ''
  successMessage.value = ''

  try {
    const token = localStorage.getItem('token')
    const response = await axios.post('/api/user/redeem', {
      key: redeemKey.value
    }, {
      headers: { Authorization: `Bearer ${token}` }
    })

    successMessage.value = t('redeem.success', { credits: response.data.credits_added })
    redeemKey.value = ''

    // 更新本地用户信息
    const user = JSON.parse(localStorage.getItem('user') || '{}')
    user.credits = response.data.current_credits
    localStorage.setItem('user', JSON.stringify(user))

    emit('success', user)
  } catch (e) {
    error.value = e.response?.data?.error || t('redeem.failed')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <form @submit.prevent="handleRedeem" class="redeem-form">
    <div class="form-group">
      <label class="form-label">{{ $t('redeem.key') }}</label>
      <input
        v-model="redeemKey"
        type="text"
        :placeholder="$t('redeem.keyPlaceholder')"
        class="form-input"
        :disabled="loading"
      />
    </div>

    <p class="form-hint">{{ $t('redeem.hint') }}</p>

    <div v-if="successMessage" class="success-alert">
      <span class="success-icon">✓</span>
      <span>{{ successMessage }}</span>
    </div>

    <div v-if="error" class="error-alert">
      <span class="error-icon">⚠️</span>
      <span>{{ error }}</span>
    </div>

    <button type="submit" :disabled="loading" class="submit-btn">
      <span v-if="loading" class="spinner"></span>
      {{ loading ? $t('redeem.redeeming') : $t('redeem.redeem') }}
    </button>
  </form>
</template>

<style scoped>
.redeem-form {
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
  padding: 14px 16px;
  font-size: 14px;
  background: var(--color-tint-white-04);
  border: 1px solid var(--color-tint-white-08);
  border-radius: 12px;
  color: var(--color-text-primary);
  transition: all 0.3s ease;
}

.form-input:focus {
  background: var(--color-tint-white-08);
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

.form-hint {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-muted);
  line-height: 1.5;
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
  animation: fadeIn 0.3s ease;
}

.success-icon {
  font-size: 16px;
}

.error-alert {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 10px;
  color: #ef4444;
  font-size: 13px;
  animation: fadeIn 0.3s ease;
}

.error-icon {
  font-size: 16px;
}

.submit-btn {
  padding: 16px;
  font-size: 15px;
  font-weight: 700;
  color: white;
  background: #00cae0;
  border: none;
  border-radius: 14px;
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
  border: 2px solid var(--color-tint-white-20);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 响应式 */
@media (max-width: 768px) {
  .form-input {
    padding: 10px 12px;
    font-size: 13px;
  }

  .form-hint {
    font-size: 11px;
  }

  .submit-btn {
    padding: 10px;
    font-size: 13px;
  }
}

@media (max-width: 480px) {
  .form-input {
    padding: 9px 11px;
    font-size: 12px;
  }

  .success-alert,
  .error-alert {
    padding: 10px 12px;
    font-size: 12px;
  }

  .submit-btn {
    padding: 10px;
    font-size: 12px;
  }
}
</style>
