<script setup>
import { onMounted, onUnmounted } from 'vue'
import { ensureThemeInitialized } from '@/utils/theme'

let cleanupThemeListener = () => {}

onMounted(() => {
  cleanupThemeListener = ensureThemeInitialized()
})

onUnmounted(() => {
  cleanupThemeListener()
})
</script>

<template>
  <RouterView/>
</template>

<style scoped>

</style>
