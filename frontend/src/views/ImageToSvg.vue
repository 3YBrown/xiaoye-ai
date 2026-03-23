<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import { NSpin, NSlider, NButton, useMessage } from 'naive-ui'
import ImageTracer from 'imagetracerjs'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const message = useMessage()

const originalSrc = ref(null)
const svgString = ref('')
const converting = ref(false)
const fileInputRef = ref(null)
const dragOver = ref(false)

const params = reactive({
  numberofcolors: 16,
  blurradius: 0,
  ltres: 1,
  qtres: 1,
  pathomit: 8
})

const presets = [
  { key: 'default', label: 'tools.svg.presetDefault', values: { numberofcolors: 16, blurradius: 0, ltres: 1, qtres: 1, pathomit: 8 } },
  { key: 'detailed', label: 'tools.svg.presetDetailed', values: { numberofcolors: 64, blurradius: 0, ltres: 0.5, qtres: 0.5, pathomit: 0 } },
  { key: 'simple', label: 'tools.svg.presetSimple', values: { numberofcolors: 8, blurradius: 5, ltres: 1, qtres: 1, pathomit: 20 } },
  { key: 'poster', label: 'tools.svg.presetPoster', values: { numberofcolors: 4, blurradius: 5, ltres: 1, qtres: 1, pathomit: 8 } }
]

const activePreset = ref('default')

const applyPreset = (preset) => {
  activePreset.value = preset.key
  Object.assign(params, preset.values)
}

const svgFileSize = () => {
  if (!svgString.value) return ''
  const bytes = new Blob([svgString.value]).size
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

const loadImageFromUrl = async (url) => {
  svgString.value = ''
  // Convert external URL to base64 to avoid canvas cross-origin issues
  try {
    const resp = await fetch(url)
    const blob = await resp.blob()
    const reader = new FileReader()
    reader.onload = (e) => {
      originalSrc.value = e.target.result
    }
    reader.readAsDataURL(blob)
  } catch {
    // Fallback: use URL directly (may fail on convert due to CORS)
    originalSrc.value = url
  }
}

const handleFileSelect = (e) => {
  const file = e.target?.files?.[0]
  if (!file) return
  readFile(file)
  // Reset so selecting the same file again triggers change
  e.target.value = ''
}

const readFile = (file) => {
  if (!file.type.startsWith('image/')) return
  const reader = new FileReader()
  reader.onload = (e) => {
    originalSrc.value = e.target.result
    svgString.value = ''
  }
  reader.readAsDataURL(file)
}

const handleDrop = (e) => {
  e.preventDefault()
  dragOver.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file) readFile(file)
}

const handleDragOver = (e) => {
  e.preventDefault()
  dragOver.value = true
}

const handleDragLeave = () => {
  dragOver.value = false
}

const openFilePicker = () => {
  fileInputRef.value?.click()
}

const convert = () => {
  if (!originalSrc.value || converting.value) return
  converting.value = true
  svgString.value = ''

  const options = {
    numberofcolors: params.numberofcolors,
    blurradius: params.blurradius,
    ltres: params.ltres,
    qtres: params.qtres,
    pathomit: params.pathomit,
    scale: 1,
    roundcoords: 2,
    desc: false,
    viewbox: true
  }

  // Use setTimeout to keep UI responsive
  setTimeout(() => {
    try {
      ImageTracer.imageToSVG(
        originalSrc.value,
        (svg) => {
          svgString.value = svg
          converting.value = false
        },
        options
      )
    } catch (e) {
      converting.value = false
      message.error(e.message || 'Conversion failed')
    }
  }, 50)
}

