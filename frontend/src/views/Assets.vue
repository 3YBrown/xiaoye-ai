<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NEmpty, NPopover, useMessage } from 'naive-ui'
import { useGenerationStore } from '../stores/generation'
import { useUserStore } from '../stores/user'
import { useInspiration } from '../composables/useInspiration'
import ShareGenerationDialog from '../components/ShareGenerationDialog.vue'

const { t } = useI18n()
const router = useRouter()
const genStore = useGenerationStore()
const userStore = useUserStore()
const { shareGeneration, unshareInspiration, listLikedInspirations, listMyInspirations, unlikeInspiration, publishInspiration } = useInspiration()
const message = useMessage()

const typeFilter = ref('all')
const subFilter = ref('all')
const likedPosts = ref([])
const likedLoading = ref(false)
const likedLoadingMore = ref(false)
const likedTotal = ref(0)
const likedOffset = ref(0)
const likedLimit = 20
const sharedPosts = ref([])
const sharedLoading = ref(false)
const sharedLoadingMore = ref(false)
const sharedTotal = ref(0)
const sharedOffset = ref(0)
const sharedLimit = 20

const canLoadMoreLiked = computed(() => likedPosts.value.length < likedTotal.value)
const canLoadMoreShared = computed(() => sharedPosts.value.length < sharedTotal.value)
const isCurrentLoading = computed(() => {
  if (subFilter.value === 'liked') return likedLoading.value || likedLoadingMore.value
  if (subFilter.value === 'shared') return sharedLoading.value || sharedLoadingMore.value
  return genStore.loading
})
const showEmpty = computed(() => !timelineGroups.value.length && !isCurrentLoading.value)

onMounted(() => {
  typeFilter.value = genStore.filters.type || 'all'
  if (genStore.filters.shared) subFilter.value = 'shared'
  else if (genStore.filters.favorite) subFilter.value = 'favorite'
  else subFilter.value = 'all'
  if (subFilter.value === 'liked') {
    loadLiked(true)
  } else if (subFilter.value === 'shared') {
    loadShared(true)
  } else {
    genStore.load(true, true)
  }
})

const setTypeFilter = async (type) => {
  typeFilter.value = type
  genStore.filters.type = type
  if (subFilter.value === 'liked') {
    await loadLiked(true)
    return
  }
  if (subFilter.value === 'shared') {
    await loadShared(true)
    return
  }
  await genStore.setFilter('type', type)
}

const setSubFilter = async (sub) => {
  if (sub === 'liked' && !userStore.requireAuth()) return
  subFilter.value = sub
  if (sub === 'favorite') {
    await genStore.setFilters({ type: typeFilter.value, favorite: true, shared: false })
  } else if (sub === 'shared') {
    await loadShared(true)
  } else if (sub === 'liked') {
    await loadLiked(true)
  } else {
    await genStore.setFilters({ type: typeFilter.value, favorite: false, shared: false })
  }
}

const buildTimelineGroups = (items, timeField) => {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const yesterday = today - 86400000
  const weekAgo = today - 7 * 86400000
  const groups = { today: [], yesterday: [], week: [], older: [] }

  const sorted = [...items].sort((a, b) => (b[timeField] || 0) - (a[timeField] || 0))
  for (const item of sorted) {
    const ts = Number(item[timeField] || 0)
    if (ts >= today) groups.today.push(item)
    else if (ts >= yesterday) groups.yesterday.push(item)
    else if (ts >= weekAgo) groups.week.push(item)
    else groups.older.push(item)
  }

  const result = []
  if (groups.today.length) result.push({ label: t('assets.today'), items: groups.today })
  if (groups.yesterday.length) result.push({ label: t('assets.yesterday'), items: groups.yesterday })
  if (groups.week.length) result.push({ label: t('assets.week'), items: groups.week })
  if (groups.older.length) result.push({ label: t('assets.older'), items: groups.older })
  return result
}

