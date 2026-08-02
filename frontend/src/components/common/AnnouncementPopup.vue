<template>
  <Teleport to="body">
    <Transition name="popup-fade">
      <div
        v-if="displayedAnnouncement"
        class="fixed inset-0 z-[120] flex items-start justify-center overflow-y-auto p-4 pt-[8vh] backdrop-blur-md"
        style="background-color: rgba(28, 31, 38, 0.45)"
      >
        <div
          class="announcement-popup w-full max-w-[680px] overflow-hidden"
          @click.stop
        >
          <!-- Header -->
          <div class="relative overflow-hidden border-b px-8 py-6" style="background: var(--nm-surface); border-color: var(--nm-border)">
            <div class="relative z-10">
              <!-- Icon and badge -->
              <div class="mb-3 flex items-center gap-2">
                <div class="announcement-icon flex h-10 w-10 items-center justify-center">
                  <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
                  </svg>
                </div>
                <span class="badge badge-primary inline-flex items-center gap-1.5">
                  <span class="relative flex h-2 w-2">
                    <span class="announcement-dot-ping absolute inline-flex h-full w-full animate-ping opacity-75"></span>
                    <span class="announcement-dot relative inline-flex h-2 w-2"></span>
                  </span>
                  {{ t('announcements.unread') }}
                </span>
              </div>

              <!-- Title -->
              <h2 class="mb-2 text-2xl font-bold leading-tight" style="color: var(--nm-ink)">
                {{ displayedAnnouncement.title }}
              </h2>

              <!-- Time -->
              <div class="flex items-center gap-1.5 text-sm" style="color: var(--nm-ink-muted)">
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <time>{{ formatRelativeWithDateTime(displayedAnnouncement.created_at) }}</time>
              </div>
            </div>
          </div>

          <!-- Body -->
          <div class="max-h-[50vh] overflow-y-auto px-8 py-8" style="background: var(--nm-bg)">
            <div class="relative">
              <div class="absolute left-0 top-0 bottom-0 w-px" style="background: var(--nm-accent)"></div>
              <div class="pl-6">
                <div
                  class="markdown-body prose prose-sm max-w-none dark:prose-invert"
                  v-html="renderedContent"
                ></div>
              </div>
            </div>
          </div>

          <!-- Footer -->
          <div class="border-t px-8 py-5" style="border-color: var(--nm-border-light); background: var(--nm-surface-soft)">
            <div class="flex items-center justify-end">
              <button
                @click="handleDismiss"
                data-testid="announcement-popup-dismiss"
                class="btn"
              >
                <span class="flex items-center gap-2">
                  <svg v-if="preview" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                  <svg v-else class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  {{ preview ? t('common.close') : t('announcements.markRead') }}
                </span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatRelativeWithDateTime } from '@/utils/format'
import type { Announcement, UserAnnouncement } from '@/types'
import '@/styles/announcement-markdown.css'

type PreviewAnnouncement = Pick<Announcement | UserAnnouncement, 'title' | 'content' | 'created_at'>

const props = withDefaults(defineProps<{
  announcement?: PreviewAnnouncement | null
  preview?: boolean
}>(), {
  announcement: null,
  preview: false,
})

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const announcementStore = useAnnouncementStore()
const displayedAnnouncement = computed(() => (
  props.preview ? props.announcement : announcementStore.currentPopup
))

marked.setOptions({
  breaks: true,
  gfm: true,
})

const renderedContent = computed(() => {
  const content = displayedAnnouncement.value?.content
  if (!content) return ''
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
})

function handleDismiss() {
  if (props.preview) {
    emit('close')
    return
  }
  announcementStore.dismissPopup()
}

// Manage body overflow — only set, never unset (bell component handles restore)
watch(
  displayedAnnouncement,
  (popup) => {
    if (popup) {
      document.body.style.overflow = 'hidden'
    } else if (props.preview) {
      document.body.style.overflow = ''
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  if (props.preview) {
    document.body.style.overflow = ''
  }
})
</script>

<style scoped>
.popup-fade-enter-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.popup-fade-leave-active {
  transition: all 0.2s cubic-bezier(0.4, 0, 1, 1);
}

.popup-fade-enter-from,
.popup-fade-leave-to {
  opacity: 0;
}

.popup-fade-enter-from > div {
  transform: scale(0.94) translateY(-12px);
  opacity: 0;
}

.popup-fade-leave-to > div {
  transform: scale(0.96) translateY(-8px);
  opacity: 0;
}

/* Scrollbar Styling */
.overflow-y-auto::-webkit-scrollbar {
  width: 8px;
}

.overflow-y-auto::-webkit-scrollbar-track {
  background: transparent;
}

.overflow-y-auto::-webkit-scrollbar-thumb {
  background: var(--nm-border);
  border-radius: 4px;
}

.dark .overflow-y-auto::-webkit-scrollbar-thumb {
  background: var(--nm-border);
}

.announcement-popup {
  background: var(--nm-bg);
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius);
}

.announcement-icon {
  background: var(--nm-accent-soft);
  color: var(--nm-accent-text);
  border: 1px solid var(--nm-accent);
  border-radius: var(--nm-radius-sm);
}

.announcement-dot,
.announcement-dot-ping {
  background: var(--nm-accent);
  border-radius: 999px;
}
</style>
