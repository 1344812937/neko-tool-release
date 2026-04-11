<template>
  <div class="projects-container">
    <div class="header">
      <div class="header-left">
        <NekoPageHeader
          title="项目管理"
          description="项目清单、目录选择和路径边界都在这里统一整理。"
          tone="blue"
          @back="router.push('/')"
        />
      </div>
      <div class="header-right">
        <el-button :icon="Refresh" @click="fetchProjects" :loading="loading">刷新</el-button>
        <el-button type="primary" @click="openAddDialog">添加项目</el-button>
      </div>
    </div>

    <el-card ref="projectsTableCardRef" class="table-card neko-surface">
      <el-table :data="projects" stripe :row-class-name="getProjectRowClassName" v-loading="loading" element-loading-text="猫咪正在清点项目清单...">
        <el-table-column prop="name" min-width="180">
          <template #header>
            <span class="neko-table-header">项目名称</span>
          </template>
          <template #default="{ row }">
            <div class="project-name-cell">
              <span>{{ row.name }}</span>
              <el-tag v-if="isRestoredProject(row)" size="small" type="success">已恢复</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="code" min-width="180">
          <template #header>
            <span class="neko-table-header">项目编码</span>
          </template>
        </el-table-column>
        <el-table-column prop="path" min-width="280" show-overflow-tooltip>
          <template #header>
            <span class="neko-table-header">目录路径</span>
          </template>
        </el-table-column>
        <el-table-column width="230" align="center">
          <template #header>
            <span class="neko-table-header is-action">操作</span>
          </template>
          <template #default="{ row }">
            <el-button size="small" @click="openBrowser(row)">浏览</el-button>
            <el-button type="primary" size="small" @click="openEditDialog(row)">编辑</el-button>
            <el-button type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
        <template #empty>
          <NekoEmptyState title="还没有项目记录" description="先添加一个目录，猫咪就会守在这里，帮你继续浏览、比较和同步。" />
        </template>
      </el-table>
    </el-card>

    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="pageNo"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="fetchProjects"
        @current-change="fetchProjects"
      />
    </div>

    <!-- 添加项目对话框 -->
    <el-dialog v-model="addDialogVisible" title="添加项目" width="500" :close-on-click-modal="false" @close="resetAddForm">
      <el-alert class="dialog-neko-tip" :closable="false" type="info" show-icon title="猫咪提醒：目录一旦选定，就会作为后续浏览和比较的项目根目录。" />
      <el-form :model="addForm" :rules="addRules" ref="addFormRef" label-width="80px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="addForm.name" placeholder="请输入项目名称" @input="handleAddNameInput" />
        </el-form-item>
        <el-form-item label="编码" prop="code">
          <el-input v-model="addForm.code" placeholder="默认与项目名称一致" @input="handleAddCodeInput" />
        </el-form-item>
        <el-form-item label="目录" prop="path">
          <el-input v-model="addForm.path" :placeholder="directoryPlaceholder" readonly>
            <template #append>
              <el-button :loading="selecting" :disabled="!capabilities.nativeDirectoryPickerEnabled && !capabilities.browserDirectoryPickerEnabled" @click="selectDirectory">
                {{ selectButtonLabel }}
              </el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <div class="capability-tip">
            <el-tag size="small" :type="capabilities.accessMode === 'local' ? 'success' : 'warning'">
              {{ capabilities.accessMode === 'local' ? '本机访问' : '远程访问' }}
            </el-tag>
            <span>{{ capabilities.message }}</span>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleAdd">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="browserPickerVisible" title="选择服务器目录" width="640" :close-on-click-modal="false">
      <div class="directory-picker-body">
        <div class="directory-toolbar">
          <el-alert :closable="false" type="info" show-icon :title="capabilities.message" />
        </div>
        <el-tree
          v-if="capabilities.browserDirectoryPickerEnabled"
          :key="directoryTreeKey"
          ref="directoryTreeRef"
          node-key="path"
          lazy
          highlight-current
          :props="treeProps"
          :load="loadDirectoryNode"
          @node-click="handleDirectoryNodeClick"
        />
        <NekoEmptyState v-else title="猫咪找不到可浏览目录" description="当前服务器还没配置允许浏览的根目录，先补上白名单，猫咪才能继续带路。" compact />
      </div>
      <template #footer>
        <div class="directory-picker-footer">
          <div class="selected-path">{{ selectedDirectoryPath || '尚未选择目录' }}</div>
          <div>
            <el-button @click="browserPickerVisible = false">取消</el-button>
            <el-button type="primary" :disabled="!selectedDirectoryPath" @click="confirmDirectorySelection">使用此目录</el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑项目对话框 -->
    <el-dialog v-model="editDialogVisible" title="编辑项目" width="500" :close-on-click-modal="false" @close="resetEditForm">
      <el-alert class="dialog-neko-tip" :closable="false" type="success" show-icon title="猫咪提醒：这里只会修改项目名称，不会改动实际目录路径。" />
      <el-form :model="editForm" :rules="editRules" ref="editFormRef" label-width="80px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="editForm.name" placeholder="请输入项目名称" />
        </el-form-item>
        <el-form-item label="编码">
          <el-input :model-value="editForm.code" disabled />
        </el-form-item>
        <el-form-item label="目录">
          <el-input :model-value="editForm.path" disabled />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleEdit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElLoading, ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import NekoPageHeader from '@/components/NekoPageHeader.vue'
