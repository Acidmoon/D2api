<template>
  <div class="relative" ref="containerRef">
    <button
      ref="triggerRef"
      type="button"
      @click="toggle"
      :disabled="disabled"
      :aria-expanded="isOpen"
      :aria-haspopup="true"
      :id="id"
      :aria-label="ariaLabel ?? 'Select option'"
      :aria-describedby="ariaDescribedby"
      :class="[
        'select-trigger',
        isOpen && 'select-trigger-open',
        error && 'select-trigger-error',
        disabled && 'select-trigger-disabled'
      ]"
      @keydown.down.prevent="onTriggerKeyDown"
      @keydown.up.prevent="onTriggerKeyDown"
    >
      <span class="select-value">
        <slot name="selected" :option="selectedOption">
          {{ selectedLabel }}
        </slot>
      </span>
      <span
        v-if="clearable && hasValue && !disabled"
        class="select-clear"
        role="button"
        tabindex="-1"
        aria-label="Clear selection"
        @click.stop="clearSelection"
        @mousedown.stop
        @keydown.enter.stop.prevent="clearSelection"
      >
        <Icon name="x" size="sm" />
      </span>
      <span class="select-icon">
        <Icon
          name="chevronDown"
          size="md"
          :class="['transition-transform duration-200', isOpen && 'rotate-180']"
        />
      </span>
    </button>

    <!-- Teleport dropdown to body to escape stacking context -->
    <Teleport to="body">
      <Transition name="select-dropdown">
        <div
          v-if="isOpen"
          ref="dropdownRef"
          class="select-dropdown-portal"
          :class="[instanceId]"
          :style="dropdownStyle"
          role="listbox"
          @click.stop
          @mousedown.stop
          @keydown="onDropdownKeyDown"
        >
          <!-- Search input -->
          <div v-if="isSearchable" class="select-search">
            <Icon name="search" size="sm" class="select-search-icon" />
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="text"
              :placeholder="searchPlaceholderText"
              :aria-label="searchPlaceholderText"
              class="select-search-input"
              @click.stop
            />
          </div>

          <!-- Options list -->
          <div class="select-options" ref="optionsListRef">
            <div
              v-for="(option, index) in filteredOptions"
              :key="`${typeof getOptionValue(option)}:${String(getOptionValue(option) ?? '')}`"
              role="option"
              :aria-selected="isSelected(option)"
              :aria-disabled="isOptionDisabled(option)"
              @click.stop="!isOptionDisabled(option) && selectOption(option)"
              @mouseenter="handleOptionMouseEnter(option, index)"
              :class="[
                'select-option',
                isGroupHeaderOption(option) && 'select-option-group',
                isSelected(option) && 'select-option-selected',
                isOptionDisabled(option) && !isGroupHeaderOption(option) && 'select-option-disabled',
                focusedIndex === index && !isGroupHeaderOption(option) && 'select-option-focused'
              ]"
            >
              <slot name="option" :option="option" :selected="isSelected(option)">
                <Icon
                  v-if="option._creatable"
                  name="search"
                  size="sm"
                  class="select-search-icon flex-shrink-0"
                />
                <span class="select-option-label" :class="option._creatable && 'select-option-creatable'">{{ getOptionLabel(option) }}</span>
                <Icon
                  v-if="isSelected(option)"
                  name="check"
                  size="sm"
                  class="select-option-check"
                  :stroke-width="2"
                />
              </slot>
            </div>

            <!-- Empty state -->
            <div v-if="filteredOptions.length === 0" class="select-empty">
              {{ emptyTextDisplay }}
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

// Instance ID for unique click-outside detection
const instanceId = `select-${Math.random().toString(36).substring(2, 9)}`

export interface SelectOption {
  value: string | number | boolean | null
  label: string
  disabled?: boolean
  [key: string]: unknown
}

interface Props {
  modelValue: string | number | boolean | null | undefined
  options: SelectOption[] | Array<Record<string, unknown>>
  placeholder?: string
  disabled?: boolean
  error?: boolean
  searchable?: boolean | 'auto'
  searchPlaceholder?: string
  emptyText?: string
  valueKey?: string
  labelKey?: string
  creatable?: boolean
  creatablePrefix?: string
  clearable?: boolean
  id?: string
  ariaLabel?: string
  ariaDescribedby?: string
}

