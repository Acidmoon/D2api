<template>
  <div>
    <!-- 铃铛按钮 -->
    <button
      @click="openModal"
      class="announcement-trigger relative flex h-9 w-9 items-center justify-center"
      :class="{ 'announcement-trigger-active': unreadCount > 0 }"
      :aria-label="t('announcements.title')"
    >
      <Icon name="bell" size="md" />
      <!-- 未读红点 -->
      <span
        v-if="unreadCount > 0"
        class="absolute right-1 top-1 flex h-2 w-2"
      >
        <span class="announcement-dot-ping absolute inline-flex h-full w-full animate-ping opacity-75"></span>
        <span class="announcement-dot relative inline-flex h-2 w-2"></span>
      </span>
    </button>

    <!-- 公告列表 Modal -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div
          v-if="isModalOpen"
          class="fixed inset-0 z-[100] flex items-start justify-center overflow-y-auto p-4 pt-[8vh]"
          style="background-color: rgba(28, 31, 38, 0.45)"
          @click="closeModal"
        >
          <div
            class="announcement-panel w-full max-w-[620px] overflow-hidden"
            @click.stop
          >
            <!-- Header -->
            <div class="relative overflow-hidden border-b px-6 py-5" style="background: var(--nm-surface); border-color: var(--nm-border)">
              <div class="relative z-10 flex items-start justify-between">
                <div>
                  <div class="flex items-center gap-2">
                    <div class="announcement-icon flex h-8 w-8 items-center justify-center">
                      <Icon name="bell" size="sm" />
                    </div>
                    <h2 class="text-lg font-semibold" style="color: var(--nm-ink)">
                      {{ t('announcements.title') }}
                    </h2>
                  </div>
                  <p v-if="unreadCount > 0" class="mt-2 text-sm" style="color: var(--nm-ink-muted)">
                    <span class="font-medium" style="color: var(--nm-accent-text)">{{ unreadCount }}</span>
                    {{ t('announcements.unread') }}
                  </p>
                </div>
                <div class="flex items-center gap-2">
                  <button
                    v-if="unreadCount > 0"
                    @click="markAllAsRead"
                    :disabled="loading"
                    class="btn btn-primary btn-sm"
                  >
                    {{ t('announcements.markAllRead') }}
                  </button>
                  <button
                    @click="closeModal"
                    class="announcement-icon-button flex h-9 w-9 items-center justify-center"
                    :aria-label="t('common.close')"
                  >
                    <Icon name="x" size="sm" />
                  </button>
                </div>
              </div>
            </div>

            <!-- Body -->
            <div class="max-h-[65vh] overflow-y-auto">
              <!-- Loading -->
              <div v-if="loading" class="flex items-center justify-center py-16">
                <div class="announcement-spinner"></div>
              </div>

              <!-- Announcements List -->
              <div v-else-if="announcements.length > 0">
                <div
                  v-for="item in announcements"
                  :key="item.id"
                  class="announcement-row group relative flex items-center gap-4 border-b px-6 py-4"
                  :class="{ 'announcement-row-unread': !item.read_at }"
                  style="min-height: 72px"
                  @click="openDetail(item)"
                >
                  <!-- Status Indicator -->
                  <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center">
                    <div
                      v-if="!item.read_at"
                      class="announcement-state-icon announcement-state-icon-unread relative flex h-10 w-10 items-center justify-center"
                    >
                      <!-- Pulse ring -->
                      <span class="announcement-icon-ping absolute inline-flex h-full w-full animate-ping opacity-75"></span>
                      <!-- Icon -->
                      <svg class="relative z-10 h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </div>
                    <div
                      v-else
                      class="announcement-state-icon flex h-10 w-10 items-center justify-center"
                    >
                      <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </div>
                  </div>

                  <!-- Content -->
                  <div class="flex min-w-0 flex-1 items-center justify-between gap-4">
                    <div class="min-w-0 flex-1">
                      <h3 class="truncate text-sm font-medium" style="color: var(--nm-ink)">
                        {{ item.title }}
                      </h3>
                      <div class="mt-1 flex items-center gap-2">
                        <time class="text-xs" style="color: var(--nm-ink-muted)">
                          {{ formatRelativeTime(item.created_at) }}
                        </time>
                        <span
                          v-if="!item.read_at"
                          class="badge badge-primary inline-flex items-center gap-1"
                        >
                          <span class="relative flex h-1.5 w-1.5">
                            <span class="announcement-dot-ping absolute inline-flex h-full w-full animate-ping opacity-75"></span>
                            <span class="announcement-dot relative inline-flex h-1.5 w-1.5"></span>
                          </span>
                          {{ t('announcements.unread') }}
                        </span>
                      </div>
                    </div>

                    <!-- Arrow -->
                    <div class="flex-shrink-0">
                      <svg
                        class="announcement-chevron h-5 w-5 transition-transform group-hover:translate-x-1"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        stroke-width="2"
                      >
                        <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
                      </svg>
                    </div>
                  </div>

                  <!-- Unread indicator bar -->
                  <div
                    v-if="!item.read_at"
                    class="absolute left-0 top-0 h-full w-px"
                    style="background: var(--nm-accent)"
                  ></div>
                </div>
              </div>

              <!-- Empty State -->
              <div v-else class="flex flex-col items-center justify-center py-16">
                <div class="relative mb-4">
                  <div class="announcement-empty-icon flex h-20 w-20 items-center justify-center">
                    <Icon name="inbox" size="xl" />
                  </div>
                  <div class="announcement-empty-check absolute -right-1 -top-1 flex h-6 w-6 items-center justify-center">
                    <svg class="h-3.5 w-3.5" fill="currentColor" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                    </svg>
                  </div>
                </div>
                <p class="text-sm font-medium" style="color: var(--nm-ink)">{{ t('announcements.empty') }}</p>
                <p class="mt-1 text-xs" style="color: var(--nm-ink-muted)">{{ t('announcements.emptyDescription') }}</p>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 公告详情 Modal -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div
          v-if="detailModalOpen && selectedAnnouncement"
          class="fixed inset-0 z-[110] flex items-start justify-center overflow-y-auto p-4 pt-[6vh]"
          style="background-color: rgba(28, 31, 38, 0.45)"
          @click="closeDetail"
        >
          <div
            class="announcement-panel w-full max-w-[780px] overflow-hidden"
            @click.stop
          >
            <!-- Header -->
            <div class="relative overflow-hidden border-b px-8 py-6" style="background: var(--nm-surface); border-color: var(--nm-border)">
              <div class="relative z-10 flex items-start justify-between gap-4">
                <div class="flex-1 min-w-0">
                  <!-- Icon and Category -->
                  <div class="mb-3 flex items-center gap-2">
                    <div class="announcement-icon flex h-10 w-10 items-center justify-center">
                      <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </div>
                    <div class="flex items-center gap-2">
                      <span class="badge badge-primary">
                        {{ t('announcements.title') }}
                      </span>
                      <span
                        v-if="!selectedAnnouncement.read_at"
                        class="badge badge-primary inline-flex items-center gap-1.5"
                      >
                        <span class="relative flex h-2 w-2">
                          <span class="announcement-dot-ping absolute inline-flex h-full w-full animate-ping opacity-75"></span>
                          <span class="announcement-dot relative inline-flex h-2 w-2"></span>
                        </span>
                        {{ t('announcements.unread') }}
                      </span>
                    </div>
                  </div>

                  <!-- Title -->
                  <h2 class="mb-3 text-2xl font-bold leading-tight" style="color: var(--nm-ink)">
                    {{ selectedAnnouncement.title }}
                  </h2>

                  <!-- Meta Info -->
                  <div class="flex items-center gap-4 text-sm" style="color: var(--nm-ink-muted)">
                    <div class="flex items-center gap-1.5">
                      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      <time>{{ formatRelativeWithDateTime(selectedAnnouncement.created_at) }}</time>
                    </div>
                    <div class="flex items-center gap-1.5">
                      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        <path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                      </svg>
                      <span>{{ selectedAnnouncement.read_at ? t('announcements.read') : t('announcements.unread') }}</span>
                    </div>
                  </div>
                </div>

                <!-- Close button -->
                <button
                  @click="closeDetail"
                  class="announcement-icon-button flex h-10 w-10 flex-shrink-0 items-center justify-center"
                  :aria-label="t('common.close')"
                >
                  <Icon name="x" size="md" />
                </button>
              </div>
            </div>

            <!-- Body with Enhanced Markdown -->
            <div class="max-h-[60vh] overflow-y-auto px-8 py-8" style="background: var(--nm-bg)">
              <!-- Content with decorative border -->
              <div class="relative">
                <!-- Decorative left border -->
                <div class="absolute left-0 top-0 bottom-0 w-px" style="background: var(--nm-accent)"></div>

                <div class="pl-6">
                  <div
                    class="markdown-body prose prose-sm max-w-none dark:prose-invert"
                    v-html="renderMarkdown(selectedAnnouncement.content)"
                  ></div>
                </div>
              </div>
            </div>

            <!-- Footer with Actions -->
            <div class="border-t px-8 py-5" style="border-color: var(--nm-border-light); background: var(--nm-surface-soft)">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2 text-xs" style="color: var(--nm-ink-muted)">
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <span>{{ selectedAnnouncement.read_at ? t('announcements.readStatus') : t('announcements.markReadHint') }}</span>
                </div>
                <div class="flex items-center gap-3">
                  <button
                    @click="closeDetail"
                    class="btn btn-secondary"
                  >
                    {{ t('common.close') }}
                  </button>
                  <button
                    v-if="!selectedAnnouncement.read_at"
                    @click="markAsReadAndClose(selectedAnnouncement.id)"
                    class="btn btn-primary"
                  >
                    <span class="flex items-center gap-2">
                      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                      </svg>
                      {{ t('announcements.markRead') }}
                    </span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useAppStore } from '@/stores/app'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatRelativeTime, formatRelativeWithDateTime } from '@/utils/format'
