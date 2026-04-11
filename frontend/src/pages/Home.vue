<template>
  <div class="home-container" :class="`is-${resolvedTheme}`">
    <div class="theme-switcher" role="group" aria-label="主题切换">
      <button
        v-for="option in themeOptions"
        :key="option.value"
        class="theme-switcher__item"
        :class="{ 'is-active': themeMode === option.value }"
        :title="option.tooltip"
        @click="setThemeMode(option.value)"
      >
        <component :is="option.icon" class="theme-switcher__icon" />
        <span>{{ option.label }}</span>
      </button>
    </div>
    <div class="hero-panel">
      <div class="home-header">
        <div class="brand-line">
          <img class="brand-badge" src="/neko-badge.svg" alt="Neko Tool 猫咪徽记" />
          <div>
            <div class="eyebrow">Neko Tool</div>
            <h1>喵咪驻场的多节点项目工作台</h1>
          </div>
        </div>
        <p class="subtitle">原创手绘动画感猫咪视觉，配合项目管理、节点管理与比较工作台，让整个工具更像一只会巡逻代码仓库的猫。</p>
      </div>
      <div class="hero-art-wrap">
        <img class="hero-art" src="/neko-hero-cat.svg" alt="原创手绘动画感猫咪插画" />
      </div>
    </div>

    <el-card class="site-info-card neko-surface" v-loading="siteInfoLoading">
      <template #header>
        <div class="site-info-header">
          <span>本站信息</span>
          <div class="site-info-actions">
            <el-button size="small" :icon="Refresh" @click="fetchSiteInfo">刷新</el-button>
            <el-button size="small" type="danger" plain @click="handleLogout">退出认证</el-button>
          </div>
        </div>
      </template>
      <div class="site-info-grid">
        <div class="site-info-item">
          <span class="site-info-label">本工作站名称</span>
          <strong>{{ siteInfo.nodeName || '-' }}</strong>
        </div>
        <div class="site-info-item">
          <span class="site-info-label">监听地址</span>
          <strong>{{ siteInfo.webAddress || '-' }}</strong>
        </div>
        <div class="site-info-item">
          <span class="site-info-label">认证状态</span>
          <el-tag :type="siteInfo.authEnabled ? 'success' : 'warning'">{{ siteInfo.authEnabled ? '已启用' : '未启用' }}</el-tag>
        </div>
        <div class="site-info-item">
          <span class="site-info-label">运行时长</span>
          <strong>{{ uptimeLabel }}</strong>
        </div>
        <div class="site-info-item">
          <span class="site-info-label">共享 token</span>
          <div class="site-token-row">
            <strong>{{ siteInfo.sharedToken || '-' }}</strong>
            <el-button size="small" @click="copySharedToken">一键复制</el-button>
          </div>
        </div>
        <div class="site-info-item">
          <span class="site-info-label">当前版本</span>
          <strong>{{ siteInfo.version || '-' }}</strong>
        </div>
      </div>
      <div class="open-source-note">
        <span class="open-source-note__label">开源说明</span>
        <p>Neko Tool 当前以开源项目方式维护，仓库地址：</p>
        <a
          class="open-source-note__link"
          href="https://github.com/1344812937/neko-tool-release.git"
          target="_blank"
          rel="noreferrer"
        >
          https://github.com/1344812937/neko-tool-release.git
        </a>
      </div>
    </el-card>

    <div class="card-grid">
      <el-card class="nav-card" shadow="hover" @click="router.push('/projects')">
        <span class="card-paw">爪</span>
        <div class="card-content">
          <el-icon :size="48" color="#409EFF"><Folder /></el-icon>
          <h3>项目管理</h3>
          <p>维护本地项目和受限目录选择</p>
        </div>
      </el-card>

      <el-card class="nav-card" shadow="hover" @click="router.push('/nodes')">
        <span class="card-paw">喵</span>
        <div class="card-content">
          <el-icon :size="48" color="#67C23A"><Connection /></el-icon>
          <h3>节点管理</h3>
          <p>配置远程服务器和连通性探测</p>
        </div>
      </el-card>

      <el-card class="nav-card" shadow="hover" @click="router.push('/compare')">
        <span class="card-paw">爪</span>
        <div class="card-content">
          <el-icon :size="48" color="#E6A23C"><Files /></el-icon>
          <h3>比较工作台</h3>
          <p>比较项目差异并执行提交、拉取</p>
        </div>
      </el-card>

      <el-card class="nav-card" shadow="hover" @click="router.push('/site/logs')">
        <span class="card-paw">喵</span>
        <div class="card-content">
          <el-icon :size="48" color="#f56c6c"><Document /></el-icon>
          <h3>本站日志</h3>
          <p>按时间查看当前环境全部项目操作日志</p>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Connection, Document, Files, Folder, Monitor, Moon, Refresh, Sunny } from '@element-plus/icons-vue'
