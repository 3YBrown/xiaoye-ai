<script setup>
import { computed, ref, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NPopover, NDrawer, NDrawerContent, NModal, NIcon } from 'naive-ui'
import { LogoWechat } from '@vicons/ionicons5'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '../stores/user'
import { useThemeStore } from '../stores/theme'
import { useLocaleStore } from '../stores/locale'
import { useNotificationStore } from '../stores/notifications'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const themeStore = useThemeStore()
const localeStore = useLocaleStore()
const notificationStore = useNotificationStore()

const handleCheckin = () => {
  if (!userStore.requireAuth()) return
  router.push('/account')
}

const showThemePanel = ref(false)
const showLangPanel = ref(false)
const showNotificationDrawer = ref(false)
const showNotificationDetail = ref(false)
const activeNotification = ref(null)

const openThemePanel = () => {
  showThemePanel.value = true
}

const closeThemePanel = () => {
  showThemePanel.value = false
}

const openLangPanel = () => {
  showLangPanel.value = true
}

const closeLangPanel = () => {
  showLangPanel.value = false
}

const handleThemeSelect = (mode) => {
  themeStore.setThemeMode(mode)
  closeThemePanel()
}

const handleLangSelect = (lang) => {
  localeStore.setLocale(lang)
  closeLangPanel()
}

const notificationItems = computed(() => notificationStore.sortedNotifications)
const unreadCount = computed(() => notificationStore.unreadCount)
const hasUnread = computed(() => unreadCount.value > 0)

const notificationUnreadText = computed(() => {
  if (!unreadCount.value) {
    return t('notifications.allRead')
  }
  if (unreadCount.value === 1) {
    return t('notifications.unreadSingle')
  }
  return t('notifications.unreadMultiple', { count: unreadCount.value })
})

const openNotificationDrawer = () => {
  if (!userStore.requireAuth()) return
  notificationStore.loadNotifications(true)
  showNotificationDrawer.value = true
}

const openNotificationDetail = (item) => {
  activeNotification.value = item
  notificationStore.markAsRead(item.id)
  showNotificationDrawer.value = false
  showNotificationDetail.value = true
}

const closeNotificationDetail = () => {
  showNotificationDetail.value = false
}

