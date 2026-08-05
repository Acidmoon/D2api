<template>
  <Teleport to="body">
    <div
      class="pointer-events-none fixed right-4 top-4 z-[9999] flex flex-col gap-3"
      aria-live="polite"
      aria-atomic="true"
    >
      <TransitionGroup
        enter-active-class="transition ease-out duration-300"
        enter-from-class="opacity-0 translate-x-full"
        enter-to-class="opacity-100 translate-x-0"
        leave-active-class="transition ease-in duration-200"
        leave-from-class="opacity-100 translate-x-0"
        leave-to-class="opacity-0 translate-x-full"
      >
        <div
          v-for="toast in toasts"
          :key="toast.id"
          :class="[
            'toast-card pointer-events-auto relative flex w-full min-w-[320px] max-w-md items-start gap-3 overflow-hidden rounded-lg border border-border border-l-4 bg-popover p-4 text-popover-foreground shadow-lg',
            getToastToneClass(toast.type)
          ]"
          :style="{ '--toast-tone': getToastToneVar(toast.type) }"
        >
          <div class="mt-0.5 flex-shrink-0">
            <component
              :is="getToastIcon(toast.type)"
              class="toast-icon h-5 w-5"
              aria-hidden="true"
            />
          </div>

          <!-- Content -->
          <div class="min-w-0 flex-1">
            <p v-if="toast.title" class="text-sm font-semibold text-foreground">
              {{ toast.title }}
            </p>
            <p
              :class="[
                'text-sm leading-relaxed text-muted-foreground',
                toast.title ? 'mt-1' : ''
              ]"
            >
              {{ toast.message }}
            </p>
          </div>

          <!-- Close button -->
          <button
            @click="removeToast(toast.id)"
            class="-m-1 flex-shrink-0 rounded-sm p-1 text-muted-foreground transition-colors hover:text-foreground"
            aria-label="Close notification"
          >
            <X class="h-4 w-4" />
          </button>

          <!-- Progress bar -->
          <div v-if="toast.duration" class="toast-progress-track absolute bottom-0 left-0 h-1 w-full">
            <div
              class="toast-progress h-full"
              :style="{ animationDuration: `${toast.duration}ms` }"
            ></div>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { CircleCheck, CircleX, Info, TriangleAlert, X } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const toasts = computed(() => appStore.toasts)

const getToastIcon = (type: string) => {
  switch (type) {
    case 'success':
      return CircleCheck
    case 'error':
      return CircleX
    case 'warning':
      return TriangleAlert
    case 'info':
    default:
      return Info
  }
}

const getToastToneVar = (type: string): string => {
  const tones: Record<string, string> = {
    success: 'var(--nm-success)',
    error: 'var(--nm-danger)',
    warning: 'var(--nm-warning)',
    info: 'var(--nm-info)'
  }
  return tones[type] || tones.info
}

const getToastToneClass = (type: string): string => {
  const tones: Record<string, string> = {
    success: 'toast-success',
    error: 'toast-error',
    warning: 'toast-warning',
    info: 'toast-info'
  }
  return tones[type] || tones.info
}

const removeToast = (id: string) => {
  appStore.hideToast(id)
}
</script>

<style scoped>
.toast-success {
  border-left-color: var(--toast-tone);
}

.toast-error {
  border-left-color: var(--toast-tone);
}

.toast-warning {
  border-left-color: var(--toast-tone);
}

.toast-info {
  border-left-color: var(--toast-tone);
}

.toast-icon {
  color: var(--toast-tone);
}

.toast-progress-track {
  background: hsl(var(--muted));
}

.toast-progress {
  width: 100%;
  background: var(--toast-tone);
  animation-name: toast-progress-shrink;
  animation-timing-function: linear;
  animation-fill-mode: forwards;
}

@keyframes toast-progress-shrink {
  from {
    width: 100%;
  }
  to {
    width: 0%;
  }
}
</style>
