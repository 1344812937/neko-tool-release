<template>
  <div class="browser-page">
    <div class="header">
      <div class="header-left">
        <NekoPageHeader
          :title="projectTitle"
          description="目录浏览、详情预览和文件内容会按列展开。"
          tone="blue"
          back-label="返回项目管理"
          @back="router.push('/projects')"
        />
      </div>
      <div class="header-right">
        <el-button :icon="Refresh" :loading="loading" @click="handleRefresh">刷新</el-button>
      </div>
    </div>

    <NekoOperationProgress
      v-if="operationProgress.visible"
      :title="operationProgress.title"
      :steps="operationProgress.steps"
      :active-step="operationProgress.activeStep"
      :status="operationProgress.status"
      :percent="operationProgress.percent"
      :processed="operationProgress.processed"
      :total="operationProgress.total"
      :current-path="operationProgress.currentPath"
      :message="operationProgress.message"
    />

    <div class="browser-layout">
      <el-card class="browser-columns-card neko-surface">
        <template #header>
          <div class="card-header">
            <span>项目目录 <span class="card-paw">喵</span></span>
          </div>
        </template>
        <div v-loading="loading" element-loading-text="猫咪正在翻找项目目录..." class="browser-columns-body">
          <button
            type="button"
            class="selected-path-bar"
            :class="{ 'is-empty': !selectedRelativePath }"
            :disabled="!selectedRelativePath"
            :title="selectedRelativePath || '选中目录或文件后，可在这里查看并复制相对路径'"
            @click="copySelectedPath"
          >
            <span class="selected-path-label">当前路径</span>
            <span class="selected-path-value">{{ selectedPathDisplay }}</span>
            <el-icon class="selected-path-icon"><CopyDocument /></el-icon>
          </button>
          <el-alert
            v-if="browserMessage"
            :closable="false"
            show-icon
            :type="manifest?.projectDeleted ? 'warning' : 'info'"
            :title="browserMessage"
          />
          <FinderColumns
            :nodes="treeNodes"
            :selected-path="selectedPath"
            empty-text="项目中暂无可浏览内容"
            @select="handleSelect"
            @expand="handleExpand"
          />
        </div>
      </el-card>

      <el-card class="browser-detail-card neko-surface">
        <template #header>
          <div class="card-header">
            <span>详情预览 <span class="card-paw">爪</span></span>
          </div>
        </template>
        <div v-if="loading" class="detail-loading">
          <div class="neko-loading-copy">猫咪正在整理目录摘要和当前预览内容。</div>
          <el-skeleton :rows="8" animated />
        </div>
        <div v-else-if="selectedNode" class="detail-content">
            <div class="detail-actions">
              <el-button
                v-if="canDeleteSelected"
                type="danger"
                plain
                :icon="Delete"
                :disabled="loading || fileLoading"
                @click="handleDeleteSelected"
              >
                删除{{ selectedNode.entryType === 'directory' ? '目录' : '文件' }}
              </el-button>
            </div>
          <el-descriptions border :column="1">
            <el-descriptions-item label="名称">{{ selectedNode.name }}</el-descriptions-item>
            <el-descriptions-item label="类型">{{ selectedNode.entryType === 'directory' ? '目录' : '文件' }}</el-descriptions-item>
            <el-descriptions-item label="状态">{{ selectedNode.deleted ? '已删除' : '正常' }}</el-descriptions-item>
            <el-descriptions-item label="相对路径">{{ selectedNode.path }}</el-descriptions-item>
            <el-descriptions-item label="大小">{{ selectedNode.size || 0 }} bytes</el-descriptions-item>
            <el-descriptions-item label="摘要">{{ selectedNode.hash || '-' }}</el-descriptions-item>
          </el-descriptions>

          <div v-if="selectedNode.entryType === 'directory'" class="directory-summary">
            <el-alert
              :closable="false"
              :type="selectedNode.deleted ? 'warning' : 'success'"
              show-icon
              :title="selectedNode.deleted ? '该目录已在磁盘上删除，当前展示的是缓存记录。' : '已选中目录，可继续在右侧分栏中深入浏览。'"
            />
          </div>

          <div v-else class="file-preview">
            <div class="preview-header">
              <span>文件预览</span>
              <el-tag v-if="filePreview?.deleted || selectedNode.deleted" type="danger">已删除</el-tag>
              <el-tag v-if="filePreview?.text" type="success">文本</el-tag>
              <el-tag v-else-if="filePreview?.exists" type="warning">二进制</el-tag>
              <el-tag v-if="filePreview?.text" type="info">{{ previewLanguageLabel }}</el-tag>
              <el-tag
                class="file-log-tag"
                :type="fileLogResult.count > 0 ? 'warning' : 'info'"
                @click="openFileLogDialog"
              >
                修改 {{ fileLogResult.count }} 次
              </el-tag>
              <div class="preview-header-actions">
                <el-button v-if="canFullscreenPreview" size="small" :icon="FullScreen" @click="openFullscreenPreview">全屏预览</el-button>
              </div>
            </div>
            <div class="preview-panel">
              <div v-if="fileLoading" class="preview-state-shell file-loading-state">
                <div class="neko-loading-copy">猫咪正在把文件内容从柜子里翻出来。</div>
                <el-skeleton :rows="8" animated />
              </div>
              <div v-else-if="filePreview && (filePreview.deleted || filePreview.exists === false)" class="preview-state-shell">
                <el-alert :closable="false" type="warning" show-icon title="该文件已在磁盘上删除，当前不再提供内容预览。" />
              </div>
              <div v-else-if="filePreview?.text" class="preview-editor-shell">
                <MonacoCodePreview :content="filePreview.content || ''" :language="previewLanguage" />
              </div>
              <div v-else class="preview-state-shell">
                <NekoEmptyState title="猫咪闻到的是二进制味道" description="当前文件不是文本文件，暂时不能直接展开预览内容。" compact />
              </div>
            </div>
          </div>
        </div>
        <NekoEmptyState v-else title="先挑一个目录或文件" description="猫咪已经把列视图铺开了，从左边选中目标后，这里就会显示详情。" compact />
      </el-card>
    </div>

    <el-dialog v-model="previewFullscreen" class="preview-fullscreen-dialog" fullscreen destroy-on-close>
      <template #header>
        <div class="preview-fullscreen-header">
          <div class="preview-fullscreen-copy">
            <strong>文件全屏预览</strong>
            <span>{{ selectedRelativePath || '未选择文件' }}</span>
          </div>
          <div class="preview-fullscreen-tags">
            <el-tag v-if="filePreview?.text" type="info">{{ previewLanguageLabel }}</el-tag>
          </div>
        </div>
      </template>
      <div class="preview-fullscreen-body">
        <MonacoCodePreview v-if="filePreview?.text" :content="filePreview.content || ''" :language="previewLanguage" />
      </div>
    </el-dialog>

    <el-dialog
      v-model="fileLogDialogVisible"
      width="1120"
      :close-on-click-modal="false"
      class="file-log-dialog"
    >
      <template #header>
        <div class="file-log-dialog-header">
          <div class="file-log-dialog-copy">
            <strong>文件修改记录</strong>
            <span>{{ selectedRelativePath || '未选择文件' }}</span>
          </div>
          <el-tag type="warning">共 {{ fileLogResult.count }} 条</el-tag>
        </div>
      </template>
      <div v-loading="fileLogLoading" class="file-log-dialog-body">
        <NekoEmptyState
          v-if="!fileLogLoading && fileLogResult.count === 0"
          title="当前文件还没有修改记录"
          description="同步覆盖会直接记录日志；浏览页在读取文件和刷新项目时也会继续补充本地变更日志。"
          compact
        />
        <el-table v-else :data="fileLogResult.items" row-key="id" max-height="520">
          <el-table-column prop="operatedAt" label="修改时间" width="180" />
          <el-table-column label="变更类型" width="120">
            <template #default="{ row }">
              <el-tag size="small" :type="fileLogTypeMap[row.changeType] || 'info'">{{ fileLogLabelMap[row.changeType] || row.changeType }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="executorNodeName" label="修改工作站" width="140" show-overflow-tooltip />
          <el-table-column prop="executorNodeAddress" label="操作站地址" width="160" show-overflow-tooltip />
          <el-table-column prop="operatorIP" label="修改人 IP" width="120" show-overflow-tooltip />
          <el-table-column label="修改摘要" min-width="320" show-overflow-tooltip>
            <template #default="{ row }">
              {{ (row.beforeHash || '-') + ' -> ' + (row.afterHash || '-') }}
            </template>
          </el-table-column>
          <el-table-column label="详情" width="110" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link @click="openFileLogDetail(row)">查看详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <el-dialog
      v-model="fileLogDetailVisible"
      width="1240"
      :close-on-click-modal="false"
      class="file-log-detail-dialog"
    >
      <template #header>
        <div class="file-log-dialog-header">
          <div class="file-log-dialog-copy">
            <strong>修改详情</strong>
            <span>{{ fileLogDetail?.relativePath || selectedRelativePath || '未选择文件' }}</span>
          </div>
          <el-tag v-if="fileLogDetail" :type="fileLogTypeMap[fileLogDetail.changeType] || 'info'">{{ fileLogLabelMap[fileLogDetail.changeType] || fileLogDetail.changeType }}</el-tag>
        </div>
      </template>
      <div v-if="fileLogDetailLoading" class="detail-loading">
        <div class="neko-loading-copy">猫咪正在翻查本次修改的详细内容。</div>
        <el-skeleton :rows="8" animated />
      </div>
      <div v-else-if="fileLogDetail" class="file-log-detail-body">
        <el-descriptions border :column="2">
          <el-descriptions-item label="修改时间">{{ formatLogDisplayTime(fileLogDetail.operatedAt, fileLogDetail.modifyTime) }}</el-descriptions-item>
          <el-descriptions-item label="修改工作站">{{ fileLogDetail.executorNodeName || '-' }}</el-descriptions-item>
          <el-descriptions-item label="操作站地址">{{ fileLogDetail.executorNodeAddress || '-' }}</el-descriptions-item>
          <el-descriptions-item label="修改人 IP">{{ fileLogDetail.operatorIP || '-' }}</el-descriptions-item>
          <el-descriptions-item label="修改范围">{{ fileLogDetail.scopeType || '-' }}</el-descriptions-item>
          <el-descriptions-item label="修改前摘要">{{ fileLogDetail.beforeHash || '-' }}</el-descriptions-item>
          <el-descriptions-item label="修改后摘要">{{ fileLogDetail.afterHash || '-' }}</el-descriptions-item>
        </el-descriptions>

        <div v-if="canRenderFileLogDiff(fileLogDetail)" class="file-log-diff-shell">
          <MonacoDiff
            :left-content="getFileLogText(fileLogDetail.beforeEncoding, fileLogDetail.beforeContent)"
            :right-content="getFileLogText(fileLogDetail.afterEncoding, fileLogDetail.afterContent)"
            :language="previewLanguage"
          />
        </div>
        <div v-else class="file-log-binary-shell">
          <el-alert :closable="false" type="info" show-icon :title="getFileLogDetailNotice(fileLogDetail)" />
          <el-descriptions border :column="1">
            <el-descriptions-item label="修改前编码">{{ fileLogDetail.beforeEncoding || '-' }}</el-descriptions-item>
            <el-descriptions-item label="修改前存储方式">{{ formatFileLogStorageKind(fileLogDetail.beforeStorageKind) }}</el-descriptions-item>
            <el-descriptions-item label="修改前大小">{{ formatFileLogSize(fileLogDetail.beforeContentSize) }}</el-descriptions-item>
            <el-descriptions-item label="修改后编码">{{ fileLogDetail.afterEncoding || '-' }}</el-descriptions-item>
            <el-descriptions-item label="修改后存储方式">{{ formatFileLogStorageKind(fileLogDetail.afterStorageKind) }}</el-descriptions-item>
            <el-descriptions-item label="修改后大小">{{ formatFileLogSize(fileLogDetail.afterContentSize) }}</el-descriptions-item>
            <el-descriptions-item label="修改前是否存在">{{ fileLogDetail.beforeExists ? '是' : '否' }}</el-descriptions-item>
            <el-descriptions-item label="修改后是否存在">{{ fileLogDetail.afterEncoding === 'none' ? '否' : '是' }}</el-descriptions-item>
          </el-descriptions>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CopyDocument, Delete, FullScreen, Refresh } from '@element-plus/icons-vue'
