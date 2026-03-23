import { ref } from 'vue'
import axios from 'axios'
import { useUserStore } from '../stores/user'

export function useGenerate() {
  const userStore = useUserStore()
  const activePolls = ref({})

  // 统一生成接口（支持 image/video/ecommerce）
  async function generate(type, payload) {
    const response = await axios.post('/api/generate', {
      type,
      ...payload
    }, {
      headers: { Authorization: `Bearer ${userStore.token}` },
      timeout: 30000
    })
    return { task_id: response.data.task_id }
  }

  // 兼容旧接口
  async function generateImage(payload) {
    return generate('image', payload)
  }

  async function generateEcommerce(payload) {
    return generate('ecommerce', payload)
  }

  async function generateVideo(payload) {
    return generate('video', payload)
  }

  // 获取单个生成记录详情（统一查询接口）
  async function getGeneration(id) {
    const { data } = await axios.get(`/api/generations/${id}`, {
      headers: { Authorization: `Bearer ${userStore.token}` }
    })
    return data
  }

  // 统一的轮询任务状态（支持 image/ecommerce/video）
  async function pollTask(taskId, onUpdate) {
    if (activePolls.value[taskId]) return
    activePolls.value[taskId] = true

    while (activePolls.value[taskId]) {
      try {
        // 使用 generations API 查询状态（通用的生成记录查询）
        const { data: gen } = await axios.get(`/api/generations/${taskId}`, {
          headers: { Authorization: `Bearer ${userStore.token}` }
        })
        if (!activePolls.value[taskId]) break

        // 状态映射：success -> 完成，failed -> 失败，其他 -> 进行中
        if (gen.status === 'success') {
          delete activePolls.value[taskId]
          onUpdate({ 
            status: 'success', 
            images: gen.images || [], 
            video_url: gen.video_url,
            credits_spent: gen.credits_cost 
          })
          return
        }
        if (gen.status === 'failed') {
          delete activePolls.value[taskId]
          onUpdate({ status: 'failed', error_msg: gen.error_msg || '生成失败' })
          return
        }
        // generating, queued, running 都显示为进行中
        onUpdate({ status: gen.status || 'generating' })
      } catch (e) {
        console.error(`[Poll] 网络错误 task=${taskId}:`, e?.message || e)
      }
      await new Promise(r => setTimeout(r, 3000))  // 3秒轮询一次
    }
  }

  // 兼容旧接口
  async function pollVideoTask(taskId, onUpdate) {
    return pollTask(taskId, onUpdate)
  }

  function stopAllPolls() {
    Object.keys(activePolls.value).forEach(k => delete activePolls.value[k])
  }

  async function uploadImageToOSS(base64Data) {
    const response = await axios.post('/api/user/upload/image', {
      image: base64Data.split(',')[1]
    }, {
      headers: { Authorization: `Bearer ${userStore.token}` }
    })
    return response.data.url
  }

  async function optimizePrompt(payload) {
    const response = await axios.post('/api/prompt/optimize', payload, {
      headers: { Authorization: `Bearer ${userStore.token}` },
      timeout: 45000
    })
    return response.data
  }

  async function reversePrompt(payload) {
    const response = await axios.post('/api/tools/reverse-prompt', payload, {
      headers: { Authorization: `Bearer ${userStore.token}` },
      timeout: 60000
    })
    return response.data
  }

  return {
    generate,
    generateImage,
    generateEcommerce,
    generateVideo,
    pollTask,
    pollVideoTask,
    getGeneration,
    stopAllPolls,
    uploadImageToOSS,
    optimizePrompt,
    reversePrompt,
    activePolls
  }
}
