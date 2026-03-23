<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NSpin, NImage, NImageGroup, NButton, NPopover, useMessage } from 'naive-ui'
import { useUserStore } from '../stores/user'
import { useGenerationStore } from '../stores/generation'
import { useGenerate } from '../composables/useGenerate'
import { useInspiration } from '../composables/useInspiration'
import { useComposerDraftStore } from '../stores/composerDraft'
import ComposerBar from '../components/ComposerBar.vue'
import ShareGenerationDialog from '../components/ShareGenerationDialog.vue'
import ImageEditor from '../components/ImageEditor.vue'

const { t } = useI18n()
const router = useRouter()
const userStore = useUserStore()
const genStore = useGenerationStore()
const composerDraftStore = useComposerDraftStore()
const message = useMessage()
const { generate, pollVideoTask, pollTask, stopAllPolls, uploadImageToOSS } = useGenerate()
const { markRemix, shareGeneration, unshareInspiration } = useInspiration()

const composerRef = ref(null)
const creativeMode = ref('image')
const loading = ref(false)
const resultsContainer = ref(null)
const currentResults = ref([])
const isFirstLoad = ref(true)
const shareLoadingIds = ref({})
const generationIDToShare = ref(null)
const generationToShare = ref(null)
const showShareDialog = ref(false)
const shareLoading = ref(false)
const showImageEditor = ref(false)
const imageEditorSrc = ref('')
const imageEditorRef = ref(null)

onMounted(async () => {
  if (!genStore.hasLoaded) {
    await genStore.load(true)
  }
  isFirstLoad.value = false

  resumePendingPolls()

  if (genStore.pendingResult) {
    const pending = genStore.pendingResult
    genStore.pendingResult = null

    const resultItem = reactive({ ...pending })
    delete resultItem._payload
    currentResults.value.push(resultItem)

    if (pending._payload) {
      await executeGeneration(resultItem, pending._payload)
    } else if (pending.id && !['success', 'failed'].includes(pending.status)) {
      startPoll(resultItem, pending.id)
    }
  }

  await nextTick()
  applyComposerDraft()
  scrollToBottom()
})

function applyComposerDraft() {
  const draft = composerDraftStore.consumeDraft()
  if (!draft) return

  const mode = draft.mode || 'image'
  creativeMode.value = mode

  composerRef.value?.fillFromGeneration({
    type: mode,
    prompt: draft.prompt || '',
    params: draft.params || {}
  })

  if (draft.referenceImage) {
    composerRef.value?.fillEditImage(draft.referenceImage)
  }

  if (draft.shareId) {
    markRemix(draft.shareId).catch(() => {})
  }
}

function resumePendingPolls() {
  const pendingStatuses = ['queued', 'running', 'generating']
  for (const gen of genStore.generations) {
    const taskId = gen.id || gen.task_id
    if (taskId && pendingStatuses.includes(gen.status)) {
      pollTask(taskId, (update) => {
        if (update.status === 'success') {
          gen.status = 'success'
          gen.images = update.images || gen.images
          gen.video_url = update.video_url || gen.video_url
          gen.credits_cost = update.credits_spent
          userStore.fetchUserInfo()
        } else if (update.status === 'failed') {
          gen.status = 'failed'
          gen.error_msg = update.error_msg
          userStore.fetchUserInfo()
        } else {
          gen.status = update.status
        }
      })
    }
  }
}

onUnmounted(() => {
  stopAllPolls()
})

