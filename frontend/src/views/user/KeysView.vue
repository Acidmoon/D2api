<template>
  <AppLayout>
    <div class="flex flex-col gap-3">
      <!-- Page header: 28px/600 title + 14px muted description -->
      <header class="mb-3">
        <h1 class="text-[28px] font-semibold leading-9 tracking-[-0.01em] text-foreground">
          {{ t('keys.title') }}
        </h1>
        <p class="qw-desc mt-2 text-sm">{{ t('keys.description') }}</p>
      </header>

      <!-- Endpoint card -->
      <section v-if="endpointRows.length > 0" class="qw-card p-7">
        <div>
          <h2 class="text-xl font-semibold text-foreground">
            {{ t('keys.endpoints.title') }}
          </h2>
          <p class="qw-desc mt-2 text-sm">{{ t('keys.endpoints.description') }}</p>
        </div>
        <div class="mt-5 grid gap-3 md:grid-cols-2">
          <div
            v-for="item in endpointRows"
            :key="item.endpoint"
            class="qw-endpoint-row flex h-12 items-center gap-3 px-3"
          >
            <span
              class="qw-endpoint-chip flex h-8 min-w-0 shrink-0 items-center truncate rounded-lg px-3 text-[13px] font-medium"
              :title="item.description || item.name"
            >{{ item.name }}</span>
            <span
              v-if="item.isDefault"
              class="shrink-0 text-xs font-medium text-muted-foreground"
            >{{ t('keys.endpoints.default') }}</span>
            <span class="min-w-0 flex-1 truncate text-sm text-foreground" :title="item.endpoint">
              {{ item.endpoint }}
            </span>
            <button
              type="button"
              class="qw-icon-btn size-9 shrink-0"
              :class="copiedEndpoint === item.endpoint ? 'text-semantic-success' : ''"
              :title="
                copiedEndpoint === item.endpoint
                  ? t('keys.endpoints.copiedHint')
                  : t('keys.endpoints.clickToCopy')
              "
              :aria-label="
                copiedEndpoint === item.endpoint
                  ? t('keys.endpoints.copiedHint')
                  : t('keys.endpoints.clickToCopy')
              "
              @click="copyEndpoint(item.endpoint)"
            >
              <Check v-if="copiedEndpoint === item.endpoint" class="h-4 w-4" />
              <Clipboard v-else class="h-4 w-4" />
            </button>
            <a
              :href="endpointSpeedTestUrl(item.endpoint)"
              target="_blank"
              rel="noopener noreferrer"
              class="qw-icon-btn size-9 shrink-0"
              :title="t('keys.endpoints.speedTest')"
              :aria-label="t('keys.endpoints.speedTest')"
            >
              <Zap class="h-4 w-4" />
            </a>
          </div>
        </div>
      </section>

      <!-- API keys table card -->
      <section class="qw-card p-7">
        <div>
          <h2 class="text-xl font-semibold text-foreground">{{ t('keys.table.title') }}</h2>
          <p class="qw-desc mt-2 text-sm">
            {{ t('keys.table.description') }}
            <a
              v-if="sanitizedDocUrl"
              :href="sanitizedDocUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="qw-learn-more ml-4 inline-flex items-center gap-0.5 text-[13px] font-medium"
            >
              {{ t('keys.endpoints.learnMore') }}
              <ExternalLink class="h-3.5 w-3.5" />
            </a>
          </p>
        </div>

        <!-- Toolbar: search + filters left, actions right -->
        <div class="mt-5 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex flex-1 flex-wrap items-center gap-2">
            <SearchInput
              v-model="filterSearch"
              :placeholder="t('keys.searchPlaceholder')"
              class="qw-search w-full sm:w-[280px]"
              pills
              @search="onFilterChange"
            />
            <Select
              :model-value="filterGroupId"
              class="qw-filter-select w-40"
              :options="groupFilterOptions"
              @update:model-value="onGroupFilterChange"
            />
            <Select
              :model-value="filterStatus"
              class="qw-filter-select w-40"
              :options="statusFilterOptions"
              @update:model-value="onStatusFilterChange"
            />
            <Button
              variant="ghost"
              size="icon"
              class="qw-icon-btn size-9 rounded-full"
              :disabled="loading"
              :title="t('common.refresh')"
              :aria-label="t('common.refresh')"
              @click="loadApiKeys"
            >
              <RefreshCw :class="loading ? 'animate-spin' : ''" />
            </Button>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <div class="relative" ref="columnDropdownRef">
              <Button
                variant="ghost"
                size="icon"
                class="qw-icon-btn size-9 rounded-full"
                :title="t('keys.columnSettings')"
                @click="showColumnDropdown = !showColumnDropdown"
              >
                <Columns3 class="h-4 w-4" />
              </Button>
              <div
                v-if="showColumnDropdown"
                class="dropdown right-0 top-full mt-1.5 max-h-80 w-52 overflow-y-auto"
              >
                <button
                  v-for="col in toggleableColumns"
                  :key="col.key"
                  type="button"
                  class="dropdown-item justify-between"
                  @click="toggleColumn(col.key)"
                >
                  <span>{{ col.label }}</span>
                  <Check
                    v-if="isColumnVisible(col.key)"
                    class="h-4 w-4 shrink-0 text-brand"
                  />
                </button>
              </div>
            </div>
            <Button
              class="h-10 rounded-full px-5 text-sm font-medium"
              data-tour="keys-create-btn"
              @click="showCreateModal = true"
            >
              <Plus />
              {{ t('keys.createKey') }}
            </Button>
          </div>
        </div>

        <div class="qw-table mt-3">
        <DataTable
          :columns="columns"
          :data="apiKeys"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-id="{ value }">
            <span class="font-mono text-xs text-muted-foreground">#{{ value }}</span>
          </template>

          <template #cell-key="{ value, row }">
            <div class="flex items-center gap-2">
              <code class="code text-xs">
                {{ maskApiKey(value) }}
              </code>
              <button
                @click="copyToClipboard(value, row.id)"
                class="rounded-full p-1.5 transition-colors hover:bg-muted"
                :class="
                  copiedKeyId === row.id
                    ? 'text-semantic-success'
                    : 'text-muted-foreground hover:text-foreground'
                "
                :title="copiedKeyId === row.id ? t('keys.copied') : t('keys.copyToClipboard')"
              >
                <Check v-if="copiedKeyId === row.id" class="h-4 w-4" />
                <Clipboard v-else class="h-4 w-4" />
              </button>
            </div>
          </template>

          <template #cell-name="{ value, row }">
            <div class="flex items-center gap-1.5">
              <span class="font-medium text-foreground">{{ value }}</span>
              <Shield
                v-if="row.ip_whitelist?.length > 0 || row.ip_blacklist?.length > 0"
                class="h-4 w-4 text-muted-foreground"
                :title="t('keys.ipRestrictionEnabled')"
              />
            </div>
          </template>

          <template #cell-primary_group="{ row }">
            <div class="group/dropdown relative">
              <button
                :ref="(el) => setGroupButtonRef(row.id, 'primary', el)"
                @click="openGroupSelector(row, 'primary')"
                class="-mx-2 -my-1 flex max-w-full cursor-pointer items-center gap-2 overflow-hidden rounded-full px-2 py-1 transition-all duration-200 hover:bg-muted"
                :title="t('keys.clickToChangeGroup')"
              >
                <GroupBadge
                  v-if="row.primary_group"
                  class="min-w-0 overflow-hidden"
                  :title="row.primary_group.name"
                  :name="row.primary_group.name"
                  :platform="row.primary_group.platform"
                  :subscription-type="row.primary_group.subscription_type"
                  :rate-multiplier="row.primary_group.rate_multiplier"
                  :user-rate-multiplier="userGroupRates[row.primary_group.id]"
                />
                <span v-else class="text-sm text-muted-foreground">{{
                  t('keys.noGroup')
                }}</span>
                <span class="shrink-0 text-xs text-muted-foreground">{{ t('keys.selectGroup') }}</span>
                <svg
                  class="h-3.5 w-3.5 shrink-0 text-muted-foreground opacity-60 transition-opacity group-hover/dropdown:opacity-100"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  stroke-width="2"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M8.25 15L12 18.75 15.75 15m-7.5-6L12 5.25 15.75 9"
                  />
                </svg>
              </button>
            </div>
          </template>

          <template #cell-group="{ row }">
            <div class="group/dropdown relative">
              <button
                :ref="(el) => setGroupButtonRef(row.id, 'secondary', el)"
                @click="openGroupSelector(row, 'secondary')"
                class="-mx-2 -my-1 flex max-w-full cursor-pointer items-center gap-2 overflow-hidden rounded-full px-2 py-1 transition-all duration-200 hover:bg-muted"
                :title="t('keys.clickToChangeGroup')"
              >
                <GroupBadge
                  v-if="row.group"
                  class="min-w-0 overflow-hidden"
                  :title="row.group.name"
                  :name="row.group.name"
                  :platform="row.group.platform"
                  :subscription-type="row.group.subscription_type"
                  :rate-multiplier="row.group.rate_multiplier"
                  :user-rate-multiplier="userGroupRates[row.group.id]"
                  :peak-rate-enabled="row.group.peak_rate_enabled"
                  :peak-start="row.group.peak_start"
                  :peak-end="row.group.peak_end"
                  :peak-rate-multiplier="row.group.peak_rate_multiplier"
                />
                <span v-else class="text-sm text-muted-foreground">{{
                  t('keys.noGroup')
                }}</span>
                <span class="shrink-0 text-xs text-muted-foreground">{{ t('keys.selectGroup') }}</span>
                <svg
                  class="h-3.5 w-3.5 shrink-0 text-muted-foreground opacity-60 transition-opacity group-hover/dropdown:opacity-100"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  stroke-width="2"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M8.25 15L12 18.75 15.75 15m-7.5-6L12 5.25 15.75 9"
                  />
                </svg>
              </button>
            </div>
          </template>

          <template #cell-current_concurrency="{ value }">
            <span
              :class="[
                'inline-flex min-w-8 items-center justify-center rounded px-2 py-1 text-sm font-semibold tabular-nums',
                (value ?? 0) > 0
                  ? 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-900/25 dark:text-emerald-300 dark:ring-emerald-800'
                  : 'bg-muted text-muted-foreground'
              ]"
            >
              {{ value ?? 0 }}
            </span>
          </template>

          <template #cell-usage="{ row }">
            <div class="text-sm">
              <div class="flex items-center gap-1.5">
                <span class="text-muted-foreground">{{ t('keys.today') }}:</span>
                <span class="font-medium text-foreground">
                  ${{ (usageStats[row.id]?.today_actual_cost ?? 0).toFixed(4) }}
                </span>
              </div>
              <div class="mt-0.5 flex items-center gap-1.5">
                <span class="text-muted-foreground">{{ t('keys.total') }}:</span>
                <span class="font-medium text-foreground">
                  ${{ (usageStats[row.id]?.total_actual_cost ?? 0).toFixed(4) }}
                </span>
              </div>
              <!-- Quota progress (if quota is set) -->
              <div v-if="row.quota > 0" class="mt-1.5">
                <div class="flex items-center gap-1.5">
                  <span class="text-muted-foreground">{{ t('keys.quota') }}:</span>
                  <span :class="['font-medium', usageToneClass(row.quota_used, row.quota)]">
                    ${{ row.quota_used?.toFixed(2) || '0.00' }} / ${{ row.quota?.toFixed(2) }}
                  </span>
                </div>
                <div class="metric-progress mt-1">
                  <div
                    :class="['metric-progress-bar', usageProgressClass(row.quota_used, row.quota)]"
                    :style="{ width: Math.min((row.quota_used / row.quota) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>
          </template>

          <template #cell-rate_limit="{ row }">
            <div v-if="row.rate_limit_5h > 0 || row.rate_limit_1d > 0 || row.rate_limit_7d > 0" class="space-y-1.5 min-w-[140px]">
              <!-- 5h window -->
              <div v-if="row.rate_limit_5h > 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-muted-foreground">5h</span>
                  <span :class="['font-medium tabular-nums', usageToneClass(row.usage_5h, row.rate_limit_5h)]">
                    ${{ row.usage_5h?.toFixed(2) || '0.00' }}/${{ row.rate_limit_5h?.toFixed(2) }}
                  </span>
                </div>
                <div class="metric-progress h-1">
                  <div
                    :class="['metric-progress-bar', usageProgressClass(row.usage_5h, row.rate_limit_5h)]"
                    :style="{ width: Math.min((row.usage_5h / row.rate_limit_5h) * 100, 100) + '%' }"
                  />
                </div>
                <div v-if="row.reset_5h_at && formatResetTime(row.reset_5h_at)" class="text-[10px] text-muted-foreground tabular-nums">
                  ⟳ {{ formatResetTime(row.reset_5h_at) }}
                </div>
              </div>
              <!-- 1d window -->
              <div v-if="row.rate_limit_1d > 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-muted-foreground">1d</span>
                  <span :class="['font-medium tabular-nums', usageToneClass(row.usage_1d, row.rate_limit_1d)]">
                    ${{ row.usage_1d?.toFixed(2) || '0.00' }}/${{ row.rate_limit_1d?.toFixed(2) }}
                  </span>
                </div>
                <div class="metric-progress h-1">
                  <div
                    :class="['metric-progress-bar', usageProgressClass(row.usage_1d, row.rate_limit_1d)]"
                    :style="{ width: Math.min((row.usage_1d / row.rate_limit_1d) * 100, 100) + '%' }"
                  />
                </div>
                <div v-if="row.reset_1d_at && formatResetTime(row.reset_1d_at)" class="text-[10px] text-muted-foreground tabular-nums">
                  ⟳ {{ formatResetTime(row.reset_1d_at) }}
                </div>
              </div>
              <!-- 7d window -->
              <div v-if="row.rate_limit_7d > 0">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-muted-foreground">7d</span>
                  <span :class="['font-medium tabular-nums', usageToneClass(row.usage_7d, row.rate_limit_7d)]">
                    ${{ row.usage_7d?.toFixed(2) || '0.00' }}/${{ row.rate_limit_7d?.toFixed(2) }}
                  </span>
                </div>
                <div class="metric-progress h-1">
                  <div
                    :class="['metric-progress-bar', usageProgressClass(row.usage_7d, row.rate_limit_7d)]"
                    :style="{ width: Math.min((row.usage_7d / row.rate_limit_7d) * 100, 100) + '%' }"
                  />
                </div>
                <div v-if="row.reset_7d_at && formatResetTime(row.reset_7d_at)" class="text-[10px] text-muted-foreground tabular-nums">
                  ⟳ {{ formatResetTime(row.reset_7d_at) }}
                </div>
              </div>
              <!-- Reset button -->
              <button
                v-if="row.usage_5h > 0 || row.usage_1d > 0 || row.usage_7d > 0"
                @click.stop="confirmResetRateLimitFromTable(row)"
                class="mt-0.5 inline-flex w-fit items-center gap-1 rounded-full px-1.5 py-0.5 text-xs font-medium text-brand transition-opacity hover:opacity-80 hover:underline"
                :title="t('keys.resetRateLimitUsage')"
              >
                <Icon name="refresh" size="xs" />
                {{ t('keys.resetUsage') }}
              </button>
            </div>
            <span v-else class="text-sm text-muted-foreground">-</span>
          </template>

          <template #cell-expires_at="{ value }">
            <span v-if="value" :class="[
              'text-sm',
              new Date(value) < new Date() ? 'text-semantic-danger font-medium' : 'text-muted-foreground'
            ]">
              {{ formatDateTime(value) }}
            </span>
            <span v-else class="text-sm text-muted-foreground">{{ t('keys.noExpiration') }}</span>
          </template>

          <template #cell-status="{ value, row }">
            <div class="flex items-center gap-2">
              <button
                type="button"
                role="switch"
                :aria-checked="value === 'active'"
                :aria-label="value === 'active' ? t('keys.disable') : t('keys.enable')"
                :title="value === 'active' ? t('keys.disable') : t('keys.enable')"
                class="switch"
                :class="{ 'switch-active': value === 'active' }"
                @click="toggleKeyStatus(row)"
              >
                <span class="switch-thumb" />
              </button>
              <span
                v-if="value === 'active' || value === 'inactive'"
                class="text-xs text-muted-foreground"
              >
                {{ t('keys.status.' + value) }}
              </span>
              <span
                v-else
                :class="[
                  'badge',
                  value === 'quota_exhausted' ? 'badge-warning' :
                  value === 'expired' ? 'badge-danger' :
                  'badge-gray'
                ]"
              >
                {{ t('keys.status.' + value) }}
              </span>
            </div>
          </template>

          <template #cell-last_used_at="{ value }">
            <span v-if="value" class="text-sm text-muted-foreground">
              {{ formatDateTime(value) }}
            </span>
            <span v-else class="text-sm text-muted-foreground">-</span>
          </template>

          <template #cell-last_used_ip="{ value }">
            <span v-if="value" class="text-sm text-muted-foreground">
              {{ value }}
            </span>
            <span v-else class="text-sm text-muted-foreground">-</span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-muted-foreground">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-3 whitespace-nowrap text-sm">
              <!-- Use Key link -->
              <button
                type="button"
                class="font-medium text-brand transition-opacity hover:underline hover:opacity-80"
                :title="t('keys.useKey')"
                @click="openUseKeyModal(row)"
              >
                {{ t('keys.useKey') }}
              </button>
              <!-- Import to CC Switch link -->
              <button
                v-if="!publicSettings?.hide_ccs_import_button"
                type="button"
                class="font-medium text-brand transition-opacity hover:underline hover:opacity-80"
                :title="t('keys.importToCcSwitch')"
                @click="importToCcswitch(row)"
              >
                {{ t('keys.importToCcSwitch') }}
              </button>
              <!-- Edit link -->
              <button
                type="button"
                class="font-medium text-brand transition-opacity hover:underline hover:opacity-80"
                :title="t('common.edit')"
                @click="editKey(row)"
              >
                {{ t('common.edit') }}
              </button>
              <!-- Delete link -->
              <button
                type="button"
                class="font-medium text-destructive transition-opacity hover:underline hover:opacity-80"
                :title="t('common.delete')"
                @click="confirmDelete(row)"
              >
                {{ t('common.delete') }}
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('keys.noKeysYet')"
              :description="t('keys.createFirstKey')"
              :action-text="t('keys.createKey')"
              @action="showCreateModal = true"
            />
          </template>
        </DataTable>
        </div>

        <!-- Pagination -->
        <div v-if="pagination.total > 0" class="mt-2">
          <Pagination
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.page_size"
            @update:page="handlePageChange"
            @update:pageSize="handlePageSizeChange"
          />
        </div>
      </section>
    </div>

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="showEditModal ? t('keys.editKey') : t('keys.createKey')"
      width="normal"
      @close="closeModals"
    >
      <form id="key-form" @submit.prevent="handleSubmit" class="space-y-5">
        <div class="space-y-1.5">
          <Label for="key-form-name">{{ t('keys.nameLabel') }}</Label>
          <Input
            id="key-form-name"
            v-model="formData.name"
            type="text"
            required
            :placeholder="t('keys.namePlaceholder')"
            data-tour="key-form-name"
          />
        </div>

        <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-1.5">
          <Label>{{ t('keys.primaryGroup') }}</Label>
          <Select
            v-model="formData.primary_group_id"
            :options="groupOptions"
            :placeholder="t('keys.selectGroup')"
            :searchable="true"
            :search-placeholder="t('keys.searchGroup')"
            :clearable="true"
            data-tour="key-form-primary-group"
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                :peak-start="(option as unknown as GroupOption).peakStart"
                :peak-end="(option as unknown as GroupOption).peakEnd"
                :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
              />
              <span v-else class="text-muted-foreground">{{ t('keys.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                :peak-start="(option as unknown as GroupOption).peakStart"
                :peak-end="(option as unknown as GroupOption).peakEnd"
                :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
        </div>

        <div class="space-y-1.5">
          <Label>{{ t('keys.secondaryGroup') }}</Label>
          <Select
            v-model="formData.group_id"
            :options="groupOptions"
            :placeholder="t('keys.selectGroup')"
            :searchable="true"
            :search-placeholder="t('keys.searchGroup')"
            :clearable="true"
            data-tour="key-form-group"
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
              />
              <span v-else class="text-muted-foreground">{{ t('keys.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
        </div>
        </div>

        <!-- Custom Key Section (only for create) -->
        <div v-if="!showEditModal" class="space-y-3">
          <div class="flex items-center justify-between">
            <Label class="mb-0">{{ t('keys.customKeyLabel') }}</Label>
            <Switch
              v-model="formData.use_custom_key"
            />
          </div>
          <div v-if="formData.use_custom_key" class="space-y-1.5">
            <Input
              v-model="formData.custom_key"
              type="text"
              class="font-mono"
              :class="{ 'border-destructive focus-visible:ring-destructive': customKeyError }"
              :placeholder="t('keys.customKeyPlaceholder')"
            />
            <p v-if="customKeyError" class="text-sm text-destructive">{{ customKeyError }}</p>
            <p v-else class="text-xs text-muted-foreground">{{ t('keys.customKeyHint') }}</p>
          </div>
        </div>

        <div v-if="showEditModal" class="space-y-1.5">
          <Label>{{ t('keys.statusLabel') }}</Label>
          <Select
            v-model="formData.status"
            :options="statusOptions"
            :placeholder="t('keys.selectStatus')"
          />
        </div>

        <!-- IP Restriction Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <Label class="mb-0">{{ t('keys.ipRestriction') }}</Label>
            <Switch
              v-model="formData.enable_ip_restriction"
            />
          </div>

          <div v-if="formData.enable_ip_restriction" class="space-y-4 pt-2">
            <div class="space-y-1.5">
              <Label>{{ t('keys.ipWhitelist') }}</Label>
              <Textarea
                v-model="formData.ip_whitelist"
                :rows="3"
                class="font-mono"
                :placeholder="t('keys.ipWhitelistPlaceholder')"
              />
              <p class="text-xs text-muted-foreground">{{ t('keys.ipWhitelistHint') }}</p>
            </div>

            <div class="space-y-1.5">
              <Label>{{ t('keys.ipBlacklist') }}</Label>
              <Textarea
                v-model="formData.ip_blacklist"
                :rows="3"
                class="font-mono"
                :placeholder="t('keys.ipBlacklistPlaceholder')"
              />
              <p class="text-xs text-muted-foreground">{{ t('keys.ipBlacklistHint') }}</p>
            </div>
          </div>
        </div>

        <!-- Quota Limit Section -->
        <div class="space-y-3">
          <Label>{{ t('keys.quotaLimit') }}</Label>
          <div class="space-y-4">
            <div class="space-y-1.5">
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">$</span>
                <Input
                  v-model.number="formData.quota"
                  type="number"
                  step="0.01"
                  min="0"
                  class="pl-7"
                  :placeholder="t('keys.quotaAmountPlaceholder')"
                />
              </div>
              <p class="text-xs text-muted-foreground">{{ t('keys.quotaAmountHint') }}</p>
            </div>

            <!-- Quota used display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey && selectedKey.quota > 0" class="space-y-1.5">
              <Label>{{ t('keys.quotaUsed') }}</Label>
              <div class="flex items-center gap-2">
                <div class="flex-1 rounded-md bg-muted px-3 py-2">
                  <span class="font-medium text-foreground">
                    ${{ selectedKey.quota_used?.toFixed(4) || '0.0000' }}
                  </span>
                  <span class="mx-2 text-muted-foreground">/</span>
                  <span class="text-muted-foreground">
                    ${{ selectedKey.quota?.toFixed(2) || '0.00' }}
                  </span>
                </div>
                <Button
                  type="button"
                  @click="confirmResetQuota"
                  variant="secondary"
                  size="sm"
                  :title="t('keys.resetQuotaUsed')"
                >
                  {{ t('keys.reset') }}
                </Button>
              </div>
            </div>
          </div>
        </div>

        <!-- Rate Limit Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <Label class="mb-0">{{ t('keys.rateLimitSection') }}</Label>
            <button
              type="button"
              @click="formData.enable_rate_limit = !formData.enable_rate_limit"
              :class="['switch', formData.enable_rate_limit ? 'switch-active' : '']"
            >
              <span
                :class="['switch-thumb', formData.enable_rate_limit ? 'translate-x-5' : 'translate-x-0']"
              />
            </button>
          </div>

          <div v-if="formData.enable_rate_limit" class="space-y-4 pt-2">
            <p class="-mt-2 text-xs text-muted-foreground">{{ t('keys.rateLimitHint') }}</p>
            <!-- 5-Hour Limit -->
            <div class="space-y-1.5">
              <Label>{{ t('keys.rateLimit5h') }}</Label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">$</span>
                <Input
                  v-model.number="formData.rate_limit_5h"
                  type="number"
                  step="0.01"
                  min="0"
                  class="pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_5h > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-md bg-muted px-3 py-2 text-sm">
                    <span :class="['font-medium', usageToneClass(selectedKey.usage_5h, selectedKey.rate_limit_5h)]">
                      ${{ selectedKey.usage_5h?.toFixed(4) || '0.0000' }}
                    </span>
                    <span class="mx-2 text-muted-foreground">/</span>
                    <span class="text-muted-foreground">
                      ${{ selectedKey.rate_limit_5h?.toFixed(2) || '0.00' }}
                    </span>
                  </div>
                </div>
                <div class="metric-progress mt-1">
                  <div
                    :class="['metric-progress-bar', usageProgressClass(selectedKey.usage_5h, selectedKey.rate_limit_5h)]"
                    :style="{ width: Math.min((selectedKey.usage_5h / selectedKey.rate_limit_5h) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Daily Limit -->
            <div class="space-y-1.5">
              <Label>{{ t('keys.rateLimit1d') }}</Label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">$</span>
                <Input
                  v-model.number="formData.rate_limit_1d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_1d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-md bg-muted px-3 py-2 text-sm">
                    <span :class="['font-medium', usageToneClass(selectedKey.usage_1d, selectedKey.rate_limit_1d)]">
                      ${{ selectedKey.usage_1d?.toFixed(4) || '0.0000' }}
                    </span>
                    <span class="mx-2 text-muted-foreground">/</span>
                    <span class="text-muted-foreground">
                      ${{ selectedKey.rate_limit_1d?.toFixed(2) || '0.00' }}
                    </span>
                  </div>
                </div>
                <div class="metric-progress mt-1">
                  <div
                    :class="['metric-progress-bar', usageProgressClass(selectedKey.usage_1d, selectedKey.rate_limit_1d)]"
                    :style="{ width: Math.min((selectedKey.usage_1d / selectedKey.rate_limit_1d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- 7-Day Limit -->
            <div class="space-y-1.5">
              <Label>{{ t('keys.rateLimit7d') }}</Label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">$</span>
                <Input
                  v-model.number="formData.rate_limit_7d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="pl-7"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_7d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="flex-1 rounded-md bg-muted px-3 py-2 text-sm">
                    <span :class="['font-medium', usageToneClass(selectedKey.usage_7d, selectedKey.rate_limit_7d)]">
                      ${{ selectedKey.usage_7d?.toFixed(4) || '0.0000' }}
                    </span>
                    <span class="mx-2 text-muted-foreground">/</span>
                    <span class="text-muted-foreground">
                      ${{ selectedKey.rate_limit_7d?.toFixed(2) || '0.00' }}
                    </span>
                  </div>
                </div>
                <div class="metric-progress mt-1">
                  <div
                    :class="['metric-progress-bar', usageProgressClass(selectedKey.usage_7d, selectedKey.rate_limit_7d)]"
                    :style="{ width: Math.min((selectedKey.usage_7d / selectedKey.rate_limit_7d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Reset Rate Limit button (edit mode only) -->
            <div v-if="showEditModal && selectedKey && (selectedKey.rate_limit_5h > 0 || selectedKey.rate_limit_1d > 0 || selectedKey.rate_limit_7d > 0)">
              <Button
                type="button"
                @click="confirmResetRateLimit"
                variant="secondary"
                size="sm"
              >
                {{ t('keys.resetRateLimitUsage') }}
              </Button>
            </div>
          </div>
        </div>

        <!-- Expiration Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <Label class="mb-0">{{ t('keys.expiration') }}</Label>
            <Switch
              v-model="formData.enable_expiration"
            />
          </div>

          <div v-if="formData.enable_expiration" class="space-y-4 pt-2">
            <!-- Quick select buttons (for both create and edit mode) -->
            <div class="flex flex-wrap gap-2">
              <button
                v-for="days in ['7', '30', '90']"
                :key="days"
                type="button"
                @click="setExpirationDays(parseInt(days))"
                :class="[
                  'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                  formData.expiration_preset === days
                    ? 'bg-brand text-brand-foreground'
                    : 'bg-muted text-muted-foreground hover:bg-secondary'
                ]"
              >
                {{ showEditModal ? t('keys.extendDays', { days }) : t('keys.expiresInDays', { days }) }}
              </button>
              <button
                type="button"
                @click="formData.expiration_preset = 'custom'"
                :class="[
                  'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                  formData.expiration_preset === 'custom'
                    ? 'bg-brand text-brand-foreground'
                    : 'bg-muted text-muted-foreground hover:bg-secondary'
                ]"
              >
                {{ t('keys.customDate') }}
              </button>
            </div>

            <!-- Date picker (always show for precise adjustment) -->
            <div class="space-y-1.5">
              <Label>{{ t('keys.expirationDate') }}</Label>
              <Input
                v-model="formData.expiration_date"
                type="datetime-local"
              />
              <p class="text-xs text-muted-foreground">{{ t('keys.expirationDateHint') }}</p>
            </div>

            <!-- Current expiration display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey?.expires_at" class="text-sm">
              <span class="text-muted-foreground">{{ t('keys.currentExpiration') }}: </span>
              <span class="font-medium text-foreground">
                {{ formatDateTime(selectedKey.expires_at) }}
              </span>
            </div>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <Button variant="secondary" @click="closeModals">
            {{ t('common.cancel') }}
          </Button>
          <Button
            form="key-form"
            type="submit"
            :disabled="submitting"
            data-tour="key-form-submit"
          >
            <Loader2 v-if="submitting" class="animate-spin" />
            {{
              submitting
                ? t('keys.saving')
                : showEditModal
                  ? t('common.update')
                  : t('common.create')
            }}
          </Button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('keys.deleteKey')"
      :message="t('keys.deleteConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Reset Quota Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetQuotaDialog"
      :title="t('keys.resetQuotaTitle')"
      :message="t('keys.resetQuotaConfirmMessage', { name: selectedKey?.name, used: selectedKey?.quota_used?.toFixed(4) })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetQuotaUsed"
      @cancel="showResetQuotaDialog = false"
    />

    <!-- Reset Rate Limit Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetRateLimitDialog"
      :title="t('keys.resetRateLimitTitle')"
      :message="t('keys.resetRateLimitConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetRateLimitUsage"
      @cancel="showResetRateLimitDialog = false"
    />

    <!-- Use Key Modal -->
    <UseKeyModal
      :show="showUseKeyModal"
      :api-key="selectedKey?.key || ''"
      :base-url="publicSettings?.api_base_url || ''"
      :platform="displayGroupForKey(selectedKey)?.platform || null"
      :allow-messages-dispatch="displayGroupForKey(selectedKey)?.allow_messages_dispatch || false"
      @close="closeUseKeyModal"
    />

    <!-- CCS Client Selection Dialog for Antigravity -->
    <BaseDialog
      :show="showCcsClientSelect"
      :title="t('keys.ccsClientSelect.title')"
      width="narrow"
      @close="closeCcsClientSelect"
    >
      <div class="space-y-4">
        <p class="text-sm text-muted-foreground">
          {{ t('keys.ccsClientSelect.description') }}
        </p>
        <div class="grid grid-cols-2 gap-3">
          <button
            @click="handleCcsClientSelect('claude')"
            class="flex flex-col items-center gap-2 rounded-xl border-2 border-border p-4 transition-colors hover:border-brand hover:bg-brand/5"
          >
            <Terminal class="h-6 w-6 text-muted-foreground" />
            <span class="font-medium text-foreground">{{
              t('keys.ccsClientSelect.claudeCode')
            }}</span>
            <span class="text-xs text-muted-foreground">{{
              t('keys.ccsClientSelect.claudeCodeDesc')
            }}</span>
          </button>
          <button
            @click="handleCcsClientSelect('gemini')"
            class="flex flex-col items-center gap-2 rounded-xl border-2 border-border p-4 transition-colors hover:border-brand hover:bg-brand/5"
          >
            <Sparkles class="h-6 w-6 text-muted-foreground" />
            <span class="font-medium text-foreground">{{
              t('keys.ccsClientSelect.geminiCli')
            }}</span>
            <span class="text-xs text-muted-foreground">{{
              t('keys.ccsClientSelect.geminiCliDesc')
            }}</span>
          </button>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end">
          <Button variant="secondary" @click="closeCcsClientSelect">
            {{ t('common.cancel') }}
          </Button>
        </div>
      </template>
    </BaseDialog>

    <!-- Group Selector Dropdown (Teleported to body to avoid overflow clipping) -->
    <Teleport to="body">
      <div
        v-if="groupSelectorKeyId !== null && dropdownPosition"
        ref="dropdownRef"
        class="swiss-panel animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-max max-w-[calc(100vw-16px)] overflow-hidden duration-200 sm:min-w-[380px]"
        style="pointer-events: auto !important;"
        :style="{
          top: dropdownPosition.top !== undefined ? dropdownPosition.top + 'px' : undefined,
          bottom: dropdownPosition.bottom !== undefined ? dropdownPosition.bottom + 'px' : undefined,
          left: dropdownPosition.left + 'px'
        }"
      >
        <!-- Search box -->
        <div class="border-b border-border p-2">
          <div class="relative">
            <svg class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              v-model="groupSearchQuery"
              type="text"
              class="w-full rounded-md border border-border bg-muted py-1.5 pl-8 pr-3 text-sm text-foreground placeholder:text-muted-foreground outline-none focus:border-ring focus:ring-1 focus:ring-ring"
              :placeholder="t('keys.searchGroup')"
              @click.stop
            />
          </div>
        </div>
        <!-- Group list -->
        <div class="max-h-80 overflow-y-auto p-1.5">
          <button
            v-for="option in filteredGroupOptions"
            :key="option.value ?? 'null'"
            @click="changeGroup(selectedKeyForGroup!, groupSelectorSlot!, option.value)"
            :class="[
              'flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-sm transition-colors',
              'border-b border-border/60 last:border-0',
              selectedGroupIdForDropdown === option.value ||
              (selectedGroupIdForDropdown === null && option.value === null)
                ? 'bg-brand/10'
                : 'hover:bg-muted'
            ]"
            :title="option.description || undefined"
          >
            <GroupOptionItem
              :name="option.label"
              :platform="option.platform"
              :subscription-type="option.subscriptionType"
              :rate-multiplier="option.rate"
              :user-rate-multiplier="option.userRate"
              :peak-rate-enabled="option.peakRateEnabled"
              :peak-start="option.peakStart"
              :peak-end="option.peakEnd"
              :peak-rate-multiplier="option.peakRateMultiplier"
              :description="option.description"
              :selected="
                selectedGroupIdForDropdown === option.value ||
                (selectedGroupIdForDropdown === null && option.value === null)
              "
            />
          </button>
          <!-- Empty state when search has no results -->
          <div v-if="filteredGroupOptions.length === 0" class="py-4 text-center text-sm text-muted-foreground">
            {{ t('keys.noGroupFound') }}
          </div>
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
	import { ref, reactive, computed, onMounted, onUnmounted, type ComponentPublicInstance } from 'vue'
	import { useI18n } from 'vue-i18n'
	import { useAppStore } from '@/stores/app'
	import { useOnboardingStore } from '@/stores/onboarding'
	import { useClipboard } from '@/composables/useClipboard'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'

const { t } = useI18n()
import { keysAPI, authAPI, usageAPI, userGroupsAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
	import Pagination from '@/components/common/Pagination.vue'
	import BaseDialog from '@/components/common/BaseDialog.vue'
	import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
	import EmptyState from '@/components/common/EmptyState.vue'
	import Select from '@/components/common/Select.vue'
	import SearchInput from '@/components/common/SearchInput.vue'
	import Icon from '@/components/icons/Icon.vue'
	import UseKeyModal from '@/components/keys/UseKeyModal.vue'
	import GroupBadge from '@/components/common/GroupBadge.vue'
	import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
	import { Button } from '@/components/ui/button'
	import { Input } from '@/components/ui/input'
	import { Textarea } from '@/components/ui/textarea'
	import { Label } from '@/components/ui/label'
	import { Switch } from '@/components/ui/switch'
	import {
	  RefreshCw,
	  Columns3,
	  Plus,
	  Terminal,
	  Loader2,
	  Check,
	  Clipboard,
	  Shield,
	  Sparkles,
	  ExternalLink,
	  Zap
	} from 'lucide-vue-next'
	import type { ApiKey, Group, PublicSettings, SubscriptionType, GroupPlatform, UpdateApiKeyRequest } from '@/types'
import type { Column } from '@/components/common/types'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'
import { sanitizeUrl } from '@/utils/url'
import {
  buildCcSwitchImportDeeplink,
  type CcSwitchClientType
} from '@/utils/ccswitchImport'

// Helper to format date for datetime-local input
const formatDateTimeLocal = (isoDate: string): string => {
  const date = new Date(isoDate)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

interface GroupOption {
  value: number
  label: string
  description: string | null
  rate: number
  userRate: number | null
  peakRateEnabled: boolean
  peakStart: string
  peakEnd: string
  peakRateMultiplier: number
  subscriptionType: SubscriptionType
  platform: GroupPlatform
}

const appStore = useAppStore()
const onboardingStore = useOnboardingStore()
const { copyToClipboard: clipboardCopy } = useClipboard()

const allColumns = computed<Column[]>(() => [
  { key: 'name', label: t('common.name'), sortable: true },
  { key: 'id', label: t('keys.id'), sortable: true },
  { key: 'key', label: t('keys.apiKey'), sortable: false },
  { key: 'primary_group', label: t('keys.primaryGroup'), sortable: false, class: 'w-56' },
  { key: 'group', label: t('keys.secondaryGroup'), sortable: false, class: 'w-56' },
  { key: 'current_concurrency', label: t('keys.currentConcurrency'), sortable: true },
  { key: 'usage', label: t('keys.usage'), sortable: false },
  { key: 'rate_limit', label: t('keys.rateLimitColumn'), sortable: false },
  { key: 'expires_at', label: t('keys.expiresAt'), sortable: true },
  { key: 'status', label: t('common.status'), sortable: true },
  { key: 'last_used_at', label: t('keys.lastUsedAt'), sortable: true },
  { key: 'last_used_ip', label: t('keys.lastUsedIP'), sortable: false },
  { key: 'created_at', label: t('keys.created'), sortable: true },
  { key: 'actions', label: t('common.actions'), sortable: false }
])

const ALWAYS_VISIBLE_COLUMNS = new Set(['name', 'actions'])
const DEFAULT_HIDDEN_COLUMNS = ['id', 'rate_limit', 'last_used_at', 'last_used_ip']
const HIDDEN_COLUMNS_KEY = 'api-key-hidden-columns'
const COLUMN_SETTINGS_VERSION_KEY = 'api-key-column-settings-version'
const COLUMN_SETTINGS_VERSION = 4
const VERSION_NEW_HIDDEN_COLUMNS: Record<number, string[]> = {
  2: ['primary_group', 'last_used_ip'],
  // Version 4 ports the upstream optional ID column (upstream bump 2 -> 3,
  // but our fork already uses 3 for the primary_group restore migration).
  4: ['id']
}
const VERSION_NEW_VISIBLE_COLUMNS: Record<number, string[]> = {
  // Version 2 accidentally hid the primary group for every user. Restore it
  // once during migration while preserving all other column preferences.
  3: ['primary_group']
}

const toggleableColumns = computed(() =>
  allColumns.value.filter((col) => !ALWAYS_VISIBLE_COLUMNS.has(col.key))
)

const hiddenColumns = reactive<Set<string>>(new Set())

const saveColumnsToStorage = () => {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
    localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
  } catch (error) {
    console.error('Failed to save API key table columns:', error)
  }
}

const loadSavedColumns = () => {
  hiddenColumns.clear()
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (saved) {
      const parsed = JSON.parse(saved) as string[]
      const validColumnKeys = new Set(allColumns.value.map((col) => col.key))
      parsed
        .filter((key) =>
          typeof key === 'string' &&
          validColumnKeys.has(key) &&
          !ALWAYS_VISIBLE_COLUMNS.has(key)
        )
        .forEach((key) => hiddenColumns.add(key))
      const storedVersion = Number(localStorage.getItem(COLUMN_SETTINGS_VERSION_KEY) ?? '1')
      if (storedVersion < COLUMN_SETTINGS_VERSION) {
        for (let v = storedVersion + 1; v <= COLUMN_SETTINGS_VERSION; v++) {
          for (const key of VERSION_NEW_HIDDEN_COLUMNS[v] ?? []) {
            if (validColumnKeys.has(key) && !ALWAYS_VISIBLE_COLUMNS.has(key)) {
              hiddenColumns.add(key)
            }
          }
          for (const key of VERSION_NEW_VISIBLE_COLUMNS[v] ?? []) {
            hiddenColumns.delete(key)
          }
        }
        saveColumnsToStorage()
      } else {
        localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
      }
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
      localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
    }
  } catch (error) {
    console.error('Failed to load API key table columns:', error)
    DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
  }
}

const toggleColumn = (key: string) => {
  if (ALWAYS_VISIBLE_COLUMNS.has(key)) return
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  saveColumnsToStorage()
}

const isColumnVisible = (key: string) => !hiddenColumns.has(key)

const columns = computed<Column[]>(() =>
  allColumns.value.filter((col) => ALWAYS_VISIBLE_COLUMNS.has(col.key) || !hiddenColumns.has(col.key))
)

const apiKeys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
const submitting = ref(false)
const now = ref(new Date())
let resetTimer: ReturnType<typeof setInterval> | null = null
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({})
const userGroupRates = ref<Record<number, number>>({})

const pagination = ref({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})
const sortState = ref({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

// Filter state
const filterSearch = ref('')
const filterStatus = ref('')
const filterGroupId = ref<string | number>('')

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showResetQuotaDialog = ref(false)
const showResetRateLimitDialog = ref(false)
const showUseKeyModal = ref(false)
const showCcsClientSelect = ref(false)
const showColumnDropdown = ref(false)
const pendingCcsRow = ref<ApiKey | null>(null)
const selectedKey = ref<ApiKey | null>(null)
const copiedKeyId = ref<number | null>(null)
const groupSelectorKeyId = ref<number | null>(null)
const groupSelectorSlot = ref<'primary' | 'secondary' | null>(null)
const publicSettings = ref<PublicSettings | null>(null)
const sanitizedDocUrl = computed(() => sanitizeUrl(publicSettings.value?.doc_url ?? ''))
const dropdownRef = ref<HTMLElement | null>(null)
const columnDropdownRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<{ top?: number; bottom?: number; left: number } | null>(null)
const groupButtonRefs = ref<Map<string, HTMLElement>>(new Map())
let abortController: AbortController | null = null

// Get the currently selected key for group change
const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null
})

const selectedGroupIdForDropdown = computed(() => {
  if (!selectedKeyForGroup.value || !groupSelectorSlot.value) return null
  return groupSelectorSlot.value === 'primary'
    ? selectedKeyForGroup.value.primary_group_id
    : selectedKeyForGroup.value.group_id
})

const groupRefKey = (keyId: number, slot: 'primary' | 'secondary') => `${keyId}:${slot}`

const setGroupButtonRef = (keyId: number, slot: 'primary' | 'secondary', el: Element | ComponentPublicInstance | null) => {
  const refKey = groupRefKey(keyId, slot)
  if (el instanceof HTMLElement) {
    groupButtonRefs.value.set(refKey, el)
  } else {
    groupButtonRefs.value.delete(refKey)
  }
}

const formData = ref({
  name: '',
  primary_group_id: null as number | null,
  group_id: null as number | null,
  status: 'active' as 'active' | 'inactive',
  use_custom_key: false,
  custom_key: '',
  enable_ip_restriction: false,
  ip_whitelist: '',
  ip_blacklist: '',
  // Quota settings (empty = unlimited)
  enable_quota: false,
  quota: null as number | null,
  // Rate limit settings
  enable_rate_limit: false,
  rate_limit_5h: null as number | null,
  rate_limit_1d: null as number | null,
  rate_limit_7d: null as number | null,
  enable_expiration: false,
  expiration_preset: '30' as '7' | '30' | '90' | 'custom',
  expiration_date: ''
})

// 自定义Key验证
const customKeyError = computed(() => {
  if (!formData.value.use_custom_key || !formData.value.custom_key) {
    return ''
  }
  const key = formData.value.custom_key
  if (key.length < 16) {
    return t('keys.customKeyTooShort')
  }
  // 检查字符：只允许字母、数字、下划线、连字符
  if (!/^[a-zA-Z0-9_-]+$/.test(key)) {
    return t('keys.customKeyInvalidChars')
  }
  return ''
})

const statusOptions = computed(() => [
  { value: 'active', label: t('common.active') },
  { value: 'inactive', label: t('common.inactive') }
])

const shouldSubmitEditStatus = (key: ApiKey, status: 'active' | 'inactive') => {
  if (key.status === 'quota_exhausted' || key.status === 'expired') {
    return status === 'active'
  }
  return true
}

// Filter dropdown options
const groupFilterOptions = computed(() => [
  { value: '', label: t('keys.allGroups') },
  { value: 0, label: t('keys.noGroup') },
  ...groups.value.map((g) => ({ value: g.id, label: g.name }))
])

const statusFilterOptions = computed(() => [
  { value: '', label: t('keys.allStatus') },
  { value: 'active', label: t('keys.status.active') },
  { value: 'inactive', label: t('keys.status.inactive') },
  { value: 'quota_exhausted', label: t('keys.status.quota_exhausted') },
  { value: 'expired', label: t('keys.status.expired') }
])

const onFilterChange = () => {
  pagination.value.page = 1
  loadApiKeys()
}

const onGroupFilterChange = (value: string | number | boolean | null) => {
  filterGroupId.value = value as string | number
  onFilterChange()
}

const onStatusFilterChange = (value: string | number | boolean | null) => {
  filterStatus.value = value as string
  onFilterChange()
}

// Convert groups to Select options format with rate multiplier and subscription type
const groupOptions = computed(() =>
  groups.value.map((group) => ({
    value: group.id,
    label: group.name,
    description: group.description,
    rate: group.rate_multiplier,
    userRate: userGroupRates.value[group.id] ?? null,
    peakRateEnabled: group.peak_rate_enabled,
    peakStart: group.peak_start,
    peakEnd: group.peak_end,
    peakRateMultiplier: group.peak_rate_multiplier,
    subscriptionType: group.subscription_type,
    platform: group.platform
  }))
)

// Group dropdown search
const groupSearchQuery = ref('')
const filteredGroupOptions = computed(() => {
  const query = groupSearchQuery.value.trim().toLowerCase()
  if (!query) return groupOptions.value
  return groupOptions.value.filter((opt) => {
    return opt.label.toLowerCase().includes(query) ||
      (opt.description && opt.description.toLowerCase().includes(query))
  })
})

const copyToClipboard = async (text: string, keyId: number) => {
  const success = await clipboardCopy(text, t('keys.copied'))
  if (success) {
    copiedKeyId.value = keyId
    setTimeout(() => {
      copiedKeyId.value = null
    }, 800)
  }
}

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') return false
  const { name, code } = error as { name?: string; code?: string }
  return name === 'AbortError' || code === 'ERR_CANCELED'
}

const loadApiKeys = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  const { signal } = controller
  loading.value = true
  try {
    // Build filters
    const filters: {
      search?: string
      status?: string
      group_id?: number | string
      sort_by?: string
      sort_order?: 'asc' | 'desc'
    } = {}
    if (filterSearch.value) filters.search = filterSearch.value
    if (filterStatus.value) filters.status = filterStatus.value
    if (filterGroupId.value !== '') filters.group_id = filterGroupId.value
    filters.sort_by = sortState.value.sort_by
    filters.sort_order = sortState.value.sort_order

    const response = await keysAPI.list(pagination.value.page, pagination.value.page_size, filters, {
      signal
    })
    if (signal.aborted) return
    apiKeys.value = response.items
    pagination.value.total = response.total
    pagination.value.pages = response.pages

    // Load usage stats for all API keys in the list
    if (response.items.length > 0) {
      const keyIds = response.items.map((k) => k.id)
      try {
        const usageResponse = await usageAPI.getDashboardApiKeysUsage(keyIds, { signal })
        if (signal.aborted) return
        usageStats.value = usageResponse.stats
      } catch (e) {
        if (!isAbortError(e)) {
          console.error('Failed to load usage stats:', e)
        }
      }
    }
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    appStore.showError(t('keys.failedToLoad'))
  } finally {
    if (abortController === controller) {
      loading.value = false
    }
  }
}