const generationTimelineGroups = computed(() => {
  const g = genStore.groupedGenerations
  const result = []
  if (g.today.length) result.push({ label: t('assets.today'), items: g.today })
  if (g.yesterday.length) result.push({ label: t('assets.yesterday'), items: g.yesterday })
  if (g.week.length) result.push({ label: t('assets.week'), items: g.week })
  if (g.older.length) result.push({ label: t('assets.older'), items: g.older })
  return result
})

const likedTimelineGroups = computed(() => buildTimelineGroups(likedPosts.value, 'published_at'))
const sharedTimelineGroups = computed(() => buildTimelineGroups(sharedPosts.value, 'published_at'))

const timelineGroups = computed(() => {
  if (subFilter.value === 'liked') return likedTimelineGroups.value
  if (subFilter.value === 'shared') return sharedTimelineGroups.value
  return generationTimelineGroups.value
})

const loadLiked = async (reset = false) => {
  if (!userStore.isLoggedIn) return
  if (reset) {
    likedOffset.value = 0
    likedLoading.value = true
  } else {
    if (!canLoadMoreLiked.value || likedLoadingMore.value) return
    likedLoadingMore.value = true
  }
  try {
    const data = await listLikedInspirations({
      type: typeFilter.value,
      limit: likedLimit,
      offset: likedOffset.value
    })
    const items = (data.items || []).map(item => ({ ...item, is_shared: true }))
    likedTotal.value = data.total || 0
    if (reset) likedPosts.value = items
    else likedPosts.value.push(...items)
    likedOffset.value = likedPosts.value.length
  } catch (e) {
    if (reset) likedPosts.value = []
    message.error(e.response?.data?.error || t('inspiration.loadLikesFailed'))
  } finally {
    likedLoading.value = false
    likedLoadingMore.value = false
  }
}

const loadShared = async (reset = false) => {
  if (!userStore.isLoggedIn) return
  if (reset) {
    sharedOffset.value = 0
    sharedLoading.value = true
  } else {
    if (!canLoadMoreShared.value || sharedLoadingMore.value) return
    sharedLoadingMore.value = true
  }
  try {
    const data = await listMyInspirations({
      type: typeFilter.value,
      limit: sharedLimit,
      offset: sharedOffset.value
    })
    const items = data.items || []
    sharedTotal.value = data.total || 0
    if (reset) sharedPosts.value = items
    else sharedPosts.value.push(...items)
    sharedOffset.value = sharedPosts.value.length
  } catch (e) {
    if (reset) sharedPosts.value = []
    message.error(e.response?.data?.error || t('inspiration.loadLikesFailed'))
  } finally {
    sharedLoading.value = false
    sharedLoadingMore.value = false
  }
}

const toggleFavorite = async (id) => {
  await genStore.toggleFavorite(id)
}

const deleteAsset = async (id) => {
  if (!confirm(t('assets.confirmDelete'))) return
  await genStore.deleteGeneration(id)
}

const openImageToSvg = (imageUrl) => {
  if (!imageUrl) return
  router.push({ name: 'image-to-svg', query: { src: imageUrl } })
}

const downloadImage = (url, i) => {
  if (!url) return
  const a = document.createElement('a')
  a.href = url
  a.download = `nanobanana-${Date.now()}-${i}.png`
  a.click()
}

const viewInGenerate = (gen) => {
  genStore.pendingResult = { ...gen, fromAssets: true }
  router.push('/generate')
}

const openInspirationDetail = (post) => {
  if (!post?.share_id) return
  router.push(`/inspiration/${post.share_id}`)
}

const removeLikedPostLocal = (shareId) => {
  likedPosts.value = likedPosts.value.filter(item => item.share_id !== shareId)
  likedTotal.value = Math.max(0, likedTotal.value - 1)
  likedOffset.value = likedPosts.value.length
}

const generationIDToShare = ref(null)
const generationToShare = ref(null)
const showShareDialog = ref(false)
const shareLoading = ref(false)
const showPublishDialog = ref(false)
const publishLoading = ref(false)

const openShareDialog = (gen) => {
  if (!gen?.id) return
  if (!userStore.requireAuth()) return
  generationIDToShare.value = gen.id
  generationToShare.value = gen
  showShareDialog.value = true
}

