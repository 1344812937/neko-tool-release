<template>
  <div class="compare-container">
    <div class="header">
      <div class="header-left">
        <NekoPageHeader
          title="比较工作台"
          description="项目基准、目录范围和同步动作都在这里集中展示。"
          tone="orange"
          @back="router.push('/')"
        />
      </div>
      <div class="header-right">
        <el-button :icon="Refresh" :loading="loading" @click="bootstrap">刷新基础数据</el-button>
      </div>
    </div>

    <el-card class="selection-card neko-surface">
      <div class="selection-grid">
        <div class="selection-column">
          <div class="selection-panel">
            <div class="section-title">左侧基准 <span class="section-paw">爪</span></div>
            <el-form label-position="top">
              <el-form-item label="节点">
                <el-select v-model="form.leftNodeId" @change="handleNodeChange('left')">
                  <el-option :value="0" label="本机节点" />
                  <el-option v-for="node in enabledNodes" :key="node.id" :label="node.name" :value="node.id" />
                </el-select>
              </el-form-item>
              <el-form-item label="项目">
                <el-select v-model="form.leftProjectId" filterable @change="handleProjectChange('left')">
                  <el-option v-for="project in leftProjects" :key="project.id" :label="formatProjectOptionLabel(project)" :value="project.id" />
                </el-select>
              </el-form-item>
            </el-form>
          </div>
        </div>

        <div class="selection-column">
          <div class="selection-panel">
            <div class="section-title">右侧目标 <span class="section-paw">喵</span></div>
            <el-form label-position="top">
              <el-form-item label="节点">
                <el-select v-model="form.rightNodeId" @change="handleNodeChange('right')">
                  <el-option :value="0" label="本机节点" />
                  <el-option v-for="node in enabledNodes" :key="node.id" :label="node.name" :value="node.id" />
                </el-select>
              </el-form-item>
              <el-form-item label="项目">
                <el-select v-model="form.rightProjectId" filterable @change="handleProjectChange('right')">
                  <el-option v-for="project in rightProjects" :key="project.id" :label="formatProjectOptionLabel(project)" :value="project.id" />
                </el-select>
              </el-form-item>
            </el-form>
          </div>
        </div>

        <div class="selection-column selection-actions">
          <div class="selection-panel selection-actions-panel">
            <div class="section-title">范围与操作 <span class="section-paw">喵</span></div>
            <div class="section-subtitle">目录留空时比较整项目；同步按钮会在比较完成后自动启用。</div>
            <el-form label-position="top" class="selection-actions-form">
              <el-form-item label="目录范围">
                <el-input v-model="form.basePath" placeholder="留空表示整项目；可填写相对目录路径" />
              </el-form-item>
              <el-form-item label="操作" class="selection-actions-item">
                <div class="button-stack">
                  <el-button type="primary" :loading="comparing" :disabled="syncing" @click="runCompare()">执行比较</el-button>
                  <el-button :loading="isSyncingDirection('leftToRight')" :disabled="!canOpenBatchSync('leftToRight')" @click="openBatchSyncDialog('leftToRight')">整项目左 -> 右</el-button>
                  <el-button :loading="isSyncingDirection('rightToLeft')" :disabled="!canOpenBatchSync('rightToLeft')" @click="openBatchSyncDialog('rightToLeft')">整项目右 -> 左</el-button>
                </div>
              </el-form-item>
            </el-form>
          </div>
        </div>
      </div>
      <div v-if="projectCodeStatus" class="project-code-status-row">
        <el-tag :type="projectCodeStatus.type">{{ projectCodeStatus.title }}</el-tag>
        <span>{{ projectCodeStatus.description }}</span>
      </div>
      <div v-if="comparePathStatus" class="project-code-status-row project-path-status-row">
        <el-tag :type="comparePathStatus.type">{{ comparePathStatus.title }}</el-tag>
        <span>{{ comparePathStatus.description }}</span>
      </div>
    </el-card>

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

    <div v-if="compareResult" class="summary-grid">
      <el-card class="summary-card"><span>总项数</span><strong>{{ compareResult.summary.total }}</strong></el-card>
      <el-card class="summary-card"><span>差异文件</span><strong>{{ compareResult.summary.differentFiles }}</strong></el-card>
      <el-card class="summary-card"><span>差异目录</span><strong>{{ compareResult.summary.differentDirectories }}</strong></el-card>
      <el-card class="summary-card"><span>仅左侧</span><strong>{{ compareResult.summary.leftOnly }}</strong></el-card>
      <el-card class="summary-card"><span>仅右侧</span><strong>{{ compareResult.summary.rightOnly }}</strong></el-card>
    </div>

    <el-alert
      v-if="compareResult && !hasDiffItems"
      type="info"
      show-icon
      :closable="false"
      title="当前比较范围内未发现差异结果"
      description="可以切换项目、调整目录范围，或刷新基础数据后重新执行比较。"
    />

    <div class="content-grid">
      <el-card class="diff-list-card neko-surface">
        <template #header>
          <div class="card-header">
            <span>差异浏览</span>
            <el-tag v-if="compareResult" type="info">Finder 分栏 + Monaco Diff</el-tag>
          </div>
        </template>
        <div class="finder-panel" :class="{ 'is-empty-panel': diffTreeNodes.length === 0 }">
          <FinderColumns
            :nodes="diffTreeNodes"
            :selected-path="selectedPath"
            :status-label-map="statusLabelMap"
            :status-type-map="statusTypeMap"
            empty-text="执行比较后在这里按目录分栏浏览差异"
            @select="handleFinderSelect"
          />
        </div>
      </el-card>

      <el-card class="diff-detail-card neko-surface">
        <template #header>
          <div class="card-header">
            <span>文件差异详情</span>
            <el-tooltip v-if="selectedNode" :content="selectedNode.path" placement="top-end">
              <span class="detail-path">{{ selectedNode.path }}</span>
            </el-tooltip>
          </div>
        </template>
        <div v-if="selectedNode" class="selected-node-summary">
          <div class="selected-node-meta">
            <div class="selected-node-tags">
              <el-tag :type="selectedNode.entryType === 'directory' ? 'warning' : 'primary'">
                {{ selectedNode.entryType === 'directory' ? '目录' : '文件' }}
              </el-tag>
              <el-tag v-if="selectedNode.status && selectedNode.status !== 'context'" :type="statusTypeMap[selectedNode.status] || 'info'">
                {{ statusLabelMap[selectedNode.status] || selectedNode.status }}
              </el-tag>
            </div>
            <el-tooltip :content="selectedNode.path" placement="top-start">
              <span class="selected-node-path">{{ selectedNode.path }}</span>
            </el-tooltip>
          </div>
          <div class="selected-node-actions">
            <el-button v-if="selectedNode.entryType === 'file'" size="small" @click="openFileDiff(selectedNode)">刷新差异</el-button>
            <el-button v-if="canFullscreenDiff" size="small" :icon="FullScreen" @click="openDiffFullscreen">全屏对比</el-button>
            <el-button v-if="canSyncLeftToRight(selectedNode)" size="small" type="primary" :loading="isSyncingRow('leftToRight', selectedNode)" :disabled="syncing" @click="syncItem('leftToRight', selectedNode)">左 -> 右</el-button>
            <el-button v-if="canSyncRightToLeft(selectedNode)" size="small" type="success" :loading="isSyncingRow('rightToLeft', selectedNode)" :disabled="syncing" @click="syncItem('rightToLeft', selectedNode)">右 -> 左</el-button>
          </div>
        </div>
        <div v-if="selectedNode && selectedNode.entryType === 'directory'" class="directory-detail">
          <el-descriptions border :column="1">
            <el-descriptions-item label="目录路径">{{ selectedNode.path }}</el-descriptions-item>
            <el-descriptions-item label="差异类型">{{ statusLabelMap[selectedNode.status] || '目录上下文' }}</el-descriptions-item>
            <el-descriptions-item label="左侧摘要">{{ selectedNode.leftHash || '-' }}</el-descriptions-item>
            <el-descriptions-item label="右侧摘要">{{ selectedNode.rightHash || '-' }}</el-descriptions-item>
          </el-descriptions>
          <el-alert :closable="false" type="info" show-icon title="继续点击右侧分栏可深入到该目录的下一层差异。" />
        </div>
        <div v-if="fileDiffLoading" class="detail-state">
          <div class="neko-loading-copy">猫咪正在逐行比对两侧文件，请稍等一下。</div>
          <el-skeleton :rows="10" animated />
        </div>
        <div v-else-if="selectedNode && selectedNode.entryType === 'file' && fileDiff && canRenderTextDiff(fileDiff)" class="monaco-wrapper">
          <MonacoDiff :left-content="fileDiff.left.content || ''" :right-content="fileDiff.right.content || ''" :language="diffPreviewLanguage" />
        </div>
        <div v-else-if="selectedNode && selectedNode.entryType === 'file' && fileDiff" class="binary-detail">
          <el-descriptions border :column="1">
            <el-descriptions-item label="左侧是否存在">{{ fileDiff.left.exists ? '是' : '否' }}</el-descriptions-item>
            <el-descriptions-item label="右侧是否存在">{{ fileDiff.right.exists ? '是' : '否' }}</el-descriptions-item>
            <el-descriptions-item label="左侧摘要">{{ fileDiff.left.hash || '-' }}</el-descriptions-item>
            <el-descriptions-item label="右侧摘要">{{ fileDiff.right.hash || '-' }}</el-descriptions-item>
            <el-descriptions-item label="左侧大小">{{ fileDiff.left.size || 0 }} bytes</el-descriptions-item>
            <el-descriptions-item label="右侧大小">{{ fileDiff.right.size || 0 }} bytes</el-descriptions-item>
          </el-descriptions>
        </div>
        <div v-else class="compare-empty-panel compare-empty-panel--detail">
          <NekoEmptyState title="先让猫咪选中一个差异项" description="从左侧分栏挑一个目录或文件，它就会把差异详情和可同步动作带到这里。" compact />
        </div>
      </el-card>
    </div>

    <el-dialog
      v-model="syncDialogVisible"
      width="980"
      :close-on-click-modal="false"
      :destroy-on-close="false"
      class="sync-dialog"
    >
      <template #header>
        <div class="sync-dialog-header">
          <div class="sync-dialog-header-main">
            <strong>{{ syncDialogTitle }}</strong>
            <span>{{ syncDialogDescription }}</span>
          </div>
          <el-tag type="info">默认勾选全部文件修改项</el-tag>
        </div>
      </template>
      <div class="sync-dialog-summary">
        <span>将要执行 {{ syncDialogRows.length }} 个文件级修改项。</span>
        <span>当前已勾选 {{ syncSelectedPaths.length }} 项。</span>
      </div>
      <div class="sync-tree-shell">
        <el-tree
          :key="syncTreeVersion"
          ref="syncTreeRef"
          :data="syncTreeData"
          :default-expanded-keys="syncExpandedKeys"
          node-key="path"
          show-checkbox
          class="sync-tree"
          @check="handleSyncTreeCheck"
        >
          <template #default="{ data }">
            <div class="sync-tree-node">
              <div class="sync-tree-node-main">
                <span class="sync-tree-node-name" :title="data.path">{{ data.label }}</span>
                <el-tag v-if="data.isFile" size="small" :type="statusTypeMap[data.status] || 'info'">{{ statusLabelMap[data.status] || data.status }}</el-tag>
                <el-tag v-else size="small" type="warning">目录</el-tag>
              </div>
            </div>
          </template>
        </el-tree>
      </div>
      <template #footer>
        <el-button @click="syncDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="syncSubmitting" @click="confirmBatchSync">确认执行</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="diffFullscreen" class="diff-fullscreen-dialog" fullscreen destroy-on-close>
      <template #header>
        <div class="diff-fullscreen-header">
          <div class="diff-fullscreen-copy">
            <strong>文件全屏对比</strong>
            <span>{{ selectedNode?.path || '未选择文件' }}</span>
          </div>
        </div>
      </template>
      <div class="diff-fullscreen-body">
        <MonacoDiff
          v-if="fileDiff && canRenderTextDiff(fileDiff)"
          :left-content="fileDiff.left.content || ''"
          :right-content="fileDiff.right.content || ''"
          :language="diffPreviewLanguage"
          height="calc(100vh - 132px)"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, nextTick, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { FullScreen, Refresh } from '@element-plus/icons-vue'