import FinderColumns from '@/components/FinderColumns.vue'
import NekoOperationProgress from '@/components/NekoOperationProgress.vue'
import NekoPageHeader from '@/components/NekoPageHeader.vue'
import NekoEmptyState from '@/components/NekoEmptyState.vue'
import { buildManifestTree, getNodeByPath, mergeManifestEntries, shouldLoadChildren, updateManifestEntryState } from '@/utils/finder'

const MonacoCodePreview = defineAsyncComponent(() => import('@/components/MonacoCodePreview.vue'))
const MonacoDiff = defineAsyncComponent(() => import('@/components/MonacoDiff.vue'))

const route = useRoute()
const router = useRouter()

document.title = '项目浏览'

const MAX_AUTO_LOAD_STEPS = 8
const PATH_DISPLAY_LIMIT = 58

const loading = ref(false)
const fileLoading = ref(false)
const manifestEntries = ref([])
const manifest = ref(null)
const selectedPath = ref('')
const filePreview = ref(null)
const previewFullscreen = ref(false)
const fileLogLoading = ref(false)
const fileLogDialogVisible = ref(false)
const fileLogDetailVisible = ref(false)
const fileLogDetailLoading = ref(false)
const fileLogDetail = ref(null)
const fileLogResult = ref({ count: 0, items: [] })
const fileLogPath = ref('')
const operationProgress = ref(createOperationProgress())
const browseDepth = 3

