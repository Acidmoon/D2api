<template>
  <section class="card p-2" data-testid="dashboard-quick-actions">
    <nav class="grid grid-cols-3 gap-1 md:grid-cols-6" :aria-label="t('dashboard.quickActions')">
      <template v-for="action in actions" :key="action.key">
        <a
          v-if="action.external"
          :href="action.to"
          target="_blank"
          rel="noopener noreferrer"
          class="group flex flex-col items-center gap-2 rounded-xl px-2 py-4 transition-colors hover:bg-secondary"
        >
          <span class="flex h-10 w-10 items-center justify-center rounded-xl bg-secondary text-brand">
            <Icon :name="action.icon" size="md" />
          </span>
          <span class="w-full truncate text-center text-xs font-medium text-foreground">
            {{ action.label }}
          </span>
        </a>
        <RouterLink
          v-else
          :to="action.to"
          class="group flex flex-col items-center gap-2 rounded-xl px-2 py-4 transition-colors hover:bg-secondary"
        >
          <span class="flex h-10 w-10 items-center justify-center rounded-xl bg-secondary text-brand">
            <Icon :name="action.icon" size="md" />
          </span>
          <span class="w-full truncate text-center text-xs font-medium text-foreground">
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
