<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NButton, NSpin, NSelect, useMessage } from 'naive-ui'
import { useGenerate } from '../composables/useGenerate'
import { useUserStore } from '../stores/user'
import { useComposerDraftStore } from '../stores/composerDraft'

const { t } = useI18n()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()
const composerDraftStore = useComposerDraftStore()
const { reversePrompt } = useGenerate()

const imageSrc = ref(null)
const imageBase64 = ref(null)
const language = ref('zh')
const targetModel = ref('Nanobanana Pro')
const loading = ref(false)
const resultPrompt = ref('')
const resultMeta = ref(null)
const fileInputRef = ref(null)
const dragOver = ref(false)

const languageOptions = [
  { value: 'zh', labelKey: 'tools.reversePrompt.langZh' },
  { value: 'en', labelKey: 'tools.reversePrompt.langEn' }
]

const modelOptions = [
  { value: 'Nanobanana Pro', label: 'Nanobanana Pro' },
  { value: 'Seedream-4.5', label: 'Seedream-4.5' },
  { value: 'Midjourney', label: 'Midjourney' }
]

const readFile = (file) => {
  if (!file || !file.type.startsWith('image/')) return
  const reader = new FileReader()
  reader.onload = (e) => {
    imageSrc.value = e.target.result
    imageBase64.value = e.target.result
    resultPrompt.value = ''
    resultMeta.value = null
  }
  reader.readAsDataURL(file)
}

const handleFileSelect = (e) => {
  const file = e.target?.files?.[0]
  if (!file) return
  readFile(file)
  e.target.value = ''
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

const handlePaste = (e) => {
  const items = e.clipboardData?.items
  if (!items) return
  for (const item of items) {
    if (item.type.startsWith('image/')) {
      const file = item.getAsFile()
      if (file) readFile(file)
      break
    }
  }
}

const submit = async () => {
  if (!imageBase64.value || loading.value) return
  if (!userStore.requireAuth()) return

  loading.value = true
  resultPrompt.value = ''
  resultMeta.value = null

  try {
    const data = await reversePrompt({
      image: imageBase64.value,
      language: language.value,
      target_model: targetModel.value
    })
    resultPrompt.value = data.prompt || ''
    resultMeta.value = data.meta || null
    if (data.meta?.credits_remaining !== undefined && userStore.currentUser) {
      userStore.currentUser.credits = data.meta.credits_remaining
    }
  } catch (err) {
    const errMsg = err?.response?.data?.error || t('tools.reversePrompt.failed')
    message.error(errMsg)
    if (err?.response?.status === 402) {
      userStore.openPricing()
    }
  } finally {
    loading.value = false
  }
}

const copyPrompt = async () => {
  if (!resultPrompt.value) return
  try {
    await navigator.clipboard.writeText(resultPrompt.value)
    message.success(t('tools.reversePrompt.copied'))
  } catch {
    message.error(t('tools.reversePrompt.copyFailed'))
  }
}

const useForGenerate = () => {
  if (!resultPrompt.value) return
  composerDraftStore.setPromptDraft(resultPrompt.value)
  router.push('/generate')
}

const goBack = () => {
  router.push({ name: 'tools' })
}
</script>

<template>
  <div class="rp-page" @paste="handlePaste" tabindex="0">
    <div class="rp-topbar">
      <button class="back-btn" @click="goBack">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="15 18 9 12 15 6"/>
        </svg>
        <span>{{ $t('tools.reversePrompt.back') }}</span>
      </button>
      <h2 class="rp-page-title">{{ $t('tools.reversePrompt.title') }}</h2>
      <div class="rp-cost-badge">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
        </svg>
        <span>{{ $t('tools.reversePrompt.costPerUse') }}</span>
      </div>
    </div>

    <div class="rp-body">
      <!-- Left: Upload + Options -->
      <div class="rp-left">
        <div class="rp-section">
          <div class="section-label">{{ $t('tools.reversePrompt.uploadLabel') }}</div>
          <div
            v-if="!imageSrc"
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
            <p class="upload-text">{{ $t('tools.reversePrompt.upload') }}</p>
            <p class="upload-hint">{{ $t('tools.reversePrompt.uploadHint') }}</p>
          </div>
          <div v-else class="preview-area">
            <img :src="imageSrc" class="preview-img" alt="uploaded" />
            <button class="change-btn" @click="openFilePicker" :title="$t('tools.reversePrompt.changeImage')">
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

        <!-- Options -->
        <div class="rp-section">
          <div class="section-label">{{ $t('tools.reversePrompt.optionsLabel') }}</div>
          <div class="options-grid">
            <div class="option-group">
              <span class="option-label">{{ $t('tools.reversePrompt.language') }}</span>
              <div class="option-chips">
                <button
                  v-for="opt in languageOptions"
                  :key="opt.value"
                  :class="['chip', { active: language === opt.value }]"
                  @click="language = opt.value"
                >{{ $t(opt.labelKey) }}</button>
              </div>
            </div>
            <div class="option-group">
              <span class="option-label">{{ $t('tools.reversePrompt.targetModel') }}</span>
              <NSelect
                v-model:value="targetModel"
                :options="modelOptions"
                size="small"
                class="model-select"
              />
            </div>
          </div>
        </div>

        <!-- Submit -->
        <div class="rp-actions">
          <NButton
            type="primary"
            :disabled="!imageSrc || loading"
            :loading="loading"
            @click="submit"
            class="submit-btn"
          >
            <template #icon>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="11" cy="11" r="8"/>
                <line x1="21" y1="21" x2="16.65" y2="16.65"/>
              </svg>
            </template>
            {{ loading ? $t('tools.reversePrompt.analyzing') : $t('tools.reversePrompt.submit') }}
          </NButton>
        </div>
      </div>

      <!-- Right: Result -->
      <div class="rp-right">
        <div class="section-label">{{ $t('tools.reversePrompt.resultLabel') }}</div>
        <div v-if="loading" class="result-area result-loading">
          <NSpin size="medium" />
          <span class="loading-text">{{ $t('tools.reversePrompt.analyzing') }}</span>
        </div>
        <div v-else-if="resultPrompt" class="result-area result-ready">
          <div class="result-prompt">{{ resultPrompt }}</div>
          <div class="result-footer">
            <div class="result-meta" v-if="resultMeta">
              <span class="meta-item">{{ $t('tools.reversePrompt.creditsSpent', { n: resultMeta.credits_spent }) }}</span>
              <span class="meta-sep">·</span>
              <span class="meta-item">{{ resultMeta.latency_ms }}ms</span>
            </div>
            <div class="result-actions">
              <NButton size="small" @click="copyPrompt">
                <template #icon>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                  </svg>
                </template>
                {{ $t('tools.reversePrompt.copy') }}
              </NButton>
              <NButton size="small" type="primary" @click="useForGenerate">
                <template #icon>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polygon points="5 3 19 12 5 21 5 3"/>
                  </svg>
                </template>
                {{ $t('tools.reversePrompt.useGenerate') }}
              </NButton>
            </div>
          </div>
        </div>
        <div v-else class="result-area result-empty">
          <svg class="empty-icon" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
            <line x1="16" y1="13" x2="8" y2="13"/>
            <line x1="16" y1="17" x2="8" y2="17"/>
          </svg>
          <span class="empty-text">{{ $t('tools.reversePrompt.noResult') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.rp-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  outline: none;
}

.rp-topbar {
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

.rp-page-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
  flex: 1;
}

.rp-cost-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: rgba(255, 180, 0, 0.1);
  border: 1px solid rgba(255, 180, 0, 0.2);
  border-radius: 8px;
  color: #ffb400;
  font-size: 12px;
  font-weight: 500;
}

.rp-body {
  flex: 1;
  display: grid;
  grid-template-columns: 400px 1fr;
  gap: 0;
  min-height: 0;
  overflow: hidden;
}

.rp-left {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px 24px;
  border-right: 1px solid var(--color-tint-white-06);
  overflow-y: auto;
}

.rp-right {
  display: flex;
  flex-direction: column;
  padding: 20px 24px;
  min-height: 0;
  overflow: hidden;
}

.rp-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.upload-zone {
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
  padding: 24px;
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
  position: relative;
  border: 1px solid var(--color-tint-white-06);
  border-radius: 16px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-tint-white-02);
  min-height: 200px;
  max-height: 300px;
}

