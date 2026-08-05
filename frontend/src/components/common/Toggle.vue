<template>
  <Switch
    :model-value="modelValue"
    :disabled="disabled"
    aria-label="Toggle"
    @update:model-value="handleUpdate"
  />
</template>

<script setup lang="ts">
import { Switch } from '@/components/ui/switch'

const props = defineProps<{
  modelValue: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

const handleUpdate = (value: boolean | undefined) => {
  emit('update:modelValue', value ?? !props.modelValue)
}
</script>
