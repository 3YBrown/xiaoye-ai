<script setup>
import { ref, computed, watch, nextTick, h } from 'vue'
import { NSpin, NDropdown, NPopover, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '../stores/user'
import { useGenerate } from '../composables/useGenerate'
import { usePricingStore } from '../stores/pricing'
import { useModelsStore } from '../stores/models'

const { t } = useI18n()

const props = defineProps({
  creativeMode: { type: String, default: 'image' },
  loading: { type: Boolean, default: false }
})

const emit = defineEmits(['submit', 'update:creativeMode'])

const userStore = useUserStore()
const pricingStore = usePricingStore()
const modelsStore = useModelsStore()
const { uploadImageToOSS, optimizePrompt } = useGenerate()
const message = useMessage()

// 加载模型列表
modelsStore.loadModels()

const prompt = ref('')
const inputRef = ref(null)
const showRatioPopover = ref(false)
const uploadsExpanded = ref(false)
const optimizingPrompt = ref(false)
const optimizePanelVisible = ref(false)
const optimizePrimary = ref(null)

const optimizeBackupPrompt = ref('')
const optimizeRequestId = ref(0)  // 用于防重复请求

// --- Prompt Optimize Style ---
const selectedOptimizeStyle = ref('balanced')
const optimizeStyles = [
  { key: 'balanced', icon: '⚖️', label: computed(() => t('composer.optimizeStyleBalanced')) },
  { key: 'creative', icon: '✨', label: computed(() => t('composer.optimizeStyleCreative')) },
  { key: 'detail', icon: '🔍', label: computed(() => t('composer.optimizeStyleDetail')) },
  { key: 'commercial', icon: '💼', label: computed(() => t('composer.optimizeStyleCommercial')) }
]

const MAX_PROMPT_LENGTH = 4000

// --- Image params ---
const uploadedImagePreviews = ref([])
const uploadedImageUrls = ref([])
const uploadedImageBase64s = ref([])  // 电商模式需要 base64
const selectedModel = ref('')
const aspectRatio = ref('1:1')
const imageSize = ref('1K')

// --- Ecommerce params ---
const ecommerceType = ref('淘宝')
const imageType = ref('详情图')
const outputCount = ref(7)

const ecommercePlatforms = ['淘宝', '京东', '拼多多', '1688', '小红书', '闲鱼']
const ecommerceImageTypes = ['详情图', '白底主图', '产品单图']
const ecommerceOutputCounts = [5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]

const ecommerceSizeOptions = computed(() => pricingStore.getEcommerceSizeOptions())

const baseImageRatios = [
  { value: '1:1', w: 16, h: 16 },
  { value: '3:4', w: 12, h: 16 },
  { value: '4:3', w: 16, h: 12 },
  { value: '9:16', w: 10, h: 18 },
  { value: '16:9', w: 18, h: 10 },
  { value: '2:3', w: 11, h: 16 },
  { value: '3:2', w: 16, h: 11 },
  { value: '4:5', w: 13, h: 16 },
  { value: '5:4', w: 16, h: 13 },
  { value: '21:9', w: 20, h: 8 }
]

const extraFlashRatios = [
  { value: '1:4', w: 5, h: 20 },
  { value: '4:1', w: 20, h: 5 },
  { value: '1:8', w: 3, h: 22 },
  { value: '8:1', w: 22, h: 3 }
]

const imageRatios = ref(baseImageRatios)

// 等待模型加载完成后初始化 selectedModel
watch(() => modelsStore.loaded, (loaded) => {
  if (!loaded) return
  const saved = localStorage.getItem('selectedModel')
  const safeModelId = modelsStore.ensureModelId(saved || modelsStore.defaultModelId)
  selectedModel.value = safeModelId
}, { immediate: true })

watch(selectedModel, (newModel) => {
  if (!newModel) return
  const safeModelId = modelsStore.ensureModelId(newModel)
  if (safeModelId !== newModel) {
    selectedModel.value = safeModelId
    return
  }
  localStorage.setItem('selectedModel', safeModelId)
  // 切换模型时检查当前尺寸是否仍可用。
  const availableSizes = pricingStore.getImageSizeOptions(safeModelId)
  const sizeExists = availableSizes.some(s => s.value === imageSize.value)
  if (!sizeExists && availableSizes.length > 0) {
    imageSize.value = availableSizes[0].value
  }
  // 切换模型时更新可用宽高比
  const newRatios = safeModelId === 'gemini-3.1-flash-image-preview'
    ? [...baseImageRatios, ...extraFlashRatios]
    : baseImageRatios
  imageRatios.value = newRatios
  if (!newRatios.some(r => r.value === aspectRatio.value)) {
    aspectRatio.value = '1:1'
  }
}, { immediate: true })
watch(() => props.creativeMode, (mode) => {
  if (mode === 'ecommerce' && imageSize.value === '1K') imageSize.value = '2K'
})

// 当用户手动改写输入时，清空旧备选，避免误用过期结果。
watch(prompt, (value) => {
  if (!optimizePrimary.value) return
  if (value !== optimizePrimary.value.prompt) optimizePrimary.value = null
})

// 上传图片数量变化时自动折叠
watch(() => uploadedImagePreviews.value.length, (len) => {
  if (len >= 2) uploadsExpanded.value = false
})

const formatImageModelLabel = (model) => model.name
const imageModelOptions = computed(() => modelsStore.imageModels.map(model => ({
  label: formatImageModelLabel(model),
  value: model.id,
  icon: model.provider === 'volcengine' ? '/images/jmlogo.png' : null
})))

// 动态获取图片尺寸选项（根据选择的模型）
const imageSizeOptions = computed(() => pricingStore.getImageSizeOptions(selectedModel.value))

// --- Video params ---
const videoModel = ref('doubao-seedance-1-5-pro-251215')
const videoResolution = ref('720p')
const videoRatio = ref('16:9')
const videoDuration = ref(5)
const generateAudio = ref(true)
const firstFramePreview = ref(null)
const firstFrameUrl = ref(null)
const lastFramePreview = ref(null)
const lastFrameUrl = ref(null)

const isVeoModel = computed(() => videoModel.value.startsWith('veo-'))

// 根据上传帧数自动推断视频模式
const videoMode = computed(() => {
  if (firstFrameUrl.value && lastFrameUrl.value) return 'first-last-frame'
  if (firstFrameUrl.value) return 'first-frame'
  return 'text-to-video'
})

const allVideoResolutions = {
  seedance: [
    { label: '480P', value: '480p' },
    { label: '720P', value: '720p' },
    { label: '1080P', value: '1080p', premium: true }
  ],
  veo: [
    { label: '720P', value: '720p' },
    { label: '1080P', value: '1080p', premium: true },
    { label: '4K', value: '4k', premium: true }
  ]
}
const videoResolutions = computed(() => isVeoModel.value ? allVideoResolutions.veo : allVideoResolutions.seedance)

const allVideoRatios = {
  seedance: [
    { value: '21:9', w: 20, h: 8 },
    { value: '16:9', w: 18, h: 10 },
    { value: '4:3',  w: 16, h: 12 },
    { value: '1:1',  w: 14, h: 14 },
    { value: '3:4',  w: 12, h: 16 },
    { value: '9:16', w: 10, h: 18 }
  ],
  veo: [
    { value: '16:9', w: 18, h: 10 },
    { value: '9:16', w: 10, h: 18 }
  ]
}
const videoRatios = computed(() => isVeoModel.value ? allVideoRatios.veo : allVideoRatios.seedance)

const allVideoDurations = {
  seedance: [5, 8, 12].map(d => ({ label: `${d}s`, value: d })),
  veo: [4, 6, 8].map(d => ({ label: `${d}s`, value: d }))
}
// Veo 1080p/4k 只允许 8 秒
const veoHighResDuration = [{ label: '8s', value: 8 }]
const videoDurationOptions = computed(() => {
  if (!isVeoModel.value) return allVideoDurations.seedance
  if (videoResolution.value === '1080p' || videoResolution.value === '4k') return veoHighResDuration
  return allVideoDurations.veo
})

// 视频模型下拉选项
const videoModelOptions = [
  { key: 'doubao-seedance-1-5-pro-251215', label: 'Seedance-1.5', icon: '/images/jmlogo.png' },
  { key: 'veo-3.1-generate-preview', label: 'Veo 3.1', emoji: '✦' },
]
const currentVideoModel = computed(() => videoModelOptions.find(m => m.key === videoModel.value) || videoModelOptions[0])

// 切换模型时重置不兼容的参数
watch(videoModel, (newModel) => {
  const isVeo = newModel.startsWith('veo-')
  if (isVeo) {
    // Veo 仅支持 16:9 和 9:16
    if (videoRatio.value !== '16:9' && videoRatio.value !== '9:16') {
      videoRatio.value = '16:9'
    }
    // Veo 支持 4/6/8 秒
    if (![4, 6, 8].includes(videoDuration.value)) {
      videoDuration.value = 8
    }
    // Veo 最低 720p
    if (videoResolution.value === '480p') {
      videoResolution.value = '720p'
    }
    // Veo 1080p/4k 强制 8 秒
    if ((videoResolution.value === '1080p' || videoResolution.value === '4k') && videoDuration.value !== 8) {
      videoDuration.value = 8
    }
  } else {
    // Seedance 不支持 4k
    if (videoResolution.value === '4k') {
      videoResolution.value = '1080p'
    }
    // Seedance 不支持 4 秒
    if (videoDuration.value === 4) {
      videoDuration.value = 5
    }
    if (videoDuration.value === 6) {
      videoDuration.value = 5
    }
    // Seedance 不支持参考图，清除
    uploadedImagePreviews.value = []
    uploadedImageUrls.value = []
    uploadedImageBase64s.value = []
  }
})

// Veo 切换分辨率时：1080p/4k 强制 8 秒
watch(videoResolution, (newRes) => {
  if (isVeoModel.value && (newRes === '1080p' || newRes === '4k')) {
    videoDuration.value = 8
  }
})

// --- Computed ---
const hasUploadedImages = computed(() => uploadedImageUrls.value.length > 0)

const requiredCredits = computed(() => {
  if (props.creativeMode === 'video') {
    if (isVeoModel.value) {
      // Veo 3.1 按分辨率阶梯定价，原生含音频
      return pricingStore.getVeoCredits(videoResolution.value, videoDuration.value)
    }
    return pricingStore.getVideoCredits(videoResolution.value, videoDuration.value, generateAudio.value)
  } else if (props.creativeMode === 'ecommerce') {
    return pricingStore.getEcommerceCredits(imageSize.value, outputCount.value)
  } else {
    return pricingStore.getImageCredits(selectedModel.value, imageSize.value)
  }
})

// --- Dropdown helpers ---
const modeDropOptions = computed(() => [
  { label: '🎨 ' + t('composer.imageGen'), key: 'image' },
  { label: '🎬 ' + t('composer.videoGen'), key: 'video' },
  { label: '🛍️ ' + t('composer.ecommerceGen'), key: 'ecommerce' }
])
const onModeSelect = (key) => { emit('update:creativeMode', key) }

const modelDropOptions = computed(() => imageModelOptions.value.map(o => ({
  label: o.icon
    ? () => h('span', { style: 'display:flex;align-items:center;gap:6px' }, [
        h('img', { src: o.icon, style: 'width:16px;height:16px;border-radius:3px' }),
        o.label
      ])
    : o.label,
  key: o.value
})))

const selectedImageModel = computed(() => {
  if (!selectedModel.value) return null
  return modelsStore.getModelById(selectedModel.value) || modelsStore.getModelById(modelsStore.defaultModelId)
})
const modelLabel = computed(() => selectedImageModel.value ? formatImageModelLabel(selectedImageModel.value) : '')
const modelIcon = computed(() => {
  if (!selectedImageModel.value) return null
  return selectedImageModel.value.provider === 'volcengine' ? '/images/jmlogo.png' : null
})
const onModelSelect = (modelId) => {
  selectedModel.value = modelsStore.ensureModelId(modelId)
}
const modeLabel = computed(() => {
  const map = { image: '🎨 ' + t('composer.imageGen'), video: '🎬 ' + t('composer.videoGen'), ecommerce: '🛍️ ' + t('composer.ecommerceGen') }
  return map[props.creativeMode] || '🎨 ' + t('composer.imageGen')
})

const ePlatformDropOptions = computed(() => ecommercePlatforms.map(p => ({ label: p, key: p })))
const eImageTypeDropOptions = computed(() => ecommerceImageTypes.map(it => ({ label: it, key: it })))
const eOutputCountDropOptions = computed(() => ecommerceOutputCounts.map(c => ({ label: t('composer.count', { count: c }), key: c })))

const placeholderText = computed(() => {
  if (props.creativeMode === 'video') return t('composer.videoPlaceholder')
  if (props.creativeMode === 'ecommerce') return t('composer.ecommercePlaceholder')
  return t('composer.imagePlaceholder')
})

// --- Methods ---
const buildCurrentParams = () => {
  if (props.creativeMode === 'video') {
    return {
      model: videoModel.value,
      video_mode: videoMode.value,
      resolution: videoResolution.value,
      ratio: videoRatio.value,
      duration: videoDuration.value,
      generate_audio: generateAudio.value
    }
  }

  if (props.creativeMode === 'ecommerce') {
    return {
      model: pricingStore.ecommerceModel,
      aspectRatio: aspectRatio.value,
      imageSize: imageSize.value,
      outputCount: outputCount.value,
      imageType: imageType.value,
      ecommerceType: ecommerceType.value
    }
  }

  return {
    model: selectedModel.value,
    aspectRatio: aspectRatio.value,
    imageSize: imageSize.value
  }
}

const applyCandidate = (candidate, options = {}) => {
  if (!candidate?.prompt) return
  const { silent = false, rememberUndo = false } = options
  if (rememberUndo) {
    optimizeBackupPrompt.value = prompt.value
  }
  // 只替换提示词，不改变用户的任何其他选择（模型、比例、图片类型等）
  prompt.value = candidate.prompt
  if (!silent) {
    message.success(t('composer.optimizeApplied'))
  }
}

const runPromptOptimization = async () => {
  if (!userStore.requireAuth()) return

  const trimmedPrompt = prompt.value.trim()
  if (!trimmedPrompt) {
    message.warning(t('composer.optimizePromptRequired'))
    return
  }

  if (trimmedPrompt.length > MAX_PROMPT_LENGTH) {
    message.warning(`提示词过长，请控制在 ${MAX_PROMPT_LENGTH} 字符以内（当前 ${trimmedPrompt.length} 字符）`)
    return
  }

  if (optimizingPrompt.value) return
  optimizingPrompt.value = true
  const currentRequestId = optimizeRequestId.value + 1
  optimizeRequestId.value = currentRequestId
  try {
    const response = await optimizePrompt({
      prompt: trimmedPrompt,
      creative_mode: props.creativeMode,
      style: selectedOptimizeStyle.value,
      target_model: props.creativeMode === 'image' ? selectedModel.value : '',
      current_params: buildCurrentParams()
    })

    if (currentRequestId !== optimizeRequestId.value) return
    const candidates = Array.isArray(response?.candidates)
      ? response.candidates.filter(item => item?.prompt)
      : []
    if (!candidates.length) {
      message.warning(t('composer.optimizeEmpty'))
      return
    }

    optimizePanelVisible.value = true
    const bestCandidate = candidates[0]
    optimizePrimary.value = bestCandidate
    applyCandidate(bestCandidate, { silent: true, rememberUndo: true })
    message.success(t('composer.optimizeApplied'))
  } catch (e) {
    message.error(e.response?.data?.error || t('composer.optimizeFailed'))
  } finally {
    optimizingPrompt.value = false
  }
}
const undoOptimizedPrompt = () => {
  if (!optimizeBackupPrompt.value) return
  prompt.value = optimizeBackupPrompt.value
  optimizeBackupPrompt.value = ''
  message.success('已撤销优化结果')
}

const generateWithCandidate = (candidate) => {
  if (!candidate?.prompt) return
  applyCandidate(candidate, { silent: true })
  sendMessage()
}

const sendMessage = () => {
  if (props.creativeMode === 'ecommerce') {
    if (!prompt.value.trim()) return
    if (uploadedImageUrls.value.length === 0) { alert(t('composer.ecommerceNeedImage')); return }
  } else if (!prompt.value.trim() && !hasUploadedImages.value) return
  if (props.loading) return
  if (!userStore.requireAuth()) return

  const payload = { prompt: prompt.value.trim(), creativeMode: props.creativeMode }

  if (props.creativeMode === 'video') {
    payload.params = {
      mode: videoMode.value, model: videoModel.value,
      resolution: videoResolution.value, ratio: videoRatio.value,
      duration: videoDuration.value, generate_audio: generateAudio.value
    }
    if (firstFrameUrl.value) payload.firstFrameUrl = firstFrameUrl.value
    if (lastFrameUrl.value) payload.lastFrameUrl = lastFrameUrl.value
    // Veo 3.1 参考图
    if (isVeoModel.value && uploadedImageUrls.value.length > 0) {
      payload.images = [...uploadedImageUrls.value]
    }
  } else if (props.creativeMode === 'ecommerce') {
    payload.params = {
      model: pricingStore.ecommerceModel,
      aspectRatio: aspectRatio.value,
      imageSize: imageSize.value,
      outputCount: outputCount.value,
      imageType: imageType.value,
      ecommerceType: ecommerceType.value
    }
    payload.images = [...uploadedImageUrls.value]
    payload.imageBase64s = [...uploadedImageBase64s.value]
  } else {
    payload.params = {
      model: selectedModel.value, aspectRatio: aspectRatio.value, imageSize: imageSize.value
    }
    if (uploadedImageUrls.value.length > 0) payload.images = [...uploadedImageUrls.value]
  }

  payload.credits = requiredCredits.value
  emit('submit', payload)

  // Clear inputs
  prompt.value = ''
  uploadedImagePreviews.value = []
  uploadedImageUrls.value = []
  uploadedImageBase64s.value = []
  optimizePrimary.value = null
  optimizeBackupPrompt.value = ''
  optimizePanelVisible.value = false
  uploadsExpanded.value = false
  firstFramePreview.value = null; firstFrameUrl.value = null
  lastFramePreview.value = null; lastFrameUrl.value = null
}

const handleUpload = async (e) => {
  if (!userStore.requireAuth()) return
  const files = e.target.files
  if (!files?.length) return
  if (uploadedImageUrls.value.length + files.length > 3) return
  for (const file of files) {
    if (file.size > 10 * 1024 * 1024) continue
    const reader = new FileReader()
    reader.onload = async (ev) => {
      const dataUrl = ev.target.result
      try {
        const ossUrl = await uploadImageToOSS(dataUrl)
        uploadedImagePreviews.value.push(dataUrl)
        uploadedImageUrls.value.push(ossUrl)
        // 电商模式保存 base64 数据
        if (props.creativeMode === 'ecommerce') {
          uploadedImageBase64s.value.push(dataUrl.split(',')[1])
        }
      } catch (err) {
        console.error('上传图片失败:', err)
        alert(t('composer.uploadFailed'))
      }
    }
    reader.readAsDataURL(file)
  }
  e.target.value = ''
}
const removeUpload = (idx) => {
  uploadedImagePreviews.value.splice(idx, 1)
  uploadedImageUrls.value.splice(idx, 1)
  uploadedImageBase64s.value.splice(idx, 1)
  if (uploadedImagePreviews.value.length === 0) uploadsExpanded.value = false
}

const handleFrameUpload = async (type, e) => {
  const file = e.target.files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = async (ev) => {
    const dataUrl = ev.target.result
    try {
      const ossUrl = await uploadImageToOSS(dataUrl)
      if (type === 'first') { firstFramePreview.value = dataUrl; firstFrameUrl.value = ossUrl }
      else { lastFramePreview.value = dataUrl; lastFrameUrl.value = ossUrl }
    } catch (err) {
      console.error('上传帧图片失败:', err)
      alert(t('composer.uploadFailed'))
    }
  }
  reader.readAsDataURL(file)
  e.target.value = ''
}
const removeFrame = (type) => {
  if (type === 'first') { firstFramePreview.value = null; firstFrameUrl.value = null }
  else { lastFramePreview.value = null; lastFrameUrl.value = null }
}

const handleKeydown = (e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage() } }

