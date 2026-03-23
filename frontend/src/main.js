import { createApp } from 'vue'
import { createPinia } from 'pinia'
import naive from 'naive-ui'
import router from './router'
import i18n from './i18n'
import './style.css'
import App from './App.vue'
import { usePricingStore } from './stores/pricing'

const pinia = createPinia()
const app = createApp(App)

app.use(pinia)
app.use(router)
app.use(i18n)
app.use(naive)

// Fetch pricing data before mounting
usePricingStore().fetchPricing().then(() => {
  app.mount('#app')
})