const openPublishDialog = () => {
  if (!userStore.requireAuth()) return
  showPublishDialog.value = true
}

const shareDialogInitialData = computed(() => {
  const gen = generationToShare.value
  if (!gen) return {}
  const params = gen.params || {}
  return {
    prompt: gen.prompt || '',
    images: Array.isArray(gen.images) ? gen.images : [],
    video_url: gen.video_url || '',
    cover_url: params.cover_url || params.coverUrl || params.videoCoverUrl || '',
    type: gen.video_url ? 'video' : 'image'
  }
})

const handleShareConfirm = async ({ title, description, prompt, tags, cover_url }) => {
  if (!generationIDToShare.value) return
  shareLoading.value = true
  try {
    const post = await shareGeneration(generationIDToShare.value, { title, description, prompt, tags, cover_url })
    const gen = genStore.generations.find(g => g.id === generationIDToShare.value)
    if (gen) {
      gen.is_shared = true
      gen.share_id = post?.share_id || ''
    }
    showShareDialog.value = false
    const reviewStatus = (post?.review_status || '').toLowerCase()
    if (reviewStatus === 'approved' || reviewStatus === '') {
      message.success(t('inspiration.shareSuccessPublished'))
      message.info(t('inspiration.shareHintUnshare'))
    } else {
      message.success(t('inspiration.shareSubmittedPending'))
    }
  } catch (e) {
    const errorMsg = e.response?.data?.error
    message.error(errorMsg || t('inspiration.shareFailed'))
  } finally {
    shareLoading.value = false
    generationIDToShare.value = null
    generationToShare.value = null
  }
}

const handlePublishConfirm = async (payload) => {
  publishLoading.value = true
  try {
    const post = await publishInspiration({
      source_type: 'upload',
      title: payload.title,
      description: payload.description,
      prompt: payload.prompt,
      tags: payload.tags || [],
      images: payload.images || [],
      video_url: payload.video_url || '',
      cover_url: payload.cover_url || '',
      type: payload.type || 'image'
    })
    showPublishDialog.value = false
    const reviewStatus = (post?.review_status || '').toLowerCase()
    if (reviewStatus === 'approved' || reviewStatus === '') {
      message.success(t('inspiration.shareSuccessPublished'))
    } else {
      message.success(t('inspiration.shareSubmittedPending'))
    }
    if (subFilter.value !== 'shared') {
      subFilter.value = 'shared'
    }
    await loadShared(true)
  } catch (e) {
    message.error(e.response?.data?.error || t('inspiration.shareFailed'))
  } finally {
    publishLoading.value = false
  }
}

const removeSharedPostLocal = (shareId) => {
  sharedPosts.value = sharedPosts.value.filter(item => item.share_id !== shareId)
  sharedTotal.value = Math.max(0, sharedTotal.value - 1)
  sharedOffset.value = sharedPosts.value.length
}

const normalizeReviewStatus = (status) => {
  const value = String(status || '').toLowerCase()
  if (value === 'rejected') return 'rejected'
  if (value === 'pending') return 'pending'
  return 'approved'
}

const reviewStatusText = (status) => {
  const value = normalizeReviewStatus(status)
  if (value === 'pending') return t('inspiration.reviewPending')
  if (value === 'rejected') return t('inspiration.reviewRejected')
  return t('inspiration.reviewApproved')
}

const reviewStatusClass = (status) => `review-${normalizeReviewStatus(status)}`

const toggleLikePost = async (post) => {
  if (!post?.share_id) return
  try {
    await unlikeInspiration(post.share_id)
    removeLikedPostLocal(post.share_id)
    message.success(t('inspiration.unlikeSuccess'))
  } catch (e) {
    message.error(e.response?.data?.error || t('inspiration.unlikeFailed'))
  }
}

