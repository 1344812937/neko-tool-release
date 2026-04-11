import { computed, ref } from 'vue'

export const THEME_MODE_KEY = 'neko-theme-mode'
export const themeMode = ref('auto')

const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
let initialized = false

export const resolvedTheme = computed(() => {
  if (themeMode.value === 'auto') {
    return mediaQuery.matches ? 'dark' : 'light'
  }
  return themeMode.value
})

export function applyTheme() {
  const useDark = resolvedTheme.value === 'dark'
  document.documentElement.classList.toggle('dark', useDark)
  document.documentElement.classList.toggle('light', !useDark)
  document.documentElement.dataset.themeMode = themeMode.value
}

function handleSystemThemeChange() {
  if (themeMode.value === 'auto') {
    applyTheme()
  }
}

export function setThemeMode(mode) {
  themeMode.value = mode
  localStorage.setItem(THEME_MODE_KEY, mode)
  applyTheme()
}

export function ensureThemeInitialized() {
  if (initialized) {
    applyTheme()
    return () => {}
  }
  themeMode.value = localStorage.getItem(THEME_MODE_KEY) || 'auto'
  applyTheme()
  mediaQuery.addEventListener('change', handleSystemThemeChange)
  initialized = true
  return () => {
    mediaQuery.removeEventListener('change', handleSystemThemeChange)
    initialized = false
  }
}