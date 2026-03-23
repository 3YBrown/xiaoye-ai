<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'

const { t } = useI18n()
const emit = defineEmits(['close', 'credits-updated'])

// 状态
const loading = ref(true)
const error = ref('')
const inviteData = ref(null)

// 计算属性
const inviteLink = computed(() => {
  if (!inviteData.value?.invite_code) return ''
  const baseUrl = window.location.origin
  return `${baseUrl}?invite=${inviteData.value.invite_code}`
})

// 获取邀请数据
const fetchInvitationData = async () => {
  loading.value = true
  error.value = ''

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get('/api/user/invitations', {
      headers: { Authorization: `Bearer ${token}` }
    })
    inviteData.value = response.data
  } catch (e) {
    error.value = e.response?.data?.error || t('invite.fetchFailed')
  } finally {
    loading.value = false
  }
}

// 复制邀请码
const copyInviteCode = async () => {
  try {
    await navigator.clipboard.writeText(inviteData.value.invite_code)
    showCopySuccess(t('invite.codeCopied'))
  } catch (e) {
    console.error('复制失败:', e)
  }
}

// 复制邀请链接
const copyInviteLink = async () => {
  try {
    await navigator.clipboard.writeText(inviteLink.value)
    showCopySuccess(t('invite.linkCopied'))
  } catch (e) {
    console.error('复制失败:', e)
  }
}

// 显示复制成功提示
const copySuccessMessage = ref('')
const showCopySuccess = (message) => {
  copySuccessMessage.value = message
  setTimeout(() => {
    copySuccessMessage.value = ''
  }, 2000)
}

// 格式化日期
const formatDate = (dateStr) => {
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(() => {
  fetchInvitationData()
})
</script>

<template>
  <div class="invite-modal-overlay" @click.self="$emit('close')">
    <div class="invite-modal">
      <!-- 关闭按钮 -->
      <button class="close-btn" @click="$emit('close')">✕</button>

      <!-- Header -->
      <div class="invite-header">
        <div class="header-icon">🎁</div>
        <h2>{{ $t('invite.title') }}</h2>
        <p class="subtitle">{{ $t('invite.subtitle') }}</p>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="loading-state">
        <div class="spinner"></div>
        <p>{{ $t('invite.loading') }}</p>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="error-state">
        <span class="error-icon">⚠️</span>
        <p>{{ error }}</p>
        <button @click="fetchInvitationData" class="retry-btn">{{ $t('invite.retry') }}</button>
      </div>

      <!-- Content -->
      <template v-else-if="inviteData">
        <!-- 统计卡片 -->
        <div class="stats-grid">
          <div class="stat-card">
            <span class="stat-icon">👥</span>
            <div class="stat-info">
              <span class="stat-value">{{ inviteData.invite_count }}</span>
              <span class="stat-label">{{ $t('invite.invitedCount') }}</span>
            </div>
          </div>
          <div class="stat-card">
            <span class="stat-icon">💎</span>
            <div class="stat-info">
              <span class="stat-value">{{ inviteData.total_credits }}</span>
              <span class="stat-label">{{ $t('invite.diamondsEarned') }}</span>
            </div>
          </div>
        </div>

        <!-- 邀请码区域 -->
        <div class="invite-code-section">
          <label class="section-label">{{ $t('invite.myInviteCode') }}</label>
          <div class="code-display">
            <span class="code-text">{{ inviteData.invite_code }}</span>
            <button @click="copyInviteCode" class="copy-btn" :title="$t('invite.copyCode')">
              📋
            </button>
          </div>
        </div>

        <!-- 邀请链接区域 -->
        <div class="invite-link-section">
          <label class="section-label">{{ $t('invite.inviteLink') }}</label>
          <div class="link-display">
            <span class="link-text">{{ inviteLink }}</span>
            <button @click="copyInviteLink" class="copy-btn" :title="$t('invite.copyLink')">
              📋
            </button>
          </div>
        </div>

        <!-- 复制成功提示 -->
        <div v-if="copySuccessMessage" class="copy-success-toast">
          ✓ {{ copySuccessMessage }}
        </div>

        <!-- 邀请记录 -->
        <div class="records-section">
          <label class="section-label">{{ $t('invite.records') }}</label>
          <div v-if="inviteData.records && inviteData.records.length > 0" class="records-list">
            <div
              v-for="record in inviteData.records"
              :key="record.id"
              class="record-item"
            >
              <div class="record-info">
                <span class="record-email">{{ record.invitee_email }}</span>
                <span class="record-time">{{ formatDate(record.created_at) }}</span>
              </div>
              <div class="record-reward">
                <span class="reward-icon">💎</span>
                <span class="reward-value">+{{ record.credits_rewarded }}</span>
              </div>
            </div>
          </div>
          <div v-else class="empty-records">
            <span class="empty-icon">📭</span>
            <p>{{ $t('invite.noRecords') }}</p>
            <p class="empty-hint">{{ $t('invite.noRecordsHint') }}</p>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.invite-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--color-overlay-bg);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 20px;
}

