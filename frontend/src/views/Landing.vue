<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { useThemeStore } from '../stores/theme'
import { useLocaleStore } from '../stores/locale'

const { t } = useI18n()
const router = useRouter()
const themeStore = useThemeStore()
const localeStore = useLocaleStore()


// 落地页强制深色
themeStore.forceTheme('dark')

// Hero 背景视频轮播
const activeVideo = ref(0)
const heroVideos = [
  { src: '//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/media/1.video.7a8931e8.mp4', poster: '//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/image/1.down.08054096.jpeg' },
  { src: '//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/media/2.video.25d2b0f2.mp4', poster: '//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/image/2.down.ee91868b.jpg' },
  { src: '//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/media/3.video.667e9ea3.mp4', poster: '//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/image/3.down.482b8b3d.jpg' }
]
let videoTimer = null

const features = computed(() => [
  {
    icon: '🎨',
    title: t('landing.feature1Title'),
    desc: t('landing.feature1Desc'),
    gradient: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
  },
  {
    icon: '✏️',
    title: t('landing.feature2Title'),
    desc: t('landing.feature2Desc'),
    gradient: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)'
  },
  {
    icon: '🎬',
    title: t('landing.feature3Title'),
    desc: t('landing.feature3Desc'),
    gradient: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)'
  },
  {
    icon: '💬',
    title: t('landing.feature4Title'),
    desc: t('landing.feature4Desc'),
    gradient: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)'
  },
  {
    icon: '🛍️',
    title: t('landing.feature5Title'),
    desc: t('landing.feature5Desc'),
    gradient: 'linear-gradient(135deg, #fbbf24 0%, #f59e0b 100%)'
  }
])

const showcaseItems = computed(() => [
  { prompt: t('landing.showcase1Prompt'), image: '//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/image/1.727df331.jpg', tag: t('landing.showcase1Tag') },
  { prompt: t('landing.showcase2Prompt'), image: '//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/image/3.abaef3cc.jpg', tag: t('landing.showcase2Tag') },
  { prompt: t('landing.showcase3Prompt'), image: '//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/image/5.1e569bfc.jpg', tag: t('landing.showcase3Tag') }
])

// 滚动动画
let observer = null

onMounted(() => {
  // 视频轮播
  videoTimer = setInterval(() => {
    activeVideo.value = (activeVideo.value + 1) % heroVideos.length
  }, 3000)

  observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        entry.target.classList.add('visible')
      }
    })
  }, { threshold: 0.1, rootMargin: '0px 0px -50px 0px' })

  document.querySelectorAll('.animate-on-scroll').forEach(el => {
    observer.observe(el)
  })
})

onUnmounted(() => {
  observer?.disconnect()
  if (videoTimer) clearInterval(videoTimer)
  themeStore.clearForceTheme()
})

const handleStart = () => {
  router.push('/inspiration')
}
</script>

