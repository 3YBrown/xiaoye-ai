<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '../stores/user'
import { usePricingStore } from '../stores/pricing'

const props = defineProps({
  show: { type: Boolean, default: false },
  imageSrc: { type: String, default: '' }
})

const emit = defineEmits(['update:show', 'submit'])

const { t } = useI18n()
const userStore = useUserStore()
const pricingStore = usePricingStore()

const canvasRef = ref(null)
const imgRef = ref(null)
const containerRef = ref(null)
const prompt = ref('')
const brushSize = ref(30)
const isDrawing = ref(false)
const hasMask = ref(false)
const generating = ref(false)
const aspectRatio = ref('1:1')
const imageSize = ref('2K')
const showRatioPanel = ref(false)

const lockedModel = 'gemini-3-pro-image-preview'

const imageRatios = [
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

const imageSizeOptions = computed(() => pricingStore.getImageSizeOptions(lockedModel))

const currentCredits = computed(() => {
  return pricingStore.getImageCredits(lockedModel, imageSize.value)
})

// Canvas state
let ctx = null
let undoStack = []
let canvasWidth = 0
let canvasHeight = 0
let displayWidth = 0
let displayHeight = 0
let scaleRatio = 1

function close() {
  emit('update:show', false)
  prompt.value = ''
  hasMask.value = false
  undoStack = []
  generating.value = false
  showRatioPanel.value = false
}

function initCanvas() {
  const img = imgRef.value
  const canvas = canvasRef.value
  const container = containerRef.value
  if (!img || !canvas || !container) return

  const naturalW = img.naturalWidth
  const naturalH = img.naturalHeight
  if (!naturalW || !naturalH) return

  canvasWidth = naturalW
  canvasHeight = naturalH

  const maxW = container.clientWidth
  const maxH = container.clientHeight
  const ratio = Math.min(maxW / naturalW, maxH / naturalH, 1)
  displayWidth = Math.round(naturalW * ratio)
  displayHeight = Math.round(naturalH * ratio)
  scaleRatio = naturalW / displayWidth

  img.style.width = displayWidth + 'px'
  img.style.height = displayHeight + 'px'
  canvas.width = naturalW
  canvas.height = naturalH
  canvas.style.width = displayWidth + 'px'
  canvas.style.height = displayHeight + 'px'

  ctx = canvas.getContext('2d')
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  clearCanvas()
}

watch(() => props.show, (val) => {
  if (val) {
    nextTick(() => {
      const img = imgRef.value
      if (img && img.complete && img.naturalWidth) {
        initCanvas()
      }
    })
  }
})

function onImageLoad() {
  initCanvas()
}

function getPos(e) {
  const canvas = canvasRef.value
  if (!canvas) return { x: 0, y: 0 }
  const rect = canvas.getBoundingClientRect()
  const clientX = e.touches ? e.touches[0].clientX : e.clientX
  const clientY = e.touches ? e.touches[0].clientY : e.clientY
  return {
    x: (clientX - rect.left) * scaleRatio,
    y: (clientY - rect.top) * scaleRatio
  }
}

function startDraw(e) {
  e.preventDefault()
  isDrawing.value = true
  const pos = getPos(e)
  ctx.beginPath()
  ctx.moveTo(pos.x, pos.y)
  ctx.strokeStyle = 'rgba(255, 60, 60, 0.45)'
  ctx.lineWidth = brushSize.value * scaleRatio
}

function draw(e) {
  if (!isDrawing.value) return
  e.preventDefault()
  const pos = getPos(e)
  ctx.lineTo(pos.x, pos.y)
  ctx.stroke()
}

function endDraw(e) {
  if (!isDrawing.value) return
  e.preventDefault()
  isDrawing.value = false
  ctx.closePath()
  saveSnapshot()
  hasMask.value = true
}

function saveSnapshot() {
  const canvas = canvasRef.value
  if (!canvas || !ctx) return
  undoStack.push(ctx.getImageData(0, 0, canvas.width, canvas.height))
  if (undoStack.length > 30) undoStack.shift()
}

function undo() {
  if (undoStack.length <= 0) return
  undoStack.pop()
  const canvas = canvasRef.value
  if (!canvas || !ctx) return
  if (undoStack.length === 0) {
    clearCanvas()
    hasMask.value = false
  } else {
    ctx.putImageData(undoStack[undoStack.length - 1], 0, 0)
  }
}

function clearCanvas() {
  const canvas = canvasRef.value
  if (!canvas || !ctx) return
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  undoStack = []
  hasMask.value = false
}

function exportMask() {
  const canvas = canvasRef.value
  if (!canvas) return ''

  const maskCanvas = document.createElement('canvas')
  maskCanvas.width = canvasWidth
  maskCanvas.height = canvasHeight
  const maskCtx = maskCanvas.getContext('2d')

  maskCtx.fillStyle = '#000000'
  maskCtx.fillRect(0, 0, canvasWidth, canvasHeight)

  const imageData = ctx.getImageData(0, 0, canvasWidth, canvasHeight)
  const data = imageData.data

  const maskImageData = maskCtx.getImageData(0, 0, canvasWidth, canvasHeight)
  const maskData = maskImageData.data
  for (let i = 0; i < data.length; i += 4) {
    if (data[i + 3] > 0) {
      maskData[i] = 255
      maskData[i + 1] = 255
      maskData[i + 2] = 255
      maskData[i + 3] = 255
    }
  }
  maskCtx.putImageData(maskImageData, 0, 0)

  return maskCanvas.toDataURL('image/png')
}

async function handleGenerate() {
  if (!hasMask.value) return
  if (!prompt.value.trim()) return
  generating.value = true

  const maskBase64 = exportMask()
  emit('submit', {
    originalImageUrl: props.imageSrc,
    maskBase64,
    prompt: prompt.value.trim(),
    aspectRatio: aspectRatio.value,
    imageSize: imageSize.value
  })
}

function handleKeydown(e) {
  if (!props.show) return
  if (e.key === 'Escape') close()
  if ((e.ctrlKey || e.metaKey) && e.key === 'z') {
    e.preventDefault()
    undo()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})

defineExpose({ setGenerating: (v) => { generating.value = v } })
</script>

<template>
  <Teleport to="body">
    <Transition name="editor-fade">
      <div v-if="show" class="editor-overlay" @click.self="close">
        <div class="editor-card">
          <button class="editor-close" @click="close" :title="t('common.cancel')">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6L6 18"/><path d="M6 6l12 12"/></svg>
          </button>

          <div class="editor-layout">
            <!-- Left: Canvas area -->
            <div class="editor-canvas-area">
              <div class="editor-canvas-container" ref="containerRef">
                <img
                  ref="imgRef"
                  :src="imageSrc"
                  crossorigin="anonymous"
                  @load="onImageLoad"
                  class="editor-image"
                  draggable="false"
                />
                <canvas
                  ref="canvasRef"
                  class="editor-canvas"
                  @mousedown="startDraw"
                  @mousemove="draw"
                  @mouseup="endDraw"
                  @mouseleave="endDraw"
                  @touchstart="startDraw"
                  @touchmove="draw"
                  @touchend="endDraw"
                />
              </div>

              <div class="editor-toolbar">
                <label class="brush-label">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 19l7-7 3 3-7 7-3-3z"/><path d="M18 13l-1.5-7.5L2 2l3.5 14.5L13 18l5-5z"/><path d="M2 2l7.586 7.586"/><circle cx="11" cy="11" r="2"/></svg>
                  <span>{{ t('editor.brushSize') }}</span>
                </label>
                <input type="range" v-model.number="brushSize" min="5" max="100" class="brush-slider" />
                <span class="brush-value">{{ brushSize }}px</span>
                <div class="toolbar-spacer"></div>
                <button class="toolbar-btn" @click="undo" :disabled="undoStack.length === 0" :title="t('editor.undo')">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 7v6h6"/><path d="M21 17a9 9 0 0 0-9-9 9 9 0 0 0-6 2.3L3 13"/></svg>
                  {{ t('editor.undo') }}
                </button>
                <button class="toolbar-btn" @click="clearCanvas" :title="t('editor.clear')">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>
                  {{ t('editor.clear') }}
                </button>
              </div>
            </div>

            <!-- Right: Controls -->
            <div class="editor-controls">
              <h3 class="editor-title">{{ t('editor.title') }}</h3>

              <!-- Model pill (locked) -->
              <div class="editor-param-row">
                <button class="param-pill model-pill locked" disabled>
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                  Nanobanana Pro
                </button>
              </div>

              <!-- Ratio & Size selector -->
              <div class="editor-param-row">
                <button class="param-pill" @click="showRatioPanel = !showRatioPanel">
                  <span>▢ {{ aspectRatio }}</span>
                  <span class="pill-sep">·</span>
                  <span>{{ imageSize }}</span>
                  <span v-if="currentCredits" class="pill-badge">{{ currentCredits }}💎</span>
                  <svg class="pill-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M6 9l6 6 6-6"/></svg>
                </button>
              </div>

              <!-- Ratio/Size panel (inline) -->
              <div v-if="showRatioPanel" class="ratio-size-panel">
                <div class="panel-section-title">{{ t('composer.selectRatio') }}</div>
                <div class="ratio-grid">
                  <button
                    v-for="r in imageRatios" :key="r.value"
                    :class="['ratio-cell', { active: aspectRatio === r.value }]"
                    @click="aspectRatio = r.value"
                  >
                    <span class="ratio-icon" :style="{ width: r.w + 'px', height: r.h + 'px' }"></span>
                    <span class="ratio-label">{{ r.value }}</span>
                  </button>
                </div>
                <div class="panel-section-title" style="margin-top: 12px;">{{ t('composer.selectResolution') }}</div>
                <div class="seg-group">
                  <button
                    v-for="s in imageSizeOptions" :key="s.value"
                    :class="['seg-btn', { active: imageSize === s.value }]"
                    @click="imageSize = s.value"
                  >
                    {{ s.label }}
                    <span v-if="s.badge" class="seg-badge">{{ s.badge }}</span>
                  </button>
                </div>
              </div>

              <!-- Credits display -->
              <div class="editor-credits" v-if="userStore.user">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>
                <span>{{ t('editor.balance') }}: {{ userStore.user.credits }}</span>
              </div>

              <textarea
                v-model="prompt"
                class="editor-prompt"
                :placeholder="t('editor.promptPlaceholder')"
                rows="4"
              />

              <div v-if="!hasMask" class="editor-hint">{{ t('editor.noMask') }}</div>
              <div v-else-if="!prompt.trim()" class="editor-hint">{{ t('editor.noPrompt') }}</div>

              <button
                class="editor-generate-btn"
                :disabled="!hasMask || !prompt.trim() || generating"
                @click="handleGenerate"
              >
                <template v-if="generating">
                  <span class="spinner"></span>
                  {{ t('generate.generating') }}
                </template>
                <template v-else>
                  {{ t('editor.generate') }}
                  <span v-if="currentCredits" class="gen-cost">{{ currentCredits }}💎</span>
                </template>
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.editor-overlay {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
}

.editor-card {
  position: relative;
  width: 90vw;
  max-width: 1100px;
  max-height: 85vh;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-tint-white-08);
  border-radius: 16px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.editor-close {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 10;
  width: 32px;
  height: 32px;
  border: 1px solid var(--color-tint-white-10);
  background: var(--color-tint-white-05);
  color: var(--color-text-secondary);
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}
.editor-close:hover {
  background: var(--color-tint-white-10);
  color: var(--color-text-primary);
}

.editor-layout {
  display: flex;
  height: 100%;
  min-height: 0;
}

/* Canvas area */
.editor-canvas-area {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding: 20px;
  padding-right: 0;
}

.editor-canvas-container {
  flex: 1;
  min-height: 0;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.3);
}