// Exposed methods
const fillPrompt = (text) => {
  prompt.value = text
  nextTick(() => inputRef.value?.focus())
}

const fillFromGeneration = (gen) => {
  prompt.value = gen.prompt || ''
  const p = gen.params || {}
  const genType = gen.type || 'image'
  emit('update:creativeMode', genType)
  if (genType === 'video') {
    videoResolution.value = p.resolution || videoResolution.value
    videoRatio.value = p.ratio || videoRatio.value
    videoDuration.value = p.duration || videoDuration.value
  } else if (genType === 'ecommerce') {
    aspectRatio.value = p.aspectRatio || aspectRatio.value
    imageSize.value = p.imageSize || imageSize.value
    ecommerceType.value = p.ecommerceType || ecommerceType.value
    imageType.value = p.imageType || imageType.value
    outputCount.value = p.outputCount || outputCount.value
  } else {
    selectedModel.value = modelsStore.ensureModelId(p.model || selectedModel.value)
    aspectRatio.value = p.aspectRatio || aspectRatio.value
    imageSize.value = p.imageSize || imageSize.value
  }
  nextTick(() => inputRef.value?.focus())
}

const fillEditImage = async (imageUrl) => {
  emit('update:creativeMode', 'image')
  if (imageUrl.startsWith('data:')) {
    uploadedImagePreviews.value = [imageUrl]
    try {
      const ossUrl = await uploadImageToOSS(imageUrl)
      uploadedImageUrls.value = [ossUrl]
    } catch (e) {
      console.error('上传图片失败:', e)
      uploadedImagePreviews.value = []
      uploadedImageUrls.value = []
      return
    }
  } else {
    uploadedImagePreviews.value = [imageUrl]
    uploadedImageUrls.value = [imageUrl]
  }
  nextTick(() => inputRef.value?.focus())
}