const loadGroups = async () => {
  try {
    groups.value = await userGroupsAPI.getAvailable()
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const loadUserGroupRates = async () => {
  try {
    userGroupRates.value = await userGroupsAPI.getUserGroupRates()
  } catch (error) {
    console.error('Failed to load user group rates:', error)
  }
}

const loadPublicSettings = async () => {
  try {
    publicSettings.value = await authAPI.getPublicSettings()
  } catch (error) {
    console.error('Failed to load public settings:', error)
  }
}

// Endpoint info card rows (default base URL + custom endpoints)
const endpointRows = computed(() => {
  const rows: Array<{ name: string; endpoint: string; description: string; isDefault: boolean }> = []
  if (publicSettings.value?.api_base_url) {
    rows.push({
      name: t('keys.endpoints.title'),
      endpoint: publicSettings.value.api_base_url,
      description: '',
      isDefault: true
    })
  }
  for (const ep of publicSettings.value?.custom_endpoints ?? []) {
    rows.push({ name: ep.name, endpoint: ep.endpoint, description: ep.description, isDefault: false })
  }
  return rows
})

const copiedEndpoint = ref<string | null>(null)
let endpointCopyTimer: ReturnType<typeof setTimeout> | undefined

const copyEndpoint = async (endpoint: string) => {
  const success = await clipboardCopy(endpoint, t('keys.endpoints.copied'))
  if (!success) return
  copiedEndpoint.value = endpoint
  if (endpointCopyTimer !== undefined) {
    clearTimeout(endpointCopyTimer)
  }
  endpointCopyTimer = setTimeout(() => {
    if (copiedEndpoint.value === endpoint) {
      copiedEndpoint.value = null
    }
  }, 1800)
}

const endpointSpeedTestUrl = (endpoint: string) =>
  `https://www.tcptest.cn/http/${encodeURIComponent(endpoint)}`

const openUseKeyModal = (key: ApiKey) => {
  selectedKey.value = key
  showUseKeyModal.value = true
}

const closeUseKeyModal = () => {
  showUseKeyModal.value = false
  selectedKey.value = null
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadApiKeys()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.page_size = pageSize
  pagination.value.page = 1
  loadApiKeys()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.value.sort_by = key
  sortState.value.sort_order = order
  pagination.value.page = 1
  loadApiKeys()
}

const editKey = (key: ApiKey) => {
  selectedKey.value = key
  const hasIPRestriction = (key.ip_whitelist?.length > 0) || (key.ip_blacklist?.length > 0)
  const hasExpiration = !!key.expires_at
  formData.value = {
    name: key.name,
    primary_group_id: key.primary_group_id,
    group_id: key.group_id,
    status: key.status === 'quota_exhausted' || key.status === 'expired' ? 'inactive' : key.status,
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: hasIPRestriction,
    ip_whitelist: (key.ip_whitelist || []).join('\n'),
    ip_blacklist: (key.ip_blacklist || []).join('\n'),
    enable_quota: key.quota > 0,
    quota: key.quota > 0 ? key.quota : null,
    enable_rate_limit: (key.rate_limit_5h > 0) || (key.rate_limit_1d > 0) || (key.rate_limit_7d > 0),
    rate_limit_5h: key.rate_limit_5h || null,
    rate_limit_1d: key.rate_limit_1d || null,
    rate_limit_7d: key.rate_limit_7d || null,
    enable_expiration: hasExpiration,
    expiration_preset: 'custom',
    expiration_date: key.expires_at ? formatDateTimeLocal(key.expires_at) : ''
  }
  showEditModal.value = true
}

const toggleKeyStatus = async (key: ApiKey) => {
  const newStatus = key.status === 'active' ? 'inactive' : 'active'
  try {
    await keysAPI.toggleStatus(key.id, newStatus)
    appStore.showSuccess(
      newStatus === 'active' ? t('keys.keyEnabledSuccess') : t('keys.keyDisabledSuccess')
    )
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToUpdateStatus'))
  }
}

const openGroupSelector = (key: ApiKey, slot: 'primary' | 'secondary') => {
  if (groupSelectorKeyId.value === key.id && groupSelectorSlot.value === slot) {
    groupSelectorKeyId.value = null
    groupSelectorSlot.value = null
    dropdownPosition.value = null
  } else {
    const buttonEl = groupButtonRefs.value.get(groupRefKey(key.id, slot))
    if (buttonEl) {
      const rect = buttonEl.getBoundingClientRect()
      const dropdownEstHeight = 400 // estimated max dropdown height
      const dropdownEstWidth = Math.min(380, window.innerWidth - 16)
      const spaceBelow = window.innerHeight - rect.bottom
      const spaceAbove = rect.top
      // 夹取 left，避免窄屏下浮层超出视口右缘
      const left = Math.max(8, Math.min(rect.left, window.innerWidth - dropdownEstWidth - 8))

      if (spaceBelow < dropdownEstHeight && spaceAbove > spaceBelow) {
        // Not enough space below, pop upward
        dropdownPosition.value = {
          bottom: window.innerHeight - rect.top + 4,
          left
        }
      } else {
        // Default: pop downward
        dropdownPosition.value = {
          top: rect.bottom + 4,
          left
        }
      }
    }
    groupSelectorKeyId.value = key.id
    groupSelectorSlot.value = slot
    groupSearchQuery.value = ''
  }
}

const changeGroup = async (key: ApiKey, slot: 'primary' | 'secondary', newGroupId: number | null) => {
  groupSelectorKeyId.value = null
  groupSelectorSlot.value = null
  dropdownPosition.value = null
  const currentGroupId = slot === 'primary' ? key.primary_group_id : key.group_id
  const otherGroupId = slot === 'primary' ? key.group_id : key.primary_group_id
  if (currentGroupId === newGroupId) return
  if (newGroupId !== null && otherGroupId === newGroupId) {
    appStore.showError(t('keys.groupsMustDiffer'))
    return
  }

  try {
    await keysAPI.update(key.id, slot === 'primary' ? { primary_group_id: newGroupId } : { group_id: newGroupId })
    appStore.showSuccess(t('keys.groupChangedSuccess'))
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToChangeGroup'))
  }
}

const closeGroupSelector = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // Check if click is inside the dropdown or the trigger button
  if (!target.closest('.group\\/dropdown') && !dropdownRef.value?.contains(target)) {
    groupSelectorKeyId.value = null
    groupSelectorSlot.value = null
    dropdownPosition.value = null
  }
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(target)) {
    showColumnDropdown.value = false
  }
}