import type { UserAnnouncement } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import '@/styles/announcement-markdown.css'

const { t } = useI18n()
const appStore = useAppStore()
const announcementStore = useAnnouncementStore()

// Configure marked
marked.setOptions({
  breaks: true,
  gfm: true,
})

// Use store state (storeToRefs for reactivity)
const { announcements, loading } = storeToRefs(announcementStore)
const unreadCount = computed(() => announcementStore.unreadCount)

// Local modal state
const isModalOpen = ref(false)
const detailModalOpen = ref(false)
const selectedAnnouncement = ref<UserAnnouncement | null>(null)

// Methods
function renderMarkdown(content: string): string {
  if (!content) return ''
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
}

function openModal() {
  isModalOpen.value = true
}

function closeModal() {
  isModalOpen.value = false
}

function openDetail(announcement: UserAnnouncement) {
  selectedAnnouncement.value = announcement
  detailModalOpen.value = true
  if (!announcement.read_at) {
    markAsRead(announcement.id)
  }
}

function closeDetail() {
  detailModalOpen.value = false
  selectedAnnouncement.value = null
}

async function markAsRead(id: number) {
  try {
    await announcementStore.markAsRead(id)
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
  }
}

async function markAsReadAndClose(id: number) {
  await markAsRead(id)
  appStore.showSuccess(t('announcements.markedAsRead'))
  closeDetail()
}

