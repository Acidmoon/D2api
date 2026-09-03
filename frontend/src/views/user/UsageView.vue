<template>
  <AppLayout>
    <div class="space-y-3">
      <!-- In-content page header -->
      <header class="mb-6">
        <h1 class="text-[28px] font-semibold leading-tight tracking-tight text-foreground">
          {{ t('usage.title') }}
        </h1>
        <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1">
          <p class="qw-desc text-sm">{{ t('usage.description') }}</p>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="qw-link inline-flex items-center gap-1 text-[13px] font-medium"
          >
            <Icon name="book" size="sm" />
            <span>{{ t('usage.quickStart') }}</span>
            <Icon name="externalLink" size="xs" />
          </a>
          <a
            href="/keys"
            class="qw-link inline-flex items-center gap-1 text-[13px] font-medium"
            @click="goKeys"
          >
            <Icon name="key" size="sm" />
            <span>{{ t('usage.getApiKey') }}</span>
          </a>
        </div>
      </header>

      <!-- Segmented tabs (usage / error requests) -->
      <div
        v-if="errorViewEnabled"
        class="flex w-fit items-center gap-1 rounded-full bg-[color:var(--nm-surface)] p-1"
        role="tablist"
      >
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'usage'"
          class="qw-tab"
          :class="{ 'qw-tab-active': activeTab === 'usage' }"
          @click="activeTab = 'usage'"
        >
          {{ t('usage.tabs.usage') }}
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'errors'"
          class="qw-tab"
          :class="{ 'qw-tab-active': activeTab === 'errors' }"
          @click="switchToErrors"
        >
          {{ t('usage.tabs.errors') }}
        </button>
      </div>

      <!-- Overview row: usage / trend (left) + account summary (right) -->
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-[2fr_1fr]">
        <section class="card p-7">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <h2 class="text-xl font-semibold text-foreground">{{ t('usage.overviewTitle') }}</h2>
            <div class="qw-weak flex items-center gap-1 text-xs">
              <span>{{ t('usage.lastUpdated', { time: lastUpdatedText }) }}</span>
              <button
                type="button"
                class="flex h-8 w-8 items-center justify-center rounded-full text-[color:var(--nm-ink-faint)] transition-colors hover:bg-[color:var(--nm-surface-soft)] hover:text-foreground"
                :aria-label="t('common.refresh')"
                @click="refreshData"
              >
                <Icon name="refresh" size="sm" />
              </button>
            </div>
          </div>
          <div class="qw-flat mt-5">
            <UsageStatsCards :stats="usageStats" :show-account-cost="false" :strike-standard-cost="true" />
          </div>
          <div class="mt-2">
            <TokenUsageTrend :trend-data="trendData" :loading="chartsLoading" embedded />
          </div>
        </section>

        <section class="card p-7">
          <h2 class="text-xl font-semibold text-foreground">{{ t('usage.summaryTitle') }}</h2>
          <dl class="mt-4 divide-y divide-[color:var(--nm-border-light)]">
            <div class="flex items-center justify-between gap-4 py-3.5">
              <dt class="qw-weak text-xs">{{ t('usage.totalRequests') }}</dt>
              <dd class="text-2xl font-semibold tabular-nums text-foreground">
                {{ (usageStats?.total_requests ?? 0).toLocaleString() }}
              </dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3.5">
              <dt class="qw-weak text-xs">{{ t('usage.totalTokens') }}</dt>
              <dd class="text-2xl font-semibold tabular-nums text-foreground">
                {{ (usageStats?.total_tokens ?? 0).toLocaleString() }}
              </dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3.5">
              <dt class="qw-weak text-xs">{{ t('usage.rpmLabel') }}</dt>
              <dd class="text-2xl font-semibold tabular-nums text-foreground">
                {{ formatRate(requestsPerMinute) }}
              </dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3.5">
              <dt class="qw-weak text-xs">{{ t('usage.tpmLabel') }}</dt>
              <dd class="text-2xl font-semibold tabular-nums text-foreground">
                {{ formatRate(tokensPerMinute) }}
              </dd>
            </div>
          </dl>
        </section>
      </div>

      <!-- Usage details -->
      <section class="card p-7">
        <h2 class="text-xl font-semibold text-foreground">{{ t('usage.detailTitle') }}</h2>

        <!-- Filter row: date range + granularity + actions -->
        <div class="mt-5 flex flex-wrap items-center gap-3">
          <div class="qw-pill w-full sm:w-[17rem]">
            <DateRangePicker
              v-model:start-date="startDate"
              v-model:end-date="endDate"
              @change="onDateRangeChange"
            />
          </div>
          <div class="qw-pill w-32">
            <Select v-model="granularity" :options="granularityOptions" @change="loadChartData" />
          </div>

          <div class="ml-auto flex flex-wrap items-center gap-2">
            <button
              type="button"
              class="qw-btn"
              :disabled="activeTab === 'errors' ? errorLoading : loading"
              @click="refreshData"
            >
              {{ t('common.refresh') }}
            </button>
            <button type="button" class="qw-btn" @click="resetFilters">
              {{ t('common.reset') }}
            </button>
            <div class="relative" ref="columnDropdownRef">
              <button
                type="button"
                @click="showColumnDropdown = !showColumnDropdown"
                class="qw-btn"
                :title="t('admin.users.columnSettings')"
              >
                <Icon name="grid" size="sm" />
                <span class="hidden md:inline">{{ t('admin.users.columnSettings') }}</span>
              </button>
              <div
                v-if="showColumnDropdown"
                class="dropdown right-0 top-full mt-1.5 max-h-80 w-52 overflow-y-auto"
              >
                <button
                  v-for="col in currentToggleableColumns"
                  :key="col.key"
                  type="button"
                  @click="toggleCurrentColumn(col.key)"
                  class="dropdown-item justify-between"
                >
                  <span>{{ col.label }}</span>
                  <Icon v-if="isCurrentColumnVisible(col.key)" name="check" size="sm" class="text-brand" />
                </button>
              </div>
            </div>
            <button
              v-if="activeTab !== 'errors'"
              type="button"
              class="qw-btn qw-btn-primary"
              :disabled="exporting"
              @click="exportToCSV"
            >
              {{ exporting ? t('usage.exporting') : t('usage.exportCsv') }}
            </button>
          </div>
        </div>

        <!-- Filter row: tab-specific filters -->
        <div class="mt-3 flex flex-wrap items-center gap-3">
          <template v-if="activeTab === 'errors'">
            <div class="qw-pill w-full sm:w-56">
              <Select v-model="errorFilter.api_key_id" :options="errorKeyOptions" @change="applyErrorFilters" />
            </div>
            <div class="qw-pill w-full sm:w-56">
              <Select
                v-model="errorFilter.model"
                :options="errorModelOptions"
                searchable
                creatable
                clearable
                :placeholder="t('usage.errors.modelPlaceholder')"
                @change="applyErrorFilters"
              />
            </div>
            <div class="qw-pill w-full sm:w-52">
              <Select v-model="errorFilter.category" :options="errorCategoryOptions" @change="applyErrorFilters" />
            </div>
            <div class="qw-pill w-full sm:w-44">
              <Select v-model="errorFilter.status_code" :options="errorStatusOptions" @change="applyErrorFilters" />
            </div>
          </template>
          <template v-else>
            <div class="qw-pill w-full sm:w-56">
              <Select v-model="filters.api_key_id" :options="apiKeyOptions" @change="applyFilters" />
            </div>
            <div class="qw-pill w-full sm:w-56">
              <Select v-model="filters.model" :options="modelOptions" searchable @change="applyFilters" />
            </div>
            <div class="qw-pill w-full sm:w-52">
              <Select v-model="filters.group_id" :options="groupOptions" searchable @change="applyFilters" />
            </div>
            <div class="qw-pill w-full sm:w-44">
              <Select v-model="filters.request_type" :options="requestTypeOptions" @change="applyFilters" />
            </div>
            <div class="qw-pill w-full sm:w-52">
              <Select v-model="filters.billing_type" :options="billingTypeOptions" @change="applyFilters" />
            </div>
            <div class="qw-pill w-full sm:w-52">
              <Select v-model="filters.billing_mode" :options="billingModeOptions" @change="applyFilters" />
            </div>
          </template>
        </div>

        <!-- Load error banner -->
        <div
          v-if="chartError"
          role="alert"
          class="mt-4 flex h-12 items-center gap-2 rounded-xl bg-[color:var(--nm-danger-soft)] px-4 text-sm text-[color:var(--nm-danger-text)]"
        >
          <Icon name="xCircle" size="sm" class="shrink-0" />
          <span>{{ t('usage.chartFailed') }}</span>
        </div>

        <!-- Charts -->
        <div class="mt-6 grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div class="qw-flat">
            <ModelDistributionChart
              v-model:metric="modelDistributionMetric"
              :model-stats="requestedModelStats"
              :loading="modelStatsLoading"
              :show-source-toggle="false"
              :show-metric-toggle="true"
              :enable-breakdown="false"
              :show-account-cost="false"
              :start-date="startDate"
              :end-date="endDate"
            />
          </div>
          <div class="qw-flat">
            <GroupDistributionChart
              v-model:metric="groupDistributionMetric"
              :group-stats="groupStats"
              :loading="chartsLoading"
              :show-metric-toggle="true"
              :enable-breakdown="false"
              :show-account-cost="false"
              :start-date="startDate"
              :end-date="endDate"
            />
          </div>
          <div class="qw-flat">
            <EndpointDistributionChart
              v-model:source="endpointDistributionSource"
              v-model:metric="endpointDistributionMetric"
              :endpoint-stats="inboundEndpointStats"
              :upstream-endpoint-stats="upstreamEndpointStats"
              :endpoint-path-stats="endpointPathStats"
              :loading="endpointStatsLoading"
              :show-source-toggle="false"
              :show-metric-toggle="true"
              :enable-breakdown="false"
              :title="t('usage.endpointDistribution')"
              :start-date="startDate"
              :end-date="endDate"
            />
          </div>
        </div>

        <template v-if="activeTab === 'usage'">
          <div class="mt-6">
            <UsageTable
              :data="usageLogs"
              :loading="loading"
              :columns="visibleColumns"
              :server-side-sort="true"
              :show-account-billing="false"
              :show-upstream-endpoint="false"
              :flat="true"
              default-sort-key="created_at"
              default-sort-order="desc"
              @sort="handleSort"
              @ipGeoBatchFailed="handleIpGeoBatchFailed"
            />

            <Pagination
              v-if="pagination.total > 0"
              :page="pagination.page"
              :total="pagination.total"
              :page-size="pagination.page_size"
              @update:page="handlePageChange"
              @update:pageSize="handlePageSizeChange"
            />
          </div>
        </template>

        <div v-else-if="errorViewEnabled" class="mt-6 qw-flat">
          <UserErrorRequestsTable
            :rows="errorRows"
            :total="errorTotal"
            :loading="errorLoading"
            :page="errorPage"
            :page-size="errorPageSize"
            :visible-column-keys="errVisibleColumnKeys"
            @sort="onErrorSort"
            @update:page="onErrorPage"
            @update:pageSize="onErrorPageSize"
            @ipGeoBatchFailed="handleIpGeoBatchFailed"
          />
        </div>
      </section>
    </div>
  </AppLayout>