.editor-image {
  display: block;
  border-radius: 8px;
  user-select: none;
  pointer-events: none;
}

.editor-canvas {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  cursor: crosshair;
  touch-action: none;
  border-radius: 8px;
}

.editor-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 0 4px;
  flex-shrink: 0;
}

.brush-label {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--color-text-secondary);
  white-space: nowrap;
}

.brush-slider {
  width: 120px;
  accent-color: #00cae0;
}

.brush-value {
  font-size: 12px;
  color: var(--color-text-muted);
  min-width: 36px;
}

.toolbar-spacer {
  flex: 1;
}

.toolbar-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border: 1px solid var(--color-tint-white-10);
  background: var(--color-tint-white-05);
  color: var(--color-text-secondary);
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}
.toolbar-btn:hover:not(:disabled) {
  background: var(--color-tint-white-10);
  color: var(--color-text-primary);
}
.toolbar-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* Controls panel */
.editor-controls {
  width: 280px;
  flex-shrink: 0;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  border-left: 1px solid var(--color-tint-white-06);
  overflow-y: auto;
}

.editor-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

/* Param pills */
.editor-param-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.param-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 6px 12px;
  border: 1px solid var(--color-tint-white-10);
  background: var(--color-tint-white-05);
  color: var(--color-text-secondary);
  border-radius: 20px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
  white-space: nowrap;
}
.param-pill:hover:not(:disabled) {
  background: var(--color-tint-white-10);
  color: var(--color-text-primary);
}
.param-pill.locked {
  opacity: 0.6;
  cursor: not-allowed;
}
.pill-sep {
  opacity: 0.4;
}
.pill-badge {
  font-size: 11px;
  opacity: 0.7;
}
.pill-arrow {
  opacity: 0.5;
  flex-shrink: 0;
}

