import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const themeMode = ref(localStorage.getItem('themeMode') || 'dark')

  const systemDark = ref(
    window.matchMedia('(prefers-color-scheme: dark)').matches
  )

  // Listen for system theme changes
  const mql = window.matchMedia('(prefers-color-scheme: dark)')
  mql.addEventListener('change', (e) => {
    systemDark.value = e.matches
  })

  const resolvedTheme = computed(() => {
    if (themeMode.value === 'system') {
      return systemDark.value ? 'dark' : 'light'
    }
    return themeMode.value
  })

  const isDark = computed(() => (themeOverride.value || resolvedTheme.value) === 'dark')

  // 强制覆盖（落地页等场景用）
  const themeOverride = ref(null)

  const activeTheme = computed(() => themeOverride.value || resolvedTheme.value)

  function setThemeMode(mode) {
    themeMode.value = mode
  }

  function forceTheme(theme) {
    themeOverride.value = theme
  }

  function clearForceTheme() {
    themeOverride.value = null
  }

  // Sync to localStorage and <html> attribute
  watch(
    activeTheme,
    (theme) => {
      document.documentElement.setAttribute('data-theme', theme)
    },
    { immediate: true }
  )

  watch(themeMode, (mode) => {
    localStorage.setItem('themeMode', mode)
  })

  return {
    themeMode,
    resolvedTheme,
    activeTheme,
    isDark,
    setThemeMode,
    forceTheme,
    clearForceTheme
  }
})
