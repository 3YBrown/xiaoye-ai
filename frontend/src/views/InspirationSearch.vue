<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useInspiration } from '../composables/useInspiration'
import { useComposerDraftStore } from '../stores/composerDraft'
import { useUserStore } from '../stores/user'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()
const composerDraftStore = useComposerDraftStore()
const { listInspirations, markRemix, likeInspiration, unlikeInspiration } = useInspiration()

const pageRef = ref(null)
const posts = ref([])
const feedLoading = ref(false)
const feedLoadingMore = ref(false)
const feedLimit = 20
const feedOffset = ref(0)
const feedTotal = ref(0)
const feedType = ref('all')
const searchKeyword = ref('')
const searchQuery = ref('')
const showBackTop = ref(false)
const columnCount = ref(5)

let scrollRafId = null
let feedAbortController = null

const canLoadMore = computed(() => posts.value.length < feedTotal.value)

const feedTypeTabs = computed(() => [
  { key: 'all', label: t('inspiration.tabAll') },
  { key: 'image', label: t('inspiration.tabImage') },
  { key: 'video', label: t('inspiration.tabVideo') }
])

const updateColumnCount = () => {
  const width = window.innerWidth
  if (width <= 560) columnCount.value = 2
  else if (width <= 900) columnCount.value = 3
  else if (width <= 1200) columnCount.value = 4
  else columnCount.value = 5
}

const masonryColumns = computed(() => {
  const columns = []
  const colCount = columnCount.value
  for (let i = 0; i < colCount; i++) {
    columns.push([])
  }
  posts.value.forEach((post, index) => {
    columns[index % colCount].push(post)
  })
  return columns
})

const isCanceledError = (error) => {
  return error?.code === 'ERR_CANCELED' || error?.name === 'CanceledError'
}

const cancelFeedRequest = () => {
  if (!feedAbortController) return
  feedAbortController.abort()
  feedAbortController = null
}

const applyRouteQuery = () => {
  const keyword = String(route.query.q || '').trim()
  const changed = keyword !== searchQuery.value
  searchQuery.value = keyword
  searchKeyword.value = keyword
  return changed
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

  if (!searchQuery.value) {
    if (reset) {
      posts.value = []
      feedTotal.value = 0
      feedOffset.value = 0
    }
    feedLoading.value = false
    feedLoadingMore.value = false
    return
  }

  try {
    const data = await listInspirations({
      limit: feedLimit,
      offset: requestOffset,
      type: feedType.value,
      q: searchQuery.value
    }, { signal: currentController.signal })
    if (currentController !== feedAbortController) return

    const items = data.items || []
    feedTotal.value = data.total || 0
    if (reset) {
      posts.value = items
    } else {
      posts.value.push(...items)
    }
    feedOffset.value = posts.value.length
  } catch (e) {
    if (isCanceledError(e)) return
    if (reset) posts.value = []
    console.error('load search inspirations failed', e)
  } finally {
    if (feedAbortController === currentController) {
      feedAbortController = null
    }
    feedLoading.value = false
    feedLoadingMore.value = false
  }
}

const onFeedTypeChange = async (type) => {
  if (feedType.value === type || feedLoading.value) return
  feedType.value = type
  await fetchFeed(true)
}

const loadMore = async () => {
  if (feedLoading.value || feedLoadingMore.value) return
  await fetchFeed(false)
}

const onPageScroll = (event) => {
  if (scrollRafId !== null) return
  const target = event?.target || pageRef.value
  scrollRafId = requestAnimationFrame(() => {
    scrollRafId = null
    if (!target) return
    showBackTop.value = target.scrollTop > 300
    if (target.scrollTop + target.clientHeight >= target.scrollHeight - 1500 && canLoadMore.value && !feedLoadingMore.value) {
      loadMore()
    }
  })
}

const scrollToTop = () => {
  pageRef.value?.scrollTo({ top: 0, behavior: 'smooth' })
}

const doSearch = async () => {
  const keyword = searchKeyword.value.trim()
  if (!keyword) return
  if (keyword === searchQuery.value) {
    pageRef.value?.scrollTo({ top: 0, behavior: 'smooth' })
    await fetchFeed(true)
    return
  }
  await router.push({
    name: 'inspiration-search',
    query: { q: keyword }
  })
}