</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { keysAPI, usageAPI, userGroupsAPI } from '@/api'
import { sanitizeUrl } from '@/utils/url'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import UsageStatsCards from '@/components/admin/usage/UsageStatsCards.vue'
import UsageTable from '@/components/admin/usage/UsageTable.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import GroupDistributionChart from '@/components/charts/GroupDistributionChart.vue'
import EndpointDistributionChart from '@/components/charts/EndpointDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import Icon from '@/components/icons/Icon.vue'
import UserErrorRequestsTable from '@/components/user/UserErrorRequestsTable.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatReasoningEffort } from '@/utils/format'
import { getBillingModeLabel, getDisplayBillingMode as resolveDisplayBillingMode } from '@/utils/billingMode'
import { resolveUsageRequestType, requestTypeToLegacyStream } from '@/utils/usageRequestType'
import type {
  ApiKey,
  EndpointStat,
  Group,
  GroupStat,
  ModelStat,
  TrendDataPoint,
  UsageLog,
  UsageQueryParams,
  UsageStatsResponse,
  UserErrorRequest,
} from '@/types'
import type { Column } from '@/components/common/types'
import { COMMON_ERROR_STATUS_CODES } from '@/utils/errorBadges'

const { t } = useI18n()
const appStore = useAppStore()
const router = useRouter()

