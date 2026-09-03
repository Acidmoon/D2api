<template>
  <div>
    <section
      data-testid="profile-overview-hero"
      class="card p-7"
    >
      <div class="flex flex-col gap-6 lg:flex-row lg:items-center">
        <div class="flex min-w-0 flex-1 items-center gap-5">
          <div
            class="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-full bg-gradient-to-br from-brand to-primary-400 text-xl font-bold text-white"
          >
            <img
              v-if="avatarUrl"
              :src="avatarUrl"
              :alt="displayName"
              class="h-full w-full object-cover"
            >
            <span v-else>{{ avatarInitial }}</span>
          </div>

          <div class="min-w-0 space-y-1.5">
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="truncate text-xl font-semibold text-foreground">
                {{ displayName }}
              </h2>
              <span :class="['badge', user?.role === 'admin' ? 'badge-primary' : 'badge-gray']">
                {{ user?.role === 'admin' ? t('profile.administrator') : t('profile.user') }}
              </span>
              <span
                :class="['badge', user?.status === 'active' ? 'badge-success' : 'badge-danger']"
              >
                {{
                  user?.status === 'active'
                    ? t('common.active')
                    : t('common.disabled')
                }}
              </span>
            </div>

            <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
              <div
                v-if="primaryEmailDisplay"
                class="flex items-center gap-1"
              >
                <p class="truncate text-sm text-[#8e96a7] dark:text-[color:var(--nm-ink-faint)]">
                  {{ primaryEmailDisplay }}
                </p>
                <button
                  type="button"
                  class="rounded-full p-1.5 text-[color:var(--nm-ink-faint)] transition-colors hover:bg-[color:var(--nm-surface-soft)] hover:text-foreground"
                  :title="t('common.copy')"
                  :aria-label="t('common.copy')"
                  @click="copyEmail"
                >
                  <Icon name="copy" size="sm" />
                </button>
              </div>
              <div
                v-if="userIdDisplay"
                class="flex items-center gap-1"
              >
                <p class="truncate text-sm text-[#8e96a7] dark:text-[color:var(--nm-ink-faint)]">
                  ID: {{ userIdDisplay }}
                </p>
                <button
                  type="button"
                  class="rounded-full p-1.5 text-[color:var(--nm-ink-faint)] transition-colors hover:bg-[color:var(--nm-surface-soft)] hover:text-foreground"
                  :title="t('common.copy')"
                  :aria-label="t('common.copy')"
                  @click="copyId"
                >
                  <Icon name="copy" size="sm" />
                </button>
              </div>
            </div>

            <div
              v-if="sourceHints.length"
              class="flex flex-wrap gap-2 text-xs text-[#8e96a7] dark:text-[color:var(--nm-ink-faint)]"
            >
              <span
                v-for="hint in sourceHints"
                :key="hint.key"
                class="inline-flex items-center gap-1 rounded-full bg-[color:var(--nm-surface-soft)] px-3 py-1"
              >
                <Icon name="link" size="sm" />
                {{ hint.text }}
              </span>
            </div>
          </div>
        </div>

        <dl class="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div
            data-testid="profile-overview-metric-balance"
            class="rounded-xl bg-[color:var(--nm-surface-soft)] px-4 py-3"
          >
            <p class="text-xs font-medium text-[#8e96a7] dark:text-[color:var(--nm-ink-faint)]">
              {{ t('profile.accountBalance') }}
            </p>
            <p class="mt-1 text-base font-semibold tabular-nums text-foreground">
              {{ formatCurrency(user?.balance || 0) }}
            </p>
          </div>
          <div
            data-testid="profile-overview-metric-concurrency"
            class="rounded-xl bg-[color:var(--nm-surface-soft)] px-4 py-3"
          >
            <p class="text-xs font-medium text-[#8e96a7] dark:text-[color:var(--nm-ink-faint)]">
              {{ t('profile.concurrencyLimit') }}
            </p>
            <p class="mt-1 text-base font-semibold tabular-nums text-foreground">
              {{ user?.concurrency || 0 }}
            </p>
          </div>
          <div
            data-testid="profile-overview-metric-member-since"
            class="rounded-xl bg-[color:var(--nm-surface-soft)] px-4 py-3"
          >
            <p class="text-xs font-medium text-[#8e96a7] dark:text-[color:var(--nm-ink-faint)]">
              {{ t('profile.memberSince') }}
            </p>
            <p class="mt-1 text-base font-semibold text-foreground">
              {{ memberSinceLabel }}
            </p>
          </div>
        </dl>
      </div>
    </section>

    <h2 class="my-6 text-lg font-semibold text-foreground">
      {{ t('profile.quickSettings') }}
    </h2>

    <div class="space-y-6">
      <div data-testid="profile-main-column">
        <section
          data-testid="profile-account-card"
          class="card p-6"
        >
          <div class="flex items-center gap-5">
            <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-[color:var(--nm-surface-soft)] text-foreground">
              <Icon name="user" size="lg" />
            </span>
            <div class="min-w-0 flex-1">
              <h3 class="text-xl font-semibold text-foreground">
                {{ t('profile.accountSectionTitle') }}
              </h3>
              <p class="mt-1 text-sm text-[#7f8798] dark:text-[color:var(--nm-ink-faint)]">
                {{ t('profile.accountSectionDescription') }}
              </p>
            </div>
            <Icon
              name="chevronRight"
              size="md"
              class="shrink-0 text-[color:var(--nm-ink-faint)]"
            />
          </div>

          <div
            data-testid="profile-basics-panel"
            class="mt-6 grid gap-4 md:grid-cols-2"
          >
            <div class="rounded-2xl bg-secondary/60 p-5">
              <ProfileAvatarCard
                :user="user"
                embedded
              />
            </div>

            <div class="rounded-2xl bg-secondary/60 p-5">
              <ProfileEditForm
                :initial-username="user?.username || ''"
                embedded
              />
            </div>
          </div>

          <div
            data-testid="profile-auth-bindings-panel"
            class="mt-6"
          >
            <ProfileIdentityBindingsSection
              :user="user"
              :linuxdo-enabled="linuxdoEnabled"
              :dingtalk-enabled="dingtalkEnabled"
              :oidc-enabled="oidcEnabled"
              :oidc-provider-name="oidcProviderName"
              :wechat-enabled="wechatEnabled"
              :wechat-open-enabled="wechatOpenEnabled"
              :wechat-mp-enabled="wechatMpEnabled"
              embedded
              compact
            />
          </div>
        </section>

        <div
          data-testid="profile-side-column"
          class="mt-6"
        >
          <section
            v-if="sourceHints.length"
            class="card p-6"
          >
            <h3 class="text-sm font-semibold text-foreground">
              {{ t('profile.linkedProfileSources') }}
            </h3>
            <p class="mt-0.5 text-xs text-[#7f8798] dark:text-[color:var(--nm-ink-faint)]">
              {{ t('profile.linkedProfileSourcesDescription') }}
            </p>

            <div class="mt-4 grid gap-2">
              <div
                v-for="hint in sourceHints"
                :key="hint.key"
                class="flex items-start gap-3 rounded-xl bg-secondary/60 px-4 py-3 text-sm text-muted-foreground"
              >
                <Icon name="link" size="sm" class="mt-0.5 shrink-0 text-muted-foreground" />
                <span>{{ hint.text }}</span>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import ProfileAvatarCard from '@/components/user/profile/ProfileAvatarCard.vue'
