<script setup>
import { computed, h, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import {
  UserOutlined,
  FileSearchOutlined,
  CameraOutlined,
  LockOutlined,
  ReloadOutlined
} from '@ant-design/icons-vue'
import { useAdminInspiration } from '../composables/useAdminInspiration'

const {
  getStoredAdminToken,
  saveAdminToken,
  listAdminInspirations,
  reviewInspiration
} = useAdminInspiration()

const adminToken = ref(getStoredAdminToken())
const loginTokenInput = ref(adminToken.value)
const authLoading = ref(false)

const activeModule = ref('inspiration_review')
const loading = ref(false)
const activePostID = ref(0)
const items = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const reviewStatus = ref('pending')
const userID = ref('')
const keyword = ref('')
const startDate = ref('')
const endDate = ref('')

const moduleMenuItems = [
  { key: 'inspiration_review', icon: () => h(FileSearchOutlined), label: '灵感内容审核' },
  { key: 'user_list', icon: () => h(UserOutlined), label: '用户列表' },
  { key: 'generation_list', icon: () => h(CameraOutlined), label: '生成列表' }
]

const reviewStatusOptions = [
  { label: '待审核', value: 'pending' },
  { label: '全部', value: 'all' },
  { label: '已通过', value: 'approved' },
  { label: '已驳回', value: 'rejected' }
]

const hasAdminToken = computed(() => !!adminToken.value)
const pendingCountInPage = computed(() => items.value.filter((item) => item.review_status === 'pending').length)
const approvedCountInPage = computed(() => items.value.filter((item) => item.review_status === 'approved').length)
const rejectedCountInPage = computed(() => items.value.filter((item) => item.review_status === 'rejected').length)

const moduleTitle = computed(() => {
  if (activeModule.value === 'user_list') return '用户列表'
  if (activeModule.value === 'generation_list') return '生成列表'
  return '灵感内容审核'
})

const moduleSubTitle = computed(() => {
  if (activeModule.value === 'user_list') return '账号、状态与权限管理模块'
  if (activeModule.value === 'generation_list') return '任务、产物和状态追踪模块'
  return '发布内容审核、筛选与处理'
})

const isVideoPost = (item) => item?.type === 'video' || !!item?.video_url

const statusText = (status) => {
  if (status === 'approved') return '已通过'
  if (status === 'rejected') return '已驳回'
  return '待审核'
}

const statusColor = (status) => {
  if (status === 'approved') return 'success'
  if (status === 'rejected') return 'error'
  return 'processing'
}

const formatTime = (timestamp) => {
  if (!timestamp) return '-'
  return new Date(timestamp).toLocaleString('zh-CN', { hour12: false })
}

const extractError = (error, fallback) => error?.response?.data?.error || fallback
const isAuthError = (error) => [401, 403, 503].includes(error?.response?.status)

const clearTableState = () => {
  items.value = []
  total.value = 0
  page.value = 1
}

const logout = (withNotice = true) => {
  saveAdminToken('')
  adminToken.value = ''
  loginTokenInput.value = ''
  clearTableState()
  if (withNotice) message.info('已退出登录')
}

const verifyToken = async () => {
  await listAdminInspirations({
    limit: 1,
    offset: 0,
    review_status: 'all'
  })
}

const authenticate = async () => {
  const token = loginTokenInput.value.trim()
  if (!token) {
    message.warning('请输入管理端 Token')
    return
  }

  authLoading.value = true
  saveAdminToken(token)
  adminToken.value = getStoredAdminToken()
  try {
    await verifyToken()
    message.success('登录成功')
    if (activeModule.value === 'inspiration_review') {
      await fetchList(true)
    }
  } catch (error) {
    logout(false)
    message.error(extractError(error, 'Token 无效或服务不可用'))
  } finally {
    authLoading.value = false
  }
}

const fetchList = async (resetPage = false) => {
  if (!hasAdminToken.value) {
    clearTableState()
    return
  }
  if (resetPage) page.value = 1

  loading.value = true
  try {
    const offset = (page.value - 1) * pageSize.value
    const data = await listAdminInspirations({
      limit: pageSize.value,
      offset,
      review_status: reviewStatus.value,
      user_id: userID.value.trim(),
      q: keyword.value.trim(),
      start_date: startDate.value,
      end_date: endDate.value
    })
    items.value = data.items || []
    total.value = data.total || 0
  } catch (error) {
    if (isAuthError(error)) {
      logout(false)
      message.error('登录已失效，请重新输入 Token')
      return
    }
    clearTableState()
    message.error(extractError(error, '加载审核列表失败'))
  } finally {
    loading.value = false
  }
}

const onSearch = async () => {
  await fetchList(true)
}

const onReset = async () => {
  reviewStatus.value = 'pending'
  userID.value = ''
  keyword.value = ''
  startDate.value = ''
  endDate.value = ''
  await fetchList(true)
}

const onPageChange = async (nextPage) => {
  page.value = nextPage
  await fetchList(false)
}

const onModuleSelect = async ({ key }) => {
  activeModule.value = String(key)
  if (activeModule.value === 'inspiration_review' && hasAdminToken.value) {
    await fetchList(true)
  }
}

const refreshCurrent = async () => {
  if (activeModule.value !== 'inspiration_review') return
  await fetchList(false)
}

const doReview = async (item, action) => {
  if (!item?.id) return

  let note = ''
  if (action === 'reject') {
    const value = window.prompt('请输入驳回原因（可选）', '')
    if (value === null) return
    note = value.trim()
  }

  activePostID.value = item.id
  try {
    const data = await reviewInspiration(item.id, { action, note })
    const next = data?.item
    if (next) {
      const index = items.value.findIndex((row) => row.id === next.id)
      if (index >= 0) items.value[index] = next
    }
    message.success(action === 'approve' ? '审核通过' : '已驳回')
  } catch (error) {
    if (isAuthError(error)) {
      logout(false)
      message.error('登录已失效，请重新输入 Token')
      return
    }
    message.error(extractError(error, '提交审核失败'))
  } finally {
    activePostID.value = 0
  }
}

const columns = [
  { title: 'ID', dataIndex: 'id', width: 90 },
  { title: '封面', dataIndex: 'cover', width: 120 },
  { title: '标题', dataIndex: 'title' },
  { title: '作者', dataIndex: 'author', width: 180 },
  { title: '状态', dataIndex: 'review_status', width: 120 },
  { title: '发布时间', dataIndex: 'published_at', width: 190 },
  { title: '操作', dataIndex: 'actions', width: 180 }
]

onMounted(async () => {
  if (hasAdminToken.value && activeModule.value === 'inspiration_review') {
    loginTokenInput.value = adminToken.value
    await fetchList(true)
  }
})
</script>

<template>
  <div v-if="!hasAdminToken" class="login-shell">
    <div class="login-grid">
      <section class="intro-panel">
        <span class="intro-badge">ADMIN PANEL</span>
        <h1>NanoBanana 管理端</h1>
        <p>统一管理灵感审核、用户与生成记录。当前版本先开放 Token 登录与审核工作台。</p>
        <ul>
          <li>灵感内容审核</li>
          <li>用户列表（预留）</li>
          <li>生成列表（预留）</li>
        </ul>
      </section>

      <a-card class="login-card" :bordered="false">
        <div class="login-head">
          <span class="logo-mark">NB</span>
          <div>
            <h2>管理端登录</h2>
            <p>请输入后端配置的 ADMIN_TOKEN</p>
          </div>
        </div>
        <a-alert type="info" show-icon message="仅授权管理员可访问此后台" class="login-alert" />
        <a-input-password
          v-model:value="loginTokenInput"
          size="large"
          :prefix="h(LockOutlined)"
          placeholder="请输入 ADMIN_TOKEN"
          @press-enter="authenticate"
        />
        <a-button type="primary" size="large" block :loading="authLoading" @click="authenticate">登录</a-button>
      </a-card>
    </div>
  </div>

  <a-layout v-else class="admin-layout">
    <a-layout-sider :width="232" theme="dark" class="admin-sider">
      <div class="sider-logo">
        <span class="logo-mark">NB</span>
        <div class="logo-text">
          <strong>NanoBanana</strong>
          <span>Admin Workspace</span>
        </div>
      </div>
      <a-menu
        theme="dark"
        mode="inline"
        :items="moduleMenuItems"
        :selected-keys="[activeModule]"
        @select="onModuleSelect"
      />
    </a-layout-sider>

    <a-layout>
      <a-layout-header class="admin-header">
        <div class="header-left">
          <h2>{{ moduleTitle }}</h2>
          <span>{{ moduleSubTitle }}</span>
        </div>
        <div class="header-right">
          <a-button :icon="h(ReloadOutlined)" @click="refreshCurrent" />
          <a-tag color="blue">已登录</a-tag>
          <a-button @click="logout">退出登录</a-button>
        </div>
      </a-layout-header>

      <a-layout-content class="admin-content">
        <template v-if="activeModule === 'inspiration_review'">
          <a-row :gutter="16" class="mb16">
            <a-col :xs="24" :sm="12" :md="6">
              <a-card class="stat-card" size="small">
                <span class="stat-label">总条数</span>
                <a-statistic :value="total" />
              </a-card>
            </a-col>
            <a-col :xs="24" :sm="12" :md="6">
              <a-card class="stat-card" size="small">
                <span class="stat-label">当前页待审核</span>
                <a-statistic :value="pendingCountInPage" />
              </a-card>
            </a-col>
            <a-col :xs="24" :sm="12" :md="6">
              <a-card class="stat-card" size="small">
                <span class="stat-label">当前页已通过</span>
                <a-statistic :value="approvedCountInPage" />
              </a-card>
            </a-col>
            <a-col :xs="24" :sm="12" :md="6">
              <a-card class="stat-card" size="small">
                <span class="stat-label">当前页已驳回</span>
                <a-statistic :value="rejectedCountInPage" />
              </a-card>
            </a-col>
          </a-row>

          <a-card class="mb16 filter-card">
            <a-form layout="inline">
              <a-form-item label="审核状态">
                <a-select v-model:value="reviewStatus" style="width: 130px" :options="reviewStatusOptions" />
              </a-form-item>
              <a-form-item label="用户 ID">
                <a-input v-model:value="userID" allow-clear style="width: 140px" />
              </a-form-item>
              <a-form-item label="关键词">
                <a-input v-model:value="keyword" allow-clear style="width: 260px" placeholder="标题/提示词/share id" />
              </a-form-item>
              <a-form-item label="开始">
                <input v-model="startDate" type="date" class="native-date" />
              </a-form-item>
              <a-form-item label="结束">
                <input v-model="endDate" type="date" class="native-date" />
              </a-form-item>
              <a-form-item>
                <a-button type="primary" :disabled="loading" @click="onSearch">查询</a-button>
              </a-form-item>
              <a-form-item>
                <a-button :disabled="loading" @click="onReset">重置</a-button>
              </a-form-item>
            </a-form>
          </a-card>

          <a-card class="table-card">
            <a-table
              :columns="columns"
              :data-source="items"
              :loading="loading"
              :pagination="false"
              :row-key="(record) => record.id"
              size="middle"
              :scroll="{ x: 1000 }"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.dataIndex === 'cover'">
                  <div class="cover-box">
                    <video
                      v-if="isVideoPost(record) && record.video_url"
                      :src="record.video_url"
                      :poster="record.cover_url"
                      muted
                      playsinline
                      preload="metadata"
                    />
                    <img v-else :src="record.cover_url || record.images?.[0]" alt="cover" loading="lazy" />
                  </div>
                </template>
                <template v-else-if="column.dataIndex === 'title'">
                  <div class="title-cell">
                    <strong>{{ record.title || '未命名内容' }}</strong>
                    <span class="muted">share: {{ record.share_id }}</span>
                  </div>
                </template>
                <template v-else-if="column.dataIndex === 'author'">
                  <div class="title-cell">
                    <span>{{ record.author?.nickname || '-' }}</span>
                    <span class="muted">UID: {{ record.author?.user_id || '-' }}</span>
                  </div>
                </template>
                <template v-else-if="column.dataIndex === 'review_status'">
                  <a-tag :color="statusColor(record.review_status)">
                    {{ statusText(record.review_status) }}
                  </a-tag>
                </template>
                <template v-else-if="column.dataIndex === 'published_at'">
                  {{ formatTime(record.published_at) }}
                </template>
                <template v-else-if="column.dataIndex === 'actions'">
                  <div class="action-cell">
                    <a-button
                      type="primary"
                      size="small"
                      :disabled="record.review_status === 'approved'"
                      :loading="activePostID === record.id"
                      @click="doReview(record, 'approve')"
                    >
                      通过
                    </a-button>
                    <a-button
                      type="primary"
                      danger
                      ghost
                      size="small"
                      :disabled="record.review_status === 'rejected'"
                      :loading="activePostID === record.id"
                      @click="doReview(record, 'reject')"
                    >
                      驳回
                    </a-button>
                  </div>
                </template>
              </template>
            </a-table>
            <div class="table-pagination">
              <a-pagination
                :current="page"
                :page-size="pageSize"
                :total="total"
                :show-size-changer="false"
                @change="onPageChange"
              />
            </div>
          </a-card>
        </template>

        <a-result
          v-else
          status="info"
          :title="moduleTitle + '（建设中）'"
          sub-title="该模块将用于独立的列表检索与管理操作，提供字段后我可以直接补齐。"
        />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<style scoped>
.login-shell {
  min-height: 100vh;
  padding: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(circle at 10% 15%, rgba(22, 119, 255, 0.18), transparent 36%),
    radial-gradient(circle at 84% 5%, rgba(0, 21, 41, 0.16), transparent 42%),
    #f3f6fb;
}

.login-grid {
  width: 100%;
  max-width: 1080px;
  display: grid;
  grid-template-columns: 1.1fr 1fr;
  gap: 18px;
}

.intro-panel {
  border-radius: 16px;
  padding: 34px;
  color: #fff;
  background:
    radial-gradient(circle at 20% 20%, rgba(126, 194, 255, 0.3), transparent 40%),
    linear-gradient(150deg, #0a315f 0%, #0b3a73 50%, #1b4f9d 100%);
  box-shadow: 0 16px 40px rgba(16, 58, 110, 0.25);
}

.intro-badge {
  font-size: 11px;
  letter-spacing: 0.12em;
  opacity: 0.78;
}

.intro-panel h1 {
  margin: 14px 0 10px;
  font-size: 36px;
  line-height: 1.15;
}

.intro-panel p {
  margin: 0;
  font-size: 15px;
  color: rgba(255, 255, 255, 0.86);
}

.intro-panel ul {
  margin: 22px 0 0;
  padding-left: 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.login-card {
  border-radius: 16px;
  box-shadow: 0 12px 36px rgba(0, 0, 0, 0.08);
}

.login-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.login-head h2 {
  margin: 0;
  font-size: 22px;
}

.login-head p {
  margin: 2px 0 0;
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
}

.login-alert {
  margin-bottom: 12px;
}

.admin-layout {
  min-height: 100vh;
  background: #f3f5f9;
}

.admin-sider {
  box-shadow: 2px 0 10px rgba(0, 21, 41, 0.35);
}

.sider-logo {
  height: 64px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 16px;
}

.logo-mark {
  width: 30px;
  height: 30px;
  border-radius: 7px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  color: #001529;
  background: linear-gradient(135deg, #8fccff, #ffffff);
}

.logo-text {
  display: flex;
  flex-direction: column;
  color: #fff;
  line-height: 1.1;
}

.logo-text strong {
  font-size: 13px;
}

.logo-text span {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.72);
}

.admin-header {
  height: 64px;
  padding: 0 20px;
  background: #fff;
  border-bottom: 1px solid #e9edf3;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.header-left h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.88);
}

.header-left span {
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.admin-content {
  padding: 16px;
}

.mb16 {
  margin-bottom: 16px;
}

.stat-card {
  border: 1px solid #eef2f8;
}

.stat-label {
  display: block;
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
  margin-bottom: 4px;
}

.filter-card,
.table-card {
  border: 1px solid #edf1f7;
}

.cover-box {
  width: 72px;
  height: 72px;
  border-radius: 8px;
  overflow: hidden;
  background: #f5f5f5;
  border: 1px solid #f0f0f0;
}

.cover-box img,
.cover-box video {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.title-cell {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.title-cell strong {
  color: rgba(0, 0, 0, 0.88);
}

.muted {
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
}

.action-cell {
  display: flex;
  gap: 8px;
}

.table-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.native-date {
  width: 138px;
  height: 32px;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  padding: 0 10px;
  background: #fff;
}

:deep(.ant-table-tbody > tr:hover > td) {
  background: #f7faff !important;
}

@media (max-width: 1200px) {
  .login-grid {
    grid-template-columns: 1fr;
  }

  .intro-panel {
    min-height: 220px;
  }
}

@media (max-width: 900px) {
  .admin-header {
    padding: 0 12px;
  }

  .header-right {
    gap: 6px;
  }

  .admin-content {
    padding: 12px;
  }
}

@media (max-width: 640px) {
  .login-shell {
    padding: 12px;
  }

  .intro-panel {
    padding: 20px;
  }

  .intro-panel h1 {
    font-size: 28px;
  }
}
</style>