.preview-img {
  max-width: 100%;
  max-height: 300px;
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

.options-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.option-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.option-label {
  font-size: 13px;
  color: var(--color-text-secondary);
  font-weight: 500;
}

.option-chips {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.chip {
  padding: 6px 14px;
  border: 1px solid var(--color-tint-white-08);
  background: transparent;
  border-radius: 8px;
  color: var(--color-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

.chip:hover {
  border-color: var(--color-tint-white-15);
  background: var(--color-tint-white-04);
}

.chip.active {
  background: rgba(0, 202, 224, 0.1);
  border-color: rgba(0, 202, 224, 0.3);
  color: #00cae0;
  font-weight: 600;
}

.model-select {
  max-width: 240px;
}

.rp-actions {
  padding-top: 4px;
}

.submit-btn {
  width: 100%;
}

/* Result area */
.result-area {
  flex: 1;
  border: 1px solid var(--color-tint-white-06);
  border-radius: 16px;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.result-loading {
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.loading-text {
  font-size: 13px;
  color: var(--color-text-muted);
}

.result-empty {
  align-items: center;
  justify-content: center;
  gap: 12px;
  border-style: dashed;
}

.empty-icon {
  color: var(--color-text-muted);
  opacity: 0.4;
}

.empty-text {
  font-size: 13px;
  color: var(--color-text-muted);
  opacity: 0.6;
}

.result-ready {
  background: var(--color-tint-white-02);
}

.result-prompt {
  flex: 1;
  padding: 20px;
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text-primary);
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-word;
}

.result-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-top: 1px solid var(--color-tint-white-06);
  flex-shrink: 0;
}

.result-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-muted);
}

.meta-sep {
  opacity: 0.4;
}

.result-actions {
  display: flex;
  gap: 8px;
}

@media (max-width: 768px) {
  .rp-topbar {
    padding: 12px 14px;
  }
  .rp-body {
    grid-template-columns: 1fr;
    overflow-y: auto;
  }
  .rp-left {
    padding: 16px 14px;
    border-right: none;
    border-bottom: 1px solid var(--color-tint-white-06);
  }
  .rp-right {
    padding: 16px 14px;
    min-height: 300px;
  }
  .result-footer {
    flex-direction: column;
    gap: 8px;
    align-items: stretch;
  }
  .result-actions {
    justify-content: flex-end;
  }
}
</style>