import NekoEmptyState from '@/components/NekoEmptyState.vue'

const router = useRouter()

document.title = '项目管理'

const projects = ref([])
const loading = ref(false)
const submitting = ref(false)
const selecting = ref(false)
const capabilitiesLoading = ref(false)
const restoredProjectId = ref(null)
const projectsTableCardRef = ref()
let restoredHighlightTimer = null

const pageNo = ref(1)
const pageSize = ref(10)
const total = ref(0)

const capabilities = ref({
  accessMode: 'remote',
  nativeDirectoryPickerEnabled: false,
  browserDirectoryPickerEnabled: false,
  allowedRoots: [],
  message: '正在加载目录访问能力...',
})
const browserPickerVisible = ref(false)
const selectedDirectoryPath = ref('')
const directoryTreeKey = ref(0)
const directoryTreeRef = ref()
const treeProps = {
  label: 'name',
  children: 'children',
  isLeaf: (data) => !data.hasChildren,
}

// 添加表单
const addDialogVisible = ref(false)
const addFormRef = ref()
const addForm = ref({ name: '', code: '', path: '' })
const addCodeEdited = ref(false)
const validateAddProjectCode = (_rule, value, callback) => {
  const normalizedCode = String(value || '').trim()
  if (!normalizedCode) {
    callback(new Error('请输入项目编码'))
    return
  }
  const duplicated = projects.value.some((project) => String(project.code || '').trim() === normalizedCode)
  if (duplicated) {
    callback(new Error('项目编码已存在'))
    return
  }
  callback()
}
const addRules = {
  name: [{ required: true, message: '请输入项目名称', trigger: 'blur' }],
  code: [{ validator: validateAddProjectCode, trigger: ['blur', 'change'] }],
  path: [{ required: true, message: '请选择目录路径', trigger: 'change' }],
}

// 编辑表单
const editDialogVisible = ref(false)
const editFormRef = ref()
const editForm = ref({ id: null, name: '', code: '', path: '' })
const editRules = {
  name: [{ required: true, message: '请输入项目名称', trigger: 'blur' }],
}

const selectButtonLabel = computed(() => {
  if (capabilities.value.nativeDirectoryPickerEnabled) {
    return '系统选择'
  }
  if (capabilities.value.browserDirectoryPickerEnabled) {
    return '浏览目录'
  }
  return '不可用'
})

const directoryPlaceholder = computed(() => {
  if (capabilities.value.nativeDirectoryPickerEnabled) {
    return '当前为本机访问，将调用系统目录选择器'
  }
  if (capabilities.value.browserDirectoryPickerEnabled) {
    return '当前为远程访问，请浏览服务器目录'
  }
  return '当前未配置允许创建项目的根目录'
})

async function fetchProjects() {
  loading.value = true
  try {
    const res = await fetch(`/api/projects?pageNo=${pageNo.value}&pageSize=${pageSize.value}`)
    const data = await res.json()
    if (data.success) {
      const page = data.data || {}
      projects.value = page.result || []
      total.value = page.total || 0
      pageNo.value = page.pageNo || 1
    } else {
      ElMessage.error(data.message || '获取项目列表失败')
    }
  } catch {
    ElMessage.error('网络请求失败')
  } finally {
    loading.value = false
  }
}

function getProjectRowClassName({ row }) {
  return restoredProjectId.value && row?.id === restoredProjectId.value ? 'restored-project-row' : ''
}

function isRestoredProject(row) {
  return Boolean(restoredProjectId.value && row?.id === restoredProjectId.value)
}

