<template>
  <div class="site-logs-page">
    <div class="header">
      <div class="header-left">
        <NekoPageHeader
          title="本站操作日志"
          description="汇总当前环境所有项目的文件操作记录，可查看详情并跳到对应项目。"
          tone="orange"
          @back="router.push('/')"
        />
      </div>
      <div class="header-right">
        <el-button :icon="Refresh" :loading="loading" @click="fetchLogs">刷新</el-button>
      </div>
    </div>

    <el-card class="logs-card neko-surface">
      <div v-loading="siteInfoLoading" class="database-overview" element-loading-text="刷新数据库状态中...">
        <div class="database-overview__main">
          <span class="database-overview__label">当前数据库文件大小</span>
          <strong class="database-overview__value">{{ siteInfo.databaseSizeLabel || formatDatabaseSize(siteInfo.databaseSizeBytes) }}</strong>
          <p class="database-overview__hint">清空历史日志后会触发 SQLite 空间整理，当前页面会自动刷新最新体积。</p>
        </div>
        <el-button :icon="Refresh" :loading="siteInfoLoading" @click="fetchSiteInfo">刷新数据库状态</el-button>
      </div>
      <div class="filters-bar">
        <el-input v-model="filters.keyword" clearable placeholder="按文件路径、工作站、操作站地址、IP 关键字筛选" class="filter-input" @keyup.enter="handleSearch" />
        <el-input v-model="filters.projectName" clearable placeholder="按项目名筛选" class="filter-input" @keyup.enter="handleSearch" />
        <el-select v-model="filters.changeType" clearable placeholder="变更类型" class="filter-select">
          <el-option v-for="item in changeTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-button type="primary" @click="handleSearch">查询</el-button>
        <el-button @click="handleReset">重置筛选</el-button>
        <el-button type="danger" plain :loading="cleanupLoading" @click="cleanupLogs">清空历史日志</el-button>
      </div>
      <div class="filters-tip">“重置筛选”只清空当前筛选条件；“清空历史日志”才会删除旧版本日志。</div>
      <el-table :data="logs" v-loading="loading" row-key="id" :max-height="tableMaxHeight">
        <el-table-column prop="operatedAt" label="操作时间" width="180" />
        <el-table-column prop="targetProjectName" label="项目" min-width="180" show-overflow-tooltip />
        <el-table-column prop="relativePath" label="文件路径" min-width="320" show-overflow-tooltip />
        <el-table-column label="变更类型" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="logTypeMap[row.changeType] || 'info'">{{ logLabelMap[row.changeType] || row.changeType }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="executorNodeName" label="工作站" width="140" show-overflow-tooltip />
        <el-table-column prop="executorNodeAddress" label="操作站地址" width="160" show-overflow-tooltip />
        <el-table-column prop="operatorIP" label="操作人 IP" width="140" show-overflow-tooltip />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openLogDetail(row)">查看详情</el-button>
            <el-button type="success" link @click="jumpToProject(row)">跳转项目</el-button>
          </template>
        </el-table-column>
        <template #empty>
          <NekoEmptyState title="还没有项目操作日志" description="当前环境还没有记录到文件修改、同步或本地变更日志。" compact />
        </template>
      </el-table>
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pageNo"
          v-model:page-size="pageSize"
          :page-sizes="[20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="fetchLogs"
          @current-change="fetchLogs"
        />
      </div>
    </el-card>

    <el-dialog v-model="detailVisible" width="1240" :close-on-click-modal="false">
      <template #header>
        <div class="dialog-header">
          <div class="dialog-header-copy">
            <strong>操作日志详情</strong>
            <span>{{ detail?.relativePath || '未选择日志' }}</span>
          </div>
          <el-tag v-if="detail" :type="logTypeMap[detail.changeType] || 'info'">{{ logLabelMap[detail.changeType] || detail.changeType }}</el-tag>
        </div>
      </template>
      <div v-if="detailLoading" class="detail-loading">
        <el-skeleton :rows="8" animated />
      </div>
      <div v-else-if="detail" class="detail-body">
        <el-descriptions border :column="2">
          <el-descriptions-item label="操作时间">{{ detail.operatedAt || detail.modifyTime || '-' }}</el-descriptions-item>
          <el-descriptions-item label="项目">{{ detail.targetProjectName || '-' }}</el-descriptions-item>
          <el-descriptions-item label="文件路径">{{ detail.relativePath || '-' }}</el-descriptions-item>
          <el-descriptions-item label="工作站">{{ detail.executorNodeName || '-' }}</el-descriptions-item>
          <el-descriptions-item label="操作站地址">{{ detail.executorNodeAddress || '-' }}</el-descriptions-item>
          <el-descriptions-item label="操作人 IP">{{ detail.operatorIP || '-' }}</el-descriptions-item>
          <el-descriptions-item label="变更范围">{{ detail.scopeType || '-' }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="canRenderDiff(detail)" class="diff-shell">
          <MonacoDiff :left-content="getText(detail.beforeEncoding, detail.beforeContent)" :right-content="getText(detail.afterEncoding, detail.afterContent)" :language="detectLanguage(detail.relativePath)" />
        </div>
        <div v-else>
          <el-alert :closable="false" type="info" show-icon :title="getLogDetailNotice(detail)" />
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { defineAsyncComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import NekoPageHeader from '@/components/NekoPageHeader.vue'
import NekoEmptyState from '@/components/NekoEmptyState.vue'

const MonacoDiff = defineAsyncComponent(() => import('@/components/MonacoDiff.vue'))

const router = useRouter()

document.title = '本站操作日志'

const loading = ref(false)
const cleanupLoading = ref(false)
const detailLoading = ref(false)
const detailVisible = ref(false)
const detail = ref(null)
const logs = ref([])
const siteInfoLoading = ref(false)
const siteInfo = ref({ databaseSizeBytes: 0, databaseSizeLabel: '' })
const tableMaxHeight = ref(420)
const pageNo = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filters = ref({ keyword: '', projectName: '', changeType: '' })

const logLabelMap = {
  file_changed: '同步覆盖',
  left_only: '左侧新增',
  right_only: '右侧新增',
  local_snapshot: '本地基线',
  local_modified: '本地修改',
  local_deleted: '本地删除',
}

const logTypeMap = {
  file_changed: 'danger',
  left_only: 'primary',
  right_only: 'success',
  local_snapshot: 'info',
  local_modified: 'warning',
  local_deleted: 'danger',
}

const changeTypeOptions = [
  { label: '同步覆盖', value: 'file_changed' },
  { label: '左侧新增', value: 'left_only' },
  { label: '右侧新增', value: 'right_only' },
  { label: '本地基线', value: 'local_snapshot' },
  { label: '本地修改', value: 'local_modified' },
  { label: '本地删除', value: 'local_deleted' },
]

async function fetchLogs() {
  loading.value = true
  try {
    const res = await fetch('/api/site/logs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pageNo: pageNo.value,
        pageSize: pageSize.value,
        keyword: filters.value.keyword,
        projectName: filters.value.projectName,
        changeType: filters.value.changeType,
      }),
    })
    const data = await res.json()
    if (!data.success) {
      throw new Error(data.message || '加载本站操作日志失败')
    }
    logs.value = data.data?.items || []
    total.value = data.data?.total || 0
  } catch (error) {
    ElMessage.error(error.message || '加载本站操作日志失败')
  } finally {
    loading.value = false
  }
}