async function executeGeneration(resultItem, payload) {
  loading.value = true
  try {
    let apiPayload = {}

    if (payload.creativeMode === 'video') {
      apiPayload = {
        type: 'video',
        prompt: payload.prompt,
        model: payload.params.model,
        images: payload.images || [],
        params: {
          mode: payload.params.mode,
          resolution: payload.params.resolution,
          ratio: payload.params.ratio,
          duration: payload.params.duration,
          generate_audio: payload.params.generate_audio,
          first_frame: payload.firstFrameUrl || undefined,
          last_frame: payload.lastFrameUrl || undefined
        }
      }
    } else if (payload.creativeMode === 'ecommerce') {
      apiPayload = {
        type: 'ecommerce',
        prompt: payload.prompt,
        images: payload.imageBase64s,
        model: payload.params.model,
        params: {
          aspectRatio: payload.params.aspectRatio,
          imageSize: payload.params.imageSize,
          outputCount: payload.params.outputCount,
          imageType: payload.params.imageType,
          ecommerceType: payload.params.ecommerceType
        }
      }
    } else {
      apiPayload = {
        type: 'image',
        prompt: payload.prompt,
        images: payload.images || [],
        model: payload.params.model,
        params: {
          aspectRatio: payload.params.aspectRatio,
          imageSize: payload.params.imageSize
        }
      }
    }

    const { task_id } = await generate(apiPayload.type, apiPayload)
    resultItem.id = task_id
    userStore.fetchUserInfo()
    startPoll(resultItem, task_id)
  } catch (e) {
    resultItem.status = 'failed'
    resultItem.error_msg = e.response?.data?.error || e.message || t('generate.failed')
  } finally {
    loading.value = false
    scrollToBottom(true)
  }
}

const timelineGroups = computed(() => {
  const g = genStore.groupedGenerations
  const result = []
  if (g.older.length) result.push({ label: t('generate.older'), items: [...g.older].reverse() })
  if (g.week.length) result.push({ label: t('generate.week'), items: [...g.week].reverse() })
  if (g.yesterday.length) result.push({ label: t('generate.yesterday'), items: [...g.yesterday].reverse() })
  if (g.today.length) result.push({ label: t('generate.today'), items: [...g.today].reverse() })
  return result
})

const hasAnyContent = computed(() => currentResults.value.length > 0 || genStore.generations.length > 0)

function startPoll(resultItem, taskId) {
  pollVideoTask(taskId, (update) => {
    if (update.status === 'success') {
      resultItem.status = 'success'
      resultItem.images = update.images || []
      resultItem.video_url = update.video_url
      resultItem.credits_cost = update.credits_spent
      userStore.fetchUserInfo()
      genStore.prependGeneration({ ...resultItem, created_at: Date.now() })
      const idx = currentResults.value.findIndex(r => r.id === resultItem.id)
      if (idx !== -1) currentResults.value.splice(idx, 1)
      scrollToBottom(true)
    } else if (update.status === 'failed') {
      resultItem.status = 'failed'
      resultItem.error_msg = update.error_msg
      userStore.fetchUserInfo()
    } else {
      resultItem.status = update.status
    }
  })
}

const handleSubmit = async (payload) => {
  loading.value = true
  const resultItem = reactive({
    id: Date.now(),
    type: payload.creativeMode,
    prompt: payload.prompt,
    status: payload.creativeMode === 'video' ? 'queued' : 'generating',
    images: [],
    video_url: null,
    credits_cost: payload.credits,
    params: payload.params,
    reference_images: payload.images || []
  })
  currentResults.value.push(resultItem)
  scrollToBottom()

  await executeGeneration(resultItem, payload)
}

const regenerate = (gen) => {
  composerRef.value?.fillFromGeneration(gen)
}

const editImage = (imageUrl) => {
  imageEditorSrc.value = imageUrl
  showImageEditor.value = true
}

