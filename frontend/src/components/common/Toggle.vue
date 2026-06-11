<template>
  <button
    type="button"
    @click="toggle"
    class="switch"
    :class="{ 'switch-active': modelValue }"
    role="switch"
    :aria-checked="modelValue"
  >
    <span
      class="switch-thumb"
    />
  </button>
</template>

<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

function toggle() {
  emit('update:modelValue', !props.modelValue)
}
</script>