const confirmDelete = (key: ApiKey) => {
  selectedKey.value = key
  showDeleteDialog.value = true
}

const handleSubmit = async () => {
  if (formData.value.primary_group_id === null && formData.value.group_id === null) {
    appStore.showError(t('keys.groupRequired'))
    return
  }
  if (
    formData.value.primary_group_id !== null &&
    formData.value.group_id !== null &&
    formData.value.primary_group_id === formData.value.group_id
  ) {
    appStore.showError(t('keys.groupsMustDiffer'))
    return
  }

  // Validate custom key if enabled
  if (!showEditModal.value && formData.value.use_custom_key) {
    if (!formData.value.custom_key) {
      appStore.showError(t('keys.customKeyRequired'))
      return
    }
    if (customKeyError.value) {
      appStore.showError(customKeyError.value)
      return
    }
  }

  // Parse IP lists only if IP restriction is enabled
  const parseIPList = (text: string): string[] =>
    text.split('\n').map(ip => ip.trim()).filter(ip => ip.length > 0)
  const ipWhitelist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_whitelist) : []
  const ipBlacklist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_blacklist) : []

  // Calculate quota value (null/empty/0 = unlimited, stored as 0)
  const quota = formData.value.quota && formData.value.quota > 0 ? formData.value.quota : 0

  // Calculate expiration
  let expiresInDays: number | undefined
  let expiresAt: string | null | undefined
  if (formData.value.enable_expiration && formData.value.expiration_date) {
    if (!showEditModal.value) {
      // Create mode: calculate days from date
      const expDate = new Date(formData.value.expiration_date)
      const now = new Date()
      const diffDays = Math.ceil((expDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
      expiresInDays = diffDays > 0 ? diffDays : 1
    } else {
      // Edit mode: use custom date directly
      expiresAt = new Date(formData.value.expiration_date).toISOString()
    }
  } else if (showEditModal.value) {
    // Edit mode: if expiration disabled or date cleared, send empty string to clear
    expiresAt = ''
  }

  // Calculate rate limit values (send 0 when toggle is off)
  const rateLimitData = formData.value.enable_rate_limit ? {
    rate_limit_5h: formData.value.rate_limit_5h && formData.value.rate_limit_5h > 0 ? formData.value.rate_limit_5h : 0,
    rate_limit_1d: formData.value.rate_limit_1d && formData.value.rate_limit_1d > 0 ? formData.value.rate_limit_1d : 0,
    rate_limit_7d: formData.value.rate_limit_7d && formData.value.rate_limit_7d > 0 ? formData.value.rate_limit_7d : 0,
  } : { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 }

  submitting.value = true
  try {
    if (showEditModal.value && selectedKey.value) {
      const updates: UpdateApiKeyRequest = {
        name: formData.value.name,
        primary_group_id: formData.value.primary_group_id,
        group_id: formData.value.group_id,
        ip_whitelist: ipWhitelist,
        ip_blacklist: ipBlacklist,
        quota: quota,
        expires_at: expiresAt,
        rate_limit_5h: rateLimitData.rate_limit_5h,
        rate_limit_1d: rateLimitData.rate_limit_1d,
        rate_limit_7d: rateLimitData.rate_limit_7d,
      }
      if (shouldSubmitEditStatus(selectedKey.value, formData.value.status)) {
        updates.status = formData.value.status
      }
      await keysAPI.update(selectedKey.value.id, updates)
      appStore.showSuccess(t('keys.keyUpdatedSuccess'))
    } else {
      const customKey = formData.value.use_custom_key ? formData.value.custom_key : undefined
      await keysAPI.create(
        formData.value.name,
        formData.value.primary_group_id,
        formData.value.group_id,
        customKey,
        ipWhitelist,
        ipBlacklist,
        quota,
        expiresInDays,
        rateLimitData
      )
      appStore.showSuccess(t('keys.keyCreatedSuccess'))
      // Only advance tour if active, on submit step, and creation succeeded
      if (onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) {
        onboardingStore.nextStep(500)
      }
    }
    closeModals()
    loadApiKeys()
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToSave')
    appStore.showError(errorMsg)
    // Don't advance tour on error
  } finally {
    submitting.value = false
  }
}

/**
 * 处理删除 API Key 的操作
 * 优化：错误处理改进，优先显示后端返回的具体错误消息（如权限不足等），
 * 若后端未返回消息则显示默认的国际化文本
 */
const handleDelete = async () => {
  if (!selectedKey.value) return

  try {
    await keysAPI.delete(selectedKey.value.id)
    appStore.showSuccess(t('keys.keyDeletedSuccess'))
    showDeleteDialog.value = false
    loadApiKeys()
  } catch (error: any) {
    // 优先使用后端返回的错误消息，提供更具体的错误信息给用户
    const errorMsg = error?.message || t('keys.failedToDelete')
    appStore.showError(errorMsg)
  }
}

const closeModals = () => {
  showCreateModal.value = false
  showEditModal.value = false
  selectedKey.value = null
  formData.value = {
    name: '',
    primary_group_id: null,
    group_id: null,
    status: 'active',
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: false,
    ip_whitelist: '',
    ip_blacklist: '',
    enable_quota: false,
    quota: null,
    enable_rate_limit: false,
    rate_limit_5h: null,
    rate_limit_1d: null,
    rate_limit_7d: null,
    enable_expiration: false,
    expiration_preset: '30',
    expiration_date: ''
  }
}

// Show reset quota confirmation dialog
const confirmResetQuota = () => {
  showResetQuotaDialog.value = true
}

// Set expiration date based on quick select days
const setExpirationDays = (days: number) => {
  formData.value.expiration_preset = days.toString() as '7' | '30' | '90'
  const expDate = new Date()
  expDate.setDate(expDate.getDate() + days)
  formData.value.expiration_date = formatDateTimeLocal(expDate.toISOString())
}

// Reset quota used for an API key
const resetQuotaUsed = async () => {
  if (!selectedKey.value) return
  showResetQuotaDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_quota: true })
    appStore.showSuccess(t('keys.quotaResetSuccess'))
    // Update local state
    if (selectedKey.value) {
      selectedKey.value.quota_used = 0
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetQuota')
    appStore.showError(errorMsg)
  }
}

