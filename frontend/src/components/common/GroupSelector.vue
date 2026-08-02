<template>
  <div>
    <label class="input-label">
      {{ t('admin.users.groups') }}
      <span class="group-selector-count font-normal">{{ t('common.selectedCount', { count: modelValue.length }) }}</span>
    </label>
    <div
      v-if="isSearchable"
      class="group-selector-search flex items-center gap-2 px-3 py-2"
    >
      <Icon name="search" size="sm" class="group-selector-search-icon shrink-0" />
      <input
        v-model="searchText"
        type="text"
        :placeholder="t('common.searchPlaceholder')"
        class="group-selector-search-input flex-1 bg-transparent text-sm focus:outline-none"
      />
    </div>
    <div
      :class="[
        'group-selector-panel grid max-h-32 grid-cols-2 gap-1 overflow-y-auto p-2',
        isSearchable ? 'group-selector-panel-searchable' : ''
      ]"
    >
      <label
        v-for="group in filteredGroups"
        :key="group.id"
        class="group-selector-option flex cursor-pointer items-center gap-2 px-2 py-1.5 transition-colors"
        :title="t('admin.groups.rateAndAccounts', { rate: group.rate_multiplier, count: group.account_count || 0 })"
      >
        <input
          type="checkbox"
          :value="group.id"
          :checked="modelValue.includes(group.id)"
          @change="handleChange(group.id, ($event.target as HTMLInputElement).checked)"
          class="group-selector-checkbox h-3.5 w-3.5 shrink-0"
        />
        <GroupBadge
          :name="group.name"
          :platform="group.platform"
          :subscription-type="group.subscription_type"
          :rate-multiplier="group.rate_multiplier"
          class="min-w-0 flex-1"
        />
        <span class="group-selector-account-count shrink-0 text-xs">{{ group.account_count || 0 }}</span>
      </label>
      <div
        v-if="filteredGroups.length === 0"
        class="group-selector-empty col-span-2 py-2 text-center text-sm"
      >
        {{ t('common.noGroupsAvailable') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import GroupBadge from './GroupBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AdminGroup, GroupPlatform } from '@/types'

const { t } = useI18n()

interface Props {
  modelValue: number[]
  groups: AdminGroup[]
  platform?: GroupPlatform // Optional platform filter
  mixedScheduling?: boolean // For antigravity accounts: allow anthropic/gemini groups
  searchable?: boolean | 'auto'
}

const props = withDefaults(defineProps<Props>(), {
  searchable: 'auto'
})
const emit = defineEmits<{
  'update:modelValue': [value: number[]]
}>()

const searchText = ref('')

const isSearchable = computed(() => {
  if (props.searchable === 'auto') return props.groups.length > 5
  return props.searchable
})

// Filter groups by platform if specified
const filteredGroups = computed(() => {
  let result: AdminGroup[] = props.groups
  if (props.platform) {
    // antigravity 账户启用混合调度后，可选择 anthropic/gemini 分组
    if (props.platform === 'antigravity' && props.mixedScheduling) {
      result = result.filter(
        (g) => g.platform === 'antigravity' || g.platform === 'anthropic' || g.platform === 'gemini' || g.platform === 'composite'
      )
    } else {
      // 默认：只能选择同 platform 的分组；composite 分组可接收任意具体平台账号
      result = result.filter((g) => g.platform === props.platform || g.platform === 'composite')
    }
  }
  if (isSearchable.value && searchText.value) {
    const q = searchText.value.toLowerCase()
    result = result.filter(
      (g) => g.name.toLowerCase().includes(q) || g.description?.toLowerCase().includes(q)
    )
  }
  return result
})

const handleChange = (groupId: number, checked: boolean) => {
  const newValue = checked
    ? [...props.modelValue, groupId]
    : props.modelValue.filter((id) => id !== groupId)
  emit('update:modelValue', newValue)
}
</script>

<style scoped>
.group-selector-count,
.group-selector-search-icon,
.group-selector-account-count,
.group-selector-empty {
  color: var(--nm-ink-faint);
}

.group-selector-search {
  border: 1px solid var(--nm-border);
  border-bottom: 0;
  border-radius: var(--nm-radius) var(--nm-radius) 0 0;
  background: var(--nm-surface-soft);
}

.group-selector-search-input {
  color: var(--nm-ink);
}

.group-selector-search-input::placeholder {
  color: var(--nm-ink-faint);
}

.group-selector-panel {
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius);
  background: var(--nm-surface-soft);
}

.group-selector-panel-searchable {
  border-top: 0;
  border-radius: 0 0 var(--nm-radius) var(--nm-radius);
}

.group-selector-option {
  border: 1px solid transparent;
  border-radius: var(--nm-radius-sm);
}

.group-selector-option:hover {
  background: var(--nm-surface);
  border-color: var(--nm-border-light);
}

.group-selector-checkbox {
  accent-color: var(--nm-accent);
}
</style>
