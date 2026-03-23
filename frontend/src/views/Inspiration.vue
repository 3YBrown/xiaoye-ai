<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useGenerationStore } from '../stores/generation'
import { useUserStore } from '../stores/user'
import ComposerBar from '../components/ComposerBar.vue'
import ShareGenerationDialog from '../components/ShareGenerationDialog.vue'
import { useInspiration } from '../composables/useInspiration'
import { useComposerDraftStore } from '../stores/composerDraft'

const inspirationPageCache = {
  initialized: false,
  posts: [],
  feedOffset: 0,
  feedTotal: 0,
  feedType: 'all',
  selectedTag: '',
  availableTags: [],
  searchKeyword: '',
  searchQuery: '',
  scrollTop: 0
}

const { t } = useI18n()
const router = useRouter()
const genStore = useGenerationStore()
const userStore = useUserStore()
const message = useMessage()
const composerDraftStore = useComposerDraftStore()
const { listInspirations, listInspirationTags, markRemix, likeInspiration, unlikeInspiration, publishInspiration } = useInspiration()

const composerRef = ref(null)
const pageRef = ref(null)
const creativeMode = ref('image')
const loading = ref(false)
const showBackTop = ref(false)

const posts = ref([])
const feedLoading = ref(false)
const feedLoadingMore = ref(false)
const feedLimit = 20
const feedOffset = ref(0)
const feedTotal = ref(0)
const feedType = ref('all')
const selectedTag = ref('')
const availableTags = ref([])
const searchKeyword = ref('')
const searchQuery = ref('')
const searchInputRef = ref(null)
const showPublishDialog = ref(false)
const publishLoading = ref(false)
let scrollRafId = null
let feedAbortController = null
let tagsAbortController = null

const isCanceledError = (error) => {
  return error?.code === 'ERR_CANCELED' || error?.name === 'CanceledError'
}

const cancelFeedRequest = () => {
  if (!feedAbortController) return
  feedAbortController.abort()
  feedAbortController = null
}

const cancelTagRequest = () => {
  if (!tagsAbortController) return
  tagsAbortController.abort()
  tagsAbortController = null
}

const snapshotPosts = () => posts.value.map(item => ({
  ...item,
  tags: Array.isArray(item?.tags) ? [...item.tags] : [],
  images: Array.isArray(item?.images) ? [...item.images] : [],
  author: item?.author ? { ...item.author } : item?.author
}))

const restoreFromCache = () => {
  posts.value = inspirationPageCache.posts.map(item => ({
    ...item,
    tags: Array.isArray(item?.tags) ? [...item.tags] : [],
    images: Array.isArray(item?.images) ? [...item.images] : [],
    author: item?.author ? { ...item.author } : item?.author
  }))
  feedOffset.value = inspirationPageCache.feedOffset
  feedTotal.value = inspirationPageCache.feedTotal
  feedType.value = inspirationPageCache.feedType
  selectedTag.value = inspirationPageCache.selectedTag
  availableTags.value = inspirationPageCache.availableTags.map(item => ({ ...item }))
  searchKeyword.value = inspirationPageCache.searchKeyword
  searchQuery.value = inspirationPageCache.searchQuery
}

const saveToCache = () => {
  inspirationPageCache.initialized = true
  inspirationPageCache.posts = snapshotPosts()
  inspirationPageCache.feedOffset = feedOffset.value
  inspirationPageCache.feedTotal = feedTotal.value
  inspirationPageCache.feedType = feedType.value
  inspirationPageCache.selectedTag = selectedTag.value
  inspirationPageCache.availableTags = availableTags.value.map(item => ({ ...item }))
  inspirationPageCache.searchKeyword = searchKeyword.value
  inspirationPageCache.searchQuery = searchQuery.value
  inspirationPageCache.scrollTop = pageRef.value?.scrollTop || 0
}