import FinderColumns from '@/components/FinderColumns.vue'
import NekoOperationProgress from '@/components/NekoOperationProgress.vue'
import NekoPageHeader from '@/components/NekoPageHeader.vue'
import NekoEmptyState from '@/components/NekoEmptyState.vue'
import { buildCompareTree, getNodeByPath } from '@/utils/finder'

const MonacoDiff = defineAsyncComponent(() => import('@/components/MonacoDiff.vue'))

const router = useRouter()

document.title = '比较工作台'

const loading = ref(false)
const comparing = ref(false)
const fileDiffLoading = ref(false)
const nodes = ref([])
const localProjects = ref([])
const leftProjects = ref([])
const rightProjects = ref([])
const compareResult = ref(null)
const compareItems = ref([])
const diffTreeNodes = ref([])
const selectedPath = ref('')
const fileDiff = ref(null)
const diffFullscreen = ref(false)
const remoteProjectCache = ref({})
const syncDialogVisible = ref(false)
const syncSubmitting = ref(false)
const syncDirection = ref('leftToRight')
const syncDialogRows = ref([])
const syncTreeData = ref([])
const syncExpandedKeys = ref([])
const syncTreeVersion = ref(0)
const syncSelectedPaths = ref([])
const syncTreeRef = ref(null)
const syncAction = ref({ loading: false, direction: '', path: '' })
const operationProgress = ref(createOperationProgress())
const form = ref({
  leftNodeId: 0,
  leftProjectId: null,
  rightNodeId: 0,
  rightProjectId: null,
  basePath: '',
})

