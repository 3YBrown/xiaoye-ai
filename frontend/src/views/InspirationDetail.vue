<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useInspiration } from '../composables/useInspiration'
import { useComposerDraftStore } from '../stores/composerDraft'
import { useUserStore } from '../stores/user'
import { useModelsStore } from '../stores/models'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()
const modelsStore = useModelsStore()
const { getInspiration, getInspirationLikeStatus, likeInspiration, unlikeInspiration, markRemix } = useInspiration()
const composerDraftStore = useComposerDraftStore()

const loading = ref(true)
const likeLoading = ref(false)
const post = ref(null)
const mediaIndex = ref(0)

const isVideoUrl = (url) => {
  if (!url || typeof url !== 'string') return false
  return /\.(mp4|mov|webm|m3u8)(\?.*)?$/i.test(url)
}

const isVideoPost = computed(() => {
  if (!post.value) return false
  if (post.value.type === 'video') return true
  if (post.value.video_url) return true
  return isVideoUrl(post.value.cover_url)
})

const mediaUrls = computed(() => {
  if (!post.value) return []
  if (isVideoPost.value) {
    const posterCandidates = [post.value.cover_url, post.value.images?.[0]]
    const posters = posterCandidates.filter(Boolean).filter(url => !isVideoUrl(url))
    if (posters.length) return posters
    return []
  }
  if (Array.isArray(post.value.images) && post.value.images.length) return post.value.images
  if (post.value.cover_url) return [post.value.cover_url]
  return []
})

const activeImage = computed(() => mediaUrls.value[mediaIndex.value] || '')
const activeVideoUrl = computed(() => {
  if (!post.value) return ''
  return post.value.video_url || (isVideoUrl(post.value.cover_url) ? post.value.cover_url : '')
})

const publishedTime = computed(() => {
  if (!post.value?.published_at) return ''
  return new Date(post.value.published_at).toLocaleString()
})

const rawParams = computed(() => {
  const params = post.value?.params
  if (!params || typeof params !== 'object' || Array.isArray(params)) return {}
  return params
})

const modelName = computed(() => {
  const model = rawParams.value?.model
  if (typeof model === 'string' && model.trim()) return modelsStore.getDisplayName(model)
  return t('inspiration.modelUnknown')
})

const prettifyParamKey = (key) => {
  if (!key) return ''
  return key
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/[_-]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .replace(/^\w/, (c) => c.toUpperCase())
}

const paramLabel = (key) => {
  const i18nKey = `inspiration.paramLabel.${key}`
  const translated = t(i18nKey)
  if (translated !== i18nKey) return translated
  return prettifyParamKey(key)
}

const formatParamValue = (value) => {
  if (value === null || typeof value === 'undefined' || value === '') return '-'
  if (typeof value === 'boolean') return value ? 'true' : 'false'
  if (Array.isArray(value)) return value.length ? value.join(', ') : '-'
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value)
    } catch (_) {
      return String(value)
    }
  }
  return String(value)
}

const paramsForDisplay = computed(() => {
  const params = rawParams.value
  const preferredOrder = [
    'mode',
    'provider',
    'aspectRatio',
    'imageSize',
    'resolution',
    'ratio',
    'duration',
    'generateAudio',
    'outputCount',
    'imageType',
    'ecommerceType'
  ]
  const orderMap = new Map(preferredOrder.map((key, index) => [key, index]))

  return Object.entries(params)
    .filter(([key]) => key !== 'model')
    .sort(([a], [b]) => {
      const ai = orderMap.has(a) ? orderMap.get(a) : Number.MAX_SAFE_INTEGER
      const bi = orderMap.has(b) ? orderMap.get(b) : Number.MAX_SAFE_INTEGER
      if (ai !== bi) return ai - bi
      return a.localeCompare(b)
    })
    .map(([key, value]) => ({
      key,
      label: paramLabel(key),
      value: formatParamValue(value)
    }))
})

