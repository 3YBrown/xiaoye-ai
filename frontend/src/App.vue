<script setup>
import { onMounted, computed } from 'vue'
import { NConfigProvider, NMessageProvider, NDialogProvider, darkTheme, zhCN, dateZhCN, enUS, dateEnUS } from 'naive-ui'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from './stores/user'
import { useThemeStore } from './stores/theme'
import { useLocaleStore } from './stores/locale'
import AuthModal from './components/AuthModal.vue'
import InviteModal from './components/InviteModal.vue'
import RedeemModal from './components/RedeemModal.vue'
import PricingModal from './components/PricingModal.vue'
import BindEmailModal from './components/BindEmailModal.vue'

const userStore = useUserStore()
const themeStore = useThemeStore()
const localeStore = useLocaleStore()
const router = useRouter()
const route = useRoute()

const handleLoginSuccess = (user, token) => {
  userStore.loginSuccess(user, token)
  // 登录后如果在落地页，跳转到创作页
  if (route.name === 'landing') {
    const redirect = route.query.redirect
    router.push(redirect || '/inspiration')
  }
}

const handlePricingToRedeem = () => {
  userStore.closePricing()
  userStore.openRedeem()
}

const handleRedeemToPricing = () => {
  userStore.closeRedeem()
  userStore.openPricing()
}

// 在 setup 阶段立即初始化用户状态（早于子组件 onMounted），
// 避免子组件在 onMounted 中读到空 token。
userStore.init()

const naiveTheme = computed(() => themeStore.isDark ? darkTheme : null)
const naiveLocale = computed(() => localeStore.locale === 'en' ? enUS : zhCN)
const naiveDateLocale = computed(() => localeStore.locale === 'en' ? dateEnUS : dateZhCN)

