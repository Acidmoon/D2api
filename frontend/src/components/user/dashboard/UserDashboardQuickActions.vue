<template>
  <!-- QW 快捷导航条：白卡 24px 圆角、高 120px、px-3，6 入口 justify-between -->
  <section class="rounded-[24px] bg-card shadow-card" data-testid="dashboard-quick-actions">
    <nav
      class="flex h-[120px] items-center justify-between gap-2 overflow-x-auto px-3"
      :aria-label="t('dashboard.quickActions')"
    >
      <template v-for="action in actions" :key="action.key">
        <a
          v-if="action.external"
          :href="action.to"
          target="_blank"
          rel="noopener noreferrer"
          class="group flex h-24 min-w-0 flex-1 flex-col items-center rounded-xl pt-[18px] transition-colors hover:bg-[#F9FAFD] md:w-[202px] md:flex-none dark:hover:bg-[#1D2026]"
        >
          <span class="text-foreground"><Icon :name="action.icon" size="lg" /></span>
          <span class="mt-2 w-full truncate px-1 text-center text-base font-normal text-foreground">
            {{ action.label }}
          </span>
        </a>
        <RouterLink
          v-else
          :to="action.to"
          class="group flex h-24 min-w-0 flex-1 flex-col items-center rounded-xl pt-[18px] transition-colors hover:bg-[#F9FAFD] md:w-[202px] md:flex-none dark:hover:bg-[#1D2026]"
        >
          <span class="text-foreground"><Icon :name="action.icon" size="lg" /></span>
          <span class="mt-2 w-full truncate px-1 text-center text-base font-normal text-foreground">
            {{ action.label }}
          </span>
        </RouterLink>
      </template>
    </nav>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { sanitizeUrl } from '@/utils/url'

type QuickActionIcon = 'key' | 'chart' | 'creditCard' | 'document' | 'cube' | 'book'

interface QuickAction {
  key: string
  to: string
  icon: QuickActionIcon
  label: string
  external?: boolean
}

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const docUrl = computed(() => sanitizeUrl(appStore.docUrl))

const actions = computed<QuickAction[]>(() => {
  const list: QuickAction[] = [
    { key: 'keys', to: '/keys', icon: 'key', label: t('dashboard.apiKeys') }
  ]
  if (!authStore.isSimpleMode) {
    list.push({ key: 'usage', to: '/usage', icon: 'chart', label: t('nav.usage') })
  }
  if (!authStore.isSimpleMode && isFeatureFlagEnabled(FeatureFlags.payment)) {
    list.push({ key: 'topUp', to: '/purchase', icon: 'creditCard', label: t('dashboard.topUp') })
    list.push({ key: 'orders', to: '/orders', icon: 'document', label: t('dashboard.orders') })
  }
  if (!authStore.isSimpleMode && isFeatureFlagEnabled(FeatureFlags.availableChannels)) {
    list.push({
      key: 'channels',
      to: '/available-channels',
      icon: 'cube',
      label: t('dashboard.channels')
    })
  }
  if (docUrl.value) {
    list.push({ key: 'docs', to: docUrl.value, icon: 'book', label: t('nav.docs'), external: true })
  }
  return list
})
</script>

<style scoped>
/* 窄屏横向滚动时隐藏滚动条，保持 QW 白卡的干净观感。 */
nav {
  scrollbar-width: none;
}
nav::-webkit-scrollbar {
  display: none;
}
</style>