import ProfileEditForm from '@/components/user/profile/ProfileEditForm.vue'
import ProfileIdentityBindingsSection from '@/components/user/profile/ProfileIdentityBindingsSection.vue'
import { useClipboard } from '@/composables/useClipboard'
import type { User, UserAuthBindingStatus, UserAuthProvider, UserProfileSourceContext } from '@/types'

const props = withDefaults(defineProps<{
  user: User | null
  linuxdoEnabled?: boolean
  dingtalkEnabled?: boolean
  oidcEnabled?: boolean
  oidcProviderName?: string
  wechatEnabled?: boolean
  wechatOpenEnabled?: boolean
  wechatMpEnabled?: boolean
}>(), {
  linuxdoEnabled: false,
  dingtalkEnabled: false,
  oidcEnabled: false,
  oidcProviderName: 'OIDC',
  wechatEnabled: false,
  wechatOpenEnabled: undefined,
  wechatMpEnabled: undefined,
})

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

function copyEmail(): void {
  void copyToClipboard(primaryEmailDisplay.value)
}

const userIdDisplay = computed(() => {
  const id = props.user?.id
  return id === undefined || id === null ? '' : String(id)
})

function copyId(): void {
  void copyToClipboard(userIdDisplay.value)
}

function normalizeBindingStatus(binding: boolean | UserAuthBindingStatus | undefined): boolean | null {
  if (typeof binding === 'boolean') {
    return binding
  }
  if (!binding) {
    return null
  }
  if (typeof binding.bound === 'boolean') {
    return binding.bound
  }
  return Boolean(binding.provider_subject || binding.issuer || binding.provider_key)
}

