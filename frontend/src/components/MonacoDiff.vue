<template>
  <div ref="containerRef" class="monaco-diff-container" :style="containerStyle"></div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import 'monaco-editor/esm/vs/basic-languages/css/css.contribution'
import 'monaco-editor/esm/vs/basic-languages/go/go.contribution'
import 'monaco-editor/esm/vs/basic-languages/html/html.contribution'
import 'monaco-editor/esm/vs/basic-languages/java/java.contribution'
import 'monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution'
import 'monaco-editor/esm/vs/basic-languages/markdown/markdown.contribution'
import 'monaco-editor/esm/vs/basic-languages/python/python.contribution'
import 'monaco-editor/esm/vs/basic-languages/shell/shell.contribution'
import 'monaco-editor/esm/vs/basic-languages/sql/sql.contribution'
import 'monaco-editor/esm/vs/basic-languages/typescript/typescript.contribution'
import 'monaco-editor/esm/vs/basic-languages/xml/xml.contribution'
import 'monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution'
import 'monaco-editor/esm/vs/language/json/monaco.contribution'

if (!window.MonacoEnvironment) {
  window.MonacoEnvironment = {
    getWorker() {
      return new editorWorker()
    },
  }
}

const props = defineProps({
  leftContent: {
    type: String,
    default: '',
  },
  rightContent: {
    type: String,
    default: '',
  },
  language: {
    type: String,
    default: 'plaintext',
  },
  height: {
    type: [String, Number],
    default: 520,
  },
})

const containerRef = ref(null)
const containerStyle = computed(() => ({
  height: typeof props.height === 'number' ? `${props.height}px` : props.height,
}))
let diffEditor
let leftModel
let rightModel

function ensureModels() {
  if (!leftModel) {
    leftModel = monaco.editor.createModel(props.leftContent, props.language)
  }
  if (!rightModel) {
    rightModel = monaco.editor.createModel(props.rightContent, props.language)
  }
}

function syncModels() {
  ensureModels()
  if (leftModel.getValue() !== props.leftContent) {
    leftModel.setValue(props.leftContent)
  }
  if (rightModel.getValue() !== props.rightContent) {
    rightModel.setValue(props.rightContent)
  }
  monaco.editor.setModelLanguage(leftModel, props.language)
  monaco.editor.setModelLanguage(rightModel, props.language)
}

onMounted(() => {
  ensureModels()
  diffEditor = monaco.editor.createDiffEditor(containerRef.value, {
    automaticLayout: true,
    renderSideBySide: true,
    readOnly: true,
    theme: document.documentElement.classList.contains('dark') ? 'vs-dark' : 'vs',
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    wordWrap: 'off',
  })
  diffEditor.setModel({ original: leftModel, modified: rightModel })
  syncModels()
})

watch(() => [props.leftContent, props.rightContent, props.language], () => {
  syncModels()
})

onBeforeUnmount(() => {
  diffEditor?.dispose()
  leftModel?.dispose()
  rightModel?.dispose()
})
</script>

<style scoped>
.monaco-diff-container {
  width: 100%;
  border: 1px solid var(--el-border-color);
  border-radius: 12px;
  overflow: hidden;
}
</style>