const projectId = computed(() => Number(route.params.id || 0))
const treeNodes = computed(() => buildManifestTree(manifestEntries.value))
const selectedNode = computed(() => getNodeByPath(treeNodes.value, selectedPath.value))
const projectTitle = computed(() => manifest.value?.project?.name || `项目 #${projectId.value}`)
const browserMessage = computed(() => manifest.value?.message || '')
const selectedRelativePath = computed(() => selectedNode.value?.path || '')
const selectedPathDisplay = computed(() => formatPathForDisplay(selectedRelativePath.value, PATH_DISPLAY_LIMIT) || '选中目录或文件后，可点击复制相对路径')
const previewLanguage = computed(() => detectPreviewLanguage(selectedNode.value?.path || ''))
const previewLanguageLabel = computed(() => previewLanguageLabelMap[previewLanguage.value] || '纯文本')
const canFullscreenPreview = computed(() => Boolean(filePreview.value?.text && !fileLoading.value))
const canDeleteSelected = computed(() => Boolean(selectedNode.value && selectedNode.value.path && !selectedNode.value.deleted))

// 创建浏览页统一的操作进度模型。
function createOperationProgress() {
  return {
    visible: false,
    title: '',
    steps: [],
    activeStep: 0,
    status: 'running',
    percent: 0,
    processed: 0,
    total: 0,
    currentPath: '',
    message: '',
  }
}

