<template>
  <div ref="shellRef" class="finder-columns-shell" tabindex="0" @keydown="handleKeydown">
    <div v-if="columns.length === 0 || columns[0].items.length === 0" class="finder-empty">
      <NekoEmptyState title="猫咪还没翻到内容" :description="emptyText" compact />
    </div>
    <div v-else class="finder-stage">
      <div class="finder-stage-toolbar">
        <span class="finder-stage-title">列视图</span>
        <span class="finder-stage-hint">支持触摸板双指横向滑动</span>
      </div>
      <div ref="scrollAreaRef" class="finder-scroll-area" @wheel="handleWheel">
        <div class="finder-columns">
          <section
            v-for="(column, index) in columns"
            :key="column.key"
            :ref="(element) => setColumnRef(element, index)"
            class="finder-column"
            :class="{ 'is-current-column': isCurrentColumn(column) }"
          >
            <header class="finder-column-header">
              <span class="finder-column-title">{{ column.title }}</span>
              <span class="finder-column-count">{{ column.items.length }}</span>
            </header>
            <div class="finder-column-body">
          <button
            v-for="item in column.items"
            :key="item.path"
            :ref="(element) => setRowRef(element, item.path)"
            type="button"
            class="finder-row"
            :class="{ 'is-active': activePath === item.path, 'is-deleted': item.deleted, 'is-loading': item.loading }"
            @click="handleRowClick(item)"
          >
            <span class="finder-row-main">
              <el-icon class="finder-icon">
                <Folder v-if="item.entryType === 'directory'" />
                <Document v-else />
              </el-icon>
                <span class="finder-name" :title="item.name">{{ item.name }}</span>
                <span v-if="item.entryType !== 'directory' && item.size" class="finder-size">{{ formatSize(item.size) }}</span>
            </span>
            <span class="finder-row-meta">
              <el-tag v-if="item.loading" size="small" type="info">加载中</el-tag>
              <el-tag v-if="item.deleted" size="small" type="danger">已删除</el-tag>
              <el-tag v-if="item.status && item.status !== 'context'" size="small" :type="statusTypeMap[item.status] || 'info'">
                {{ statusLabelMap[item.status] || item.status }}
              </el-tag>
              <el-icon v-if="item.entryType === 'directory'" class="finder-chevron"><ArrowRightBold /></el-icon>
            </span>
          </button>
        </div>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { ArrowRightBold, Document, Folder } from '@element-plus/icons-vue'
import NekoEmptyState from '@/components/NekoEmptyState.vue'
import { buildFinderColumns } from '@/utils/finder'

const props = defineProps({
  nodes: {
    type: Array,
    default: () => [],
  },
  selectedPath: {
    type: String,
    default: '',
  },
  statusLabelMap: {
    type: Object,
    default: () => ({}),
  },
  statusTypeMap: {
    type: Object,
    default: () => ({}),
  },
  emptyText: {
    type: String,
    default: '暂无内容',
  },
})

const emit = defineEmits(['select', 'expand'])

const columns = computed(() => buildFinderColumns(props.nodes, props.selectedPath))
const shellRef = ref(null)
const scrollAreaRef = ref(null)
const columnRefs = ref([])
const rowRefs = ref({})

function matchesSelectedPath(item, path) {
  if (!item || !path) {
    return false
  }
  if (item.path === path) {
    return true
  }
  return Array.isArray(item.aliases) && item.aliases.includes(path)
}

const selectedItem = computed(() => {
  for (const column of columns.value) {
    const item = column.items?.find((entry) => matchesSelectedPath(entry, props.selectedPath))
    if (item) {
      return item
    }
  }
  return null
})

const activePath = computed(() => selectedItem.value?.path || props.selectedPath)

const currentColumnIndex = computed(() => {
  if (!activePath.value) {
    return 0
  }
  const index = columns.value.findIndex((column) => column.items?.some((item) => item.path === activePath.value))
  return index >= 0 ? index : 0
})

function setColumnRef(element, index) {
  if (!element) {
    return
  }
  columnRefs.value[index] = element
}

function setRowRef(element, path) {
  if (!path) {
    return
  }
  if (!element) {
    delete rowRefs.value[path]
    return
  }
  rowRefs.value[path] = element
}

