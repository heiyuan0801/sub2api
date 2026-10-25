import { onMounted, watch } from 'vue'

// Lazy dialogs can be mounted already open. Run their initialization after
// setup as well as on subsequent visibility changes (avoids setup-order TDZ).
export function useModalVisibility(source: () => boolean, callback: (visible: boolean) => void | Promise<void>) {
  watch(source, callback)
  onMounted(() => { if (source()) void callback(true) })
}