// 启动一个新的浏览操作进度。
function beginOperationProgress(title, stepTitles, patch = {}) {
  operationProgress.value = {
    ...createOperationProgress(),
    visible: true,
    title,
    steps: stepTitles.map((step, index) => ({ key: `${title}-${index}`, title: step })),
    ...patch,
  }
}

// 增量更新浏览页当前操作进度。
function updateOperationProgress(patch) {
  operationProgress.value = {
    ...operationProgress.value,
    visible: true,
    ...patch,
  }
}

// 标记浏览页当前操作已成功完成。
function finishOperationProgress(message, patch = {}) {
  updateOperationProgress({ status: 'success', percent: 100, message, ...patch })
}

// 标记浏览页当前操作失败。
function failOperationProgress(message, patch = {}) {
  updateOperationProgress({ status: 'error', message, ...patch })
}

const fileLogLabelMap = {
  file_changed: '同步覆盖',
  left_only: '左侧新增',
  right_only: '右侧新增',
  local_snapshot: '本地基线',
  local_modified: '本地修改',
  local_deleted: '本地删除',
}

const fileLogTypeMap = {
  file_changed: 'danger',
  left_only: 'primary',
  right_only: 'success',
  local_snapshot: 'info',
  local_modified: 'warning',
  local_deleted: 'danger',
}

const previewLanguageLabelMap = {
  plaintext: '纯文本',
  javascript: 'JavaScript',
  typescript: 'TypeScript',
  java: 'Java',
  go: 'Go',
  json: 'JSON',
  yaml: 'YAML',
  shell: 'Shell',
}

// 路径过长时做中间折叠，保证头部展示区域可读。
function formatPathForDisplay(path, maxLength = 58) {
  const normalized = String(path || '').trim()
  if (!normalized || normalized.length <= maxLength) {
    return normalized
  }
  const segments = normalized.split('/')
  const tail = segments.pop() || normalized
  const suffix = `/${tail}`
  const remaining = maxLength - suffix.length - 4
  if (remaining <= 0) {
    return `...${suffix}`
  }
  const prefix = segments.join('/')
  if (!prefix) {
    return `...${suffix}`
  }
  return `${prefix.slice(0, remaining).replace(/\/+$/, '')}/...${suffix}`
}

// 根据文件后缀推断 Monaco 预览语言。
function detectPreviewLanguage(path) {
  const normalized = String(path || '').toLowerCase()
  const fileName = normalized.split('/').pop() || normalized
  if (normalized.endsWith('.yaml') || normalized.endsWith('.yml')) {
    return 'yaml'
  }
  if (normalized.endsWith('.json') || normalized.endsWith('.jsonc')) {
    return 'json'
  }
  if (
    normalized.endsWith('.sh')
    || normalized.endsWith('.bash')
    || normalized.endsWith('.zsh')
    || fileName === '.bashrc'
    || fileName === '.zshrc'
    || fileName === '.profile'
  ) {
    return 'shell'
  }
  if (normalized.endsWith('.java')) {
    return 'java'
  }
  if (normalized.endsWith('.go')) {
    return 'go'
  }
  if (normalized.endsWith('.ts') || normalized.endsWith('.tsx')) {
    return 'typescript'
  }
  if (normalized.endsWith('.js') || normalized.endsWith('.jsx') || normalized.endsWith('.mjs') || normalized.endsWith('.cjs')) {
    return 'javascript'
  }
  return 'plaintext'
}

// 紧凑目录链展开后，统一修正当前选中路径。
function normalizeSelectedNodePath() {
  if (!selectedPath.value) {
    return null
  }
  const resolvedNode = getNodeByPath(treeNodes.value, selectedPath.value)
  if (resolvedNode && selectedPath.value !== resolvedNode.path) {
    selectedPath.value = resolvedNode.path
  }
  return resolvedNode
}

// Finder 的紧凑目录节点在满足条件时允许自动继续加载下一层。
function shouldAutoLoadCompactedNode(node) {
  return Boolean(node && node.entryType === 'directory' && node.compacted && node.hasChildren && !node.childrenLoaded && !node.deleted)
}

