<template>
  <label class="flex cursor-pointer flex-col items-center gap-0.5">
    <span class="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">{{ label }}</span>
    <button
      type="button"
      role="switch"
      :aria-checked="checked"
      @click="emit('toggle')"
      :class="['switch', checked ? 'switch-active' : '']"
    >
      <span :class="['switch-thumb', checked ? 'translate-x-5' : 'translate-x-0']" />
    </button>
  </label>
</template>

<script setup lang="ts">
defineProps<{ label: string; checked: boolean }>()
const emit = defineEmits<{ toggle: [] }>()
</script>
