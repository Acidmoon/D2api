<template>
  <div v-if="groups && groups.length > 0" class="relative max-w-56">
    <!-- 分组容器：固定最大宽度，最多显示2行 -->
    <div class="flex flex-wrap gap-1 max-h-14 overflow-hidden">
      <GroupBadge
        v-for="group in displayGroups"
        :key="group.id"
        :name="group.name"
        :platform="group.platform"
        :subscription-type="group.subscription_type"
        :rate-multiplier="group.rate_multiplier"
        :show-rate="false"
        class="max-w-24"
      />
      <!-- 更多数量徽章 -->
      <button
        v-if="hiddenCount > 0"
        ref="moreButtonRef"
        type="button"
        aria-haspopup="dialog"
        :aria-expanded="showPopover"
        @click.stop="showPopover = !showPopover"
        class="groups-more-button"
      >
        <span>+{{ hiddenCount }}</span>
      </button>
    </div>

    <!-- Popover 显示完整列表 -->
    <Teleport to="body">
      <Transition
        enter-active-class="transition duration-150 ease-out"
        enter-from-class="opacity-0 scale-95"
        enter-to-class="opacity-100 scale-100"
        leave-active-class="transition duration-100 ease-in"
        leave-from-class="opacity-100 scale-100"
        leave-to-class="opacity-0 scale-95"
      >
        <div
          v-if="showPopover"
          ref="popoverRef"
          class="groups-popover"
          :style="popoverStyle"
        >
          <div class="mb-2 flex items-center justify-between">
            <span class="groups-popover-title">
              {{ t('admin.accounts.groupCountTotal', { count: groups.length }) }}
            </span>
            <button
              type="button"
              :aria-label="t('common.close')"
              @click="showPopover = false"
              class="groups-close-button"
            >
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <div class="flex flex-wrap gap-1.5 max-h-64 overflow-y-auto">
            <GroupBadge
              v-for="group in groups"
              :key="group.id"
              :name="group.name"
              :platform="group.platform"
              :subscription-type="group.subscription_type"
              :rate-multiplier="group.rate_multiplier"
              :show-rate="false"
            />
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 点击外部关闭 popover -->
    <div
      v-if="showPopover"
      class="fixed inset-0 z-40"
      @click="showPopover = false"
    />
  </div>
  <span v-else class="groups-empty">-</span>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import GroupBadge from '@/components/common/GroupBadge.vue'
import type { Group } from '@/types'

interface Props {
  groups: Group[] | null | undefined
  maxDisplay?: number
}

const props = withDefaults(defineProps<Props>(), {
  maxDisplay: 4
})

const { t } = useI18n()

const moreButtonRef = ref<HTMLElement | null>(null)
const popoverRef = ref<HTMLElement | null>(null)
const showPopover = ref(false)

// 显示的分组（最多显示 maxDisplay 个）
const displayGroups = computed(() => {
  if (!props.groups) return []
  if (props.groups.length <= props.maxDisplay) {
    return props.groups
  }
  // 留一个位置给 +N 按钮
  return props.groups.slice(0, props.maxDisplay - 1)
})

// 隐藏的数量
const hiddenCount = computed(() => {
  if (!props.groups) return 0
  if (props.groups.length <= props.maxDisplay) return 0
  return props.groups.length - (props.maxDisplay - 1)
})

// Popover 位置样式
const popoverStyle = computed(() => {
  if (!moreButtonRef.value) return {}
  const rect = moreButtonRef.value.getBoundingClientRect()
  const viewportHeight = window.innerHeight
  const viewportWidth = window.innerWidth

  let top = rect.bottom + 8
  let left = rect.left

  // 如果下方空间不足，显示在上方
  if (top + 280 > viewportHeight) {
    top = Math.max(8, rect.top - 280)
  }

  // 如果右侧空间不足，向左偏移
  if (left + 384 > viewportWidth) {
    left = Math.max(8, viewportWidth - 392)
  }

  return {
    top: `${top}px`,
    left: `${left}px`
  }
})

// 关闭 popover 的键盘事件
const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    showPopover.value = false
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.groups-more-button {
  display: inline-flex;
  cursor: pointer;
  align-items: center;
  gap: 0.125rem;
  white-space: nowrap;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius-sm);
  background: var(--nm-surface);
  padding: 0.125rem 0.375rem;
  color: var(--nm-ink-muted);
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.2;
  transition: background-color 160ms ease, border-color 160ms ease, color 160ms ease;
}

.groups-more-button:hover {
  border-color: var(--nm-ink-muted);
  background: var(--nm-surface-soft);
  color: var(--nm-ink);
}

.groups-popover {
  position: fixed;
  z-index: 50;
  min-width: 12rem;
  max-width: 24rem;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius-lg);
  background: var(--nm-surface);
  padding: 0.75rem;
}

.groups-popover-title,
.groups-empty {
  color: var(--nm-ink-faint);
  font-size: 0.75rem;
}

.groups-popover-title {
  font-weight: 600;
}

.groups-empty {
  font-size: 0.875rem;
}

.groups-close-button {
  border-radius: var(--nm-radius-sm);
  padding: 0.125rem;
  color: var(--nm-ink-faint);
  transition: background-color 160ms ease, color 160ms ease;
}

.groups-close-button:hover {
  background: var(--nm-surface-soft);
  color: var(--nm-ink);
}
</style>
