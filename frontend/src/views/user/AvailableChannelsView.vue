<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div>
          <!-- In-content page header -->
          <div class="page-header">
            <h1 class="page-title">{{ t('availableChannels.title') }}</h1>
            <p class="page-description">{{ t('availableChannels.description') }}</p>
          </div>

          <!-- Summary KPI cards -->
          <section class="mb-5 grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div class="card p-5">
              <div class="flex items-center justify-between">
                <span class="stat-label">{{ t('availableChannels.stats.channels') }}</span>
                <span class="stat-icon stat-icon-primary"><Icon name="grid" size="sm" /></span>
              </div>
              <p class="stat-value mt-3">{{ formatCount(totalChannels) }}</p>
            </div>
            <div class="card p-5">
              <div class="flex items-center justify-between">
                <span class="stat-label">{{ t('availableChannels.stats.platforms') }}</span>
                <span class="stat-icon"><Icon name="globe" size="sm" /></span>
              </div>
              <p class="stat-value mt-3">{{ formatCount(totalPlatforms) }}</p>
            </div>
            <div class="card p-5">
              <div class="flex items-center justify-between">
                <span class="stat-label">{{ t('availableChannels.stats.models') }}</span>
                <span class="stat-icon stat-icon-success"><Icon name="chart" size="sm" /></span>
              </div>
              <p class="stat-value mt-3">{{ formatCount(totalModels) }}</p>
            </div>
          </section>

          <!-- Toolbar: search left, refresh right -->
          <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div class="flex flex-1 flex-wrap items-center gap-2">
              <SearchInput
                v-model="searchQuery"
                :placeholder="t('availableChannels.searchPlaceholder')"
                class="w-full sm:w-80"
                pills
              />
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <Button
                variant="ghost"
                size="icon"
                class="rounded-full"
                :disabled="loading"
                :title="t('common.refresh', 'Refresh')"
                :aria-label="t('common.refresh', 'Refresh')"
                @click="loadChannels"
              >
                <RefreshCw :class="loading ? 'animate-spin' : ''" />
              </Button>
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <AvailableChannelsTable
          :columns="columnLabels"
          :rows="filteredChannels"
          :loading="loading"
          :user-group-rates="userGroupRates"
          pricing-key-prefix="availableChannels.pricing"
          :no-pricing-label="t('availableChannels.noPricing')"
          :no-models-label="t('availableChannels.noModels')"
          :empty-label="t('availableChannels.empty')"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshCw } from 'lucide-vue-next'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import { Button } from '@/components/ui/button'
import AvailableChannelsTable from '@/components/channels/AvailableChannelsTable.vue'
import userChannelsAPI, { type UserAvailableChannel } from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const channels = ref<UserAvailableChannel[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')

const columnLabels = computed(() => ({
  name: t('availableChannels.columns.name'),
  description: t('availableChannels.columns.description'),
  platform: t('availableChannels.columns.platform'),
  groups: t('availableChannels.columns.groups'),
  supportedModels: t('availableChannels.columns.supportedModels'),
}))

// 全量统计（不受搜索过滤影响）：渠道数 / 接入平台数 / 支持模型数（按模型名去重）。
const totalChannels = computed(() => channels.value.length)
const totalPlatforms = computed(() => {
  const platforms = new Set<string>()
  for (const ch of channels.value) {
    for (const section of ch.platforms) platforms.add(section.platform)
  }
  return platforms.size
})
const totalModels = computed(() => {
  const models = new Set<string>()
  for (const ch of channels.value) {
    for (const section of ch.platforms) {
      for (const m of section.supported_models) models.add(m.name)
    }
  }
  return models.size
})

const formatCount = (value: number) => value.toLocaleString()

/**
 * 搜索过滤：
 * - 命中渠道名/描述 → 整个渠道（所有 platforms）都保留
 * - 否则按 platform/group/model 维度在 sections 里过滤，保留有匹配的 section
 * - 所有 sections 都不匹配时，渠道本身被过滤掉
 */
const filteredChannels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return channels.value
  return channels.value
    .map((ch) => {
      const nameHit = ch.name.toLowerCase().includes(q)
      const descHit = (ch.description || '').toLowerCase().includes(q)
      if (nameHit || descHit) return ch
      const matchingSections = ch.platforms.filter(
        (p) =>
          p.platform.toLowerCase().includes(q) ||
          p.groups.some((g) => g.name.toLowerCase().includes(q)) ||
          p.supported_models.some((m) => m.name.toLowerCase().includes(q)),
      )
      if (matchingSections.length === 0) return null
      return { ...ch, platforms: matchingSections }
    })
    .filter((ch): ch is UserAvailableChannel => ch !== null)
})

async function loadChannels() {
  loading.value = true
  try {
    // 渠道列表和用户专属倍率并发拉取。专属倍率失败不阻塞渠道展示——
    // 失败时只是无法渲染专属倍率角标，降级为仅显示默认倍率。
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
    ])
    channels.value = list
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadChannels)
</script>