defineExpose({ fillPrompt, fillFromGeneration, fillEditImage })
</script>

<template>
  <div class="composer-wrap">
    <div class="composer-card">
      <div class="composer-body">
        <!-- Image/Ecommerce/Veo-video mode uploads -->
        <div v-if="creativeMode === 'image' || creativeMode === 'ecommerce' || (creativeMode === 'video' && isVeoModel)" class="composer-uploads">
          <!-- 多张图片折叠态 -->
          <div v-if="uploadedImagePreviews.length >= 2 && !uploadsExpanded"
            class="upload-stack" @click="uploadsExpanded = true" title="点击展开查看">
            <div v-for="(img, i) in uploadedImagePreviews" :key="i"
              class="stack-card" :style="{
                '--stack-rotate': (i === 0 ? -8 : i === 1 ? 5 : -2) + 'deg',
                'z-index': uploadedImagePreviews.length - i
              }">
              <img :src="img" />
            </div>
            <span class="stack-badge">{{ uploadedImagePreviews.length }}</span>
          </div>
          <!-- 单张图片 / 展开态 -->
          <template v-if="uploadedImagePreviews.length === 1 || uploadsExpanded">
            <div v-for="(img, i) in uploadedImagePreviews" :key="i"
              class="upload-card" :style="{ '--tilt': (i % 2 === 0 ? -6 : 5) + 'deg' }">
              <img :src="img" />
              <button class="card-remove" @click="removeUpload(i)">×</button>
            </div>
          </template>
          <!-- 添加按钮（空态 / 单张 / 展开态，且未满 3 张） -->
          <label v-if="uploadedImageUrls.length < 3 && (uploadedImagePreviews.length <= 1 || uploadsExpanded)"
            class="upload-card" :style="{ '--tilt': uploadedImagePreviews.length === 0 ? '-4deg' : '3deg' }">
            <input type="file" accept="image/*" multiple @change="handleUpload" hidden />
            <div class="frame-placeholder">
              <span class="add-icon">+</span>
              <span class="frame-lbl">{{ creativeMode === 'ecommerce' ? $t('composer.productImage') : $t('composer.refImage') }}</span>
            </div>
          </label>
          <!-- 展开态收起按钮 -->
          <button v-if="uploadsExpanded && uploadedImagePreviews.length >= 2"
            class="stack-collapse-btn" @click="uploadsExpanded = false" title="收起">−</button>
        </div>

        <!-- Video mode: always show first + last frame uploads -->
        <div v-if="creativeMode === 'video'" class="composer-uploads">
          <div class="upload-card" :style="{ '--tilt': '-5deg' }">
            <template v-if="firstFramePreview">
              <img :src="firstFramePreview" />
              <button class="card-remove" @click="removeFrame('first')">×</button>
            </template>
            <label v-else class="frame-placeholder">
              <input type="file" accept="image/*" @change="handleFrameUpload('first', $event)" hidden />
              <span class="add-icon">+</span>
              <span class="frame-lbl">{{ $t('composer.firstFrame') }}</span>
            </label>
          </div>
          <span class="frame-sep">→</span>
          <div class="upload-card" :style="{ '--tilt': '5deg' }">
            <template v-if="lastFramePreview">
              <img :src="lastFramePreview" />
              <button class="card-remove" @click="removeFrame('last')">×</button>
            </template>
            <label v-else class="frame-placeholder">
              <input type="file" accept="image/*" @change="handleFrameUpload('last', $event)" hidden />
              <span class="add-icon">+</span>
              <span class="frame-lbl">{{ $t('composer.lastFrame') }}</span>
            </label>
          </div>
        </div>

        <div class="input-wrapper">
          <textarea
            ref="inputRef"
            v-model="prompt"
            @keydown="handleKeydown"
            :placeholder="placeholderText"
            class="composer-input"
          />
          <div v-if="prompt.trim()" class="input-actions">
            <button class="action-chip optimize-chip" :disabled="optimizingPrompt" @click="runPromptOptimization()" :title="$t('composer.optimizeAction')">
              <NSpin v-if="optimizingPrompt" size="small" :stroke-width="20" />
              <template v-else>
                <svg class="chip-diamond" width="9" height="9" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L4 8L12 22L20 8L12 2Z M6 8H18L12 3L6 8Z"/></svg>
                <span class="optimize-cost">2</span>
                <span>AI</span>
              </template>
            </button>
          </div>
        </div>
      </div>

      <!-- Toolbar -->
      <div class="composer-toolbar">
        <div class="toolbar-left">
          <NDropdown :options="modeDropOptions" @select="onModeSelect" trigger="click" placement="bottom-start">
            <button class="pill pill-primary">
              {{ modeLabel }}
              <svg class="pill-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M6 9l6 6 6-6"/></svg>
            </button>
          </NDropdown>

          <!-- Image params -->
          <template v-if="creativeMode === 'image'">
            <NDropdown :options="modelDropOptions" @select="onModelSelect" trigger="click" placement="bottom-start">
              <button class="pill">
                <img v-if="modelIcon" :src="modelIcon" class="pill-icon" />
                {{ modelLabel }}
              </button>
            </NDropdown>

            <NPopover trigger="click" placement="bottom-start" :show-arrow="false" raw
              content-class="ratio-popover" v-model:show="showRatioPopover">
              <template #trigger>
                <button class="pill">▢ {{ aspectRatio }}&nbsp;&nbsp;{{ imageSize }}
                  <svg class="pill-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M6 9l6 6 6-6"/></svg>
                </button>
              </template>
              <div class="pop-panel">
                <div class="pop-section-title">{{ $t('composer.selectRatio') }}</div>
                <div class="ratio-grid">
                  <button v-for="r in imageRatios" :key="r.value"
                    :class="['ratio-cell', { active: aspectRatio === r.value }]"
                    @click="aspectRatio = r.value">
                    <span class="ratio-icon" :style="{ width: r.w + 'px', height: r.h + 'px' }"></span>
                    <span class="ratio-label">{{ r.value }}</span>
                  </button>
                </div>
                <div class="pop-section-title" style="margin-top: 14px;">{{ $t('composer.selectResolution') }}</div>
                <div class="seg-group">
                  <button v-for="s in imageSizeOptions" :key="s.value"
                    :class="['seg-btn', { active: imageSize === s.value }]"
                    @click="imageSize = s.value">
                    {{ s.label }} <span v-if="s.badge" class="seg-badge">{{ s.badge }}</span>
                  </button>
                </div>
              </div>
            </NPopover>
          </template>

          <!-- Ecommerce params -->
          <template v-else-if="creativeMode === 'ecommerce'">
            <button class="pill" style="cursor: default; opacity: .65"><img src="/images/jmlogo.png" class="pill-icon" /> Seedream-4.5</button>

            <NDropdown :options="ePlatformDropOptions" @select="k => ecommerceType = k" trigger="click" placement="bottom-start">
              <button class="pill">🏪 {{ ecommerceType }}
                <svg class="pill-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M6 9l6 6 6-6"/></svg>
              </button>
            </NDropdown>

            <NDropdown :options="eImageTypeDropOptions" @select="k => imageType = k" trigger="click" placement="bottom-start">
              <button class="pill">📷 {{ imageType }}
                <svg class="pill-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M6 9l6 6 6-6"/></svg>
              </button>
            </NDropdown>

            <NDropdown :options="eOutputCountDropOptions" @select="k => outputCount = k" trigger="click" placement="bottom-start">
              <button class="pill">🔢 {{ $t('composer.count', { count: outputCount }) }}
                <svg class="pill-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M6 9l6 6 6-6"/></svg>
              </button>
            </NDropdown>

            <NPopover trigger="click" placement="bottom-start" :show-arrow="false" raw content-class="ratio-popover">
              <template #trigger>
                <button class="pill">▢ {{ aspectRatio }}&nbsp;&nbsp;{{ imageSize }}
                  <svg class="pill-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M6 9l6 6 6-6"/></svg>
                </button>
              </template>
              <div class="pop-panel">
                <div class="pop-section-title">{{ $t('composer.selectRatio') }}</div>
                <div class="ratio-grid">
                  <button v-for="r in imageRatios" :key="r.value"
                    :class="['ratio-cell', { active: aspectRatio === r.value }]"
                    @click="aspectRatio = r.value">
                    <span class="ratio-icon" :style="{ width: r.w + 'px', height: r.h + 'px' }"></span>
                    <span class="ratio-label">{{ r.value }}</span>
                  </button>
                </div>
                <div class="pop-section-title" style="margin-top: 14px;">{{ $t('composer.selectResolution') }}</div>
                <div class="seg-group">
                  <button v-for="s in ecommerceSizeOptions" :key="s.value"
                    :class="['seg-btn', { active: imageSize === s.value }]"
                    @click="imageSize = s.value">
                    {{ s.label }} <span v-if="s.badge" class="seg-badge">{{ s.badge }}</span>
                  </button>
                </div>
              </div>
            </NPopover>
          </template>

          <!-- Video params -->
          <template v-else-if="creativeMode === 'video'">
            <NDropdown
              :options="videoModelOptions.map(m => ({
                label: () => h('span', { style: 'display:flex;align-items:center;gap:6px' }, [
                  m.icon ? h('img', { src: m.icon, style: 'width:16px;height:16px;border-radius:3px' }) : h('span', { style: 'width:16px;text-align:center;font-size:14px' }, m.emoji),
                  m.label
                ]),
                key: m.key
              }))"
              @select="k => videoModel = k"
              trigger="click"
              placement="bottom-start"
            >
              <button class="pill">
                <img v-if="currentVideoModel.icon" :src="currentVideoModel.icon" class="pill-icon" />
                <span v-else style="font-size:14px;margin-right:2px">{{ currentVideoModel.emoji }}</span>
                {{ currentVideoModel.label }}
                <svg class="pill-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M6 9l6 6 6-6"/></svg>
              </button>
            </NDropdown>

            <NPopover trigger="click" placement="bottom-start" :show-arrow="false" raw content-class="ratio-popover">
              <template #trigger>
                <button class="pill">▢ {{ videoRatio }}&nbsp;&nbsp;{{ videoResolution.toUpperCase() }}
                  <svg class="pill-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M6 9l6 6 6-6"/></svg>
                </button>
              </template>
              <div class="pop-panel">
                <div class="pop-section-title">{{ $t('composer.selectRatio') }}</div>
                <div class="ratio-grid">
                  <button v-for="r in videoRatios" :key="r.value"
                    :class="['ratio-cell', { active: videoRatio === r.value }]"
                    @click="videoRatio = r.value">
                    <span class="ratio-icon" :style="{ width: r.w + 'px', height: r.h + 'px' }"></span>
                    <span class="ratio-label">{{ r.value }}</span>
                  </button>
                </div>
                <div class="pop-section-title" style="margin-top: 14px;">{{ $t('composer.selectResolution') }}</div>
                <div class="seg-group">
                  <button v-for="res in videoResolutions" :key="res.value"
                    :class="['seg-btn', { active: videoResolution === res.value }]"
                    @click="videoResolution = res.value">
                    {{ res.label }} <span v-if="res.premium" class="seg-star">★</span>
                  </button>
                </div>
              </div>
            </NPopover>

            <NDropdown :options="videoDurationOptions.map(d => ({ label: d.label, key: d.value }))" @select="k => videoDuration = k" trigger="click" placement="bottom-start">
              <button class="pill">⏱ {{ videoDuration }}s</button>
            </NDropdown>

            <button class="pill" :class="{ 'pill-active': generateAudio }" @click="generateAudio = !generateAudio" :title="$t('composer.generateAudio')">
              🔊
            </button>
          </template>
        </div>

        <div class="toolbar-right">
          <span class="credits-badge"><svg class="credits-icon" width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L4 8L12 22L20 8L12 2Z"/></svg> {{ requiredCredits }}</span>
          <button class="send-btn" :disabled="loading || (!prompt.trim() && !hasUploadedImages) || (creativeMode === 'ecommerce' && !hasUploadedImages)" @click="sendMessage">
            <NSpin v-if="loading" size="small" />
            <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M4 12l1.41 1.41L11 7.83V20h2V7.83l5.58 5.59L20 12l-8-8-8 8z"/></svg>
          </button>
        </div>
      </div>

      <div v-if="optimizePanelVisible" class="optimize-panel">
        <div class="optimize-header">
          <div class="optimize-title">{{ $t('composer.optimizePanelTitle') }}</div>
          <button class="panel-close" @click="optimizePanelVisible = false">&times;</button>
        </div>

        <div v-if="optimizingPrompt" class="optimize-loading">
          <NSpin size="small" />
          <span>{{ $t('composer.optimizing') }}</span>
        </div>

        <template v-else>
          <!-- 标准单条结果 -->
          <div v-if="optimizePrimary" class="optimize-card">
            <div class="optimize-card-head">
              <strong>{{ optimizePrimary.title || $t('composer.optimizeCandidate') }}</strong>
              <span class="optimize-reason" v-if="optimizePrimary.reason">{{ optimizePrimary.reason }}</span>
            </div>
            <p class="optimize-prompt">{{ optimizePrimary.prompt }}</p>
            <div class="optimize-actions">
              <button class="pill" :disabled="!optimizeBackupPrompt" @click="undoOptimizedPrompt">撤销</button>
              <button class="pill pill-primary" @click="generateWithCandidate(optimizePrimary)">{{ $t('composer.optimizeGenerate') }}</button>
            </div>
          </div>


          <div v-else class="optimize-empty">{{ $t('composer.optimizeEmptyHint') }}</div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.composer-wrap { flex-shrink: 0; padding: 12px 24px 16px; max-width: 1100px; width: 100%; margin: 0 auto; }

