<script setup>
import { useI18n } from 'vue-i18n'
import { useUserStore } from '../stores/user'
import RedeemForm from './RedeemForm.vue'

const { t } = useI18n()
const emit = defineEmits(['close', 'success', 'open-pricing'])
const userStore = useUserStore()

const handleSuccess = () => {
  userStore.fetchUserInfo()
  emit('success')
}
</script>

<template>
  <div class="redeem-modal-overlay" @click.self="emit('close')">
    <div class="redeem-modal">
      <!-- 关闭按钮 -->
      <button class="close-btn" @click="emit('close')">✕</button>

      <!-- Header -->
      <div class="redeem-header">
        <div class="header-icon">💎</div>
        <h2>{{ $t('redeem.title') }}</h2>
        <p class="subtitle">{{ $t('redeem.subtitle') }}</p>
      </div>

      <!-- 余额展示 -->
      <div class="redeem-balance">
        <span class="balance-label">{{ $t('redeem.currentBalance') }}</span>
        <span class="balance-value">{{ userStore.userCredits }}</span>
        <span class="balance-unit">💎</span>
      </div>

      <!-- 兑换表单 -->
      <RedeemForm @success="handleSuccess" />

      <!-- 购买引导 -->
      <div class="buy-section">
        <div class="buy-divider">
          <span>{{ $t('redeem.noKey') }}</span>
        </div>
        <button class="buy-link" @click="emit('open-pricing')">
          💎 {{ $t('redeem.viewPricing') }} →
        </button>
        <p class="buy-tip">{{ $t('redeem.buyTip') }}</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.redeem-modal-overlay {
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

.redeem-modal {
  width: 100%;
  max-width: 520px;
  max-height: 90vh;
  overflow-y: auto;
  padding: 28px 24px;
  background: var(--color-modal-bg);
  border: 1px solid var(--color-tint-white-08);
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

/* Header */
.redeem-header {
  text-align: center;
  margin-bottom: 24px;
}

.header-icon {
  font-size: 56px;
  margin-bottom: 16px;
  animation: float 3s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-6px); }
}

.redeem-header h2 {
  font-family: 'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  font-size: 20px;
  font-weight: 900;
  margin: 0 0 8px 0;
  color: var(--color-text-primary);
  letter-spacing: 0.02em;
  transform: skewX(-3deg);
  display: inline-block;
}

.subtitle {
  color: var(--color-text-secondary);
  font-size: 14px;
  margin: 0;
  letter-spacing: 0.02em;
}

/* 余额 */
.redeem-balance {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 14px;
  background: var(--color-tint-white-03);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 10px;
  margin-bottom: 20px;
}

.balance-label {
  color: var(--color-text-secondary);
  font-size: 14px;
}

.balance-value {
  color: #fbbf24;
  font-size: 22px;
  font-weight: 700;
}

.balance-unit {
  font-size: 16px;
}

/* 购买区域 */
.buy-section {
  margin-top: 24px;
  text-align: center;
}

.buy-divider {
  position: relative;
  margin: 20px 0;
  text-align: center;
}

.buy-divider::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 0;
  right: 0;
  height: 1px;
  background: var(--color-tint-white-08);
}

.buy-divider span {
  position: relative;
  background: var(--color-bg-card, #1a1a2e);
  padding: 0 16px;
  color: var(--color-text-muted);
  font-size: 13px;
}

.buy-link {
  display: inline-block;
  padding: 8px 20px;
  background: linear-gradient(135deg, #667eea, #764ba2);
  color: white;
  border: none;
  border-radius: 10px;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}

.buy-link:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
}

.buy-tip {
  margin-top: 10px;
  font-size: 12px;
  color: var(--color-text-muted);
}

/* 响应式 */
@media (max-width: 768px) {
  .redeem-modal {
    max-width: calc(100% - 32px);
    padding: 28px 20px;
  }
}

@media (max-width: 480px) {
  .redeem-modal {
    max-width: calc(100% - 24px);
    padding: 24px 16px;
  }

  .redeem-header h2 {
    font-size: 20px;
  }
}
</style>