/* Ratio/Size inline panel */
.ratio-size-panel {
  padding: 12px;
  background: var(--color-tint-white-04);
  border: 1px solid var(--color-tint-white-08);
  border-radius: 12px;
}

.panel-section-title {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 8px;
}

.ratio-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 4px;
}

.ratio-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  padding: 6px 2px;
  border: 1px solid transparent;
  background: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
  font-family: inherit;
}
.ratio-cell:hover {
  background: var(--color-tint-white-06);
}
.ratio-cell.active {
  border-color: #00cae0;
  background: rgba(0, 202, 224, 0.1);
}

.ratio-icon {
  border: 1.5px solid var(--color-text-muted);
  border-radius: 2px;
  transition: border-color 0.15s;
}
.ratio-cell.active .ratio-icon {
  border-color: #00cae0;
}

.ratio-label {
  font-size: 10px;
  color: var(--color-text-muted);
}
.ratio-cell.active .ratio-label {
  color: #00cae0;
}

.seg-group {
  display: flex;
  gap: 6px;
}

.seg-btn {
  flex: 1;
  padding: 6px 8px;
  border: 1px solid var(--color-tint-white-10);
  background: var(--color-tint-white-04);
  color: var(--color-text-secondary);
  border-radius: 8px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
  text-align: center;
  font-family: inherit;
}
.seg-btn:hover {
  background: var(--color-tint-white-08);
}
.seg-btn.active {
  border-color: #00cae0;
  background: rgba(0, 202, 224, 0.12);
  color: #00cae0;
}
.seg-badge {
  font-size: 10px;
  opacity: 0.7;
  margin-left: 2px;
}