type DistributionMetric = 'tokens' | 'actual_cost'
type EndpointSource = 'inbound' | 'upstream' | 'path'

// QW header links: doc external link (hidden when unset) + /keys link.
// `|| ''` 兜底：settings 未加载时 appStore.docUrl 可能为 undefined，sanitizeUrl 需要字符串。
const docUrl = computed(() => sanitizeUrl(appStore.docUrl || ''))
const goKeys = (event: MouseEvent) => {
  event.preventDefault()
  void router.push('/keys')
}

// "Last updated" stamp + chart load error banner (QW header/detail card affordances).
const lastUpdated = ref<Date | null>(null)
const lastUpdatedText = computed(() => {
  if (!lastUpdated.value) return '--:--:--'
  return lastUpdated.value.toLocaleTimeString(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
})
const chartError = ref(false)

const usageStats = ref<UsageStatsResponse | null>(null)
const usageLogs = ref<UsageLog[]>([])
const trendData = ref<TrendDataPoint[]>([])
const requestedModelStats = ref<ModelStat[]>([])
const groupStats = ref<GroupStat[]>([])
const inboundEndpointStats = ref<EndpointStat[]>([])
const upstreamEndpointStats = ref<EndpointStat[]>([])
const endpointPathStats = ref<EndpointStat[]>([])

const loading = ref(false)
const chartsLoading = ref(false)
const modelStatsLoading = ref(false)
const endpointStatsLoading = ref(false)
const exporting = ref(false)
const errorRows = ref<UserErrorRequest[]>([])
const errorLoading = ref(false)
const errorPage = ref(1)
const errorPageSize = ref(20)
const errorSortBy = ref('created_at')
const errorSortOrder = ref<'asc' | 'desc'>('desc')
const errorTotal = ref(0)
const errorFilter = ref<{ model: string | null; category: string; api_key_id: number | null; status_code: number | null }>({
  model: '',
  category: '',
  api_key_id: null,
  status_code: null,
})

const errorKeyOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.errors.allKeys') },
  ...apiKeys.value.map((k) => ({ value: k.id, label: k.name })),
])

