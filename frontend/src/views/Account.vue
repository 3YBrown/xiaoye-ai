<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { NCard, NButton, NInput, NDataTable, NEmpty, useMessage } from 'naive-ui'
import { useUserStore } from '../stores/user'
import axios from 'axios'

const { t } = useI18n()
const userStore = useUserStore()
const message = useMessage()

const checkinLoading = ref(false)
const checkinRewards = [5, 6, 7, 8, 9, 10, 15]

const performCheckin = async () => {
  checkinLoading.value = true
  try {
    const result = await userStore.dailyCheckin()
    message.success(t('checkin.success', { credits: result.credits_added }))
    await loadTransactions()
  } catch (e) {
    message.error(e.response?.data?.error || t('checkin.failed'))
  } finally {
    checkinLoading.value = false
  }
}

const redeemKey = ref('')
const redeemLoading = ref(false)
const redeemMessage = ref('')
const redeemError = ref('')

const invitations = ref([])
const loadingInvitations = ref(false)
const invitationPage = ref(1)

const transactions = ref([])
const loadingTransactions = ref(false)
const transactionPage = ref(1)
const isMobileView = ref(false)

const pageSize = 5

const onPageVisible = () => {
  if (document.visibilityState === 'visible') {
    loadInvitations()
    loadTransactions()
  }
}

onMounted(() => {
  updateViewport()
  window.addEventListener('resize', updateViewport)
  document.addEventListener('visibilitychange', onPageVisible)
  loadInvitations()
  loadTransactions()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateViewport)
  document.removeEventListener('visibilitychange', onPageVisible)
})

const updateViewport = () => {
  isMobileView.value = window.innerWidth <= 768
}

const redeemCredits = async () => {
  if (!redeemKey.value.trim()) return
  redeemLoading.value = true
  redeemError.value = ''
  redeemMessage.value = ''
  try {
    const response = await axios.post('/api/user/redeem', { key: redeemKey.value.trim() }, {
      headers: { Authorization: `Bearer ${userStore.token}` }
    })
    redeemMessage.value = t('account.redeemSuccess', { credits: response.data.credits_added })
    redeemKey.value = ''
    await userStore.fetchUserInfo()
    await loadTransactions()
  } catch (e) {
    redeemError.value = e.response?.data?.error || t('account.redeemFailed')
  } finally {
    redeemLoading.value = false
  }
}

const loadInvitations = async () => {
  loadingInvitations.value = true
  try {
    const response = await axios.get('/api/user/invitations', {
      headers: { Authorization: `Bearer ${userStore.token}` }
    })
    invitations.value = response.data.records || []
    invitationPage.value = 1
  } catch (e) {
    console.error('加载邀请记录失败', e)
  } finally {
    loadingInvitations.value = false
  }
}

const txTypeText = (type) => {
  const map = {
    register_gift: t('account.txTypeRegisterGift'),
    invite_reward: t('account.txTypeInviteReward'),
    redeem: t('account.txTypeRedeem'),
    generate_cost: t('account.txTypeGenerateCost'),
    prompt_optimize_cost: t('account.txTypePromptOptimizeCost'),
    reverse_prompt_cost: t('account.txTypeReversePromptCost'),
    refund: t('account.txTypeRefund'),
    inspiration_review_reward: t('account.txTypeInspirationReviewReward'),
    daily_checkin: t('account.txTypeDailyCheckin'),
    online_payment: t('account.txTypeOnlinePayment')
  }
  return map[type] || type
}

const loadTransactions = async () => {
  loadingTransactions.value = true
  try {
    const response = await axios.get('/api/user/credits/transactions?limit=50', {
      headers: { Authorization: `Bearer ${userStore.token}` }
    })
    transactions.value = response.data.transactions || []
    transactionPage.value = 1
  } catch (e) {
    console.error('加载钻石流水失败', e)
  } finally {
    loadingTransactions.value = false
  }
}

const copyInviteLink = () => {
  const link = `${window.location.origin}/?invite=${userStore.inviteCode}`
  navigator.clipboard.writeText(link).catch(() => {})
}

const formatDate = (val) => new Date(val).toLocaleString()

const mobilePagedInvitations = computed(() => {
  const start = (invitationPage.value - 1) * pageSize
  return invitations.value.slice(start, start + pageSize)
})