const toggleShareInspiration = async (gen) => {
  if (!gen?.id && !gen?.share_id) return
  if (!userStore.requireAuth()) return
  const isShared = subFilter.value === 'shared' || !!gen.is_shared
  if (isShared) {
    try {
      if (!gen.share_id) throw new Error('missing share id')
      const shareId = gen.share_id
      await unshareInspiration(shareId)
      gen.is_shared = false
      gen.share_id = ''
      message.success(t('inspiration.unshareSuccess'))
      if (subFilter.value === 'shared') {
        removeSharedPostLocal(shareId)
      }
    } catch (e) {
      message.error(e.response?.data?.error || t('inspiration.unshareFailed'))
    }
  } else {
    openShareDialog(gen)
  }
}

const handleScroll = (e) => {
  const el = e.target
  if (subFilter.value === 'liked') {
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 200) {
      loadLiked(false)
    }
    return
  }
  if (subFilter.value === 'shared') {
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 200) {
      loadShared(false)
    }
    return
  }
  if (el.scrollHeight - el.scrollTop - el.clientHeight < 200 && !genStore.loading) {
    genStore.loadMore()
  }
}
</script>

<template>
  <div class="assets-page">
    <!-- Filter bar -->
    <div class="filter-bar">
      <div class="filter-row">
        <button :class="['filter-tab', { active: typeFilter === 'all' }]" @click="setTypeFilter('all')">{{ $t('assets.all') }}</button>
        <button :class="['filter-tab', { active: typeFilter === 'image' }]" @click="setTypeFilter('image')">{{ $t('assets.images') }}</button>
        <button :class="['filter-tab', { active: typeFilter === 'video' }]" @click="setTypeFilter('video')">{{ $t('assets.videos') }}</button>
      </div>
      <div class="filter-row sub-row">
        <button :class="['filter-chip', { active: subFilter === 'all' }]" @click="setSubFilter('all')">{{ $t('assets.allItems') }}</button>
        <button :class="['filter-chip', { active: subFilter === 'favorite' }]" @click="setSubFilter('favorite')">{{ $t('assets.favorites') }}</button>
        <button :class="['filter-chip', { active: subFilter === 'shared' }]" @click="setSubFilter('shared')">{{ $t('assets.myShares') }}</button>
        <button :class="['filter-chip', { active: subFilter === 'liked' }]" @click="setSubFilter('liked')">{{ $t('assets.myLikes') }}</button>
        <button v-if="subFilter === 'shared'" class="publish-btn" @click="openPublishDialog">{{ t('inspiration.publishAction') }}</button>
      </div>
    </div>
    <!-- Assets grid -->
    <div class="assets-scroll" @scroll="handleScroll">
      <div v-for="group in timelineGroups" :key="group.label" class="timeline-group">
        <div class="section-label">{{ group.label }}</div>
        <div class="assets-grid">
          <div v-for="item in group.items" :key="(subFilter === 'liked' || subFilter === 'shared') ? item.share_id : item.id" class="asset-card">
            <div v-if="subFilter === 'liked'" class="asset-preview" @click="openInspirationDetail(item)">
              <img :src="item.cover_url || item.images?.[0] || item.video_url" class="asset-thumb" loading="lazy" />
            </div>
            <div v-else-if="subFilter === 'shared'" class="asset-preview" @click="openInspirationDetail(item)">
              <video v-if="item.video_url" :src="item.video_url" class="asset-thumb" preload="metadata" muted />
              <img v-else :src="item.cover_url || item.images?.[0]" class="asset-thumb" loading="lazy" />
            </div>
            <!-- Image asset -->
            <div v-else-if="item.images?.length" class="asset-preview" @click="viewInGenerate(item)">
              <img :src="item.images[0]" class="asset-thumb" loading="lazy" />
              <div v-if="item.images.length > 1" class="asset-count">+{{ item.images.length - 1 }}</div>
            </div>
            <!-- Video asset -->
            <div v-else-if="item.video_url" class="asset-preview" @click="viewInGenerate(item)">
              <video :src="item.video_url" class="asset-thumb" preload="metadata" muted />
              <div class="asset-play">▶</div>
            </div>

            <div class="asset-info">
              <div v-if="subFilter === 'shared'" class="review-badge" :class="reviewStatusClass(item.review_status)">
                {{ reviewStatusText(item.review_status) }}
              </div>
              <p class="asset-prompt">{{ item.prompt }}</p>
              <div v-if="subFilter === 'liked'" class="asset-actions">
                <button class="action-btn favorited" @click="toggleLikePost(item)" :title="t('inspiration.unlikeAction')">
                  <svg class="action-icon" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                    <path d="M12 21s-6.7-4.3-9.2-8.2c-1.6-2.5-1-5.9 1.8-7.4 2.1-1.2 4.5-.6 6.1 1.2 1.6-1.8 4-2.4 6.1-1.2 2.8 1.5 3.4 4.9 1.8 7.4C18.7 16.7 12 21 12 21z" />
                  </svg>
                </button>
                <button class="action-btn" @click="openInspirationDetail(item)" :title="t('inspiration.viewDetail')">
                  <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M14 5h5v5" />
                    <path d="M10 14L19 5" />
                    <path d="M19 14v4a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h4" />
                  </svg>
                </button>
                <button class="action-btn" @click="downloadImage(item.cover_url || item.images?.[0] || item.video_url, 0)" :title="t('generate.download')">
                  <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M12 4v10" />
                    <path d="M8 10l4 4 4-4" />
                    <path d="M5 19h14" />
                  </svg>
                </button>
                <NPopover v-if="!item.video_url" trigger="click" placement="top" :show-arrow="false">
                  <template #trigger>
                    <button class="action-btn" :title="t('generate.toolbox')">
                      <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                        <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
                      </svg>
                    </button>
                  </template>
                  <div class="toolbox-menu">
                    <button class="toolbox-menu-item" @click="openImageToSvg(item.cover_url || item.images?.[0])">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
                      {{ t('generate.toSvg') }}
                    </button>
                  </div>
                </NPopover>
              </div>
              <div v-else-if="subFilter === 'shared'" class="asset-actions">
                <button class="action-btn shared" @click="toggleShareInspiration(item)" :title="t('common.unshare')">
                  <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M5 12l4 4L19 6" />
                  </svg>
                </button>
                <button class="action-btn" @click="openInspirationDetail(item)" :title="t('inspiration.viewDetail')">
                  <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M14 5h5v5" />
                    <path d="M10 14L19 5" />
                    <path d="M19 14v4a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h4" />
                  </svg>
                </button>
                <button class="action-btn" @click="downloadImage(item.cover_url || item.images?.[0] || item.video_url, 0)" :title="t('generate.download')">
                  <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M12 4v10" />
                    <path d="M8 10l4 4 4-4" />
                    <path d="M5 19h14" />
                  </svg>
                </button>
                <NPopover v-if="!item.video_url" trigger="click" placement="top" :show-arrow="false">
                  <template #trigger>
                    <button class="action-btn" :title="t('generate.toolbox')">
                      <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                        <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
                      </svg>
                    </button>
                  </template>
                  <div class="toolbox-menu">
                    <button class="toolbox-menu-item" @click="openImageToSvg(item.cover_url || item.images?.[0])">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
                      {{ t('generate.toSvg') }}
                    </button>
                  </div>
                </NPopover>
              </div>
              <div v-else class="asset-actions">
                <button class="action-btn" :class="{ favorited: item.is_favorite }" @click="toggleFavorite(item.id)" :title="t('assets.favorites')">
                  <svg class="action-icon" viewBox="0 0 24 24" :fill="item.is_favorite ? 'currentColor' : 'none'" aria-hidden="true">
                    <path d="M12 21s-6.7-4.3-9.2-8.2c-1.6-2.5-1-5.9 1.8-7.4 2.1-1.2 4.5-.6 6.1 1.2 1.6-1.8 4-2.4 6.1-1.2 2.8 1.5 3.4 4.9 1.8 7.4C18.7 16.7 12 21 12 21z" />
                  </svg>
                </button>
                <button class="action-btn" @click="downloadImage(item.images?.[0] || item.video_url, 0)" :title="t('generate.download')">
                  <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M12 4v10" />
                    <path d="M8 10l4 4 4-4" />
                    <path d="M5 19h14" />
                  </svg>
                </button>
                <button
                  class="action-btn"
                  :class="{ shared: item.is_shared }"
                  @click="toggleShareInspiration(item)"
                  :title="item.is_shared ? t('common.unshare') : t('common.share')"
                >
                  <svg v-if="item.is_shared" class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M5 12l4 4L19 6" />
                  </svg>
                  <svg v-else class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M12 5v10" />
                    <path d="M8 9l4-4 4 4" />
                    <path d="M5 14v4a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-4" />
                  </svg>
                </button>
                <button class="action-btn delete-btn" @click="deleteAsset(item.id)" :title="t('common.delete')">
                  <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M4 7h16" />
                    <path d="M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
                    <path d="M8 7v12a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V7" />
                    <path d="M10 11v6" />
                    <path d="M14 11v6" />
                  </svg>
                </button>
                <NPopover v-if="item.images?.length" trigger="click" placement="top" :show-arrow="false">
                  <template #trigger>
                    <button class="action-btn" :title="t('generate.toolbox')">
                      <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                        <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
                      </svg>
                    </button>
                  </template>
                  <div class="toolbox-menu">
                    <button class="toolbox-menu-item" @click="openImageToSvg(item.images[0])">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
                      {{ t('generate.toSvg') }}
                    </button>
                  </div>
                </NPopover>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="showEmpty && subFilter === 'shared'" class="empty-publish-wrap">
        <NEmpty :description="$t('assets.noAssets')" />
        <button class="empty-publish-btn" @click="openPublishDialog">{{ t('inspiration.publishAction') }}</button>
      </div>
      <NEmpty v-else-if="showEmpty" :description="$t('assets.noAssets')" style="margin-top: 80px;" />

      <div
        v-if="isCurrentLoading"
        class="loading-more"
      >
        {{ $t('assets.loading') }}
      </div>
    </div>

    <ShareGenerationDialog
      v-model:show="showShareDialog"
      :loading="shareLoading"
      mode="generation"
      :initial-data="shareDialogInitialData"
      @confirm="handleShareConfirm"
    />

    <ShareGenerationDialog
      v-model:show="showPublishDialog"
      :loading="publishLoading"
      mode="upload"
      @confirm="handlePublishConfirm"
    />
  </div>
