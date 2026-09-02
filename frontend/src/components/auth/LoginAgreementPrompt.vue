<template>
  <div
    v-if="mode === 'checkbox' && documents.length > 0"
    class="px-0.5"
  >
    <div class="flex items-start gap-2">
      <input
        id="login-agreement-consent"
        type="checkbox"
        :checked="accepted"
        class="mt-[2px] h-4 w-4 flex-shrink-0 rounded border-border"
        style="accent-color: var(--nm-accent)"
        @change="handleCheckboxChange"
      />
      <div class="min-w-0 flex-1">
        <p class="text-[13px] leading-5 text-muted-foreground">
          <label
            for="login-agreement-consent"
            class="cursor-pointer text-foreground"
          >
            {{ t('legal.loginAgreementPrompt.checkboxPrefix') }}
          </label>
          <template v-for="(doc, index) in documents" :key="doc.id || doc.title">
            <RouterLink
              :to="documentRoute(doc)"
              target="_blank"
              rel="noopener noreferrer"
              class="font-medium text-brand underline-offset-4 transition hover:text-[color:var(--nm-accent-strong)] hover:underline"
            >
              {{ doc.title }}
            </RouterLink>
            <span v-if="index < documents.length - 1">{{ t('legal.loginAgreementPrompt.documentSeparator') }}</span>
          </template>
        </p>
      </div>
    </div>
  </div>

  <div
    v-else-if="!accepted && documents.length > 0"
    class="card rounded-2xl p-3 text-sm"
    style="color: var(--nm-ink)"
  >
    <div class="flex items-start gap-3">
      <Icon name="shield" size="sm" class="mt-0.5 flex-shrink-0" style="color: var(--nm-accent-text)" />
      <div class="min-w-0 flex-1">
        <p class="font-medium">{{ t('legal.loginAgreementPrompt.noticeTitle') }}</p>
        <p class="mt-1 text-[color:var(--nm-accent-text)]">
          {{ t('legal.loginAgreementPrompt.noticeDescription') }}
        </p>
      </div>
      <button
        type="button"
        class="btn btn-sm flex-shrink-0"
        @click="emit('open')"
      >
        {{ t('legal.loginAgreementPrompt.viewTerms') }}
      </button>
    </div>
  </div>

  <Teleport to="body">
    <Transition name="agreement-fade">
      <div
        v-if="dialogVisible"
        class="fixed inset-0 z-[140] flex items-center justify-center overflow-y-auto p-4"
        style="background-color: rgba(28, 31, 38, 0.45)"
      >
        <div
          class="card w-full max-w-[600px] overflow-hidden rounded-2xl"
        >
          <div class="border-b px-6 py-6" style="border-color: var(--nm-border-light)">
            <div class="flex items-start gap-4">
              <span
                class="flex h-12 w-12 flex-shrink-0 items-center justify-center border"
                style="border-color: var(--nm-border); background: var(--nm-surface-soft); color: var(--nm-accent-text); border-radius: var(--nm-radius)"
              >
                <Icon name="shield" size="md" />
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <h2 class="text-xl font-bold tracking-normal text-foreground">
                    {{ t('legal.loginAgreementPrompt.dialogTitle') }}
                  </h2>
                  <span
                    v-if="updatedAt"
                    class="border px-2.5 py-1 text-xs font-medium"
                    style="border-color: var(--nm-border); background: var(--nm-surface-soft); color: var(--nm-ink-muted); border-radius: var(--nm-radius-sm)"
                  >
                    {{ updatedAt }}
                  </span>
                </div>
                <p class="mt-2 text-sm leading-6 text-muted-foreground">
                  {{
                    t('legal.loginAgreementPrompt.dialogDescription', {
                      date: updatedAt || t('legal.loginAgreementPrompt.recently'),
                    })
                  }}
                </p>
              </div>
            </div>
          </div>

          <div class="max-h-[58vh] overflow-y-auto px-6 py-5">
            <div class="mb-3 flex items-center justify-between gap-3">
              <p class="text-sm font-semibold text-foreground">{{ t('legal.loginAgreementPrompt.relatedDocuments') }}</p>
            </div>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <RouterLink
                v-for="(doc, index) in documents"
                :key="doc.id || doc.title"
                :to="documentRoute(doc)"
                target="_blank"
                rel="noopener noreferrer"
                class="agreement-doc-link group flex min-h-[72px] w-full items-center gap-3 border px-4 py-3 text-left transition"
                style="border-color: var(--nm-border); background: var(--nm-surface); border-radius: var(--nm-radius)"
              >
                <span
                  class="flex h-10 w-10 flex-shrink-0 items-center justify-center border transition"
                  style="border-color: var(--nm-border-light); background: var(--nm-bg); color: var(--nm-ink); border-radius: var(--nm-radius-sm)"
                >
                  <Icon :name="documentIcon(index, doc.title)" size="sm" />
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-sm font-semibold" style="color: var(--nm-ink)">{{ doc.title }}</span>
                </span>
                <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center transition" style="color: var(--nm-ink-muted)">
                  <Icon name="externalLink" size="sm" />
                </span>
              </RouterLink>
            </div>
          </div>

          <div class="border-t px-6 py-4" style="border-color: var(--nm-border-light); background: var(--nm-surface-soft)">
            <div class="grid grid-cols-2 gap-3">
              <button
                type="button"
                class="btn btn-secondary px-4 py-3"
                @click="emit('reject')"
              >
                {{ t('legal.loginAgreementPrompt.reject') }}
              </button>
              <button
                type="button"
                class="btn btn-primary px-4 py-3"
                @click="emit('accept')"
              >
                {{ t('legal.loginAgreementPrompt.accept') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { LoginAgreementDocument } from '@/types'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  accepted: boolean
  documents: LoginAgreementDocument[]
  mode: 'modal' | 'checkbox' | string
  updatedAt?: string
  visible: boolean
}>(), {
  updatedAt: ''
})