<template>
  <div class="landing">

    <!-- 导航 -->
    <nav class="landing-nav">
      <div class="nav-container">
        <div class="nav-brand">
          <img src="/images/icon.png" alt="" class="brand-icon" />
          <span class="brand-text">小野 AI</span>
        </div>
        <div class="nav-links">
          <a href="#features" class="nav-link">{{ $t('landing.navFeatures') }}</a>
          <a href="#showcase" class="nav-link">{{ $t('landing.navShowcase') }}</a>
          <a href="#models" class="nav-link">{{ $t('landing.navModels') }}</a>
        </div>
        <div class="nav-actions">
          <div class="lang-switcher">
            <div class="lang-slider" :class="{ 'is-en': localeStore.locale === 'en' }"></div>
            <button
              class="lang-opt"
              :class="{ active: localeStore.locale === 'zh' }"
              @click="localeStore.setLocale('zh')"
            >中</button>
            <button
              class="lang-opt"
              :class="{ active: localeStore.locale === 'en' }"
              @click="localeStore.setLocale('en')"
            >EN</button>
          </div>
          <NButton type="primary" @click="handleStart" strong class="nav-cta">
            {{ $t('landing.startCreating') }}
          </NButton>
        </div>
      </div>
    </nav>

    <!-- ========== Hero 区域 ========== -->
    <header class="hero">
      <div class="hero-slideshow">
        <video
          v-for="(v, i) in heroVideos"
          :key="i"
          class="hero-slide"
          :class="{ active: activeVideo === i }"
          :src="v.src"
          :poster="v.poster"
          preload="auto"
          muted
          loop
          playsinline
          autoplay
          disablepictureinpicture
          disableremoteplayback
        />
        <div class="hero-overlay"></div>
      </div>
      <div class="hero-content">
        <h1 class="hero-title animate-on-scroll">
          <span class="title-line title-gradient">{{ $t('landing.heroTitle') }}</span>
        </h1>
      </div>
    </header>

    <!-- ========== 灵感展示 ========== -->
    <section id="showcase" class="showcase-section">
      <div class="section-container">
        <div class="section-header animate-on-scroll">
          <h2 class="section-title title-line title-gradient">{{ $t('landing.showcaseTitle') }}</h2>
          <p class="section-subtitle">{{ $t('landing.showcaseSubtitle') }}</p>
        </div>
      </div>

      <div class="showcase-grid">
          <div
            v-for="(item, i) in showcaseItems"
            :key="i"
            class="showcase-card animate-on-scroll"
            :style="{ '--delay': (i * 0.08) + 's' }"
            @click="handleStart"
          >
            <div class="showcase-card-inner">
              <img :src="item.image" :alt="item.prompt" class="showcase-img" loading="lazy" />
              <div class="showcase-overlay">
                <span class="showcase-tag">{{ item.tag }}</span>
                <p class="showcase-prompt">{{ item.prompt }}</p>
                <span class="showcase-try">
                  {{ $t('landing.tryThis') }}
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
                </span>
              </div>
            </div>
          </div>
      </div>
    </section>

    <!-- ========== 功能介绍 ========== -->
    <section id="features" class="features-section">
      <div class="section-container">
        <div class="section-header animate-on-scroll">
          <h2 class="section-title title-line title-gradient">{{ $t('landing.featuresTitle') }}</h2>
          <p class="section-subtitle">{{ $t('landing.featuresSubtitle') }}</p>
        </div>

        <div class="features-grid">
          <div
            v-for="(f, i) in features"
            :key="i"
            class="feature-card animate-on-scroll"
            :style="{ '--delay': (i * 0.1) + 's' }"
          >
            <div class="feature-icon-wrap" :style="{ background: f.gradient }">
              <span class="feature-icon">{{ f.icon }}</span>
            </div>
            <h3 class="feature-title">{{ f.title }}</h3>
            <p class="feature-desc">{{ f.desc }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- ========== 视频生成展示 ========== -->
    <section class="video-section">
      <div class="section-container">
        <div class="section-header animate-on-scroll">
          <h2 class="section-title title-line title-gradient">{{ $t('landing.videoTitle') }}</h2>
          <p class="section-subtitle">{{ $t('landing.videoSubtitle') }}</p>
        </div>
      </div>
      <div class="video-list">
        <!-- 01 -->
        <div class="video-row animate-on-scroll">
          <div class="video-text-col">
            <div class="video-text-inner">
              <div class="video-num">01</div>
              <h3 class="video-heading">{{ $t('landing.video01Title') }}</h3>
              <p class="video-body">{{ $t('landing.video01Desc') }}</p>
              <button class="video-cta" @click="handleStart">{{ $t('landing.createNow') }}</button>
            </div>
          </div>
          <div class="video-media-col">
            <video src="//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/media/0.485b16d2.mp4" poster="//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/image/0.8f3410ca.jpg" class="video-media" muted loop playsinline autoplay preload="auto" />
          </div>
        </div>
        <!-- 02 -->
        <div class="video-row video-row-reverse animate-on-scroll">
          <div class="video-text-col">
            <div class="video-text-inner">
              <div class="video-num">02</div>
              <h3 class="video-heading" v-html="$t('landing.video02Title')"></h3>
              <p class="video-body">{{ $t('landing.video02Desc') }}</p>
              <button class="video-cta" @click="handleStart">{{ $t('landing.createNow') }}</button>
            </div>
          </div>
          <div class="video-media-col">
            <video src="//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/media/1.55160634.mp4" poster="//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/image/1.b3d7df69.jpg" class="video-media" muted loop playsinline autoplay preload="auto" />
          </div>
        </div>
        <!-- 03 -->
        <div class="video-row animate-on-scroll">
          <div class="video-text-col">
            <div class="video-text-inner">
              <div class="video-num">03</div>
              <h3 class="video-heading">{{ $t('landing.video03Title') }}</h3>
              <p class="video-body">{{ $t('landing.video03Desc') }}</p>
              <button class="video-cta" @click="handleStart">{{ $t('landing.createNow') }}</button>
            </div>
          </div>
          <div class="video-media-col">
            <video src="//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/media/2.7045d286.mp4" poster="//lf3-lv-buz.vlabstatic.com/obj/image-lvweb-buz/growth/jimeng/landing_page/static/image/2.6028618f.jpg" class="video-media" muted loop playsinline autoplay preload="auto" />
          </div>
        </div>
      </div>
    </section>

    <!-- ========== 模型支持 ========== -->
    <section id="models" class="models-section">
      <div class="section-container">
        <div class="section-header animate-on-scroll">
          <h2 class="section-title title-line title-gradient">{{ $t('landing.modelsTitle') }}</h2>
          <p class="section-subtitle">{{ $t('landing.modelsSubtitle') }}</p>
        </div>

        <div class="models-grid">
          <div class="model-card animate-on-scroll" style="--delay: 0s">
            <div class="model-icon">🍌</div>
            <div class="model-name">Nanobanana Pro</div>
            <div class="model-provider">Gemini 3 Pro</div>
            <div class="model-badge">{{ $t('landing.modelRecommended') }}</div>
          </div>
          <div class="model-card animate-on-scroll" style="--delay: 0.1s">
            <div class="model-icon">🎨</div>
            <div class="model-name">Seedream-4.5</div>
            <div class="model-provider">Volcengine</div>
            <div class="model-badge hot">{{ $t('landing.modelHot') }}</div>
          </div>
          <div class="model-card animate-on-scroll" style="--delay: 0.2s">
            <div class="model-icon">🎬</div>
            <div class="model-name">Seedance-1.5</div>
            <div class="model-provider">Volcengine</div>
            <div class="model-badge new">NEW</div>
          </div>
          <div class="model-card coming-soon animate-on-scroll" style="--delay: 0.3s">
            <div class="model-icon">🔮</div>
            <div class="model-name">{{ $t('landing.moreModels') }}</div>
            <div class="model-provider">{{ $t('landing.comingSoon') }}</div>
          </div>
        </div>
      </div>
    </section>

    <!-- ========== CTA ========== -->
    <section class="cta-section">
      <div class="section-container animate-on-scroll">
        <h2 class="cta-title">{{ $t('landing.ctaTitle') }}</h2>
        <p class="cta-desc">{{ $t('landing.ctaDesc') }}</p>
        <button class="cta-primary large" @click="handleStart">
          <span class="cta-content">{{ $t('landing.ctaButton') }}</span>
        </button>
      </div>
    </section>

    <!-- ========== 社群 ========== -->
    <section class="community-section">
      <div class="section-container animate-on-scroll">
        <div class="community-card">
          <div class="community-qr">
            <img src="/images/qrcode.jpg" alt="微信群二维码" class="qr-image" />
          </div>
          <div class="community-info">
            <h3 class="community-title">{{ $t('landing.communityTitle') }}</h3>
            <p class="community-desc" style="white-space: pre-line">{{ $t('landing.communityDesc') }}</p>
            <div class="community-tags">
              <span class="community-tag">{{ $t('landing.communityTagArt') }}</span>
              <span class="community-tag">{{ $t('landing.communityTagVideo') }}</span>
              <span class="community-tag">{{ $t('landing.communityTagExchange') }}</span>
              <span class="community-tag">{{ $t('landing.communityTagFree') }}</span>
            </div>
            <div class="community-hint">{{ $t('landing.communityHint') }}</div>
          </div>
        </div>
      </div>
    </section>

    <!-- ========== Footer 备案 ========== -->
    <footer class="site-footer">
      <div class="footer-beian">
        <a href="https://beian.miit.gov.cn/" target="_blank" rel="noopener noreferrer">苏ICP备2025223139号-1</a>
        <span class="footer-divider">|</span>
        <a href="https://www.beian.gov.cn/portal/registerSystemInfo?recordcode=32050602014047" target="_blank" rel="noopener noreferrer">苏公网安备32050602014047号</a>
      </div>
    </footer>

  </div>
</template>

<style scoped>
.landing {
  min-height: 100vh;
  position: relative;
  overflow: hidden;
}

/* ========== 导航 ========== */
.landing-nav {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 1000;
  background: transparent;
}

.nav-container {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 48px;
}

.nav-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 20px;
  font-weight: 800;
}