// 统一封装复制逻辑，优先走 clipboard API，失败时回退 textarea。
async function copyText(text) {
  if (navigator?.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'absolute'
  textarea.style.left = '-9999px'
  document.body.appendChild(textarea)
  textarea.select()
  document.execCommand('copy')
  document.body.removeChild(textarea)
}

// 复制当前选中的相对路径。
async function copySelectedPath() {
  if (!selectedRelativePath.value) {
    return
  }
  try {
    await copyText(selectedRelativePath.value)
    ElMessage.success('相对路径已复制')
  } catch {
    ElMessage.error('复制路径失败')
  }
}

// 文本文件时允许切换到全屏预览模式。
function openFullscreenPreview() {
  if (!canFullscreenPreview.value) {
    return
  }
  previewFullscreen.value = true
}

// 请求目录浏览接口，refresh 控制是否触发后端重新扫描。
async function requestBrowser(basePath = '', refresh = false) {
  const res = await fetch(refresh ? '/api/compare/browser/refresh' : '/api/compare/browser', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ nodeId: 0, projectId: projectId.value, basePath, depth: browseDepth }),
  })
  const data = await res.json()
  if (!data.success) {
    throw new Error(data.message || '加载项目目录失败')
  }
  return data.data
}

// 请求删除当前项目中的文件或目录。
async function requestBrowserDelete(path) {
  const res = await fetch('/api/compare/browser/delete', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ projectId: projectId.value, path }),
  })
  const data = await res.json()
  if (!data.success) {
    throw new Error(data.message || '删除失败')
  }
}

// 当目标路径失效时，自动回退到当前树中的首个可选节点。
async function selectFallbackNode() {
  const nextNode = getNodeByPath(treeNodes.value, selectedPath.value) || treeNodes.value[0]
  if (!nextNode) {
    selectedPath.value = ''
    filePreview.value = null
    resetFileLogState()
    return
  }
  await handleSelect(nextNode)
}

// 根据 query path 逐层展开目录并尝试定位目标节点。
async function focusTargetPath(targetPath) {
  const normalizedPath = String(targetPath || '').trim().replace(/^\/+/, '')
  if (!normalizedPath) {
    await selectFallbackNode()
    return
  }
  const segments = normalizedPath.split('/').filter(Boolean)
  let currentPath = ''
  for (let index = 0; index < segments.length - 1; index += 1) {
    currentPath = currentPath ? `${currentPath}/${segments[index]}` : segments[index]
    await ensureDirectoryLoaded(currentPath)
  }
  const targetNode = getNodeByPath(treeNodes.value, normalizedPath)
  if (targetNode) {
    await handleSelect(targetNode)
    return
  }
  await selectFallbackNode()
}

// 加载项目目录，并同步更新页面的进度提示。
async function loadBrowser(refresh = false) {
  if (!projectId.value) {
    ElMessage.error('无效的项目 ID')
    return false
  }
  loading.value = true
  filePreview.value = null
  resetFileLogState()
  beginOperationProgress(refresh ? '刷新项目目录' : '读取项目目录', ['请求目录数据', '合并缓存结果', '恢复当前选中'], {
    activeStep: 0,
    percent: 12,
    message: refresh ? '正在重新扫描项目目录并读取缓存结果。' : '正在加载项目目录数据。',
  })
  try {
    updateOperationProgress({ activeStep: 1, percent: 45, message: '目录数据已返回，正在合并到当前视图。' })
    const data = await requestBrowser('', refresh)
    manifest.value = data
    manifestEntries.value = mergeManifestEntries([], data, { replace: true })
    updateOperationProgress({ activeStep: 2, percent: 78, message: '目录结构已更新，正在恢复当前选中路径。', currentPath: String(route.query.path || '') })
    await focusTargetPath(route.query.path)
    finishOperationProgress(refresh ? '项目目录已刷新，并已完成文件内容与修改日志的比对。' : '项目目录加载完成。', { currentPath: '' })
    return true
  } catch (error) {
    failOperationProgress(error.message || '加载项目目录失败')
    ElMessage.error(error.message || '加载项目目录失败')
    return false
  } finally {
    loading.value = false
  }
}

// 按需加载目录下一层内容，并兼容紧凑目录自动续展开。
async function ensureDirectoryLoaded(path, options = {}) {
  let nextPath = path
  let resolvedNode = getNodeByPath(treeNodes.value, nextPath)
  for (let step = 0; step < MAX_AUTO_LOAD_STEPS; step += 1) {
    const actualPath = resolvedNode?.path || nextPath
    if (!shouldLoadChildren(resolvedNode) || resolvedNode.loading) {
      break
    }
    manifestEntries.value = updateManifestEntryState(manifestEntries.value, actualPath, { loading: true })
    try {
      const data = await requestBrowser(actualPath, false)
      manifest.value = { ...manifest.value, ...data }
      manifestEntries.value = mergeManifestEntries(manifestEntries.value, data)
      resolvedNode = normalizeSelectedNodePath() || getNodeByPath(treeNodes.value, actualPath) || getNodeByPath(treeNodes.value, nextPath)
      if (!shouldAutoLoadCompactedNode(resolvedNode)) {
        break
      }
      nextPath = resolvedNode.path
    } catch (error) {
      manifestEntries.value = updateManifestEntryState(manifestEntries.value, actualPath, { loading: false })
      ElMessage.error(error.message || '加载目录失败')
      return
    }
  }
  if (options.focusFirstChild) {
    const nextChild = resolvedNode?.children?.[0]
    if (nextChild) {
      await handleSelect(nextChild)
    }
  }
}

