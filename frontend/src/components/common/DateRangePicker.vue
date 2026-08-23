<template>
  <div class="relative" ref="containerRef">
    <button
      type="button"
      @click="toggle"
      :class="['date-picker-trigger', isOpen && 'date-picker-trigger-open']"
    >
      <span class="date-picker-icon">
        <Icon name="calendar" size="sm" />
      </span>
      <span class="date-picker-value">
        {{ displayValue }}
      </span>
      <span class="date-picker-chevron">
        <Icon
          name="chevronDown"
          size="sm"
          :class="['transition-transform duration-200', isOpen && 'rotate-180']"
        />
      </span>
    </button>

    <Transition name="date-picker-dropdown">
      <div v-if="isOpen" class="date-picker-dropdown">
        <!-- Quick presets -->
        <div class="date-picker-presets">
          <button
            v-for="preset in presets"
            :key="preset.value"
            @click="selectPreset(preset)"
            :class="['date-picker-preset', isPresetActive(preset) && 'date-picker-preset-active']"
          >
            {{ t(preset.labelKey) }}
          </button>
        </div>

        <div class="date-picker-divider"></div>

        <!-- Custom date range inputs -->
        <div class="date-picker-custom">
          <div class="date-picker-field">
            <label class="date-picker-label">{{ t('dates.startDate') }}</label>
            <input
              type="date"
              v-model="localStartDate"
              :max="localEndDate || tomorrow"
              class="date-picker-input"
              @change="onDateChange"
            />
          </div>
          <div class="date-picker-separator">
            <Icon name="arrowRight" size="sm" />
          </div>
          <div class="date-picker-field">
            <label class="date-picker-label">{{ t('dates.endDate') }}</label>
            <input
              type="date"
              v-model="localEndDate"
              :min="localStartDate"
              :max="tomorrow"
              class="date-picker-input"
              @change="onDateChange"
            />
          </div>
        </div>

        <!-- Apply button -->
        <div class="date-picker-actions">
          <button @click="apply" class="date-picker-apply">
            {{ t('dates.apply') }}
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

interface DatePreset {
  labelKey: string
  value: string
  getRange: () => { start: string; end: string }
}

interface Props {
  startDate: string
  endDate: string
}

interface Emits {
  (e: 'update:startDate', value: string): void
  (e: 'update:endDate', value: string): void
  (e: 'change', range: { startDate: string; endDate: string; preset: string | null }): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t, locale } = useI18n()

const isOpen = ref(false)
const containerRef = ref<HTMLElement | null>(null)
const localStartDate = ref(props.startDate)
const localEndDate = ref(props.endDate)
const activePreset = ref<string | null>('last24Hours')

const today = computed(() => {
  // Use local timezone to avoid UTC timezone issues
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
})

// Tomorrow's date - used for max date to handle timezone differences
// When user is in a timezone behind the server, "today" on server might be "tomorrow" locally
const tomorrow = computed(() => {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  return formatDateToString(d)
})

// Helper function to format date to YYYY-MM-DD using local timezone
const formatDateToString = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const presets: DatePreset[] = [
  {
    labelKey: 'dates.today',
    value: 'today',
    getRange: () => {
      const t = today.value
      return { start: t, end: t }
    }
  },
  {
    labelKey: 'dates.yesterday',
    value: 'yesterday',
    getRange: () => {
      const d = new Date()
      d.setDate(d.getDate() - 1)
      const yesterday = formatDateToString(d)
      return { start: yesterday, end: yesterday }
    }
  },
  {
    labelKey: 'dates.last24Hours',
    value: 'last24Hours',
    getRange: () => {
      const end = new Date()
      const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
      return {
        start: formatDateToString(start),
        end: formatDateToString(end)
      }
    }
  },
  {
    labelKey: 'dates.last7Days',
    value: '7days',
    getRange: () => {
      const end = today.value
      const d = new Date()
      d.setDate(d.getDate() - 6)
      const start = formatDateToString(d)
      return { start, end }
    }
  },
  {
    labelKey: 'dates.last14Days',
    value: '14days',
    getRange: () => {
      const end = today.value
      const d = new Date()
      d.setDate(d.getDate() - 13)
      const start = formatDateToString(d)
      return { start, end }
    }
  },
  {
    labelKey: 'dates.last30Days',
    value: '30days',
    getRange: () => {
      const end = today.value
      const d = new Date()
      d.setDate(d.getDate() - 29)
      const start = formatDateToString(d)
      return { start, end }
    }
  },
  {
    labelKey: 'dates.thisMonth',
    value: 'thisMonth',
    getRange: () => {
      const now = new Date()
      const start = formatDateToString(new Date(now.getFullYear(), now.getMonth(), 1))
      return { start, end: today.value }
    }
  },
  {
    labelKey: 'dates.lastMonth',
    value: 'lastMonth',
    getRange: () => {
      const now = new Date()
      const start = formatDateToString(new Date(now.getFullYear(), now.getMonth() - 1, 1))
      const end = formatDateToString(new Date(now.getFullYear(), now.getMonth(), 0))
      return { start, end }
    }
  }
]

const displayValue = computed(() => {
  if (activePreset.value) {
    const preset = presets.find((p) => p.value === activePreset.value)
    if (preset) return t(preset.labelKey)
  }

  if (localStartDate.value && localEndDate.value) {
    if (localStartDate.value === localEndDate.value) {
      return formatDate(localStartDate.value)
    }
    return `${formatDate(localStartDate.value)} - ${formatDate(localEndDate.value)}`
  }

  return t('dates.selectDateRange')
})

const formatDate = (dateStr: string): string => {
  const date = new Date(dateStr + 'T00:00:00')
  const dateLocale = locale.value === 'zh' ? 'zh-CN' : 'en-US'
  return date.toLocaleDateString(dateLocale, { month: 'short', day: 'numeric' })
}

