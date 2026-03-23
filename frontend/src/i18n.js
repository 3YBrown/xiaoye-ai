import { createI18n } from 'vue-i18n'
import zh from './locales/zh.json'
import en from './locales/en.json'

const savedLocale = localStorage.getItem('locale')
const browserLang = navigator.language?.startsWith('en') ? 'en' : 'zh'

const i18n = createI18n({
  legacy: false,
  locale: savedLocale || browserLang,
  fallbackLocale: 'zh',
  messages: { zh, en }
})

export default i18n