const mobilePagedTransactions = computed(() => {
  const start = (transactionPage.value - 1) * pageSize
  return transactions.value.slice(start, start + pageSize)
})

const inviteColumns = computed(() => [
  { title: t('account.invitee'), key: 'invitee_email', width: 180 },
  { title: t('account.registerTime'), key: 'created_at', width: 180, render: (row) => formatDate(row.created_at) },
  { title: t('account.reward'), key: 'credits_rewarded', width: 100, render: (row) => `+${row.credits_rewarded || 0} 💎` }
])

const transactionColumns = computed(() => [
  { title: t('account.txTime'), key: 'created_at', width: 180, render: (row) => formatDate(row.created_at) },
  { title: t('account.txType'), key: 'type', width: 120, render: (row) => txTypeText(row.type) },
  { title: t('account.txDelta'), key: 'delta', width: 80, render: (row) => `${row.delta > 0 ? '+' : ''}${row.delta}` },
  { title: t('account.txBalanceAfter'), key: 'balance_after', width: 80 },
  { title: t('account.txNote'), key: 'note', minWidth: 170, render: (row) => row.note || '-' }
])

const invitationTotalPages = computed(() => Math.max(1, Math.ceil(invitations.value.length / pageSize)))
const transactionTotalPages = computed(() => Math.max(1, Math.ceil(transactions.value.length / pageSize)))
</script>

