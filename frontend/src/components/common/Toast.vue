<template>
  <Teleport to="body">
    <div
      class="pointer-events-none fixed right-4 top-4 z-[9999] space-y-3"
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
            'toast-card pointer-events-auto min-w-[320px] max-w-md overflow-hidden',
            getToastToneClass(toast.type)
          ]"
        >
          <div class="p-4">
            <div class="flex items-start gap-3">
              <!-- Icon -->
              <div class="mt-0.5 flex-shrink-0">
                <Icon
                  :name="getToastIconName(toast.type)"
                  size="md"
                  class="toast-icon"
                  aria-hidden="true"
                />
              </div>

              <!-- Content -->
              <div class="min-w-0 flex-1">
                <p v-if="toast.title" class="text-sm font-semibold" style="color: var(--nm-ink)">
                  {{ toast.title }}
                </p>
                <p
                  :class="[
                    'text-sm leading-relaxed',
                    toast.title ? 'mt-1' : ''
                  ]"
                  style="color: var(--nm-ink-muted)"
                >
                  {{ toast.message }}
                </p>
              </div>

              <!-- Close button -->
              <button
                @click="removeToast(toast.id)"
                class="toast-close -m-1 flex-shrink-0 p-1"
                aria-label="Close notification"
              >
                <Icon name="x" size="sm" />
              </button>
            </div>
          </div>

          <!-- Progress bar -->
          <div v-if="toast.duration" class="toast-progress-track h-1">
            <div
              class="h-full toast-progress"
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
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const toasts = computed(() => appStore.toasts)

const getToastIconName = (type: string): 'checkCircle' | 'xCircle' | 'exclamationTriangle' | 'infoCircle' => {
  switch (type) {
    case 'success':
      return 'checkCircle'
    case 'error':
      return 'xCircle'
    case 'warning':
      return 'exclamationTriangle'
    case 'info':
    default:
      return 'infoCircle'
  }
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
.toast-card {
  background: var(--nm-surface);
  border: 1px solid var(--nm-border);
  border-left-width: 4px;
  border-radius: var(--nm-radius-lg);
}

.toast-success {
  --toast-tone: var(--nm-success);
}

.toast-error {
  --toast-tone: var(--nm-danger);
}

.toast-warning {
  --toast-tone: var(--nm-warning);
}

.toast-info {
  --toast-tone: var(--nm-info);
}

.toast-card {
  border-left-color: var(--toast-tone);
}

.toast-icon {
  color: var(--toast-tone);
}

.toast-close {
  color: var(--nm-ink-faint);
  border-radius: var(--nm-radius-sm);
  transition: background-color 160ms ease, color 160ms ease;
}

.toast-close:hover {
  color: var(--nm-ink);
  background: var(--nm-surface-soft);
}

.toast-progress-track {
  background: var(--nm-surface-soft);
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
