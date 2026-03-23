import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

export const useUserStore = defineStore('user', () => {
  // 状态
  const isLoggedIn = ref(false)
  const currentUser = ref(null)
  const token = ref(null)
  const showAuthModal = ref(false)
  const showRedeemModal = ref(false)
  const showInviteModal = ref(false)
  const showPricingModal = ref(false)
  const showBindEmailModal = ref(false)

  // 计算属性
  const userCredits = computed(() => currentUser.value?.credits || 0)
  const userNickname = computed(() =>
    currentUser.value?.nickname || currentUser.value?.email?.split('@')[0] || '用户'
  )
  const userAvatar = computed(() =>
    userNickname.value.charAt(0).toUpperCase()
  )
  const inviteCode = computed(() => currentUser.value?.invite_code || '')
  const dailyCheckinAvailable = computed(() => currentUser.value?.daily_checkin_available ?? false)
  const checkinStreak = computed(() => currentUser.value?.checkin_streak || 0)
  const nextCheckinReward = computed(() => currentUser.value?.next_checkin_reward || 5)
  const isLinuxDoUser = computed(() => currentUser.value?.is_linuxdo === true)

  // 初始化 - 从 localStorage 恢复登录状态
  function init() {
    const savedToken = localStorage.getItem('token')
    const savedUser = localStorage.getItem('user')

    if (savedToken && savedUser) {
      try {
        token.value = savedToken
        currentUser.value = JSON.parse(savedUser)
        isLoggedIn.value = true
        fetchUserInfo()
      } catch (e) {
        console.error('解析用户信息失败:', e)
        logout()
      }
    }
  }

  // 获取最新用户信息
  async function fetchUserInfo() {
    if (!token.value) return
    try {
      const response = await axios.get('/api/user/me', {
        headers: { Authorization: `Bearer ${token.value}` }
      })
      if (response.status === 200) {
        currentUser.value = response.data
        localStorage.setItem('user', JSON.stringify(response.data))
        if (response.data.email_verified === false) {
          showBindEmailModal.value = true
        }
      }
    } catch (e) {
      if (e.response?.status === 401) {
        logout()
      }
      console.error('获取用户信息失败:', e)
    }
  }

  // 登录成功
  function loginSuccess(user, newToken) {
    token.value = newToken
    currentUser.value = user
    isLoggedIn.value = true
    localStorage.setItem('token', newToken)
    localStorage.setItem('user', JSON.stringify(user))
    showAuthModal.value = false
    showRedeemModal.value = false
    fetchUserInfo()
  }

  // 退出登录
  function logout() {
    token.value = null
    currentUser.value = null
    isLoggedIn.value = false
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  // 要求登录（返回是否已登录）
  function requireAuth() {
    if (!isLoggedIn.value) {
      showAuthModal.value = true
      return false
    }
    return true
  }

  // 打开弹窗
  function openAuth() { showAuthModal.value = true }
  function closeAuth() { showAuthModal.value = false }
  function openRedeem() { showRedeemModal.value = true }
  function closeRedeem() { showRedeemModal.value = false }
  function openInvite() { showInviteModal.value = true }
  function closeInvite() { showInviteModal.value = false }
  function openPricing() { showPricingModal.value = true }
  function closePricing() { showPricingModal.value = false }
  function closeBindEmail() { showBindEmailModal.value = false }

  // 每日签到
  async function dailyCheckin() {
    if (!token.value) return
    const response = await axios.post('/api/user/daily-checkin', {}, {
      headers: { Authorization: `Bearer ${token.value}` }
    })
    await fetchUserInfo()
    return response.data
  }

  // 绑定邮箱
  async function bindEmail(email, code) {
    if (!token.value) return
    const response = await axios.post('/api/user/bind-email', { email, code }, {
      headers: { Authorization: `Bearer ${token.value}` }
    })
    // Update local user info
    await fetchUserInfo()
    return response.data
  }

  // 创建在线支付订单
  async function createPaymentOrder(plan) {
    if (!token.value) return null
    const response = await axios.post('/api/user/payment/create', { plan }, {
      headers: { Authorization: `Bearer ${token.value}` }
    })
    return response.data
  }

  // 查询支付订单状态
  async function getPaymentStatus(orderNo) {
    if (!token.value) return null
    const response = await axios.get(`/api/user/payment/status/${orderNo}`, {
      headers: { Authorization: `Bearer ${token.value}` }
    })
    return response.data
  }

  return {
    // 状态
    isLoggedIn,
    currentUser,
    token,
    showAuthModal,
    showRedeemModal,
    showInviteModal,
    showPricingModal,
    showBindEmailModal,
    // 计算属性
    userCredits,
    userNickname,
    userAvatar,
    inviteCode,
    dailyCheckinAvailable,
    checkinStreak,
    nextCheckinReward,
    isLinuxDoUser,
    // 方法
    init,
    fetchUserInfo,
    loginSuccess,
    logout,
    requireAuth,
    openAuth,
    closeAuth,
    openRedeem,
    closeRedeem,
    openInvite,
    closeInvite,
    openPricing,
    closePricing,
    closeBindEmail,
    bindEmail,
    dailyCheckin,
    createPaymentOrder,
    getPaymentStatus
  }
})
