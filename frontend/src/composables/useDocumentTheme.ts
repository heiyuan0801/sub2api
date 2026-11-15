import { onBeforeUnmount, onMounted, ref } from 'vue'

/** Share the root theme and saved preference across the console and public layouts. */
export function useDocumentTheme() {
  const readTheme = () => typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
  const isDark = ref(readTheme())
  let observer: MutationObserver | undefined

  onMounted(() => {
    isDark.value = readTheme()
    observer = new MutationObserver(() => { isDark.value = readTheme() })
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  })
  onBeforeUnmount(() => observer?.disconnect())

  function toggleTheme() {
    const dark = !readTheme()
    document.documentElement.classList.toggle('dark', dark)
    isDark.value = dark
    try {
      window.localStorage.setItem('theme', dark ? 'dark' : 'light')
    } catch {
      // The current session can still use the theme if storage is disabled.
    }
  }

  return { isDark, toggleTheme }
}
