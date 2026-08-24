<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.editUser')"
    width="normal"
    @close="$emit('close')"
  >
    <form v-if="user" id="edit-user-form" @submit.prevent="handleUpdateUser" class="space-y-5">
      <div>
        <label class="input-label">{{ t('admin.users.email') }}</label>
        <input v-model="form.email" type="email" class="input" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.password') }}</label>
        <div class="flex gap-2">
          <div class="relative flex-1">
            <input v-model="form.password" type="text" class="input pr-10" :placeholder="t('admin.users.enterNewPassword')" />
            <button v-if="form.password" type="button" @click="copyPassword" class="absolute right-2 top-1/2 -translate-y-1/2 rounded-lg p-1 transition-colors hover:bg-gray-100 dark:hover:bg-dark-700" :class="passwordCopied ? 'text-green-500' : 'text-gray-400'">
              <svg v-if="passwordCopied" class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg>
              <svg v-else class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" /></svg>
            </button>
          </div>
          <button type="button" @click="generatePassword" class="btn btn-secondary px-3">
            <Icon name="refresh" size="md" />
          </button>
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.username') }}</label>
        <input v-model="form.username" type="text" class="input" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.form.roleLabel') }}</label>
        <Select
          v-model="form.role"
          :options="roleOptions"
          :searchable="false"
        />
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.notes') }}</label>
        <textarea v-model="form.notes" rows="3" class="input"></textarea>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.columns.concurrency') }}</label>
        <input
          v-model.number="form.concurrency"
          type="number"
          min="0"
          step="1"
          class="input"
          :placeholder="t('admin.users.form.concurrencyPlaceholder')"
          data-test="concurrency-input"
        />
        <p class="input-hint">{{ t('admin.users.form.concurrencyHint') }}</p>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.form.rpmLimit') }}</label>
        <input
          v-model.number="form.rpm_limit"
          type="number"
          min="0"
          step="1"
          class="input"
          :placeholder="t('admin.users.form.rpmLimitPlaceholder')"
        />
        <p class="input-hint">{{ t('admin.users.form.rpmLimitHint') }}</p>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.form.rechargeMultiplier') }}</label>
        <input
          v-model.number="form.recharge_multiplier"
          type="number"
          min="0"
          step="0.01"
          class="input"
          :placeholder="t('admin.users.form.rechargeMultiplierPlaceholder')"
        />
        <p class="input-hint">{{ t('admin.users.form.rechargeMultiplierHint') }}</p>
      </div>
      <UserAttributeForm v-model="form.customAttributes" :user-id="user?.id" />
      <div v-if="!violationBan.loading" class="rounded-lg border px-4 py-3 text-sm" :class="violationBan.banned ? 'border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950/30' : 'border-gray-200 dark:border-dark-700'">
        <template v-if="violationBan.banned">
          <p class="font-medium text-red-700 dark:text-red-300">{{ t('admin.users.violationBan.bannedTitle') }}</p>
          <p class="mt-1 text-red-600 dark:text-red-200">{{ t('admin.users.violationBan.bannedHint', { until: formatBanUntil(violationBan.until) }) }}</p>
          <button type="button" class="btn btn-secondary btn-sm mt-2" :disabled="violationBan.clearing" data-test="clear-violation-ban" @click="handleClearViolationBan">
            {{ violationBan.clearing ? t('admin.users.violationBan.clearing') : t('admin.users.violationBan.clear') }}
          </button>
        </template>
        <template v-else>
          <p class="font-medium text-gray-800 dark:text-dark-100">{{ t('admin.users.violationBan.manualTitle') }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.users.violationBan.manualHint') }}</p>
          <div class="mt-2 flex flex-wrap items-center gap-2">
            <select v-model="manualBanDuration" class="input w-auto text-sm" :aria-label="t('admin.users.violationBan.duration')" data-test="manual-ban-duration">
              <option :value="10">{{ t('admin.users.violationBan.durations.10m') }}</option>
              <option :value="30">{{ t('admin.users.violationBan.durations.30m') }}</option>
              <option :value="60">{{ t('admin.users.violationBan.durations.1h') }}</option>
              <option :value="1440">{{ t('admin.users.violationBan.durations.1d') }}</option>
              <option :value="0">{{ t('admin.users.violationBan.durations.custom') }}</option>
            </select>
            <input
              v-if="manualBanDuration === 0"
              v-model.number="manualBanCustomMinutes"
              type="number"
              min="1"
              max="10080"
              class="input w-32 text-sm"
              :placeholder="t('admin.users.violationBan.customMinutes')"
              :aria-label="t('admin.users.violationBan.customMinutes')"
              data-test="manual-ban-custom-minutes"
            />
            <button type="button" class="btn btn-secondary btn-sm" :disabled="violationBan.banning" data-test="manual-ban-submit" @click="handleManualViolationBan">
              {{ violationBan.banning ? t('admin.users.violationBan.banning') : t('admin.users.violationBan.manualBan') }}
            </button>
          </div>
        </template>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" type="button" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button type="submit" form="edit-user-form" :disabled="submitting" class="btn btn-primary">
          {{ submitting ? t('admin.users.updating') : t('common.update') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <!-- 角色提升为管理员时后端要求 step-up 2FA，弹出 TOTP 验证后自动重试 -->
  <TotpStepUpDialog :controller="stepUp" />
</template>

<script setup lang="ts">
import { computed, ref, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { adminAPI } from '@/api/admin'
import type { AdminUser, UserAttributeValuesMap } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import UserAttributeForm from '@/components/user/UserAttributeForm.vue'
import Icon from '@/components/icons/Icon.vue'
import { useStepUp, isStepUpBlocked, isStepUpCancelled, stepUpBlockReason } from '@/composables/useStepUp'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'

const props = defineProps<{ show: boolean, user: AdminUser | null }>()
const emit = defineEmits(['close', 'success'])
const { t } = useI18n(); const appStore = useAppStore(); const { copyToClipboard } = useClipboard()

const submitting = ref(false); const passwordCopied = ref(false)
const form = reactive({ email: '', password: '', username: '', notes: '', role: 'user' as AdminUser['role'], concurrency: 1, rpm_limit: 0, recharge_multiplier: '' as string | number, customAttributes: {} as UserAttributeValuesMap })

// 用户内容违规临时封禁状态（独立于表单，只读展示 + 解除操作）
const violationBan = reactive<{ loading: boolean; banned: boolean; until: string | null; clearing: boolean; banning: boolean }>({ loading: false, banned: false, until: null, clearing: false, banning: false })
const manualBanDuration = ref(30)
const manualBanCustomMinutes = ref(60)
const handleManualViolationBan = async () => {
  if (!props.user || violationBan.banning) return
  const minutes = manualBanDuration.value === 0 ? manualBanCustomMinutes.value : manualBanDuration.value
  if (!Number.isInteger(minutes) || minutes < 1 || minutes > 10080) {
    appStore.showError(t('admin.users.violationBan.invalidDuration'))
    return
  }
  violationBan.banning = true
  try {
    const status = await adminAPI.users.banUser(props.user.id, minutes)
    violationBan.banned = status.banned
    violationBan.until = status.until ?? null
    appStore.showSuccess(t('admin.users.violationBan.banApplied'))
  } catch (e: any) {
    appStore.showError(e?.message || t('admin.users.violationBan.banFailed'))
  } finally { violationBan.banning = false }
}
const loadViolationBan = async (userId: number) => {
  violationBan.loading = true
  violationBan.banned = false
  violationBan.until = null
  try {
    const status = await adminAPI.users.getViolationBan(userId)
    violationBan.banned = status.banned
    violationBan.until = status.until ?? null
  } catch {
    // 查询失败不影响编辑表单主流程
  } finally { violationBan.loading = false }
}
const formatBanUntil = (until: string | null): string => {
  if (!until) return '-'
  const date = new Date(until)
  return Number.isNaN(date.getTime()) ? until : date.toLocaleString()
}
const handleClearViolationBan = async () => {
  if (!props.user || violationBan.clearing) return
  violationBan.clearing = true
  try {
    await adminAPI.users.clearViolationBan(props.user.id)
    violationBan.banned = false
    violationBan.until = null
    appStore.showSuccess(t('admin.users.violationBan.cleared'))
  } catch (e: any) {
    appStore.showError(e?.message || t('admin.users.violationBan.clearFailed'))
  } finally { violationBan.clearing = false }
}
const roleOptions = computed(() => [
  { value: 'user', label: t('admin.users.roles.user') },
  { value: 'admin', label: t('admin.users.roles.admin') }
])

watch(() => props.user, (u) => {
  if (u) {
    Object.assign(form, { email: u.email, password: '', username: u.username || '', notes: u.notes || '', role: u.role || 'user', concurrency: u.concurrency, rpm_limit: u.rpm_limit ?? 0, recharge_multiplier: u.recharge_multiplier ?? '', customAttributes: {} })
    passwordCopied.value = false
    void loadViolationBan(u.id)
  }
}, { immediate: true })

const generatePassword = () => {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#$%^&*'
  let p = ''; for (let i = 0; i < 16; i++) p += chars.charAt(Math.floor(Math.random() * chars.length))
  form.password = p
}
const copyPassword = async () => {
  if (form.password && await copyToClipboard(form.password, t('admin.users.passwordCopied'))) {
    passwordCopied.value = true; setTimeout(() => passwordCopied.value = false, 2000)
  }
}
const stepUp = useStepUp()

const handleUpdateUser = async () => {
  if (!props.user) return
  if (!form.email.trim()) {
    appStore.showError(t('admin.users.emailRequired'))
    return
  }
  // 0 = 不限制，与网关 (AcquireUserSlot: maxConcurrency <= 0) 和批量改限额一致
  if (!Number.isInteger(form.concurrency) || form.concurrency < 0) {
    appStore.showError(t('admin.users.concurrencyNonNegative'))
    return
  }
  // 充值倍率：留空 = 清除用户级覆盖（跟随分组/全局）；填写时必须 > 0
  if (form.recharge_multiplier !== '' && !(Number(form.recharge_multiplier) > 0)) {
    appStore.showError(t('admin.users.form.rechargeMultiplierInvalid'))
    return
  }
  const userId = props.user.id
  submitting.value = true
  try {
    const data: any = { email: form.email, username: form.username, notes: form.notes, role: form.role, concurrency: form.concurrency, rpm_limit: form.rpm_limit, recharge_multiplier: form.recharge_multiplier === '' ? null : Number(form.recharge_multiplier) }
    if (form.password.trim()) data.password = form.password.trim()
    // 提升为管理员属敏感操作：后端返回 STEP_UP_REQUIRED 时弹 TOTP 验证并重试
    await stepUp.run(() => adminAPI.users.update(userId, data))
    if (Object.keys(form.customAttributes).length > 0) await adminAPI.userAttributes.updateUserAttributeValues(userId, form.customAttributes)
    appStore.showSuccess(t('admin.users.userUpdated'))
    emit('success'); emit('close')
  } catch (e: any) {
    if (isStepUpCancelled(e)) {
      // 用户主动取消二次验证：静默返回，表单保持打开。
    } else if (isStepUpBlocked(e)) {
      appStore.showError(
        stepUpBlockReason(e) === 'STEP_UP_ADMIN_API_KEY_FORBIDDEN'
          ? t('stepUp.adminApiKeyForbidden')
          : t('stepUp.notEnabled')
      )
    } else {
      appStore.showError(e?.message || t('admin.users.failedToUpdate'))
    }
  } finally { submitting.value = false }
}
</script>