const downloadSvg = () => {
  if (!svgString.value) return
  const blob = new Blob([svgString.value], { type: 'image/svg+xml' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `converted-${Date.now()}.svg`
  a.click()
  URL.revokeObjectURL(url)
}

const copySvgCode = async () => {
  if (!svgString.value) return
  try {
    await navigator.clipboard.writeText(svgString.value)
    message.success(t('tools.svg.copied'))
  } catch {
    message.error(t('tools.svg.copyFailed'))
  }
}

const goBack = () => {
  router.push({ name: 'tools' })
}

onMounted(() => {
  const src = route.query.src
  if (src) {
    loadImageFromUrl(src)
  }
})
</script>

<template>
  <div class="svg-page">
    <div class="svg-topbar">
      <button class="back-btn" @click="goBack">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="15 18 9 12 15 6"/>
        </svg>
        <span>{{ $t('tools.svg.back') }}</span>
      </button>
      <h2 class="svg-page-title">{{ $t('tools.imageToSvg') }}</h2>
    </div>

    <div class="svg-main">
      <!-- Left: Original image -->
      <div class="svg-panel">
        <div class="panel-label">{{ $t('tools.svg.original') }}</div>
        <div
          v-if="!originalSrc"
          class="upload-zone"
          :class="{ 'drag-over': dragOver }"
          @click="openFilePicker"
          @drop="handleDrop"
          @dragover="handleDragOver"
          @dragleave="handleDragLeave"
        >
          <svg class="upload-icon" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="17 8 12 3 7 8"/>
            <line x1="12" y1="3" x2="12" y2="15"/>
          </svg>
          <p class="upload-text">{{ $t('tools.svg.upload') }}</p>
          <p class="upload-hint">{{ $t('tools.svg.uploadHint') }}</p>
        </div>
        <div v-else class="preview-area">
          <img :src="originalSrc" class="preview-img" alt="original" />
          <button class="change-btn" @click="openFilePicker">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="17 8 12 3 7 8"/>
              <line x1="12" y1="3" x2="12" y2="15"/>
            </svg>
          </button>
        </div>
        <input
          ref="fileInputRef"
          type="file"
          accept="image/png,image/jpeg,image/bmp"
          style="display:none"
          @change="handleFileSelect"
        />
      </div>

      <!-- Right: SVG Preview -->
      <div class="svg-panel">
        <div class="panel-label">{{ $t('tools.svg.result') }}</div>
        <div v-if="converting" class="preview-area loading-area">
          <NSpin size="medium" />
          <span class="converting-text">{{ $t('tools.svg.converting') }}</span>
        </div>
        <div v-else-if="svgString" class="preview-area svg-preview">
          <div class="svg-render" v-html="svgString"></div>
          <div class="svg-size-badge">{{ svgFileSize() }}</div>
        </div>
        <div v-else class="preview-area empty-preview">
          <span class="empty-text">{{ $t('tools.svg.noResult') }}</span>
        </div>
      </div>
    </div>

    <!-- Params + Actions -->
    <div class="svg-controls">
      <div class="params-row">
        <div class="preset-group">
          <span class="param-label">{{ $t('tools.svg.preset') }}</span>
          <div class="preset-chips">
            <button
              v-for="p in presets"
              :key="p.key"
              :class="['preset-chip', { active: activePreset === p.key }]"
              @click="applyPreset(p)"
            >{{ $t(p.label) }}</button>
          </div>
        </div>
        <div class="slider-group">
          <span class="param-label">{{ $t('tools.svg.colors') }}: {{ params.numberofcolors }}</span>
          <NSlider v-model:value="params.numberofcolors" :min="2" :max="64" :step="1" :tooltip="false" @update:value="activePreset = ''" />
        </div>
        <div class="slider-group">
          <span class="param-label">{{ $t('tools.svg.blur') }}: {{ params.blurradius }}</span>
          <NSlider v-model:value="params.blurradius" :min="0" :max="10" :step="1" :tooltip="false" @update:value="activePreset = ''" />
        </div>
        <div class="slider-group">
          <span class="param-label">{{ $t('tools.svg.simplify') }}: {{ params.pathomit }}</span>
          <NSlider v-model:value="params.pathomit" :min="0" :max="40" :step="1" :tooltip="false" @update:value="activePreset = ''" />
        </div>
      </div>
      <div class="action-row">
        <NButton
          type="primary"
          :disabled="!originalSrc || converting"
          :loading="converting"
          @click="convert"
        >
          {{ svgString ? $t('tools.svg.reconvert') : $t('tools.svg.convertBtn') }}
        </NButton>
        <NButton :disabled="!svgString" @click="downloadSvg">
          {{ $t('tools.svg.download') }}
        </NButton>
        <NButton :disabled="!svgString" @click="copySvgCode">
          {{ $t('tools.svg.copyCode') }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.svg-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.svg-topbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 24px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--color-tint-white-06);
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 10px;
  border: 1px solid var(--color-tint-white-08);
  background: transparent;
  border-radius: 8px;
  color: var(--color-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

.back-btn:hover {
  border-color: rgba(0, 202, 224, 0.35);
  color: #00cae0;
}

.svg-page-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.svg-main {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  padding: 16px 24px;
  min-height: 0;
  overflow: hidden;
}

.svg-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.panel-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-muted);
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.upload-zone {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border: 2px dashed var(--color-tint-white-12);
  border-radius: 16px;
  cursor: pointer;
  transition: all 0.25s;
  min-height: 200px;
}

.upload-zone:hover,
.upload-zone.drag-over {
  border-color: rgba(0, 202, 224, 0.5);
  background: rgba(0, 202, 224, 0.04);
}

.upload-icon {
  color: var(--color-text-muted);
  opacity: 0.6;
}

.upload-text {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin: 0;
}

.upload-hint {
  font-size: 12px;
  color: var(--color-text-muted);
  margin: 0;
}

.preview-area {
  flex: 1;
  border: 1px solid var(--color-tint-white-06);
  border-radius: 16px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  background: var(--color-tint-white-02);
  min-height: 200px;
}

.preview-img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.change-btn {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 32px;
  height: 32px;
  border: 1px solid rgba(255, 255, 255, 0.25);
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.change-btn:hover {
  background: rgba(0, 202, 224, 0.3);
  border-color: rgba(0, 202, 224, 0.5);
}

.loading-area {
  flex-direction: column;
  gap: 12px;
}

.converting-text {
  font-size: 13px;
  color: var(--color-text-muted);
}

.svg-preview {
  overflow: auto;
}

.svg-render {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.svg-render :deep(svg) {
  max-width: 100%;
  max-height: 100%;
}

.svg-size-badge {
  position: absolute;
  bottom: 8px;
  right: 8px;
  font-size: 11px;
  padding: 3px 8px;
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  border-radius: 6px;
  backdrop-filter: blur(4px);
}

.empty-preview {
  border-style: dashed;
}

.empty-text {
  font-size: 13px;
  color: var(--color-text-muted);
  opacity: 0.6;
}

.svg-controls {
  flex-shrink: 0;
  padding: 16px 24px 20px;
  border-top: 1px solid var(--color-tint-white-06);
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.params-row {
  display: flex;
  gap: 20px;
  align-items: flex-start;
  flex-wrap: wrap;
}

.preset-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.param-label {
  font-size: 12px;
  color: var(--color-text-muted);
  font-weight: 500;
  white-space: nowrap;
}

.preset-chips {
  display: flex;
  gap: 4px;
}

.preset-chip {
  padding: 4px 12px;
  border: 1px solid var(--color-tint-white-08);
  background: transparent;
  border-radius: 8px;
  color: var(--color-text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

.preset-chip:hover {
  border-color: var(--color-tint-white-15);
  background: var(--color-tint-white-04);
}

.preset-chip.active {
  background: rgba(0, 202, 224, 0.1);
  border-color: rgba(0, 202, 224, 0.3);
  color: #00cae0;
  font-weight: 600;
}

.slider-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 120px;
  flex: 1;
  max-width: 200px;
}

.action-row {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

@media (max-width: 768px) {
  .svg-topbar {
    padding: 12px 14px;
  }
  .svg-main {
    grid-template-columns: 1fr;
    padding: 12px 14px;
    overflow-y: auto;
  }
  .preview-area {
    min-height: 180px;
  }
  .svg-controls {
    padding: 12px 14px 16px;
  }
  .params-row {
    flex-direction: column;
    gap: 12px;
  }
  .slider-group {
    max-width: 100%;
  }
  .action-row {
    flex-wrap: wrap;
  }
}
</style>