const loadDetail = async () => {
  loading.value = true
  try {
    post.value = await getInspiration(route.params.shareId)
    if (post.value && userStore.isLoggedIn) {
      try {
        const state = await getInspirationLikeStatus(post.value.share_id)
        post.value.is_liked = !!state?.liked
        if (typeof state?.like_count === 'number') {
          post.value.like_count = state.like_count
        }
      } catch (_) {}
    }
    mediaIndex.value = 0
  } catch (e) {
    post.value = null
    console.error('load inspiration detail failed', e)
  } finally {
    loading.value = false
  }
}

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
    return
  }
  router.push('/inspiration')
}

const handleRemix = async () => {
  if (!post.value) return
  if (!userStore.requireAuth()) return
  composerDraftStore.setRemixDraft(post.value)
  await markRemix(post.value.share_id).catch(() => {})
  router.push('/generate')
}

const handleReference = async () => {
  if (!post.value) return
  if (!userStore.requireAuth()) return
  const imageUrl = activeImage.value || post.value.cover_url
  if (!imageUrl) return
  composerDraftStore.setReferenceDraft(post.value, imageUrl)
  await markRemix(post.value.share_id).catch(() => {})
  router.push('/generate')
}

const toggleLike = async () => {
  if (!post.value?.share_id) return
  if (likeLoading.value) return
  if (!userStore.requireAuth()) return

  likeLoading.value = true
  const isLiked = !!post.value.is_liked
  try {
    if (isLiked) {
      const resp = await unlikeInspiration(post.value.share_id)
      post.value.is_liked = false
      if (typeof resp?.like_count === 'number') {
        post.value.like_count = resp.like_count
      } else {
        post.value.like_count = Math.max(0, (post.value.like_count || 0) - 1)
      }
      message.success(t('inspiration.unlikeSuccess'))
    } else {
      const resp = await likeInspiration(post.value.share_id)
      post.value.is_liked = true
      if (typeof resp?.like_count === 'number') {
        post.value.like_count = resp.like_count
      } else {
        post.value.like_count = (post.value.like_count || 0) + 1
      }
      message.success(t('inspiration.likeSuccess'))
    }
  } catch (e) {
    const msg = e.response?.data?.error
    if (isLiked) {
      message.error(msg || t('inspiration.unlikeFailed'))
    } else {
      message.error(msg || t('inspiration.likeFailed'))
    }
  } finally {
    likeLoading.value = false
  }
}

const copyPrompt = async () => {
  const text = String(post.value?.prompt || '').trim()
  if (!text) return

  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const ta = document.createElement('textarea')
      ta.value = text
      ta.setAttribute('readonly', 'readonly')
      ta.style.position = 'fixed'
      ta.style.top = '-1000px'
      ta.style.left = '-1000px'
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    message.success('Copied')
  } catch (_) {
    message.error('Copy failed')
  }
}

onMounted(loadDetail)
</script>