const statusLabelMap = {
  left_only: '仅左侧',
  right_only: '仅右侧',
  file_changed: '文件变更',
  directory_changed: '目录变更',
  type_changed: '类型变更',
}

const statusTypeMap = {
  left_only: 'primary',
  right_only: 'success',
  file_changed: 'danger',
  directory_changed: 'warning',
  type_changed: 'info',
}

const batchSyncStatusMap = {
  leftToRight: ['left_only', 'file_changed'],
  rightToLeft: ['right_only', 'file_changed'],
}

const enabledNodes = computed(() => nodes.value.filter((node) => node.enabled === 1))
const selectedNode = computed(() => getNodeByPath(diffTreeNodes.value, selectedPath.value))
const hasDiffItems = computed(() => (compareItems.value || []).length > 0)
const syncing = computed(() => syncAction.value.loading || syncSubmitting.value)
const diffPreviewLanguage = computed(() => detectPreviewLanguage(selectedNode.value?.path || ''))
const canFullscreenDiff = computed(() => Boolean(selectedNode.value?.entryType === 'file' && fileDiff.value && canRenderTextDiff(fileDiff.value) && !fileDiffLoading.value))
const projectCodeStatus = computed(() => {
  const mismatch = getProjectCodeMismatch()
  if (mismatch) {
    return {
      type: 'warning',
      title: '编码不一致',
      description: `左侧 ${mismatch.leftCode}，右侧 ${mismatch.rightCode}。编码不一致可能不是同一个项目。`,
    }
  }
  const leftProject = getSelectedProject('left')
  const rightProject = getSelectedProject('right')
  if (!leftProject || !rightProject) {
    return null
  }
  const code = String(leftProject.code || '').trim()
  if (!code) {
    return null
  }
  return {
    type: 'success',
    title: '编码一致',
    description: `左右项目编码均为 ${code}。`,
  }
})
const comparePathStatus = computed(() => {
  if (!compareResult.value) {
    return null
  }
  const leftCaseSensitive = Boolean(compareResult.value.leftPathCaseSensitive)
  const rightCaseSensitive = Boolean(compareResult.value.rightPathCaseSensitive)
  const leftLabel = leftCaseSensitive ? '区分大小写' : '不区分大小写'
  const rightLabel = rightCaseSensitive ? '区分大小写' : '不区分大小写'
  if (leftCaseSensitive === rightCaseSensitive) {
    return {
      type: leftCaseSensitive ? 'info' : 'success',
      title: leftCaseSensitive ? '路径大小写敏感' : '路径大小写不敏感',
      description: `左侧 ${leftLabel}，右侧 ${rightLabel}。${leftCaseSensitive ? '仅大小写不同的路径会被视为不同文件。' : '仅大小写不同的路径会被视为同一路径。'}`,
    }
  }
  return {
    type: 'warning',
    title: '路径规则不一致',
    description: `左侧 ${leftLabel}，右侧 ${rightLabel}。仅大小写不同的路径在比较时可能被折叠，同步时若会覆盖已有大小写别名文件，系统会阻止写入。`,
  }
})
const syncDialogTitle = computed(() => {
  const sourceLabel = buildEndpointLabel(syncDirection.value === 'leftToRight' ? 'left' : 'right')
  const targetLabel = buildEndpointLabel(syncDirection.value === 'leftToRight' ? 'right' : 'left')
  return `${sourceLabel} 覆盖 ${targetLabel}`
})
const syncDialogDescription = computed(() => form.value.basePath ? `当前比较范围：${form.value.basePath}` : '当前比较范围：整项目')

// 创建统一的操作进度模型，供比较、同步等动作复用。
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

// 初始化某次操作的步骤进度，并重置上一次残留状态。
function beginOperationProgress(title, stepTitles, patch = {}) {
  operationProgress.value = {
    ...createOperationProgress(),
    visible: true,
    title,
    steps: stepTitles.map((step, index) => ({ key: `${title}-${index}`, title: step })),
    ...patch,
  }
}

// 增量更新操作进度，不覆盖未传入的已有字段。
function updateOperationProgress(patch) {
  operationProgress.value = {
    ...operationProgress.value,
    visible: true,
    ...patch,
  }
}

// 将当前操作标记为成功完成。
function finishOperationProgress(message, patch = {}) {
  updateOperationProgress({ status: 'success', percent: 100, message, ...patch })
}

// 将当前操作标记为失败，供页面统一展示异常状态。
function failOperationProgress(message, patch = {}) {
  updateOperationProgress({ status: 'error', message, ...patch })
}

// 首次进入页面时并行加载节点和项目基础数据。
async function bootstrap() {
  loading.value = true
  try {
    await Promise.all([fetchNodes(), fetchLocalProjects()])
    if (form.value.leftNodeId === form.value.rightNodeId) {
      form.value.rightNodeId = pickAlternativeNodeId(form.value.leftNodeId)
    }
    await Promise.all([loadProjectsForSide('left'), loadProjectsForSide('right')])
  } finally {
    loading.value = false
  }
}

// 拉取远程节点列表，供左右基准选择使用。
async function fetchNodes() {
  try {
    const res = await fetch('/api/nodes')
    const data = await res.json()
    if (data.success) {
      nodes.value = data.data || []
    } else {
      ElMessage.error(data.message || '获取节点列表失败')
    }
  } catch {
    ElMessage.error('获取节点列表失败')
  }
}

