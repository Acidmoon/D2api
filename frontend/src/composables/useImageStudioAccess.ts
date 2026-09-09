import { computed, ref } from 'vue'
import { keysAPI } from '@/api/keys'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey } from '@/types'

/**
 * 生图创作中心的入口/权限判断。
 *
 * 判定口径：用户存在至少一个 active 的 API Key，且其所属分组开启了
 * `allow_image_generation`。侧边栏据此决定是否显示入口，页面据此决定
 * 顶部 Key 选择器里有哪些候选项。
 */

const loaded = ref(false)
const loading = ref(false)
const imageKeys = ref<ApiKey[]>([])
let pendingLoad: Promise<ApiKey[]> | null = null
const pageSize = 100

function keyAllowsImageStudio(key: ApiKey): boolean {
  return key.status === 'active' && key.group?.allow_image_generation === true
}

async function loadImageStudioAccess(force = false): Promise<ApiKey[]> {
  const authStore = useAuthStore()
  if (!authStore.isAuthenticated) {
    loaded.value = true
    imageKeys.value = []
    return []
  }

  if (loaded.value && !force) {
    return imageKeys.value
  }
  if (pendingLoad && !force) {
    return pendingLoad
  }

  loading.value = true
  pendingLoad = (async () => {
    const collected: ApiKey[] = []
    let page = 1
    while (true) {
      const response = await keysAPI.list(page, pageSize, {
        status: 'active',
        sort_by: 'created_at',
        sort_order: 'desc'
      })
      const items = response.items || []
      collected.push(...items.filter(keyAllowsImageStudio))
      if (page >= response.pages || items.length === 0) break
      page += 1
    }
    imageKeys.value = collected
    loaded.value = true
    return collected
  })()
    .catch(() => {
      imageKeys.value = []
      loaded.value = true
      return [] as ApiKey[]
    })
    .finally(() => {
      loading.value = false
      pendingLoad = null
    })

  return pendingLoad
}

export function useImageStudioAccess() {
  return {
    canUseImageStudio: computed(() => imageKeys.value.length > 0),
    imageStudioAccessLoaded: computed(() => loaded.value),
    imageStudioAccessLoading: computed(() => loading.value),
    imageStudioKeys: computed(() => imageKeys.value),
    loadImageStudioAccess
  }
}
