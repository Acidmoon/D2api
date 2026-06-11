<script setup lang="ts">
defineProps<{
  colorClass: string
  tooltip?: string
  current: string | number
  max: string | number
  suffix?: string
}>()
</script>

<template>
  <span
    :class="[
      'badge capacity-badge',
      colorClass
    ]"
    :title="tooltip"
  >
    <slot />
    <span class="font-mono">{{ current }}</span>
    <span class="capacity-separator">/</span>
    <span class="font-mono">{{ max }}</span>
    <span v-if="suffix" class="text-[9px] opacity-60">{{ suffix }}</span>
  </span>
</template>

<style scoped>
.capacity-badge {
  gap: 0.25rem;
  padding: 1px 0.375rem;
  font-size: 0.625rem;
  line-height: 1.2;
}

.capacity-separator {
  color: var(--nm-ink-faint);
}
</style>