// 读取全部本地项目，比较页需要在本机节点模式下复用。
async function fetchLocalProjects() {
  try {
    const result = []
    let pageNo = 1
    let total = 0
    do {
      const res = await fetch(`/api/projects?pageNo=${pageNo}&pageSize=100`)
      const data = await res.json()
      if (!data.success) {
        ElMessage.error(data.message || '获取本地项目失败')
        return
      }
      const page = data.data || {}
      const rows = page.result || []
      result.push(...rows)
      total = page.total || 0
      pageNo += 1
      if (rows.length === 0) {
        break
      }
    } while (result.length < total)
    localProjects.value = result
  } catch {
    ElMessage.error('获取本地项目失败')
  }
}

// 仅在左右两侧都可按文本展示时启用 Monaco Diff。
function canRenderTextDiff(diffData) {
  const leftTextCapable = diffData.left.exists ? diffData.left.text : true
  const rightTextCapable = diffData.right.exists ? diffData.right.text : true
  return leftTextCapable && rightTextCapable
}

// 远程项目列表按节点缓存，避免同节点重复请求。
async function fetchRemoteProjects(nodeId) {
  if (remoteProjectCache.value[nodeId]) {
    return remoteProjectCache.value[nodeId]
  }
  const res = await fetch(`/api/nodes/${nodeId}/projects`)
  const data = await res.json()
  if (!data.success) {
    throw new Error(data.message || '获取远程项目失败')
  }
  remoteProjectCache.value[nodeId] = data.data || []
  return remoteProjectCache.value[nodeId]
}

// 根据当前侧选择的节点来源切换可选项目列表。
async function loadProjectsForSide(side) {
  const nodeId = side === 'left' ? form.value.leftNodeId : form.value.rightNodeId
  const projectRef = side === 'left' ? leftProjects : rightProjects
  const idKey = side === 'left' ? 'leftProjectId' : 'rightProjectId'
  projectRef.value = []
  form.value[idKey] = null
  if (nodeId === 0) {
    projectRef.value = localProjects.value
    return
  }
  try {
    projectRef.value = await fetchRemoteProjects(nodeId)
  } catch (error) {
    ElMessage.error(error.message || '获取远程项目失败')
  }
}

// 当两侧选中了同一节点时，优先挑一个替代节点避免默认对比自己。
function pickAlternativeNodeId(currentNodeId) {
  const candidates = [0, ...enabledNodes.value.map((node) => node.id)].filter((nodeId, index, list) => list.indexOf(nodeId) === index)
  const preferred = candidates.filter((nodeId) => nodeId !== currentNodeId)
  if (currentNodeId === 0) {
    return preferred.find((nodeId) => nodeId !== 0) ?? currentNodeId
  }
  return preferred[0] ?? currentNodeId
}

// 节点变化时同时重置比较状态，并尽量保持左右节点不同。
async function handleNodeChange(side) {
  const oppositeSide = side === 'left' ? 'right' : 'left'
  const nodeKey = side === 'left' ? 'leftNodeId' : 'rightNodeId'
  const oppositeNodeKey = oppositeSide === 'left' ? 'leftNodeId' : 'rightNodeId'
  resetCompareState()
  if (form.value[nodeKey] === form.value[oppositeNodeKey]) {
    form.value[oppositeNodeKey] = pickAlternativeNodeId(form.value[nodeKey])
  }
  await Promise.all([loadProjectsForSide(side), loadProjectsForSide(oppositeSide)])
}

// 清空当前比较结果、选中项和 diff 详情。
function resetCompareState() {
  compareResult.value = null
  compareItems.value = []
  diffTreeNodes.value = []
  selectedPath.value = ''
  fileDiff.value = null
  diffFullscreen.value = false
}

// 执行比较，并同步推进步骤进度与默认选中逻辑。
async function runCompare(options = {}) {
  const { silentSuccess = false, skipCodeConfirm = false, reportProgress = true } = options
  if (!form.value.leftProjectId || !form.value.rightProjectId) {
    ElMessage.warning('请先选择左右两侧项目')
    return
  }
  if (syncing.value) {
    ElMessage.warning('同步操作进行中，请稍后再执行比较')
    return
  }
  if (!skipCodeConfirm) {
    const confirmed = await confirmProjectCodeMismatch('执行比较')
    if (!confirmed) {
      return
    }
  }
  if (reportProgress) {
    beginOperationProgress('执行比较', ['校验比较对象', '请求差异数据', '整理差异树', '定位首个差异'], {
      activeStep: 0,
      percent: 10,
      message: '正在校验左右项目与目录范围。',
    })
  }
  comparing.value = true
  fileDiff.value = null
  try {
    if (reportProgress) {
      updateOperationProgress({ activeStep: 1, percent: 35, message: '正在请求服务端生成差异结果。' })
    }
    const res = await fetch('/api/compare/projects', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        leftNodeId: form.value.leftNodeId,
        leftProjectId: form.value.leftProjectId,
        rightNodeId: form.value.rightNodeId,
        rightProjectId: form.value.rightProjectId,
        basePath: form.value.basePath,
      }),
    })
    const data = await res.json()
    if (data.success) {
      compareResult.value = data.data
      compareItems.value = data.data?.items || []
      diffTreeNodes.value = buildCompareTree(compareItems.value)
      selectedPath.value = ''
      fileDiff.value = null
      const firstNode = findFirstDisplayNode(diffTreeNodes.value)
      if (reportProgress) {
        updateOperationProgress({ activeStep: 2, percent: 70, message: `差异树已生成，共 ${compareItems.value.length} 项。` })
      }
      if (firstNode) {
        if (reportProgress) {
          updateOperationProgress({ activeStep: 3, percent: 90, currentPath: firstNode.path, message: '正在定位首个差异项。' })
        }
        handleFinderSelect(firstNode)
      }
      if (compareItems.value.length === 0) {
        ElMessage.info('比较完成，当前范围内未发现差异结果')
      } else if (!silentSuccess) {
        ElMessage.success(`比较完成，发现 ${compareItems.value.length} 项差异`)
      }
      if (reportProgress) {
        finishOperationProgress(compareItems.value.length === 0 ? '比较完成，当前范围内未发现差异结果。' : `比较完成，发现 ${compareItems.value.length} 项差异。`)
      }
    }
    else {
      if (reportProgress) {
        failOperationProgress(data.message || '比较失败')
      }
      ElMessage.error(data.message || '比较失败')
    }
  } catch {
    if (reportProgress) {
      failOperationProgress('比较失败，请稍后重试。')
    }
    ElMessage.error('比较失败')
  } finally {
    comparing.value = false
  }
}

// Finder 默认展示差异树的第一项，减少首次进入的空白状态。
function findFirstDisplayNode(nodes) {
  if (!nodes?.length) {
    return null
  }
  return nodes[0]
}

