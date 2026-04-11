<template>
  <div class="nodes-container">
    <div class="header">
      <div class="header-left">
        <NekoPageHeader
          title="节点管理"
          description="远程节点的地址、共享令牌和探测结果都在这里汇总。"
          tone="green"
          @back="router.push('/')"
        />
      </div>
      <div class="header-right">
        <el-button :icon="Refresh" :loading="refreshing" @click="refreshNodes">刷新</el-button>
        <el-button type="primary" @click="openCreateDialog">添加节点</el-button>
      </div>
    </div>

    <el-card class="table-card neko-surface">
      <el-table :data="nodes" stripe v-loading="loading" element-loading-text="猫咪正在巡检节点状态...">
        <el-table-column prop="name" min-width="180">
          <template #header>
            <span class="neko-table-header">节点名称</span>
          </template>
        </el-table-column>
        <el-table-column prop="baseUrl" min-width="260" show-overflow-tooltip>
          <template #header>
            <span class="neko-table-header">地址</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" min-width="220" show-overflow-tooltip>
          <template #header>
            <span class="neko-table-header">说明</span>
          </template>
        </el-table-column>
        <el-table-column width="100" align="center">
          <template #header>
            <span class="neko-table-header">状态</span>
          </template>
          <template #default="{ row }">
            <el-tag :type="row.enabled === 1 ? 'success' : 'info'">
              {{ row.enabled === 1 ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column min-width="300" align="center">
          <template #header>
            <span class="neko-table-header is-action">操作</span>
          </template>
          <template #default="{ row }">
            <el-button size="small" @click="handlePing(row)">探测</el-button>
            <el-button size="small" @click="showProjects(row)">远程项目</el-button>
            <el-button size="small" type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
        <template #empty>
          <NekoEmptyState title="还没有远程节点" description="先登记一个节点，猫咪才有巡逻路线，可以帮你探测连通性和读取远程项目。" />
        </template>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑节点' : '添加节点'" width="560" :close-on-click-modal="false" @close="resetForm">
      <el-alert class="dialog-neko-tip" :closable="false" type="info" show-icon title="猫咪提醒：节点地址填写服务根地址，共享令牌填写对端节点当前的 shared_token，节点名称会从远端自动读取。" />
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="节点地址" prop="baseUrl">
          <el-input v-model="form.baseUrl" placeholder="例如：http://10.0.0.12:8888，不要填写 /static 或页面路径" />
        </el-form-item>
        <el-form-item label="共享令牌" prop="apiToken">
          <el-input v-model="form.apiToken" placeholder="填写远程节点 node_config.shared_token；旧版本双空节点可留空" show-password />
        </el-form-item>
        <el-form-item label="远端名称">
          <div class="remote-name-row">
            <el-input :model-value="resolvedNodeInfo.name || ''" readonly placeholder="点击右侧按钮读取远端节点名称" />
            <el-button :loading="remoteInfoLoading" @click="resolveNodeInfo()">读取节点信息</el-button>
          </div>
          <div v-if="resolvedNodeInfo.system" class="remote-node-meta">
            <el-tag size="small" effect="plain">{{ formatSystemLabel(resolvedNodeInfo.system) }}</el-tag>
            <el-tag size="small" effect="plain" type="success">
              {{ resolvedNodeInfo.pathCaseSensitive ? '路径大小写敏感' : '路径大小写不敏感' }}
            </el-tag>
            <span class="remote-node-meta__hint">路径分隔符 {{ resolvedNodeInfo.pathSeparator || '/' }}</span>
          </div>
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="记录用途或环境信息" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.enabled" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="projectsDialogVisible" :title="`远程项目 - ${currentNode?.name || ''}`" width="720">
      <el-table :data="remoteProjects" stripe v-loading="projectsLoading" element-loading-text="猫咪正在远程节点翻找项目...">
        <el-table-column prop="name" label="项目名称" min-width="180" />
        <el-table-column prop="path" label="项目路径" min-width="360" show-overflow-tooltip />
        <template #empty>
          <NekoEmptyState title="远程节点还没带回项目" description="这个节点暂时没有可用项目，或者猫咪还没在那边翻到可访问的目录。" compact />
        </template>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import NekoPageHeader from '@/components/NekoPageHeader.vue'
import NekoEmptyState from '@/components/NekoEmptyState.vue'
import { formatRequestError } from '@/utils/request-errors'

const router = useRouter()

document.title = '节点管理'

const loading = ref(false)
const refreshing = ref(false)
const submitting = ref(false)
const projectsLoading = ref(false)
const remoteInfoLoading = ref(false)
const nodes = ref([])
const dialogVisible = ref(false)
const projectsDialogVisible = ref(false)
const remoteProjects = ref([])
const currentNode = ref(null)
const emptyResolvedNodeInfo = () => ({
  name: '',
  tokenConfigured: false,
  system: '',
  pathSeparator: '',
  pathCaseSensitive: false,
})

const resolvedNodeInfo = ref(emptyResolvedNodeInfo())
const formRef = ref()
const form = ref({
  id: null,
  baseUrl: '',
  apiToken: '',
  description: '',
  enabled: 1,
})

const rules = {
  baseUrl: [{ required: true, message: '请输入节点地址', trigger: 'blur' }],
}

function showRequestError(errorLike, fallback) {
  ElMessage.error(formatRequestError(errorLike, fallback))
}

async function fetchNodes() {
  loading.value = true
  try {
    const res = await fetch('/api/nodes')
    const data = await res.json()
    if (data.success) {
      nodes.value = data.data || []
    } else {
      showRequestError(data.message, '获取节点列表失败')
    }
  } catch (error) {
    showRequestError(error, '获取节点列表失败')
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  resetForm()
  dialogVisible.value = true
}

function openEditDialog(row) {
  form.value = {
    id: row.id,
    baseUrl: row.baseUrl,
    apiToken: row.apiToken,
    description: row.description,
    enabled: row.enabled,
  }
  resolvedNodeInfo.value = {
    ...emptyResolvedNodeInfo(),
    name: row.name,
    tokenConfigured: Boolean(row.apiToken),
  }
  dialogVisible.value = true
}

function resetForm() {
  form.value = { id: null, baseUrl: '', apiToken: '', description: '', enabled: 1 }
  resolvedNodeInfo.value = emptyResolvedNodeInfo()
  formRef.value?.resetFields()
}

function formatSystemLabel(system) {
  switch (String(system || '').toLowerCase()) {
    case 'windows':
      return 'Windows'
    case 'darwin':
      return 'macOS'
    case 'linux':
      return 'Linux'
    default:
      return system || '未知系统'
  }
}

async function resolveNodeInfo(showSuccess = true) {
  const baseUrl = String(form.value.baseUrl || '').trim()
  if (!baseUrl) {
    ElMessage.warning('请先填写节点地址')
    return null
  }
  remoteInfoLoading.value = true
  try {
    const res = await fetch('/api/nodes/resolve-info', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        baseUrl: baseUrl,
        apiToken: form.value.apiToken,
      }),
    })
    const data = await res.json()
    if (!data.success) {
      showRequestError(data.message, '读取节点信息失败')
      return null
    }
    resolvedNodeInfo.value = {
      ...emptyResolvedNodeInfo(),
      ...(data.data || {}),
    }
    if (showSuccess) {
      ElMessage.success(`已读取远端节点：${resolvedNodeInfo.value.name || '未命名节点'}`)
    }
    return resolvedNodeInfo.value
  } catch (error) {
    showRequestError(error, '读取节点信息失败')
    return null
  } finally {
    remoteInfoLoading.value = false
  }
}

