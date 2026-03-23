<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NSpin, NSlider, NButton, useMessage } from 'naive-ui'

const { t } = useI18n()
const router = useRouter()
const message = useMessage()

const originalSrc = ref(null)
const originalFile = ref(null)
const resultBlob = ref(null)
const resultUrl = ref(null)
const converting = ref(false)
const fileInputRef = ref(null)
const dragOver = ref(false)

const outputFormat = ref('webp')
const quality = ref(80)

const formats = [
  { value: 'png', label: 'PNG' },
  { value: 'jpeg', label: 'JPG' },
  { value: 'webp', label: 'WebP' }
]

const showQuality = computed(() => outputFormat.value !== 'png')

const originalSize = computed(() => {
  if (!originalFile.value) return ''
  return formatBytes(originalFile.value.size)
})

const convertedSize = computed(() => {
  if (!resultBlob.value) return ''
  return formatBytes(resultBlob.value.size)
})

const compressionRatio = computed(() => {
  if (!originalFile.value || !resultBlob.value) return ''
  const ratio = ((1 - resultBlob.value.size / originalFile.value.size) * 100).toFixed(1)
  return ratio > 0 ? `-${ratio}%` : `+${Math.abs(ratio)}%`
})

function formatBytes(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

const handleFileSelect = (e) => {
  const file = e.target?.files?.[0]
  if (!file) return
  readFile(file)
  e.target.value = ''
}

const readFile = (file) => {
  if (!file.type.startsWith('image/')) return
  originalFile.value = file
  const reader = new FileReader()
  reader.onload = (e) => {
    originalSrc.value = e.target.result
    clearResult()
  }
  reader.readAsDataURL(file)
}

const clearResult = () => {
  if (resultUrl.value) URL.revokeObjectURL(resultUrl.value)
  resultBlob.value = null
  resultUrl.value = null
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
  clearResult()

  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')
  const img = new Image()
  img.onload = () => {
    canvas.width = img.naturalWidth
    canvas.height = img.naturalHeight
    ctx.drawImage(img, 0, 0)
    const mimeType = `image/${outputFormat.value}`
    const q = outputFormat.value === 'png' ? undefined : quality.value / 100
    canvas.toBlob((blob) => {
      if (!blob) {
        converting.value = false
        message.error('Conversion failed')
        return
      }
      resultBlob.value = blob
      resultUrl.value = URL.createObjectURL(blob)
      converting.value = false
    }, mimeType, q)
  }
  img.onerror = () => {
    converting.value = false
    message.error('Failed to load image')
  }
  img.src = originalSrc.value
}

const download = () => {
  if (!resultUrl.value) return
  const ext = outputFormat.value === 'jpeg' ? 'jpg' : outputFormat.value
  const a = document.createElement('a')
  a.href = resultUrl.value
  a.download = `converted-${Date.now()}.${ext}`
  a.click()
}

const goBack = () => {
  router.push({ name: 'tools' })
}
</script>

<template>
  <div class="convert-page">
    <div class="convert-topbar">
      <button class="back-btn" @click="goBack">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="15 18 9 12 15 6"/>
        </svg>
        <span>{{ $t('tools.convert.back') }}</span>
      </button>
      <h2 class="convert-page-title">{{ $t('tools.imageConvert.name') }}</h2>
    </div>

    <div class="convert-main">
      <!-- Left: Original image -->
      <div class="convert-panel">
        <div class="panel-label">{{ $t('tools.convert.original') }}</div>
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
          <p class="upload-text">{{ $t('tools.convert.upload') }}</p>
          <p class="upload-hint">{{ $t('tools.convert.uploadHint') }}</p>
        </div>
        <div v-else class="preview-area">
          <img :src="originalSrc" class="preview-img" alt="original" />
          <div v-if="originalSize" class="size-badge">{{ originalSize }}</div>
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
          accept="image/png,image/jpeg,image/webp,image/bmp"
          style="display:none"
          @change="handleFileSelect"
        />
      </div>

      <!-- Right: Result Preview -->
      <div class="convert-panel">
        <div class="panel-label">{{ $t('tools.convert.result') }}</div>
        <div v-if="converting" class="preview-area loading-area">
          <NSpin size="medium" />
          <span class="converting-text">{{ $t('tools.convert.converting') }}</span>
        </div>
        <div v-else-if="resultUrl" class="preview-area">
          <img :src="resultUrl" class="preview-img" alt="result" />
          <div class="size-badge-group">
            <div class="size-badge">{{ convertedSize }}</div>
            <div class="ratio-badge" :class="{ shrink: compressionRatio.startsWith('-') }">{{ compressionRatio }}</div>
          </div>
        </div>
        <div v-else class="preview-area empty-preview">
          <span class="empty-text">{{ $t('tools.convert.noResult') }}</span>
        </div>
      </div>
    </div>

    <!-- Params + Actions -->
    <div class="convert-controls">
      <div class="params-row">
        <div class="format-group">
          <span class="param-label">{{ $t('tools.convert.format') }}</span>
          <div class="format-chips">
            <button
              v-for="f in formats"
              :key="f.value"
              :class="['format-chip', { active: outputFormat === f.value }]"
              @click="outputFormat = f.value"
            >{{ f.label }}</button>
          </div>
        </div>
        <div v-if="showQuality" class="slider-group">
          <span class="param-label">{{ $t('tools.convert.quality') }}: {{ quality }}</span>
          <NSlider v-model:value="quality" :min="1" :max="100" :step="1" :tooltip="false" />
        </div>
        <div v-if="originalFile && resultBlob" class="stats-group">
          <span class="stat-item">{{ $t('tools.convert.originalSize') }}: {{ originalSize }}</span>
          <span class="stat-item">{{ $t('tools.convert.convertedSize') }}: {{ convertedSize }}</span>
          <span class="stat-item" :class="{ shrink: compressionRatio.startsWith('-') }">{{ $t('tools.convert.compressionRatio') }}: {{ compressionRatio }}</span>
        </div>
      </div>
      <div class="action-row">
        <NButton
          type="primary"
          :disabled="!originalSrc || converting"
          :loading="converting"
          @click="convert"
        >
          {{ resultUrl ? $t('tools.convert.reconvert') : $t('tools.convert.convertBtn') }}
        </NButton>
        <NButton :disabled="!resultUrl" @click="download">
          {{ $t('tools.convert.download') }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.convert-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.convert-topbar {
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

.convert-page-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.convert-main {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  padding: 16px 24px;
  min-height: 0;
  overflow: hidden;
}

.convert-panel {
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

.empty-preview {
  border-style: dashed;
}

.empty-text {
  font-size: 13px;
  color: var(--color-text-muted);
  opacity: 0.6;
}

.size-badge {
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

.size-badge-group {
  position: absolute;
  bottom: 8px;
  right: 8px;
  display: flex;
  gap: 4px;
}

.size-badge-group .size-badge {
  position: static;
}

.ratio-badge {
  font-size: 11px;
  padding: 3px 8px;
  background: rgba(0, 0, 0, 0.5);
  color: #fbbf24;
  border-radius: 6px;
  backdrop-filter: blur(4px);
}

.ratio-badge.shrink {
  color: #34d399;
}

.convert-controls {
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

.format-group {
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

.format-chips {
  display: flex;
  gap: 4px;
}

.format-chip {
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

.format-chip:hover {
  border-color: var(--color-tint-white-15);
  background: var(--color-tint-white-04);
}

.format-chip.active {
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

.stats-group {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
  margin-left: auto;
}

.stat-item {
  font-size: 12px;
  color: var(--color-text-muted);
}

.stat-item.shrink {
  color: #34d399;
}

.action-row {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

@media (max-width: 768px) {
  .convert-topbar {
    padding: 12px 14px;
  }
  .convert-main {
    grid-template-columns: 1fr;
    padding: 12px 14px;
    overflow-y: auto;
  }
  .preview-area {
    min-height: 180px;
  }
  .convert-controls {
    padding: 12px 14px 16px;
  }
  .params-row {
    flex-direction: column;
    gap: 12px;
  }
  .slider-group {
    max-width: 100%;
  }
  .stats-group {
    margin-left: 0;
  }
  .action-row {
    flex-wrap: wrap;
  }
}
</style>