const formatNotificationTime = (value) => {
  const date = new Date(typeof value === 'number' ? value : Number(value) || value)
  if (Number.isNaN(date.getTime())) return ''
  const locale = localeStore.locale === 'en' ? 'en-US' : 'zh-CN'
  return new Intl.DateTimeFormat(locale, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

const navItems = computed(() => [
  { name: 'inspiration', path: '/inspiration', icon: 'inspiration', label: t('nav.inspiration') },
  { name: 'generate', path: '/generate', icon: 'generate', label: t('nav.generate') },
  { name: 'assets', path: '/assets', icon: 'assets', label: t('nav.assets') },
  { name: 'tools', path: '/tools', icon: 'tools', label: t('nav.tools') }
])

const isNavActive = (itemName) => {
  const name = currentRoute.value
  if (itemName === 'tools') return name === 'tools' || name === 'image-to-svg' || name === 'reverse-prompt' || name === 'image-convert'
  return name === itemName
}

const currentRoute = computed(() => route.name)

const navigateTo = (path) => {
  router.push(path)
}

const navigateToAccount = () => {
  if (!userStore.requireAuth()) return
  router.push('/account')
}

const handleGuestAvatarClick = () => {
  userStore.openAuth()
}

const handleOpenPricing = () => {
  if (!userStore.requireAuth()) return
  userStore.openPricing()
}

const handleLogout = () => {
  userStore.logout()
  router.push('/')
}

const themeOptions = computed(() => [
  { mode: 'light', label: t('sidebar.themeLight'), icon: '\u2600\ufe0f' },
  { mode: 'dark', label: t('sidebar.themeDark'), icon: '\ud83c\udf19' },
  { mode: 'system', label: t('sidebar.themeSystem'), icon: '\ud83d\udda5' }
])

const langOptions = [
  { lang: 'zh', label: '\u4e2d\u6587', icon: '\ud83c\udde8\ud83c\uddf3' },
  { lang: 'en', label: 'English', icon: '\ud83c\uddfa\ud83c\uddf8' }
]

const themeIcon = computed(() => {
  if (themeStore.themeMode === 'system') return '\ud83d\udda5'
  return themeStore.isDark ? '\ud83c\udf19' : '\u2600\ufe0f'
})

const langIcon = computed(() => localeStore.locale === 'en' ? '\ud83c\uddfa\ud83c\uddf8' : '\ud83c\udde8\ud83c\uddf3')
watch(
  () => userStore.token,
  (token) => {
    if (token) {
      notificationStore.loadNotifications(true)
      return
    }
    notificationStore.reset()
  },
  { immediate: true }
)
</script>

<template>
  <aside class="app-sidebar">
    <!-- Logo -->
    <div class="sidebar-logo" @click="navigateTo('/inspiration')">
      <img src="/images/icon.png" alt="小野 AI" class="logo-icon" />
    </div>

    <!-- Nav icons -->
    <nav class="sidebar-nav">
      <button
        v-for="item in navItems"
        :key="item.name"
        :class="['nav-item', { active: isNavActive(item.name) }]"
        @click="navigateTo(item.path)"
        :aria-label="item.label"
      >
        <svg v-if="item.icon === 'inspiration'" class="nav-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 2L15.09 8.26L22 9.27L17 14.14L18.18 21.02L12 17.77L5.82 21.02L7 14.14L2 9.27L8.91 8.26L12 2z"/>
        </svg>
        <svg v-else-if="item.icon === 'generate'" class="nav-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
          <circle cx="8.5" cy="8.5" r="1.5"/>
          <polyline points="21 15 16 10 5 21"/>
        </svg>
        <svg v-else-if="item.icon === 'assets'" class="nav-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
        </svg>
        <svg v-else-if="item.icon === 'tools'" class="nav-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>
        </svg>
        <span class="nav-label">{{ item.label }}</span>
      </button>
    </nav>

    <!-- Bottom area -->
    <div class="sidebar-bottom">
      <div class="bottom-divider"></div>

      <!-- Daily Checkin (only show when available) -->
      <button
        v-if="userStore.isLoggedIn && userStore.dailyCheckinAvailable"
        class="bottom-item checkin-item available"
        :data-tip="$t('checkin.sidebarTip')"
        :aria-label="$t('checkin.sidebarTip')"
        @click="handleCheckin"
      >
        <span class="checkin-icon">📅</span>
        <span class="checkin-label">+{{ userStore.nextCheckinReward }}</span>
      </button>

      <!-- Diamonds -->
      <button class="bottom-item credits-item" @click="handleOpenPricing" :data-tip="$t('sidebar.getDiamonds')" :aria-label="$t('sidebar.getDiamonds')">
        <span class="credits-icon-lg">&#x1F48E;</span>
        <span class="credits-num">{{ userStore.userCredits }}</span>
      </button>

      <!-- Community -->
      <NPopover trigger="click" placement="right-end" :show-arrow="false">
        <template #trigger>
          <button class="bottom-item community-item" :data-tip="$t('sidebar.joinCommunity')" :aria-label="$t('sidebar.joinCommunity')">
            <NIcon class="community-icon">
              <LogoWechat />
            </NIcon>
          </button>
        </template>
        <div class="community-popover">
          <div class="community-qr">
            <img src="/images/qrcode.jpg" :alt="$t('sidebar.communityTitle')" class="qr-image" />
          </div>
          <div class="community-body">
            <div class="community-title">{{ $t('sidebar.communityTitle') }}</div>
            <div class="community-desc" style="white-space: pre-line;">{{ $t('sidebar.communityDesc') }}</div>
            <div class="community-hint">{{ $t('sidebar.communityHint') }}</div>
          </div>
        </div>
      </NPopover>

      <!-- Notifications -->
      <button class="bottom-item notification-item" @click="openNotificationDrawer" :data-tip="$t('sidebar.notifications')" :aria-label="$t('sidebar.notifications')">
        <svg class="notification-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round">
          <path d="M15 17h5l-1.4-1.4a2 2 0 0 1-.6-1.4V11a6 6 0 1 0-12 0v3.2c0 .5-.2 1-.6 1.4L4 17h5"/>
          <path d="M9 17a3 3 0 0 0 6 0"/>
        </svg>
        <span v-if="hasUnread" class="notification-dot"></span>
      </button>

      <!-- User avatar -->
      <NPopover
        v-if="userStore.isLoggedIn"
        trigger="click"
        placement="right-end"
        :show-arrow="false"
        @update:show="v => { if (!v) { showThemePanel = false; showLangPanel = false } }"
      >
        <template #trigger>
          <button class="bottom-item avatar-item" :data-tip="userStore.userNickname" :aria-label="userStore.userNickname">
            {{ userStore.userAvatar }}
          </button>
        </template>
        <div class="user-popover">
          <template v-if="!showThemePanel && !showLangPanel">
            <div class="user-info">
              <div class="user-name">{{ userStore.userNickname }}</div>
              <div class="user-email">{{ userStore.currentUser?.email }}</div>
            </div>
            <div class="popover-divider"></div>
            <button class="popover-item" @click="navigateToAccount">
              <span>&#x1F464;</span> {{ $t('sidebar.myAccount') }}
            </button>
            <div class="popover-divider"></div>
            <button class="popover-item" @click="openThemePanel">
              <span>&#x1F3A8;</span> {{ $t('sidebar.theme') }}
              <span class="submenu-current">{{ themeIcon }}</span>
              <svg class="submenu-arrow" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="9 18 15 12 9 6"/>
              </svg>
            </button>
            <button class="popover-item" @click="openLangPanel">
              <span>&#x1F310;</span> {{ $t('sidebar.language') }}
              <span class="submenu-current">{{ langIcon }}</span>
              <svg class="submenu-arrow" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="9 18 15 12 9 6"/>
              </svg>
            </button>
            <div class="popover-divider"></div>
            <button class="popover-item logout" @click="handleLogout">
              <span>&#x1F6AA;</span> {{ $t('sidebar.logout') }}
            </button>
          </template>
          <template v-else-if="showThemePanel">
            <button class="popover-item subpanel-back-item" @click="closeThemePanel">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="15 18 9 12 15 6"/>
              </svg>
              <span style="font-weight: 600;">{{ $t('sidebar.theme') }}</span>
            </button>
            <div class="popover-divider"></div>
            <button
              v-for="opt in themeOptions"
              :key="opt.mode"
              :class="['popover-item theme-option', { active: themeStore.themeMode === opt.mode }]"
              @click="handleThemeSelect(opt.mode)"
            >
              <span>{{ opt.icon }}</span>
              <span class="theme-label">{{ opt.label }}</span>
              <svg v-if="themeStore.themeMode === opt.mode" class="check-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
            </button>
          </template>
          <template v-else-if="showLangPanel">
            <button class="popover-item subpanel-back-item" @click="closeLangPanel">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="15 18 9 12 15 6"/>
              </svg>
              <span style="font-weight: 600;">{{ $t('sidebar.language') }}</span>
            </button>
            <div class="popover-divider"></div>
            <button
              v-for="opt in langOptions"
              :key="opt.lang"
              :class="['popover-item theme-option', { active: localeStore.locale === opt.lang }]"
              @click="handleLangSelect(opt.lang)"
            >
              <span>{{ opt.icon }}</span>
              <span class="theme-label">{{ opt.label }}</span>
              <svg v-if="localeStore.locale === opt.lang" class="check-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
            </button>
          </template>
        </div>
      </NPopover>
      <button
        v-else
        class="bottom-item avatar-item"
        :data-tip="$t('auth.login')"
        :aria-label="$t('auth.login')"
        @click="handleGuestAvatarClick"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round">
          <path d="M20 21a8 8 0 0 0-16 0"/>
          <circle cx="12" cy="7" r="4"/>
        </svg>
      </button>
    </div>
  </aside>

  <NDrawer v-model:show="showNotificationDrawer" placement="left" :width="320" :trap-focus="false" :auto-focus="false">
    <NDrawerContent :title="$t('notifications.panelTitle')" closable>
      <template #header-extra>
        <button class="notification-mark-read" @click="notificationStore.markAllAsRead">
          {{ $t('notifications.markAllRead') }}
        </button>
      </template>

      <div class="notification-status">{{ notificationUnreadText }}</div>
      <div v-if="notificationItems.length" class="notification-list">
        <button
          v-for="item in notificationItems"
          :key="item.id"
          :class="['notification-row', { unread: !item.read }]"
          @click="openNotificationDetail(item)"
        >
          <div class="notification-row-head">
            <div class="notification-row-title">{{ item.title }}</div>
            <div class="notification-row-time">{{ formatNotificationTime(item.createdAt) }}</div>
          </div>
          <div class="notification-row-summary">{{ item.summary }}</div>
        </button>
      </div>
      <div v-else class="notification-empty">{{ $t('notifications.empty') }}</div>
    </NDrawerContent>
  </NDrawer>

  <NModal
    v-model:show="showNotificationDetail"
    preset="card"
    :style="{ width: 'min(560px, calc(100vw - 32px))' }"
    :mask-closable="true"
  >
    <div v-if="activeNotification" class="notification-detail">
      <div class="notification-detail-title">
        {{ activeNotification.title || $t('notifications.detailTitle') }}
      </div>
      <div class="notification-detail-time">{{ formatNotificationTime(activeNotification.createdAt) }}</div>
      <div class="notification-detail-content">{{ activeNotification.content }}</div>
      <div class="notification-detail-footer">
        <button class="notification-ack-btn" @click="closeNotificationDetail">
          {{ $t('notifications.acknowledge') }}
        </button>
      </div>
    </div>
  </NModal>
</template>

<style scoped>
.app-sidebar {
  width: 64px;
  height: 100vh;
  background: var(--color-sidebar-bg);
  backdrop-filter: blur(16px);
  border-right: 1px solid var(--color-sidebar-border);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0 16px;
  flex-shrink: 0;
  z-index: 100;
  transition: background 0.3s, border-color 0.3s;
}

/* ====== Logo ====== */
.sidebar-logo {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  margin-bottom: 28px;
  border-radius: 10px;
  transition: background 0.2s;
}
.sidebar-logo:hover {
  background: var(--color-sidebar-hover);
}
.logo-icon {
  width: 24px;
  height: 24px;
  object-fit: contain;
}

/* ====== Navigation ====== */
.sidebar-nav {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  flex: 1;
  justify-content: center;
  margin-bottom: 120px;
}

.nav-item {
  width: 48px;
  height: 52px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: none;
  background: transparent;
  border-radius: 12px;
  color: var(--color-sidebar-text);
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
}
.nav-item:hover {
  background: var(--color-sidebar-hover);
  color: var(--color-sidebar-text-hover);
}
.nav-item.active {
  background: rgba(0, 202, 224, 0.08);
  color: #00cae0;
}

.nav-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.nav-label {
  font-size: 10px;
  line-height: 1;
  white-space: nowrap;
  letter-spacing: 0.02em;
}

/* ====== Bottom ====== */
.sidebar-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  margin-top: auto;
  padding-top: 12px;
}

.bottom-divider {
  width: 28px;
  height: 1px;
  background: var(--color-sidebar-divider);
  margin-bottom: 4px;
}

.bottom-item {
  width: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  border: none;
  background: transparent;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
  padding: 8px 0;
  position: relative;
}
.bottom-item:hover {
  background: var(--color-sidebar-hover);
}

/* Custom tooltip for sidebar buttons: faster than native title tooltip */
.nav-item[data-tip]::after,
.bottom-item[data-tip]::after {
  content: attr(data-tip);
  position: absolute;
  left: calc(100% + 10px);
  top: 50%;
  transform: translateY(-50%) translateX(-4px);
  opacity: 0;
  pointer-events: none;
  white-space: nowrap;
  font-size: 12px;
  line-height: 1;
  color: var(--color-text-primary);
  background: color-mix(in oklab, var(--color-sidebar-bg) 88%, #000 12%);
  border: 1px solid var(--color-sidebar-border);
  border-radius: 8px;
  padding: 7px 9px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.22);
  z-index: 80;
  transition: opacity 0.14s ease, transform 0.14s ease;
}

.nav-item[data-tip]::before,
.bottom-item[data-tip]::before {
  content: '';
  position: absolute;
  left: calc(100% + 5px);
  top: 50%;
  transform: translateY(-50%);
  opacity: 0;
  pointer-events: none;
  border: 5px solid transparent;
  border-right-color: color-mix(in oklab, var(--color-sidebar-bg) 88%, #000 12%);
  z-index: 81;
  transition: opacity 0.14s ease;
}

.nav-item[data-tip]:hover::after,
.bottom-item[data-tip]:hover::after {
  opacity: 1;
  transform: translateY(-50%) translateX(0);
}

.nav-item[data-tip]:hover::before,
.bottom-item[data-tip]:hover::before {
  opacity: 1;
}

.credits-item {
  color: #fbbf24;
}
.credits-icon-lg {
  font-size: 16px;
}
.credits-num {
  font-size: 10px;
  font-weight: 600;
}

/* ====== Checkin ====== */
.checkin-item {
  width: 48px;
  height: 44px;
  position: relative;
  cursor: pointer;
  animation: checkin-breathe 2s ease-in-out infinite;
}
.checkin-item:hover {
  transform: scale(1.08);
}
.checkin-icon {
  font-size: 15px;
  line-height: 1;
}
.checkin-label {
  font-size: 10px;
  font-weight: 700;
  color: #fbbf24;
  line-height: 1;
}
.checkin-label.done-label {
  color: var(--color-text-muted);
  font-weight: 600;
}

@keyframes checkin-breathe {
  0%, 100% { box-shadow: 0 0 0 0 rgba(251, 191, 36, 0); }
  50% { box-shadow: 0 0 8px 2px rgba(251, 191, 36, 0.25); }
}

/* ====== Community ====== */
.community-item {
  width: 36px;
  height: 36px;
  padding: 0;
}
.community-icon {
  font-size: 18px;
  color: #07c160;
}

.notification-item {
  width: 36px;
  height: 36px;
  padding: 0;
  position: relative;
  color: var(--color-sidebar-text);
}
.notification-item:hover {
  color: var(--color-sidebar-text-hover);
}
.notification-icon {
  width: 18px;
  height: 18px;
}
.notification-dot {
  position: absolute;
  top: 7px;
  right: 7px;
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #ef4444;
  border: 1.5px solid var(--color-sidebar-bg);
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.12);
}

.community-popover {
  padding: 16px;
  width: 220px;
  text-align: center;
}
.community-qr {
  display: flex;
  justify-content: center;
  margin-bottom: 12px;
}
.qr-image {
  width: 160px;
  height: 160px;
  border-radius: 10px;
  object-fit: cover;
  background: #fff;
}
.community-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: 6px;
}
.community-desc {
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-text-muted);
}
.community-hint {
  margin-top: 8px;
  font-size: 11px;
  color: #00cae0;
  font-weight: 500;
}

