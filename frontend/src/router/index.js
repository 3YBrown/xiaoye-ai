import { createRouter, createWebHistory } from 'vue-router'
import i18n from '../i18n'

const OAuthCallback = () => import('../views/OAuthCallback.vue')
const Landing = () => import('../views/Landing.vue')
const AppLayout = () => import('../layouts/AppLayout.vue')
const TermsOfService = () => import('../views/TermsOfService.vue')
const PrivacyPolicy = () => import('../views/PrivacyPolicy.vue')
const Inspiration = () => import('../views/Inspiration.vue')
const InspirationSearch = () => import('../views/InspirationSearch.vue')
const InspirationDetail = () => import('../views/InspirationDetail.vue')
const Generate = () => import('../views/Generate.vue')
const Assets = () => import('../views/Assets.vue')
const Account = () => import('../views/Account.vue')
const Tools = () => import('../views/Tools.vue')
const ImageToSvg = () => import('../views/ImageToSvg.vue')
const ReversePrompt = () => import('../views/ReversePrompt.vue')
const ImageConvert = () => import('../views/ImageConvert.vue')

const routes = [
  {
    path: '/',
    name: 'landing',
    component: Landing,
    meta: {
      titleKey: 'seo.landing.title',
      descriptionKey: 'seo.landing.description',
      keywordKey: 'seo.landing.keywords',
      guest: true
    }
  },
  {
    path: '/terms-of-service',
    name: 'terms-of-service',
    component: TermsOfService,
    meta: {
      titleKey: 'seo.terms.title',
      descriptionKey: 'seo.terms.description',
      guest: true
    }
  },
  {
    path: '/privacy-policy',
    name: 'privacy-policy',
    component: PrivacyPolicy,
    meta: {
      titleKey: 'seo.privacy.title',
      descriptionKey: 'seo.privacy.description',
      guest: true
    }
  },
  {
    path: '/oauth/callback',
    name: 'oauth-callback',
    component: OAuthCallback,
    meta: { guest: true }
  },
  {
    path: '/',
    component: AppLayout,
    children: [
      {
        path: 'inspiration',
        name: 'inspiration',
        component: Inspiration,
        meta: {
          titleKey: 'seo.inspiration.title',
          descriptionKey: 'seo.inspiration.description',
          keywordKey: 'seo.inspiration.keywords'
        }
      },
      {
        path: 'inspiration/search',
        name: 'inspiration-search',
        component: InspirationSearch,
        meta: {
          titleKey: 'seo.inspiration.title',
          descriptionKey: 'seo.inspiration.description',
          keywordKey: 'seo.inspiration.keywords'
        }
      },
      {
        path: 'inspiration/:shareId',
        name: 'inspiration-detail',
        component: InspirationDetail,
        meta: {
          titleKey: 'seo.inspirationDetail.title',
          descriptionKey: 'seo.inspirationDetail.description'
        }
      },
      {
        path: 'generate',
        name: 'generate',
        component: Generate,
        meta: {
          titleKey: 'seo.generate.title',
          descriptionKey: 'seo.generate.description',
          keywordKey: 'seo.generate.keywords'
        }
      },
      {
        path: 'assets',
        name: 'assets',
        component: Assets,
        meta: {
          titleKey: 'seo.assets.title',
          descriptionKey: 'seo.assets.description'
        }
      },
      {
        path: 'account',
        name: 'account',
        component: Account,
        meta: {
          titleKey: 'seo.account.title',
          descriptionKey: 'seo.account.description',
          requiresAuth: true
        }
      },
      {
        path: 'tools',
        name: 'tools',
        component: Tools,
        meta: {
          titleKey: 'seo.tools.title',
          descriptionKey: 'seo.tools.description',
          keywordKey: 'seo.tools.keywords'
        }
      },
      {
        path: 'tools/image-to-svg',
        name: 'image-to-svg',
        component: ImageToSvg,
        meta: {
          titleKey: 'seo.imageToSvg.title',
          descriptionKey: 'seo.imageToSvg.description',
          keywordKey: 'seo.imageToSvg.keywords'
        }
      },
      {
        path: 'tools/reverse-prompt',
        name: 'reverse-prompt',
        component: ReversePrompt,
        meta: {
          titleKey: 'seo.reversePrompt.title',
          descriptionKey: 'seo.reversePrompt.description',
          keywordKey: 'seo.reversePrompt.keywords'
        }
      },
      {
        path: 'tools/image-convert',
        name: 'image-convert',
        component: ImageConvert,
        meta: {
          titleKey: 'seo.imageConvert.title',
          descriptionKey: 'seo.imageConvert.description',
          keywordKey: 'seo.imageConvert.keywords'
        }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

function resolveMetaText(meta, field, keyField, fallback) {
  if (meta?.[keyField]) {
    return i18n.global.t(meta[keyField])
  }
  return meta?.[field] || fallback
}

function applyRouteMeta(to) {
  document.title = resolveMetaText(to.meta, 'title', 'titleKey', '小野 AI')

  const descTag = document.querySelector('meta[name="description"]')
  if (descTag) {
    const description = resolveMetaText(to.meta, 'description', 'descriptionKey', '')
    if (description) {
      descTag.setAttribute('content', description)
    }
  }

  const keywordsTag = document.querySelector('meta[name="keywords"]')
  if (keywordsTag) {
    const keywords = resolveMetaText(
      to.meta,
      'keywords',
      'keywordKey',
      'Seedance, Seedance 2.0, Seedance 2.0提示词, AI video generation, text to video'
    )
    if (keywords) {
      keywordsTag.setAttribute('content', keywords)
    }
  }
}

router.beforeEach((to, from, next) => {
  applyRouteMeta(to)

  const token = localStorage.getItem('token')
  const isLoggedIn = !!token

  if (to.meta.requiresAuth && !isLoggedIn) {
    next({ name: 'landing', query: { redirect: to.fullPath } })
    return
  }

  if (to.name === 'landing' && isLoggedIn && !to.query.invite) {
    next({ name: 'inspiration' })
    return
  }

  next()
})

if (typeof window !== 'undefined') {
  window.addEventListener('locale-changed', () => {
    applyRouteMeta(router.currentRoute.value)
  })
}

export default router