async function markAllAsRead() {
  try {
    await announcementStore.markAllAsRead()
    appStore.showSuccess(t('announcements.allMarkedAsRead'))
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
  }
}

function handleEscape(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (detailModalOpen.value) {
      closeDetail()
    } else if (isModalOpen.value) {
      closeModal()
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleEscape)
  document.body.style.overflow = ''
})

watch(
  [isModalOpen, detailModalOpen, () => announcementStore.currentPopup],
  ([modal, detail, popup]) => {
    document.body.style.overflow = (modal || detail || popup) ? 'hidden' : ''
  }
)
</script>

<style scoped>
.announcement-trigger {
  color: var(--nm-ink-muted);
  border: 1px solid transparent;
  border-radius: var(--nm-radius);
  transition: background-color 160ms ease, border-color 160ms ease, color 160ms ease;
}

.announcement-trigger:hover,
.announcement-trigger-active {
  color: var(--nm-accent-text);
  background: var(--nm-accent-soft);
  border-color: var(--nm-accent);
}

.announcement-panel {
  background: var(--nm-surface);
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius-lg);
}

.announcement-icon,
.announcement-state-icon-unread {
  color: var(--nm-accent-text);
  background: var(--nm-accent-soft);
  border: 1px solid var(--nm-accent);
  border-radius: var(--nm-radius);
}

.announcement-icon-button {
  color: var(--nm-ink-muted);
  background: var(--nm-surface-soft);
  border: 1px solid var(--nm-border-light);
  border-radius: var(--nm-radius);
  transition: background-color 160ms ease, border-color 160ms ease, color 160ms ease;
}

.announcement-icon-button:hover {
  color: var(--nm-ink);
  border-color: var(--nm-border);
}

.announcement-dot,
.announcement-dot-ping,
.announcement-icon-ping {
  background: var(--nm-accent);
  border-radius: 999px;
}

.announcement-row {
  cursor: pointer;
  border-color: var(--nm-border-light);
  color: var(--nm-ink);
  transition: background-color 160ms ease;
}

.announcement-row:hover,
.announcement-row-unread {
  background: var(--nm-surface-soft);
}

.announcement-state-icon {
  color: var(--nm-ink-faint);
  background: var(--nm-surface-soft);
  border: 1px solid var(--nm-border-light);
  border-radius: var(--nm-radius);
}

.announcement-chevron {
  color: var(--nm-ink-faint);
}

.announcement-empty-icon {
  color: var(--nm-ink-faint);
  background: var(--nm-surface-soft);
  border: 1px solid var(--nm-border-light);
  border-radius: var(--nm-radius-lg);
}

.announcement-empty-check {
  color: var(--nm-on-accent);
  background: var(--nm-success);
  border-radius: var(--nm-radius);
}