.avatar-item {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: rgba(0, 202, 224, 0.15);
  color: #00cae0;
  font-weight: 700;
  font-size: 13px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1.5px solid rgba(0, 202, 224, 0.25);
}
.avatar-item:hover {
  transform: scale(1.08);
  background: rgba(0, 202, 224, 0.22);
  border-color: rgba(0, 202, 224, 0.4);
}

/* ====== User popover ====== */
.user-popover {
  padding: 8px 0;
  min-width: 200px;
}
.user-info {
  padding: 12px 16px;
}
.user-name {
  font-weight: 600;
  font-size: 15px;
  color: var(--color-text-primary);
}
.user-email {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-top: 2px;
}
.popover-divider {
  height: 1px;
  background: var(--color-popover-divider);
  margin: 4px 0;
}
.popover-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 16px;
  background: none;
  border: none;
  color: var(--color-text-secondary);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.15s;
}
.popover-item:hover {
  background: var(--color-popover-hover);
  color: var(--color-text-primary);
}
.popover-item.logout {
  color: var(--color-error);
}
.popover-item.logout:hover {
  background: rgba(239, 68, 68, 0.1);
}

/* ====== Submenu arrow & current ====== */
.submenu-arrow {
  opacity: 0.5;
  flex-shrink: 0;
}
.submenu-current {
  margin-left: auto;
  font-size: 13px;
}

