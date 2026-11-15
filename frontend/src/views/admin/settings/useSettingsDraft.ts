import { computed, shallowRef } from 'vue'

/** Track a form's draft in memory; no storage, requests, or navigation side effects. */
export function useSettingsDraft(read: () => unknown) {
  const baseline = shallowRef<string>()
  const serialized = computed(() => JSON.stringify(read()))
  const dirty = computed(() => baseline.value !== undefined && baseline.value !== serialized.value)

  function markSaved(snapshot = serialized.value) {
    baseline.value = snapshot
  }

  return { dirty, markSaved, capture: () => serialized.value }
}
