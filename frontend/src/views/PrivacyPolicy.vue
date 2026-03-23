<template>
  <main class="legal-page">
    <article class="legal-card">
      <h1>隐私政策</h1>
      <p class="meta">更新日期：2026-02-13</p>

      <h2>1. 我们收集的信息</h2>
      <p>为提供服务，我们可能收集您提交的账户信息、登录信息、设备与日志信息，以及您在平台内上传或生成的内容数据。</p>

      <h2>2. 信息使用目的</h2>
      <p>我们将信息用于账户认证、服务交付、功能优化、安全风控、客户支持及符合法律法规要求的场景。</p>

      <h2>3. 信息共享与披露</h2>
      <p>未经您同意，我们不会向无关第三方出售您的个人信息。仅在法律法规要求、保护合法权益或履行服务所必需时进行共享或披露。</p>

      <h2>4. 信息存储与保护</h2>
      <p>我们采取合理的技术和管理措施保护您的数据安全，防止未经授权的访问、篡改、泄露或丢失。</p>

      <h2>5. Cookie 与类似技术</h2>
      <p>平台可能使用 Cookie 或本地存储来维持登录状态、记住偏好并提升使用体验。您可在浏览器中管理相关设置。</p>

      <h2>6. 您的权利</h2>
      <p>在适用法律范围内，您可申请查询、更正、删除相关个人信息，或申请注销账户。我们将在合理期限内处理您的请求。</p>

      <h2>7. 未成年人保护</h2>
      <p>若您为未成年人，应在监护人指导下使用本平台服务。</p>

      <h2>8. 政策更新</h2>
      <p>本政策可能根据法律法规或业务调整而更新，更新后将通过页面公示。</p>

      <h2>9. 联系方式</h2>
      <p>如您对本政策有疑问，可通过平台内公告或客服渠道与我们联系。</p>

      <router-link to="/" class="back-link">返回首页</router-link>
    </article>
  </main>
</template>

<style scoped>
.legal-page {
  min-height: 100vh;
  padding: 32px 16px;
  background: var(--color-bg-deep);
}

.legal-card {
  width: min(920px, 100%);
  margin: 0 auto;
  padding: 28px 22px;
  border-radius: 16px;
  border: 1px solid var(--color-border-medium);
  background: var(--color-card-solid);
  color: var(--color-text-primary);
  line-height: 1.7;
}

h1 {
  margin: 0 0 6px;
  font-size: 30px;
}

.meta {
  margin: 0 0 22px;
  color: var(--color-text-muted);
  font-size: 13px;
}

h2 {
  margin: 20px 0 6px;
  font-size: 18px;
}

p {
  margin: 0;
  color: var(--color-text-secondary);
}

.back-link {
  display: inline-block;
  margin-top: 26px;
  color: var(--color-accent);
  text-decoration: none;
  font-weight: 600;
}

.back-link:hover {
  text-decoration: underline;
}
</style>
