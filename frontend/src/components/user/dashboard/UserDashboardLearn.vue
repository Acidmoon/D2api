<template>
  <section v-if="cards.length > 0" data-testid="dashboard-learn">
    <div class="flex items-center justify-between gap-3">
      <h2 class="text-base font-semibold text-foreground">{{ t('dashboard.learn.title') }}</h2>
      <a
        v-if="docUrl"
        :href="docUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="swiss-action"
      >
        {{ t('dashboard.learn.docs') }}
        <Icon name="externalLink" size="xs" />
      </a>
    </div>
    <div class="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <RouterLink
        v-for="card in cards"
        :key="card.key"
        :to="card.to"
        class="card learn-card group flex flex-col p-5 transition-colors"
      >
        <span class="flex h-10 w-10 items-center justify-center rounded-xl bg-secondary text-brand">
          <Icon :name="card.icon" size="md" />
        </span>
        <p class="mt-4 flex items-center gap-1.5 text-sm font-semibold text-foreground">
          {{ card.title }}
          <Icon
            name="arrowRight"
            size="xs"
            class="text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100"
          />
        </p>
        <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ card.desc }}</p>
      </RouterLink>
    </div>
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
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

type LearnIcon = 'sparkles' | 'cube' | 'trophy' | 'gift'

interface LearnCard {
  key: string
  to: string
  icon: LearnIcon
  title: string
  desc: string
}

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
// 批量生图权限与侧边栏共用同一份模块级缓存，这里只读取、不触发额外请求。
const { canUseBatchImage } = useBatchImageAccess()

const docUrl = computed(() => sanitizeUrl(appStore.docUrl))

const cards = computed<LearnCard[]>(() => {
  if (authStore.isSimpleMode) return []
  const list: LearnCard[] = []
  if (canUseBatchImage.value) {
    list.push({
      key: 'batchImage',
      to: '/batch-image',
      icon: 'sparkles',
      title: t('dashboard.learn.batchImage.title'),
      desc: t('dashboard.learn.batchImage.desc')
    })
  }
  // 与侧边栏一致：受功能开关控制的目的地只在开关开启时出现。
  if (isFeatureFlagEnabled(FeatureFlags.availableChannels)) {
    list.push({
      key: 'channels',
      to: '/available-channels',
      icon: 'cube',
      title: t('dashboard.learn.channels.title'),
      desc: t('dashboard.learn.channels.desc')
    })
  }
  if (isFeatureFlagEnabled(FeatureFlags.leaderboard)) {
    list.push({
      key: 'leaderboard',
      to: '/leaderboard',
      icon: 'trophy',
      title: t('dashboard.learn.leaderboard.title'),
      desc: t('dashboard.learn.leaderboard.desc')
    })
  }
  list.push({
    key: 'redeem',
    to: '/redeem',
    icon: 'gift',
    title: t('dashboard.learn.redeem.title'),
    desc: t('dashboard.learn.redeem.desc')
  })
  return list
})
</script>

<style scoped>
/* 同 UserDashboardStats：brand 透明度修饰符不可用，hover 边框用 var token。 */
.learn-card:hover {
  border-color: hsl(var(--brand) / 0.45);
}
</style>