const emit = defineEmits<{
  accept: []
  reject: []
  open: []
}>()

const dialogVisible = computed(() => props.visible && documents.value.length > 0)
const documents = computed(() => props.documents.filter((doc) => doc.title.trim()))
const updatedAt = computed(() => props.updatedAt || '')
const accepted = computed(() => props.accepted)
const mode = computed(() => props.mode === 'checkbox' ? 'checkbox' : 'modal')

function documentRoute(doc: LoginAgreementDocument) {
  return {
    name: 'LegalDocument',
    params: {
      documentId: doc.id || doc.title,
    },
  }
}

function handleCheckboxChange(event: Event): void {
  const checked = (event.target as HTMLInputElement).checked
  if (checked) {
    emit('accept')
  } else {
    emit('reject')
  }
}

function documentIcon(index: number, title: string): 'document' | 'shield' | 'globe' | 'cog' {
  const normalizedTitle = title.toLowerCase()
  if (
    normalizedTitle.includes('policy') ||
    normalizedTitle.includes('privacy') ||
    title.includes('政策') ||
    title.includes('隐私')
  ) {
    return 'shield'
  }
  if (
    normalizedTitle.includes('country') ||
    normalizedTitle.includes('region') ||
    title.includes('国家') ||
    title.includes('地区')
  ) {
    return 'globe'
  }
  if (index === 3) {
    return 'cog'
  }
  return 'document'
}
</script>

<style scoped>
.agreement-fade-enter-active,
.agreement-fade-leave-active {
  transition: opacity 0.18s ease;
}

.agreement-fade-enter-from,
.agreement-fade-leave-to {
  opacity: 0;
}

.agreement-fade-enter-active > div,
.agreement-fade-leave-active > div {
  transition: transform 0.18s ease, opacity 0.18s ease;
}

.agreement-fade-enter-from > div,
.agreement-fade-leave-to > div {
  opacity: 0;
  transform: translateY(8px) scale(0.98);
}

.agreement-doc-link:hover {
  border-color: var(--nm-accent);
  background: var(--nm-surface-soft);
}
</style>