import { currentRoutePath, logoutAndRedirect } from '@/utils/auth'
import { resolvedTheme, setThemeMode, themeMode } from '@/utils/theme'

const router = useRouter()
const siteInfoLoading = ref(false)
const siteInfo = ref({ startedAt: '', uptimeSeconds: 0, nodeName: '', webAddress: '', authEnabled: false, sharedToken: '', version: '' })
const uptimeSeconds = ref(0)
let uptimeTimer = null

const themeOptions = [
  { value: 'light', label: '明', icon: Sunny, tooltip: '切换到明亮模式' },
  { value: 'dark', label: '暗', icon: Moon, tooltip: '切换到暗黑模式' },
  { value: 'auto', label: '自', icon: Monitor, tooltip: '切换到自动模式' },
]

const uptimeLabel = computed(() => formatDuration(uptimeSeconds.value))

async function fetchSiteInfo() {
  siteInfoLoading.value = true
  try {
    const res = await fetch('/api/site/info')
    const data = await res.json()
    if (!data.success) {
      throw new Error(data.message || '加载本站信息失败')
    }
    siteInfo.value = data.data || siteInfo.value
    uptimeSeconds.value = Number(siteInfo.value.uptimeSeconds || 0)
    startUptimeTimer()
  } catch (error) {
    ElMessage.error(error.message || '加载本站信息失败')
  } finally {
    siteInfoLoading.value = false
  }
}

function startUptimeTimer() {
  if (uptimeTimer) {
    window.clearInterval(uptimeTimer)
  }
  uptimeTimer = window.setInterval(() => {
    uptimeSeconds.value += 1
  }, 1000)
}

function formatDuration(totalSeconds) {
  const seconds = Math.max(0, Number(totalSeconds || 0))
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const remainSeconds = seconds % 60
  if (days > 0) return `${days} 天 ${hours} 小时 ${minutes} 分`
  if (hours > 0) return `${hours} 小时 ${minutes} 分 ${remainSeconds} 秒`
  if (minutes > 0) return `${minutes} 分 ${remainSeconds} 秒`
  return `${remainSeconds} 秒`
}

async function copySharedToken() {
  if (!siteInfo.value.sharedToken) {
    return
  }
  try {
    await navigator.clipboard.writeText(siteInfo.value.sharedToken)
    ElMessage.success('共享 token 已复制')
  } catch {
    ElMessage.error('复制共享 token 失败')
  }
}

function handleLogout() {
  logoutAndRedirect(currentRoutePath())
}

onMounted(fetchSiteInfo)

onBeforeUnmount(() => {
  if (uptimeTimer) {
    window.clearInterval(uptimeTimer)
  }
})

document.title = 'Neko Tool'
</script>

<style scoped>
.home-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 84px 24px 28px;
  background:
    radial-gradient(circle at top left, rgba(255, 235, 188, 0.95), transparent 36%),
    radial-gradient(circle at bottom right, rgba(223, 188, 135, 0.45), transparent 32%),
    linear-gradient(180deg, #fffaf0 0%, #f6ead8 100%);
  transition: background 0.35s ease;
}

.theme-switcher {
  position: fixed;
  top: 18px;
  right: 20px;
  z-index: 1200;
  display: inline-flex;
  gap: 6px;
  padding: 6px;
  border-radius: 999px;
  border: 1px solid rgba(126, 92, 61, 0.12);
  background: rgba(255, 250, 243, 0.82);
  box-shadow: 0 12px 28px rgba(88, 61, 36, 0.14);
  backdrop-filter: blur(14px);
}

.theme-switcher__item {
  border: 0;
  background: transparent;
  color: #7d5b44;
  width: 42px;
  height: 42px;
  border-radius: 999px;
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1px;
  cursor: pointer;
  transition: transform 0.18s ease, background-color 0.18s ease, color 0.18s ease;
  font-size: 11px;
  font-weight: 700;
}