// 响应式列数
const columnCount = ref(5)
const updateColumnCount = () => {
  const width = window.innerWidth
  if (width <= 560) columnCount.value = 2
  else if (width <= 900) columnCount.value = 3
  else if (width <= 1200) columnCount.value = 4
  else columnCount.value = 5
}

// 将过滤后的 posts 分组到各列 - 按顺序轮询分配到各列
const masonryColumns = computed(() => {
  const columns = []
  const colCount = columnCount.value
  for (let i = 0; i < colCount; i++) {
    columns.push([])
  }
  
  posts.value.forEach((post, index) => {
    const colIndex = index % colCount
    columns[colIndex].push(post)
  })
  
  return columns
})

const canLoadMore = computed(() => posts.value.length < feedTotal.value)

const stableHash = (input) => {
  let hash = 2166136261
  for (let i = 0; i < input.length; i++) {
    hash ^= input.charCodeAt(i)
    hash = Math.imul(hash, 16777619)
  }
  return hash >>> 0
}

const shuffleExploreBatch = (items, offset) => {
  if (!Array.isArray(items) || items.length <= 1 || offset <= 0) return items
  const daySeed = new Date().toISOString().slice(0, 10)
  return [...items].sort((a, b) => {
    const aKey = `${daySeed}:${offset}:${a?.share_id || a?.id || ''}`
    const bKey = `${daySeed}:${offset}:${b?.share_id || b?.id || ''}`
    return stableHash(aKey) - stableHash(bKey)
  })
}

const isVideoUrl = (url) => {
  if (!url || typeof url !== 'string') return false
  return /\.(mp4|mov|webm|m3u8)(\?.*)?$/i.test(url)
}

const isVideoPost = (item) => {
  if (!item) return false
  if (item.type === 'video') return true
  if (item.video_url) return true
  return isVideoUrl(item.cover_url)
}

const cardMediaSrc = (item) => {
  if (!item) return ''
  if (isVideoPost(item)) return item.video_url || item.cover_url || ''
  return item.cover_url || item.images?.[0] || item.video_url || ''
}

const cardVideoPoster = (item) => {
  if (!item) return ''
  // 如果 cover_url 是视频 URL，则返回空，让浏览器自动从视频加载第一帧
  const cover = item.cover_url
  if (cover && !isVideoUrl(cover)) {
    return cover
  }
  // 尝试使用 images 数组中的图片
  if (item.images?.length > 0 && !isVideoUrl(item.images[0])) {
    return item.images[0]
  }
  // 返回空，浏览器会从视频自动截取
  return ''
}

const referenceImageForPost = (item) => {
  if (!item) return ''
  return item.cover_url || item.images?.[0] || ''
}

const videoCardSrc = (item) => {
  if (!item) return ''
  if (item.video_url) return item.video_url
  return isVideoUrl(item.cover_url) ? item.cover_url : ''
}

const normalizeMediaUrl = (url) => {
  if (!url || typeof url !== 'string') return ''
  return url.split('?')[0].replace(/\/+$/, '')
}

const isSameMediaUrl = (a, b) => {
  const na = normalizeMediaUrl(a)
  const nb = normalizeMediaUrl(b)
  return !!na && na === nb
}

const imageCardSrc = (item) => {
  if (!item) return ''
  const videoSrc = videoCardSrc(item)
  const candidates = [item.cover_url, ...(item.images || [])]
  const imageUrl = candidates.find((url) => url && !isVideoUrl(url) && !isSameMediaUrl(url, videoSrc))
  return imageUrl || ''
}

const videoCardPoster = (item) => imageCardSrc(item)

const videoCardRefs = new Map()
const videoHoverPlayTimers = new Map()
const VIDEO_HOVER_PLAY_DELAY_MS = 320

const getVideoCardKey = (item) => item?.share_id || item?.id || ''

const setVideoCardRef = (item, el) => {
  const key = getVideoCardKey(item)
  if (!key) return
  if (el) {
    videoCardRefs.set(key, el)
  } else {
    videoCardRefs.delete(key)
  }
}

