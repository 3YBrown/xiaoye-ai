import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

export const usePricingStore = defineStore('pricing', () => {
  const loaded = ref(false)

  // Raw data from backend
  const imagePricingRaw = ref([])   // [{model, model_name, prices: [{size, credits, description}]}]
  const videoPricingRaw = ref({ base_per_second: {}, audio_multiplier: 1.2 })
  const ecommerceRaw = ref({ model: '', model_name: '', prices: {} })

  // Reactive computed data matching old config/pricing.js shape
  const imagePricing = computed(() => {
    const result = {}
    for (const entry of imagePricingRaw.value) {
      result[entry.model] = {}
      for (const p of entry.prices) {
        result[entry.model][p.size] = { credits: p.credits, description: p.description }
      }
    }
    return result
  })

  const videoPricing = computed(() => ({
    basePerSecond: videoPricingRaw.value.base_per_second,
    audioMultiplier: videoPricingRaw.value.audio_multiplier,
    veoBasePerSecond: videoPricingRaw.value.veo_base_per_second || { '720p': 45, '1080p': 65, '4k': 90 },
  }))

  const ecommercePricing = computed(() => ecommerceRaw.value)
  const ecommerceModel = computed(() => ecommerceRaw.value.model || 'doubao-seedream-4-5')

  // --- Methods matching old config/pricing.js API ---

  function getImageCredits(model, size) {
    const modelPricing = imagePricing.value[model]
    if (!modelPricing) return 10
    const pricing = modelPricing[size]
    if (!pricing) {
      const firstSize = Object.keys(modelPricing)[0]
      return modelPricing[firstSize]?.credits || 10
    }
    return pricing.credits
  }

  function getVideoCredits(resolution, duration, generateAudio = false) {
    const base = videoPricing.value.basePerSecond[resolution] || 10
    let total = base * duration
    if (generateAudio) {
      total = Math.ceil(total * videoPricing.value.audioMultiplier)
    }
    return total
  }

  function getVeoCredits(resolution, duration) {
    const base = videoPricing.value.veoBasePerSecond[resolution] || 45
    return base * duration
  }

  function getEcommerceCredits(size, count) {
    return getImageCredits(ecommerceModel.value, size) * count
  }

  function getPriceBadge(credits) {
    return `${credits}💎`
  }

  function getImageSizeOptions(model) {
    const pricing = imagePricing.value[model]
    if (!pricing) return []
    return Object.entries(pricing).map(([size, config]) => ({
      label: size,
      badge: getPriceBadge(config.credits),
      value: size,
      description: config.description,
    }))
  }

  function getEcommerceSizeOptions() {
    return getImageSizeOptions(ecommerceModel.value)
  }

  async function fetchPricing() {
    try {
      const { data } = await axios.get('/api/pricing')
      imagePricingRaw.value = data.image || []
      videoPricingRaw.value = data.video || { base_per_second: {}, audio_multiplier: 1.2 }
      ecommerceRaw.value = data.ecommerce || { model: 'doubao-seedream-4-5', prices: {} }
      loaded.value = true
    } catch (e) {
      console.error('[PricingStore] Failed to fetch pricing:', e.message)
    }
  }

  return {
    loaded,
    imagePricing,
    videoPricing,
    ecommercePricing,
    ecommerceModel,
    getImageCredits,
    getVideoCredits,
    getVeoCredits,
    getEcommerceCredits,
    getPriceBadge,
    getImageSizeOptions,
    getEcommerceSizeOptions,
    fetchPricing,
  }
})