</template>

<style scoped>
.assets-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.filter-bar {
  flex-shrink: 0;
  padding: 16px 24px 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.filter-row {
  display: flex;
  gap: 4px;
}

.filter-tab {
  padding: 8px 20px;
  background: transparent;
  border: 1px solid var(--color-tint-white-08);
  border-radius: 10px;
  color: var(--color-text-secondary);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all .2s;
  font-family: inherit;
}
.filter-tab:hover {
  background: var(--color-tint-white-04);
  border-color: var(--color-tint-white-15);
}
.filter-tab.active {
  background: rgba(0, 202, 224, 0.1);
  border-color: rgba(0, 202, 224, 0.3);
  color: #00cae0;
  font-weight: 600;
}

.filter-chip {
  padding: 5px 14px;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: var(--color-text-muted);
  font-size: 13px;
  cursor: pointer;
  transition: all .2s;
  font-family: inherit;
}
.filter-chip:hover {
  color: var(--color-text-secondary);
  background: var(--color-tint-white-04);
}
.filter-chip.active {
  color: #00cae0;
  background: rgba(0, 202, 224, 0.08);
}

.sub-row {
  align-items: center;
}

.publish-btn {
  margin-left: auto;
  height: 32px;
  padding: 0 12px;
  border-radius: 9px;
  border: 1px solid rgba(0, 202, 224, 0.35);
  background: rgba(0, 202, 224, 0.16);
  color: #d8fbff;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all .2s;
}

.publish-btn:hover {
  background: rgba(0, 202, 224, 0.28);
  box-shadow: 0 2px 10px rgba(0, 202, 224, 0.2);
}

.assets-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px 24px;
}

