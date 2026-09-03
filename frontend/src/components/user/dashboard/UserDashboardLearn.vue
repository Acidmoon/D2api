<template>
  <!-- QW 学习卡：全宽白卡，标题 + 描边药丸，4 列教程卡 -->
  <section v-if="cards.length > 0" class="rounded-[24px] bg-card p-7 shadow-card" data-testid="dashboard-learn">
    <div class="flex items-center justify-between gap-3">
      <h2 class="text-xl font-semibold text-foreground">{{ t('dashboard.learn.title') }}</h2>
      <a
        v-if="docUrl"
        :href="docUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="qw-pill"
      >
        {{ t('dashboard.learn.docs') }}
        <svg
          class="h-3.5 w-3.5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M7 17L17 7M7 7h10v10" />
        </svg>
      </a>
    </div>
    <div class="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <template v-for="card in cards" :key="card.key">
        <a
          v-if="card.external"
          :href="card.to"
          target="_blank"
          rel="noopener noreferrer"
          class="learn-card group flex min-h-[162px] flex-col rounded-[18px] border border-[color:var(--nm-border-light)] p-5 transition-colors"
        >
          <div class="flex items-start justify-between gap-3">
            <p class="line-clamp-2 text-base font-semibold leading-6 text-foreground">
              {{ card.title }}
            </p>
            <span
              class="flex h-[58px] w-[58px] shrink-0 items-center justify-center rounded-full bg-[color:var(--nm-surface-soft)] text-foreground"
            >
              <Icon :name="card.icon" size="lg" />
            </span>
          </div>
          <p class="mt-auto line-clamp-2 text-sm leading-5 text-[#7F8798] dark:text-[#B4BCC6]">
            {{ card.desc }}
          </p>
        </a>
        <RouterLink
          v-else
          :to="card.to"
          class="learn-card group flex min-h-[162px] flex-col rounded-[18px] border border-[color:var(--nm-border-light)] p-5 transition-colors"
        >
          <div class="flex items-start justify-between gap-3">
            <p class="line-clamp-2 text-base font-semibold leading-6 text-foreground">
              {{ card.title }}
            </p>
            <span
              class="flex h-[58px] w-[58px] shrink-0 items-center justify-center rounded-full bg-[color:var(--nm-surface-soft)] text-foreground"
            >
              <Icon :name="card.icon" size="lg" />
            </span>
          </div>
          <p class="mt-auto line-clamp-2 text-sm leading-5 text-[#7F8798] dark:text-[#B4BCC6]">
            {{ card.desc }}
          </p>
        </RouterLink>
      </template>
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

type LearnIcon = 'sparkles' | 'cube' | 'trophy' | 'book'

interface LearnCard {
  key: string
  to: string
  icon: LearnIcon
  title: string
  desc: string
  /** 外链（如管理员配置的文档地址）时渲染为 <a> 而非 RouterLink。 */
  external?: boolean
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
  // 文档卡：链接管理员在 设置→站点 可配置的 doc_url（运营者自建文档页）。
  if (docUrl.value) {
    list.push({
      key: 'docs',
      to: docUrl.value,
      icon: 'book',
      title: t('dashboard.learn.docsCard.title'),
      desc: t('dashboard.learn.docsCard.desc'),
      external: true
    })
  }
  return list
})
</script>

<style scoped>
/* QW 描边药丸按钮：白底 1px #D1D7E2 描边，dark 下换用边框 token。 */
.qw-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  flex-shrink: 0;
  padding: 0 18px;
  border-radius: 999px;
  border: 1px solid #d1d7e2;
  font-size: 13px;
  font-weight: 500;
  line-height: 1;
  color: var(--nm-ink);
  white-space: nowrap;
  transition: background-color 150ms ease;
}

.qw-pill:hover {
  background-color: var(--nm-surface-soft);
}

:global(.dark) .qw-pill {
  border-color: var(--nm-border);
}

/* 教程卡 hover：描边由中性色过渡到 accent（brand 透明度修饰符对
   hsl(var(--x)) 不可用，直接用 token 表达）。 */
.learn-card:hover {
  border-color: hsl(var(--brand) / 0.45);
}
</style>