.theme-switcher__item:hover {
  transform: translateY(-1px);
  background: rgba(255, 240, 219, 0.8);
}

.theme-switcher__item.is-active {
  background: linear-gradient(180deg, #f0c98f 0%, #dfa66f 100%);
  color: #4f3527;
}

.theme-switcher__icon {
  width: 16px;
  height: 16px;
}

.hero-panel {
  width: min(1120px, 100%);
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(320px, 0.95fr);
  gap: 28px;
  align-items: center;
  margin-bottom: 34px;
  padding: 24px;
  border-radius: 36px;
  background: rgba(255, 250, 243, 0.45);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.45);
  backdrop-filter: blur(10px);
}

.home-header {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.brand-line {
  display: flex;
  align-items: center;
  gap: 20px;
}

.brand-badge {
  width: 104px;
  height: 104px;
  flex-shrink: 0;
  filter: drop-shadow(0 12px 24px rgba(126, 84, 46, 0.18));
}

.eyebrow {
  display: inline-flex;
  align-items: center;
  padding: 6px 12px;
  border-radius: 999px;
  background: rgba(141, 94, 60, 0.12);
  color: #8b5e3c;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  margin-bottom: 10px;
}

.home-header h1 {
  max-width: 580px;
  font-size: clamp(32px, 4vw, 54px);
  line-height: 1.08;
  font-weight: 700;
  color: #4f3527;
  margin: 0;
}

.subtitle {
  max-width: 620px;
  font-size: 16px;
  line-height: 1.8;
  color: #745747;
}

.hero-art-wrap {
  display: flex;
  justify-content: center;
}

.hero-art {
  width: min(100%, 520px);
  border-radius: 32px;
  box-shadow: 0 24px 40px rgba(117, 81, 48, 0.18);
}

.card-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  justify-content: center;
  max-width: 1040px;
  width: 100%;
}

.site-info-card {
  width: min(1120px, 100%);
  margin-bottom: 24px;
}

.site-info-card :deep(.el-card__header) {
  backdrop-filter: blur(12px);
}

.site-info-card :deep(.el-card__body) {
  padding: 18px;
}

.site-info-header,
.site-info-actions,
.site-token-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.site-info-header {
  justify-content: space-between;
}

.site-info-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.open-source-note {
  margin-top: 18px;
  padding: 18px 20px;
  border-radius: 22px;
  border: 1px dashed rgba(186, 148, 110, 0.36);
  background: rgba(255, 248, 236, 0.72);
  color: #6d5341;
}

.open-source-note__label {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  margin-bottom: 10px;
  border-radius: 999px;
  background: rgba(141, 94, 60, 0.12);
  color: #8b5e3c;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.04em;
}

.open-source-note p {
  margin: 0 0 8px;
  line-height: 1.7;
}

.open-source-note__link {
  color: #b45b2f;
  font-weight: 700;
  text-decoration: none;
  word-break: break-all;
}

.open-source-note__link:hover {
  text-decoration: underline;
}

.site-info-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  padding: 16px 18px;
  border-radius: 22px;
  border: 1px solid rgba(186, 148, 110, 0.12);
  background: var(--neko-surface-panel-bg);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.45);
}

.site-info-item :deep(.el-tag) {
  align-self: flex-start;
}

.site-info-label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.site-token-row strong,
.site-info-item strong {
  overflow: hidden;
  text-overflow: ellipsis;
  color: #584535;
  font-weight: 700;
}

.site-token-row {
  justify-content: space-between;
}

.site-info-card :deep(.el-button) {
  border-color: rgba(191, 152, 112, 0.2);
}