.brand-icon {
  width: 28px;
  height: 28px;
  object-fit: contain;
}

.brand-text {
  color: var(--color-text-primary);
  letter-spacing: -0.02em;
}

.nav-links {
  display: flex;
  gap: 32px;
}

.nav-link {
  color: var(--color-text-muted);
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
  transition: color 0.2s;
  position: relative;
}

.nav-link::after {
  content: '';
  position: absolute;
  bottom: -4px;
  left: 0;
  width: 0;
  height: 2px;
  background: var(--color-tint-white-05);
  border-radius: 1px;
  transition: width 0.3s;
}

.nav-link:hover { color: var(--color-text-primary); }
.nav-link:hover::after { width: 100%; }

.nav-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.lang-switcher {
  position: relative;
  display: flex;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  padding: 2px;
}

.lang-slider {
  position: absolute;
  top: 2px;
  left: 2px;
  width: calc(50% - 2px);
  height: calc(100% - 4px);
  background: rgba(255, 255, 255, 0.12);
  border-radius: 6px;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  pointer-events: none;
}

.lang-slider.is-en {
  transform: translateX(100%);
}

.lang-opt {
  position: relative;
  z-index: 1;
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.4);
  font-size: 12px;
  font-weight: 600;
  padding: 4px 12px;
  cursor: pointer;
  transition: color 0.25s;
  line-height: 1.2;
}

