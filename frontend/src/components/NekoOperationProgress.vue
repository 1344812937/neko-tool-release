<template>
  <el-card class="operation-progress-card neko-surface">
    <div class="operation-progress-header">
      <div class="operation-progress-copy">
        <strong>{{ title }}</strong>
        <span>{{ statusText }}</span>
      </div>
      <el-tag :type="tagType">{{ tagLabel }}</el-tag>
    </div>

    <el-steps :active="Math.max(activeStep, 0)" finish-status="success" process-status="process" simple>
      <el-step v-for="(step, index) in steps" :key="step.key || step.title || index" :title="step.title" />
    </el-steps>

    <div class="operation-progress-meta">
      <div v-if="showPercent" class="operation-progress-bar">
        <el-progress :percentage="safePercent" :status="progressStatus" :stroke-width="12" />
      </div>
      <div class="operation-progress-details">
        <span v-if="summaryText">{{ summaryText }}</span>
        <span v-if="currentPath" class="operation-progress-path" :title="currentPath">当前处理：{{ currentPath }}</span>
        <span v-if="message">{{ message }}</span>
      </div>
    </div>
  </el-card>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  title: { type: String, default: '' },
  steps: { type: Array, default: () => [] },
  activeStep: { type: Number, default: 0 },
  status: { type: String, default: 'running' },
  percent: { type: Number, default: 0 },
  processed: { type: Number, default: 0 },
  total: { type: Number, default: 0 },
  currentPath: { type: String, default: '' },
  message: { type: String, default: '' },
})

const safePercent = computed(() => {
  const value = Number(props.percent || 0)
  if (Number.isNaN(value)) {
    return 0
  }
  return Math.min(100, Math.max(0, Math.round(value)))
})

const tagType = computed(() => {
  if (props.status === 'success') {
    return 'success'
  }
  if (props.status === 'error') {
    return 'danger'
  }
  return 'warning'
})

const tagLabel = computed(() => {
  if (props.status === 'success') {
    return '已完成'
  }
  if (props.status === 'error') {
    return '有异常'
  }
  return '进行中'
})

const statusText = computed(() => {
  if (props.status === 'success') {
    return '本次操作已经完成。'
  }
  if (props.status === 'error') {
    return '本次操作出现异常，请查看提示信息。'
  }
  return '正在按步骤推进当前操作。'
})

const showPercent = computed(() => props.total > 0 || safePercent.value > 0)

const summaryText = computed(() => {
  if (props.total > 0) {
    return `已处理 ${props.processed} / ${props.total}`
  }
  return ''
})

const progressStatus = computed(() => (props.status === 'error' ? 'exception' : props.status === 'success' ? 'success' : ''))
</script>

<style scoped>
.operation-progress-card :deep(.el-card__body) {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.operation-progress-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.operation-progress-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.operation-progress-copy strong {
  font-size: 15px;
  color: var(--el-text-color-primary);
}

.operation-progress-copy span,
.operation-progress-details {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.operation-progress-meta {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.operation-progress-details {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
}

.operation-progress-path {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>