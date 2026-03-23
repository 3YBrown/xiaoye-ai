<template>
  <main class="legal-page">
    <article class="legal-card">
      <h1>用户协议</h1>
      <p class="meta">更新日期：2026-02-13</p>

      <h2>1. 协议范围</h2>
      <p>本协议适用于您使用 小野 AI 网站及相关服务的全部行为。您访问、注册或使用本平台，即视为已阅读并同意本协议。</p>

      <h2>2. 账户与安全</h2>
      <p>您应提供真实、准确、完整的注册信息，并妥善保管账户凭证。因您保管不当导致的风险或损失由您自行承担。</p>

      <h2>3. 服务内容</h2>
      <p>平台提供 AI 内容生成、灵感浏览、作品管理等功能。我们可根据产品迭代需要调整、升级或优化相关服务。</p>

      <h2>4. 用户行为规范</h2>
      <p>您不得利用平台发布、生成、传播违法违规内容，不得侵犯第三方合法权益，不得进行干扰平台正常运行的行为。</p>

      <h2>5. 知识产权</h2>
      <p>平台软件、界面、标识及相关技术成果的知识产权归平台或权利人所有。您上传或生成的内容，其权利归属以法律法规和平台规则为准。</p>

      <h2>6. 责任限制</h2>
      <p>在法律允许范围内，平台不对因网络故障、系统维护、第三方原因或不可抗力导致的服务中断、数据丢失承担超出法定范围的责任。</p>

      <h2>7. 协议变更</h2>
      <p>我们可能根据业务需要更新本协议。更新后将通过页面公示；您继续使用平台即表示接受更新后的协议。</p>

      <h2>8. 联系方式</h2>
      <p>如您对本协议有疑问，可通过平台内公告或客服渠道与我们联系。</p>

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