const handleInpaintSubmit = async ({ originalImageUrl, maskBase64, prompt: editorPrompt, aspectRatio: aratio, imageSize: isize }) => {
  try {
    // Upload mask to OSS
    const maskOssUrl = await uploadImageToOSS(maskBase64)

    // Create result item
    const resultItem = reactive({
      id: Date.now(),
      type: 'image',
      prompt: editorPrompt,
      status: 'generating',
      images: [],
      video_url: null,
      credits_cost: 0,
      params: { model: 'gemini-3-pro-image-preview', mode: 'inpainting', aspectRatio: aratio, imageSize: isize },
      reference_images: [originalImageUrl]
    })
    currentResults.value.push(resultItem)
    showImageEditor.value = false
    scrollToBottom()

    loading.value = true
    const apiPayload = {
      type: 'image',
      prompt: editorPrompt,
      images: [originalImageUrl],
      mask: maskOssUrl,
      model: 'gemini-3-pro-image-preview',
      params: {
        aspectRatio: aratio || '1:1',
        imageSize: isize || '2K'
      }
    }

    const { task_id } = await generate(apiPayload.type, apiPayload)
    resultItem.id = task_id
    userStore.fetchUserInfo()
    startPoll(resultItem, task_id)
  } catch (e) {
    message.error(e.response?.data?.error || e.message || 'Inpainting failed')
  } finally {
    loading.value = false
    if (imageEditorRef.value) imageEditorRef.value.setGenerating(false)
    scrollToBottom(true)
  }
}

const openImageToSvg = (imageUrl) => {
  router.push({ name: 'image-to-svg', query: { src: imageUrl } })
}

const isShareLoading = (gen) => !!shareLoadingIds.value[gen?.id]

const openShareDialog = (gen) => {
  if (!gen?.id) return
  if (!userStore.requireAuth()) return
  generationIDToShare.value = gen.id
  generationToShare.value = gen
  showShareDialog.value = true
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
  shareLoadingIds.value[generationIDToShare.value] = true
  try {
    const post = await shareGeneration(generationIDToShare.value, { title, description, prompt, tags, cover_url })
    const sharedID = post?.share_id || ''
    const currentGen = currentResults.value.find(g => g.id === generationIDToShare.value)
    if (currentGen) {
      currentGen.is_shared = true
      currentGen.share_id = sharedID
    }
    const storedGen = genStore.generations.find(g => g.id === generationIDToShare.value)
    if (storedGen) {
      storedGen.is_shared = true
      storedGen.share_id = sharedID
    }
    showShareDialog.value = false
    const reviewStatus = (post?.review_status || '').toLowerCase()
    if (reviewStatus === 'approved' || reviewStatus === '') {
      message.success(t('inspiration.shareSuccessPublished'))
    } else {
      message.success(t('inspiration.shareSubmittedPending'))
    }
    if (post?.share_id && (reviewStatus === 'approved' || reviewStatus === '')) {
      message.info(t('inspiration.shareHintUnshare'))
    }
  } catch (e) {
    const errorMsg = e.response?.data?.error
    message.error(errorMsg || t('inspiration.shareFailed'))
  } finally {
    shareLoading.value = false
    delete shareLoadingIds.value[generationIDToShare.value]
    generationIDToShare.value = null
    generationToShare.value = null
  }
}

const toggleShareInspiration = async (gen) => {
  if (!gen?.id) return
  if (shareLoadingIds.value[gen.id]) return
  if (!userStore.requireAuth()) return
  const isShared = !!gen.is_shared

  if (isShared) {
    shareLoadingIds.value[gen.id] = true
    try {
      if (!gen.share_id) throw new Error('missing share id')
      await unshareInspiration(gen.share_id)
      gen.is_shared = false
      gen.share_id = ''
      message.success(t('inspiration.unshareSuccess'))
    } catch (e) {
      message.error(e.response?.data?.error || t('inspiration.unshareFailed'))
    } finally {
      delete shareLoadingIds.value[gen.id]
    }
  } else {
    openShareDialog(gen)
  }
}

const downloadAsset = (url, i, fallbackExt = 'png') => {
  if (!url) return
  let ext = fallbackExt
  try {
    const pathname = new URL(url, window.location.origin).pathname
    const match = pathname.match(/\.([a-z0-9]+)$/i)
    if (match?.[1]) ext = match[1].toLowerCase()
  } catch (_) {
    // ignore parse failures and use fallback extension
  }
  const a = document.createElement('a')
  a.href = url
  a.download = `nanobanana-${Date.now()}-${i}.${ext}`
  a.click()
}

const scrollToBottom = (smooth = false) => {
  nextTick(() => {
    if (resultsContainer.value) {
      resultsContainer.value.scrollTo({
        top: resultsContainer.value.scrollHeight,
        behavior: smooth ? 'smooth' : 'auto'
      })
    }
  })
}