<template>
  <div class="account-page">
    <div class="account-content">
      <div class="account-container">
        <h1 class="page-title">{{ $t('account.title') }}</h1>

        <NCard class="hero-card" :bordered="false">
          <div class="hero-main">
            <div class="hero-user">
              <div class="avatar-large">{{ userStore.userAvatar }}</div>
              <div class="user-meta">
                <div class="user-name">{{ userStore.userNickname }}</div>
                <div class="user-email">{{ userStore.currentUser?.email }}</div>
              </div>
            </div>
            <div class="balance-panel">
              <span class="balance-label">💎 {{ $t('account.diamondBalance') }}</span>
              <span class="balance-number">{{ userStore.userCredits }}</span>
            </div>
          </div>
          <div class="hero-subline">
            <span class="subline-item">{{ $t('account.invitedCount', { count: userStore.currentUser?.invite_count || 0 }) }}</span>
            <span class="subline-dot">•</span>
            <span class="subline-item">{{ $t('account.txTypeRedeem') }}: {{ userStore.currentUser?.total_redeemed || 0 }}</span>
          </div>
        </NCard>

        <div class="content-grid">
          <div class="left-col">
            <NCard :title="'📅 ' + $t('checkin.title')" class="section-card checkin-card" :bordered="false">
              <p class="checkin-subtitle">{{ $t('checkin.subtitle') }}</p>
              <div class="checkin-dots">
                <div
                  v-for="(reward, idx) in checkinRewards"
                  :key="idx"
                  class="checkin-dot"
                  :class="{
                    done: !userStore.dailyCheckinAvailable
                      ? idx < ((userStore.checkinStreak - 1) % 7 + 1)
                      : idx < (userStore.checkinStreak % 7),
                    today: !userStore.dailyCheckinAvailable && idx === ((userStore.checkinStreak - 1) % 7)
                  }"
                >
                  <span class="dot-day">{{ $t('checkin.day', { day: idx + 1 }) }}</span>
                  <span class="dot-reward">+{{ reward }} 💎</span>
                </div>
              </div>
              <div class="checkin-action">
                <div class="checkin-info">
                  <span class="checkin-streak-text" v-if="userStore.checkinStreak > 0">
                    {{ $t('checkin.streak') }}: {{ userStore.checkinStreak }} {{ $t('checkin.days') }}
                  </span>
                  <span class="checkin-reward-text" v-if="userStore.dailyCheckinAvailable">
                    {{ $t('checkin.todayReward') }}: +{{ userStore.nextCheckinReward }} 💎
                  </span>
                </div>
                <NButton
                  type="primary"
                  :loading="checkinLoading"
                  :disabled="!userStore.dailyCheckinAvailable"
                  @click="performCheckin"
                >
                  {{ userStore.dailyCheckinAvailable ? $t('checkin.btn') : $t('checkin.done') }}
                </NButton>
              </div>
            </NCard>

            <NCard :title="'🔑 ' + $t('account.redeemDiamonds')" class="section-card" :bordered="false">
              <div class="redeem-row">
                <NInput
                  v-model:value="redeemKey"
                  :placeholder="$t('account.redeemPlaceholder')"
                  @keydown.enter="redeemCredits"
                />
                <NButton type="primary" :loading="redeemLoading" @click="redeemCredits" :disabled="!redeemKey.trim()">
                  {{ $t('account.redeem') }}
                </NButton>
              </div>
              <div v-if="redeemMessage" class="success-msg">{{ redeemMessage }}</div>
              <div v-if="redeemError" class="error-msg">{{ redeemError }}</div>
            </NCard>

            <NCard :title="'🎁 ' + $t('account.inviteFriends')" class="section-card" :bordered="false">
              <p class="invite-desc">{{ $t('account.inviteDesc') }}</p>
              <div class="invite-code-row">
                <code class="invite-code">{{ userStore.inviteCode }}</code>
                <NButton type="primary" size="small" @click="copyInviteLink">{{ $t('account.copyInviteLink') }}</NButton>
              </div>
              <p class="invite-count">{{ $t('account.invitedCount', { count: userStore.currentUser?.invite_count || 0 }) }}</p>

              <div v-if="invitations.length && !isMobileView" class="records-wrap">
                <NDataTable
                  :columns="inviteColumns"
                  :data="mobilePagedInvitations"
                  size="small"
                />
              </div>
              <div v-else-if="mobilePagedInvitations.length" class="mobile-list">
                <div v-for="row in mobilePagedInvitations" :key="row.id" class="record-card">
                  <div class="record-line strong">{{ row.invitee_email }}</div>
                  <div class="record-line">{{ formatDate(row.created_at) }}</div>
                  <div class="record-line gain">+{{ row.credits_rewarded || 0 }} 💎</div>
                </div>
              </div>
              <NEmpty v-else-if="!loadingInvitations" :description="$t('account.noInvitations')" />
              <div v-if="invitations.length > pageSize" class="more-row">
                <NButton text :disabled="invitationPage <= 1" @click="invitationPage--">上一页</NButton>
                <span class="page-text">{{ invitationPage }} / {{ invitationTotalPages }}</span>
                <NButton text :disabled="invitationPage >= invitationTotalPages" @click="invitationPage++">下一页</NButton>
              </div>
            </NCard>
          </div>

          <div class="right-col">
            <NCard :title="'📒 ' + $t('account.creditRecords')" class="section-card ledger-card" :bordered="false">
              <div v-if="transactions.length && !isMobileView" class="records-wrap">
                <NDataTable
                  :columns="transactionColumns"
                  :data="mobilePagedTransactions"
                  size="small"
                />
              </div>
              <div v-else-if="mobilePagedTransactions.length" class="mobile-list">
                <div v-for="row in mobilePagedTransactions" :key="row.id" class="record-card">
                  <div class="record-top">
                    <span class="record-type">{{ txTypeText(row.type) }}</span>
                    <span class="record-delta" :class="{ loss: row.delta < 0, gain: row.delta > 0 }">
                      {{ row.delta > 0 ? '+' : '' }}{{ row.delta }}
                    </span>
                  </div>
                  <div class="record-line">{{ formatDate(row.created_at) }}</div>
                  <div class="record-line">{{ $t('account.txBalanceAfter') }}: {{ row.balance_after }}</div>
                  <div class="record-line muted">{{ row.note || '-' }}</div>
                </div>
              </div>
              <NEmpty v-else-if="!loadingTransactions" :description="$t('account.noCreditRecords')" />
              <div v-if="transactions.length > pageSize" class="more-row">
                <NButton text :disabled="transactionPage <= 1" @click="transactionPage--">上一页</NButton>
                <span class="page-text">{{ transactionPage }} / {{ transactionTotalPages }}</span>
                <NButton text :disabled="transactionPage >= transactionTotalPages" @click="transactionPage++">下一页</NButton>
              </div>
            </NCard>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.account-page {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.account-content {
  flex: 1;
  padding: 32px 28px;
  overflow-y: auto;
}

.account-container {
  width: 100%;
  max-width: 1460px;
  margin: 0 auto;
}

.page-title {
  font-family: 'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  font-size: 30px;
  font-weight: 900;
  margin-bottom: 22px;
  letter-spacing: 0.02em;
  color: var(--color-text-primary);
  transform: skewX(-3deg);
  display: inline-block;
}

.hero-card {
  margin-bottom: 22px;
  border: 1px solid var(--color-tint-white-10);
  border-radius: 16px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.04), rgba(255, 255, 255, 0.01)),
    radial-gradient(circle at right top, rgba(0, 202, 224, 0.08), transparent 46%),
    var(--color-card-bg, rgba(255, 255, 255, 0.02));
  box-shadow: 0 10px 26px rgba(0, 0, 0, 0.14);
}