const isPresetActive = (preset: DatePreset): boolean => {
  return activePreset.value === preset.value
}

const selectPreset = (preset: DatePreset) => {
  const range = preset.getRange()
  localStartDate.value = range.start
  localEndDate.value = range.end
  activePreset.value = preset.value
}

const onDateChange = () => {
  // Check if current dates match any preset
  activePreset.value = null
  for (const preset of presets) {
    const range = preset.getRange()
    if (range.start === localStartDate.value && range.end === localEndDate.value) {
      activePreset.value = preset.value
      break
    }
  }
}

const toggle = () => {
  isOpen.value = !isOpen.value
}

const apply = () => {
  emit('update:startDate', localStartDate.value)
  emit('update:endDate', localEndDate.value)
  emit('change', {
    startDate: localStartDate.value,
    endDate: localEndDate.value,
    preset: activePreset.value
  })
  isOpen.value = false
}

const handleClickOutside = (event: MouseEvent) => {
  if (containerRef.value && !containerRef.value.contains(event.target as Node)) {
    isOpen.value = false
  }
}

const handleEscape = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && isOpen.value) {
    isOpen.value = false
  }
}

// Sync local state with props
watch(
  () => props.startDate,
  (val) => {
    localStartDate.value = val
    onDateChange()
  }
)

watch(
  () => props.endDate,
  (val) => {
    localEndDate.value = val
    onDateChange()
  }
)

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  document.addEventListener('keydown', handleEscape)
  // Initialize active preset detection
  onDateChange()
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('keydown', handleEscape)
})
</script>

<style scoped>
.date-picker-trigger {
  display: flex;
  cursor: pointer;
  align-items: center;
  gap: 0.5rem;
  min-height: 2.5rem;
  padding: 0.5rem 0.75rem;
  border: 1px solid hsl(var(--input));
  border-radius: var(--radius);
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  font-size: 0.875rem;
  box-shadow: 0 1px 2px hsl(var(--foreground) / 0.04);
  transition: background-color 160ms ease, border-color 160ms ease, color 160ms ease, box-shadow 160ms ease;
}

.date-picker-trigger:hover {
  border-color: hsl(var(--ring));
  color: hsl(var(--foreground));
}

.date-picker-trigger-open {
  border-color: hsl(var(--ring));
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  box-shadow: 0 0 0 2px hsl(var(--ring) / 0.2);
}

.date-picker-icon {
  color: hsl(var(--muted-foreground));
}

.date-picker-value {
  font-weight: 600;
}

.date-picker-chevron {
  color: hsl(var(--muted-foreground));
}

.date-picker-dropdown {
  position: absolute;
  left: 0;
  z-index: 100;
  min-width: 320px;
  margin-top: 0.5rem;
  overflow: hidden;
  background: hsl(var(--popover));
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  box-shadow: 0 10px 38px -10px rgb(22 23 24 / 0.35), 0 10px 20px -15px rgb(22 23 24 / 0.2);
}

.date-picker-presets {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.25rem;
  padding: 0.5rem;
}

.date-picker-preset {
  padding: 0.375rem 0.75rem;
  border: 1px solid transparent;
  border-radius: calc(var(--radius) - 2px);
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
  font-weight: 600;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease;
}

.date-picker-preset:hover {
  background: hsl(var(--accent));
  color: hsl(var(--accent-foreground));
}

.date-picker-preset-active {
  background: hsl(var(--brand) / 0.1);
  border-color: hsl(var(--brand) / 0.3);
  color: hsl(var(--brand));
}

.date-picker-divider {
  border-top: 1px solid hsl(var(--border));
}

.date-picker-custom {
  display: flex;
  align-items: flex-end;
  gap: 0.5rem;
  padding: 0.75rem;
}

.date-picker-field {
  flex: 1 1 0%;
}

.date-picker-label {
  display: block;
  margin-bottom: 0.25rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
  font-weight: 600;
}

.date-picker-input {
  width: 100%;
  padding: 0.375rem 0.5rem;
  border: 1px solid hsl(var(--input));
  border-radius: calc(var(--radius) - 2px);
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  font-size: 0.875rem;
  transition: border-color 160ms ease, box-shadow 160ms ease;
}

.date-picker-input:focus {
  border-color: hsl(var(--ring));
  outline: none;
  box-shadow: 0 0 0 2px hsl(var(--ring) / 0.2);
}

.date-picker-input::-webkit-calendar-picker-indicator {
  cursor: pointer;
  opacity: 0.65;
  filter: invert(0.5);
}

.date-picker-input::-webkit-calendar-picker-indicator:hover {
  opacity: 1;
}
.dark .date-picker-input::-webkit-calendar-picker-indicator {
  filter: none;
}

.date-picker-separator {
  display: flex;
  align-items: center;
  justify-content: center;
  padding-bottom: 0.25rem;
  color: hsl(var(--muted-foreground));
}

.date-picker-actions {
  display: flex;
  justify-content: flex-end;
  padding: 0 0.5rem 0.5rem;
}

.date-picker-apply {
  min-height: 2.25rem;
  padding: 0.375rem 1rem;
  border: 1px solid hsl(var(--primary));
  border-radius: var(--radius);
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
  font-size: 0.875rem;
  font-weight: 600;
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.1);
  transition: background-color 150ms ease, border-color 150ms ease, box-shadow 150ms ease;
}

.date-picker-apply:hover {
  background: hsl(var(--primary) / 0.9);
  border-color: hsl(var(--primary) / 0.9);
}

.date-picker-dropdown-enter-active,
.date-picker-dropdown-leave-active {
  transition: all 0.2s ease;
}

.date-picker-dropdown-enter-from,
.date-picker-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
