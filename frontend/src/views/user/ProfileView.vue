<template>
  <AppLayout>
    <div
      data-testid="profile-shell"
      class="space-y-6"
    >
      <header>
        <h1 class="text-[28px] font-semibold leading-tight tracking-tight text-foreground">
          {{ t('profile.title') }}
        </h1>
        <p class="mt-2 text-sm text-[#7f8798] dark:text-[color:var(--nm-ink-faint)]">
          {{ t('profile.description') }}
        </p>
      </header>

      <ProfileInfoCard
        :user="user"
        :linuxdo-enabled="linuxdoOAuthEnabled"
        :dingtalk-enabled="dingtalkOAuthEnabled"
        :oidc-enabled="oidcOAuthEnabled"
        :oidc-provider-name="oidcOAuthProviderName"
        :wechat-enabled="wechatOAuthEnabled"
        :wechat-open-enabled="wechatOAuthOpenEnabled"
        :wechat-mp-enabled="wechatOAuthMPEnabled"
      />

      <section
        v-if="contactInfo"
        class="card p-6"
      >
        <div class="flex items-center gap-5">
          <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-[color:var(--nm-surface-soft)] text-foreground">
            <Icon name="chat" size="lg" />
          </span>
          <div class="min-w-0">
            <h3 class="text-xl font-semibold text-foreground">
              {{ t('common.contactSupport') }}
            </h3>
            <p class="mt-1 break-all text-sm text-[#7f8798] dark:text-[color:var(--nm-ink-faint)]">
              {{ contactInfo }}
            </p>
          </div>
        </div>
      </section>

      <section class="card p-6">
        <div class="flex items-center gap-5">
          <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-[color:var(--nm-surface-soft)] text-foreground">
            <Icon name="lock" size="lg" />
          </span>
          <div class="min-w-0 flex-1">
            <h3 class="text-xl font-semibold text-foreground">
              {{ t('profile.changePassword') }}
            </h3>
            <p class="mt-1 text-sm text-[#7f8798] dark:text-[color:var(--nm-ink-faint)]">
              {{ t('profile.passwordDescription') }}
            </p>
          </div>
          <Icon
            name="chevronRight"
            size="md"
            class="shrink-0 text-[color:var(--nm-ink-faint)]"
          />
        </div>
        <div class="mt-6 max-w-md">
          <ProfilePasswordForm embedded />
        </div>
      </section>

      <ProfileBalanceNotifyCard
        v-if="user && balanceLowNotifyEnabled"
        :enabled="user.balance_notify_enabled ?? true"
        :threshold="user.balance_notify_threshold"
        :extra-emails="user.balance_notify_extra_emails ?? []"
        :system-default-threshold="systemDefaultThreshold"
        :user-email="user.email"
      />

      <ProfileTotpCard />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'
import AppLayout from '@/components/layout/AppLayout.vue'
import ProfileBalanceNotifyCard from '@/components/user/profile/ProfileBalanceNotifyCard.vue'
import ProfileInfoCard from '@/components/user/profile/ProfileInfoCard.vue'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import ProfileTotpCard from '@/components/user/profile/ProfileTotpCard.vue'
import { isWeChatWebOAuthEnabled } from '@/api/auth'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const user = computed(() => authStore.user)

const contactInfo = ref('')
const balanceLowNotifyEnabled = ref(false)
const systemDefaultThreshold = ref(0)
const linuxdoOAuthEnabled = ref(false)
const dingtalkOAuthEnabled = ref(false)
const wechatOAuthEnabled = ref(false)
const wechatOAuthOpenEnabled = ref<boolean | undefined>(undefined)
const wechatOAuthMPEnabled = ref<boolean | undefined>(undefined)
const oidcOAuthEnabled = ref(false)
const oidcOAuthProviderName = ref('OIDC')

onMounted(async () => {
  const profileRefresh = authStore.refreshUser().catch((error) => {
    console.error('Failed to refresh profile:', error)
  })

  const settingsLoad = appStore.fetchPublicSettings()
    .then((settings) => {
      if (!settings) {
        return
      }
      contactInfo.value = settings.contact_info || ''
      balanceLowNotifyEnabled.value = settings.balance_low_notify_enabled ?? false
      systemDefaultThreshold.value = settings.balance_low_notify_threshold ?? 0
      linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled ?? false
      dingtalkOAuthEnabled.value = settings.dingtalk_oauth_enabled ?? false
      wechatOAuthEnabled.value = isWeChatWebOAuthEnabled(settings)
      wechatOAuthOpenEnabled.value = typeof settings.wechat_oauth_open_enabled === 'boolean'
        ? settings.wechat_oauth_open_enabled
        : undefined
      wechatOAuthMPEnabled.value = typeof settings.wechat_oauth_mp_enabled === 'boolean'
        ? settings.wechat_oauth_mp_enabled
        : undefined
      oidcOAuthEnabled.value = settings.oidc_oauth_enabled ?? false
      oidcOAuthProviderName.value = settings.oidc_oauth_provider_name || 'OIDC'
    })
    .catch((error) => {
      console.error('Failed to load settings:', error)
    })

  await Promise.all([profileRefresh, settingsLoad])
})
</script>
