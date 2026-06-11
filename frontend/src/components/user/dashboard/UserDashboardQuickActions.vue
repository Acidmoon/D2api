<template>
  <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
    <button @click="router.push('/keys')" class="quick-action card">
      <span class="quick-ic"><Icon name="key" size="lg" /></span>
      <span class="min-w-0 flex-1 text-left">
        <span class="block text-sm font-medium" style="color: var(--nm-ink)">{{ t('dashboard.createApiKey') }}</span>
        <span class="block text-xs" style="color: var(--nm-ink-faint)">{{ t('dashboard.generateNewKey') }}</span>
      </span>
      <Icon name="chevronRight" size="md" style="color: var(--nm-ink-faint)" />
    </button>

    <button @click="router.push('/usage')" class="quick-action card">
      <span class="quick-ic"><Icon name="chart" size="lg" /></span>
      <span class="min-w-0 flex-1 text-left">
        <span class="block text-sm font-medium" style="color: var(--nm-ink)">{{ t('dashboard.viewUsage') }}</span>
        <span class="block text-xs" style="color: var(--nm-ink-faint)">{{ t('dashboard.checkDetailedLogs') }}</span>
      </span>
      <Icon name="chevronRight" size="md" style="color: var(--nm-ink-faint)" />
    </button>

    <button @click="router.push('/redeem')" class="quick-action card">
      <span class="quick-ic"><Icon name="gift" size="lg" /></span>
      <span class="min-w-0 flex-1 text-left">
        <span class="block text-sm font-medium" style="color: var(--nm-ink)">{{ t('dashboard.redeemCode') }}</span>
        <span class="block text-xs" style="color: var(--nm-ink-faint)">{{ t('dashboard.addBalanceWithCode') }}</span>
      </span>
      <Icon name="chevronRight" size="md" style="color: var(--nm-ink-faint)" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
const router = useRouter()
const { t } = useI18n()
</script>

<style scoped>
.quick-action {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  cursor: pointer;
  transition: box-shadow 200ms ease, transform 200ms ease;
}

.quick-action:hover {
  box-shadow: var(--nm-shadow-raised-lg);
  transform: translateY(-2px);
}

.quick-action:active {
  box-shadow: var(--nm-shadow-pressed);
  transform: translateY(0);
}

.quick-ic {
  display: flex;
  height: 2.5rem;
  width: 2.5rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: var(--nm-radius);
  background: var(--nm-bg);
  box-shadow: var(--nm-shadow-raised-sm);
  color: var(--nm-accent-text);
}
</style>
