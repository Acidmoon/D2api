<template>
  <div class="flex items-center gap-1.5">
    <span
      :class="[
        'status-dot inline-block h-2 w-2',
        variantClass
      ]"
    ></span>
    <span class="text-sm" style="color: var(--nm-ink-muted)">
      {{ label }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: string
  label: string
}>()

const variantClass = computed(() => {
  switch (props.status) {
    case 'active':
    case 'success':
      return 'status-dot-success'
    case 'disabled':
    case 'inactive':
    case 'warning':
      return 'status-dot-warning'
    case 'error':
    case 'danger':
      return 'status-dot-danger'
    default:
      return 'status-dot-muted'
  }
})
</script>

<style scoped>
.status-dot {
  border-radius: 999px;
}

.status-dot-success {
  background: var(--nm-success);
}

.status-dot-warning {
  background: var(--nm-warning);
}

.status-dot-danger {
  background: var(--nm-danger);
}

.status-dot-muted {
  background: var(--nm-border);
}
</style>