async function fetchSiteInfo() {
  siteInfoLoading.value = true
  try {
    const res = await fetch('/api/site/info')
    const data = await res.json()
    if (!data.success) {
      throw new Error(data.message || '加载数据库状态失败')
    }
    siteInfo.value = {
      databaseSizeBytes: Number(data.data?.databaseSizeBytes || 0),
      databaseSizeLabel: data.data?.databaseSizeLabel || '',
    }
  } catch (error) {
    ElMessage.error(error.message || '加载数据库状态失败')
  } finally {
    siteInfoLoading.value = false
  }
}

function handleSearch() {
  pageNo.value = 1
  fetchLogs()
}

function handleReset() {
  filters.value = { keyword: '', projectName: '', changeType: '' }
  pageNo.value = 1
  fetchLogs()
}

async function openLogDetail(row) {
  detailLoading.value = true
  detailVisible.value = true
  detail.value = null
  try {
    const res = await fetch('/api/site/log-detail', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ logId: row.id }),
    })
    const data = await res.json()
    if (!data.success) {
      throw new Error(data.message || '加载日志详情失败')
    }
    detail.value = data.data
  } catch (error) {
    detailVisible.value = false
    ElMessage.error(error.message || '加载日志详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function cleanupLogs() {
  try {
    await ElMessageBox.confirm(
      '清理后，每个仍然存在的文件只保留最后一条修改日志；如果文件已经不存在，则该文件的全部日志都会清空。确定继续吗？',
      '清空本站日志',
      {
        type: 'warning',
        confirmButtonText: '确认清空',
        cancelButtonText: '取消',
      },
    )
  } catch {
    return
  }
  cleanupLoading.value = true
  try {
    const res = await fetch('/api/site/logs/cleanup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    })
    const data = await res.json()
    if (!data.success) {
      throw new Error(data.message || '清理本站操作日志失败')
    }
    const result = data.data || {}
    const sizeBefore = result.databaseSizeBeforeLabel || formatDatabaseSize(result.databaseSizeBeforeBytes)
    const sizeAfter = result.databaseSizeAfterLabel || formatDatabaseSize(result.databaseSizeAfterBytes)
    const summary = `已扫描 ${result.filesScanned || 0} 个文件，保留 ${result.keptLogs || 0} 条最新日志，清理 ${result.clearedLogs || 0} 条旧日志，数据库 ${sizeBefore} -> ${sizeAfter}。`
    if (result.vacuumError) {
      ElMessage.warning(`${summary} 日志已清理，但数据库整理失败：${result.vacuumError}`)
    } else {
      ElMessage.success(summary)
    }
    pageNo.value = 1
    await Promise.all([fetchLogs(), fetchSiteInfo()])
  } catch (error) {
    ElMessage.error(error.message || '清理本站操作日志失败')
  } finally {
    cleanupLoading.value = false
  }
}

function formatDatabaseSize(sizeBytes) {
	const numericSize = Number(sizeBytes || 0)
	if (!numericSize) {
		return '0 MB'
	}
	const gb = 1024 * 1024 * 1024
	const mb = 1024 * 1024
	if (numericSize >= gb) {
		return `${(numericSize / gb).toFixed(2)} GB`
	}
	return `${(numericSize / mb).toFixed(2)} MB`
}

function updateTableMaxHeight() {
  if (typeof window === 'undefined') {
    return
  }
  const reservedHeight = 460
  tableMaxHeight.value = Math.max(260, window.innerHeight - reservedHeight)
}

function jumpToProject(row) {
  router.push(`/projects/${row.targetProjectId}/browser?path=${encodeURIComponent(row.relativePath)}`)
}

function canRenderDiff(row) {
  return canPreviewLogSide(row?.beforeEncoding, row?.beforeStorageKind, row?.beforeOmittedReason)
    && canPreviewLogSide(row?.afterEncoding, row?.afterStorageKind, row?.afterOmittedReason)
}

function canPreviewLogSide(encoding, storageKind, omittedReason) {
  if (omittedReason) {
    return false
  }
  const normalizedEncoding = encoding || 'none'
  const normalizedStorageKind = storageKind || 'legacy_full'
  return ['text', 'none', ''].includes(normalizedEncoding)
    && ['full_text', 'compressed_full_text', 'reverse_patch', 'compressed_reverse_patch', 'legacy_full', 'none', ''].includes(normalizedStorageKind)
}

function getText(encoding, content) {
  return encoding === 'text' ? content || '' : ''
}

function getLogDetailNotice(row) {
  const reasons = [row?.beforeOmittedReason, row?.afterOmittedReason].filter(Boolean)
  if (reasons.includes('size_limit')) {
    return '当前文件超过 15MB，日志仅保留摘要信息。'
  }
  if (reasons.includes('binary')) {
    return '当前文件包含二进制内容，日志仅保留摘要信息。'
  }
  return '当前详情无法直接渲染 diff，仅展示元数据。'
}

function detectLanguage(path) {
  const normalized = String(path || '').toLowerCase()
  if (normalized.endsWith('.java')) return 'java'
  if (normalized.endsWith('.go')) return 'go'
  if (normalized.endsWith('.json')) return 'json'
  if (normalized.endsWith('.yml') || normalized.endsWith('.yaml')) return 'yaml'
  if (normalized.endsWith('.ts') || normalized.endsWith('.tsx')) return 'typescript'
  if (normalized.endsWith('.js') || normalized.endsWith('.jsx')) return 'javascript'
  return 'plaintext'
}

onMounted(() => {
  updateTableMaxHeight()
  window.addEventListener('resize', updateTableMaxHeight)
  fetchLogs()
  fetchSiteInfo()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateTableMaxHeight)
})
</script>

<style scoped>
.site-logs-page {
  min-height: 100vh;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  background:
    radial-gradient(circle at top left, rgba(243, 178, 95, 0.09), transparent 28%),
    radial-gradient(circle at bottom right, rgba(95, 140, 216, 0.08), transparent 26%),
    var(--el-bg-color-page);
}

.header,
.header-left,
.header-right,
.pagination-wrapper,
.dialog-header {
  display: flex;
  align-items: center;
}

.header {
  justify-content: space-between;
}

.header-left,
.header-right {
  gap: 12px;
}

.header-left {
  min-width: 0;
  flex: 1;
}

.logs-card {
  flex: 1;
}

.logs-card :deep(.el-card__body) {
  padding: 18px;
}

.database-overview {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  margin-bottom: 16px;
  border-radius: 24px;
  border: 1px solid rgba(203, 161, 118, 0.18);
  background:
    linear-gradient(135deg, rgba(255, 247, 232, 0.96), rgba(250, 236, 214, 0.88)),
    var(--neko-surface-panel-strong-bg);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.5);
}