function markRestoredProject(projectId) {
  restoredProjectId.value = projectId || null
  if (restoredHighlightTimer) {
    window.clearTimeout(restoredHighlightTimer)
  }
  if (!projectId) {
    return
  }
  restoredHighlightTimer = window.setTimeout(() => {
    restoredProjectId.value = null
  }, 8000)
}

async function scrollToRestoredProject() {
  if (!restoredProjectId.value) {
    return
  }
  await nextTick()
  const rowElement = projectsTableCardRef.value?.$el?.querySelector('.restored-project-row')
  if (rowElement?.scrollIntoView) {
    rowElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
}

async function fetchCapabilities() {
  capabilitiesLoading.value = true
  try {
    const res = await fetch('/api/project-access/capabilities')
    const data = await res.json()
    if (data.success && data.data) {
      capabilities.value = data.data
    } else {
      ElMessage.error(data.message || '获取目录访问能力失败')
    }
  } catch {
    ElMessage.error('获取目录访问能力失败')
  } finally {
    capabilitiesLoading.value = false
  }
}

async function selectDirectory() {
  if (capabilitiesLoading.value) {
    return
  }
  selecting.value = true
  try {
    if (capabilities.value.nativeDirectoryPickerEnabled) {
      await selectNativeDirectory()
      return
    }
    if (capabilities.value.browserDirectoryPickerEnabled) {
      openBrowserPicker()
      return
    }
    ElMessage.warning(capabilities.value.message || '当前没有可用的目录选择方式')
  } finally {
    selecting.value = false
  }
}

async function selectNativeDirectory() {
  try {
    const res = await fetch('/api/select-directory')
    const data = await res.json()
    if (data.success) {
      applySelectedPath(data.data)
    } else if (data.code !== 400) {
      ElMessage.error(data.message || '选择目录失败')
    }
  } catch {
    ElMessage.error('网络请求失败')
  }
}

function openAddDialog() {
  fetchCapabilities()
  addDialogVisible.value = true
}

function openBrowserPicker() {
  selectedDirectoryPath.value = addForm.value.path || ''
  directoryTreeKey.value += 1
  browserPickerVisible.value = true
}

function applySelectedPath(path) {
  addForm.value.path = path
  const leafName = extractPathLeaf(path)
  if (!addForm.value.name) {
    addForm.value.name = leafName
  }
  if (!addCodeEdited.value) {
    addForm.value.code = leafName || addForm.value.name
  }
}

function extractPathLeaf(path) {
  const normalizedPath = String(path || '').replace(/[/\\]+$/, '')
  const lastSep = Math.max(normalizedPath.lastIndexOf('/'), normalizedPath.lastIndexOf('\\'))
  return lastSep >= 0 ? normalizedPath.substring(lastSep + 1) : normalizedPath
}

function handleAddNameInput(value) {
  if (!addCodeEdited.value) {
    addForm.value.code = value
  }
}

function handleAddCodeInput() {
  addCodeEdited.value = true
}

async function loadDirectoryNode(node, resolve) {
  const currentPath = node.level === 0 ? '' : node.data.path
  try {
    const query = currentPath ? `?path=${encodeURIComponent(currentPath)}` : ''
    const res = await fetch(`/api/project-access/directories${query}`)
    const data = await res.json()
    if (data.success) {
      resolve(data.data || [])
    } else {
      ElMessage.error(data.message || '读取目录失败')
      resolve([])
    }
  } catch {
    ElMessage.error('读取目录失败')
    resolve([])
  }
}

function handleDirectoryNodeClick(data) {
  selectedDirectoryPath.value = data.path
}

function confirmDirectorySelection() {
  if (!selectedDirectoryPath.value) {
    ElMessage.warning('请先选择一个目录')
    return
  }
  applySelectedPath(selectedDirectoryPath.value)
  browserPickerVisible.value = false
}

function resetAddForm() {
  addForm.value = { name: '', code: '', path: '' }
  addCodeEdited.value = false
  selectedDirectoryPath.value = ''
  addFormRef.value?.resetFields()
}

async function handleAdd() {
  addForm.value.name = String(addForm.value.name || '').trim()
  addForm.value.code = String(addForm.value.code || '').trim()
  const valid = await addFormRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  const loadingInstance = ElLoading.service({
    lock: true,
    text: '猫咪正在创建项目；若命中历史删除项目，会自动恢复并重新比对日志...',
    background: 'rgba(24, 33, 43, 0.28)',
  })
  try {
    const res = await fetch('/api/projects', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(addForm.value),
    })
    const data = await res.json()
    if (data.success) {
      if (data.data?.restored) {
        markRestoredProject(data.data?.project?.id)
        ElNotification({
          title: '项目已恢复',
          type: 'success',
          duration: 6000,
          message: '检测到同编码同路径的逻辑删除项目，已完成恢复、重扫，并重新比对历史文件日志。点击通知可直接进入项目浏览。',
          onClick: () => {
            if (data.data?.project?.id) {
              router.push(`/projects/${data.data.project.id}/browser`)
            }
          },
        })
      } else {
        markRestoredProject(null)
        ElMessage.success('添加成功')
      }
      addDialogVisible.value = false
      await fetchProjects()
      if (data.data?.restored) {
        await scrollToRestoredProject()
      }
    } else {
      ElMessage.error(data.message || '添加失败')
    }
  } catch {
    ElMessage.error('网络请求失败')
  } finally {
    loadingInstance.close()
    submitting.value = false
  }
}

