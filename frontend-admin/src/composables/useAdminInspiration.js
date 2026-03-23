import axios from 'axios'

const ADMIN_TOKEN_KEY = 'admin_token'

function getStoredAdminToken() {
  return localStorage.getItem(ADMIN_TOKEN_KEY) || ''
}

function saveAdminToken(token) {
  const value = (token || '').trim()
  if (value) {
    localStorage.setItem(ADMIN_TOKEN_KEY, value)
    return
  }
  localStorage.removeItem(ADMIN_TOKEN_KEY)
}

function adminHeaders() {
  const token = getStoredAdminToken()
  if (!token) return {}
  return { 'X-Admin-Token': token }
}

export function useAdminInspiration() {
  async function listAdminInspirations(params = {}) {
    const { data } = await axios.get('/api/admin/inspirations', {
      params,
      headers: adminHeaders()
    })
    return data
  }

  async function reviewInspiration(postID, payload = {}) {
    const { data } = await axios.post(`/api/admin/inspirations/${postID}/review`, payload, {
      headers: adminHeaders()
    })
    return data
  }

  return {
    getStoredAdminToken,
    saveAdminToken,
    listAdminInspirations,
    reviewInspiration
  }
}