.lang-opt:hover {
  color: rgba(255, 255, 255, 0.7);
}

.lang-opt.active {
  color: #fff;
}

.nav-cta {
  border-radius: 6px !important;
  background-color: #00cae0 !important;
  border-color: #00cae0 !important;
  color: #ffffff !important;
}

/* ========== Hero 背景轮播 ========== */
.hero-slideshow {
  position: absolute;
  inset: 0;
  z-index: 0;
  overflow: hidden;
}

.hero-slide {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center;
  opacity: 0;
  transition: opacity 1s ease-in-out;
}

.hero-slide.active {
  opacity: 1;
}

.hero-overlay {
  position: absolute;
  inset: 0;
  background: transparent;
  z-index: 1;
}

/* ========== Hero ========== */
.hero {
  position: relative;
  padding: 100px 20px 48px;
  text-align: center;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
}

.hero-content {
  max-width: 800px;
  width: 100%;
  margin: 0 auto;
  margin-top: -60px;
  position: relative;
  z-index: 2;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  border-radius: 20px;
  background: rgba(102, 126, 234, 0.08);
  border: 1px solid rgba(102, 126, 234, 0.2);
  color: #a5b4fc;
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 24px;
  backdrop-filter: blur(10px);
}

.badge-dot {
  width: 6px;
  height: 6px;
  background: #a5b4fc;
  border-radius: 50%;
  animation: pulse 2s ease-in-out infinite;
}

.hero-title {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 20px;
}

.title-line {
  font-family: 'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  font-size: 64px;
  font-weight: 900;
  line-height: 1.15;
  color: var(--color-text-primary);
  letter-spacing: 0.04em;
  text-shadow: 0 2px 20px var(--color-tint-black-30);
  font-style: italic;
  transform: skewX(-6deg);
  display: inline-block;
}

.title-gradient {
  font-size: 64px;
  font-weight: 900;
  letter-spacing: 0.06em;
  font-style: italic;
  transform: skewX(-6deg);
  display: inline-block;
  color: var(--color-text-primary);
  text-shadow: 0 2px 20px var(--color-tint-black-30);
}

@keyframes gradientFlow {
  0% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
  100% { background-position: 0% 50%; }
}

.hero-subtitle {
  font-size: 16px;
  color: var(--color-text-secondary);
  line-height: 1.75;
  margin-bottom: 32px;
  max-width: 600px;
  margin-left: auto;
  margin-right: auto;
}

.hero-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-bottom: 44px;
}