/* ====== Theme options (inline in popover) ====== */
.subpanel-back-item svg {
  opacity: 0.5;
}
.theme-option.active {
  color: #00cae0;
  font-weight: 500;
}
.theme-label {
  flex: 1;
}
.check-icon {
  flex-shrink: 0;
}

.notification-mark-read {
  border: none;
  background: transparent;
  color: #00cae0;
  font-size: 12px;
  padding: 0;
}
.notification-mark-read:hover {
  opacity: 0.8;
}

.notification-status {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 12px;
}

.notification-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.notification-row {
  border: none;
  border-bottom: 1px solid var(--color-divider);
  background: transparent;
  border-radius: 0;
  width: 100%;
  text-align: left;
  padding: 10px 6px;
  transition: all 0.2s;
}
.notification-row:hover {
  background: var(--color-tint-white-05);
}
.notification-row.unread {
  background: linear-gradient(90deg, rgba(0, 202, 224, 0.08), transparent 45%);
}
.notification-row:last-child {
  border-bottom: none;
}

.notification-row-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
}

.notification-row-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.notification-row-time {
  font-size: 11px;
  color: var(--color-text-muted);
  flex-shrink: 0;
  margin-top: 2px;
}

.notification-row-summary {
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-secondary);
}

.notification-empty {
  color: var(--color-text-muted);
  font-size: 13px;
  padding: 16px 0;
  text-align: center;
}

