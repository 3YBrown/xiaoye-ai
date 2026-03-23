import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import i18n from '../i18n'

export const useLocaleStore = defineStore('locale', () => {
  const locale = ref(i18n.global.locale.value)

  function setLocale(lang) {
    locale.value = lang
    i18n.global.locale.value = lang
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new Event('locale-changed'))
    }
  }

  watch(locale, (val) => {
    localStorage.setItem('locale', val)
  })

  return { locale, setLocale }
})