const hasUserScrolled = ref(false)
let scrollTimeout = null

const handleScroll = (e) => {
  const el = e.target

  if (!scrollTimeout) {
    hasUserScrolled.value = true
    scrollTimeout = setTimeout(() => {
      scrollTimeout = null
    }, 150)
  }

  if (el.scrollTop < 50 && !genStore.loading && !isFirstLoad.value) {
    genStore.loadMore()
  }
}

const getStatusText = (s) => ({ queued: t('generate.queued'), running: t('generate.running'), generating: t('generate.generating') }[s] || s)
</script>

<template>
  <div class="generate-page">
    <div class="timeline-area" ref="resultsContainer" @scroll="handleScroll">
      <div v-if="!hasAnyContent" class="empty-state">
        <p>{{ $t('generate.noRecords') }}</p>
        <p class="empty-hint">{{ $t('generate.noRecordsHint') }}</p>
      </div>

      <template v-for="group in timelineGroups" :key="group.label">
        <div class="date-divider"><span>{{ group.label }}</span></div>
        <template v-for="gen in group.items" :key="gen.id">
          <div class="chat-row user-row">
            <div class="bubble user-bubble">
              <p class="bubble-text">{{ gen.prompt }}</p>
            </div>
          </div>
          <div class="chat-row ai-row">
            <div class="ai-avatar"><img src="/images/icon.png" alt="" class="ai-avatar-img" /></div>
            <div class="bubble ai-bubble">
              <div v-if="['generating','queued','running'].includes(gen.status)" class="generating-state">
                <NSpin size="small" /><span>{{ getStatusText(gen.status) }}</span>
              </div>
              <div v-else-if="gen.status === 'failed'" class="error-state">
                <span>❌</span><span>{{ gen.error_msg || $t('generate.failed') }}</span>
                <NButton size="tiny" quaternary @click="regenerate(gen)">{{ $t('generate.retry') }}</NButton>
              </div>
              <div v-else-if="gen.images?.length" class="result-images">
                <NImageGroup>
                  <div class="image-grid" :class="{ single: gen.images.length === 1 }">
                    <div v-for="(img, i) in gen.images" :key="i" class="image-item">
                      <NImage :src="img" object-fit="contain" lazy :preview-src="img" />
                    </div>
                  </div>
                </NImageGroup>
                <div class="result-actions">
                  <button class="result-action-btn" type="button" :title="t('generate.download')" @click="downloadAsset(gen.images[0], 0, 'png')">
                    <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M12 4v10" />
                      <path d="M8 10l4 4 4-4" />
                      <path d="M5 19h14" />
                    </svg>
                  </button>
                  <button class="result-action-btn" type="button" :title="t('generate.regenerate')" @click="regenerate(gen)">
                    <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M20 8a8 8 0 0 0-14-3" />
                      <path d="M6 5H3V2" />
                      <path d="M4 16a8 8 0 0 0 14 3" />
                      <path d="M18 19h3v3" />
                    </svg>
                  </button>
                  <button class="result-action-btn" type="button" :title="t('generate.editFromThis')" @click="editImage(gen.images[0])">
                    <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M4 20l4.5-1 10-10a2.1 2.1 0 0 0-3-3l-10 10L4 20z" />
                      <path d="M14.5 5.5l4 4" />
                    </svg>
                  </button>
                  <button
                    class="result-action-btn"
                    :class="{ shared: gen.is_shared, loading: isShareLoading(gen) }"
                    type="button"
                    :title="gen.is_shared ? t('common.unshare') : t('common.share')"
                    :disabled="isShareLoading(gen)"
                    @click="toggleShareInspiration(gen)"
                  >
                    <svg v-if="gen.is_shared" class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M5 12l4 4L19 6" />
                    </svg>
                    <svg v-else class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M12 5v10" />
                      <path d="M8 9l4-4 4 4" />
                      <path d="M5 14v4a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-4" />
                    </svg>
                  </button>
                  <NPopover trigger="click" placement="top" :show-arrow="false">
                    <template #trigger>
                      <button class="result-action-btn" type="button" :title="t('generate.toolbox')">
                        <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                          <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
                        </svg>
                      </button>
                    </template>
                    <div class="toolbox-menu">
                      <button class="toolbox-menu-item" @click="openImageToSvg(gen.images[0])">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
                        {{ t('generate.toSvg') }}
                      </button>
                    </div>
                  </NPopover>
                </div>
              </div>
              <div v-else-if="gen.video_url" class="video-result">
                <video controls :src="gen.video_url" class="result-video" preload="metadata" />
                <div class="result-actions">
                  <button class="result-action-btn" type="button" :title="t('generate.download')" @click="downloadAsset(gen.video_url, 0, 'mp4')">
                    <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M12 4v10" />
                      <path d="M8 10l4 4 4-4" />
                      <path d="M5 19h14" />
                    </svg>
                  </button>
                  <button class="result-action-btn" type="button" :title="t('generate.regenerate')" @click="regenerate(gen)">
                    <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M20 8a8 8 0 0 0-14-3" />
                      <path d="M6 5H3V2" />
                      <path d="M4 16a8 8 0 0 0 14 3" />
                      <path d="M18 19h3v3" />
                    </svg>
                  </button>
                  <button
                    class="result-action-btn"
                    :class="{ shared: gen.is_shared, loading: isShareLoading(gen) }"
                    type="button"
                    :title="gen.is_shared ? t('common.unshare') : t('common.share')"
                    :disabled="isShareLoading(gen)"
                    @click="toggleShareInspiration(gen)"
                  >
                    <svg v-if="gen.is_shared" class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M5 12l4 4L19 6" />
                    </svg>
                    <svg v-else class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M12 5v10" />
                      <path d="M8 9l4-4 4 4" />
                      <path d="M5 14v4a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-4" />
                    </svg>
                  </button>
                </div>
              </div>
              <div v-if="gen.credits_cost" class="credits-used">{{ $t('generate.creditsUsed', { credits: gen.credits_cost }) }}</div>
            </div>
          </div>
        </template>
      </template>

      <template v-if="currentResults.length">
        <div class="date-divider"><span>{{ $t('generate.currentSession') }}</span></div>
        <template v-for="result in currentResults" :key="result.id">
          <div class="chat-row user-row">
            <div class="bubble user-bubble">
              <span class="bubble-tag">{{ result.type === 'video' ? $t('generate.video') : result.type === 'ecommerce' ? $t('generate.ecommerce') : $t('generate.image') }}</span>
              <p class="bubble-text">{{ result.prompt }}</p>
            </div>
          </div>
          <div class="chat-row ai-row">
            <div class="ai-avatar"><img src="/images/icon.png" alt="" class="ai-avatar-img" /></div>
            <div class="bubble ai-bubble">
              <div v-if="['generating','queued','running'].includes(result.status)" class="generating-state">
                <NSpin size="small" /><span>{{ getStatusText(result.status) }}</span>
              </div>
              <div v-else-if="result.status === 'failed'" class="error-state">
                <span>❌</span><span>{{ result.error_msg || $t('generate.failed') }}</span>
                <NButton size="tiny" quaternary @click="regenerate(result)">{{ $t('generate.retry') }}</NButton>
              </div>
              <div v-else-if="result.video_url" class="video-result">
                <video controls :src="result.video_url" class="result-video" preload="metadata" />
                <div class="result-actions">
                  <button class="result-action-btn" type="button" :title="t('generate.download')" @click="downloadAsset(result.video_url, 0, 'mp4')">
                    <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M12 4v10" />
                      <path d="M8 10l4 4 4-4" />
                      <path d="M5 19h14" />
                    </svg>
                  </button>
                  <button class="result-action-btn" type="button" :title="t('generate.regenerate')" @click="regenerate(result)">
                    <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M20 8a8 8 0 0 0-14-3" />
                      <path d="M6 5H3V2" />
                      <path d="M4 16a8 8 0 0 0 14 3" />
                      <path d="M18 19h3v3" />
                    </svg>
                  </button>
                  <button
                    class="result-action-btn"
                    :class="{ shared: result.is_shared, loading: isShareLoading(result) }"
                    type="button"
                    :title="result.is_shared ? t('common.unshare') : t('common.share')"
                    :disabled="isShareLoading(result)"
                    @click="toggleShareInspiration(result)"
                  >
                    <svg v-if="result.is_shared" class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M5 12l4 4L19 6" />
                    </svg>
                    <svg v-else class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M12 5v10" />
                      <path d="M8 9l4-4 4 4" />
                      <path d="M5 14v4a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-4" />
                    </svg>
                  </button>
                </div>
                <div v-if="result.credits_cost" class="credits-used">{{ $t('generate.creditsUsed', { credits: result.credits_cost }) }}</div>
              </div>
              <div v-else-if="result.images?.length" class="result-images">
                <NImageGroup>
                  <div class="image-grid" :class="{ single: result.images.length === 1 }">
                    <div v-for="(img, i) in result.images" :key="i" class="image-item">
                      <NImage :src="img" object-fit="contain" lazy :preview-src="img" />
                    </div>
                  </div>
                </NImageGroup>
                <div class="result-actions">
                  <button class="result-action-btn" type="button" :title="t('generate.download')" @click="downloadAsset(result.images[0], 0, 'png')">
                    <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M12 4v10" />
                      <path d="M8 10l4 4 4-4" />
                      <path d="M5 19h14" />
                    </svg>
                  </button>
                  <button class="result-action-btn" type="button" :title="t('generate.regenerate')" @click="regenerate(result)">
                    <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M20 8a8 8 0 0 0-14-3" />
                      <path d="M6 5H3V2" />
                      <path d="M4 16a8 8 0 0 0 14 3" />
                      <path d="M18 19h3v3" />
                    </svg>
                  </button>
                  <button class="result-action-btn" type="button" :title="t('generate.editFromThis')" @click="editImage(result.images[0])">
                    <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M4 20l4.5-1 10-10a2.1 2.1 0 0 0-3-3l-10 10L4 20z" />
                      <path d="M14.5 5.5l4 4" />
                    </svg>
                  </button>
                  <button
                    class="result-action-btn"
                    :class="{ shared: result.is_shared, loading: isShareLoading(result) }"
                    type="button"
                    :title="result.is_shared ? t('common.unshare') : t('common.share')"
                    :disabled="isShareLoading(result)"
                    @click="toggleShareInspiration(result)"
                  >
                    <svg v-if="result.is_shared" class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M5 12l4 4L19 6" />
                    </svg>
                    <svg v-else class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M12 5v10" />
                      <path d="M8 9l4-4 4 4" />
                      <path d="M5 14v4a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-4" />
                    </svg>
                  </button>
                  <NPopover trigger="click" placement="top" :show-arrow="false">
                    <template #trigger>
                      <button class="result-action-btn" type="button" :title="t('generate.toolbox')">
                        <svg class="result-action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                          <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
                        </svg>
                      </button>
                    </template>
                    <div class="toolbox-menu">
                      <button class="toolbox-menu-item" @click="openImageToSvg(result.images[0])">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
                        {{ t('generate.toSvg') }}
                      </button>
                    </div>
                  </NPopover>
                </div>
                <div v-if="result.credits_cost" class="credits-used">{{ $t('generate.creditsUsed', { credits: result.credits_cost }) }}</div>
              </div>
            </div>
          </div>
        </template>
      </template>
    </div>

    <ComposerBar
      ref="composerRef"
      v-model:creative-mode="creativeMode"
      :loading="loading"
      @submit="handleSubmit"
    />

    <ShareGenerationDialog
      v-model:show="showShareDialog"
      :loading="shareLoading"
      mode="generation"
      :initial-data="shareDialogInitialData"
      @confirm="handleShareConfirm"
    />

    <ImageEditor
      ref="imageEditorRef"
      v-model:show="showImageEditor"
      :image-src="imageEditorSrc"
      @submit="handleInpaintSubmit"
    />
  </div>
