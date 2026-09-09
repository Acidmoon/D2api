import { computed, ref } from 'vue'
import { keysAPI } from '@/api/keys'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey } from '@/types'

/**
 * 生图创作中心的入口/权限判断。
 *
 * 只认「创作分组」下的 Key：分组名包含 CREATION_GROUP_KEYWORD（默认“创作”）
 * 且开启了 allow_image_generation。其余分组即使开了生图权限也不在本页出现。
 * 分组名是当前唯一可用的判据（没有专门的“创作分组”配置项）；若后续改名，
 * 调整下面的常量即可。
 */
export const CREATION_GROUP_KEYWORD = '创作'

const loaded = ref(false)
const loading = ref(false)
const imageKeys = ref<ApiKey[]>([])
let pendingLoad: Promise<ApiKey[]> | null = null
const pageSize = 100

/** 是否为生图创作中心可用的 Key（纯函数，便于单测）。 */
export function isImageStudioKey(key: ApiKey): boolean {
  if (key.status !== 'active') return false
  const group = key.group
  if (!group || group.allow_image_generation !== true) return false
  return (group.name || '').includes(CREATION_GROUP_KEYWORD)
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
      collected.push(...items.filter(isImageStudioKey))
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