const playVideoCard = async (item) => {
  if (!isVideoPost(item)) return
  const key = getVideoCardKey(item)
  if (!key) return
  const existingTimer = videoHoverPlayTimers.get(key)
  if (existingTimer) {
    clearTimeout(existingTimer)
  }
  const timer = setTimeout(async () => {
    videoHoverPlayTimers.delete(key)
    const videoEl = videoCardRefs.get(key)
    if (!videoEl) return
    try {
      await videoEl.play()
    } catch {
    }
  }, VIDEO_HOVER_PLAY_DELAY_MS)
  videoHoverPlayTimers.set(key, timer)
}

const stopVideoCard = (item) => {
  if (!isVideoPost(item)) return
  const key = getVideoCardKey(item)
  if (key) {
    const timer = videoHoverPlayTimers.get(key)
    if (timer) {
      clearTimeout(timer)
      videoHoverPlayTimers.delete(key)
    }
  }
  const videoEl = key ? videoCardRefs.get(key) : null
  if (!videoEl) return
  videoEl.pause()
  try {
    videoEl.currentTime = 0
  } catch {
  }
}

const onPageScroll = () => {
  if (scrollRafId !== null) return
  scrollRafId = requestAnimationFrame(() => {
    scrollRafId = null
    handlePageScroll()
  })
}

const handlePageScroll = () => {
  const el = pageRef.value
  if (!el) return
  showBackTop.value = el.scrollTop > 300
  // 距离底部 1500px 时就开始加载
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 1500 && canLoadMore.value && !feedLoadingMore.value) {
    loadMore()
  }
}

const scrollToTop = () => {
  pageRef.value?.scrollTo({ top: 0, behavior: 'smooth' })
}

const featureCards = computed(() => [
  { icon: '🎨', label: t('inspiration.featureImageGen'), desc: t('inspiration.featureImageDesc'), mode: 'image' },
  { icon: '🎬', label: t('inspiration.featureVideoGen'), desc: t('inspiration.featureVideoDesc'), mode: 'video' },
  { icon: '🛍️', label: t('inspiration.featureEcommerce'), desc: t('inspiration.featureEcommerceDesc'), mode: 'ecommerce' }
])

const feedTypeTabs = computed(() => [
  { key: 'all', label: t('inspiration.tabAll') },
  { key: 'image', label: t('inspiration.tabImage') },
  { key: 'video', label: t('inspiration.tabVideo') }
])

const tagTabs = computed(() => {
  const base = [{ key: '', label: t('inspiration.tabAll') }]
  const extras = availableTags.value.map(item => ({
    key: item.slug || item.name,
    label: item.name
  }))
  return [...base, ...extras]
})

const onFeedTypeChange = async (type) => {
  if (feedType.value === type || feedLoading.value) return
  feedType.value = type
  await fetchFeed(true)
}

const onTagChange = async (tagKey) => {
  if (selectedTag.value === tagKey || feedLoading.value) return
  selectedTag.value = tagKey
  await fetchFeed(true)
}

const doSearch = async () => {
  const keyword = searchKeyword.value.trim()
  if (!keyword) return
  await router.push({
    name: 'inspiration-search',
    query: { q: keyword }
  })
}

const openPublishDialog = () => {
  if (!userStore.requireAuth()) return
  showPublishDialog.value = true
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
    await fetchFeed(true)
  } catch (e) {
    message.error(e.response?.data?.error || t('inspiration.shareFailed'))
  } finally {
    publishLoading.value = false
  }
}

const selectFeature = (mode) => {
  creativeMode.value = mode
  composerRef.value?.focus?.()
}

