<template>
  <span
    :class="cn(badgeVariants({ variant: 'outline' }), 'inline-flex items-center gap-1.5')"
  >
    <span
      :class="[
        'inline-block h-2 w-2 rounded-full',
        dotColorClass
      ]"
    ></span>
    <span class="text-sm text-muted-foreground">
      {{ label }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { badgeVariants } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

const props = defineProps<{
  status: string
  label: string
}>()

const dotColorClass = computed(() => {
  switch (props.status) {
    case 'active':
    case 'success':
      return 'bg-[var(--nm-success)]'
    case 'disabled':
    case 'inactive':
    case 'warning':
      return 'bg-[var(--nm-warning)]'
    case 'error':
    case 'danger':
      return 'bg-destructive'
    default:
      return 'bg-[var(--nm-border)]'
  }
})
</script>
