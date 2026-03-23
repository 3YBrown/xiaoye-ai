import axios from 'axios'
import { useUserStore } from '../stores/user'

export function useInspiration() {
  const userStore = useUserStore()

  const authHeaders = () => {
    if (!userStore.token) return {}
    return { Authorization: `Bearer ${userStore.token}` }
  }

  async function listInspirations(params = {}, requestConfig = {}) {
    const { data } = await axios.get('/api/inspirations', {
      params,
      headers: authHeaders(),
      ...requestConfig
    })
    return data
  }

  async function listLikedInspirations(params = {}) {
    const { data } = await axios.get('/api/inspirations/liked', {
      params,
      headers: authHeaders()
    })
    return data
  }

  async function listMyInspirations(params = {}) {
    const { data } = await axios.get('/api/inspirations/mine', {
      params,
      headers: authHeaders()
    })
    return data
  }

  async function getInspiration(shareId) {
    const { data } = await axios.get(`/api/inspirations/${shareId}`, {
      headers: authHeaders()
    })
    return data
  }

  async function getInspirationLikeStatus(shareId) {
    const { data } = await axios.get(`/api/inspirations/${shareId}/liked`, {
      headers: authHeaders()
    })
    return data
  }

  async function likeInspiration(shareId) {
    const { data } = await axios.post(`/api/inspirations/${shareId}/like`, {}, {
      headers: authHeaders()
    })
    return data
  }

  async function unlikeInspiration(shareId) {
    const { data } = await axios.delete(`/api/inspirations/${shareId}/like`, {
      headers: authHeaders()
    })
    return data
  }

  async function markRemix(shareId) {
    if (!userStore.token) return
    await axios.post(`/api/inspirations/${shareId}/remix`, {}, { headers: authHeaders() })
  }

  async function shareGeneration(generationId, { title, description, prompt, tags, cover_url } = {}) {
    const { data } = await axios.post(
      `/api/generations/${generationId}/share`,
      { title, description, prompt, tags, cover_url },
      { headers: authHeaders() }
    )
    return data
  }

  async function publishInspiration(payload = {}) {
    const { data } = await axios.post('/api/inspirations/publish', payload, { headers: authHeaders() })
    return data
  }

  async function listInspirationTags(params = {}, requestConfig = {}) {
    const { data } = await axios.get('/api/inspiration-tags', { params, headers: authHeaders(), ...requestConfig })
    return data
  }

  async function uploadVideo(file) {
    const form = new FormData()
    form.append('file', file)
    const { data } = await axios.post('/api/user/upload/video', form, {
      headers: authHeaders()
    })
    return data?.url || ''
  }

  async function unshareInspiration(shareId) {
    await axios.delete(`/api/inspirations/${shareId}`, { headers: authHeaders() })
  }

  return {
    listInspirations,
    listLikedInspirations,
    listMyInspirations,
    getInspiration,
    getInspirationLikeStatus,
    likeInspiration,
    unlikeInspiration,
    markRemix,
    shareGeneration,
    publishInspiration,
    listInspirationTags,
    uploadVideo,
    unshareInspiration
  }
}