const fetchFeed = async (reset = false) => {
  cancelFeedRequest()
  feedAbortController = new AbortController()
  const currentController = feedAbortController
  const requestOffset = reset ? 0 : feedOffset.value
  if (reset) {
    feedOffset.value = 0
    feedLoading.value = true
  } else {
    if (!canLoadMore.value) return
    feedLoadingMore.value = true
  }

  try {
    const data = await listInspirations({
      limit: feedLimit,
      offset: requestOffset,
      type: feedType.value,
      tag: selectedTag.value || undefined,
      q: searchQuery.value || undefined
    }, { signal: currentController.signal })
    if (currentController !== feedAbortController) return
    const rawItems = data.items || []
    const items = shuffleExploreBatch(rawItems, requestOffset)
    feedTotal.value = data.total || 0

    if (reset) {
      posts.value = items
    } else {
      posts.value.push(...items)
    }
    feedOffset.value = posts.value.length
    saveToCache()
  } catch (e) {
    if (isCanceledError(e)) return
    if (reset) posts.value = []
    console.error('load inspirations failed', e)
  } finally {
    if (feedAbortController === currentController) {
      feedAbortController = null
    }
    feedLoading.value = false
    feedLoadingMore.value = false
  }
}

const fetchTags = async () => {
  cancelTagRequest()
  tagsAbortController = new AbortController()
  const currentController = tagsAbortController
  try {
    const data = await listInspirationTags({ limit: 20 }, { signal: currentController.signal })
    if (currentController !== tagsAbortController) return
    availableTags.value = data.items || []
    saveToCache()
  } catch (e) {
    if (isCanceledError(e)) return
    availableTags.value = []
  } finally {
    if (tagsAbortController === currentController) {
      tagsAbortController = null
    }
  }
}

const loadMore = async () => {
  if (feedLoading.value || feedLoadingMore.value) return
  await fetchFeed(false)
}

const openDetail = (post) => {
  if (!post?.share_id) return
  router.push(`/inspiration/${post.share_id}`)
}

const quickRemix = async (item) => {
  if (!item) return
  if (!userStore.requireAuth()) return
  composerDraftStore.setRemixDraft(item)
  await markRemix(item.share_id).catch(() => {})
  router.push('/generate')
}

const quickReference = async (item) => {
  if (!item) return
  if (!userStore.requireAuth()) return
  const imageUrl = videoCardPoster(item) || imageCardSrc(item) || referenceImageForPost(item)
  if (!imageUrl) return
  composerDraftStore.setReferenceDraft(item, imageUrl)
  await markRemix(item.share_id).catch(() => {})
  router.push('/generate')
}

const quickToggleLike = async (item) => {
  if (!item?.share_id) return
  if (!userStore.requireAuth()) return

  const isLiked = !!item.is_liked
  try {
    if (isLiked) {
      const resp = await unlikeInspiration(item.share_id)
      item.is_liked = false
      if (typeof resp?.like_count === 'number') item.like_count = resp.like_count
    } else {
      const resp = await likeInspiration(item.share_id)
      item.is_liked = true
      if (typeof resp?.like_count === 'number') item.like_count = resp.like_count
    }
  } catch (e) {
    const errMsg = e.response?.data?.error
    message.error(errMsg || t(isLiked ? 'inspiration.unlikeFailed' : 'inspiration.likeFailed'))
  }
}

const handleSubmit = (payload) => {
  loading.value = true

  genStore.pendingResult = {
    id: Date.now(),
    type: payload.creativeMode,
    prompt: payload.prompt,
    status: payload.creativeMode === 'video' ? 'queued' : 'generating',
    images: [],
    video_url: null,
    credits_cost: payload.credits,
    params: payload.params,
    reference_images: payload.images || [],
    _payload: payload
  }

  router.push('/generate')
}

onMounted(async () => {
  updateColumnCount()
  window.addEventListener('resize', updateColumnCount)
  if (inspirationPageCache.initialized) {
    restoreFromCache()
  } else {
    await fetchTags()
    await fetchFeed(true)
  }
  pageRef.value?.addEventListener('scroll', onPageScroll)
  if (inspirationPageCache.scrollTop > 0) {
    pageRef.value?.scrollTo({ top: inspirationPageCache.scrollTop, behavior: 'auto' })
  }
  handlePageScroll()
})