interface Emits {
  (e: 'update:modelValue', value: string | number | boolean | null): void
  (e: 'change', value: string | number | boolean | null, option: SelectOption | null): void
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  error: false,
  searchable: 'auto',
  creatable: false,
  creatablePrefix: '',
  clearable: false,
  valueKey: 'value',
  labelKey: 'label'
})

const emit = defineEmits<Emits>()

const isOpen = ref(false)
const searchQuery = ref('')
const focusedIndex = ref(-1)
const containerRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const optionsListRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<'bottom' | 'top'>('bottom')
const triggerRect = ref<DOMRect | null>(null)
const dropdownViewportPadding = 8
const dropdownMinimumWidth = 200

// i18n placeholders
const placeholderText = computed(() => props.placeholder ?? t('common.selectOption'))
const searchPlaceholderText = computed(() => props.searchPlaceholder ?? t('common.searchPlaceholder'))
const emptyTextDisplay = computed(() => props.emptyText ?? t('common.noOptionsFound'))

const isSearchable = computed(() => {
  if (props.searchable === 'auto') return props.options.length > 5
  return props.searchable
})

// Computed style for teleported dropdown
const dropdownStyle = computed(() => {
  if (!triggerRect.value) return {}

  const rect = triggerRect.value
  const viewportRight = Math.max(dropdownViewportPadding, window.innerWidth - dropdownViewportPadding)
  const left = Math.min(
    Math.max(dropdownViewportPadding, rect.left),
    viewportRight
  )
  const availableWidth = Math.max(0, viewportRight - left)
  const preferredMinWidth = Math.max(dropdownMinimumWidth, rect.width)
  const minWidth = Math.min(preferredMinWidth, availableWidth)
  const style: Record<string, string> = {
    position: 'fixed',
    left: `${left}px`,
    minWidth: `${minWidth}px`,
    maxWidth: `${availableWidth}px`,
    zIndex: '100000020'
  }

  if (dropdownPosition.value === 'top') {
    style.bottom = `${window.innerHeight - rect.top + 4}px`
  } else {
    style.top = `${rect.bottom + 4}px`
  }

  return style
})

const getOptionValue = (option: any): any => {
  if (typeof option === 'object' && option !== null) {
    return option[props.valueKey]
  }
  return option
}

const getOptionLabel = (option: any): string => {
  if (typeof option === 'object' && option !== null) {
    return String(option[props.labelKey] ?? '')
  }
  return String(option ?? '')
}

const isOptionDisabled = (option: any): boolean => {
  if (typeof option === 'object' && option !== null) {
    return !!option.disabled
  }
  return false
}

const isGroupHeaderOption = (option: any): boolean => {
  if (typeof option === 'object' && option !== null) {
    return option.kind === 'group'
  }
  return false
}

const selectedOption = computed(() => {
  return props.options.find((opt) => getOptionValue(opt) === props.modelValue) || null
})

const selectedLabel = computed(() => {
  if (selectedOption.value) {
    return getOptionLabel(selectedOption.value)
  }
  // In creatable mode, show the raw value if no matching option
  if (props.creatable && props.modelValue) {
    return String(props.modelValue)
  }
  return placeholderText.value
})

const hasValue = computed(
  () => props.modelValue !== null && props.modelValue !== undefined && props.modelValue !== ''
)

const filteredOptions = computed(() => {
  let opts = props.options as any[]
  if (isSearchable.value && searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    opts = opts.filter((opt) => {
      // Match label
      if (getOptionLabel(opt).toLowerCase().includes(query)) return true
      // Also match description if present
      if (opt.description && String(opt.description).toLowerCase().includes(query)) return true
      return false
    })
    // In creatable mode, always prepend a fuzzy search option
    if (props.creatable && searchQuery.value.trim()) {
      const trimmed = searchQuery.value.trim()
      const prefix = props.creatablePrefix || t('common.search')
      opts = [{ [props.valueKey]: trimmed, [props.labelKey]: `${prefix} "${trimmed}"`, _creatable: true }, ...opts]
    }
  }
  return opts
})

const isSelected = (option: any): boolean => {
  return getOptionValue(option) === props.modelValue
}