function handleRowClick(item) {
  shellRef.value?.focus({ preventScroll: true })
  emit('select', item)
}

function isCurrentColumn(column) {
  if (!activePath.value) {
    return column.key === 'root'
  }
  return column.items?.some((item) => item.path === activePath.value)
}

function scrollToActiveColumn() {
  nextTick(() => {
    const scrollArea = scrollAreaRef.value
    if (!scrollArea) {
      return
    }
    const activeColumn = columnRefs.value.find((column) => column?.classList.contains('is-current-column')) || columnRefs.value[columnRefs.value.length - 1]
    activeColumn?.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
    const activeRow = rowRefs.value[activePath.value]
    activeRow?.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' })
    activeRow?.focus({ preventScroll: true })
  })
}

function selectItem(item) {
  if (item) {
    emit('select', item)
  }
}

function findCurrentColumnItems() {
  return columns.value[currentColumnIndex.value]?.items || []
}

function findSelectedIndex() {
  return findCurrentColumnItems().findIndex((item) => item.path === props.selectedPath)
}

function handleKeydown(event) {
  if (!columns.value.length) {
    return
  }
  const currentItems = findCurrentColumnItems()
  const currentIndex = findSelectedIndex()
  const currentItem = selectedItem.value || currentItems[0]

  switch (event.key) {
    case 'ArrowDown': {
      event.preventDefault()
      const nextIndex = currentIndex >= 0 ? Math.min(currentIndex + 1, currentItems.length - 1) : 0
      selectItem(currentItems[nextIndex])
      return
    }
    case 'ArrowUp': {
      event.preventDefault()
      const prevIndex = currentIndex >= 0 ? Math.max(currentIndex - 1, 0) : 0
      selectItem(currentItems[prevIndex])
      return
    }
    case 'ArrowRight': {
      event.preventDefault()
      if (!currentItem) {
        return
      }
      const nextColumn = columns.value[currentColumnIndex.value + 1]
      if (currentItem.entryType === 'directory' && currentItem.hasChildren && !currentItem.childrenLoaded) {
        emit('expand', { item: currentItem, focusFirstChild: true })
        return
      }
      if (currentItem.entryType === 'directory' && nextColumn?.items?.length) {
        selectItem(nextColumn.items[0])
      }
      return
    }
    case 'ArrowLeft': {
      event.preventDefault()
      if (!currentItem?.parentPath) {
        return
      }
      const parentItem = columns.value
        .flatMap((column) => column.items || [])
        .find((item) => item.path === currentItem.parentPath)
      selectItem(parentItem)
      return
    }
    case 'Home': {
      event.preventDefault()
      selectItem(currentItems[0])
      return
    }
    case 'End': {
      event.preventDefault()
      selectItem(currentItems[currentItems.length - 1])
      return
    }
    default:
      return
  }
}

function handleWheel(event) {
  const scrollArea = scrollAreaRef.value
  if (!scrollArea || scrollArea.scrollWidth <= scrollArea.clientWidth) {
    return
  }
  if (event.shiftKey) {
    event.preventDefault()
    scrollArea.scrollLeft += event.deltaY
  }
}