/* ========== CTA 按钮 ========== */
.cta-primary {
  position: relative;
  padding: 12px 24px;
  border-radius: 12px;
  background-color: #00cae0;
  color: white;
  font-size: 15px;
  font-weight: 700;
  border: none;
  cursor: pointer;
  overflow: hidden;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 4px 20px rgba(0, 202, 224, 0.35);
}

.cta-primary:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 32px rgba(0, 202, 224, 0.45);
}

.cta-primary:active { transform: translateY(-1px); }

.cta-primary.small {
  padding: 10px 24px;
  font-size: 14px;
  border-radius: 10px;
}

.cta-primary.large {
  padding: 14px 32px;
  font-size: 16px;
  border-radius: 12px;
}

.cta-content {
  display: flex;
  align-items: center;
  gap: 8px;
  position: relative;
  z-index: 1;
}

.cta-secondary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 13px 24px;
  border-radius: 14px;
  background: var(--color-tint-white-04);
  border: 1px solid var(--color-tint-white-10);
  color: var(--color-text-secondary);
  font-size: 15px;
  font-weight: 600;
  text-decoration: none;
  transition: all 0.3s;
  backdrop-filter: blur(10px);
}

.cta-secondary:hover {
  background: var(--color-tint-white-08);
  border-color: var(--color-tint-white-20);
  color: var(--color-text-primary);
  transform: translateY(-2px);
}

/* ========== 数据统计 ========== */
.hero-stats {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 36px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.stat-number {
  font-size: 24px;
  font-weight: 800;
  background: linear-gradient(135deg, #a5b4fc, #f093fb);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.stat-label {
  font-size: 12px;
  color: var(--color-text-muted);
}

.stat-divider {
  width: 1px;
  height: 28px;
  background: var(--color-tint-white-08);
}

/* ========== 通用 Section ========== */
.section-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 48px;
}

.section-header {
  text-align: center;
  margin: 0 0 40px;
  padding: 0;
}

.section-tag {
  display: inline-block;
  padding: 5px 14px;
  border-radius: 14px;
  background: rgba(102, 126, 234, 0.1);
  border: 1px solid rgba(102, 126, 234, 0.2);
  color: #a5b4fc;
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 14px;
  letter-spacing: 0.05em;
}

.section-title {
  font-size: 32px;
  font-weight: 800;
  color: var(--color-text-primary);
  margin: 0 0 12px;
  letter-spacing: -0.02em;
}

.section-subtitle {
  color: var(--color-text-secondary);
  font-size: 15px;
  max-width: 480px;
  margin: 0 auto;
  line-height: 1.7;
}

/* ========== 灵感展示 ========== */
.showcase-section {
  padding: 48px 0 64px;
  position: relative;
}

.showcase-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 4px;
}

.showcase-card {
  cursor: pointer;
  animation-delay: var(--delay);
}

.showcase-card-inner {
  border-radius: 0;
  height: 100%;
  position: relative;
  overflow: hidden;
  aspect-ratio: 2 / 1;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.showcase-card:hover .showcase-card-inner {
  z-index: 2;
  box-shadow: 0 16px 48px var(--color-tint-black-50);
}

.showcase-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform 0.5s cubic-bezier(0.4, 0, 0.2, 1);
}

.showcase-card:hover .showcase-img {
  transform: scale(1.05);
}

.showcase-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  padding: 20px;
  background: linear-gradient(to top, var(--color-overlay-bg) 0%, var(--color-tint-black-10) 50%, transparent 100%);
  opacity: 0;
  transition: opacity 0.35s ease;
}

.showcase-card:hover .showcase-overlay {
  opacity: 1;
}

.showcase-tag {
  display: inline-block;
  width: fit-content;
  padding: 3px 10px;
  border-radius: 6px;
  background: rgba(102, 126, 234, 0.25);
  backdrop-filter: blur(8px);
  color: #c4b5fd;
  font-size: 11px;
  font-weight: 600;
  margin-bottom: 8px;
}

.showcase-prompt {
  color: var(--color-text-primary);
  font-size: 14px;
  line-height: 1.5;
  margin-bottom: 10px;
  text-shadow: 0 1px 4px var(--color-tint-black-50);
}

.showcase-try {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: #a5b4fc;
  font-size: 12px;
  font-weight: 600;
  transform: translateX(-6px);
  transition: all 0.3s 0.1s;
  opacity: 0;
}