.hero-card :deep(.n-card-header) {
  padding: 18px 20px 10px;
}

.hero-card :deep(.n-card) {
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.hero-card :deep(.n-card__content) {
  padding: 18px 20px;
}

.hero-main {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  column-gap: 20px;
  align-items: center;
  min-height: 84px;
}

.hero-user {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.avatar-large {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: rgba(0, 202, 224, 0.15);
  color: #00cae0;
  font-size: 18px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 1.5px solid rgba(0, 202, 224, 0.25);
}

.user-meta {
  min-width: 0;
}

.user-name {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.user-email {
  margin-top: 6px;
  color: var(--color-text-muted);
  font-size: 13px;
  word-break: break-all;
}

.balance-panel {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: flex-end;
  place-self: center end;
  padding: 12px 18px;
  border-radius: 12px;
  background: rgba(251, 191, 36, 0.08);
  border: 1px solid rgba(251, 191, 36, 0.2);
  min-height: 72px;
}

.balance-label {
  font-size: 12px;
  color: var(--color-text-muted);
  text-align: right;
}

.balance-number {
  font-size: 28px;
  font-weight: 800;
  color: #fbbf24;
  line-height: 1.2;
  text-align: right;
}

.hero-subline {
  margin-top: 14px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  color: var(--color-text-muted);
  font-size: 12px;
}

.subline-item {
  opacity: 0.92;
}

.subline-dot {
  opacity: 0.5;
}

.checkin-card {
  background:
    linear-gradient(135deg, rgba(251, 191, 36, 0.06), rgba(251, 191, 36, 0.01)),
    var(--color-card-bg, rgba(255, 255, 255, 0.02));
}

.checkin-subtitle {
  color: var(--color-text-secondary);
  font-size: 13px;
  margin-bottom: 14px;
}

.checkin-dots {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 6px;
  margin-bottom: 16px;
}

.checkin-dot {
  padding: 8px 4px;
  border-radius: 10px;
  border: 1px solid var(--color-tint-white-10);
  background: var(--color-tint-white-03);
  text-align: center;
  transition: all 0.2s;
}

.checkin-dot.done {
  border-color: rgba(251, 191, 36, 0.4);
  background: rgba(251, 191, 36, 0.1);
}

.checkin-dot.today {
  border-color: rgba(34, 197, 94, 0.5);
  background: rgba(34, 197, 94, 0.12);
}

.dot-day {
  display: block;
  font-size: 11px;
  color: var(--color-text-muted);
  margin-bottom: 2px;
}

.checkin-dot.done .dot-day,
.checkin-dot.today .dot-day {
  color: var(--color-text-primary);
  font-weight: 600;
}

.dot-reward {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.checkin-dot.done .dot-reward {
  color: #fbbf24;
}

.checkin-dot.today .dot-reward {
  color: #22c55e;
}

.checkin-action {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.checkin-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.checkin-streak-text {
  font-size: 13px;
  color: var(--color-text-secondary);
}

.checkin-reward-text {
  font-size: 14px;
  font-weight: 600;
  color: #fbbf24;
}

.section-card {
  margin-bottom: 18px;
  border: 1px solid var(--color-tint-white-08);
  border-radius: 14px;
}

.section-card :deep(.n-card-header) {
  padding: 16px 18px 8px;
}

.section-card :deep(.n-card__content) {
  padding: 8px 18px 16px;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.06fr) minmax(0, 1.06fr);
  gap: 18px;
  align-items: start;
}

.left-col,
.right-col {
  min-width: 0;
}

.redeem-row {
  display: flex;
  gap: 12px;
}

.redeem-row :deep(.n-input .n-input-wrapper) {
  border-radius: 12px;
  overflow: hidden;
}

.redeem-row :deep(.n-input .n-input__state-border) {
  border-radius: 12px;
}

.redeem-row :deep(.n-input .n-input-wrapper),
.redeem-row :deep(.n-input .n-input-wrapper:hover),
.redeem-row :deep(.n-input.n-input--focus .n-input-wrapper),
.redeem-row :deep(.n-input.n-input--focus .n-input-wrapper:hover) {
  box-shadow: none !important;
  outline: none !important;
}

.redeem-row :deep(.n-input .n-input-wrapper::before),
.redeem-row :deep(.n-input .n-input-wrapper::after) {
  box-shadow: none !important;
  border: 0 !important;
}

.redeem-row :deep(.n-input .n-input__state-border),
.redeem-row :deep(.n-input .n-input__border) {
  opacity: 0 !important;
  border-color: transparent !important;
  box-shadow: none !important;
}

.redeem-row :deep(.n-input input:focus) {
  outline: none !important;
  box-shadow: none !important;
}

.success-msg {
  color: var(--color-success);
  margin-top: 12px;
  font-size: 13px;
}

.error-msg {
  color: var(--color-error);
  margin-top: 12px;
  font-size: 13px;
}

.invite-desc {
  color: var(--color-text-secondary);
  font-size: 13px;
  margin-bottom: 14px;
}

.invite-code-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.invite-code {
  padding: 9px 14px;
  background: rgba(0, 202, 224, 0.08);
  border: 1px solid rgba(0, 202, 224, 0.25);
  border-radius: 10px;
  font-size: 16px;
  font-weight: 700;
  color: #00cae0;
  letter-spacing: 0.08em;
  font-family: 'SF Mono', 'Fira Code', monospace;
}

.invite-count {
  margin: 14px 0 0;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.records-wrap {
  margin-top: 16px;
  overflow-x: auto;
}

.records-wrap :deep(.n-data-table) {
  min-width: 760px;
}

.ledger-card .records-wrap :deep(.n-data-table) {
  min-width: 860px;
}

.ledger-card {
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.02), rgba(255, 255, 255, 0.008)),
    var(--color-card-bg, rgba(255, 255, 255, 0.02));
}

.ledger-card :deep(.n-card-header) {
  padding-bottom: 8px;
}

.ledger-card :deep(.n-data-table-th) {
  font-weight: 600;
}

.more-row {
  display: flex;
  align-items: center;
  gap: 10px;
  justify-content: center;
  margin-top: 12px;
}

.page-text {
  font-size: 12px;
  color: var(--color-text-muted);
}

.mobile-list {
  margin-top: 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.record-card {
  border: 1px solid var(--color-tint-white-08);
  border-radius: 10px;
  padding: 12px 14px;
  background: var(--color-tint-white-03);
}

.record-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.record-type {
  font-weight: 600;
  font-size: 13px;
}

.record-delta {
  font-weight: 700;
}

.record-line {
  font-size: 12px;
  color: var(--color-text-secondary);
  line-height: 1.5;
}

.record-line.strong {
  font-size: 14px;
  color: var(--color-text-primary);
}

.record-line.muted {
  color: var(--color-text-muted);
}

.gain {
  color: #22c55e;
}

.loss {
  color: #ef4444;
}

@media (max-width: 768px) {
  .account-content {
    padding: 16px 14px;
  }

  .page-title {
    font-size: 24px;
    margin-bottom: 14px;
  }

  .hero-main {
    grid-template-columns: 1fr;
    row-gap: 10px;
  }

  .balance-panel {
    align-items: flex-start;
    align-self: stretch;
  }

  .hero-subline {
    gap: 6px;
    margin-top: 10px;
  }

  .invite-code-row {
    flex-direction: column;
    align-items: stretch;
  }

  .invite-code {
    text-align: center;
  }

  .checkin-dots {
    grid-template-columns: repeat(4, 1fr);
  }

  .checkin-action {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
  }

  .redeem-row {
    flex-direction: column;
    gap: 10px;
  }

  .content-grid {
    grid-template-columns: 1fr;
  }

  .hero-card :deep(.n-card-header) {
    padding: 14px 14px 8px;
  }

  .hero-card :deep(.n-card__content) {
    padding: 14px;
  }

  .section-card :deep(.n-card-header) {
    padding: 12px 12px 6px;
  }

  .section-card :deep(.n-card__content) {
    padding: 6px 12px 12px;
  }

}

@media (max-width: 1160px) {
  .content-grid {
    grid-template-columns: 1fr;
  }
}
</style>