// 将节点 ID 转成用户可读的展示名称。
function getNodeLabel(nodeId) {
  if (nodeId === 0) {
    return '本机节点'
  }
  return nodes.value.find((node) => node.id === nodeId)?.name || `节点 #${nodeId}`
}

// 返回左右侧当前已选项目的展示标签。
function getProjectLabel(side) {
  const projectId = side === 'left' ? form.value.leftProjectId : form.value.rightProjectId
  const projectList = side === 'left' ? leftProjects.value : rightProjects.value
  const project = projectList.find((item) => item.id === projectId)
  return formatProjectOptionLabel(project) || `项目 #${projectId}`
}

// 统一项目下拉文案，编码存在时一并展示。
function formatProjectOptionLabel(project) {
  if (!project) {
    return ''
  }
  const code = String(project.code || '').trim()
  return code ? `${project.name} [${code}]` : project.name
}

// 读取当前侧已选中的完整项目对象。
function getSelectedProject(side) {
  const projectId = side === 'left' ? form.value.leftProjectId : form.value.rightProjectId
  const projectList = side === 'left' ? leftProjects.value : rightProjects.value
  return projectList.find((project) => project.id === projectId) || null
}

// 按项目编码匹配对侧候选项目，用于自动联动选择。
function findProjectByCode(projects, code) {
  const normalizedCode = String(code || '').trim()
  if (!normalizedCode) {
    return null
  }
  return projects.find((project) => String(project.code || '').trim() === normalizedCode) || null
}

// 比较左右项目编码是否一致，用于同步和比较前确认。
function getProjectCodeMismatch() {
  const leftProject = getSelectedProject('left')
  const rightProject = getSelectedProject('right')
  if (!leftProject || !rightProject) {
    return null
  }
  const leftCode = String(leftProject.code || '').trim()
  const rightCode = String(rightProject.code || '').trim()
  if (!leftCode || !rightCode || leftCode === rightCode) {
    return null
  }
  return { leftProject, rightProject, leftCode, rightCode }
}

// 编码不一致时弹出确认，避免误把不同项目互相覆盖。
async function confirmProjectCodeMismatch(actionLabel) {
  const mismatch = getProjectCodeMismatch()
  if (!mismatch) {
    return true
  }
  try {
    await ElMessageBox.confirm(
      `左侧项目编码为 ${mismatch.leftCode}，右侧项目编码为 ${mismatch.rightCode}。编码不一致可能不是同一个项目，确定继续${actionLabel}吗？`,
      '项目编码不一致',
      {
        type: 'warning',
        confirmButtonText: '继续',
        cancelButtonText: '取消',
      },
    )
    return true
  } catch {
    return false
  }
}

// 路径大小写规则不一致时提示风险，避免不同文件系统之间误解“仅大小写不同”的路径。
async function confirmPathCaseRuleMismatch(actionLabel) {
  if (!compareResult.value) {
    return true
  }
  const leftCaseSensitive = Boolean(compareResult.value.leftPathCaseSensitive)
  const rightCaseSensitive = Boolean(compareResult.value.rightPathCaseSensitive)
  if (leftCaseSensitive === rightCaseSensitive) {
    return true
  }
  try {
    await ElMessageBox.confirm(
      `左侧工作站${leftCaseSensitive ? '' : '不'}区分路径大小写，右侧工作站${rightCaseSensitive ? '' : '不'}区分路径大小写。仅大小写不同的路径在两端可能代表不同语义；如果目标侧已存在大小写别名文件，系统会阻止写入。确定继续${actionLabel}吗？`,
      '路径大小写规则不一致',
      {
        type: 'warning',
        confirmButtonText: '继续',
        cancelButtonText: '取消',
      },
    )
    return true
  } catch {
    return false
  }
}

// 选择一侧项目后，尝试按编码自动联动对侧项目。
function trySyncOppositeProject(side) {
  const selectedProject = getSelectedProject(side)
  if (!selectedProject) {
    return
  }
  const oppositeSide = side === 'left' ? 'right' : 'left'
  const oppositeProjects = oppositeSide === 'left' ? leftProjects.value : rightProjects.value
  const oppositeKey = oppositeSide === 'left' ? 'leftProjectId' : 'rightProjectId'
  const matchedProject = findProjectByCode(oppositeProjects, selectedProject.code)
  if (!matchedProject || form.value[oppositeKey] === matchedProject.id) {
    return
  }
  form.value[oppositeKey] = matchedProject.id
}

// 项目变化后清空现有比较结果，并触发对侧自动匹配。
function handleProjectChange(side) {
  resetCompareState()
  trySyncOppositeProject(side)
}

// 构造“节点 / 项目”组合标签，供同步标题和提示文案复用。
function buildEndpointLabel(side) {
  const nodeId = side === 'left' ? form.value.leftNodeId : form.value.rightNodeId
  return `${getNodeLabel(nodeId)} / ${getProjectLabel(side)}`
}

// 根据同步方向筛出允许执行批量覆盖的文件项。
function getBatchSyncRows(direction) {
  const allowedStatuses = batchSyncStatusMap[direction] || []
  return (compareItems.value || [])
    .filter((item) => item.entryType === 'file' && allowedStatuses.includes(item.status))
    .map((item) => ({ ...item }))
}

// 判断当前方向是否允许打开整项目覆盖确认窗。
function canOpenBatchSync(direction) {
  return Boolean(compareResult.value && !comparing.value && !syncing.value && getBatchSyncRows(direction).length > 0)
}

// 判断某个方向的整批同步是否正在执行。
function isSyncingDirection(direction) {
  return syncAction.value.loading && syncAction.value.direction === direction && !syncAction.value.path
}

// 判断某个行级按钮是否处于同步中。
function isSyncingRow(direction, row) {
  return syncAction.value.loading && syncAction.value.direction === direction && syncAction.value.path === row?.path
}

// 打开整项目覆盖弹窗，并预构建树状勾选数据。
async function openBatchSyncDialog(direction) {
  if (syncing.value) {
    ElMessage.warning('同步操作进行中，请稍后再试')
    return
  }
  if (!compareResult.value) {
    ElMessage.warning('请先执行比较，再选择需要同步的文件修改项')
    return
  }
  const rows = getBatchSyncRows(direction)
  if (rows.length === 0) {
    ElMessage.warning('当前没有可执行的文件级修改项')
    return
  }
  syncDirection.value = direction
  syncDialogRows.value = rows
  const syncTree = buildSyncTree(rows)
  syncTreeData.value = syncTree.data
  syncExpandedKeys.value = syncTree.expandedKeys
  syncTreeVersion.value += 1
  syncSelectedPaths.value = rows.map((item) => item.path)
  syncDialogVisible.value = true
  await nextTick()
  syncTreeRef.value?.setCheckedKeys(syncSelectedPaths.value)
}