.showcase-card:hover .showcase-try {
  opacity: 1;
  transform: translateX(0);
}

/* ========== 功能卡片 ========== */
.features-section {
  padding: 48px 0 64px;
  position: relative;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
}

.feature-card {
  background: var(--color-tint-white-02);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 14px;
  padding: 24px 20px;
  text-align: center;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
  animation-delay: var(--delay);
}

.feature-card:hover {
  border-color: var(--color-tint-white-12);
  transform: translateY(-4px);
  box-shadow: 0 16px 48px var(--color-tint-black-35);
}

.feature-icon-wrap {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 16px;
  box-shadow: 0 4px 14px var(--color-tint-black-25);
}

.feature-icon { font-size: 24px; }

.feature-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: 8px;
}

.feature-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.65;
}

/* ========== 视频展示 ========== */
.video-section {
  padding: 48px 0 64px;
}

.video-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 0;
}

.video-row {
  display: grid;
  grid-template-columns: 5fr 7fr;
  min-height: 420px;
  background: rgba(255, 255, 255, 0.015);
  border-top: 1px solid var(--color-tint-white-04);
}

.video-row-reverse .video-text-col { order: 2; }
.video-row-reverse .video-media-col { order: 1; }

.video-text-col {
  display: flex;
  align-items: center;
  padding: 48px 56px;
}

.video-text-inner {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  max-width: 440px;
}

.video-num {
  font-size: 80px;
  font-weight: 900;
  line-height: 1;
  color: var(--color-tint-white-04);
  letter-spacing: -0.04em;
  margin-bottom: 16px;
  user-select: none;
}

.video-heading {
  font-size: 30px;
  font-weight: 800;
  color: var(--color-text-primary);
  margin-bottom: 14px;
  letter-spacing: -0.02em;
  line-height: 1.35;
}

.video-body {
  font-size: 15px;
  color: var(--color-text-secondary);
  line-height: 1.8;
  margin-bottom: 28px;
}

.video-cta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 11px 26px;
  border-radius: 12px;
  background-color: #00cae0;
  color: white;
  font-size: 14px;
  font-weight: 600;
  border: none;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 4px 20px rgba(0, 202, 224, 0.3);
}

.video-cta:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 28px rgba(0, 202, 224, 0.4);
}

.video-media-col {
  position: relative;
  overflow: hidden;
}

.video-media {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

/* ========== 模型卡片 ========== */
.models-section { padding: 48px 0 64px; }

.models-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
}

.model-card {
  background: var(--color-tint-white-02);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 14px;
  padding: 24px 18px;
  text-align: center;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
  animation-delay: var(--delay);
}

.model-card:hover {
  border-color: var(--color-tint-white-12);
  transform: translateY(-3px);
  box-shadow: 0 10px 32px var(--color-tint-black-30);
}

.model-card.coming-soon { opacity: 0.45; }

.model-icon {
  font-size: 34px;
  margin-bottom: 12px;
}

.model-name {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: 4px;
}

.model-provider {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 10px;
}

.model-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 6px;
  background: rgba(102, 126, 234, 0.12);
  color: #a5b4fc;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.05em;
}

.model-badge.hot {
  background: rgba(240, 147, 251, 0.12);
  color: #f093fb;
}

.model-badge.new {
  background: rgba(79, 172, 254, 0.12);
  color: #4facfe;
}

/* ========== CTA Section ========== */
.cta-section {
  padding: 48px 0 80px;
  text-align: center;
  position: relative;
}

.cta-title {
  font-size: 32px;
  font-weight: 800;
  color: var(--color-text-primary);
  margin-top: 0;
  margin-bottom: 12px;
  letter-spacing: -0.02em;
}

.cta-desc {
  font-size: 15px;
  color: var(--color-text-secondary);
  margin-bottom: 32px;
}

/* ========== 滚动动画 ========== */
.animate-on-scroll {
  opacity: 0;
  transform: translateY(24px);
  transition: opacity 0.7s cubic-bezier(0.4, 0, 0.2, 1), transform 0.7s cubic-bezier(0.4, 0, 0.2, 1);
  transition-delay: var(--delay, 0s);
}

.animate-on-scroll.visible {
  opacity: 1;
  transform: translateY(0);
}