const goBack = () => {
  router.push('/inspiration')
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
  const imageUrl = videoCardPoster(item) || imageCardSrc(item) || item.cover_url || item.images?.[0] || ''
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

watch(() => route.query.q, async () => {
  const changed = applyRouteQuery()
  if (!changed) return
  pageRef.value?.scrollTo({ top: 0, behavior: 'auto' })
  await fetchFeed(true)
})

onMounted(async () => {
  updateColumnCount()
  window.addEventListener('resize', updateColumnCount)
  applyRouteQuery()
  await fetchFeed(true)
})

onBeforeUnmount(() => {
  cancelFeedRequest()
  window.removeEventListener('resize', updateColumnCount)
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
  <div class="search-page" ref="pageRef" @scroll="onPageScroll">
    <div class="search-content">
      <div class="search-toolbar">
        <button class="back-btn" @click="goBack" aria-label="back">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M15 18l-6-6 6-6" />
          </svg>
        </button>
        <div class="search-box">
          <input
            v-model="searchKeyword"
            type="text"
            :placeholder="t('inspiration.searchPlaceholder')"
            class="search-input"
            @keyup.enter="doSearch"
          />
          <button class="search-btn" @click="doSearch">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8" />
              <path d="m21 21-4.35-4.35" />
            </svg>
          </button>
        </div>
      </div>

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
      <div v-else-if="searchQuery" class="feed-placeholder">{{ t('inspiration.searchEmpty') }}</div>

      <div v-if="feedLoadingMore" class="feed-more">{{ t('inspiration.feedLoading') }}</div>
    </div>

    <transition name="fade">
      <button v-show="showBackTop" class="back-top-btn" @click="scrollToTop">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M18 15l-6-6-6 6" />
        </svg>
      </button>
    </transition>
  </div>
</template>

<style scoped>
.search-page {
  height: 100%;
  overflow-y: auto;
}

.search-content {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 20px 28px 40px;
}

.search-toolbar {
  position: sticky;
  top: 0;
  z-index: 8;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  backdrop-filter: blur(8px);
}

.back-btn {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  border: 1px solid var(--color-tint-white-08, rgba(255, 255, 255, 0.08));
  background: var(--color-tint-white-04, rgba(255, 255, 255, 0.04));
  color: var(--color-text-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.back-btn svg {
  width: 16px;
  height: 16px;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  max-width: 420px;
}

.search-input {
  width: 100%;
  height: 36px;
  padding: 0 42px 0 12px;
  font-size: 15px;
  line-height: 1.65;
  color: var(--color-text-primary);
  background: var(--color-tint-white-03, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--color-tint-white-08, rgba(255, 255, 255, 0.08));
  border-radius: 10px;
  outline: none;
}

.search-input::placeholder {
  color: rgba(148, 158, 175, 0.64) !important;
  font-size: 15px;
  line-height: 1.65;
  opacity: 1;
}

.search-btn {
  position: absolute;
  right: 6px;
  width: 26px;
  height: 26px;
  padding: 0;
  border: none;
  border-radius: 8px;
  background: rgba(0, 202, 224, 0.9);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.search-btn svg {
  width: 14px;
  height: 14px;
}

.feed-type-tabs {
  display: flex;
  gap: 8px;
  background: var(--color-tint-white-03, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--color-tint-white-06, rgba(255, 255, 255, 0.06));
  border-radius: 10px;
  padding: 4px;
  width: fit-content;
}

.feed-type-tab {
  padding: 6px 14px;
  font-size: 13px;
  color: var(--color-text-tertiary);
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
}

.feed-type-tab.active {
  color: #fff;
  background: rgba(0, 202, 224, 0.9);
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
  overflow: hidden;
  cursor: pointer;
  background: rgba(0, 0, 0, 0.3);
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
  background: #0b0f15;
}

.showcase-overlay {
  position: absolute;
  inset: 0;
  padding: 10px;
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  opacity: 0;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.06) 0%, rgba(0, 0, 0, 0.5) 100%);
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
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.quick-btn:hover {
  border-color: rgba(0, 202, 224, 0.45);
  background: rgba(0, 202, 224, 0.2);
}

.quick-btn.liked {
  color: #ff879f;
}

.quick-icon {
  width: 14px;
  height: 14px;
  display: block;
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
  .search-content {
    padding: 14px 12px 68px;
  }

  .search-toolbar {
    gap: 8px;
  }

  .search-box {
    max-width: none;
  }
}
</style>