function formatSize(size) {
  if (!size) {
    return ''
  }
  if (size < 1024) {
    return `${size} B`
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`
  }
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

watch(columns, () => {
  columnRefs.value = []
  rowRefs.value = {}
  scrollToActiveColumn()
}, { deep: true })

watch(() => props.selectedPath, () => {
  scrollToActiveColumn()
})
</script>

<style scoped>
.finder-columns-shell {
  height: 100%;
  min-height: 0;
  overflow: hidden;
  outline: none;
}

.finder-columns-shell:focus-visible {
  box-shadow: inset 0 0 0 2px color-mix(in srgb, var(--el-color-primary) 40%, transparent);
  border-radius: 18px;
}

.finder-empty {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.finder-stage {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.finder-stage-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 4px;
}

.finder-stage-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.finder-stage-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.finder-scroll-area {
  flex: 1;
  min-height: 0;
  box-sizing: border-box;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-gutter: stable both-edges;
  scroll-behavior: smooth;
  overscroll-behavior-x: contain;
  touch-action: pan-x pan-y;
  -webkit-overflow-scrolling: touch;
}

.finder-scroll-area::-webkit-scrollbar {
  height: 11px;
}

.finder-scroll-area::-webkit-scrollbar-track {
  background: color-mix(in srgb, var(--el-fill-color-light) 70%, transparent);
  border-radius: 999px;
}

.finder-scroll-area::-webkit-scrollbar-thumb {
  background: color-mix(in srgb, var(--el-color-primary-light-5) 68%, var(--el-fill-color-darker));
  border-radius: 999px;
  border: 2px solid transparent;
}

.finder-columns {
  height: 100%;
  width: max-content;
  min-width: 100%;
  display: flex;
  border: 1px solid var(--el-border-color-light);
  border-radius: 18px;
  background:
    linear-gradient(
      180deg,
      color-mix(in srgb, var(--el-color-primary-light-9) 38%, var(--el-bg-color-overlay)),
      color-mix(in srgb, var(--el-fill-color) 86%, var(--el-bg-color-overlay))
    );
  backdrop-filter: blur(18px);
  box-shadow:
    inset 0 1px 0 color-mix(in srgb, white 18%, transparent),
    0 12px 28px color-mix(in srgb, var(--el-color-primary) 8%, transparent);
}

.finder-column {
  flex: 0 0 270px;
  min-width: 270px;
  max-width: 270px;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--el-border-color-lighter);
}

.finder-column.is-current-column {
  background: linear-gradient(
    180deg,
    color-mix(in srgb, var(--el-color-primary-light-8) 46%, transparent),
    color-mix(in srgb, var(--el-color-primary-light-9) 24%, transparent)
  );
}

.finder-column:last-child {
  border-right: none;
}

.finder-column-header {
  position: sticky;
  top: 0;
  z-index: 1;
  padding: 12px 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  background: color-mix(in srgb, var(--el-bg-color-overlay) 82%, var(--el-color-primary-light-9));
  border-bottom: 1px solid var(--el-border-color-lighter);
  backdrop-filter: blur(12px);
}

.finder-column-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.finder-column-count {
  flex-shrink: 0;
  min-width: 24px;
  padding: 1px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--el-fill-color-dark) 10%, transparent);
  color: var(--el-text-color-secondary);
  text-align: center;
  font-size: 12px;
}

.finder-column-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px 8px 18px;
  scrollbar-width: thin;
}

.finder-row {
  width: 100%;
  border: none;
  background: color-mix(in srgb, var(--el-fill-color) 82%, transparent);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 9px 12px;
  margin-bottom: 4px;
  border-radius: 12px;
  cursor: pointer;
  text-align: left;
  transition: background-color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease;
  outline: none;
}

.finder-row.is-deleted {
  opacity: 0.7;
  background: color-mix(in srgb, var(--el-color-danger-light-9) 40%, transparent);
}

.finder-row.is-loading {
  background: color-mix(in srgb, var(--el-color-info-light-9) 42%, transparent);
}

.finder-row:hover {
  background: color-mix(in srgb, var(--el-color-primary-light-8) 54%, var(--el-bg-color-overlay));
  transform: translateY(-1px);
}

.finder-row.is-active {
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--el-color-primary-light-5) 62%, var(--el-bg-color-overlay)),
    color-mix(in srgb, var(--el-color-primary-light-8) 82%, var(--el-bg-color-overlay))
  );
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 36%, transparent), 0 8px 20px rgba(64, 158, 255, 0.08);
}

.finder-row:focus-visible {
  box-shadow: inset 0 0 0 2px color-mix(in srgb, var(--el-color-primary) 42%, transparent), 0 8px 20px rgba(64, 158, 255, 0.1);
}

.finder-row-main,
.finder-row-meta {
  display: flex;
  align-items: center;
  gap: 10px;
}

.finder-row-main {
  min-width: 0;
  flex: 1;
}

.finder-row-meta {
  flex-shrink: 0;
}

.finder-icon {
  color: var(--el-color-primary);
  flex-shrink: 0;
}

.finder-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--el-text-color-primary);
}

.finder-size {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.finder-chevron {
  color: var(--el-text-color-placeholder);
}

@media (max-width: 768px) {
  .finder-column {
    flex-basis: 232px;
    min-width: 232px;
    max-width: 232px;
  }

  .finder-stage-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>