// 模型候选取自当前已加载错误中出现过的模型；creatable 允许输入任意片段做后端模糊。
const errorModelOptions = computed<SelectOption[]>(() => {
  const seen = new Set<string>()
  const opts: SelectOption[] = []
  for (const r of errorRows.value) {
    if (r.model && !seen.has(r.model)) {
      seen.add(r.model)
      opts.push({ value: r.model, label: r.model })
    }
  }
  return opts
})

const errorCategoryCodes = ['auth', 'rate_limit', 'quota', 'invalid_request', 'service_unavailable', 'upstream', 'internal', 'cyber']

const errorCategoryOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('usage.errors.allCategories') },
  ...errorCategoryCodes.map((c) => ({ value: c, label: t('usage.errors.categories.' + c) })),
])

// 状态码候选用固定常用列表(与管理端 UsageFilters 共用常量),不受当前页数据限制:
// 后端 status_code 过滤对全量生效,若只列当前页出现过的码,用户就选不到仅在后续页的码。
const errorStatusOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.errors.allStatuses') },
  ...COMMON_ERROR_STATUS_CODES.map((c) => ({ value: c, label: String(c) })),
])

const applyErrorFilters = () => {
  errorPage.value = 1
  void loadErrors()
}

let abortController: AbortController | null = null
let chartReqSeq = 0
let statsReqSeq = 0
let modelStatsReqSeq = 0

const formatLocalDate = (date: Date): string =>
  `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`

const getLast24HoursRangeDates = () => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return { start: formatLocalDate(start), end: formatLocalDate(end) }
}

const getGranularityForRange = (start: string, end: string): 'day' | 'hour' => {
  const startTime = new Date(`${start}T00:00:00`).getTime()
  const endTime = new Date(`${end}T00:00:00`).getTime()
  return Math.ceil((endTime - startTime) / (1000 * 60 * 60 * 24)) <= 1 ? 'hour' : 'day'
}

const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)
const granularity = ref<'day' | 'hour'>(getGranularityForRange(startDate.value, endDate.value))

// Average request/token rates over the selected range (account summary card).
const rangeMinutes = computed(() => {
  const start = new Date(`${startDate.value}T00:00:00`).getTime()
  const end = new Date(`${endDate.value}T23:59:59`).getTime()
  if (Number.isNaN(start) || Number.isNaN(end) || end <= start) return 1
  return Math.max(1, Math.round((end - start) / 60000))
})
const requestsPerMinute = computed(() => (usageStats.value?.total_requests ?? 0) / rangeMinutes.value)
const tokensPerMinute = computed(() => (usageStats.value?.total_tokens ?? 0) / rangeMinutes.value)
const formatRate = (value: number): string => (value >= 100 ? Math.round(value).toLocaleString() : value.toFixed(1))

const modelDistributionMetric = ref<DistributionMetric>('tokens')
const groupDistributionMetric = ref<DistributionMetric>('tokens')
const endpointDistributionMetric = ref<DistributionMetric>('tokens')
const endpointDistributionSource = ref<EndpointSource>('inbound')
const activeTab = ref<'usage' | 'errors'>('usage')
const errorViewEnabled = computed(() => appStore.cachedPublicSettings?.allow_user_view_error_requests ?? false)

const filters = ref<UsageQueryParams>({
  start_date: startDate.value,
  end_date: endDate.value,
  request_type: undefined,
  billing_type: null,
  billing_mode: null,
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
})
const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc',
})

const granularityOptions = computed<SelectOption[]>(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') },
])
const requestTypeOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allTypes') },
  { value: 'ws_v2', label: t('usage.ws') },
  { value: 'live', label: t('usage.live') },
  { value: 'stream', label: t('usage.stream') },
  { value: 'sync', label: t('usage.sync') },
])
const billingTypeOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allBillingTypes') },
  { value: 0, label: t('admin.usage.billingTypeBalance') },
  { value: 1, label: t('admin.usage.billingTypeSubscription') },
])
const billingModeOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allBillingModes') },
  { value: 'token', label: t('admin.usage.billingModeToken') },
  { value: 'per_request', label: t('admin.usage.billingModePerRequest') },
  { value: 'image', label: t('admin.usage.billingModeImage') },
  { value: 'video', label: t('admin.usage.billingModeVideo') },
])