.database-overview__main {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.database-overview__label {
  font-size: 13px;
  color: #8a735d;
}

.database-overview__value {
  font-size: 28px;
  line-height: 1.1;
  color: #6d4425;
}

.database-overview__hint {
  margin: 0;
  font-size: 13px;
  color: #8a735d;
}

.filters-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
  padding: 14px;
  border-radius: 20px;
  border: 1px solid rgba(195, 153, 107, 0.14);
  background: var(--neko-surface-panel-strong-bg);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.42);
}

.filters-tip {
  margin: 0 0 16px;
  color: #8a735d;
  font-size: 13px;
}

.filter-input {
  width: min(280px, 100%);
}

.filter-select {
  width: 180px;
}

.logs-card :deep(.el-table__inner-wrapper) {
  background: linear-gradient(180deg, rgba(252, 246, 236, 0.38) 0%, rgba(255, 250, 244, 0.2) 100%);
}

.logs-card :deep(.el-table td.el-table__cell) {
  color: #5d4a39;
}

.logs-card :deep(.el-table__body-wrapper) {
  border-radius: 0 0 22px 22px;
}

.pagination-wrapper {
  justify-content: flex-end;
  padding-top: 16px;
  border-top: 1px solid var(--neko-surface-divider);
}

.dialog-header {
  justify-content: space-between;
  gap: 16px;
}