/* ========== 响应式 ========== */
@media (max-width: 1024px) {
  .features-grid,
  .models-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .video-row {
    grid-template-columns: 1fr;
    min-height: auto;
  }

  .video-row-reverse .video-text-col { order: 0; }
  .video-row-reverse .video-media-col { order: 0; }

  .video-text-col {
    padding: 36px 32px;
  }

  .video-media-col {
    position: relative;
    height: 280px;
  }

  .video-num { font-size: 56px; }

  .showcase-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }

  .section-container { padding: 0 32px; }
  .nav-container { padding: 12px 32px; }
}

@media (max-width: 768px) {
  .hero {
    padding: 100px 16px 40px;
    min-height: auto;
  }

  .title-line,
  .title-gradient { font-size: 36px; }
  .hero-subtitle { font-size: 14px; }

  .hero-actions {
    flex-direction: column;
    gap: 10px;
  }

  .hero-stats { gap: 20px; }
  .stat-number { font-size: 20px; }

  .showcase-grid { grid-template-columns: 1fr; }
  .features-grid { grid-template-columns: 1fr 1fr; }
  .models-grid { grid-template-columns: 1fr 1fr; }

  .section-title { font-size: 24px; }
  .section-container { padding: 0 20px; }
  .nav-links { display: none; }
  .nav-container { padding: 10px 20px; }

  .cta-title { font-size: 26px; }
  .video-heading { font-size: 22px; }
  .video-text-col { padding: 28px 20px; }
  .video-num { font-size: 48px; }
  .video-media-col { height: 220px; }

  .features-section,
  .showcase-section,
  .video-section,
  .models-section { padding: 16px 0 32px; }

  .cta-section { padding: 16px 0 48px; }
  .hide-mobile { display: none; }
}

@media (max-width: 480px) {
  .title-line { font-size: 32px; }
  .title-gradient { font-size: 22px; }
  .hero { padding: 88px 10px 32px; }
  .section-container { padding: 0 16px; }

  .showcase-grid { grid-template-columns: 1fr; }
  .features-grid { grid-template-columns: 1fr; }
  .models-grid { grid-template-columns: 1fr 1fr; }

  .cta-primary {
    width: 100%;
    text-align: center;
    justify-content: center;
  }

  .cta-secondary {
    width: 100%;
    text-align: center;
    justify-content: center;
  }

  .section-header { margin-bottom: 16px; }
  .section-title { font-size: 22px; }
  .cta-title { font-size: 22px; }
}

/* ====== Community ====== */
.community-section {
  padding: 0 0 48px;
}

.community-card {
  display: flex;
  align-items: center;
  gap: 40px;
  max-width: 640px;
  margin: 0 auto;
  padding: 32px 40px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 20px;
  backdrop-filter: blur(12px);
}

.community-qr {
  flex-shrink: 0;
}
.qr-image {
  width: 160px;
  height: 160px;
  border-radius: 12px;
  background: #fff;
  object-fit: cover;
}

.community-info {
  flex: 1;
  min-width: 0;
}

.community-title {
  font-size: 20px;
  font-weight: 700;
  color: #fff;
  margin-bottom: 8px;
}

.community-desc {
  font-size: 13px;
  line-height: 1.7;
  color: rgba(255, 255, 255, 0.5);
  margin-bottom: 14px;
}

.community-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 14px;
}

.community-tag {
  padding: 4px 12px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 20px;
}

.community-hint {
  font-size: 12px;
  color: #00cae0;
  font-weight: 500;
}

@media (max-width: 768px) {
  .community-card {
    flex-direction: column;
    text-align: center;
    gap: 20px;
    padding: 28px 24px;
  }
  .qr-image {
    width: 140px;
    height: 140px;
  }
  .community-tags {
    justify-content: center;
  }
  .community-desc br {
    display: none;
  }
}

/* ====== Footer ====== */
.site-footer {
  padding: 24px 20px 32px;
  text-align: center;
}

.footer-beian {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.35);
}

.footer-beian a {
  color: rgba(255, 255, 255, 0.35);
  text-decoration: none;
  transition: color 0.2s;
}

.footer-beian a:hover {
  color: rgba(255, 255, 255, 0.6);
}

.footer-divider {
  opacity: 0.4;
}
</style>