.timeline-group {
  margin-bottom: 24px;
}

.section-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-muted);
  margin-bottom: 12px;
  padding-left: 4px;
}

.assets-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.asset-card {
  background: var(--color-tint-white-02);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 14px;
  overflow: hidden;
  transition: all .25s;
}
.asset-card:hover {
  border-color: var(--color-tint-white-12);
  transform: translateY(-2px);
  box-shadow: 0 8px 24px var(--color-tint-black-30);
}

.asset-preview {
  position: relative;
  aspect-ratio: 1;
  overflow: hidden;
  cursor: pointer;
}

.asset-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform .3s;
}
.asset-preview:hover .asset-thumb {
  transform: scale(1.05);
}

.asset-count {
  position: absolute;
  top: 8px;
  right: 8px;
  background: var(--color-actions-overlay);
  color: white;
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 8px;
  backdrop-filter: blur(4px);
}

.asset-play {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 40px;
  height: 40px;
  background: var(--color-tint-black-50);
  backdrop-filter: blur(4px);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 14px;
}

.asset-info {
  padding: 10px 12px;
}

.review-badge {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  margin-bottom: 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  border: 1px solid transparent;
}

.review-badge.review-approved {
  color: #8cefff;
  border-color: rgba(0, 202, 224, 0.35);
  background: rgba(0, 202, 224, 0.12);
}

