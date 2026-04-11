<template>
  <div ref="containerRef" class="monaco-code-preview"></div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import 'monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution'
import 'monaco-editor/esm/vs/basic-languages/typescript/typescript.contribution'
import 'monaco-editor/esm/vs/basic-languages/java/java.contribution'
import 'monaco-editor/esm/vs/basic-languages/go/go.contribution'
import 'monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution'
import 'monaco-editor/esm/vs/basic-languages/shell/shell.contribution'
import 'monaco-editor/esm/vs/language/json/monaco.contribution'

if (!window.MonacoEnvironment) {
  window.MonacoEnvironment = {
    getWorker() {
      return new editorWorker()
    },
  }
}

const props = defineProps({
  content: {
    type: String,
    default: '',
  },
  language: {
    type: String,
    default: 'plaintext',
  },
})

const containerRef = ref(null)
let editor
let model

function ensureModel() {
  if (!model) {
    model = monaco.editor.createModel(props.content, props.language)
  }
}

function syncModel() {
  ensureModel()
  if (model.getValue() !== props.content) {
    model.setValue(props.content)
  }
  monaco.editor.setModelLanguage(model, props.language)
}

onMounted(() => {
  ensureModel()
  editor = monaco.editor.create(containerRef.value, {
    model,
    automaticLayout: true,
    readOnly: true,
    theme: document.documentElement.classList.contains('dark') ? 'vs-dark' : 'vs',
    minimap: { enabled: false },
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    wordWrap: 'off',
    glyphMargin: false,
    folding: false,
    renderLineHighlight: 'none',
    overviewRulerLanes: 0,
  })
  syncModel()
})

watch(() => [props.content, props.language], () => {
  syncModel()
})

onBeforeUnmount(() => {
  editor?.dispose()
  model?.dispose()
})
</script>

<style scoped>
.monaco-code-preview {
  width: 100%;
  height: 100%;
  border: 1px solid var(--el-border-color);
  border-radius: 12px;
  overflow: hidden;
}
</style>