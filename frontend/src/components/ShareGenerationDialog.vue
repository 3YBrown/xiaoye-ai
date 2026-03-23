<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { useGenerate } from '../composables/useGenerate'
import { useInspiration } from '../composables/useInspiration'

const { t } = useI18n()
const message = useMessage()
const { uploadImageToOSS } = useGenerate()
const { uploadVideo, listInspirationTags } = useInspiration()

const props = defineProps({
  show: {
    type: Boolean,
    default: false
  },
  loading: {
    type: Boolean,
    default: false
  },
  mode: {
    type: String,
    default: 'generation'
  },
  initialData: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:show', 'confirm', 'cancel'])

const title = ref('')
const description = ref('')
const prompt = ref('')
const tags = ref([])
const tagInput = ref('')
const showTagDropdown = ref(false)
const images = ref([])
const videoUrl = ref('')
const coverUrl = ref('')
const postType = ref('image')
const titleInputRef = ref(null)
const uploadInputRef = ref(null)
const coverUploadInputRef = ref(null)
const uploadLoading = ref(false)
const coverLoading = ref(false)
const uploadDragOver = ref(false)
const tagSuggestions = ref([])
let tagBlurTimer = null

const isUploadMode = computed(() => props.mode === 'upload')
const hasUploadMedia = computed(() => postType.value === 'video' ? !!videoUrl.value : images.value.length > 0)
const hasPreviewMedia = computed(() => !!videoUrl.value || images.value.length > 0 || !!coverUrl.value)
const previewImageList = computed(() => {
  if (images.value.length > 0) return images.value
  if (coverUrl.value && postType.value !== 'video') return [coverUrl.value]
  return []
})
const uploadAccept = computed(() => postType.value === 'video'
  ? '.mp4,.mov,.webm,.m4v,video/mp4,video/webm,video/quicktime'
  : 'image/*'
)
const uploadMultiple = computed(() => postType.value !== 'video')
const uploadButtonText = computed(() => postType.value === 'video' ? t('publishDialog.uploadVideo') : t('publishDialog.uploadImage'))
const uploadHintText = computed(() => postType.value === 'video' ? t('publishDialog.dropHintVideo') : t('publishDialog.dropHintImage'))
const uploaderDisabled = computed(() => uploadLoading.value || props.loading)
const coverActionsDisabled = computed(() => uploadLoading.value || coverLoading.value || props.loading)
const hasRequiredVideoCover = computed(() => postType.value !== 'video' || !!coverUrl.value)
const confirmDisabled = computed(() => {
  if (!title.value.trim() || !prompt.value.trim()) return true
  if (props.loading || uploadLoading.value || coverLoading.value) return true
  if (isUploadMode.value && !hasUploadMedia.value) return true
  if (!hasRequiredVideoCover.value) return true
  return false
})

const resetForm = () => {
  title.value = ''
  description.value = ''
  prompt.value = ''
  tags.value = []
  tagInput.value = ''
  showTagDropdown.value = false
  images.value = []
  videoUrl.value = ''
  coverUrl.value = ''
  postType.value = 'image'
  uploadDragOver.value = false
}

const hydrateFromInitialData = () => {
  const data = props.initialData || {}
  title.value = data.title || ''
  description.value = data.description || ''
  prompt.value = data.prompt || ''
  tags.value = Array.isArray(data.tags) ? [...new Set(data.tags.filter(Boolean))].slice(0, 5) : []
  images.value = Array.isArray(data.images) ? data.images.filter(Boolean) : []
  videoUrl.value = data.video_url || ''
  coverUrl.value = data.cover_url || ''
  if (data.type === 'video' || data.video_url) {
    postType.value = 'video'
  } else {
    postType.value = 'image'
  }
}

const normalizeTag = (value) => {
  if (!value) return ''
  return value.trim().replace(/\s+/g, ' ')
}

const addTag = (value = tagInput.value) => {
  const tag = normalizeTag(value)
  if (!tag) return
  if (tag.length > 24) {
    message.warning(t('publishDialog.tagTooLong'))
    return
  }
  if (tags.value.includes(tag)) {
    tagInput.value = ''
    return
  }
  if (tags.value.length >= 5) {
    message.warning(t('publishDialog.tagTooMany'))
    return
  }
  tags.value.push(tag)
  tagInput.value = ''
}

const filteredTagSuggestions = computed(() => {
  const keyword = normalizeTag(tagInput.value).toLowerCase()
  return tagSuggestions.value
    .filter(item => !!item)
    .filter(item => !tags.value.includes(item))
    .filter(item => !keyword || item.toLowerCase().includes(keyword))
    .slice(0, 8)
})

const clearTagBlurTimer = () => {
  if (!tagBlurTimer) return
  clearTimeout(tagBlurTimer)
  tagBlurTimer = null
}

const handleTagInputFocus = () => {
  clearTagBlurTimer()
  showTagDropdown.value = true
}

const handleTagInputBlur = () => {
  clearTagBlurTimer()
  tagBlurTimer = setTimeout(() => {
    showTagDropdown.value = false
  }, 120)
}

const selectTagSuggestion = (tag) => {
  addTag(tag)
  showTagDropdown.value = false
}

const commitPendingTagInput = () => {
  const pendingTag = normalizeTag(tagInput.value)
  if (!pendingTag) return true
  if (pendingTag.length > 24) {
    message.warning(t('publishDialog.tagTooLong'))
    return false
  }
  if (tags.value.includes(pendingTag)) {
    tagInput.value = ''
    return true
  }
  if (tags.value.length >= 5) {
    message.warning(t('publishDialog.tagTooMany'))
    return false
  }
  tags.value.push(pendingTag)
  tagInput.value = ''
  return true
}

const removeTag = (tag) => {
  tags.value = tags.value.filter(item => item !== tag)
}

const readFileAsDataUrl = (file) => new Promise((resolve, reject) => {
  const reader = new FileReader()
  reader.onload = () => resolve(String(reader.result || ''))
  reader.onerror = () => reject(new Error('read file failed'))
  reader.readAsDataURL(file)
})

const extractVideoFrameDataUrl = (videoSource, seekSeconds = 0) => new Promise((resolve, reject) => {
  const video = document.createElement('video')
  let done = false

  const cleanup = () => {
    video.pause()
    video.removeAttribute('src')
    video.load()
  }

  const finish = (error, value = '') => {
    if (done) return
    done = true
    cleanup()
    if (error) reject(error)
    else resolve(value)
  }

  const timeout = setTimeout(() => {
    finish(new Error('extract frame timeout'))
  }, 12000)

  const safeFinish = (error, value = '') => {
    clearTimeout(timeout)
    finish(error, value)
  }

  const capture = () => {
    if (!video.videoWidth || !video.videoHeight) {
      safeFinish(new Error('invalid video dimensions'))
      return
    }
    const canvas = document.createElement('canvas')
    canvas.width = video.videoWidth
    canvas.height = video.videoHeight
    const context = canvas.getContext('2d')
    if (!context) {
      safeFinish(new Error('canvas context unavailable'))
      return
    }
    try {
      context.drawImage(video, 0, 0, canvas.width, canvas.height)
      const dataUrl = canvas.toDataURL('image/jpeg', 0.92)
      if (!dataUrl) {
        safeFinish(new Error('failed to export frame'))
        return
      }
      safeFinish(null, dataUrl)
    } catch (error) {
      safeFinish(error)
    }
  }

  const seekAndCapture = () => {
    const target = Math.max(0, Number(seekSeconds) || 0)
    if (target <= 0) {
      capture()
      return
    }
    const onSeeked = () => {
      video.removeEventListener('seeked', onSeeked)
      capture()
    }
    video.addEventListener('seeked', onSeeked)
    try {
      const maxTime = Math.max(0, (video.duration || target) - 0.05)
      video.currentTime = Math.min(target, maxTime)
    } catch (_) {
      video.removeEventListener('seeked', onSeeked)
      capture()
    }
  }

  video.addEventListener('loadeddata', seekAndCapture, { once: true })
  video.addEventListener('error', () => safeFinish(new Error('video load failed')), { once: true })
  video.muted = true
  video.preload = 'auto'
  video.playsInline = true
  if (typeof videoSource === 'string' && /^https?:\/\//i.test(videoSource)) {
    video.crossOrigin = 'anonymous'
  }
  video.src = videoSource
  video.load()
})

const isImageFile = (file) => !!file?.type?.startsWith('image/')
const isVideoFile = (file) => !!file?.type?.startsWith('video/')

const uploadCoverFromDataUrl = async (dataUrl) => {
  const uploaded = await uploadImageToOSS(dataUrl)
  if (!uploaded) throw new Error('cover upload failed')
  coverUrl.value = uploaded
  return uploaded
}

const generateVideoCover = async (videoSource, { force = false, silent = false } = {}) => {
  if (!videoSource) return false
  if (coverLoading.value) return false
  if (coverUrl.value && !force) return true

  coverLoading.value = true
  try {
    const frameDataUrl = await extractVideoFrameDataUrl(videoSource, 0)
    await uploadCoverFromDataUrl(frameDataUrl)
    return true
  } catch (_) {
    if (!silent) {
      message.warning(t('publishDialog.autoCoverFailed'))
    }
    return false
  } finally {
    coverLoading.value = false
  }
}

const openCoverPicker = () => {
  if (coverActionsDisabled.value) return
  coverUploadInputRef.value?.click()
}

const handleCoverSelect = async (event) => {
  const file = event.target?.files?.[0]
  try {
    if (!file) return
    if (!isImageFile(file)) {
      message.warning(t('publishDialog.mediaTypeMismatchImage'))
      return
    }
    coverLoading.value = true
    const dataUrl = await readFileAsDataUrl(file)
    await uploadCoverFromDataUrl(dataUrl)
  } catch (_) {
    message.error(t('publishDialog.uploadCoverFailed'))
  } finally {
    coverLoading.value = false
    if (event?.target) event.target.value = ''
  }
}

const clearCover = () => {
  coverUrl.value = ''
}

const regenerateCoverFromVideo = async () => {
  if (!videoUrl.value) {
    message.warning(t('publishDialog.videoRequiredBeforeCover'))
    return
  }
  await generateVideoCover(videoUrl.value, { force: true })
}

const handleImageSelect = async (fileList) => {
  if (postType.value !== 'image') {
    message.warning(t('publishDialog.mediaTypeMismatchImage'))
    return
  }
  if (!fileList?.length) return
  const files = Array.from(fileList)
  if (!files.every(isImageFile)) {
    message.warning(t('publishDialog.mediaTypeMismatchImage'))
    return
  }
  uploadLoading.value = true
  try {
    for (const file of files) {
      const dataUrl = await readFileAsDataUrl(file)
      const url = await uploadImageToOSS(dataUrl)
      if (url) images.value.push(url)
    }
    if (!coverUrl.value && images.value.length > 0) {
      coverUrl.value = images.value[0]
    }
  } catch (_) {
    message.error(t('publishDialog.uploadImageFailed'))
  } finally {
    uploadLoading.value = false
  }
}

const handleVideoSelect = async (fileList) => {
  if (postType.value !== 'video') {
    message.warning(t('publishDialog.mediaTypeMismatchVideo'))
    return
  }
  const file = fileList?.[0]
  if (!file) return
  if (!isVideoFile(file)) {
    message.warning(t('publishDialog.mediaTypeMismatchVideo'))
    return
  }
  uploadLoading.value = true
  try {
    const url = await uploadVideo(file)
    if (url) {
      videoUrl.value = url
      postType.value = 'video'
      const objectUrl = URL.createObjectURL(file)
      try {
        const generated = await generateVideoCover(objectUrl, { force: !coverUrl.value, silent: true })
        if (!generated && !coverUrl.value) {
          message.warning(t('publishDialog.autoCoverFailed'))
        }
      } finally {
        URL.revokeObjectURL(objectUrl)
      }
    }
  } catch (_) {
    message.error(t('publishDialog.uploadVideoFailed'))
  } finally {
    uploadLoading.value = false
  }
}

const handleMediaTypeChange = (type) => {
  if (type === postType.value) return
  postType.value = type
  uploadDragOver.value = false
  if (type === 'image') {
    clearVideo()
    clearCover()
  } else {
    images.value = []
    coverUrl.value = ''
  }
}

const processSelectedFiles = async (selectedFiles) => {
  if (!selectedFiles.length) return
  if (postType.value === 'video') {
    await handleVideoSelect(selectedFiles.slice(0, 1))
    return
  }
  await handleImageSelect(selectedFiles)
}

const openMediaPicker = () => {
  if (uploaderDisabled.value) return
  uploadInputRef.value?.click()
}

const handleDropZoneDragOver = () => {
  if (uploaderDisabled.value) return
  uploadDragOver.value = true
}

const handleDropZoneDragLeave = () => {
  uploadDragOver.value = false
}

const handleDropZoneDrop = async (event) => {
  uploadDragOver.value = false
  if (uploaderDisabled.value) return
  const selectedFiles = Array.from(event.dataTransfer?.files || [])
  await processSelectedFiles(selectedFiles)
}

const handleMediaSelect = async (event) => {
  const selectedFiles = Array.from(event.target?.files || [])
  try {
    await processSelectedFiles(selectedFiles)
  } finally {
    if (event?.target) event.target.value = ''
  }
}

const removeImage = (url) => {
  images.value = images.value.filter(item => item !== url)
  if (coverUrl.value === url) {
    coverUrl.value = images.value[0] || ''
  }
}

const clearVideo = () => {
  videoUrl.value = ''
  clearCover()
  if (postType.value === 'video' && images.value.length > 0) {
    postType.value = 'image'
  }
}

const handleConfirm = () => {
  if (!commitPendingTagInput()) return
  const trimmedTitle = title.value.trim()
  const trimmedPrompt = prompt.value.trim()
  if (!trimmedTitle || !trimmedPrompt) return
  if (isUploadMode.value && !hasUploadMedia.value) {
    message.warning(t('publishDialog.mediaRequired'))
    return
  }
  if (postType.value === 'video' && !coverUrl.value) {
    message.warning(t('publishDialog.videoCoverRequired'))
    return
  }

  emit('confirm', {
    title: trimmedTitle,
    description: description.value.trim(),
    prompt: trimmedPrompt,
    tags: tags.value,
    images: postType.value === 'image' ? images.value : [],
    video_url: postType.value === 'video' ? videoUrl.value : '',
    cover_url: postType.value === 'video' ? coverUrl.value : (coverUrl.value || images.value[0] || ''),
    type: postType.value
  })
}

const handleCancel = () => {
  clearTagBlurTimer()
  emit('update:show', false)
  emit('cancel')
  resetForm()
}

const handleClose = () => {
  clearTagBlurTimer()
  emit('update:show', false)
  resetForm()
}

const loadTagSuggestions = async () => {
  try {
    const data = await listInspirationTags({ limit: 30 })
    tagSuggestions.value = (data?.items || []).map(item => item.name).filter(Boolean)
  } catch (_) {
    tagSuggestions.value = []
  }
}

watch(() => props.show, async (val) => {
  if (!val) {
    clearTagBlurTimer()
    return
  }
  clearTagBlurTimer()
  resetForm()
  hydrateFromInitialData()
  await loadTagSuggestions()
  nextTick(() => {
    titleInputRef.value?.focus()
  })
})
</script>

<template>
  <Transition name="fade">
    <div v-if="show" class="share-dialog-overlay" @click="handleClose">
      <div class="share-dialog" @click.stop>
        <div class="share-dialog-header">
          <h3 class="share-dialog-title">{{ mode === 'upload' ? t('publishDialog.titleUpload') : t('shareDialog.title') }}</h3>
          <button class="share-dialog-close" @click="handleClose">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 6L6 18M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="share-dialog-body">
          <div class="form-field">
            <label class="form-label">{{ t('shareDialog.titleLabel') }}<span class="form-required">*</span></label>
            <div class="field-with-counter field-with-counter-input">
              <input
                ref="titleInputRef"
                v-model="title"
                type="text"
                class="form-input"
                :placeholder="t('shareDialog.titlePlaceholder')"
                maxlength="200"
              />
              <span class="field-counter">{{ title.length }}/200</span>
            </div>
          </div>

          <div class="form-field">
            <label class="form-label">{{ t('publishDialog.promptLabel') }}<span class="form-required">*</span></label>
            <textarea
              v-model="prompt"
              class="form-textarea"
              :placeholder="t('publishDialog.promptPlaceholder')"
              rows="4"
            />
          </div>

          <div v-if="!isUploadMode && hasPreviewMedia" class="form-field">
            <label class="form-label">{{ t('publishDialog.uploadMediaLabel') }}</label>
            <div v-if="postType === 'video' && videoUrl" class="share-preview-video">
              <video :src="videoUrl" controls preload="metadata" playsinline />
            </div>
            <div v-else-if="previewImageList.length" class="preview-image-grid">
              <div v-for="url in previewImageList" :key="url" class="preview-image-item">
                <img :src="url" alt="preview" />
              </div>
            </div>
          </div>

          <div v-if="isUploadMode" class="form-field">
            <label class="form-label">{{ t('publishDialog.uploadMediaLabel') }}<span class="form-required">*</span></label>
            <div class="media-type-switch">
              <button
                class="type-btn"
                :class="{ active: postType === 'image' }"
                type="button"
                :disabled="uploadLoading || loading"
                @click="handleMediaTypeChange('image')"
              >
                {{ t('publishDialog.uploadImage') }}
              </button>
              <button
                class="type-btn"
                :class="{ active: postType === 'video' }"
                type="button"
                :disabled="uploadLoading || loading"
                @click="handleMediaTypeChange('video')"
              >
                {{ t('publishDialog.uploadVideo') }}
              </button>
            </div>
            <div class="upload-actions">
              <div
                class="upload-dropzone"
                :class="{ active: uploadDragOver, disabled: uploaderDisabled }"
                @dragover.prevent="handleDropZoneDragOver"
                @dragleave.prevent="handleDropZoneDragLeave"
                @drop.prevent="handleDropZoneDrop"
                @click="openMediaPicker"
              >
                <input
                  ref="uploadInputRef"
                  class="hidden-upload-input"
                  type="file"
                  :accept="uploadAccept"
                  :multiple="uploadMultiple"
                  :disabled="uploaderDisabled"
                  @change="handleMediaSelect"
                />
                <div class="dropzone-title">{{ t('publishDialog.dropOrClick') }}</div>
                <div class="dropzone-hint">{{ uploadHintText }}</div>
                <button class="btn btn-secondary tiny single-upload-btn" type="button" :disabled="uploaderDisabled" @click.stop="openMediaPicker">
                  {{ uploadButtonText }}
                </button>
              </div>
            </div>
            <div v-if="uploadLoading" class="upload-status">{{ t('publishDialog.uploading') }}</div>

            <div v-if="postType === 'image' && images.length" class="image-preview-list">
              <div v-for="url in images" :key="url" class="image-preview-item">
                <img :src="url" alt="preview" />
                <button type="button" class="remove-btn" @click="removeImage(url)">{{ t('publishDialog.delete') }}</button>
              </div>
            </div>

            <div v-if="postType === 'video' && videoUrl" class="video-preview-row">
              <span class="video-url">{{ videoUrl }}</span>
              <button type="button" class="remove-btn" @click="clearVideo">{{ t('publishDialog.removeVideo') }}</button>
            </div>
          </div>

          <div v-if="postType === 'video'" class="form-field">
            <label class="form-label">{{ t('publishDialog.coverLabel') }}<span class="form-required">*</span></label>
            <input
              ref="coverUploadInputRef"
              class="hidden-upload-input"
              type="file"
              accept="image/*"
              :disabled="coverActionsDisabled"
              @change="handleCoverSelect"
            />

            <div class="cover-editor">
              <div
                class="cover-preview"
                :class="{ clickable: !coverActionsDisabled }"
                @click="openCoverPicker"
              >
                <img v-if="coverUrl" :src="coverUrl" alt="video cover" />
                <div v-else class="cover-placeholder">{{ t('publishDialog.coverRequiredHint') }}</div>
              </div>

              <div class="cover-actions">
                <button class="btn btn-secondary tiny" type="button" :disabled="coverActionsDisabled || !videoUrl" @click="regenerateCoverFromVideo">
                  {{ coverLoading ? t('publishDialog.coverGenerating') : t('publishDialog.autoExtractCover') }}
                </button>
              </div>
            </div>
          </div>

          <div class="form-field">
            <label class="form-label">{{ t('shareDialog.descriptionLabel') }}</label>
            <div class="field-with-counter field-with-counter-textarea">
              <textarea
                v-model="description"
                class="form-textarea"
                :placeholder="t('shareDialog.descriptionPlaceholder')"
                maxlength="1000"
                rows="3"
              />
              <span class="field-counter">{{ description.length }}/1000</span>
            </div>
          </div>

          <div class="form-field">
            <label class="form-label">{{ t('publishDialog.tagLabel') }}</label>
            <div class="tag-input-wrap">
              <div class="tag-input-row">
                <input
                  v-model="tagInput"
                  type="text"
                  class="form-input"
                  :placeholder="t('publishDialog.tagPlaceholder')"
                  @focus="handleTagInputFocus"
                  @blur="handleTagInputBlur"
                  @keyup.enter.prevent="addTag(); showTagDropdown = false"
                  @keydown.down.prevent="showTagDropdown = true"
                />
                <button class="btn btn-secondary tiny" type="button" @click="addTag(); showTagDropdown = false">{{ t('publishDialog.addTag') }}</button>
              </div>
              <div v-if="showTagDropdown && filteredTagSuggestions.length" class="tag-suggestion-panel">
                <button
                  v-for="item in filteredTagSuggestions"
                  :key="item"
                  class="tag-suggestion-item"
                  type="button"
                  @mousedown.prevent
                  @click="selectTagSuggestion(item)"
                >
                  #{{ item }}
                </button>
              </div>
            </div>
            <div v-if="tags.length" class="tag-list">
              <button v-for="tag in tags" :key="tag" type="button" class="tag-chip" @click="removeTag(tag)">
                {{ tag }} x
              </button>
            </div>
          </div>
        </div>

        <div class="share-dialog-footer">
          <button class="btn btn-secondary" @click="handleCancel" :disabled="loading || uploadLoading">{{ t('common.cancel') }}</button>
          <button
            class="btn btn-primary"
            :disabled="confirmDisabled"
            @click="handleConfirm"
          >
            <span v-if="loading" class="btn-loading">{{ t('publishDialog.publishing') }}</span>
            <span v-else>{{ t('shareDialog.publish') }}</span>
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.share-dialog-overlay { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.6); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 20px; }
.share-dialog { width: 100%; max-width: 620px; max-height: calc(100vh - 40px); overflow: auto; background: var(--color-bg-elevated, #1a1d24); border: 1px solid var(--color-tint-white-08, rgba(255, 255, 255, 0.08)); border-radius: 16px; box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4); }
.share-dialog-header { display: flex; align-items: center; justify-content: space-between; padding: 20px 24px 16px; border-bottom: 1px solid var(--color-tint-white-06, rgba(255, 255, 255, 0.06)); }
.share-dialog-title { font-size: 18px; font-weight: 600; color: var(--color-text-primary); margin: 0; }
.share-dialog-close { width: 32px; height: 32px; display: flex; align-items: center; justify-content: center; background: transparent; border: none; border-radius: 8px; color: var(--color-text-secondary); cursor: pointer; }
.share-dialog-body { padding: 20px 24px; display: flex; flex-direction: column; gap: 16px; }
.form-field { display: flex; flex-direction: column; gap: 8px; }
.form-label { font-size: 14px; font-weight: 500; color: var(--color-text-secondary); }
.form-required { color: #ff6b6b; margin-left: 4px; }
.form-input, .form-textarea { padding: 12px 16px; background: var(--color-tint-white-03, rgba(255, 255, 255, 0.03)); border: 1px solid var(--color-tint-white-08, rgba(255, 255, 255, 0.08)); border-radius: 10px; color: var(--color-text-primary); font-size: 14px; font-family: inherit; width: 100%; box-sizing: border-box; }
.form-textarea { resize: vertical; min-height: 80px; }
.field-with-counter {
  position: relative;
}
.field-with-counter-input .form-input {
  padding-right: 84px;
}
.field-with-counter-textarea .form-textarea {
  padding-right: 88px;
  padding-bottom: 28px;
}
.field-counter {
  position: absolute;
  right: 12px;
  bottom: 8px;
  font-size: 12px;
  line-height: 1;
  color: var(--color-text-muted);
  pointer-events: none;
}
.tag-input-wrap { position: relative; }
.tag-input-row { display: flex; gap: 8px; }
.tag-input-row .form-input { flex: 1; }
.tag-suggestion-panel {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  right: 0;
  z-index: 20;
  max-height: 220px;
  overflow-y: auto;
  border: 1px solid rgba(0, 202, 224, 0.3);
  border-radius: 12px;
  background: #11151d;
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.45);
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.tag-suggestion-item {
  width: 100%;
  text-align: left;
  border: 1px solid transparent;
  background: transparent;
  color: var(--color-text-secondary);
  border-radius: 9px;
  padding: 8px 10px;
  font-size: 13px;
  cursor: pointer;
  transition: all .18s ease;
}
.tag-suggestion-item:hover {
  border-color: rgba(0, 202, 224, 0.25);
  background: rgba(0, 202, 224, 0.12);
  color: #d6fbff;
}
.tag-list { display: flex; flex-wrap: wrap; gap: 8px; }
.tag-chip { border: 1px solid rgba(0, 202, 224, 0.35); background: rgba(0, 202, 224, 0.12); color: #b5f4ff; border-radius: 999px; padding: 4px 10px; font-size: 12px; cursor: pointer; }
.media-type-switch { display: flex; gap: 8px; }
.type-btn { flex: 1; padding: 8px 10px; border-radius: 10px; border: 1px solid var(--color-tint-white-08, rgba(255, 255, 255, 0.08)); background: var(--color-tint-white-03, rgba(255, 255, 255, 0.03)); color: var(--color-text-secondary); cursor: pointer; font-size: 13px; }
.type-btn.active { border-color: rgba(0, 202, 224, 0.45); background: rgba(0, 202, 224, 0.14); color: #c9f8ff; }
.upload-actions { display: flex; gap: 8px; }
.upload-dropzone { width: 100%; border: 1px dashed rgba(0, 202, 224, 0.38); border-radius: 12px; background: rgba(0, 202, 224, 0.06); padding: 14px; display: flex; flex-direction: column; align-items: flex-start; gap: 8px; transition: border-color .2s ease, background .2s ease; cursor: pointer; }
.upload-dropzone.active { border-color: rgba(0, 202, 224, 0.78); background: rgba(0, 202, 224, 0.16); }
.upload-dropzone.disabled { opacity: .6; cursor: not-allowed; }
.hidden-upload-input { display: none; }
.dropzone-title { font-size: 13px; color: var(--color-text-primary); font-weight: 600; }
.dropzone-hint { font-size: 12px; color: var(--color-text-muted); }
.single-upload-btn { width: fit-content; }
.share-preview-video { border: 1px solid var(--color-tint-white-08, rgba(255,255,255,.08)); border-radius: 12px; background: var(--color-tint-white-03, rgba(255,255,255,.03)); padding: 8px; }
.share-preview-video video { width: 100%; max-height: 280px; object-fit: contain; border-radius: 8px; background: #000; display: block; }
.preview-image-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
.preview-image-item { border: 1px solid var(--color-tint-white-08, rgba(255,255,255,.08)); border-radius: 10px; padding: 4px; background: var(--color-tint-white-03, rgba(255,255,255,.03)); }
.preview-image-item img { width: 100%; aspect-ratio: 1 / 1; object-fit: cover; border-radius: 8px; display: block; }
.upload-status { font-size: 12px; color: var(--color-text-muted); }
.image-preview-list { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
.image-preview-item { border: 1px solid var(--color-tint-white-08, rgba(255,255,255,.08)); border-radius: 10px; padding: 6px; display: flex; flex-direction: column; gap: 6px; }
.image-preview-item img { width: 100%; aspect-ratio: 1 / 1; object-fit: cover; border-radius: 8px; }
.video-preview-row { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 8px 10px; border-radius: 10px; background: var(--color-tint-white-03, rgba(255,255,255,.03)); border: 1px solid var(--color-tint-white-08, rgba(255,255,255,.08)); }
.video-url { font-size: 12px; color: var(--color-text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.remove-btn { border: none; background: rgba(255, 107, 107, 0.2); color: #ff9d9d; border-radius: 8px; padding: 4px 8px; cursor: pointer; font-size: 12px; flex-shrink: 0; }
.cover-editor { display: flex; flex-direction: column; gap: 10px; }
.cover-preview {
  width: 180px;
  aspect-ratio: 16 / 9;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--color-tint-white-08, rgba(255,255,255,.08));
  background: var(--color-tint-white-03, rgba(255,255,255,.03));
}
.cover-preview.clickable {
  cursor: pointer;
}
.cover-preview.clickable:hover {
  border-color: rgba(0, 202, 224, 0.55);
  box-shadow: 0 0 0 1px rgba(0, 202, 224, 0.24) inset;
}
.cover-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.cover-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  font-size: 12px;
  line-height: 1.4;
  color: var(--color-text-muted);
  padding: 8px;
}
.cover-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.share-dialog-footer { display: flex; justify-content: flex-end; gap: 12px; padding: 16px 24px 20px; border-top: 1px solid var(--color-tint-white-06, rgba(255, 255, 255, 0.06)); }
.btn { padding: 10px 16px; font-size: 14px; font-weight: 500; border-radius: 10px; cursor: pointer; border: none; }
.btn.tiny { padding: 8px 12px; font-size: 12px; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-secondary { background: var(--color-tint-white-06, rgba(255, 255, 255, 0.06)); color: var(--color-text-secondary); border: 1px solid var(--color-tint-white-08, rgba(255, 255, 255, 0.08)); }
.btn-primary { background: rgba(0, 202, 224, 0.9); color: #fff; box-shadow: 0 4px 14px rgba(0, 202, 224, 0.3); }
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