.composer-card {
  background: var(--color-card-solid);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 16px;
  box-shadow: 0 4px 32px var(--color-tint-black-20), 0 0 0 1px var(--color-tint-white-02);
  transition: border-color 0.3s;
  isolation: isolate;
}
.composer-card :deep(input),
.composer-card :deep(textarea),
.composer-card :deep(button),
.composer-card :deep(label) {
  backdrop-filter: none !important;
}
.composer-card:focus-within {
  border-color: rgba(0, 202, 224, 0.2);
  box-shadow: 0 4px 32px var(--color-tint-black-20), 0 0 24px rgba(0, 202, 224, 0.1);
}

.composer-body {
  display: flex; align-items: flex-start; gap: 14px;
  padding: 16px 18px 12px; min-height: 100px;
}

.composer-input {
  flex: 1; background: transparent !important; border: none !important;
  color: var(--color-text-primary); font-size: 15px; line-height: 1.65;
  resize: none; padding: 0 !important; min-height: 80px; max-height: 200px;
  outline: none !important; font-family: inherit;
  box-shadow: none !important; backdrop-filter: none !important;
  border-radius: 0;
}
.composer-input:focus { background: transparent !important; border: none !important; box-shadow: none !important; outline: none !important; }
.composer-input:hover { background: transparent !important; border: none !important; }
.composer-input::placeholder { color: var(--color-text-muted); }