.home-container.is-dark .site-info-item {
  border-color: rgba(230, 201, 164, 0.1);
  background: var(--neko-surface-panel-bg);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.home-container.is-dark .site-info-label {
  color: #baa993;
}

.home-container.is-dark .site-token-row strong,
.home-container.is-dark .site-info-item strong {
  color: #f0dec9;
}

.home-container.is-dark .site-info-card :deep(.el-button) {
  border-color: rgba(235, 201, 154, 0.12);
}

.home-container.is-dark .open-source-note {
  border-color: rgba(230, 201, 164, 0.18);
  background: rgba(46, 39, 35, 0.72);
  color: #e2cfbb;
}

.home-container.is-dark .open-source-note__label {
  background: rgba(240, 190, 131, 0.14);
  color: #f0c995;
}

.home-container.is-dark .open-source-note__link {
  color: #f0be83;
}

@media (max-width: 920px) {
  .site-info-grid {
    grid-template-columns: 1fr;
  }

  .site-info-header {
    flex-direction: column;
    align-items: flex-start;
  }
}

.nav-card {
  position: relative;
  width: 240px;
  cursor: pointer;
  transition: transform 0.22s, border-color 0.22s, box-shadow 0.22s;
  border: 1px solid rgba(162, 120, 76, 0.14);
  background: rgba(255, 251, 244, 0.85);
  overflow: hidden;
}

.nav-card:hover {
  transform: translateY(-6px) rotate(-0.4deg);
  border-color: rgba(173, 116, 64, 0.45);
  box-shadow: 0 16px 28px rgba(121, 84, 53, 0.14);
}

.card-paw {
  position: absolute;
  top: 14px;
  right: 16px;
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: rgba(255, 239, 213, 0.9);
  color: #a3643b;
  font-size: 12px;
  font-weight: 700;
}

.card-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 16px 0;
}

.card-content h3 {
  font-size: 16px;
  font-weight: 700;
  color: #52392a;
  margin: 0;
}

.card-content p {
  font-size: 13px;
  color: #7a5f4c;
  margin: 0;
  text-align: center;
}

.home-container.is-dark {
  background:
    radial-gradient(circle at top left, rgba(110, 90, 62, 0.34), transparent 34%),
    radial-gradient(circle at bottom right, rgba(70, 92, 109, 0.28), transparent 32%),
    linear-gradient(180deg, #17181d 0%, #23252d 100%);
}

.home-container.is-dark .theme-switcher {
  border-color: rgba(240, 214, 180, 0.14);
  background: rgba(33, 34, 40, 0.72);
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.32);
}

.home-container.is-dark .theme-switcher__item {
  color: #f1d8c0;
}

.home-container.is-dark .theme-switcher__item:hover {
  background: rgba(115, 95, 80, 0.28);
}

.home-container.is-dark .theme-switcher__item.is-active {
  background: linear-gradient(180deg, #f0be83 0%, #cf8e55 100%);
  color: #33231a;
}

.home-container.is-dark .hero-panel {
  background: rgba(34, 36, 42, 0.72);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.05), 0 24px 40px rgba(0, 0, 0, 0.22);
}

.home-container.is-dark .brand-badge {
  filter: drop-shadow(0 12px 24px rgba(0, 0, 0, 0.22));
}

.home-container.is-dark .eyebrow {
  background: rgba(239, 189, 126, 0.14);
  color: #f0c68a;
}

.home-container.is-dark .home-header h1 {
  color: #f7ead8;
}

.home-container.is-dark .subtitle {
  color: #ccb79f;
}

.home-container.is-dark .hero-art {
  box-shadow: 0 28px 44px rgba(0, 0, 0, 0.28);
}

.home-container.is-dark .nav-card {
  border-color: rgba(241, 216, 189, 0.12);
  background: rgba(31, 33, 38, 0.86);
}

.home-container.is-dark .nav-card:hover {
  border-color: rgba(232, 181, 121, 0.48);
  box-shadow: 0 18px 30px rgba(0, 0, 0, 0.24);
}

.home-container.is-dark .card-paw {
  background: rgba(242, 194, 138, 0.12);
  color: #f1cb9a;
}

.home-container.is-dark .card-content h3 {
  color: #f4e4d4;
}

.home-container.is-dark .card-content p {
  color: #c3af9c;
}

@media (max-width: 900px) {
  .hero-panel {
    grid-template-columns: 1fr;
  }

  .brand-line {
    align-items: flex-start;
  }

  .brand-badge {
    width: 88px;
    height: 88px;
  }
}

@media (max-width: 640px) {
  .home-container {
    padding-inline: 16px;
    padding-top: 84px;
  }

  .theme-switcher {
    top: 14px;
    right: 14px;
  }

  .brand-line {
    flex-direction: column;
    gap: 12px;
  }

  .home-header h1 {
    font-size: 32px;
  }

  .nav-card {
    width: 100%;
    max-width: 340px;
  }
}
</style>
