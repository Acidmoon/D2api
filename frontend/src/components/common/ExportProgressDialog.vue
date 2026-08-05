<template>
  <BaseDialog :show="show" :title="t('usage.exporting')" width="narrow" @close="handleCancel">
    <div class="space-y-4">
      <div class="text-sm text-muted-foreground">
        {{ t('usage.exportingProgress') }}
      </div>
      <div class="flex items-center justify-between text-sm text-muted-foreground">
        <span>{{ t('usage.exportedCount', { current, total }) }}</span>
        <span class="font-medium text-foreground">{{ normalizedProgress }}%</span>
      </div>
      <Progress
        :model-value="normalizedProgress"
        :aria-label="`${t('usage.exportingProgress')}: ${normalizedProgress}%`"
        class="h-2 w-full"
      />
      <div v-if="estimatedTime" class="text-xs text-muted-foreground" aria-live="polite" aria-atomic="true">
        {{ t('usage.estimatedTime', { time: estimatedTime }) }}
      </div>
    </div>

    <template #footer>
      <Button
        variant="outline"
        @click="handleCancel"
      >
        {{ t('usage.cancelExport') }}
      </Button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from './BaseDialog.vue'
import { Progress } from '@/components/ui/progress'
import { Button } from '@/components/ui/button'

interface Props {
  show: boolean
  progress: number
  current: number
  total: number
  estimatedTime: string
}

interface Emits {
  (e: 'cancel'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const { t } = useI18n()

const normalizedProgress = computed(() => {
  const value = Number.isFinite(props.progress) ? props.progress : 0
  return Math.min(100, Math.max(0, Math.round(value)))
})

const handleCancel = () => {
  emit('cancel')
}
</script>