.dialog-header-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.dialog-header-copy span {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.detail-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.diff-shell {
  min-height: 520px;
}

.site-logs-page :deep(.el-dialog) {
  border: 1px solid var(--neko-surface-border);
  background: var(--neko-surface-bg);
  box-shadow: var(--neko-surface-shadow);
}

.site-logs-page :deep(.el-dialog__header) {
  border-bottom: 1px solid var(--neko-surface-divider);
  background: var(--neko-surface-header-bg);
}

.site-logs-page :deep(.el-dialog__body) {
  background: transparent;
}

:global(html.dark) .filters-bar {
  border-color: rgba(229, 201, 164, 0.1);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

:global(html.dark) .database-overview {
  border-color: rgba(229, 201, 164, 0.12);
  background: linear-gradient(135deg, rgba(70, 52, 36, 0.72), rgba(49, 39, 31, 0.82));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

:global(html.dark) .database-overview__label,
:global(html.dark) .database-overview__hint {
  color: #cbb59a;
}

:global(html.dark) .database-overview__value {
  color: #f6ddbc;
}

:global(html.dark) .logs-card :deep(.el-table__inner-wrapper) {
  background: linear-gradient(180deg, rgba(58, 48, 39, 0.28) 0%, rgba(31, 33, 39, 0.22) 100%);
}

:global(html.dark) .logs-card :deep(.el-table td.el-table__cell) {
  color: #e7d5c0;
}
</style>