</template>

<style scoped>
.generate-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.timeline-area {
  flex: 1;
  overflow-y: auto;
  padding: 24px 24px 16px;
}

.date-divider {
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 20px 0 16px;
}
.date-divider span {
  font-size: 12px;
  color: var(--color-text-muted);
  background: var(--color-tint-white-04);
  padding: 3px 14px;
  border-radius: 10px;
  white-space: nowrap;
}

.chat-row {
  display: flex;
  max-width: 1100px;
  margin: 0 auto 16px;
  animation: fadeIn .3s ease;
}

.user-row {
  justify-content: flex-end;
}

.ai-row {
  justify-content: flex-start;
  align-items: flex-start;
  gap: 10px;
}

.ai-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: rgba(0, 202, 224, 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-top: 2px;
  overflow: hidden;
}
.ai-avatar-img {
  width: 20px;
  height: 20px;
  object-fit: contain;
}

.bubble {
  padding: 0;
  animation: fadeIn .25s ease;
}

.user-bubble {
  max-width: 70%;
  color: var(--color-text-primary);
  font-size: 15px;
  line-height: 1.6;
  text-align: right;
}
.bubble-text {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
}

.ai-bubble {
  flex: 1;
  min-width: 0;
}

.generating-state {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 4px;
  color: var(--color-text-secondary);
  font-size: 14px;
}
.error-state {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  background: rgba(239,68,68,.1);
  border: 1px solid rgba(239,68,68,.2);
  border-radius: 12px;
  color: var(--color-error-light);
  font-size: 14px;
}

