<script setup>
import { ref, computed, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '../stores/user'

const { t } = useI18n()
const emit = defineEmits(['close', 'open-redeem'])
const userStore = useUserStore()

const showFishCard = ref(false)

// Payment state
const paymentLoading = ref(false)
const paymentOrderNo = ref('')
const paymentStatus = ref('') // '' | 'waiting' | 'success' | 'failed' | 'expired'
const paymentPlanName = ref('')
const paymentDiamonds = ref(0)
let pollTimer = null
let pollStartTime = null
const POLL_TIMEOUT = 5 * 60 * 1000 // 5 minutes

// Plans with linux.do credit pricing
const plans = computed(() => [
  {
    name: t('pricing.starter'),
    key: 'starter',
    price: 12.9,
    credits: 129,
    diamonds: 100,
    unitPrice: '0.129',
    features: [t('pricing.gen2k', { count: 16 }), t('pricing.gen4k', { count: 10 }), t('pricing.lightUse')]
  },
  {
    name: t('pricing.popular'),
    key: 'popular',
    price: 55.9,
    credits: 559,
    diamonds: 500,
    unitPrice: '0.112',
    recommended: true,
    badge: t('pricing.bestValue'),
    features: [t('pricing.gen2k', { count: 83 }), t('pricing.gen4k', { count: 50 }), t('pricing.bestValue')]
  },
  {
    name: t('pricing.pro'),
    key: 'pro',
    price: 99.9,
    credits: 999,
    diamonds: 1000,
    unitPrice: '0.100',
    features: [t('pricing.gen2k', { count: 166 }), t('pricing.gen4k', { count: 100 }), t('pricing.heavyCreator')]
  }
])

const handleBuy = () => {
  showFishCard.value = true
}

const handleCreditPay = async (plan) => {
  if (paymentLoading.value) return
  paymentLoading.value = true
  paymentPlanName.value = plan.name
  paymentDiamonds.value = plan.diamonds

  // Pre-open window synchronously in the click handler to avoid popup blocker.
  // We'll navigate it once we have the payment URL.
  const payWindow = window.open('about:blank', '_blank')

  try {
    const data = await userStore.createPaymentOrder(plan.key)
    if (!data || !data.payment_url) {
      if (payWindow) payWindow.close()
      paymentStatus.value = 'failed'
      paymentLoading.value = false
      return
    }

    paymentOrderNo.value = data.order_no
    paymentStatus.value = 'waiting'
    paymentLoading.value = false

    // Navigate the pre-opened window to the payment URL
    if (payWindow) {
      payWindow.location.href = data.payment_url
    } else {
      // Fallback: if popup was still blocked, use location directly
      window.open(data.payment_url, '_blank')
    }

    // Start polling for status
    pollStartTime = Date.now()
    startPolling()
  } catch (e) {
    console.error('创建支付订单失败:', e)
    if (payWindow) payWindow.close()
    paymentStatus.value = 'failed'
    paymentLoading.value = false
  }
}

const startPolling = () => {
  stopPolling()
  pollTimer = setInterval(async () => {
    if (Date.now() - pollStartTime > POLL_TIMEOUT) {
      stopPolling()
      paymentStatus.value = 'expired'
      return
    }

    try {
      const data = await userStore.getPaymentStatus(paymentOrderNo.value)
      if (data && data.status === 'paid') {
        stopPolling()
        paymentStatus.value = 'success'
        await userStore.fetchUserInfo()
      }
    } catch (e) {
      console.error('查询支付状态失败:', e)
    }
  }, 3000)
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

const resetPayment = () => {
  paymentStatus.value = ''
  paymentOrderNo.value = ''
  paymentPlanName.value = ''
  paymentDiamonds.value = 0
  stopPolling()
}

onBeforeUnmount(() => {
  stopPolling()
})
</script>

<template>
  <div class="pricing-overlay" @click.self="emit('close')">
    <div class="pricing-modal">
      <!-- 关闭按钮 -->
      <button class="close-btn" @click="emit('close')">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>

      <!-- Payment Status Overlay -->
      <div v-if="paymentStatus" class="payment-status-overlay">
        <!-- Waiting -->
        <div v-if="paymentStatus === 'waiting'" class="payment-status-card">
          <div class="status-spinner"></div>
          <h3>{{ $t('payment.waitingTitle') }}</h3>
          <p class="status-sub">{{ $t('payment.waitingDesc') }}</p>
          <div class="status-order">{{ $t('payment.orderNo') }}: {{ paymentOrderNo }}</div>
          <button class="status-cancel-btn" @click="resetPayment">{{ $t('payment.cancel') }}</button>
        </div>

        <!-- Success -->
        <div v-else-if="paymentStatus === 'success'" class="payment-status-card">
          <div class="status-icon success-icon">✓</div>
          <h3>{{ $t('payment.successTitle') }}</h3>
          <p class="status-sub">{{ $t('payment.successDesc', { diamonds: paymentDiamonds }) }}</p>
          <button class="status-done-btn" @click="resetPayment">{{ $t('payment.done') }}</button>
        </div>

        <!-- Failed -->
        <div v-else-if="paymentStatus === 'failed'" class="payment-status-card">
          <div class="status-icon fail-icon">✕</div>
          <h3>{{ $t('payment.failedTitle') }}</h3>
          <p class="status-sub">{{ $t('payment.failedDesc') }}</p>
          <button class="status-done-btn" @click="resetPayment">{{ $t('payment.retry') }}</button>
        </div>

        <!-- Expired -->
        <div v-else-if="paymentStatus === 'expired'" class="payment-status-card">
          <div class="status-icon fail-icon">⏱</div>
          <h3>{{ $t('payment.expiredTitle') }}</h3>
          <p class="status-sub">{{ $t('payment.expiredDesc') }}</p>
          <button class="status-done-btn" @click="resetPayment">{{ $t('payment.retry') }}</button>
        </div>
      </div>

      <!-- Header -->
      <div class="pricing-header">
        <h2>{{ $t('pricing.title') }} <span class="header-accent">{{ $t('pricing.titleAccent') }}</span></h2>
        <p class="subtitle">{{ $t('pricing.subtitle') }} <button class="inline-link" @click="emit('open-redeem')">{{ $t('pricing.keyRedeem') }}</button></p>
      </div>

      <!-- 定价卡片 -->
      <div class="pricing-cards">
        <div
          v-for="plan in plans"
          :key="plan.key"
          :class="['pricing-card', { recommended: plan.recommended }]"
        >
          <div v-if="plan.recommended" class="recommend-badge">{{ plan.badge }}</div>

          <div class="card-top">
            <div class="card-name">{{ plan.name }}</div>
            <div class="card-price-row">
              <span class="currency">¥</span>
              <span class="amount">{{ plan.price }}</span>
            </div>
            <div class="card-unit">{{ $t('pricing.unitPrice', { price: plan.unitPrice }) }}</div>
          </div>

          <div class="card-divider"></div>

          <div class="card-diamonds">
            <span class="diamond-icon">💎</span>
            <span class="diamond-count">{{ plan.diamonds }}</span>
            <span class="diamond-label">{{ $t('pricing.diamonds') }}</span>
          </div>

          <ul class="card-features">
            <li v-for="feature in plan.features" :key="feature">
              <span class="check">✓</span>
              {{ feature }}
            </li>
          </ul>

          <!-- Buttons area -->
          <div class="card-buttons">
            <button
              v-if="userStore.isLinuxDoUser"
              :class="['buy-btn']"
              :disabled="paymentLoading"
              @click="handleCreditPay(plan)"
            >
              {{ paymentLoading ? $t('payment.processing') : $t('payment.creditPay', { credits: plan.credits }) }}
            </button>
            <button
              :class="['buy-btn', { primary: plan.recommended && !userStore.isLinuxDoUser }]"
              @click="handleBuy"
            >
              {{ userStore.isLinuxDoUser ? $t('payment.keyPurchase') : (plan.recommended ? $t('pricing.buyNow') : $t('pricing.goToBuy')) }}
            </button>
          </div>
        </div>
      </div>

      <!-- 底部引导 -->
      <div class="pricing-footer">
        <span class="footer-text">{{ $t('pricing.hasKey') }}</span>
        <button class="redeem-trigger" @click="emit('open-redeem')">
          🔑 {{ $t('pricing.redeemNow') }}
        </button>
      </div>

      <!-- 闲鱼商品卡片弹窗 -->
      <div v-if="showFishCard" class="fish-overlay" @click.self="showFishCard = false">
        <div class="fish-card">
          <button class="fish-close" @click="showFishCard = false">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
          <img src="/images/fish.jpg" alt="闲鱼商品" class="fish-image" />
          <p class="fish-tip">{{ $t('pricing.fishTip') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ====== Overlay ====== */
.pricing-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(12px) saturate(120%);
  padding: 24px;
}

/* ====== Modal ====== */
.pricing-modal {
  width: 100%;
  max-width: 1060px;
  max-height: 90vh;
  overflow-y: auto;
  padding: 52px 52px 40px;
  background: rgba(18, 18, 32, 0.92);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 24px;
  box-shadow:
    0 40px 120px rgba(0, 0, 0, 0.6),
    0 0 0 1px rgba(255, 255, 255, 0.04),
    0 0 80px rgba(0, 202, 224, 0.04);
  position: relative;
  animation: modalIn 0.45s cubic-bezier(0.16, 1, 0.3, 1);
  backdrop-filter: blur(40px) saturate(150%);
}

@keyframes modalIn {
  from {
    opacity: 0;
    transform: scale(0.92) translateY(-20px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

/* ====== Close ====== */
.close-btn {
  position: absolute;
  top: 18px;
  right: 18px;
  width: 36px;
  height: 36px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.04);
  border-radius: 50%;
  color: rgba(255, 255, 255, 0.4);
  cursor: pointer;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.25s ease;
  z-index: 10;
}

.close-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.8);
  transform: rotate(90deg);
}

/* ====== Header ====== */
.pricing-header {
  text-align: center;
  margin-bottom: 46px;
}

.pricing-header h2 {
  font-size: 26px;
  font-weight: 800;
  margin: 0 0 12px 0;
  color: #fff;
  letter-spacing: -0.01em;
}

.header-accent {
  background: linear-gradient(135deg, #00cae0, #00e4a0);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.subtitle {
  color: rgba(255, 255, 255, 0.45);
  font-size: 14px;
  margin: 0;
}

.inline-link {
  background: none;
  border: none;
  color: #00cae0;
  font-size: 14px;
  cursor: pointer;
  padding: 0;
  text-decoration: underline;
  text-decoration-color: rgba(0, 202, 224, 0.3);
  text-underline-offset: 2px;
  transition: color 0.2s;
}

.inline-link:hover {
  color: #5cf0ff;
}

/* ====== Cards Grid ====== */
.pricing-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 22px;
  margin-bottom: 40px;
}

/* ====== Card ====== */
.pricing-card {
  position: relative;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 18px;
  padding: 36px 28px 32px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  transition: all 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}

.pricing-card:hover {
  transform: translateY(-6px);
  background: rgba(255, 255, 255, 0.05);
  box-shadow: 0 20px 48px rgba(0, 0, 0, 0.3);
}

.pricing-card.recommended {
  border-color: rgba(0, 202, 224, 0.5);
  background: linear-gradient(180deg, rgba(0, 202, 224, 0.1) 0%, rgba(0, 202, 224, 0.02) 100%);
  box-shadow:
    0 0 40px rgba(0, 202, 224, 0.08),
    inset 0 1px 0 rgba(0, 202, 224, 0.15);
}

.pricing-card.recommended:hover {
  box-shadow:
    0 20px 48px rgba(0, 0, 0, 0.3),
    0 0 60px rgba(0, 202, 224, 0.12);
  border-color: rgba(0, 202, 224, 0.7);
}

/* ====== Badge ====== */
.recommend-badge {
  position: absolute;
  top: -1px;
  left: 50%;
  transform: translateX(-50%);
  background: linear-gradient(135deg, #00cae0, #00b4d8);
  color: white;
  font-size: 11px;
  font-weight: 700;
  padding: 4px 20px;
  border-radius: 0 0 10px 10px;
  letter-spacing: 0.08em;
  box-shadow: 0 4px 12px rgba(0, 202, 224, 0.3);
}

/* ====== Card Top ====== */
.card-top {
  width: 100%;
  margin-bottom: 0;
}

.card-name {
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.5);
  margin-top: 4px;
  margin-bottom: 24px;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.card-price-row {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 2px;
  margin-bottom: 6px;
}

.currency {
  font-size: 18px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.85);
  align-self: flex-start;
  margin-top: 6px;
}

.amount {
  font-size: 46px;
  font-weight: 800;
  color: #fff;
  line-height: 1;
  letter-spacing: -0.02em;
}

.recommended .amount {
  background: linear-gradient(135deg, #fff, #a0f0ff);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.card-credit-price {
  font-size: 13px;
  color: rgba(0, 202, 224, 0.8);
  font-weight: 600;
  margin-bottom: 4px;
}

.card-unit {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.3);
  margin-bottom: 0;
}

/* ====== Card Divider ====== */
.card-divider {
  width: 40px;
  height: 1px;
  background: rgba(255, 255, 255, 0.08);
  margin: 20px 0;
}

.recommended .card-divider {
  background: rgba(0, 202, 224, 0.25);
  width: 50px;
}

/* ====== Diamonds ====== */
.card-diamonds {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 24px;
}

.diamond-icon {
  font-size: 22px;
}

.diamond-count {
  font-size: 28px;
  font-weight: 800;
  background: linear-gradient(135deg, #fbbf24, #f59e0b);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  line-height: 1;
}

.diamond-label {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.35);
  font-weight: 500;
}

/* ====== Features ====== */
.card-features {
  list-style: none;
  padding: 0;
  margin: 0 0 28px 0;
  width: 100%;
  text-align: left;
}

.card-features li {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.5);
  padding: 5px 0;
}

.card-features .check {
  color: rgba(0, 202, 224, 0.7);
  font-weight: 700;
  font-size: 12px;
  flex-shrink: 0;
}

/* ====== Card Buttons ====== */
.card-buttons {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: auto;
}

/* ====== Buy Button ====== */
.buy-btn {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  border-radius: 12px;
  color: rgba(255, 255, 255, 0.75);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.25s ease;
}

.buy-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.18);
  color: #fff;
  transform: translateY(-1px);
}

