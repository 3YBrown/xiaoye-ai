import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

const api = axios.create({ baseURL: '/api' })

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

export const useGenerationStore = defineStore('generation', () => {
  const generations = ref([])
  const loading = ref(false)
  const total = ref(0)
  const pendingResult = ref(null)
  const hasLoaded = ref(false)
  const filters = ref({
    type: 'all',
    favorite: false,
    shared: false,
    limit: 20,
    offset: 0
  })

  async function load(resetOffset = false, force = false) {
    if (hasLoaded.value && !resetOffset && !force) return

    if (resetOffset) filters.value.offset = 0

    loading.value = true
    try {
      const params = {
        type: filters.value.type,
        favorite: filters.value.favorite,
        shared: filters.value.shared,
        limit: filters.value.limit,
        offset: filters.value.offset
      }
      const { data } = await api.get('/generations', { params })

      // 过滤掉生成失败的数据
      const validGenerations = (data.generations || []).filter(g => g.status !== 'failed')
      if (resetOffset) {
        generations.value = validGenerations
      } else {
        generations.value.push(...validGenerations)
      }

      total.value = data.total || 0
      hasLoaded.value = true
    } catch (e) {
      console.error('Failed to load generations', e)
      generations.value = []
      total.value = 0
    } finally {
      loading.value = false
    }
  }

  async function setFilter(key, value) {
    filters.value[key] = value
    await load(true, true)
  }

  async function setFilters(updates) {
    Object.assign(filters.value, updates)
    await load(true, true)
  }

  async function loadMore() {
    if (loading.value) return
    if (generations.value.length >= total.value) return
    if (!hasLoaded.value) return

    filters.value.offset = generations.value.length
    await load(false, true)
  }

  function prependGeneration(gen) {
    generations.value.unshift(gen)
    total.value++
  }

  function updateGenerationLocal(id, updates) {
    const gen = generations.value.find(g => g.id === id)
    if (gen) Object.assign(gen, updates)
  }

  async function getGeneration(id) {
    try {
      const { data } = await api.get(`/generations/${id}`)
      return data
    } catch (e) {
      console.error('Failed to get generation', e)
      return null
    }
  }

  async function updateGeneration(id, updates) {
    try {
      const { data } = await api.put(`/generations/${id}`, updates)
      const gen = generations.value.find(g => g.id === id)
      if (gen) Object.assign(gen, data)
      return data
    } catch (e) {
      console.error('Failed to update generation', e)
      return null
    }
  }

  async function toggleFavorite(id) {
    const gen = generations.value.find(g => g.id === id)
    if (!gen) return

    const next = !gen.is_favorite
    await updateGeneration(id, { is_favorite: next })
  }

  async function deleteGeneration(id) {
    try {
      await api.delete(`/generations/${id}`)
      generations.value = generations.value.filter(g => g.id !== id)
      total.value = Math.max(0, total.value - 1)
    } catch (e) {
      console.error('Failed to delete generation', e)
    }
  }

  const groupedGenerations = computed(() => {
    const now = new Date()
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
    const yesterday = today - 86400000
    const weekAgo = today - 7 * 86400000

    const groups = { today: [], yesterday: [], week: [], older: [] }
    const sorted = [...generations.value].sort((a, b) => b.created_at - a.created_at)

    for (const gen of sorted) {
      const ts = gen.created_at
      if (ts >= today) groups.today.push(gen)
      else if (ts >= yesterday) groups.yesterday.push(gen)
      else if (ts >= weekAgo) groups.week.push(gen)
      else groups.older.push(gen)
    }

    return groups
  })

  function reset() {
    generations.value = []
    total.value = 0
    hasLoaded.value = false
    filters.value = {
      type: 'all',
      favorite: false,
      shared: false,
      limit: 20,
      offset: 0
    }
  }

  return {
    generations,
    loading,
    total,
    hasLoaded,
    filters,
    pendingResult,
    groupedGenerations,
    load,
    setFilter,
    setFilters,
    loadMore,
    getGeneration,
    updateGeneration,
    prependGeneration,
    updateGenerationLocal,
    toggleFavorite,
    deleteGeneration,
    reset
  }
})