.result-images { margin-top: 4px; }
.image-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 6px; }
.image-grid.single { grid-template-columns: 1fr; max-width: 200px; }
.image-item { border-radius: 10px; overflow: hidden; }
.image-item :deep(.n-image) { width: 100%; }
.image-item :deep(.n-image img) { width: 100%; height: auto; border-radius: 10px; object-fit: cover; }
.result-actions {
  display: flex;
  gap: 6px;
  margin-top: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.result-action-btn {
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
  transition: all .2s;
}
.result-action-btn:hover {
  border-color: rgba(0, 202, 224, 0.45);
  background: rgba(0, 202, 224, 0.2);
}
.result-action-btn.shared {
  color: #8cefff;
  border-color: rgba(0, 202, 224, 0.45);
  background: rgba(0, 202, 224, 0.18);
}
.result-action-btn.loading,
.result-action-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.result-action-icon {
  width: 15px;
  height: 15px;
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.video-result { margin-top: 4px; }
.result-video { max-width: 240px; max-height: 180px; border-radius: 10px; background: #000; }
.credits-used { font-size: 12px; color: var(--color-text-muted); margin-top: 6px; }

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: var(--color-text-muted);
  font-size: 16px;
}
.empty-hint {
  font-size: 13px;
  margin-top: 8px;
  color: var(--color-text-muted);
  opacity: 0.7;
}

.generate-page :deep(.composer-wrap) {
  max-width: 900px;
}

@media (max-width: 768px) {
  .timeline-area { padding: 12px 12px 10px; }
  .chat-row { padding: 0; }
  .user-bubble { max-width: 85%; font-size: 14px; }
  .ai-bubble { flex: 1; }
  .ai-avatar { width: 24px; height: 24px; }
  .ai-avatar-img { width: 16px; height: 16px; }
  .image-grid { grid-template-columns: repeat(2, 1fr); gap: 6px; }
  .image-grid.single { grid-template-columns: 1fr; max-width: 200px; }
  .result-video { max-width: 100%; max-height: 240px; }
  .result-actions { gap: 6px; }
  .generate-page :deep(.composer-wrap) { max-width: 100%; }
  .empty-state { min-height: 200px; font-size: 14px; }
  .empty-hint { font-size: 12px; }
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

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