.review-badge.review-pending {
  color: #ffd28f;
  border-color: rgba(255, 184, 92, 0.4);
  background: rgba(255, 184, 92, 0.14);
}

.review-badge.review-rejected {
  color: #ff9d9d;
  border-color: rgba(239, 68, 68, 0.45);
  background: rgba(239, 68, 68, 0.16);
}

.asset-prompt {
  font-size: 12px;
  line-height: 1.45;
  color: var(--color-text-secondary);
  margin: 0 0 8px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.asset-actions {
  display: flex;
  gap: 6px;
  align-items: center;
}

.action-btn {
  width: 28px;
  height: 28px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(0, 0, 0, 0.4);
  color: #fff;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: all .2s;
}
.action-icon {
  width: 15px;
  height: 15px;
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.action-btn:hover {
  border-color: rgba(0, 202, 224, 0.45);
  background: rgba(0, 202, 224, 0.2);
}
.action-btn.favorited {
  color: #ff879f;
  border-color: rgba(255, 135, 159, 0.45);
  background: rgba(255, 125, 152, 0.12);
}
.action-btn.shared {
  color: #8cefff;
  border-color: rgba(0, 202, 224, 0.45);
  background: rgba(0, 202, 224, 0.18);
}
.action-btn.delete-btn:hover {
  border-color: rgba(239, 68, 68, 0.45);
  background: rgba(239, 68, 68, 0.2);
}

.loading-more {
  text-align: center;
  padding: 20px;
  color: var(--color-text-muted);
  font-size: 13px;
}

.empty-publish-wrap {
  margin-top: 80px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.empty-publish-btn {
  height: 34px;
  padding: 0 14px;
  border-radius: 10px;
  border: 1px solid rgba(0, 202, 224, 0.35);
  background: rgba(0, 202, 224, 0.16);
  color: #d8fbff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all .2s;
}

.empty-publish-btn:hover {
  background: rgba(0, 202, 224, 0.28);
}

.toolbox-menu {
  padding: 4px 0;
  min-width: 120px;
}
.toolbox-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  background: none;
  border: none;
  color: var(--color-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
  font-family: inherit;
}
.toolbox-menu-item:hover {
  background: var(--color-popover-hover, rgba(0, 202, 224, 0.08));
  color: var(--color-text-primary);
}

@media (max-width: 768px) {
  .filter-bar { padding: 12px 14px 0; }
  .assets-scroll { padding: 12px 14px 16px; }
  .assets-grid {
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 8px;
  }
  .asset-info { padding: 8px 10px; }
  .asset-prompt { font-size: 11px; margin-bottom: 6px; }
  .action-btn { width: 28px; height: 28px; font-size: 13px; }
}

@media (max-width: 380px) {
  .assets-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 6px;
  }
}
</style>