.notification-detail-time {
  color: var(--color-text-muted);
  font-size: 12px;
  margin-bottom: 10px;
}

.notification-detail-content {
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text-primary);
  white-space: pre-line;
}

.notification-detail {
  padding: 18px 20px 16px;
}

.notification-detail-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: 8px;
}

.notification-detail-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 14px;
}

.notification-ack-btn {
  border: none;
  background: #00cae0;
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  border-radius: 10px;
  padding: 8px 16px;
}

.notification-ack-btn:hover {
  opacity: 0.92;
}

/* ====== Mobile: bottom bar ====== */
@media (max-width: 768px) {
  .app-sidebar {
    width: 100%;
    height: 56px;
    flex-direction: row;
    padding: 0 8px;
    position: fixed;
    bottom: 0;
    left: 0;
    border-right: none;
    border-top: 1px solid var(--color-sidebar-border);
    z-index: 200;
  }
  .sidebar-logo { display: none; }
  .sidebar-nav {
    flex-direction: row;
    flex: 1;
    justify-content: center;
    gap: 4px;
    margin-bottom: 0;
  }
  .nav-item {
    width: 48px;
    height: 48px;
  }
  .bottom-divider { display: none; }
  .sidebar-bottom {
    flex-direction: row;
    gap: 2px;
    margin-top: 0;
    padding-top: 0;
    align-items: center;
  }
  .community-item { display: none; }
  .bottom-item { padding: 6px 0; }
  .checkin-item { width: auto; height: 32px; flex-direction: row; gap: 3px; padding: 4px 8px; }
  .checkin-item.available { animation: none; }
  .checkin-icon { font-size: 13px; }
  .credits-item { flex-direction: row; gap: 4px; width: auto; padding: 6px 8px; }
  .notification-item { width: 32px; height: 32px; }
  .notification-dot { top: 5px; right: 5px; }
  .avatar-item { width: 30px; height: 30px; font-size: 12px; }
  .nav-item[data-tip]::before,
  .bottom-item[data-tip]::before,
  .nav-item[data-tip]::after,
  .bottom-item[data-tip]::after { display: none; }
}
</style>