/* Input wrapper with floating actions */
.input-wrapper {
  position: relative;
  flex: 1;
  display: flex;
  flex-direction: column;
}
.input-wrapper .composer-input {
  padding-bottom: 32px !important;  /* 为底部按钮留出空间 */
}
.input-actions {
  position: absolute;
  bottom: 4px;
  right: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  opacity: 0.8;
  transition: opacity 0.2s;
}
.input-wrapper:hover .input-actions {
  opacity: 1;
}
.action-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  height: 22px;
  padding: 0 8px;
  font-size: 11px;
  font-weight: 500;
  background: rgba(255, 255, 255, 0.08);
  border: none;
  border-radius: 11px;
  color: rgba(255, 255, 255, 0.5);
  cursor: pointer;
  transition: all 0.2s ease;
  backdrop-filter: blur(4px);
}
.action-chip:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.8);
  transform: translateY(-1px);
}
.action-chip:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
.optimize-chip {
  background: linear-gradient(135deg, rgba(0, 202, 224, 0.15), rgba(0, 202, 224, 0.08));
  color: rgba(0, 202, 224, 0.8);
  padding: 0 8px 0 6px;
  gap: 4px;
}
.optimize-chip:hover:not(:disabled) {
  background: linear-gradient(135deg, rgba(0, 202, 224, 0.25), rgba(0, 202, 224, 0.15));
  color: #00cae0;
  box-shadow: 0 2px 8px rgba(0, 202, 224, 0.2);
}
.chip-diamond {
  opacity: 0.9;
  margin-right: 0;
}
.optimize-cost {
  font-size: 10px;
  font-weight: 600;
  margin-left: -1px;
  margin-right: 3px;
}

