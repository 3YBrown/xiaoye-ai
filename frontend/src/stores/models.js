import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

export const useModelsStore = defineStore('models', () => {
  // State
  const models = ref([])
  const loading = ref(false)
  const loaded = ref(false)
  const defaultModelId = ref('')

  // Getters
  const imageModels = computed(() => {
    return models.value.filter(m => m.id.includes('image') || m.id.includes('seedream'))
  })

  const availableModels = computed(() => {
    return models.value.filter(m => m.available !== false)
  })

  const getModelById = computed(() => (id) => {
    return models.value.find(m => m.id === id) || null
  })

  // 获取模型显示名称（找不到时返回 null）
  const getDisplayName = computed(() => (id) => {
    const model = models.value.find(m => m.id === id)
    return model ? model.name : null
  })

  const ensureModelId = computed(() => (id) => {
    const validIds = models.value.map(m => m.id)
    if (validIds.includes(id)) return id
    if (defaultModelId.value) return defaultModelId.value
    if (validIds.length > 0) return validIds[0]
    return id
  })

  // Actions
  async function loadModels(force = false) {
    if (loaded.value && !force) return
    
    loading.value = true
    try {
      const { data } = await axios.get('/api/models')
      models.value = data.models || []
      
      // 设置默认模型（优先 Nanobanana Pro，否则第一个可用模型）
      const priorityOrder = ['gemini-3.1-flash-image-preview', 'gemini-3-pro-image-preview', 'doubao-seedream-4-5']
      for (const id of priorityOrder) {
        const found = models.value.find(m => m.id === id && m.available !== false)
        if (found) {
          defaultModelId.value = id
          break
        }
      }
      if (!defaultModelId.value && models.value.length > 0) {
        defaultModelId.value = models.value[0].id
      }
      
      loaded.value = true
    } catch (e) {
      console.error('Failed to load models:', e)
      models.value = []
    } finally {
      loading.value = false
    }
  }

  function reset() {
    models.value = []
    loading.value = false
    loaded.value = false
    defaultModelId.value = ''
  }

  return {
    models,
    loading,
    loaded,
    defaultModelId,
    imageModels,
    availableModels,
    getModelById,
    ensureModelId,
    getDisplayName,
    loadModels,
    reset
  }
})