async function submitForm() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  const nodeInfo = await resolveNodeInfo(false)
  if (!nodeInfo?.name) {
    return
  }

  submitting.value = true
  try {
    const isEdit = !!form.value.id
    const url = isEdit ? `/api/nodes/${form.value.id}` : '/api/nodes'
    const method = isEdit ? 'PUT' : 'POST'
    const payload = {
      baseUrl: form.value.baseUrl,
      apiToken: form.value.apiToken,
      description: form.value.description,
      enabled: form.value.enabled,
    }
    const res = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    const data = await res.json()
    if (data.success) {
      ElMessage.success(isEdit ? '节点更新成功' : '节点创建成功')
      dialogVisible.value = false
      await fetchNodes()
    } else {
      showRequestError(data.message, '保存节点失败')
    }
  } catch (error) {
    showRequestError(error, '保存节点失败')
  } finally {
    submitting.value = false
  }
}

async function refreshNodes() {
  refreshing.value = true
  try {
    const res = await fetch('/api/nodes/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    })
    const data = await res.json()
    if (!data.success) {
      showRequestError(data.message, '刷新节点失败')
      return
    }
    const checked = Number(data.data?.checked || 0)
    const updated = Number(data.data?.updated || 0)
    const failed = Array.isArray(data.data?.failed) ? data.data.failed : []
    if (failed.length > 0) {
      ElMessage.warning(`刷新完成，检查 ${checked} 个节点，更新 ${updated} 个，失败 ${failed.length} 个`)
    } else {
      ElMessage.success(`刷新完成，检查 ${checked} 个节点，更新 ${updated} 个`)
    }
    await fetchNodes()
  } catch (error) {
    showRequestError(error, '刷新节点失败')
  } finally {
    refreshing.value = false
  }
}

async function handlePing(row) {
  try {
    const res = await fetch(`/api/nodes/${row.id}/ping`)
    const data = await res.json()
    if (data.success) {
      ElMessage.success(`节点连通成功：${data.data?.name || row.name}`)
    } else {
      showRequestError(data.message, '节点探测失败')
    }
  } catch (error) {
    showRequestError(error, '节点探测失败')
  }
}

async function showProjects(row) {
  currentNode.value = row
  remoteProjects.value = []
  projectsDialogVisible.value = true
  projectsLoading.value = true
  try {
    const res = await fetch(`/api/nodes/${row.id}/projects`)
    const data = await res.json()
    if (data.success) {
      remoteProjects.value = data.data || []
    } else {
      showRequestError(data.message, '读取远程项目失败')
    }
  } catch (error) {
    showRequestError(error, '读取远程项目失败')
  } finally {
    projectsLoading.value = false
  }
}

async function handleDelete(row) {
  await ElMessageBox.confirm(`确定删除节点“${row.name}”吗？`, '提示', { type: 'warning' })
  try {
    const res = await fetch(`/api/nodes/${row.id}`, { method: 'DELETE' })
    const data = await res.json()
    if (data.success) {
      ElMessage.success('节点删除成功')
      await fetchNodes()
    } else {
      showRequestError(data.message, '删除节点失败')
    }
  } catch (error) {
    showRequestError(error, '删除节点失败')
  }
}

onMounted(fetchNodes)
</script>

<style scoped>
.nodes-container {
  height: 100vh;
  padding: 24px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 16px;
  background:
    radial-gradient(circle at top left, rgba(111, 196, 133, 0.09), transparent 28%),
    radial-gradient(circle at bottom right, rgba(242, 185, 126, 0.1), transparent 26%),
    var(--el-bg-color-page);
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-left,
.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-left {
  min-width: 0;
  flex: 1;
}

.neko-table-header {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 700;
}

.neko-table-header::after {
  content: '爪';
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: rgba(103, 194, 58, 0.12);
  color: #529d51;
  font-size: 10px;
}

.neko-table-header.is-action::after {
  content: '喵';
}

:global(html.dark) .neko-table-header::after {
  background: rgba(103, 194, 58, 0.18);
  color: #aed7a0;
}

.table-card {
  flex: 1;
}

.table-card :deep(.el-card__body) {
  height: 100%;
  padding: 0;
}

.table-card :deep(.el-table) {
  height: 100%;
}

.table-card :deep(.el-table__inner-wrapper) {
  min-height: 100%;
  background: linear-gradient(180deg, rgba(252, 246, 236, 0.36) 0%, rgba(255, 250, 244, 0.18) 100%);
}

.table-card :deep(.el-table th.el-table__cell) {
  box-shadow: inset 0 -1px 0 rgba(150, 195, 130, 0.12);
}

.table-card :deep(.el-table td.el-table__cell) {
  color: #5c4a3a;
}

.table-card :deep(.el-table__empty-block) {
  background: transparent;
}

.dialog-neko-tip {
  margin-bottom: 14px;
}

.remote-name-row {
  width: 100%;
  display: flex;
  gap: 10px;
}

.remote-name-row .el-input {
  flex: 1;
}

.remote-node-meta {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.remote-node-meta__hint {
  font-size: 12px;
  color: #8a735d;
}

.nodes-container :deep(.el-dialog) {
  border: 1px solid var(--neko-surface-border);
  background: var(--neko-surface-bg);
  box-shadow: var(--neko-surface-shadow);
}

.nodes-container :deep(.el-dialog__header) {
  border-bottom: 1px solid var(--neko-surface-divider);
  background: var(--neko-surface-header-bg);
}

.nodes-container :deep(.el-dialog__body),
.nodes-container :deep(.el-dialog__footer) {
  background: transparent;
}

:global(html.dark) .nodes-container {
  background:
    radial-gradient(circle at top left, rgba(60, 111, 75, 0.22), transparent 28%),
    radial-gradient(circle at bottom right, rgba(126, 88, 54, 0.16), transparent 26%),
    var(--el-bg-color-page);
}

:global(html.dark) .table-card :deep(.el-table__inner-wrapper) {
  background: linear-gradient(180deg, rgba(54, 49, 41, 0.3) 0%, rgba(30, 32, 38, 0.22) 100%);
}

:global(html.dark) .table-card :deep(.el-table td.el-table__cell) {
  color: #e1cfbb;
}

</style>