const apiKeys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const modelOptionValues = ref<string[]>([])

const apiKeyOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.allApiKeys') },
  ...apiKeys.value.map((key) => ({ value: key.id, label: key.name })),
])
const groupOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allGroups') },
  ...groups.value.map((group) => ({ value: group.id, label: group.name })),
])
const modelOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allModels') },
  ...modelOptionValues.value.map((model) => ({ value: model, label: model })),
])

const normalizedFilters = computed<UsageQueryParams>(() => {
  const requestType = filters.value.request_type
  const legacyStream = requestType ? requestTypeToLegacyStream(requestType) : filters.value.stream
  return {
    ...filters.value,
    start_date: startDate.value,
    end_date: endDate.value,
    stream: legacyStream === null ? undefined : legacyStream,
  }
})

const buildUsageListParams = (page: number, pageSize: number): UsageQueryParams => ({
  page,
  page_size: pageSize,
  ...normalizedFilters.value,
  sort_by: sortState.sort_by,
  sort_order: sortState.sort_order,
})

const loadLogs = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  try {
    const res = await usageAPI.query(buildUsageListParams(pagination.page, pagination.page_size), {
      signal: controller.signal,
    })
    if (!controller.signal.aborted) {
      usageLogs.value = res.items
      pagination.total = res.total
    }
  } catch (error: any) {
    if (error?.name !== 'AbortError' && error?.code !== 'ERR_CANCELED') {
      appStore.showError(t('usage.failedToLoad'))
    }
  } finally {
    if (abortController === controller) loading.value = false
  }
}

const loadStats = async () => {
  const seq = ++statsReqSeq
  endpointStatsLoading.value = true
  try {
    const stats = await usageAPI.getStats(normalizedFilters.value)
    if (seq !== statsReqSeq) return
    usageStats.value = stats
    lastUpdated.value = new Date()
    inboundEndpointStats.value = stats.endpoints || []
    upstreamEndpointStats.value = []
    endpointPathStats.value = []
  } catch (error) {
    if (seq !== statsReqSeq) return
    console.error('Failed to load usage stats:', error)
    inboundEndpointStats.value = []
    upstreamEndpointStats.value = []
    endpointPathStats.value = []
  } finally {
    if (seq === statsReqSeq) endpointStatsLoading.value = false
  }
}

const loadModelStats = async () => {
  const seq = ++modelStatsReqSeq
  modelStatsLoading.value = true
  try {
    const response = await usageAPI.getDashboardModels({
      ...normalizedFilters.value,
      model_source: 'requested',
    })
    if (seq !== modelStatsReqSeq) return
    requestedModelStats.value = response.models || []
    refreshModelOptions(response.models || [])
  } catch (error) {
    if (seq !== modelStatsReqSeq) return
    console.error('Failed to load model stats:', error)
    requestedModelStats.value = []
  } finally {
    if (seq === modelStatsReqSeq) modelStatsLoading.value = false
  }
}

const loadChartData = async () => {
  const seq = ++chartReqSeq
  chartsLoading.value = true
  try {
    const snapshot = await usageAPI.getDashboardSnapshotV2({
      ...normalizedFilters.value,
      granularity: granularity.value,
      include_trend: true,
      include_model_stats: false,
      include_group_stats: true,
    })
    if (seq !== chartReqSeq) return
    trendData.value = snapshot.trend || []
    groupStats.value = snapshot.groups || []
    chartError.value = false
  } catch (error) {
    if (seq !== chartReqSeq) return
    console.error('Failed to load chart data:', error)
    trendData.value = []
    groupStats.value = []
    chartError.value = true
  } finally {
    if (seq === chartReqSeq) chartsLoading.value = false
  }
}

const refreshModelOptions = (models: ModelStat[]) => {
  const current = filters.value.model
  const set = new Set(modelOptionValues.value)
  models.forEach((item) => {
    if (item.model) set.add(item.model)
  })
  if (current) set.add(current)
  modelOptionValues.value = Array.from(set).sort()
}

const applyFilters = () => {
  pagination.page = 1
  void loadLogs()
  void loadStats()
  void loadModelStats()
  void loadChartData()
  resetErrorRows()
}

const refreshData = () => {
  void loadLogs()
  void loadStats()
  void loadModelStats()
  void loadChartData()
  if (activeTab.value === 'errors') void loadErrors()
}

const resetFilters = () => {
  const range = getLast24HoursRangeDates()
  startDate.value = range.start
  endDate.value = range.end
  filters.value = {
    start_date: range.start,
    end_date: range.end,
    request_type: undefined,
    billing_type: null,
    billing_mode: null,
  }
  granularity.value = getGranularityForRange(range.start, range.end)
  applyFilters()
  if (activeTab.value === 'errors') {
    errorFilter.value = { model: '', category: '', api_key_id: null, status_code: null }
    applyErrorFilters()
  }
}