// Naive UI 主题覆盖 - 根据当前主题返回不同配置
const themeOverrides = computed(() => {
  if (themeStore.isDark) {
    return {
      common: {
        primaryColor: '#667eea',
        primaryColorHover: '#a5b4fc',
        primaryColorPressed: '#5a67d8',
        primaryColorSuppl: '#667eea',
        successColor: '#10b981',
        successColorHover: '#34d399',
        errorColor: '#ef4444',
        errorColorHover: '#f87171',
        warningColor: '#f59e0b',
        warningColorHover: '#fbbf24',
        bodyColor: '#0f0f1a',
        cardColor: 'rgba(16, 16, 28, 0.96)',
        modalColor: 'rgba(16, 16, 28, 0.96)',
        popoverColor: '#1a1a2e',
        inputColor: 'rgba(255, 255, 255, 0.04)',
        inputColorDisabled: 'rgba(255, 255, 255, 0.02)',
        borderColor: 'rgba(255, 255, 255, 0.07)',
        hoverColor: 'rgba(255, 255, 255, 0.06)',
        borderRadius: '12px',
        borderRadiusSmall: '8px',
        fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
        fontSize: '14px',
        textColorBase: '#fff',
        textColor1: 'rgba(255, 255, 255, 0.9)',
        textColor2: 'rgba(255, 255, 255, 0.6)',
        textColor3: 'rgba(255, 255, 255, 0.4)',
        placeholderColor: 'rgba(255, 255, 255, 0.2)',
        closeIconColor: 'rgba(255, 255, 255, 0.35)',
        closeIconColorHover: 'rgba(255, 255, 255, 0.8)',
        closeColorHover: 'rgba(255, 255, 255, 0.1)',
        closeColorPressed: 'rgba(255, 255, 255, 0.15)'
      },
      Button: {
        borderRadiusMedium: '10px',
        borderRadiusSmall: '8px',
        fontWeight: '600',
        fontSizeMedium: '13px',
        heightMedium: '38px',
        paddingMedium: '0 16px',
        colorPrimary: '#667eea',
        colorHoverPrimary: '#7c8ff0',
        colorPressedPrimary: '#5a67d8',
        textColorPrimary: '#fff',
        textColorGhostPrimary: '#a5b4fc',
        borderPrimary: 'none',
        borderHoverPrimary: 'none',
        borderPressedPrimary: 'none'
      },
      Input: {
        borderRadius: '10px',
        heightMedium: '38px',
        fontSizeMedium: '13px',
        color: 'rgba(255,255,255,0.04)',
        colorFocus: 'rgba(255,255,255,0.06)',
        border: '1px solid rgba(255,255,255,0.07)',
        borderHover: '1px solid rgba(255,255,255,0.15)',
        borderFocus: '1px solid rgba(102,126,234,0.45)',
        boxShadowFocus: '0 0 0 4px rgba(102,126,234,0.1)',
        placeholderColor: 'rgba(255,255,255,0.2)',
        colorDisabled: 'rgba(255,255,255,0.02)',
        textColorDisabled: 'rgba(255,255,255,0.25)',
        caretColor: '#a5b4fc'
      },
      Card: {
        borderRadius: '18px',
        color: 'rgba(16, 16, 28, 0.96)',
        borderColor: 'rgba(255, 255, 255, 0.06)',
        paddingMedium: '0',
        paddingSmall: '0',
        titleFontSizeMedium: '18px',
        titleFontWeight: '700',
        titleTextColor: '#e0e7ff',
        closeIconColor: 'rgba(255, 255, 255, 0.35)',
        closeIconColorHover: 'rgba(255, 255, 255, 0.8)',
        closeColorHover: 'rgba(255, 255, 255, 0.1)'
      },
      Modal: {
        color: 'rgba(16, 16, 28, 0.96)',
        borderRadius: '18px',
        boxShadow: '0 40px 120px rgba(0,0,0,0.7), 0 0 80px rgba(102,126,234,0.06)'
      },
      Tabs: {
        tabBorderRadius: '8px',
        tabFontSizeMedium: '13px',
        tabFontWeightActive: '600',
        tabGapMediumSegment: '3px',
        tabPaddingMediumSegment: '8px 14px',
        colorSegment: 'rgba(255,255,255,0.03)',
        tabColorSegment: 'transparent',
        tabTextColorSegment: 'rgba(255,255,255,0.4)',
        tabTextColorActiveSegment: '#fff',
        tabTextColorHoverSegment: 'rgba(255,255,255,0.65)'
      },
      Alert: {
        borderRadius: '10px',
        fontSize: '13px',
        iconSizeMedium: '18px',
        padding: '10px 12px'
      },
      Spin: {
        color: '#a5b4fc'
      },
      Tag: {
        borderRadius: '6px'
      },
      Empty: {
        textColor: 'rgba(255,255,255,0.3)',
        iconColor: 'rgba(255,255,255,0.15)',
        iconSizeMedium: '40px'
      }
    }
  } else {
    // Light theme overrides
    return {
      common: {
        primaryColor: '#5a6fd6',
        primaryColorHover: '#8b9cf0',
        primaryColorPressed: '#4a5bc0',
        primaryColorSuppl: '#5a6fd6',
        successColor: '#059669',
        successColorHover: '#10b981',
        errorColor: '#dc2626',
        errorColorHover: '#ef4444',
        warningColor: '#d97706',
        warningColorHover: '#f59e0b',
        bodyColor: '#f8f9fc',
        cardColor: '#ffffff',
        modalColor: '#ffffff',
        popoverColor: '#ffffff',
        inputColor: 'rgba(0, 0, 0, 0.03)',
        inputColorDisabled: 'rgba(0, 0, 0, 0.02)',
        borderColor: 'rgba(0, 0, 0, 0.08)',
        hoverColor: 'rgba(0, 0, 0, 0.04)',
        borderRadius: '12px',
        borderRadiusSmall: '8px',
        fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
        fontSize: '14px',
        textColorBase: '#1a1b2e',
        textColor1: 'rgba(0, 0, 0, 0.85)',
        textColor2: 'rgba(0, 0, 0, 0.55)',
        textColor3: 'rgba(0, 0, 0, 0.35)',
        placeholderColor: 'rgba(0, 0, 0, 0.25)',
        closeIconColor: 'rgba(0, 0, 0, 0.35)',
        closeIconColorHover: 'rgba(0, 0, 0, 0.7)',
        closeColorHover: 'rgba(0, 0, 0, 0.08)',
        closeColorPressed: 'rgba(0, 0, 0, 0.12)'
      },
      Button: {
        borderRadiusMedium: '10px',
        borderRadiusSmall: '8px',
        fontWeight: '600',
        fontSizeMedium: '13px',
        heightMedium: '38px',
        paddingMedium: '0 16px',
        colorPrimary: '#5a6fd6',
        colorHoverPrimary: '#6b80e0',
        colorPressedPrimary: '#4a5bc0',
        textColorPrimary: '#fff',
        textColorGhostPrimary: '#5a6fd6',
        borderPrimary: 'none',
        borderHoverPrimary: 'none',
        borderPressedPrimary: 'none'
      },
      Input: {
        borderRadius: '10px',
        heightMedium: '38px',
        fontSizeMedium: '13px',
        color: 'rgba(0,0,0,0.03)',
        colorFocus: 'rgba(0,0,0,0.04)',
        border: '1px solid rgba(0,0,0,0.08)',
        borderHover: '1px solid rgba(0,0,0,0.15)',
        borderFocus: '1px solid rgba(90,111,214,0.45)',
        boxShadowFocus: '0 0 0 4px rgba(90,111,214,0.1)',
        placeholderColor: 'rgba(0,0,0,0.25)',
        colorDisabled: 'rgba(0,0,0,0.02)',
        textColorDisabled: 'rgba(0,0,0,0.25)',
        caretColor: '#5a6fd6'
      },
      Card: {
        borderRadius: '18px',
        color: '#ffffff',
        borderColor: 'rgba(0, 0, 0, 0.06)',
        paddingMedium: '0',
        paddingSmall: '0',
        titleFontSizeMedium: '18px',
        titleFontWeight: '700',
        titleTextColor: '#1a1b2e',
        closeIconColor: 'rgba(0, 0, 0, 0.35)',
        closeIconColorHover: 'rgba(0, 0, 0, 0.7)',
        closeColorHover: 'rgba(0, 0, 0, 0.08)'
      },
      Modal: {
        color: '#ffffff',
        borderRadius: '18px',
        boxShadow: '0 40px 120px rgba(0,0,0,0.15), 0 0 80px rgba(90,111,214,0.04)'
      },
      Tabs: {
        tabBorderRadius: '8px',
        tabFontSizeMedium: '13px',
        tabFontWeightActive: '600',
        tabGapMediumSegment: '3px',
        tabPaddingMediumSegment: '8px 14px',
        colorSegment: 'rgba(0,0,0,0.04)',
        tabColorSegment: 'transparent',
        tabTextColorSegment: 'rgba(0,0,0,0.4)',
        tabTextColorActiveSegment: '#1a1b2e',
        tabTextColorHoverSegment: 'rgba(0,0,0,0.6)'
      },
      Alert: {
        borderRadius: '10px',
        fontSize: '13px',
        iconSizeMedium: '18px',
        padding: '10px 12px'
      },
      Spin: {
        color: '#5a6fd6'
      },
      Tag: {
        borderRadius: '6px'
      },
      Empty: {
        textColor: 'rgba(0,0,0,0.3)',
        iconColor: 'rgba(0,0,0,0.15)',
        iconSizeMedium: '40px'
      }
    }
  }
})