// Show reset rate limit confirmation dialog (from edit modal)
const confirmResetRateLimit = () => {
  showResetRateLimitDialog.value = true
}

// Show reset rate limit confirmation dialog (from table row)
const confirmResetRateLimitFromTable = (row: ApiKey) => {
  selectedKey.value = row
  showResetRateLimitDialog.value = true
}

// Reset rate limit usage for an API key
const resetRateLimitUsage = async () => {
  if (!selectedKey.value) return
  showResetRateLimitDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_rate_limit_usage: true })
    appStore.showSuccess(t('keys.rateLimitResetSuccess'))
    // Refresh key data
    await loadApiKeys()
    // Update the editing key with fresh data
    const refreshedKey = apiKeys.value.find(k => k.id === selectedKey.value!.id)
    if (refreshedKey) {
      selectedKey.value = refreshedKey
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetRateLimit')
    appStore.showError(errorMsg)
  }
}

const displayGroupForKey = (row: ApiKey | null | undefined) => row?.primary_group || row?.group || null

const importToCcswitch = (row: ApiKey) => {
  const platform = displayGroupForKey(row)?.platform || 'anthropic'

  // For antigravity platform, show client selection dialog
  if (platform === 'antigravity') {
    pendingCcsRow.value = row
    showCcsClientSelect.value = true
    return
  }

  // For other platforms, execute directly
  executeCcsImport(row, platform === 'gemini' ? 'gemini' : 'claude')
}