// 根据树控件的勾选结果同步出最终要提交的文件路径列表。
function handleSyncTreeCheck() {
  syncSelectedPaths.value = syncTreeRef.value?.getCheckedKeys(true) || []
}

// 校验勾选结果后发起整项目覆盖。
async function confirmBatchSync() {
  if (syncAction.value.loading) {
    ElMessage.warning('同步操作进行中，请勿重复提交')
    return
  }
  if (syncSelectedPaths.value.length === 0) {
    ElMessage.warning('请至少勾选一个文件修改项')
    return
  }
  syncSubmitting.value = true
  try {
    await syncRequest(syncDirection.value, form.value.basePath ? 'directory' : 'project', form.value.basePath || '', syncSelectedPaths.value)
    syncDialogVisible.value = false
  } finally {
    syncSubmitting.value = false
  }
}

// Finder 选中变化时，同步切换右侧详情面板内容。
function handleFinderSelect(node) {
  selectedPath.value = node.path
  if (node.entryType === 'file') {
    openFileDiff(node)
  } else {
    fileDiff.value = null
    diffFullscreen.value = false
  }
}

// 单独读取某个文件的左右两侧差异内容。
async function openFileDiff(row) {
  fileDiffLoading.value = true
  fileDiff.value = null
  try {
    const res = await fetch('/api/compare/file-diff', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        leftNodeId: form.value.leftNodeId,
        leftProjectId: form.value.leftProjectId,
        rightNodeId: form.value.rightNodeId,
        rightProjectId: form.value.rightProjectId,
        path: row.path,
      }),
    })
    const data = await res.json()
    if (data.success) {
      fileDiff.value = data.data
    } else {
      diffFullscreen.value = false
      ElMessage.error(data.message || '读取文件差异失败')
    }
  } catch {
    diffFullscreen.value = false
    ElMessage.error('读取文件差异失败')
  } finally {
    fileDiffLoading.value = false
  }
}

// 文本差异时允许切换到全屏对比模式。
function openDiffFullscreen() {
  if (!canFullscreenDiff.value) {
    return
  }
  diffFullscreen.value = true
}

// 判断当前差异项是否允许从左侧覆盖到右侧。
function canSyncLeftToRight(row) {
  return ['left_only', 'file_changed', 'directory_changed', 'type_changed'].includes(row.status)
}

// 判断当前差异项是否允许从右侧覆盖到左侧。
function canSyncRightToLeft(row) {
  return ['right_only', 'file_changed', 'directory_changed', 'type_changed'].includes(row.status)
}

// 目录走目录同步，文件走单文件同步。
async function syncItem(direction, row) {
  const scopeType = row.entryType === 'directory' ? 'directory' : 'file'
  await syncRequest(direction, scopeType, row.path)
}

// 统一处理单文件、目录和整项目同步，并驱动进度面板刷新。
async function syncRequest(direction, scopeType, path, selectedPaths = []) {
  if (!form.value.leftProjectId || !form.value.rightProjectId) {
    ElMessage.warning('请先完成项目比较')
    return
  }
  if (syncAction.value.loading) {
    ElMessage.warning('同步操作进行中，请勿重复操作')
    return
  }
  const confirmed = await confirmProjectCodeMismatch(direction === 'leftToRight' ? '执行合并' : '执行拉取')
  if (!confirmed) {
    return
  }
  const pathRuleConfirmed = await confirmPathCaseRuleMismatch(direction === 'leftToRight' ? '执行合并' : '执行拉取')
  if (!pathRuleConfirmed) {
    return
  }
  const leftToRight = direction === 'leftToRight'
  const syncRows = getSyncRowsForRequest(direction, scopeType, path, selectedPaths)
  const progressTitle = selectedPaths.length > 0 || scopeType === 'project'
    ? (leftToRight ? '整项目左 -> 右' : '整项目右 -> 左')
    : scopeType === 'directory'
      ? (leftToRight ? '目录左 -> 右' : '目录右 -> 左')
      : (leftToRight ? '单文件左 -> 右' : '单文件右 -> 左')

  beginOperationProgress(progressTitle, ['确认同步方向', '准备待同步项', '写入目标文件', '刷新比较结果'], {
    activeStep: 0,
    percent: 8,
    total: syncRows.length,
    processed: 0,
    message: '正在确认本次同步方向与范围。',
  })

  syncAction.value = {
    loading: true,
    direction,
    path: selectedPaths.length > 0 ? '' : path,
  }
  try {
    updateOperationProgress({ activeStep: 1, percent: syncRows.length > 0 ? 16 : 35, total: syncRows.length, message: syncRows.length > 0 ? `已准备 ${syncRows.length} 个待同步文件。` : '当前范围内没有可直接逐项同步的文件，改为走服务端批量同步。' })
    let copied = 0
    let skipped = 0
    const failed = []
    if (syncRows.length > 0) {
      updateOperationProgress({ activeStep: 2, percent: 18, message: '正在逐项写入目标文件。' })
      for (let index = 0; index < syncRows.length; index += 1) {
        const row = syncRows[index]
        updateOperationProgress({
          activeStep: 2,
          processed: index,
          total: syncRows.length,
          percent: 18 + Math.round((index / syncRows.length) * 72),
          currentPath: row.path,
          message: `正在同步第 ${index + 1} / ${syncRows.length} 项。`,
        })
        const result = await requestSyncApi(direction, 'file', row.path, [])
        if (!result.success) {
          failed.push({ path: row.path, message: result.message || '同步失败' })
          continue
        }
        copied += result.data?.copied || 0
        skipped += result.data?.skipped || 0
        failed.push(...(result.data?.failed || []))
      }
      updateOperationProgress({ activeStep: 2, processed: syncRows.length, total: syncRows.length, percent: 92, message: '文件写入完成，准备刷新比较结果。' })
    } else {
      updateOperationProgress({ activeStep: 2, percent: 60, message: '正在执行服务端批量同步。' })
      const result = await requestSyncApi(direction, scopeType, path, selectedPaths)
      if (!result.success) {
        failOperationProgress(result.message || '同步失败')
        ElMessage.error(result.message || '同步失败')
        return
      }
      copied = result.data?.copied || 0
      skipped = result.data?.skipped || 0
      failed.push(...(result.data?.failed || []))
    }

    updateOperationProgress({ activeStep: 3, percent: 96, message: '同步完成，正在刷新比较结果。' })
    await runCompare({ silentSuccess: true, skipCodeConfirm: true, reportProgress: false })
    if (copied === 0 && failed.length === 0) {
      ElMessage.info('同步完成，但当前没有实际写入的差异项')
    } else if (failed.length > 0) {
      ElMessage.warning(`同步完成，成功 ${copied} 项，跳过 ${skipped} 项，失败 ${failed.length} 项`)
    } else {
      ElMessage.success(`同步完成，成功 ${copied} 项${skipped > 0 ? `，跳过 ${skipped} 项` : ''}`)
    }
    finishOperationProgress(failed.length > 0 ? `同步完成，成功 ${copied} 项，失败 ${failed.length} 项。` : `同步完成，成功 ${copied} 项${skipped > 0 ? `，跳过 ${skipped} 项` : ''}。`, {
      processed: syncRows.length,
      total: syncRows.length,
      currentPath: '',
    })
  } catch {
    failOperationProgress('同步失败，请稍后重试。')
    ElMessage.error('同步失败')
  } finally {
    syncAction.value = { loading: false, direction: '', path: '' }
  }
}