onBeforeUnmount(() => {
  saveToCache()
  cancelFeedRequest()
  cancelTagRequest()
  window.removeEventListener('resize', updateColumnCount)
  pageRef.value?.removeEventListener('scroll', onPageScroll)
  if (scrollRafId !== null) {
    cancelAnimationFrame(scrollRafId)
    scrollRafId = null
  }
  for (const videoEl of videoCardRefs.values()) {
    try {
      videoEl.pause()
      videoEl.removeAttribute('src')
      videoEl.load()
    } catch {
    }
  }
  for (const timer of videoHoverPlayTimers.values()) {
    clearTimeout(timer)
  }
  videoHoverPlayTimers.clear()
  videoCardRefs.clear()
})
</script>

<template>
  <div class="inspiration-page" ref="pageRef">
    <div class="inspiration-content">
      <ComposerBar
        ref="composerRef"
        v-model:creative-mode="creativeMode"
        :loading="loading"
        @submit="handleSubmit"
      />

      <div class="feature-section">
        <div class="feature-grid">
          <button
            v-for="card in featureCards"
            :key="card.mode"
            class="feature-card"
            :class="{ active: creativeMode === card.mode }"
            @click="selectFeature(card.mode)"
          >
            <span class="feature-icon">{{ card.icon }}</span>
            <div class="feature-info">
              <span class="feature-label">{{ card.label }}</span>
              <span class="feature-desc">{{ card.desc }}</span>
            </div>
          </button>
        </div>
      </div>

      <div class="showcase-section">
        <div class="showcase-header">
          <div class="showcase-header-left">
            <div class="feed-type-tabs">
              <button
                v-for="tab in feedTypeTabs"
                :key="tab.key"
                class="feed-type-tab"
                :class="{ active: feedType === tab.key }"
                @click="onFeedTypeChange(tab.key)"
              >
                {{ tab.label }}
              </button>
            </div>
            <div class="search-box">
              <input
                ref="searchInputRef"
                v-model="searchKeyword"
                type="text"
                :placeholder="t('inspiration.searchPlaceholder')"
                class="search-input"
                @keyup.enter="doSearch"
              />
              <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/>
                <path d="m21 21-4.35-4.35"/>
              </svg>
              <button class="search-btn" @click="doSearch">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="11" cy="11" r="8"/>
                  <path d="m21 21-4.35-4.35"/>
                </svg>
              </button>
            </div>
            <button class="publish-btn" @click="openPublishDialog">{{ t('inspiration.publishAction') }}</button>
          </div>
        </div>

        <div v-if="tagTabs.length > 1" class="tag-tabs">
          <button
            v-for="tag in tagTabs"
            :key="tag.key || 'all'"
            class="tag-tab"
            :class="{ active: selectedTag === tag.key }"
            @click="onTagChange(tag.key)"
          >
            {{ tag.label }}
          </button>
        </div>

        <div v-if="!feedLoading && posts.length" class="showcase-masonry">
          <div
            v-for="(column, colIndex) in masonryColumns"
            :key="colIndex"
            class="masonry-column"
          >
            <article
              v-for="item in column"
              :key="item.share_id"
              class="showcase-item"
              @mouseenter="playVideoCard(item)"
              @mouseleave="stopVideoCard(item)"
              @click="openDetail(item)"
            >
              <video
                v-if="isVideoPost(item) && videoCardSrc(item)"
                :ref="(el) => setVideoCardRef(item, el)"
                class="showcase-media"
                :src="videoCardSrc(item)"
                :poster="videoCardPoster(item)"
                preload="metadata"
                muted
                playsinline
                loop
              />
              <img
                v-else-if="imageCardSrc(item)"
                class="showcase-media"
                :src="imageCardSrc(item)"
                :alt="item.prompt || 'inspiration'"
                loading="lazy"
              />
              <span v-if="isVideoPost(item)" class="showcase-video-badge" title="video">
                <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                  <path d="M8 6v12l10-6z" />
                </svg>
              </span>
              <div class="showcase-overlay">
                <div class="showcase-info">
                  <span v-if="item.title" class="showcase-title-text">{{ item.title }}</span>
                  <div v-if="item.tags?.length" class="showcase-tags">
                    <span v-for="tag in item.tags.slice(0, 2)" :key="tag" class="showcase-tag">#{{ tag }}</span>
                  </div>
                  <span class="showcase-author">{{ item.author?.nickname || t('inspiration.creatorFallback') }}</span>
                </div>

                <div class="showcase-quick-actions">
                  <button class="quick-btn" :title="t('inspiration.remixAction')" @click.stop="quickRemix(item)">
                    <svg class="quick-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                      <path d="M3 12a9 9 0 0 1 15.3-6.36L21 8" stroke-linecap="round" stroke-linejoin="round" />
                      <path d="M21 3v5h-5" stroke-linecap="round" stroke-linejoin="round" />
                      <path d="M21 12a9 9 0 0 1-15.3 6.36L3 16" stroke-linecap="round" stroke-linejoin="round" />
                      <path d="M3 21v-5h5" stroke-linecap="round" stroke-linejoin="round" />
                    </svg>
                  </button>
                  <button class="quick-btn" :title="t('inspiration.referenceAction')" @click.stop="quickReference(item)">
                    <svg class="quick-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                      <rect x="3" y="4" width="18" height="16" rx="2" ry="2" />
                      <circle cx="8.5" cy="9" r="1.5" />
                      <path d="M21 16l-5-5-4 4-2-2-7 7" stroke-linecap="round" stroke-linejoin="round" />
                    </svg>
                  </button>
                  <button class="quick-btn quick-like-btn" :class="{ liked: item.is_liked }" :title="item.is_liked ? t('inspiration.unlikeAction') : t('inspiration.likeAction')" @click.stop="quickToggleLike(item)">
                    <svg v-if="item.is_liked" class="quick-icon" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                      <path d="M12 21s-7.2-4.35-9.58-8.14C.4 9.67 1.52 5.5 5.5 4.59A5.57 5.57 0 0 1 12 7.09a5.57 5.57 0 0 1 6.5-2.5c3.98.91 5.1 5.08 3.08 8.27C19.2 16.65 12 21 12 21z" />
                    </svg>
                    <svg v-else class="quick-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                      <path d="M12 21s-7.2-4.35-9.58-8.14C.4 9.67 1.52 5.5 5.5 4.59A5.57 5.57 0 0 1 12 7.09a5.57 5.57 0 0 1 6.5-2.5c3.98.91 5.1 5.08 3.08 8.27C19.2 16.65 12 21 12 21z" />
                    </svg>
                    <span class="quick-like-count">{{ item.like_count || 0 }}</span>
                  </button>
                </div>
              </div>
            </article>
          </div>
        </div>

        <div v-else-if="feedLoading" class="feed-placeholder">{{ t('inspiration.feedLoading') }}</div>
        <div v-else-if="searchQuery && !posts.length" class="feed-placeholder">{{ t('inspiration.searchEmpty') }}</div>
        <div v-else class="feed-placeholder">{{ t('inspiration.feedEmpty') }}</div>

        <div v-if="feedLoadingMore" class="feed-more">{{ t('inspiration.feedLoading') }}</div>
      </div>
    </div>

    <transition name="fade">
      <button v-show="showBackTop" class="back-top-btn" @click="scrollToTop">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M18 15l-6-6-6 6"/>
        </svg>
      </button>
    </transition>

    <ShareGenerationDialog
      v-model:show="showPublishDialog"
      :loading="publishLoading"
      mode="upload"
      @confirm="handlePublishConfirm"
    />
  </div>