const findNextEnabledIndex = (startIndex: number): number => {
  const opts = filteredOptions.value
  if (opts.length === 0) return -1
  for (let offset = 0; offset < opts.length; offset++) {
    const idx = (startIndex + offset) % opts.length
    if (!isOptionDisabled(opts[idx])) return idx
  }
  return -1
}

const findPrevEnabledIndex = (startIndex: number): number => {
  const opts = filteredOptions.value
  if (opts.length === 0) return -1
  for (let offset = 0; offset < opts.length; offset++) {
    const idx = (startIndex - offset + opts.length) % opts.length
    if (!isOptionDisabled(opts[idx])) return idx
  }
  return -1
}

const handleOptionMouseEnter = (option: any, index: number) => {
  if (isOptionDisabled(option) || isGroupHeaderOption(option)) return
  focusedIndex.value = index
}

// Update trigger rect periodically while open to follow scroll/resize
const updateTriggerRect = () => {
  if (containerRef.value) {
    triggerRect.value = containerRef.value.getBoundingClientRect()
  }
}

const calculateDropdownPosition = () => {
  if (!containerRef.value) return
  updateTriggerRect()

  nextTick(() => {
    if (!dropdownRef.value || !triggerRect.value) return
    const dropdownHeight = dropdownRef.value.offsetHeight || 240
    const spaceBelow = window.innerHeight - triggerRect.value.bottom
    const spaceAbove = triggerRect.value.top

    if (spaceBelow < dropdownHeight && spaceAbove > dropdownHeight) {
      dropdownPosition.value = 'top'
    } else {
      dropdownPosition.value = 'bottom'
    }
  })
}

const toggle = () => {
  if (props.disabled) return
  isOpen.value = !isOpen.value
}

watch(isOpen, (open) => {
  if (open) {
    calculateDropdownPosition()
    // Reset focused index to current selection or first item
    if (filteredOptions.value.length === 0) {
      focusedIndex.value = -1
    } else {
      const selectedIdx = filteredOptions.value.findIndex(isSelected)
      const initialIdx = selectedIdx >= 0 ? selectedIdx : 0
      focusedIndex.value = isOptionDisabled(filteredOptions.value[initialIdx])
        ? findNextEnabledIndex(initialIdx + 1)
        : initialIdx
    }

    if (isSearchable.value) {
      nextTick(() => searchInputRef.value?.focus())
    }
    // Add scroll listener to update position
    window.addEventListener('scroll', updateTriggerRect, { capture: true, passive: true })
    window.addEventListener('resize', calculateDropdownPosition)
  } else {
    searchQuery.value = ''
    focusedIndex.value = -1
    window.removeEventListener('scroll', updateTriggerRect, { capture: true })
    window.removeEventListener('resize', calculateDropdownPosition)
  }
})

const selectOption = (option: any) => {
  const value = getOptionValue(option) ?? null
  emit('update:modelValue', value)
  emit('change', value, option)
  isOpen.value = false
  triggerRef.value?.focus()
}

const clearSelection = () => {
  if (props.disabled) return
  emit('update:modelValue', null)
  emit('change', null, null)
}

// Keyboards
const onTriggerKeyDown = () => {
  if (!isOpen.value) {
    isOpen.value = true
  }
}

const onDropdownKeyDown = (e: KeyboardEvent) => {
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      focusedIndex.value = findNextEnabledIndex(focusedIndex.value + 1)
      if (focusedIndex.value >= 0) scrollToFocused()
      break
    case 'ArrowUp':
      e.preventDefault()
      focusedIndex.value = findPrevEnabledIndex(focusedIndex.value - 1)
      if (focusedIndex.value >= 0) scrollToFocused()
      break
    case 'Enter':
      e.preventDefault()
      if (focusedIndex.value >= 0 && focusedIndex.value < filteredOptions.value.length) {
        const opt = filteredOptions.value[focusedIndex.value]
        if (!isOptionDisabled(opt)) selectOption(opt)
      }
      break
    case 'Escape':
      e.preventDefault()
      isOpen.value = false
      triggerRef.value?.focus()
      break
    case 'Tab':
      isOpen.value = false
      break
  }
}