.editor-credits {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.editor-prompt {
  width: 100%;
  min-height: 80px;
  padding: 10px 12px;
  border: 1px solid var(--color-tint-white-10);
  background: var(--color-tint-white-04);
  color: var(--color-text-primary);
  border-radius: 10px;
  font-size: 14px;
  line-height: 1.5;
  resize: vertical;
  font-family: inherit;
  transition: border-color 0.2s;
}
.editor-prompt:focus {
  outline: none;
  border-color: rgba(0, 202, 224, 0.4);
}
.editor-prompt::placeholder {
  color: var(--color-text-muted);
}

.editor-hint {
  font-size: 12px;
  color: var(--color-text-muted);
  padding: 2px 0;
}

.editor-generate-btn {
  width: 100%;
  padding: 12px;
  border: none;
  background: #00cae0;
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-family: inherit;
  margin-top: auto;
}
.editor-generate-btn:hover:not(:disabled) {
  background: #00b8cc;
}
.editor-generate-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.gen-cost {
  font-size: 13px;
  font-weight: 400;
  opacity: 0.8;
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Transitions */
.editor-fade-enter-active,
.editor-fade-leave-active {
  transition: opacity 0.25s ease;
}
.editor-fade-enter-from,
.editor-fade-leave-to {
  opacity: 0;
}

/* Mobile */
@media (max-width: 768px) {
  .editor-card {
    width: 100vw;
    max-width: 100vw;
    height: 100vh;
    max-height: 100vh;
    border-radius: 0;
  }

  .editor-layout {
    flex-direction: column;
  }

  .editor-canvas-area {
    flex: 1;
    padding: 12px;
    padding-bottom: 0;
  }

  .editor-controls {
    width: 100%;
    border-left: none;
    border-top: 1px solid var(--color-tint-white-06);
    padding: 14px;
    gap: 10px;
    max-height: 40vh;
  }

  .editor-toolbar {
    flex-wrap: wrap;
    gap: 8px;
  }

  .brush-slider {
    width: 80px;
  }

  .ratio-grid {
    grid-template-columns: repeat(5, 1fr);
  }
}
</style>