<template>
  <div class="detail-page">
    <div v-if="loading" class="detail-loading">{{ t('inspiration.detailLoading') }}</div>
    <div v-else-if="!post" class="detail-loading">{{ t('inspiration.detailNotFound') }}</div>
    <div v-else class="detail-layout">
      <section class="detail-media">
        <button class="back-btn" @click="goBack">×</button>
        <video
          v-if="isVideoPost && activeVideoUrl"
          class="main-video"
          :src="activeVideoUrl"
          :poster="activeImage || ''"
          controls
          playsinline
          preload="metadata"
          autoplay
          muted
          loop
        />
        <img v-else-if="activeImage" :src="activeImage" alt="inspiration" class="main-image" />
        <div v-if="mediaUrls.length > 1" class="thumb-row">
          <button
            v-for="(url, idx) in mediaUrls"
            :key="url + idx"
            class="thumb-btn"
            :class="{ active: mediaIndex === idx }"
            @click="mediaIndex = idx"
          >
            <img :src="url" alt="thumb" />
          </button>
        </div>
      </section>

      <aside class="detail-panel">
        <div class="creator-row">
          <div class="creator-avatar">
            {{ (post.author?.nickname || t('inspiration.creatorFallback')).slice(0, 1).toUpperCase() }}
          </div>
          <div class="creator-meta">
            <div class="creator-name">{{ post.author?.nickname || t('inspiration.creatorFallback') }}</div>
            <div class="creator-time">{{ publishedTime }}</div>
          </div>
          <button class="creator-stats" :class="{ active: post.is_liked }" :disabled="likeLoading" @click="toggleLike">
            {{ post.is_liked ? '❤' : '♡' }} {{ post.like_count || 0 }}
          </button>
        </div>

        <div v-if="post.title" class="title-wrap">
          <h1 class="post-title">{{ post.title }}</h1>
          <p v-if="post.description" class="post-description">{{ post.description }}</p>
        </div>
        <div v-if="post.tags?.length" class="post-tags">
          <span v-for="tag in post.tags" :key="tag" class="post-tag">#{{ tag }}</span>
        </div>

        <div class="prompt-wrap prompt-scroll">
          <div class="prompt-title-row">
            <div class="prompt-title">{{ t('inspiration.promptTitle') }}</div>
            <button class="prompt-copy-btn" type="button" title="Copy prompt" @click="copyPrompt">
              <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <rect x="9" y="9" width="10" height="10" rx="2" />
                <path d="M6 15H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v1" />
              </svg>
            </button>
          </div>
          <p class="prompt-text">{{ post.prompt }}</p>
        </div>

        <div v-if="paramsForDisplay.length || modelName || post.type" class="prompt-wrap">
          <div class="prompt-title">{{ t('inspiration.paramsTitle') }}<span v-if="post.type" class="title-type"> · {{ post.type === 'video' ? '视频' : '图片' }}</span></div>
          <div class="params-inline">
            <span v-if="modelName" class="param-item">{{ modelName }}</span>
            <span v-if="modelName && paramsForDisplay.length" class="param-separator">|</span>
            <span v-for="(item, index) in paramsForDisplay" :key="item.key" class="param-item">
              {{ item.value }}<span v-if="index < paramsForDisplay.length - 1" class="param-separator">|</span>
            </span>
          </div>
        </div>

        <div class="meta-row">
          <span>{{ post.view_count || 0 }} {{ t('inspiration.views') }}</span>
          <span>{{ post.remix_count || 0 }} {{ t('inspiration.remixes') }}</span>
        </div>

        <div class="action-row">
          <button class="action-btn ghost" @click="toggleLike" :disabled="likeLoading">
            {{ post.is_liked ? t('inspiration.unlikeAction') : t('inspiration.likeAction') }}
          </button>
          <button class="action-btn" @click="handleRemix">{{ t('inspiration.remixAction') }}</button>
          <button class="action-btn ghost" @click="handleReference">{{ t('inspiration.referenceAction') }}</button>
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.detail-page {
  height: 100%;
  overflow: auto;
  padding: 16px;
}

.detail-layout {
  min-height: 100%;
  display: grid;
  grid-template-columns: minmax(320px, 1fr) 360px;
  gap: 18px;
  align-items: start;
}

.detail-media {
  position: sticky;
  top: 16px;
  height: calc(100vh - 32px);
  border-radius: 16px;
  border: 1px solid var(--color-tint-white-08);
  background: rgba(0, 0, 0, 0.28);
  padding: 14px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.back-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  border: none;
  color: #fff;
  background: rgba(0, 0, 0, 0.45);
  cursor: pointer;
  font-size: 24px;
  line-height: 1;
}

.main-image {
  width: min(100%, 640px);
  max-height: 78vh;
  object-fit: contain;
  border-radius: 12px;
}

.main-video {
  width: min(100%, 720px);
  max-height: 78vh;
  border-radius: 12px;
  background: #000;
}