const scrollToFocused = () => {
  nextTick(() => {
    const list = optionsListRef.value
    if (!list) return
    const focusedEl = list.children[focusedIndex.value] as HTMLElement
    if (!focusedEl) return

    if (focusedEl.offsetTop < list.scrollTop) {
      list.scrollTop = focusedEl.offsetTop
    } else if (focusedEl.offsetTop + focusedEl.offsetHeight > list.scrollTop + list.offsetHeight) {
      list.scrollTop = focusedEl.offsetTop + focusedEl.offsetHeight - list.offsetHeight
    }
  })
}

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // Check if click is inside THIS specific instance's dropdown or trigger
  const isInDropdown = !!target.closest(`.${instanceId}`)
  const isInTrigger = containerRef.value?.contains(target)

  if (!isInDropdown && !isInTrigger && isOpen.value) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('scroll', updateTriggerRect, { capture: true })
  window.removeEventListener('resize', calculateDropdownPosition)
})
</script>

<style scoped>
.select-trigger {
  display: flex;
  width: 100%;
  cursor: pointer;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  min-height: 2.75rem;
  padding: 0.625rem 1rem;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius);
  background: var(--nm-surface);
  color: var(--nm-ink);
  font-size: 0.875rem;
  transition: background-color 160ms ease, border-color 160ms ease, color 160ms ease;
}

.select-trigger-open {
  border-color: var(--nm-accent);
  background: var(--nm-accent-soft);
}

.select-trigger-error {
  border-color: var(--nm-danger);
}

.select-trigger-disabled {
  cursor: not-allowed;
  background: var(--nm-surface-soft);
  opacity: 0.6;
}

.select-value {
  flex: 1 1 0%;
  overflow: hidden;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.select-icon {
  flex-shrink: 0;
  color: var(--nm-ink-faint);
}

.select-clear {
  display: flex;
  flex-shrink: 0;
  cursor: pointer;
  align-items: center;
  justify-content: center;
  color: var(--nm-ink-faint);
  border-radius: var(--nm-radius-sm);
  transition: color 160ms ease;
}

.select-clear:hover {
  color: var(--nm-ink);
}
</style>

<style>
.select-dropdown-portal {
  width: max-content;
  min-width: 200px;
  overflow: hidden;
  background: var(--nm-surface);
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius-lg);
  pointer-events: auto !important;
}

.select-dropdown-portal .select-search {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-bottom: 1px solid var(--nm-border-light);
}

.select-dropdown-portal .select-search-icon,
.select-option-check {
  color: var(--nm-accent-text);
}

.select-dropdown-portal .select-search-input {
  flex: 1 1 0%;
  background: transparent;
  color: var(--nm-ink);
  font-size: 0.875rem;
  outline: none;
}

.select-dropdown-portal .select-search-input::placeholder {
  color: var(--nm-ink-faint);
}

.select-dropdown-portal .select-options {
  max-height: 20rem;
  overflow-y: auto;
  padding: 0.25rem 0;
  outline: none;
}

.select-dropdown-portal .select-option {
  display: flex;
  cursor: pointer;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
  color: var(--nm-ink-muted);
  font-size: 0.875rem;
  transition: background-color 150ms ease, color 150ms ease;
  pointer-events: auto !important;
}

.select-dropdown-portal .select-option:hover {
  background: var(--nm-surface-soft);
  color: var(--nm-ink);
}

.select-dropdown-portal .select-option-selected {
  background: var(--nm-accent-soft);
  color: var(--nm-accent-text);
}

.select-dropdown-portal .select-option-focused {
  background: var(--nm-surface-soft);
  color: var(--nm-ink);
}

.select-dropdown-portal .select-option-disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.select-dropdown-portal .select-option-group {
  cursor: default;
  user-select: none;
  background: var(--nm-surface-soft);
  color: var(--nm-ink-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0;
  text-transform: uppercase;
}

.select-dropdown-portal .select-option-group:hover {
  background: var(--nm-surface-soft);
}

.select-dropdown-portal .select-option-label {
  min-width: 0;
  flex: 1 1 0%;
  overflow: hidden;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.select-dropdown-portal .select-option-creatable {
  color: var(--nm-ink-muted);
  font-style: italic;
}

.select-dropdown-portal .select-empty {
  padding: 2rem 1rem;
  color: var(--nm-ink-muted);
  text-align: center;
  font-size: 0.875rem;
}

.select-dropdown-enter-active,
.select-dropdown-leave-active {
  transition: all 0.2s ease;
}

.select-dropdown-enter-from,
.select-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