</template>

<style scoped>
.inspiration-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  scroll-behavior: auto;
}

.inspiration-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 6vh 40px 48px;
  gap: 28px;
}

.feature-section {
  max-width: 900px;
  width: 100%;
}

.feature-grid {
  display: flex;
  gap: 12px;
}

.feature-card {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  background: var(--color-tint-white-03, rgba(255,255,255,.03));
  border: 1px solid var(--color-tint-white-06, rgba(255,255,255,.06));
  border-radius: 12px;
  cursor: pointer;
  text-align: left;
  color: var(--color-text-primary);
  transition: all .3s cubic-bezier(.4,0,.2,1);
}

.feature-card:hover {
  background: rgba(0, 202, 224, 0.06);
  border-color: rgba(0, 202, 224, 0.2);
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0, 202, 224, 0.12);
}

.feature-card.active {
  background: rgba(0, 202, 224, 0.1);
  border-color: rgba(0, 202, 224, 0.4);
  box-shadow: 0 0 0 1px rgba(0, 202, 224, 0.2), 0 4px 16px rgba(0, 202, 224, 0.15);
}

.feature-icon {
  font-size: 22px;
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-tint-white-04, rgba(255,255,255,.04));
  border-radius: 10px;
}

.feature-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.feature-label {
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}