.invite-modal {
  background: var(--color-modal-bg);
  border-radius: 18px;
  padding: 28px 24px;
  width: 100%;
  max-width: 520px;
  max-height: 90vh;
  overflow-y: auto;
  position: relative;
  border: 1px solid var(--color-tint-white-08);
  box-shadow: 0 25px 80px var(--color-tint-black-50);
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

.invite-header {
  text-align: center;
  margin-bottom: 28px;
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

.invite-header h2 {
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

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--color-tint-white-05);
  border-radius: 14px;
  padding: 14px;
  display: flex;
  align-items: center;
  gap: 12px;
  border: 1px solid var(--color-tint-white-08);
}

.stat-icon {
  font-size: 28px;
}

.stat-info {
  display: flex;
  flex-direction: column;
}

.stat-value {
  color: var(--color-text-primary);
  font-size: 24px;
  font-weight: 700;
}

.stat-label {
  color: var(--color-text-muted);
  font-size: 12px;
}

/* 邀请码区域 */
.section-label {
  display: block;
  color: var(--color-text-muted);
  font-size: 12px;
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.invite-code-section,
.invite-link-section {
  margin-bottom: 20px;
}

.code-display,
.link-display {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(0, 202, 224, 0.08);
  border: 1px solid rgba(0, 202, 224, 0.3);
  border-radius: 10px;
  padding: 12px 14px;
}

.code-text {
  flex: 1;
  color: #00cae0;
  font-size: 24px;
  font-weight: 700;
  font-family: 'Consolas', monospace;
  letter-spacing: 2px;
}

.link-text {
  flex: 1;
  color: #00cae0;
  font-size: 12px;
  word-break: break-all;
}

.copy-btn {
  background: rgba(0, 202, 224, 0.15);
  border: none;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 18px;
  transition: all 0.3s ease;
}

.copy-btn:hover {
  background: rgba(0, 202, 224, 0.25);
  transform: translateY(-2px);
}

.copy-success-toast {
  background: rgba(34, 197, 94, 0.2);
  color: #4ade80;
  padding: 12px 16px;
  border-radius: 8px;
  text-align: center;
  margin-bottom: 20px;
  font-size: 14px;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-10px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 邀请记录 */
.records-section {
  margin-top: 24px;
}

.records-list {
  background: var(--color-tint-white-03);
  border-radius: 10px;
  overflow: hidden;
  max-height: 300px;
  overflow-y: auto;
}

.record-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid var(--color-tint-white-05);
}

.record-item:last-child {
  border-bottom: none;
}

.record-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.record-email {
  color: var(--color-text-primary);
  font-size: 14px;
}

.record-time {
  color: var(--color-text-muted);
  font-size: 12px;
}

.record-reward {
  display: flex;
  align-items: center;
  gap: 4px;
  background: rgba(34, 197, 94, 0.1);
  padding: 6px 10px;
  border-radius: 8px;
}

.reward-icon {
  font-size: 14px;
}

.reward-value {
  color: #4ade80;
  font-weight: 600;
  font-size: 14px;
}

.empty-records {
  text-align: center;
  padding: 40px 20px;
  color: var(--color-text-muted);
}

.empty-icon {
  font-size: 48px;
  display: block;
  margin-bottom: 16px;
}

.empty-records p {
  margin: 0 0 8px 0;
}

.empty-hint {
  font-size: 12px;
  color: var(--color-text-muted);
}

/* Loading & Error */
.loading-state,
.error-state {
  text-align: center;
  padding: 40px 20px;
  color: var(--color-text-muted);
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid rgba(0, 202, 224, 0.2);
  border-top-color: #00cae0;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 16px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-icon {
  font-size: 48px;
  display: block;
  margin-bottom: 16px;
}

.retry-btn {
  background: #00cae0;
  color: white;
  border: none;
  padding: 10px 24px;
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  margin-top: 16px;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 4px 16px rgba(0, 202, 224, 0.3);
}

.retry-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0, 202, 224, 0.4);
}
</style>