// 目录与文件选中后进入不同的详情分支。
async function handleSelect(node) {
  if (!node) {
    return
  }
  previewFullscreen.value = false
  selectedPath.value = node.path
  if (node.entryType === 'file') {
    await loadFile(node.path)
  } else {
    filePreview.value = null
    resetFileLogState()
    if (!node.deleted) {
      await ensureDirectoryLoaded(node.path)
    }
  }
}

// Finder 触发展开时按需补充该目录的子项。
async function handleExpand(payload) {
  if (!payload?.item) {
    return
  }
  await ensureDirectoryLoaded(payload.item.path, { focusFirstChild: payload.focusFirstChild })
}

// 手动刷新项目目录，并在成功后提示已完成日志比对。
async function handleRefresh() {
  previewFullscreen.value = false
  selectedPath.value = ''
  const success = await loadBrowser(true)
  if (success) {
    ElMessage.success('项目目录已刷新，并已完成文件内容与修改日志的比对')
  }
}

// 删除当前选中的文件或目录，并在完成后强制刷新目录缓存。
async function handleDeleteSelected() {
  if (!selectedNode.value?.path || !canDeleteSelected.value) {
    return
  }
  const targetNode = selectedNode.value
  const isDirectory = targetNode.entryType === 'directory'
  const actionLabel = isDirectory ? '目录' : '文件'
  const warningText = isDirectory ? '删除后会移除该目录及其全部内容，且不会放入回收站。' : '删除后不会放入回收站。'
  try {
    await ElMessageBox.confirm(
      `确定要删除${actionLabel}“${targetNode.path}”吗？${warningText}`,
      `删除${actionLabel}`,
      {
        type: 'warning',
        confirmButtonText: '确认删除',
        cancelButtonText: '取消',
        confirmButtonClass: 'el-button--danger',
      },
    )
  } catch {
    return
  }

  const deletedPath = targetNode.path
  beginOperationProgress(`删除${actionLabel}`, ['确认删除目标', '执行磁盘删除', '刷新目录缓存', '清理预览状态'], {
    activeStep: 0,
    percent: 12,
    currentPath: deletedPath,
    message: `已确认删除目标，准备删除${actionLabel}。`,
  })
  try {
    updateOperationProgress({ activeStep: 1, percent: 38, currentPath: deletedPath, message: '正在执行磁盘删除。' })
    await requestBrowserDelete(deletedPath)
    updateOperationProgress({ activeStep: 2, percent: 68, currentPath: deletedPath, message: '删除成功，正在刷新目录缓存。' })
    if (isPathAffectedByDeletion(selectedPath.value, deletedPath)) {
      selectedPath.value = ''
      filePreview.value = null
      previewFullscreen.value = false
      resetFileLogState()
    }
    const success = await loadBrowser(true)
    if (!success) {
      failOperationProgress(`已删除${actionLabel}，但目录刷新失败。`, { currentPath: deletedPath })
      return
    }
    updateOperationProgress({ activeStep: 3, percent: 96, currentPath: '', message: '目录缓存已刷新，正在清理预览状态。' })
    finishOperationProgress(`${actionLabel}已删除，目录缓存已刷新。`, { currentPath: '' })
    ElMessage.success(`${actionLabel}已删除`)
  } catch (error) {
    failOperationProgress(error.message || `删除${actionLabel}失败`, { currentPath: deletedPath })
    ElMessage.error(error.message || `删除${actionLabel}失败`)
  }
}

// 读取单文件内容，再串行拉取对应的修改日志。
async function loadFile(path) {
  fileLoading.value = true
  beginOperationProgress('读取文件内容', ['读取文件内容', '更新文件预览', '拉取修改日志'], {
    activeStep: 0,
    percent: 18,
    currentPath: path,
    message: '正在读取文件内容。',
  })
  try {
    const res = await fetch('/api/compare/browser/file', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nodeId: 0, projectId: projectId.value, path }),
    })
    const data = await res.json()
    if (data.success) {
      filePreview.value = data.data
      updateOperationProgress({ activeStep: 1, percent: 58, currentPath: path, message: '文件预览已更新，正在拉取修改日志。' })
      const logsLoaded = await loadFileLogs(path)
      if (data.data?.deleted) {
        manifestEntries.value = updateManifestEntryState(manifestEntries.value, path, { deleted: true, loading: false })
      }
      if (logsLoaded) {
        finishOperationProgress('文件内容与修改日志已加载完成。', { currentPath: path })
      } else {
        failOperationProgress('文件内容已加载，但修改日志拉取失败。', { currentPath: path, percent: 78 })
      }
    } else {
      resetFileLogState()
      failOperationProgress(data.message || '读取文件失败', { currentPath: path })
      ElMessage.error(data.message || '读取文件失败')
    }
  } catch {
    resetFileLogState()
    failOperationProgress('读取文件失败', { currentPath: path })
    ElMessage.error('读取文件失败')
  } finally {
    fileLoading.value = false
  }
}