function openEditDialog(row) {
  editForm.value = { id: row.id, name: row.name, code: row.code, path: row.path }
  editDialogVisible.value = true
}

function openBrowser(row) {
  router.push(`/projects/${row.id}/browser`)
}

function resetEditForm() {
  editForm.value = { id: null, name: '', code: '', path: '' }
  editFormRef.value?.resetFields()
}

async function handleEdit() {
  const valid = await editFormRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    const res = await fetch(`/api/projects/${editForm.value.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: editForm.value.name }),
    })
    const data = await res.json()
    if (data.success) {
      ElMessage.success('修改成功')
      editDialogVisible.value = false
      await fetchProjects()
    } else {
      ElMessage.error(data.message || '修改失败')
    }
  } catch {
    ElMessage.error('网络请求失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row) {
  await ElMessageBox.confirm(
    `确定删除项目“${row.name}”吗？\n项目编码：${row.code || '-'}\n目录路径：${row.path || '-'}\n\n当前删除为逻辑删除，后续若新增同编码且同路径的项目，会自动恢复这条记录。`,
    '逻辑删除确认',
    { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' },
  )
  try {
    const res = await fetch(`/api/projects/${row.id}`, { method: 'DELETE' })
    const data = await res.json()
    if (data.success) {
      ElMessage.success('删除成功')
      await fetchProjects()
    } else {
      ElMessage.error(data.message || '删除失败')
    }
  } catch {
    ElMessage.error('网络请求失败')
  }
}

onMounted(() => {
  fetchProjects()
  fetchCapabilities()
})

onBeforeUnmount(() => {
  if (restoredHighlightTimer) {
    window.clearTimeout(restoredHighlightTimer)
  }
})
</script>

<style scoped>
.projects-container {
  height: 100vh;
  padding: 24px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  background:
    radial-gradient(circle at top left, rgba(82, 155, 228, 0.08), transparent 30%),
    radial-gradient(circle at bottom right, rgba(239, 188, 125, 0.1), transparent 28%),
    var(--el-bg-color-page);
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  flex: 1;
}

.header-title {
  display: flex;
  flex-direction: column;
  gap: 8px;
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
  background: rgba(84, 150, 215, 0.1);
  color: #5b88bb;
  font-size: 10px;
}

.neko-table-header.is-action::after {
  content: '喵';
}

:global(html.dark) .neko-table-header::after {
  background: rgba(84, 150, 215, 0.16);
  color: #a8c7e8;
}

.table-card {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.table-card :deep(.el-table .restored-project-row > td) {
  background: color-mix(in srgb, var(--el-color-success-light-8) 75%, transparent) !important;
}

.project-name-cell {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

:global(html.dark) .projects-container {
  background:
    radial-gradient(circle at top left, rgba(64, 98, 131, 0.18), transparent 30%),
    radial-gradient(circle at bottom right, rgba(134, 92, 53, 0.16), transparent 28%),
    var(--el-bg-color-page);
}

.table-card :deep(.el-card__body) {
  flex: 1;
  overflow: auto;
  padding: 0;
}

.dialog-neko-tip {
  margin-bottom: 14px;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
  flex-shrink: 0;
}

.capability-tip {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.directory-picker-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 360px;
}

.directory-toolbar {
  flex-shrink: 0;
}

.directory-picker-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.selected-path {
  flex: 1;
  color: var(--el-text-color-secondary);
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