.buy-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

.buy-btn.primary {
  background: linear-gradient(135deg, #00cae0, #00b4d8);
  border: none;
  color: white;
  font-size: 15px;
  padding: 13px 16px;
  box-shadow: 0 6px 20px rgba(0, 202, 224, 0.3);
}

.buy-btn.primary:hover {
  box-shadow: 0 8px 32px rgba(0, 202, 224, 0.45);
  transform: translateY(-2px);
}

.buy-btn.credit-btn {
  background: linear-gradient(135deg, #667eea, #764ba2);
  border: none;
  color: white;
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.3);
}

.buy-btn.credit-btn.primary {
  background: linear-gradient(135deg, #667eea, #764ba2);
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.3);
}

.buy-btn.credit-btn:hover {
  box-shadow: 0 8px 32px rgba(102, 126, 234, 0.45);
  transform: translateY(-2px);
}

.buy-btn.secondary {
  font-size: 13px;
  padding: 10px 16px;
  color: rgba(255, 255, 255, 0.5);
}

/* ====== Footer ====== */
.pricing-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding-top: 8px;
}

.footer-text {
  color: rgba(255, 255, 255, 0.3);
  font-size: 13px;
}

.redeem-trigger {
  background: none;
  border: none;
  color: #00cae0;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  padding: 4px 0;
  transition: color 0.2s;
}

