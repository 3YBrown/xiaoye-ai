import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useComposerDraftStore = defineStore('composerDraft', () => {
  const draft = ref(null)

  function setRemixDraft(post) {
    if (!post) return
    draft.value = {
      mode: post.type || 'image',
      prompt: post.prompt || '',
      params: post.params || {},
      shareId: post.share_id || ''
    }
  }

  function setReferenceDraft(post, imageUrl) {
    draft.value = {
      mode: 'image',
      prompt: '',
      params: {},
      referenceImage: imageUrl || post?.cover_url || '',
      shareId: post?.share_id || ''
    }
  }

  function consumeDraft() {
    const current = draft.value
    draft.value = null
    return current
  }

  function setPromptDraft(prompt) {
    draft.value = {
      mode: 'image',
      prompt: prompt || '',
      params: {}
    }
  }

  function clearDraft() {
    draft.value = null
  }

  return {
    draft,
    setRemixDraft,
    setReferenceDraft,
    setPromptDraft,
    consumeDraft,
    clearDraft
  }
})
