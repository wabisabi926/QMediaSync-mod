import { defineAsyncComponent, defineComponent, h, type AsyncComponentLoader } from 'vue'
import RouteLoadError from '@/components/common/RouteLoadError.vue'
import RouteLoading from '@/components/common/RouteLoading.vue'

export function createAsyncRouteComponent(name: string, loader: AsyncComponentLoader) {
  const AsyncRoutePage = defineAsyncComponent({
    loader,
    loadingComponent: RouteLoading,
    errorComponent: RouteLoadError,
    delay: 0,
    timeout: 30_000,
    suspensible: false,
  })

  return defineComponent({
    name,
    setup: () => () => h(AsyncRoutePage),
  })
}