// 将平铺的文件路径整理成弹窗树，并压缩连续单子目录链。
function buildSyncTree(rows) {
  const root = []
  const nodeMap = new Map()
  rows.forEach((row) => {
    const segments = row.path.split('/').filter(Boolean)
    let currentLevel = root
    let currentPath = ''
    segments.forEach((segment, index) => {
      currentPath = currentPath ? `${currentPath}/${segment}` : segment
      let node = nodeMap.get(currentPath)
      if (!node) {
        node = {
          path: currentPath,
          label: segment,
          children: [],
          isFile: index === segments.length - 1,
          status: index === segments.length - 1 ? row.status : 'context',
        }
        nodeMap.set(currentPath, node)
        currentLevel.push(node)
      }
      currentLevel = node.children
    })
  })
  const compacted = compactSyncTreeNodes(sortSyncTree(root))
  return {
    data: compacted,
    expandedKeys: collectSyncExpandedKeys(compacted),
  }
}

// 目录优先、同层按名称排序，保证树状勾选稳定可读。
function sortSyncTree(nodes) {
  return [...nodes]
    .sort((left, right) => {
      if (left.isFile !== right.isFile) {
        return left.isFile ? 1 : -1
      }
      return left.label.localeCompare(right.label, 'zh-Hans-CN')
    })
    .map((node) => ({
      ...node,
      children: sortSyncTree(node.children || []),
    }))
}

// 将连续且只有单个目录子节点的路径压缩为一层展示，避免树太深。
function compactSyncTreeNodes(nodes) {
  return (nodes || []).map((node) => compactSyncTreeNode(node))
}

function compactSyncTreeNode(node) {
  const nextChildren = compactSyncTreeNodes(node.children || [])
  const baseNode = {
    ...node,
    children: nextChildren,
    aliases: node.aliases?.length ? [...node.aliases] : (node.path ? [node.path] : []),
  }
  if (baseNode.isFile) {
    return baseNode
  }

  const chainNodes = [baseNode]
  let current = baseNode
  while (current.children?.length === 1 && !current.children[0].isFile) {
    current = current.children[0]
    chainNodes.push(current)
  }

  if (chainNodes.length === 1) {
    return baseNode
  }

  const tailNode = chainNodes[chainNodes.length - 1]
  return {
    ...tailNode,
    label: chainNodes.map((item) => item.label).join(' / '),
    compacted: true,
    aliases: chainNodes.flatMap((item) => item.aliases || [item.path]),
    children: tailNode.children || [],
  }
}

// 默认只展开到首个实际文件差异分支，避免整棵树同时铺开。
function collectSyncExpandedKeys(nodes) {
  const expandedKeys = []
  const firstBranch = findFirstSyncBranch(nodes)
  firstBranch.forEach((node) => {
    if (!node.isFile) {
      expandedKeys.push(node.path)
    }
  })
  return expandedKeys
}

function findFirstSyncBranch(nodes) {
  for (const node of nodes || []) {
    if (node.isFile) {
      return [node]
    }
    const childBranch = findFirstSyncBranch(node.children || [])
    if (childBranch.length > 0) {
      return [node, ...childBranch]
    }
  }
  return []
}

function walkSyncTree(nodes, visitor) {
  ;(nodes || []).forEach((node) => {
    visitor(node)
    if (node.children?.length) {
      walkSyncTree(node.children, visitor)
    }
  })
}

// 按当前提交范围计算真正需要逐项同步的文件列表。
function getSyncRowsForRequest(direction, scopeType, path, selectedPaths) {
  const normalizedPath = String(path || '').trim().replace(/^\/+/, '')
  const selectedSet = new Set((selectedPaths || []).map((item) => String(item || '').trim().replace(/^\/+/, '')).filter(Boolean))
  const rows = getBatchSyncRows(direction)
  if (selectedSet.size > 0) {
    return rows.filter((item) => selectedSet.has(item.path))
  }
  if (scopeType === 'file') {
    return rows.filter((item) => item.path === normalizedPath)
  }
  if (scopeType === 'directory' && normalizedPath) {
    return rows.filter((item) => item.path === normalizedPath || item.path.startsWith(`${normalizedPath}/`))
  }
  return rows
}

// 统一封装同步接口请求，避免多处分散拼装参数。
async function requestSyncApi(direction, scopeType, path, selectedPaths) {
  const leftToRight = direction === 'leftToRight'
  const res = await fetch('/api/compare/sync', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      sourceNodeId: leftToRight ? form.value.leftNodeId : form.value.rightNodeId,
      sourceProjectId: leftToRight ? form.value.leftProjectId : form.value.rightProjectId,
      targetNodeId: leftToRight ? form.value.rightNodeId : form.value.leftNodeId,
      targetProjectId: leftToRight ? form.value.rightProjectId : form.value.leftProjectId,
      scopeType,
      path,
      selectedPaths,
    }),
  })
  return res.json()
}