.feature-desc {
  font-size: 11px;
  color: var(--color-text-tertiary);
  white-space: nowrap;
}

.showcase-section {
  width: 100%;
  margin-top: 12px;
}

.showcase-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  padding-left: 4px;
}

.showcase-header-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
}

.search-input {
  width: 240px;
  height: 36px;
  padding: 0 40px 0 36px;
  font-size: 14px;
  color: var(--color-text-primary);
  background: var(--color-tint-white-03, rgba(255,255,255,.03));
  border: 1px solid var(--color-tint-white-06, rgba(255,255,255,.06));
  border-radius: 10px;
  outline: none;
  transition: all .2s;
}

.search-input::placeholder {
  color: rgba(148, 158, 175, 0.64) !important;
  opacity: 1;
}

.search-input:focus {
  background: var(--color-tint-white-05, rgba(255,255,255,.05));
  border-color: rgba(0, 202, 224, 0.5);
  box-shadow: 0 0 0 3px rgba(0, 202, 224, 0.1);
}

.search-icon {
  position: absolute;
  left: 12px;
  width: 16px;
  height: 16px;
  color: var(--color-text-tertiary);
  pointer-events: none;
}

.search-btn {
  position: absolute;
  right: 6px;
  width: 28px;
  height: 28px;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 202, 224, 0.9);
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all .2s;
}

.search-btn svg {
  width: 14px;
  height: 14px;
  color: #fff;
}

.search-btn:hover {
  background: rgba(0, 202, 224, 1);
  box-shadow: 0 2px 8px rgba(0, 202, 224, 0.35);
}

.publish-btn {
  height: 36px;
  padding: 0 14px;
  border-radius: 10px;
  border: 1px solid rgba(0, 202, 224, 0.35);
  background: rgba(0, 202, 224, 0.18);
  color: #d8fbff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all .2s;
}

.publish-btn:hover {
  background: rgba(0, 202, 224, 0.3);
  box-shadow: 0 2px 10px rgba(0, 202, 224, 0.25);
}

.feed-type-tabs {
  display: flex;
  gap: 8px;
  background: var(--color-tint-white-03, rgba(255,255,255,.03));
  border: 1px solid var(--color-tint-white-06, rgba(255,255,255,.06));
  border-radius: 10px;
  padding: 4px;
}

.feed-type-tab {
  padding: 6px 14px;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-tertiary);
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all .2s;
}