function isEmailBound(user: User | null | undefined): boolean {
  if (typeof user?.email_bound === 'boolean') {
    return user.email_bound
  }

  const nested = user?.auth_bindings?.email ?? user?.identity_bindings?.email
  const normalized = normalizeBindingStatus(nested)
  return normalized ?? false
}

const avatarUrl = computed(() => props.user?.avatar_url?.trim() || '')
const displayName = computed(() => props.user?.username?.trim() || props.user?.email?.trim() || t('profile.user'))
const primaryEmailDisplay = computed(() => {
  const email = props.user?.email?.trim() || ''
  if (!email) {
    return ''
  }
  if (email.endsWith('.invalid') && !isEmailBound(props.user)) {
    return ''
  }
  return email
})
const avatarInitial = computed(() => displayName.value.charAt(0).toUpperCase() || 'U')
const memberSinceLabel = computed(() => {
  const raw = props.user?.created_at?.trim()
  if (!raw) {
    return '-'
  }

  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) {
    return '-'
  }

  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: 'short',
  }).format(date)
})

const providerLabels = computed<Record<UserAuthProvider, string>>(() => ({
  email: t('profile.authBindings.providers.email'),
  linuxdo: t('profile.authBindings.providers.linuxdo'),
  dingtalk: t('profile.authBindings.providers.dingtalk'),
  oidc: t('profile.authBindings.providers.oidc', { providerName: props.oidcProviderName }),
  wechat: t('profile.authBindings.providers.wechat'),
  github: 'GitHub',
  google: 'Google'
}))

function formatCurrency(value: number): string {
  return `$${value.toFixed(2)}`
}

function normalizeProvider(value: string): UserAuthProvider | null {
  const normalized = value.trim().toLowerCase()
  if (
    normalized === 'email' ||
    normalized === 'linuxdo' ||
    normalized === 'wechat' ||
    normalized === 'github' ||
    normalized === 'google'
  ) {
    return normalized
  }
  if (normalized === 'oidc' || normalized.startsWith('oidc:') || normalized.startsWith('oidc/')) {
    return 'oidc'
  }
  return null
}

function readObjectString(source: Record<string, unknown>, ...keys: string[]): string {
  for (const key of keys) {
    const value = source[key]
    if (typeof value === 'string' && value.trim()) {
      return value.trim()
    }
  }
  return ''
}

function resolveThirdPartySource(
  rawSource: string | UserProfileSourceContext | null | undefined
): { provider: UserAuthProvider; label: string } | null {
  if (!rawSource) {
    return null
  }

  if (typeof rawSource === 'string') {
    const provider = normalizeProvider(rawSource)
    if (!provider || provider === 'email') {
      return null
    }
    return {
      provider,
      label: providerLabels.value[provider]
    }
  }

  const sourceRecord = rawSource as Record<string, unknown>
  const provider = normalizeProvider(
    readObjectString(sourceRecord, 'provider', 'source', 'provider_type', 'auth_provider')
  )
  if (!provider || provider === 'email') {
    return null
  }

  const explicitLabel = readObjectString(
    sourceRecord,
    'provider_label',
    'label',
    'provider_name',
    'providerName'
  )

  return {
    provider,
    label: explicitLabel || providerLabels.value[provider]
  }
}

const sourceHints = computed(() => {
  const currentUser = props.user
  if (!currentUser) {
    return []
  }

  const hints: Array<{ key: string; text: string }> = []
  const avatarSource = resolveThirdPartySource(
    currentUser.profile_sources?.avatar ?? currentUser.avatar_source
  )
  const usernameSource = resolveThirdPartySource(
    currentUser.profile_sources?.username ??
      currentUser.profile_sources?.display_name ??
      currentUser.profile_sources?.nickname ??
      currentUser.display_name_source ??
      currentUser.username_source ??
      currentUser.nickname_source
  )

  if (avatarSource) {
    hints.push({
      key: 'avatar',
      text: t('profile.authBindings.source.avatar', { providerName: avatarSource.label })
    })
  }

  if (usernameSource) {
    hints.push({
      key: 'username',
      text: t('profile.authBindings.source.username', { providerName: usernameSource.label })
    })
  }

  return hints
})
</script>
