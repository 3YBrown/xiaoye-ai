<script setup>
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const router = useRouter()

const tools = [
  {
    key: 'reverse-prompt',
    route: 'reverse-prompt',
    icon: 'reverse-prompt',
    labelKey: 'tools.reversePrompt.name',
    descKey: 'tools.reversePrompt.desc'
  },
  {
    key: 'image-to-svg',
    route: 'image-to-svg',
    icon: 'svg',
    labelKey: 'tools.imageToSvg',
    descKey: 'tools.imageToSvgDesc'
  },
  {
    key: 'image-convert',
    route: 'image-convert',
    icon: 'convert',
    labelKey: 'tools.imageConvert.name',
    descKey: 'tools.imageConvert.desc'
  }
]

const navigateTo = (routeName) => {
  router.push({ name: routeName })
}
</script>

<template>
  <div class="tools-page">
    <div class="tools-header">
      <h1 class="tools-title">{{ $t('tools.title') }}</h1>
      <p class="tools-subtitle">{{ $t('tools.subtitle') }}</p>
    </div>
    <div class="tools-grid">
      <button
        v-for="tool in tools"
        :key="tool.key"
        class="tool-card"
        @click="navigateTo(tool.route)"
      >
        <div class="tool-icon-wrap">
          <svg v-if="tool.icon === 'reverse-prompt'" class="tool-icon" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <circle cx="8.5" cy="8.5" r="1.5"/>
            <polyline points="21 15 16 10 5 21"/>
            <path d="M14 2v4M18 6l-4-4"/>
          </svg>
          <svg v-else-if="tool.icon === 'svg'" class="tool-icon" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
            <polyline points="7.5 4.21 12 6.81 16.5 4.21"/>
            <polyline points="7.5 19.79 7.5 14.6 3 12"/>
            <polyline points="21 12 16.5 14.6 16.5 19.79"/>
            <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
            <line x1="12" y1="22.08" x2="12" y2="12"/>
          </svg>
          <svg v-else-if="tool.icon === 'convert'" class="tool-icon" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <polyline points="16 3 16 9 21 9"/>
            <path d="M8 14l2 2 4-4"/>
          </svg>
        </div>
        <div class="tool-info">
          <div class="tool-name">{{ $t(tool.labelKey) }}</div>
          <div class="tool-desc">{{ $t(tool.descKey) }}</div>
        </div>
        <svg class="tool-arrow" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="9 18 15 12 9 6"/>
        </svg>
      </button>

      <div class="tool-card coming-soon">
        <div class="tool-icon-wrap placeholder">
          <svg class="tool-icon" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        </div>
        <div class="tool-info">
          <div class="tool-name placeholder-text">{{ $t('tools.comingSoon') }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tools-page {
  height: 100%;
  overflow-y: auto;
  padding: 32px 24px;
}

.tools-header {
  max-width: 800px;
  margin: 0 auto 32px;
}

.tools-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0 0 8px;
}

.tools-subtitle {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0;
}

.tools-grid {
  max-width: 800px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 12px;
}

.tool-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: var(--color-tint-white-02);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 16px;
  cursor: pointer;
  transition: all 0.25s;
  text-align: left;
  width: 100%;
  font-family: inherit;
  color: inherit;
}

.tool-card:hover {
  border-color: rgba(0, 202, 224, 0.35);
  background: rgba(0, 202, 224, 0.04);
  transform: translateY(-2px);
  box-shadow: 0 8px 24px var(--color-tint-black-30);
}

.tool-card.coming-soon {
  cursor: default;
  opacity: 0.5;
}

.tool-card.coming-soon:hover {
  border-color: var(--color-tint-white-06);
  background: var(--color-tint-white-02);
  transform: none;
  box-shadow: none;
}

.tool-icon-wrap {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: rgba(0, 202, 224, 0.1);
  color: #00cae0;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.tool-icon-wrap.placeholder {
  background: var(--color-tint-white-04);
  color: var(--color-text-muted);
}

.tool-info {
  flex: 1;
  min-width: 0;
}

.tool-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 4px;
}

.tool-desc {
  font-size: 13px;
  color: var(--color-text-muted);
  line-height: 1.4;
}

.placeholder-text {
  color: var(--color-text-muted);
  font-weight: 500;
}

.tool-arrow {
  color: var(--color-text-muted);
  flex-shrink: 0;
  opacity: 0.5;
  transition: opacity 0.2s;
}

.tool-card:hover .tool-arrow {
  opacity: 1;
  color: #00cae0;
}

.coming-soon .tool-arrow {
  display: none;
}

@media (max-width: 768px) {
  .tools-page {
    padding: 20px 14px;
  }
  .tools-grid {
    grid-template-columns: 1fr;
  }
  .tool-card {
    padding: 16px;
  }
}
</style>