// 拉取当前文件的修改日志列表，供弹窗和计数展示。
async function loadFileLogs(path) {
  fileLogLoading.value = true
  try {
    const res = await fetch('/api/compare/browser/file/logs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId: projectId.value, path }),
    })
    const data = await res.json()
    if (!data.success) {
      throw new Error(data.message || '加载文件修改记录失败')
    }
    if (selectedRelativePath.value && selectedRelativePath.value !== path) {
      return
    }
    fileLogResult.value = data.data || { count: 0, items: [] }
    fileLogPath.value = path
    return true
  } catch (error) {
    resetFileLogState()
    ElMessage.error(error.message || '加载文件修改记录失败')
    return false
  } finally {
    fileLogLoading.value = false
  }
}

// 打开文件日志列表弹窗。
function openFileLogDialog() {
  if (!selectedRelativePath.value) {
    return
  }
  fileLogDialogVisible.value = true
}

// 拉取单条文件修改日志的详细内容。
async function openFileLogDetail(row) {
  fileLogDetailLoading.value = true
  fileLogDetailVisible.value = true
  fileLogDetail.value = null
  try {
    const res = await fetch('/api/compare/browser/file/log-detail', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ logId: row.id }),
    })
    const data = await res.json()
    if (!data.success) {
      throw new Error(data.message || '加载修改详情失败')
    }
    fileLogDetail.value = data.data
  } catch (error) {
    fileLogDetailVisible.value = false
    ElMessage.error(error.message || '加载修改详情失败')
  } finally {
    fileLogDetailLoading.value = false
  }
}

// 切换文件或目录时清空日志相关的局部状态。
function resetFileLogState() {
  fileLogResult.value = { count: 0, items: [] }
  fileLogPath.value = ''
  fileLogDialogVisible.value = false
  fileLogDetailVisible.value = false
  fileLogDetail.value = null
}

function isPathAffectedByDeletion(currentPath, deletedPath) {
  const normalizedCurrent = String(currentPath || '').trim().replace(/^\/+/, '')
  const normalizedDeleted = String(deletedPath || '').trim().replace(/^\/+/, '')
  if (!normalizedCurrent || !normalizedDeleted) {
    return false
  }
  return normalizedCurrent === normalizedDeleted || normalizedCurrent.startsWith(`${normalizedDeleted}/`)
}

// 统一格式化日志时间展示，兼容 operatedAt 和 modifyTime。
function formatLogDisplayTime(operatedAt, modifyTime) {
  return operatedAt || modifyTime || '-'
}

// 仅在文本编码时返回可供 diff 使用的内容。
function getFileLogText(encoding, content) {
  return encoding === 'text' ? content || '' : ''
}

// 文件日志详情仅在前后内容都可文本化时展示 diff。
function canRenderFileLogDiff(detail) {
  return canPreviewFileLogSide(detail?.beforeEncoding, detail?.beforeStorageKind, detail?.beforeOmittedReason)
    && canPreviewFileLogSide(detail?.afterEncoding, detail?.afterStorageKind, detail?.afterOmittedReason)
}

function canPreviewFileLogSide(encoding, storageKind, omittedReason) {
  if (omittedReason) {
    return false
  }
  const normalizedEncoding = encoding || 'none'
  const normalizedStorageKind = storageKind || 'legacy_full'
  return ['text', 'none', ''].includes(normalizedEncoding)
    && ['full_text', 'compressed_full_text', 'reverse_patch', 'compressed_reverse_patch', 'legacy_full', 'none', ''].includes(normalizedStorageKind)
}

function getFileLogDetailNotice(detail) {
  const reasons = [detail?.beforeOmittedReason, detail?.afterOmittedReason].filter(Boolean)
  if (reasons.includes('size_limit')) {
    return '当前文件超过 15MB，日志仅保留摘要信息。'
  }
  if (reasons.includes('binary')) {
    return '当前文件包含二进制内容，日志仅保留摘要信息。'
  }
  return '当前修改记录无法直接渲染 diff，下面仅展示元数据摘要。'
}

function formatFileLogStorageKind(storageKind) {
  const mapping = {
    full_text: '完整文本',
    compressed_full_text: '压缩完整文本',
    reverse_patch: '逆向差异补丁',
    compressed_reverse_patch: '压缩逆向差异补丁',
    hash_only: '仅摘要',
    none: '无内容',
    legacy_full: '历史完整内容',
  }
  return mapping[storageKind] || storageKind || '-'
}

