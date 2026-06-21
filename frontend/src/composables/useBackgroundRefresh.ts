import { computed, ref } from 'vue'

export function useBackgroundRefresh() {
  const hasLoaded = ref(false)
  const isRefreshing = ref(false)

  const initialLoading = computed(() => !hasLoaded.value && isRefreshing.value)
  const backgroundRefreshing = computed(() => hasLoaded.value && isRefreshing.value)

  const runRefresh = async <T>(loader: () => Promise<T>): Promise<T | undefined> => {
    if (isRefreshing.value) {
      return undefined
    }

    isRefreshing.value = true
    try {
      const result = await loader()
      hasLoaded.value = true
      return result
    } finally {
      isRefreshing.value = false
    }
  }

  return {
    hasLoaded,
    isRefreshing,
    initialLoading,
    backgroundRefreshing,
    runRefresh,
  }
}