/* Upload cards */
.composer-uploads { display: flex; gap: 10px; align-items: center; flex-shrink: 0; }
.upload-card {
  position: relative; width: 64px; height: 64px;
  border-radius: 10px; overflow: visible;
  transform: rotate(var(--tilt, 0deg));
  transition: transform .25s ease; flex-shrink: 0;
}
.upload-card:hover { transform: rotate(0deg) scale(1.05); }
.upload-card img {
  width: 100%; height: 100%; object-fit: cover; border-radius: 10px;
  border: 2px solid var(--color-tint-white-10);
  box-shadow: 0 4px 12px var(--color-tint-black-25);
}
/* Stacked uploads */
.upload-stack {
  position: relative; cursor: pointer; flex-shrink: 0;
  width: 64px; height: 64px;
}
.stack-card {
  position: absolute; width: 64px; height: 64px;
  left: 0; top: 0;
  transform: rotate(var(--stack-rotate, 0deg));
  transition: all .25s ease;
}
.stack-card img {
  width: 100%; height: 100%; object-fit: cover; border-radius: 10px;
  border: 2px solid var(--color-tint-white-10);
  box-shadow: 0 4px 12px var(--color-tint-black-25);
}
.upload-stack:hover .stack-card {
  filter: brightness(1.1);
}
.stack-badge {
  position: absolute; bottom: -6px; right: -6px; z-index: 10;
  min-width: 20px; height: 20px; padding: 0 5px;
  border-radius: 10px; background: #00cae0; color: #fff;
  font-size: 11px; font-weight: 700; line-height: 20px; text-align: center;
  box-shadow: 0 2px 6px rgba(0, 202, 224, 0.4);
}
.stack-collapse-btn {
  width: 24px; height: 24px; border-radius: 50%;
  background: var(--color-tint-white-06); border: 1px solid var(--color-tint-white-10);
  color: var(--color-text-muted); font-size: 14px; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0; transition: all .2s;
}
.stack-collapse-btn:hover { background: var(--color-tint-white-10); color: var(--color-text-primary); }