.thumb-row {
  width: 100%;
  margin-top: 12px;
  display: flex;
  gap: 8px;
  overflow-x: auto;
}

.thumb-btn {
  flex: 0 0 auto;
  width: 68px;
  height: 68px;
  border-radius: 10px;
  border: 1px solid transparent;
  background: transparent;
  padding: 0;
  cursor: pointer;
  overflow: hidden;
}

.thumb-btn img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.thumb-btn.active {
  border-color: #00cae0;
}

.detail-panel {
  position: sticky;
  top: 16px;
  height: calc(100vh - 32px);
  border-radius: 16px;
  border: 1px solid var(--color-tint-white-08);
  background: rgba(0, 0, 0, 0.22);
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
}

.creator-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.creator-avatar {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: rgba(0, 202, 224, 0.2);
  color: #8cefff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
}

.creator-meta {
  flex: 1;
  min-width: 0;
}

.creator-name {
  font-size: 14px;
  font-weight: 600;
}

.creator-time {
  margin-top: 2px;
  font-size: 12px;
  color: var(--color-text-muted);
}

.creator-stats {
  border: none;
  background: transparent;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 999px;
  font-size: 13px;
  color: var(--color-text-secondary);
}
.creator-stats.active {
  color: #ff7d98;
  background: rgba(255, 125, 152, 0.12);
}

.title-wrap {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.post-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
  line-height: 1.4;
}

.post-description {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.post-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0 0 10px;
}

.post-tag {
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 12px;
  color: #bceef6;
  border: 1px solid rgba(0, 202, 224, 0.35);
  background: rgba(0, 202, 224, 0.12);
}

.prompt-wrap {
  background: var(--color-tint-white-03);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 12px;
  padding: 12px;
}

.prompt-wrap.prompt-scroll {
  flex: 1;
  min-height: 120px;
  display: flex;
  flex-direction: column;
}

.prompt-title {
  font-size: 12px;
  color: var(--color-text-muted);
}

.prompt-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.prompt-copy-btn {
  width: 26px;
  height: 26px;
  border-radius: 8px;
  border: 1px solid var(--color-tint-white-08);
  background: var(--color-tint-white-03);
  color: var(--color-text-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all .18s ease;
}

.prompt-copy-btn:hover {
  border-color: rgba(0, 202, 224, 0.4);
  color: #c9f8ff;
  background: rgba(0, 202, 224, 0.12);
}

.prompt-copy-btn svg {
  width: 15px;
  height: 15px;
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.prompt-text {
  margin: 8px 0 0;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

.prompt-wrap.prompt-scroll .prompt-text {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-right: 6px;
}

.params-inline {
  margin-top: 8px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 8px;
  font-size: 13px;
  color: var(--color-text-primary);
}

.title-type {
  color: var(--color-text-muted);
  font-weight: 400;
}

.param-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.param-separator {
  color: var(--color-text-muted);
  opacity: 0.5;
}

.meta-row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  color: var(--color-text-muted);
  font-size: 12px;
}

.action-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 10px;
}

.action-btn {
  height: 40px;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  font-weight: 600;
  color: #0b1018;
  background: #9befff;
}

.action-btn.ghost {
  color: #d8f8ff;
  background: rgba(155, 239, 255, 0.16);
}

.detail-loading {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
}

@media (max-width: 980px) {
  .detail-layout {
    grid-template-columns: 1fr;
  }

  .detail-media {
    position: static;
    top: auto;
    height: auto;
    overflow: visible;
  }

  .detail-panel {
    position: static;
    top: auto;
    height: auto;
    overflow: visible;
  }

  .prompt-wrap.prompt-scroll {
    min-height: 0;
  }

  .prompt-wrap.prompt-scroll .prompt-text {
    overflow: visible;
    padding-right: 0;
  }

  .main-image {
    max-height: 60vh;
  }
}
</style>