onMounted(() => {
  // 检查 URL 邀请码
  const urlParams = new URLSearchParams(window.location.search)
  const inviteCode = urlParams.get('invite')
  if (inviteCode) {
    localStorage.setItem('pendingInviteCode', inviteCode)
    window.history.replaceState({}, document.title, window.location.pathname)
    if (!userStore.isLoggedIn) {
      userStore.openAuth()
    }
  }
})
</script>

<template>
  <NConfigProvider :theme="naiveTheme" :theme-overrides="themeOverrides" :locale="naiveLocale" :date-locale="naiveDateLocale">
    <NMessageProvider>
      <NDialogProvider>
        <div class="app-root">
          <!-- 全局粒子背景 -->
          <div class="particle-field">
            <div class="particle" v-for="i in 30" :key="i" :style="{
              '--x': Math.random() * 100 + '%',
              '--y': Math.random() * 100 + '%',
              '--size': (Math.random() * 3 + 1) + 'px',
              '--duration': (Math.random() * 20 + 10) + 's',
              '--delay': (Math.random() * 10) + 's',
              '--opacity': Math.random() * 0.5 + 0.1
            }"></div>
          </div>
          <router-view />

          <!-- 全局弹窗 -->
          <AuthModal
            v-if="userStore.showAuthModal"
            @login-success="handleLoginSuccess"
            @close="userStore.closeAuth()"
          />
          <RedeemModal
            v-if="userStore.showRedeemModal && userStore.isLoggedIn"
            @close="userStore.closeRedeem()"
            @success="userStore.fetchUserInfo()"
            @open-pricing="handleRedeemToPricing"
          />
          <PricingModal
            v-if="userStore.showPricingModal"
            @close="userStore.closePricing()"
            @open-redeem="handlePricingToRedeem"
          />
          <InviteModal
            v-if="userStore.showInviteModal"
            @close="userStore.closeInvite()"
            @credits-updated="userStore.fetchUserInfo()"
          />
          <BindEmailModal
            v-if="userStore.showBindEmailModal"
          />
        </div>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style scoped>
.app-root {
  width: 100%;
  min-height: 100vh;
  position: relative;
}

/* 全局粒子背景 */
.particle-field {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
}

.particle {
  position: absolute;
  left: var(--x);
  top: var(--y);
  width: var(--size);
  height: var(--size);
  background: rgba(var(--particle-color), var(--opacity));
  border-radius: 50%;
  animation: particleFloat var(--duration) ease-in-out var(--delay) infinite;
}

@keyframes particleFloat {
  0%, 100% { transform: translate(0, 0) scale(1); opacity: var(--opacity); }
  25% { transform: translate(30px, -40px) scale(1.2); }
  50% { transform: translate(-20px, -80px) scale(0.8); opacity: calc(var(--opacity) * 1.5); }
  75% { transform: translate(40px, -40px) scale(1.1); }
}
</style>
