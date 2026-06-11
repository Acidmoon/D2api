<script setup lang="ts">
import { QUOTA_THRESHOLD_TYPE_FIXED, QUOTA_THRESHOLD_TYPE_PERCENTAGE, type QuotaThresholdType } from '@/constants/account'

defineProps<{
  enabled: boolean | null
  threshold: number | null
  thresholdType: QuotaThresholdType | null
}>()

const emit = defineEmits<{
  'update:enabled': [value: boolean | null]
  'update:threshold': [value: number | null]
  'update:thresholdType': [value: QuotaThresholdType | null]
}>()
</script>

<template>
  <div class="flex items-center gap-1.5">
    <button
      type="button"
      @click="emit('update:enabled', !enabled)"
      :class="[
        'quota-notify-switch',
        enabled ? 'quota-notify-switch-on' : 'quota-notify-switch-off'
      ]"
    >
      <span
        :class="[
          'quota-notify-thumb',
          enabled ? 'translate-x-4' : 'translate-x-0'
        ]"
      />
    </button>
    <template v-if="enabled">
      <input
        :value="threshold"
        @input="emit('update:threshold', parseFloat(($event.target as HTMLInputElement).value) || null)"
        type="number"
        min="0"
        :max="thresholdType === QUOTA_THRESHOLD_TYPE_PERCENTAGE ? 100 : undefined"
        :step="thresholdType === QUOTA_THRESHOLD_TYPE_PERCENTAGE ? 1 : 0.01"
        class="input py-1 text-sm flex-1 min-w-0"
      />
      <select
        :value="thresholdType || QUOTA_THRESHOLD_TYPE_FIXED"
        @change="emit('update:thresholdType', ($event.target as HTMLSelectElement).value as QuotaThresholdType)"
        class="input py-1 text-xs w-[4.5rem] flex-shrink-0 text-center"
      >
        <option :value="QUOTA_THRESHOLD_TYPE_FIXED">$</option>
        <option :value="QUOTA_THRESHOLD_TYPE_PERCENTAGE">%</option>
      </select>
    </template>
  </div>
</template>

<style scoped>
.quota-notify-switch {
  position: relative;
  display: inline-flex;
  height: 1.25rem;
  width: 2.25rem;
  flex-shrink: 0;
  cursor: pointer;
  align-items: center;
  border: 1px solid var(--nm-border);
  border-radius: 999px;
  transition: background-color 160ms ease, border-color 160ms ease;
}

.quota-notify-switch:focus {
  outline: 3px solid color-mix(in srgb, var(--nm-accent) 28%, transparent);
  outline-offset: 2px;
}

.quota-notify-switch-on {
  border-color: var(--nm-accent);
  background: var(--nm-accent);
}

.quota-notify-switch-off {
  background: var(--nm-surface-soft);
}

.quota-notify-thumb {
  pointer-events: none;
  display: inline-block;
  height: 0.875rem;
  width: 0.875rem;
  border: 1px solid var(--nm-border);
  border-radius: 999px;
  background: var(--nm-surface);
  transition: transform 160ms ease;
}
</style>