// 根据文件后缀推断 Monaco Diff 语言。
function detectPreviewLanguage(path) {
  const normalized = String(path || '').toLowerCase()
  const fileName = normalized.split('/').pop() || normalized
  if (normalized.endsWith('.yaml') || normalized.endsWith('.yml')) {
    return 'yaml'
  }
  if (normalized.endsWith('.json') || normalized.endsWith('.jsonc')) {
    return 'json'
  }
  if (normalized.endsWith('.css') || normalized.endsWith('.scss') || normalized.endsWith('.less')) {
    return 'css'
  }
  if (normalized.endsWith('.html') || normalized.endsWith('.htm') || normalized.endsWith('.vue')) {
    return 'html'
  }
  if (normalized.endsWith('.xml') || normalized.endsWith('.svg')) {
    return 'xml'
  }
  if (normalized.endsWith('.md') || normalized.endsWith('.markdown')) {
    return 'markdown'
  }
  if (normalized.endsWith('.py')) {
    return 'python'
  }
  if (normalized.endsWith('.sql')) {
    return 'sql'
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

onMounted(bootstrap)
</script>

<style scoped>
.compare-container {
  min-height: 100vh;
  padding: 24px;
  box-sizing: border-box;
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
.header-right {
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

.selection-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(0, 1.08fr) minmax(280px, 0.82fr);
  gap: 16px;
}

.selection-column {
  min-width: 0;
}

.selection-panel {
  height: 100%;
  padding: 16px 16px 14px;
  border: 1px solid color-mix(in srgb, var(--el-border-color) 76%, transparent);
  border-radius: 18px;
  background:
    linear-gradient(
      180deg,
      color-mix(in srgb, var(--el-fill-color-light) 70%, var(--el-bg-color-overlay)),
      color-mix(in srgb, var(--el-fill-color) 82%, var(--el-bg-color-overlay))
    );
  box-shadow: inset 0 1px 0 color-mix(in srgb, white 12%, transparent);
}

.selection-actions {
  display: flex;
  align-items: stretch;
}

.project-code-status-row {
  margin-top: 14px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.project-path-status-row {
  align-items: flex-start;
}

.selection-actions-panel {
  display: flex;
  flex-direction: column;
}

.selection-actions-form {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.selection-actions-item {
  flex: 1;
  margin-bottom: 0;
}

.section-title {
  margin-bottom: 12px;
  font-size: 15px;
  font-weight: 600;
}

.section-subtitle {
  margin: -2px 0 14px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.section-paw {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  margin-left: 6px;
  padding: 0 6px;
  border-radius: 999px;
  background: rgba(230, 162, 60, 0.12);
  color: #b56b1e;
  font-size: 11px;
  font-weight: 700;
}

:global(html.dark) .section-paw {
  background: rgba(230, 162, 60, 0.18);
  color: #f0c286;
}

.button-stack {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.button-stack :deep(.el-button) {
  width: 100%;
  margin-left: 0;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.summary-card {
  border-radius: 18px;
}

.summary-card :deep(.el-card__body) {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 88px;
}

.summary-card strong {
  font-size: 28px;
  line-height: 1;
}

.summary-card span {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(0, 1fr);
  gap: 16px;
  flex: 1;
  min-height: 0;
}

.diff-list-card,
.diff-detail-card {
  min-height: 640px;
}

.diff-list-card :deep(.el-card__body),
.diff-detail-card :deep(.el-card__body) {
  height: calc(100% - 8px);
}

.finder-panel {
  height: 100%;
}

.finder-panel.is-empty-panel {
  min-height: 100%;
}

.finder-panel.is-empty-panel :deep(.finder-empty) {
  min-height: 100%;
  border: 1px dashed color-mix(in srgb, var(--el-border-color) 72%, transparent);
  border-radius: 20px;
  background:
    radial-gradient(circle at top, color-mix(in srgb, var(--el-color-warning-light-8) 28%, transparent), transparent 55%),
    color-mix(in srgb, var(--el-fill-color-light) 72%, transparent);
}

.finder-panel.is-empty-panel :deep(.neko-empty-state) {
  min-height: 240px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.detail-path {
  flex: 1;
  min-width: 0;
  text-align: right;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: default;
}

.selected-node-summary {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 12px;
  min-width: 0;
}

.selected-node-meta {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.selected-node-tags {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.selected-node-path {
  display: block;
  min-width: 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: default;
}

.selected-node-actions {
  display: flex;
  flex: none;
  align-items: center;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  position: relative;
  z-index: 1;
}

.directory-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-state,
.binary-detail,
.monaco-wrapper {
  height: 100%;
}

.compare-empty-panel {
  min-height: 260px;
  border: 1px dashed color-mix(in srgb, var(--el-border-color) 70%, transparent);
  border-radius: 20px;
  background:
    radial-gradient(circle at top, color-mix(in srgb, var(--el-color-warning-light-8) 24%, transparent), transparent 58%),
    color-mix(in srgb, var(--el-fill-color-light) 74%, transparent);
}

.compare-empty-panel :deep(.neko-empty-state) {
  min-height: 100%;
}

.compare-empty-panel--detail {
  min-height: 320px;
}

.diff-fullscreen-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.diff-fullscreen-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.diff-fullscreen-copy span {
  color: var(--el-text-color-secondary);
  word-break: break-all;
}

.diff-fullscreen-body {
  min-height: calc(100vh - 132px);
}

.sync-dialog-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.sync-dialog-header-main {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sync-dialog-header-main strong {
  font-size: 16px;
}

.sync-dialog-header-main span {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.sync-dialog-summary {
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.sync-dialog-table {
  margin-bottom: 8px;
}

.sync-tree-shell {
  margin-bottom: 8px;
  max-height: 460px;
  overflow: auto;
  padding: 8px 10px;
  border: 1px solid color-mix(in srgb, var(--el-border-color) 78%, transparent);
  border-radius: 16px;
  background: color-mix(in srgb, var(--el-fill-color-light) 66%, transparent);
}

.sync-tree-node {
  width: 100%;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.sync-tree-node-main {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.sync-tree-node-main {
  flex: 1;
}

.sync-tree-node-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.neko-loading-copy {
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

@media (max-width: 1280px) {
  .selection-grid,
  .summary-grid,
  .content-grid {
    grid-template-columns: 1fr;
  }

  .selection-actions {
    align-items: initial;
  }

  .detail-path {
    display: none;
  }

  .selected-node-summary {
    flex-direction: column;
    align-items: stretch;
  }

  .selected-node-actions {
    justify-content: flex-start;
  }

  .sync-dialog-header,
  .sync-dialog-summary {
    flex-direction: column;
    align-items: flex-start;
  }

  .sync-tree-node {
    flex-direction: column;
    align-items: flex-start;
  }
}

:global(html.dark) .compare-container {
  background:
    radial-gradient(circle at top left, rgba(122, 88, 41, 0.22), transparent 28%),
    radial-gradient(circle at bottom right, rgba(58, 83, 130, 0.18), transparent 26%),
    var(--el-bg-color-page);
}

</style>