.redeem-trigger:hover {
  color: #5cf0ff;
}

/* ====== Payment Status Overlay ====== */
.payment-status-overlay {
  position: absolute;
  inset: 0;
  background: rgba(18, 18, 32, 0.96);
  border-radius: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 20;
  animation: fadeIn 0.25s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.payment-status-card {
  text-align: center;
  max-width: 360px;
  padding: 40px;
}

.payment-status-card h3 {
  color: #fff;
  font-size: 20px;
  font-weight: 700;
  margin: 20px 0 12px;
}

.status-sub {
  color: rgba(255, 255, 255, 0.5);
  font-size: 14px;
  margin: 0 0 20px;
  line-height: 1.6;
}

.status-order {
  color: rgba(255, 255, 255, 0.3);
  font-size: 12px;
  font-family: monospace;
  margin-bottom: 24px;
}

.status-spinner {
  width: 48px;
  height: 48px;
  border: 3px solid rgba(255, 255, 255, 0.1);
  border-top-color: #00cae0;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.status-icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: 800;
  margin: 0 auto;
}

.success-icon {
  background: rgba(0, 202, 224, 0.15);
  color: #00cae0;
}

.fail-icon {
  background: rgba(255, 100, 100, 0.15);
  color: #ff6464;
}

.status-cancel-btn,
.status-done-btn {
  padding: 10px 32px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.status-cancel-btn {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.6);
}

.status-cancel-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.status-done-btn {
  background: linear-gradient(135deg, #00cae0, #00b4d8);
  border: none;
  color: white;
  box-shadow: 0 4px 16px rgba(0, 202, 224, 0.3);
}

.status-done-btn:hover {
  box-shadow: 0 6px 24px rgba(0, 202, 224, 0.45);
  transform: translateY(-1px);
}

@media (max-width: 1100px) {
  .pricing-modal {
    max-width: 940px;
    padding: 44px 34px 34px;
  }

  .pricing-cards {
    gap: 16px;
    margin-bottom: 34px;
  }

  .pricing-card {
    padding: 30px 22px 26px;
  }

  .amount {
    font-size: 42px;
  }
}

/* ====== Fish Card ====== */
.fish-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1100;
  backdrop-filter: blur(8px);
}

.fish-card {
  position: relative;
  max-width: 360px;
  width: 90%;
  animation: modalIn 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.fish-close {
  position: absolute;
  top: -44px;
  right: 0;
  width: 34px;
  height: 34px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  background: rgba(255, 255, 255, 0.08);
  border-radius: 50%;
  color: rgba(255, 255, 255, 0.6);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.fish-close:hover {
  background: rgba(255, 255, 255, 0.15);
  color: white;
  transform: scale(1.1);
}

.fish-image {
  width: 100%;
  border-radius: 16px;
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
}

.fish-tip {
  text-align: center;
  color: rgba(255, 255, 255, 0.55);
  font-size: 13px;
  margin: 14px 0 0;
}

/* ====== Responsive ====== */
@media (max-width: 768px) {
  .pricing-modal {
    max-width: calc(100% - 32px);
    padding: 36px 20px 28px;
  }

  .pricing-header h2 {
    font-size: 22px;
  }

  .pricing-cards {
    grid-template-columns: 1fr;
    gap: 14px;
    max-width: 340px;
    margin-left: auto;
    margin-right: auto;
  }

  .pricing-card {
    padding: 28px 20px 24px;
  }

  .amount {
    font-size: 36px;
  }
}

@media (max-width: 480px) {
  .pricing-modal {
    max-width: calc(100% - 24px);
    padding: 32px 16px 24px;
  }

  .pricing-header {
    margin-bottom: 28px;
  }
}
</style>
