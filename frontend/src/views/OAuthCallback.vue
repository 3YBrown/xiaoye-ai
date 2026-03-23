<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'

const { t } = useI18n()

const status = ref('loading') // 'loading', 'success', 'error'
const errorMsg = ref('')

onMounted(async () => {
  const params = new URLSearchParams(window.location.search)
  const code = params.get('code')
  const state = params.get('state')

  if (!code || !state) {
    status.value = 'error'
    errorMsg.value = t('auth.oauthFailed')
    return
  }

  try {
    const response = await axios.post('/api/auth/oauth/linuxdo/callback', {
      code,
      state
    })

    const { token, user } = response.data
    status.value = 'success'

    if (window.opener) {
      // Popup mode: notify parent window
      window.opener.postMessage({
        type: 'oauth-login-success',
        token,
        user
      }, window.location.origin)
      setTimeout(() => window.close(), 1000)
    } else {
      // Direct navigation (no popup): save to localStorage and redirect
      localStorage.setItem('token', token)
      localStorage.setItem('user', JSON.stringify(user))
      setTimeout(() => {
        window.location.href = '/inspiration'
      }, 800)
    }
  } catch (e) {
    status.value = 'error'
    errorMsg.value = e.response?.data?.error || t('auth.oauthFailed')
  }
})

const retry = () => {
  window.close()
}
</script>

<template>
  <div class="oauth-callback">
    <div class="callback-card">
      <img src="/images/icon.png" alt="小野 AI" class="logo" />

      <template v-if="status === 'loading'">
        <div class="spinner"></div>
        <p class="status-text">{{ $t('auth.oauthLoading') }}</p>
      </template>

      <template v-if="status === 'success'">
        <div class="success-icon">✓</div>
        <p class="status-text success">{{ $t('auth.oauthSuccess') }}</p>
      </template>

      <template v-if="status === 'error'">
        <div class="error-icon">✕</div>
        <p class="status-text error">{{ errorMsg }}</p>
        <button @click="retry" class="retry-btn">{{ $t('auth.oauthRetry') }}</button>
      </template>
    </div>
  </div>
</template>

<style scoped>
.oauth-callback {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0a0a0f;
}

.callback-card {
  text-align: center;
  padding: 48px 40px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 18px;
  min-width: 320px;
}

.logo {
  width: 48px;
  height: 48px;
  margin-bottom: 24px;
}

.spinner {
  width: 36px;
  height: 36px;
  border: 3px solid rgba(0, 202, 224, 0.2);
  border-top-color: #00cae0;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin: 0 auto 16px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.status-text {
  color: rgba(255, 255, 255, 0.7);
  font-size: 15px;
  margin: 0;
}

.status-text.success {
  color: #4caf50;
}

.status-text.error {
  color: #ff6b6b;
}

.success-icon {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(76, 175, 80, 0.15);
  color: #4caf50;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: bold;
  margin: 0 auto 16px;
}

.error-icon {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(255, 107, 107, 0.15);
  color: #ff6b6b;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: bold;
  margin: 0 auto 16px;
}

.retry-btn {
  margin-top: 20px;
  padding: 10px 28px;
  background: #00cae0;
  border: none;
  border-radius: 8px;
  color: white;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
}

.retry-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 16px rgba(0, 202, 224, 0.3);
}
</style>