.add-icon { font-size: 22px; color: var(--color-text-muted); font-weight: 300; line-height: 1; }
.card-remove {
  position: absolute; top: -7px; right: -7px; z-index: 2;
  width: 20px; height: 20px; border-radius: 50%;
  background: var(--color-error); color: white;
  border: 2px solid var(--color-bg-card); font-size: 10px; cursor: pointer;
  display: flex; align-items: center; justify-content: center; line-height: 1;
}
.frame-placeholder {
  width: 100%; height: 100%;
  border: 2px dashed var(--color-tint-white-12); border-radius: 10px;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 2px; cursor: pointer; transition: border-color .2s;
}
.frame-placeholder:hover { border-color: rgba(102,126,234,.4); }
.frame-lbl { font-size: 11px; color: var(--color-text-muted); }
.frame-sep { color: var(--color-text-muted); font-size: 16px; margin: 0 2px; flex-shrink: 0; }

/* Toolbar */
.composer-toolbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 18px 10px; gap: 8px;
  border-top: 1px solid var(--color-tint-white-05);
}
.toolbar-left { display: flex; gap: 4px; flex-wrap: wrap; align-items: center; }
.toolbar-right { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }

.pill {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 5px 10px; background: transparent;
  border: 1px solid transparent; border-radius: 8px;
  color: var(--color-text-secondary); font-size: 13px;
  cursor: pointer; white-space: nowrap; transition: all .15s;
  font-family: inherit;
}
.pill:hover { background: var(--color-input-bg-hover); color: var(--color-text-primary); }
.pill-icon { width: 16px; height: 16px; border-radius: 3px; flex-shrink: 0; }
.pill-arrow { opacity: .4; flex-shrink: 0; }
.pill-primary {
  background: rgba(0, 202, 224, 0.1); border-color: rgba(0, 202, 224, 0.25);
  border-radius: 8px; padding: 5px 14px;
  color: #00cae0; font-weight: 600;
}
.pill-primary:hover { background: rgba(0, 202, 224, 0.15); border-color: rgba(0, 202, 224, 0.35); }
.pill-active { background: rgba(0, 202, 224, 0.12); color: #00cae0; }

.credits-badge { font-size: 12px; color: var(--color-text-muted); white-space: nowrap; letter-spacing: .02em; display: inline-flex; align-items: center; gap: 3px; }
.credits-icon { opacity: 0.7; vertical-align: middle; }
.send-btn {
  width: 38px; height: 38px; border-radius: 50%;
  background: #00cae0; color: white; border: none;
  cursor: pointer; display: flex; align-items: center; justify-content: center;
  flex-shrink: 0; transition: all .25s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 4px 16px rgba(0, 202, 224, 0.35);
}
.send-btn:hover:not(:disabled) { transform: scale(1.08) translateY(-1px); box-shadow: 0 6px 24px rgba(0, 202, 224, 0.45); }
.send-btn:disabled { opacity: .15; cursor: not-allowed; background: var(--color-tint-white-15); box-shadow: none; }
.input-hint { text-align: center; font-size: 11px; color: var(--color-tint-white-18); margin-top: 6px; }

.optimize-panel {
  border-top: 1px solid var(--color-tint-white-05);
  padding: 12px 16px 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.optimize-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.optimize-title {
  font-size: 13px;
  color: var(--color-text-primary);
  font-weight: 600;
}
.panel-close {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 7px;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: 16px;
}
.panel-close:hover {
  color: var(--color-text-primary);
  background: var(--color-tint-white-06);
}
.optimize-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.optimize-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--color-text-muted);
  font-size: 12px;
}
.optimize-card {
  border: 1px solid var(--color-tint-white-08);
  border-radius: 12px;
  padding: 10px 12px;
  background: var(--color-tint-white-02);
}
.optimize-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
}
.optimize-reason {
  font-size: 12px;
  color: var(--color-text-muted);
}
.optimize-prompt {
  margin: 0;
  font-size: 13px;
  line-height: 1.55;
  color: var(--color-text-secondary);
  white-space: pre-wrap;
}
.optimize-actions {
  display: flex;
  gap: 6px;
  margin-top: 10px;
  flex-wrap: wrap;
}
.optimize-empty {
  color: var(--color-text-muted);
  font-size: 12px;
}

