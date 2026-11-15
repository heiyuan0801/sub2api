import { computed } from 'vue'
import { useDocumentTheme } from '@/composables/useDocumentTheme'
import './chartColors.css'

const fallback = {
  brand: '211 90% 50%',
  success: '152 56% 39%',
  warning: '35 92% 46%',
  danger: '0 72% 51%',
  'muted-foreground': '0 0% 43%',
  foreground: '0 0% 9%',
  border: '0 0% 90%',
  popover: '0 0% 100%',
  'chart-violet': '258 66% 49%',
  'chart-rose': '326 68% 43%',
  'chart-teal': '173 76% 32%',
  'chart-orange': '21 83% 43%',
  'chart-indigo': '233 59% 51%',
  'chart-lime': '83 64% 31%',
  'chart-cyan': '191 79% 33%',
  'chart-purple': '277 69% 48%'
} as const

type ColorToken = keyof typeof fallback
const series: readonly ColorToken[] = [
  'brand', 'success', 'warning', 'danger', 'chart-violet', 'chart-rose',
  'chart-teal', 'chart-orange', 'chart-indigo', 'chart-lime', 'chart-cyan', 'chart-purple'
]

export function useChartColors() {
  const { isDark } = useDocumentTheme()
  const themedStyle = computed(() => {
    // Reading the reactive flag invalidates the resolved palette on a theme change.
    const selector = isDark.value ? ':root.dark' : ':root'
    const root = typeof document === 'undefined' ? null : document.querySelector(selector)
    return root ? getComputedStyle(root) : null
  })

  function color(token: ColorToken, alpha = 1) {
    const [h, s, l] = (themedStyle.value?.getPropertyValue(`--${token}`).trim() || fallback[token]).split(/\s+/)
    // Canvas must receive literal colors; CSS var() expressions are not resolved there.
    return `hsla(${h}, ${s}, ${l}, ${alpha})`
  }

  return {
    color,
    seriesColor: (index: number, alpha = 1) => color(series[index % series.length], alpha)
  }
}