const executeCcsImport = (row: ApiKey, clientType: CcSwitchClientType) => {
  const baseUrl = publicSettings.value?.api_base_url || window.location.origin
  const platform = displayGroupForKey(row)?.platform || 'anthropic'

  const usageScript = `({
    request: {
      url: "{{baseUrl}}/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`
  const providerName = (publicSettings.value?.site_name || 'sub2api').trim() || 'sub2api'
  const deeplink = buildCcSwitchImportDeeplink({
    baseUrl,
    platform,
    clientType,
    providerName,
    apiKey: row.key,
    usageScript
  })

  try {
    window.open(deeplink, '_self')

    // Check if the protocol handler worked by detecting if we're still focused
    setTimeout(() => {
      if (document.hasFocus()) {
        // Still focused means the protocol handler likely failed
        appStore.showError(t('keys.ccSwitchNotInstalled'))
      }
    }, 100)
  } catch (error) {
    appStore.showError(t('keys.ccSwitchNotInstalled'))
  }
}

const handleCcsClientSelect = (clientType: CcSwitchClientType) => {
  if (pendingCcsRow.value) {
    executeCcsImport(pendingCcsRow.value, clientType)
  }
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

const closeCcsClientSelect = () => {
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

function formatResetTime(resetAt: string | null): string {
  if (!resetAt) return ''
  const diff = new Date(resetAt).getTime() - now.value.getTime()
  if (diff <= 0) return t('keys.resetNow')
  const days = Math.floor(diff / 86400000)
  const hours = Math.floor((diff % 86400000) / 3600000)
  const mins = Math.floor((diff % 3600000) / 60000)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

function usageToneClass(used: number | null | undefined, limit: number | null | undefined): string {
  if (!limit || limit <= 0) return 'text-foreground'
  const ratio = (used || 0) / limit
  if (ratio >= 1) return 'text-semantic-danger'
  if (ratio >= 0.8) return 'text-semantic-warning'
  return 'text-foreground'
}

function usageProgressClass(used: number | null | undefined, limit: number | null | undefined): string {
  if (!limit || limit <= 0) return ''
  const ratio = (used || 0) / limit
  if (ratio >= 1) return 'metric-progress-bar-danger'
  if (ratio >= 0.8) return 'metric-progress-bar-warning'
  return 'metric-progress-bar-success'
}

onMounted(() => {
  loadSavedColumns()
  loadApiKeys()
  loadGroups()
  loadUserGroupRates()
  loadPublicSettings()
  document.addEventListener('click', closeGroupSelector)
  resetTimer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  document.removeEventListener('click', closeGroupSelector)
  if (resetTimer) clearInterval(resetTimer)
  if (endpointCopyTimer !== undefined) clearTimeout(endpointCopyTimer)
})
</script>

<style scoped>
/* ===== QW pixel spec tokens (light hex from design-ref; dark via --nm-* vars) ===== */
.qw-desc {
  color: #7f8798;
}
:global(.dark) .qw-desc {
  color: var(--nm-ink-muted);
}

/* Page-level card: white, 24px radius, hairline border */
.qw-card {
  background-color: hsl(var(--card));
  border: 1px solid var(--nm-border);
  border-radius: 24px;
}

/* Endpoint URL row: h-12 rounded-xl hairline box */
.qw-endpoint-row {
  background-color: hsl(var(--card));
  border: 1px solid var(--nm-border);
  border-radius: 12px;
}

/* Endpoint chip: h-8 accent-soft pill label */
.qw-endpoint-chip {
  background-color: var(--nm-accent-soft);
  color: hsl(var(--brand));
}

/* 36x36 quiet icon button */
.qw-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: hsl(var(--muted-foreground));
  border-radius: 9999px;
  transition: background-color 160ms ease, color 160ms ease;
}
.qw-icon-btn:hover {
  background-color: var(--nm-surface-soft);
  color: hsl(var(--foreground));
}
.qw-icon-btn:disabled {
  opacity: 0.5;
  pointer-events: none;
}

/* "Learn more" accent link */
.qw-learn-more {
  color: hsl(var(--brand));
}
.qw-learn-more:hover {
  text-decoration: underline;
}

/* Search pill: h-10 w-280 rounded-full hairline border, 13px placeholder */
.qw-search :deep(input) {
  height: 40px;
  border: 1px solid var(--nm-border);
  border-radius: 9999px;
  background-color: hsl(var(--card));
  box-shadow: none;
}
.qw-search :deep(input)::placeholder {
  color: #8e96a7;
  font-size: 13px;
}
:global(.dark) .qw-search :deep(input) {
  background-color: hsl(var(--card));
  border-color: var(--nm-border);
}

/* Filter select pill: h-10 rounded-full hairline border, 13px label */
.qw-filter-select :deep(.select-trigger) {
  height: 40px;
  min-height: 40px;
  border: 1px solid var(--nm-border);
  border-radius: 9999px;
  background-color: hsl(var(--card));
}
.qw-filter-select :deep(.select-value) {
  font-size: 13px;
}

/* ===== Table spec: header row h-11 rounded-lg on #F9FAFD, th 12px/500 #707889;
     body rows h-14 with #EEF1F5 hairline, td 14px ===== */
.qw-table :deep(thead),
.qw-table :deep(.table-wrapper .table-header) {
  background: transparent;
}
.qw-table :deep(.sticky-header-cell) {
  height: 44px;
  padding-top: 0;
  padding-bottom: 0;
  background: #f9fafd;
  color: #707889;
  font-size: 12px;
  font-weight: 500;
  text-transform: none;
  letter-spacing: 0;
}
.qw-table :deep(thead tr th:first-child) {
  border-top-left-radius: 8px;
  border-bottom-left-radius: 8px;
}
.qw-table :deep(thead tr th:last-child) {
  border-top-right-radius: 8px;
  border-bottom-right-radius: 8px;
}
.qw-table :deep(tbody tr.dt-row) {
  height: 56px;
}
.qw-table :deep(tbody tr.dt-row td) {
  font-size: 14px;
  border-bottom: 1px solid #eef1f5;
}
.qw-table :deep(tbody tr.dt-row:last-child td) {
  border-bottom: none;
}

:global(.dark) .qw-table :deep(thead),
:global(.dark) .qw-table :deep(.table-wrapper .table-header),
:global(.dark) .qw-table :deep(.sticky-header-cell) {
  background: var(--nm-surface-soft);
}
:global(.dark) .qw-table :deep(.sticky-header-cell) {
  color: var(--nm-ink-faint);
}
:global(.dark) .qw-table :deep(tbody tr.dt-row td) {
  border-bottom-color: var(--nm-border-light);
}
</style>