/* Popover panel */
.pop-panel {
  background: var(--color-card-solid);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 16px; padding: 18px 20px;
  min-width: 280px;
  box-shadow: 0 12px 48px var(--color-tint-black-50), 0 0 0 1px var(--color-tint-white-03);
}
.pop-section-title { font-size: 12px; color: var(--color-text-muted); margin-bottom: 10px; }
.ratio-grid { display: flex; flex-wrap: wrap; gap: 6px; }
.ratio-cell {
  display: flex; flex-direction: column; align-items: center; gap: 4px;
  width: 52px; padding: 8px 0 6px;
  border-radius: 10px; border: 1.5px solid var(--color-tint-white-08);
  background: transparent; color: var(--color-text-secondary);
  cursor: pointer; transition: all .2s; font-family: inherit;
}
.ratio-cell:hover { border-color: var(--color-tint-white-20); background: var(--color-tint-white-04); }
.ratio-cell.active { border-color: rgba(0, 202, 224, 0.5); background: rgba(0, 202, 224, 0.1); color: #00cae0; }
.ratio-icon { border: 1.5px solid currentColor; border-radius: 3px; display: block; transition: inherit; }
.ratio-label { font-size: 11px; }
.seg-group { display: flex; border: 1px solid var(--color-tint-white-10); border-radius: 10px; overflow: hidden; }
.seg-btn {
  flex: 1; padding: 7px 0; background: transparent;
  border: none; color: var(--color-text-secondary); font-size: 13px;
  cursor: pointer; transition: all .2s; display: flex;
  align-items: center; justify-content: center; gap: 4px; font-family: inherit;
}
.seg-btn:not(:last-child) { border-right: 1px solid var(--color-tint-white-10); }
.seg-btn:hover { background: var(--color-tint-white-04); }
.seg-btn.active { background: rgba(0, 202, 224, 0.15); color: #00cae0; font-weight: 600; }
.seg-badge { font-size: 11px; opacity: .7; }
.seg-star { color: #00cae0; font-size: 11px; }

@media (max-width: 768px) {
  .composer-wrap { padding: 8px 10px 12px; }
  .composer-body { flex-wrap: wrap; padding: 12px 14px 10px; gap: 10px; min-height: 70px; }
  .composer-input { min-height: 56px; font-size: 14px; }
  .composer-uploads { gap: 8px; }
  .upload-card { width: 56px; height: 56px; }
  .upload-stack { width: 56px; height: 56px; }
  .stack-card { width: 56px; height: 56px; }
  .card-remove { width: 26px; height: 26px; font-size: 12px; top: -8px; right: -8px; }
  .composer-toolbar { padding: 6px 10px 8px; gap: 4px; }
  .toolbar-left {
    gap: 3px;
    flex-wrap: nowrap;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: none;
    max-width: calc(100vw - 110px);
  }
  .toolbar-left::-webkit-scrollbar { display: none; }
  .pill { padding: 6px 10px; font-size: 12px; min-height: 34px; flex-shrink: 0; }
  .pill-primary { padding: 6px 12px; }
  .pop-panel {
    min-width: unset;
    width: calc(100vw - 40px);
    max-width: 320px;
    padding: 14px 16px;
    border-radius: 14px;
  }
  .ratio-grid { gap: 5px; justify-content: flex-start; }
  .ratio-cell { width: 52px; min-height: 52px; padding: 8px 0 6px; }
  .seg-btn { padding: 10px 0; font-size: 13px; min-height: 40px; }
  .send-btn { width: 36px; height: 36px; }
  .credits-badge { font-size: 11px; }
  .input-hint { display: none; }
  .frame-sep { font-size: 14px; }
  .optimize-panel { padding: 10px 10px 12px; }
  .optimize-card { padding: 9px 10px; }
  .optimize-card-head { flex-direction: column; align-items: flex-start; gap: 4px; }
}

@media (max-width: 380px) {
  .composer-wrap { padding: 6px 6px 10px; }
  .composer-body { padding: 10px 12px 8px; }
  .composer-toolbar { padding: 5px 8px 6px; }
  .pill { padding: 5px 8px; font-size: 11px; }
  .pill-primary { padding: 5px 10px; }
  .toolbar-left { max-width: calc(100vw - 96px); }
  .upload-card { width: 48px; height: 48px; }
  .upload-stack { width: 48px; height: 48px; }
  .stack-card { width: 48px; height: 48px; }
}
</style>