const onDateRangeChange = (range: { startDate: string; endDate: string; preset: string | null }) => {
  startDate.value = range.startDate
  endDate.value = range.endDate
  filters.value.start_date = range.startDate
  filters.value.end_date = range.endDate
  granularity.value = getGranularityForRange(range.startDate, range.endDate)
  applyFilters()
}

const handlePageChange = (page: number) => {
  pagination.page = page
  void loadLogs()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadLogs()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  void loadLogs()
}

const handleIpGeoBatchFailed = () => {
  appStore.showError(t('usage.ipGeo.batchFailed'))
}

const getRequestTypeExportText = (log: UsageLog): string => {
  const requestType = resolveUsageRequestType(log)
  if (requestType === 'cyber') return 'Cyber'
  if (requestType === 'live') return 'Live'
  if (requestType === 'ws_v2') return 'WS'
  if (requestType === 'stream') return 'Stream'
  if (requestType === 'sync') return 'Sync'
  return 'Unknown'
}

const getDisplayBillingMode = (
  row: Pick<UsageLog, 'billing_mode' | 'image_count'> | null | undefined
): string | null | undefined => resolveDisplayBillingMode(row)

const escapeCSVValue = (value: unknown): string => {
  if (value == null) return ''
  const str = String(value)
  const escaped = str.replace(/"/g, '""')
  if (/^[=+\-@\t\r]/.test(str)) return `"\'${escaped}"`
  if (/[,"\n\r]/.test(str)) return `"${escaped}"`
  return str
}

const exportToCSV = async () => {
  if (pagination.total === 0) {
    appStore.showWarning(t('usage.noDataToExport'))
    return
  }
  exporting.value = true
  appStore.showInfo(t('usage.preparingExport'))
  try {
    const allLogs: UsageLog[] = []
    const pageSize = 100
    const totalPages = Math.ceil(pagination.total / pageSize)
    for (let page = 1; page <= totalPages; page++) {
      const response = await usageAPI.query(buildUsageListParams(page, pageSize))
      allLogs.push(...response.items)
    }
    if (allLogs.length === 0) {
      appStore.showWarning(t('usage.noDataToExport'))
      return
    }
    const headers = [
      'Time',
      'API Key Name',
      'Model',
      'Reasoning Effort',
      'Inbound Endpoint',
      'IP Address',
      'Type',
      'Billing Mode',
      'Input Tokens',
      'Output Tokens',
      'Cache Read Tokens',
      'Cache Creation Tokens',
      'Rate Multiplier',
      'Billed Cost',
      'Original Cost',
      'First Token (ms)',
      'Duration (ms)',
    ]
    const rows = allLogs.map((log) => [
      log.created_at,
      log.api_key?.name || '',
      log.model,
      formatReasoningEffort(log.reasoning_effort),
      log.inbound_endpoint || '',
      log.ip_address || '',
      getRequestTypeExportText(log),
      getBillingModeLabel(getDisplayBillingMode(log), t),
      log.input_tokens,
      log.output_tokens,
      log.cache_read_tokens,
      log.cache_creation_tokens,
      log.rate_multiplier,
      log.actual_cost.toFixed(8),
      log.total_cost.toFixed(8),
      log.first_token_ms ?? '',
      log.duration_ms ?? '',
    ].map(escapeCSVValue))
    const csvContent = [
      headers.map(escapeCSVValue).join(','),
      ...rows.map((row) => row.join(',')),
    ].join('\n')
    const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `usage_${startDate.value}_to_${endDate.value}.csv`
    link.click()
    window.URL.revokeObjectURL(url)
    appStore.showSuccess(t('usage.exportSuccess'))
  } catch (error) {
    console.error('CSV Export failed:', error)
    appStore.showError(t('usage.exportFailed'))
  } finally {
    exporting.value = false
  }
}

const ALWAYS_VISIBLE = ['created_at']
const DEFAULT_HIDDEN_COLUMNS = ['user_agent']
const HIDDEN_COLUMNS_KEY = 'user-usage-hidden-columns'

const allColumns = computed<Column[]>(() => [
  { key: 'api_key', label: t('usage.apiKeyFilter'), sortable: false },
  { key: 'model', label: t('usage.model'), sortable: true },
  { key: 'reasoning_effort', label: t('usage.reasoningEffort'), sortable: false },
  { key: 'endpoint', label: t('usage.endpoint'), sortable: false },
  { key: 'ip_address', label: 'IP', sortable: false },
  { key: 'group', label: t('admin.usage.group'), sortable: false },
  { key: 'stream', label: t('usage.type'), sortable: false },
  { key: 'billing_mode', label: t('admin.usage.billingMode'), sortable: false },
  { key: 'tokens', label: t('usage.tokens'), sortable: false },
  { key: 'cost', label: t('usage.cost'), sortable: false },
  { key: 'latency', label: t('usage.latency'), sortable: false },
  { key: 'created_at', label: t('usage.time'), sortable: true },
  { key: 'user_agent', label: t('usage.userAgent'), sortable: false },
])

const hiddenColumns = reactive<Set<string>>(new Set())
const toggleableColumns = computed(() => allColumns.value.filter((col) => !ALWAYS_VISIBLE.includes(col.key)))
const visibleColumns = computed(() =>
  allColumns.value.filter((col) => ALWAYS_VISIBLE.includes(col.key) || !hiddenColumns.has(col.key))
)
const isColumnVisible = (key: string) => !hiddenColumns.has(key)
const toggleColumn = (key: string) => {
  if (hiddenColumns.has(key)) hiddenColumns.delete(key)
  else hiddenColumns.add(key)
  localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
}
const loadSavedColumns = () => {
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    const values = saved ? JSON.parse(saved) as string[] : DEFAULT_HIDDEN_COLUMNS
    values.forEach((key) => hiddenColumns.add(key))
  } catch {
    DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
  }
}

// 错误请求 tab 独立列设置(机制同用量列设置,存储互不影响)
const ERR_ALWAYS_VISIBLE = ['status', 'created_at']
const ERR_DEFAULT_HIDDEN_COLUMNS = ['user_agent']
const ERR_HIDDEN_COLUMNS_KEY = 'user-usage-error-hidden-columns'

// key 须与 UserErrorRequestsTable 的 allColumns 一致
const errAllColumns = computed<Column[]>(() => [
  { key: 'key_name', label: t('usage.errors.keyName') },
  { key: 'model', label: t('usage.errors.model') },
  { key: 'endpoint', label: t('usage.errors.endpoint') },
  { key: 'client_ip', label: 'IP' },
  { key: 'group', label: t('admin.usage.group') },
  { key: 'type', label: t('usage.type') },
  { key: 'platform', label: t('usage.errors.platform') },
  { key: 'category', label: t('usage.errors.category') },
  { key: 'status', label: t('usage.errors.status') },
  { key: 'message', label: t('usage.errors.message') },
  { key: 'created_at', label: t('usage.errors.time') },
  { key: 'user_agent', label: t('usage.userAgent') },
])

const errHiddenColumns = reactive<Set<string>>(new Set())
const errToggleableColumns = computed(() =>
  errAllColumns.value.filter((col) => !ERR_ALWAYS_VISIBLE.includes(col.key))
)
const errVisibleColumnKeys = computed(() =>
  errAllColumns.value
    .filter((col) => ERR_ALWAYS_VISIBLE.includes(col.key) || !errHiddenColumns.has(col.key))
    .map((col) => col.key)
)
const isErrColumnVisible = (key: string) => !errHiddenColumns.has(key)
const toggleErrColumn = (key: string) => {
  if (errHiddenColumns.has(key)) errHiddenColumns.delete(key)
  else errHiddenColumns.add(key)
  localStorage.setItem(ERR_HIDDEN_COLUMNS_KEY, JSON.stringify([...errHiddenColumns]))
}
const loadSavedErrColumns = () => {
  try {
    const saved = localStorage.getItem(ERR_HIDDEN_COLUMNS_KEY)
    const values = saved ? (JSON.parse(saved) as string[]) : ERR_DEFAULT_HIDDEN_COLUMNS
    values.forEach((key) => errHiddenColumns.add(key))
  } catch {
    ERR_DEFAULT_HIDDEN_COLUMNS.forEach((key) => errHiddenColumns.add(key))
  }
}

// 列设置下拉按当前 tab 分发
const currentToggleableColumns = computed(() =>
  activeTab.value === 'errors' ? errToggleableColumns.value : toggleableColumns.value
)
const isCurrentColumnVisible = (key: string) =>
  activeTab.value === 'errors' ? isErrColumnVisible(key) : isColumnVisible(key)
const toggleCurrentColumn = (key: string) => {
  if (activeTab.value === 'errors') toggleErrColumn(key)
  else toggleColumn(key)
}

const showColumnDropdown = ref(false)
const columnDropdownRef = ref<HTMLElement | null>(null)
const handleColumnClickOutside = (event: MouseEvent) => {
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(event.target as HTMLElement)) {
    showColumnDropdown.value = false
  }
}

const loadFilterOptions = async () => {
  try {
    const [keys, availableGroups] = await Promise.all([
      keysAPI.list(1, 100),
      userGroupsAPI.getAvailable(),
    ])
    apiKeys.value = keys.items
    groups.value = availableGroups
  } catch (error) {
    console.error('Failed to load usage filter options:', error)
  }
}

const resetErrorRows = () => {
  errorPage.value = 1
  if (activeTab.value === 'errors') {
    void loadErrors()
  } else {
    errorRows.value = []
    errorTotal.value = 0
  }
}

const loadErrors = async () => {
  errorLoading.value = true
  try {
    const resp = await usageAPI.listMyErrorRequests({
      page: errorPage.value,
      page_size: errorPageSize.value,
      start_date: startDate.value,
      end_date: endDate.value,
      model: (errorFilter.value.model ?? '').trim() || undefined,
      category: errorFilter.value.category || undefined,
      api_key_id: errorFilter.value.api_key_id ?? undefined,
      status_code: errorFilter.value.status_code ?? undefined,
      sort_by: errorSortBy.value,
      sort_order: errorSortOrder.value,
    })
    errorRows.value = resp.items
    errorTotal.value = resp.total
  } catch (error) {
    console.error('[UsageView] loadErrors failed:', error)
    appStore.showError(t('usage.errors.failedToLoad'))
  } finally {
    errorLoading.value = false
  }
}

const onErrorSort = (sortBy: string, sortOrder: 'asc' | 'desc') => {
  errorSortBy.value = sortBy
  errorSortOrder.value = sortOrder
  errorPage.value = 1
  void loadErrors()
}

const onErrorPage = (page: number) => {
  errorPage.value = page
  void loadErrors()
}

const onErrorPageSize = (pageSize: number) => {
  errorPageSize.value = pageSize
  errorPage.value = 1
  void loadErrors()
}

const switchToErrors = () => {
  activeTab.value = 'errors'
  if (errorRows.value.length === 0) void loadErrors()
}

onMounted(() => {
  loadSavedColumns()
  loadSavedErrColumns()
  document.addEventListener('click', handleColumnClickOutside)
  void loadFilterOptions()
  refreshData()
})

onUnmounted(() => {
  abortController?.abort()
  document.removeEventListener('click', handleColumnClickOutside)
})

watch(endpointDistributionSource, () => {
  // Endpoint source switching is handled by the chart component using already loaded stats.
})
</script>

<style scoped>
/* QW text tones — exact light hexes, token-based dark fallbacks. */
.qw-desc {
  color: #7f8798;
}
.qw-weak {
  color: #8e96a7;
}
:global(.dark) .qw-desc,
:global(.dark) .qw-weak {
  color: var(--nm-ink-faint);
}

/* Accent links in the title row. */
.qw-link {
  color: var(--nm-accent);
}
.qw-link:hover {
  color: var(--nm-accent-strong);
  text-decoration: underline;
  text-underline-offset: 3px;
}

/* Segmented pill tabs. */
.qw-tab {
  display: inline-flex;
  align-items: center;
  height: 2rem;
  padding: 0 1rem;
  border-radius: 9999px;
  font-size: 12px;
  color: var(--nm-ink);
  transition: background-color 160ms ease;
}
.qw-tab:hover {
  background: var(--nm-surface-soft);
}
.qw-tab-active {
  background: var(--nm-surface-soft);
  font-weight: 500;
}

/* Pill-shaped filter controls (Select / DateRangePicker triggers). */
.qw-pill :deep(.select-trigger),
.qw-pill :deep(.date-picker-trigger) {
  height: 2.5rem;
  min-height: 2.5rem;
  padding: 0 1rem;
  border-radius: 9999px;
  border-color: var(--nm-border);
  background: var(--nm-surface);
  font-size: 13px;
  box-shadow: none;
}
.qw-pill :deep(.select-trigger-open),
.qw-pill :deep(.date-picker-trigger-open) {
  border-color: hsl(var(--ring));
}

/* Outline pill actions. */
.qw-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  height: 2.25rem;
  padding: 0 18px;
  border-radius: 9999px;
  border: 1px solid var(--nm-border-strong);
  background: transparent;
  font-size: 13px;
  font-weight: 500;
  color: var(--nm-ink);
  transition: background-color 160ms ease, opacity 160ms ease;
}
.qw-btn:hover:not(:disabled) {
  background: var(--nm-surface-soft);
}
.qw-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.qw-btn-primary {
  background: var(--nm-ink);
  border-color: var(--nm-ink);
  color: var(--nm-surface);
}
.qw-btn-primary:hover:not(:disabled) {
  background: var(--nm-ink);
  opacity: 0.9;
}

/* Flatten nested cards so charts/stats sit directly on the QW card. */
.qw-flat :deep(.card) {
  background: transparent;
  border: none;
  box-shadow: none;
  padding: 0;
}
</style>