function formatFileLogSize(size) {
  const numericSize = Number(size || 0)
  if (!numericSize) {
    return '0 B'
  }
  if (numericSize < 1024) {
    return `${numericSize} B`
  }
  if (numericSize < 1024 * 1024) {
    return `${(numericSize / 1024).toFixed(1)} KB`
  }
  return `${(numericSize / 1024 / 1024).toFixed(2)} MB`
}

onMounted(() => loadBrowser(false))

watch(() => route.query.path, async (nextPath, previousPath) => {
  if (!manifest.value || nextPath === previousPath) {
    return
  }
  await focusTargetPath(nextPath)
})

watch(treeNodes, () => {
  normalizeSelectedNodePath()
})
</script>

<style scoped>
.browser-page {
  width: 100vw;
  height: 100vh;
  padding: 24px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
  background:
    radial-gradient(circle at top left, rgba(64, 158, 255, 0.08), transparent 32%),
    radial-gradient(circle at bottom right, rgba(239, 188, 125, 0.1), transparent 28%),
    var(--el-bg-color-page);
}

.header,
.header-left,
.header-right {
  display: flex;
  align-items: center;
}

.header {
  justify-content: space-between;
  gap: 16px;
}

.header-left,
.header-right {
  gap: 14px;
}

.header-left {
  min-width: 0;
  flex: 1;
}

.browser-layout {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: grid;
  grid-template-columns: minmax(0, 1.3fr) minmax(320px, 0.8fr);
  gap: 16px;
}

.browser-columns-card,
.browser-detail-card {
  min-height: 720px;
}

.browser-columns-card :deep(.el-card__body),
.browser-detail-card :deep(.el-card__body) {
  height: calc(100% - 12px);
}

.browser-columns-body {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-actions {
	display: flex;
	justify-content: flex-end;
	margin-bottom: 12px;
}

.selected-path-bar {
  width: 100%;
  min-height: 44px;
  padding: 0 14px;
  border: 1px solid color-mix(in srgb, var(--el-color-danger) 62%, white);
  border-radius: 12px;
  background: color-mix(in srgb, var(--el-fill-color-light) 88%, white);
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s ease, transform 0.2s ease, box-shadow 0.2s ease;
}

.selected-path-bar:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--el-color-primary) 52%, white);
  box-shadow: 0 8px 20px color-mix(in srgb, var(--el-color-primary) 14%, transparent);
}

.selected-path-bar:disabled {
  cursor: default;
  opacity: 0.72;
}

.selected-path-bar.is-empty {
  border-style: dashed;
}

.selected-path-label {
  flex: none;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}

.selected-path-value {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.selected-path-icon {
  flex: none;
  font-size: 16px;
  color: var(--el-color-primary);
}

.browser-columns-body :deep(.el-alert) {
	border-radius: 14px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.card-paw {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  margin-left: 6px;
  padding: 0 6px;
  border-radius: 999px;
  background: rgba(80, 154, 225, 0.1);
  color: #5b88bb;
  font-size: 11px;
  font-weight: 700;
}

:global(html.dark) .card-paw {
  background: rgba(80, 154, 225, 0.16);
  color: #a7c7ea;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.detail-loading,
.file-loading-state {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.neko-loading-copy {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.file-preview {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
}

.preview-header {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.file-log-tag {
  cursor: pointer;
}

.preview-header-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.preview-panel {
  height: 420px;
  min-height: 420px;
}

.preview-editor-shell,
.preview-state-shell {
  height: 100%;
}

.preview-state-shell {
  overflow: auto;
  padding-right: 4px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.preview-fullscreen-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-width: 0;
}

.preview-fullscreen-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.preview-fullscreen-copy span {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.preview-fullscreen-tags {
  display: flex;
  align-items: center;
  gap: 8px;
}

.preview-fullscreen-body {
  height: calc(100vh - 110px);
}

.preview-fullscreen-dialog :deep(.el-dialog__body) {
  padding-top: 8px;
  height: calc(100vh - 78px);
  box-sizing: border-box;
}

.file-log-dialog-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.file-log-dialog-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.file-log-dialog-copy span {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-log-dialog-body,
.file-log-detail-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.file-log-diff-shell {
  min-height: 540px;
}

.file-log-binary-shell {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

@media (max-width: 1280px) {
  .browser-layout {
    grid-template-columns: 1fr;
  }

  .file-log-dialog-header {
    flex-direction: column;
    align-items: flex-start;
  }
}

:global(html.dark) .browser-page {
  background:
    radial-gradient(circle at top left, rgba(51, 94, 134, 0.2), transparent 32%),
    radial-gradient(circle at bottom right, rgba(128, 87, 51, 0.14), transparent 28%),
    var(--el-bg-color-page);
}

</style>