.announcement-spinner {
  width: 2.5rem;
  height: 2.5rem;
  border: 2px solid var(--nm-border-light);
  border-top-color: var(--nm-accent);
  border-radius: 999px;
  animation: announcement-spin 800ms linear infinite;
}

@keyframes announcement-spin {
  to {
    transform: rotate(360deg);
  }
}

.modal-fade-enter-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.modal-fade-leave-active {
  transition: all 0.2s cubic-bezier(0.4, 0, 1, 1);
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

.modal-fade-enter-from > div {
  transform: scale(0.94) translateY(-12px);
  opacity: 0;
}

.modal-fade-leave-to > div {
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

.overflow-y-auto::-webkit-scrollbar-thumb:hover {
  background: var(--nm-ink-faint);
}
</style>

<style>
.markdown-body {
  color: var(--nm-ink-muted);
  font-size: 15px;
  line-height: 1.75;
}

.markdown-body h1 {
  margin: 2rem 0 1.5rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--nm-border);
  color: var(--nm-ink);
  font-size: 1.875rem;
  font-weight: 700;
}

.markdown-body h2 {
  margin: 1.75rem 0 1rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--nm-border-light);
  color: var(--nm-ink);
  font-size: 1.5rem;
  font-weight: 700;
}

.markdown-body h3 {
  margin: 1.5rem 0 0.75rem;
  color: var(--nm-ink);
  font-size: 1.25rem;
  font-weight: 600;
}

.markdown-body h4 {
  margin: 1.25rem 0 0.5rem;
  color: var(--nm-ink);
  font-size: 1.125rem;
  font-weight: 600;
}

.markdown-body p {
  margin-bottom: 1rem;
  line-height: 1.75;
}

.markdown-body a {
  color: var(--nm-accent-text);
  font-weight: 600;
  text-decoration: underline;
  text-decoration-color: color-mix(in srgb, var(--nm-accent) 40%, transparent);
  text-decoration-thickness: 2px;
  text-underline-offset: 3px;
}

.markdown-body ul,
.markdown-body ol {
  margin: 0 0 1rem 1.5rem;
}

.markdown-body ul {
  list-style: disc;
}

.markdown-body ol {
  list-style: decimal;
}

.markdown-body li {
  padding-left: 0.5rem;
  line-height: 1.75;
}

.markdown-body li + li {
  margin-top: 0.5rem;
}

.markdown-body li::marker {
  color: var(--nm-accent-text);
}

.markdown-body blockquote {
  position: relative;
  margin: 1.25rem 0;
  padding: 0.75rem 1rem 0.75rem 1.25rem;
  border-left: 2px solid var(--nm-accent);
  background: var(--nm-accent-soft);
  color: var(--nm-ink);
  font-style: italic;
}

.markdown-body blockquote::before {
  content: '"';
  position: absolute;
  top: 0;
  left: -0.25rem;
  color: color-mix(in srgb, var(--nm-accent) 30%, transparent);
  font-family: serif;
  font-size: 3rem;
}

.markdown-body code {
  padding: 0.125rem 0.375rem;
  border: 1px solid var(--nm-border-light);
  border-radius: var(--nm-radius-sm);
  background: var(--nm-surface-soft);
  color: var(--nm-danger-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
}

.markdown-body pre {
  margin: 1.25rem 0;
  overflow-x: auto;
  padding: 1.25rem;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius);
  background: var(--nm-surface-soft);
}

.markdown-body pre code {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--nm-ink);
  font-size: 13px;
}

.markdown-body hr {
  margin: 2rem 0;
  border: 0;
  border-top: 1px solid var(--nm-border);
}

.markdown-body table {
  width: 100%;
  margin-bottom: 1.25rem;
  overflow: hidden;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius);
}

.markdown-body th,
.markdown-body td {
  padding: 0.75rem 1rem;
  border-right: 1px solid var(--nm-border-light);
  border-bottom: 1px solid var(--nm-border-light);
  text-align: left;
}

.markdown-body th:last-child,
.markdown-body td:last-child {
  @apply border-r-0;
}

.markdown-body tr:last-child td {
  @apply border-b-0;
}

.markdown-body th {
  color: var(--nm-ink);
  font-weight: 600;
  background: var(--nm-surface-soft);
}

.markdown-body tbody tr {
  transition: background-color 160ms ease;
}

.markdown-body tbody tr:hover {
  background: var(--nm-surface-soft);
}

.markdown-body img {
  max-width: 100%;
  margin: 1.25rem 0;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius);
}

.markdown-body strong {
  color: var(--nm-ink);
  font-weight: 600;
}

.markdown-body em {
  color: var(--nm-ink-muted);
  font-style: italic;
}
</style>