.feed-type-tab:hover {
  color: var(--color-text-primary);
  background: rgba(255,255,255,.05);
}

.feed-type-tab.active {
  color: #fff;
  background: rgba(0, 202, 224, 0.9);
  box-shadow: 0 2px 8px rgba(0, 202, 224, 0.25);
}

.tag-tabs {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin: 0 0 12px 4px;
}

.tag-tab {
  padding: 4px 10px;
  border-radius: 999px;
  border: 1px solid var(--color-tint-white-08, rgba(255,255,255,.08));
  background: var(--color-tint-white-03, rgba(255,255,255,.03));
  color: var(--color-text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all .2s;
}

.tag-tab:hover {
  border-color: rgba(0, 202, 224, 0.35);
  color: var(--color-text-primary);
}

.tag-tab.active {
  border-color: rgba(0, 202, 224, 0.45);
  background: rgba(0, 202, 224, 0.14);
  color: #c9f8ff;
}

@media (max-width: 900px) {
  .showcase-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  .showcase-header-left {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    width: 100%;
  }
  .search-input {
    width: 100%;
    max-width: 300px;
  }
}

.showcase-masonry {
  display: flex;
  gap: 4px;
}

.masonry-column {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.showcase-item {
  position: relative;
  border-radius: 0;
  overflow: hidden;
  cursor: pointer;
  border: none;
  background: rgba(0, 0, 0, 0.3);
  display: block;
  width: 100%;
}

.showcase-video-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 3;
  width: 24px;
  height: 24px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.56);
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.showcase-video-badge svg {
  width: 14px;
  height: 14px;
}

.showcase-media {
  width: 100%;
  height: auto;
  display: block;
  transition: transform .35s ease;
  background: #0b0f15;
}

.showcase-item:hover .showcase-media {
  transform: scale(1.04);
}

.showcase-overlay {
  position: absolute;
  inset: 0;
  padding: 10px;
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  opacity: 0;
  background: linear-gradient(180deg, rgba(0,0,0,0.06) 0%, rgba(0,0,0,0.5) 100%);
  transition: opacity .22s ease;
}

.showcase-item:hover .showcase-overlay {
  opacity: 1;
}

.showcase-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 60%;
}

.showcase-title-text {
  color: #fff;
  font-size: 14px;
  font-weight: 700;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.showcase-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.showcase-tag {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.9);
  background: rgba(0, 202, 224, 0.24);
  border: 1px solid rgba(0, 202, 224, 0.35);
  padding: 1px 6px;
  border-radius: 999px;
}

.showcase-author {
  color: rgba(255, 255, 255, 0.92);
  font-size: 12px;
  font-weight: 600;
}

.showcase-quick-actions {
  display: flex;
  flex-direction: row;
  gap: 6px;
}

.quick-btn {
  width: 28px;
  height: 28px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(0, 0, 0, 0.4);
  color: #fff;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: all .2s;
}

.quick-icon {
  width: 14px;
  height: 14px;
  display: block;
}

.quick-btn:hover {
  border-color: rgba(0, 202, 224, 0.45);
  background: rgba(0, 202, 224, 0.2);
}

.quick-btn.liked {
  color: #ff879f;
}

.quick-like-btn {
  width: auto;
  min-width: 40px;
  padding: 0 8px;
  gap: 4px;
}

.quick-like-count {
  font-size: 12px;
  line-height: 1;
}

.feed-placeholder,
.feed-more {
  width: 100%;
  text-align: center;
  padding: 24px 0;
  color: var(--color-text-muted);
  font-size: 13px;
}

.back-top-btn {
  position: fixed;
  right: 22px;
  bottom: 22px;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(0, 0, 0, 0.42);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  backdrop-filter: blur(6px);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity .2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 900px) {
  .inspiration-content {
    padding: 20px 16px 72px;
    gap: 20px;
  }

  .feature-grid {
    display: grid;
    grid-template-columns: 1fr;
  }
}
</style>
