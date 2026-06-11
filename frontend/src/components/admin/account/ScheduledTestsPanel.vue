<template>
  <BaseDialog
    :show="show"
    :title="t('admin.scheduledTests.title')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <!-- Add Plan Button -->
      <div class="flex items-center justify-between">
        <p class="scheduled-muted text-sm">
          {{ t('admin.scheduledTests.title') }}
        </p>
        <button
          @click="showAddForm = !showAddForm"
          class="btn btn-primary flex items-center gap-1.5 text-sm"
        >
          <Icon name="plus" size="sm" :stroke-width="2" />
          {{ t('admin.scheduledTests.addPlan') }}
        </button>
      </div>

      <!-- Add Plan Form -->
      <div
        v-if="showAddForm"
        class="scheduled-form-panel p-4"
      >
        <div class="scheduled-section-title mb-3 text-sm font-medium">
          {{ t('admin.scheduledTests.addPlan') }}
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="scheduled-field-label mb-1 block text-xs font-medium">
              {{ t('admin.scheduledTests.model') }}
            </label>
            <Select
              v-model="newPlan.model_id"
              :options="modelOptions"
              :placeholder="t('admin.scheduledTests.model')"
              :searchable="modelOptions.length > 5"
            />
          </div>
          <div>
            <label class="scheduled-field-label mb-1 flex items-center gap-1 text-xs font-medium">
              {{ t('admin.scheduledTests.cronExpression') }}
              <HelpTooltip>
                <template #trigger>
                  <span class="scheduled-help-dot inline-flex h-4 w-4 cursor-help items-center justify-center text-[10px] font-semibold transition-colors">
                    ?
                  </span>
                </template>
                <div class="space-y-1.5">
                  <p class="font-medium">{{ t('admin.scheduledTests.cronTooltipTitle') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipMeaning') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipExampleEvery30Min') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipExampleHourly') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipExampleDaily') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipExampleWeekly') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipRange') }}</p>
                </div>
              </HelpTooltip>
            </label>
            <Input
              v-model="newPlan.cron_expression"
              :placeholder="'*/30 * * * *'"
              :hint="t('admin.scheduledTests.cronHelp')"
            />
          </div>
          <div>
            <label class="scheduled-field-label mb-1 flex items-center gap-1 text-xs font-medium">
              {{ t('admin.scheduledTests.maxResults') }}
              <HelpTooltip>
                <template #trigger>
                  <span class="scheduled-help-dot inline-flex h-4 w-4 cursor-help items-center justify-center text-[10px] font-semibold transition-colors">
                    ?
                  </span>
                </template>
                <div class="space-y-1.5">
                  <p class="font-medium">{{ t('admin.scheduledTests.maxResultsTooltipTitle') }}</p>
                  <p>{{ t('admin.scheduledTests.maxResultsTooltipMeaning') }}</p>
                  <p>{{ t('admin.scheduledTests.maxResultsTooltipBody') }}</p>
                  <p>{{ t('admin.scheduledTests.maxResultsTooltipExample') }}</p>
                  <p>{{ t('admin.scheduledTests.maxResultsTooltipRange') }}</p>
                </div>
              </HelpTooltip>
            </label>
            <Input
              v-model="newPlan.max_results"
              type="number"
              placeholder="100"
            />
          </div>
          <div class="flex items-end">
            <label class="scheduled-toggle-label flex items-center gap-2 text-sm">
              <Toggle v-model="newPlan.enabled" />
              {{ t('admin.scheduledTests.enabled') }}
            </label>
          </div>
          <div class="flex items-end">
            <div>
              <label class="scheduled-toggle-label flex items-center gap-2 text-sm">
                <Toggle v-model="newPlan.auto_recover" />
                {{ t('admin.scheduledTests.autoRecover') }}
              </label>
              <p class="scheduled-help-text mt-0.5 text-xs">
                {{ t('admin.scheduledTests.autoRecoverHelp') }}
              </p>
            </div>
          </div>
        </div>
        <div class="mt-3 flex justify-end gap-2">
          <button
            @click="showAddForm = false; resetNewPlan()"
            class="scheduled-button scheduled-button-secondary px-3 py-1.5 text-sm font-medium transition-colors"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            @click="handleCreate"
            :disabled="!newPlan.model_id || !newPlan.cron_expression || creating"
            class="scheduled-button scheduled-button-primary flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Icon v-if="creating" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
            {{ t('common.save') }}
          </button>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <Icon name="refresh" size="md" class="scheduled-muted animate-spin" :stroke-width="2" />
        <span class="scheduled-muted ml-2 text-sm">{{ t('common.loading') }}...</span>
      </div>

      <!-- Empty State -->
      <div
        v-else-if="plans.length === 0"
        class="scheduled-empty py-10 text-center"
      >
        <Icon name="calendar" size="lg" class="scheduled-muted mx-auto mb-2" :stroke-width="1.5" />
        <p class="scheduled-muted text-sm">
          {{ t('admin.scheduledTests.noPlans') }}
        </p>
      </div>

      <!-- Plans List -->
      <div v-else class="space-y-3">
        <div
          v-for="plan in plans"
          :key="plan.id"
          class="scheduled-plan transition-colors"
        >
          <!-- Plan Header -->
          <div
            class="flex cursor-pointer items-center justify-between px-4 py-3"
            @click="toggleExpand(plan.id)"
          >
            <div class="flex flex-1 items-center gap-4">
              <!-- Model -->
              <div class="min-w-0">
                <div class="scheduled-plan-title text-sm font-medium">
                  {{ plan.model_id }}
                </div>
                <div class="scheduled-muted mt-0.5 font-mono text-xs">
                  {{ plan.cron_expression }}
                </div>
              </div>

              <!-- Enabled Toggle -->
              <div class="flex items-center gap-1.5" @click.stop>
                <Toggle
                  :model-value="plan.enabled"
                  @update:model-value="(val: boolean) => handleToggleEnabled(plan, val)"
                />
                <span class="scheduled-muted text-xs">
                  {{ plan.enabled ? t('admin.scheduledTests.enabled') : '' }}
                </span>
              </div>

              <!-- Auto Recover Badge -->
              <span
                v-if="plan.auto_recover"
                class="scheduled-badge scheduled-badge-success inline-flex items-center px-2 py-0.5 text-xs font-medium"
              >
                {{ t('admin.scheduledTests.autoRecover') }}
              </span>
            </div>

            <div class="flex items-center gap-3">
              <!-- Last Run -->
              <div v-if="plan.last_run_at" class="scheduled-muted hidden text-right text-xs sm:block">
                <div>{{ t('admin.scheduledTests.lastRun') }}</div>
                <div>{{ formatDateTime(plan.last_run_at) }}</div>
              </div>

              <!-- Next Run -->
              <div v-if="plan.next_run_at" class="scheduled-muted hidden text-right text-xs sm:block">
                <div>{{ t('admin.scheduledTests.nextRun') }}</div>
                <div>{{ formatDateTime(plan.next_run_at) }}</div>
              </div>

              <!-- Actions -->
              <div class="flex items-center gap-1" @click.stop>
                <button
                  @click="startEdit(plan)"
                  class="scheduled-icon-button p-1.5 transition-colors"
                  :title="t('admin.scheduledTests.editPlan')"
                  :aria-label="t('admin.scheduledTests.editPlan')"
                >
                  <Icon name="edit" size="sm" :stroke-width="2" />
                </button>
                <button
                  @click="confirmDeletePlan(plan)"
                  class="scheduled-icon-button scheduled-icon-button-danger p-1.5 transition-colors"
                  :title="t('admin.scheduledTests.deletePlan')"
                  :aria-label="t('admin.scheduledTests.deletePlan')"
                >
                  <Icon name="trash" size="sm" :stroke-width="2" />
                </button>
              </div>

              <!-- Expand indicator -->
              <Icon
                name="chevronDown"
                size="sm"
                :class="[
                  'scheduled-muted transition-transform duration-200',
                  expandedPlanId === plan.id ? 'rotate-180' : ''
                ]"
              />
            </div>
          </div>

          <!-- Edit Form -->
          <div
            v-if="editingPlanId === plan.id"
            class="scheduled-edit-panel px-4 py-3"
            @click.stop
          >
            <div class="scheduled-field-label mb-2 text-xs font-medium">
              {{ t('admin.scheduledTests.editPlan') }}
            </div>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label class="scheduled-field-label mb-1 block text-xs font-medium">
                  {{ t('admin.scheduledTests.model') }}
                </label>
                <Select
                  v-model="editForm.model_id"
                  :options="modelOptions"
                  :placeholder="t('admin.scheduledTests.model')"
                  :searchable="modelOptions.length > 5"
                />
              </div>
              <div>
                <label class="scheduled-field-label mb-1 flex items-center gap-1 text-xs font-medium">
                  {{ t('admin.scheduledTests.cronExpression') }}
                  <HelpTooltip>
                    <template #trigger>
                      <span class="scheduled-help-dot inline-flex h-4 w-4 cursor-help items-center justify-center text-[10px] font-semibold transition-colors">
                        ?
                      </span>
                    </template>
                    <div class="space-y-1.5">
                      <p class="font-medium">{{ t('admin.scheduledTests.cronTooltipTitle') }}</p>
                      <p>{{ t('admin.scheduledTests.cronTooltipMeaning') }}</p>
                      <p>{{ t('admin.scheduledTests.cronTooltipExampleEvery30Min') }}</p>
                      <p>{{ t('admin.scheduledTests.cronTooltipExampleHourly') }}</p>
                      <p>{{ t('admin.scheduledTests.cronTooltipExampleDaily') }}</p>
                      <p>{{ t('admin.scheduledTests.cronTooltipExampleWeekly') }}</p>
                      <p>{{ t('admin.scheduledTests.cronTooltipRange') }}</p>
                    </div>
                  </HelpTooltip>
                </label>
                <Input
                  v-model="editForm.cron_expression"
                  :placeholder="'*/30 * * * *'"
                  :hint="t('admin.scheduledTests.cronHelp')"
                />
              </div>
              <div>
                <label class="scheduled-field-label mb-1 flex items-center gap-1 text-xs font-medium">
                  {{ t('admin.scheduledTests.maxResults') }}
                  <HelpTooltip>
                    <template #trigger>
                      <span class="scheduled-help-dot inline-flex h-4 w-4 cursor-help items-center justify-center text-[10px] font-semibold transition-colors">
                        ?
                      </span>
                    </template>
                    <div class="space-y-1.5">
                      <p class="font-medium">{{ t('admin.scheduledTests.maxResultsTooltipTitle') }}</p>
                      <p>{{ t('admin.scheduledTests.maxResultsTooltipMeaning') }}</p>
                      <p>{{ t('admin.scheduledTests.maxResultsTooltipBody') }}</p>
                      <p>{{ t('admin.scheduledTests.maxResultsTooltipExample') }}</p>
                      <p>{{ t('admin.scheduledTests.maxResultsTooltipRange') }}</p>
                    </div>
                  </HelpTooltip>
                </label>
                <Input
                  v-model="editForm.max_results"
                  type="number"
                  placeholder="100"
                />
              </div>
              <div class="flex items-end">
                <label class="scheduled-toggle-label flex items-center gap-2 text-sm">
                  <Toggle v-model="editForm.enabled" />
                  {{ t('admin.scheduledTests.enabled') }}
                </label>
              </div>
              <div class="flex items-end">
                <div>
                  <label class="scheduled-toggle-label flex items-center gap-2 text-sm">
                    <Toggle v-model="editForm.auto_recover" />
                    {{ t('admin.scheduledTests.autoRecover') }}
                  </label>
                  <p class="scheduled-help-text mt-0.5 text-xs">
                    {{ t('admin.scheduledTests.autoRecoverHelp') }}
                  </p>
                </div>
              </div>
            </div>
            <div class="mt-3 flex justify-end gap-2">
              <button
                @click="cancelEdit"
                class="scheduled-button scheduled-button-secondary px-3 py-1.5 text-sm font-medium transition-colors"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                @click="handleEdit"
                :disabled="!editForm.model_id || !editForm.cron_expression || updating"
                class="scheduled-button scheduled-button-primary flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50"
              >
                <Icon v-if="updating" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
                {{ t('common.save') }}
              </button>
            </div>
          </div>

          <!-- Expanded Results Section -->
          <div
            v-if="expandedPlanId === plan.id"
            class="scheduled-results-panel px-4 py-3"
          >
            <div class="scheduled-field-label mb-2 text-xs font-medium">
              {{ t('admin.scheduledTests.results') }}
            </div>

            <!-- Results Loading -->
            <div v-if="loadingResults" class="flex items-center justify-center py-4">
              <Icon name="refresh" size="sm" class="scheduled-muted animate-spin" :stroke-width="2" />
              <span class="scheduled-muted ml-2 text-xs">{{ t('common.loading') }}...</span>
            </div>

            <!-- No Results -->
            <div
              v-else-if="results.length === 0"
              class="scheduled-muted py-4 text-center text-xs"
            >
              {{ t('admin.scheduledTests.noResults') }}
            </div>

            <!-- Results List -->
            <div v-else class="max-h-64 space-y-2 overflow-y-auto">
              <div
                v-for="result in results"
                :key="result.id"
                class="scheduled-result-item p-3"
              >
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2">
                    <!-- Status Badge -->
                    <span
                      :class="[
                        'scheduled-badge inline-flex items-center px-2 py-0.5 text-xs font-medium',
                        resultStatusClass(result.status)
                      ]"
                    >
                      {{
                        result.status === 'success'
                          ? t('admin.scheduledTests.success')
                          : result.status === 'running'
                            ? t('admin.scheduledTests.running')
                            : t('admin.scheduledTests.failed')
                      }}
                    </span>

                    <!-- Latency -->
                    <span v-if="result.latency_ms > 0" class="scheduled-muted text-xs">
                      {{ result.latency_ms }}ms
                    </span>
                  </div>

                  <!-- Started At -->
                  <span class="scheduled-help-text text-xs">
                    {{ formatDateTime(result.started_at) }}
                  </span>
                </div>

                <!-- Response / Error (collapsible) -->
                <div v-if="result.error_message" class="mt-2">
                  <div
                    class="scheduled-detail-toggle scheduled-detail-toggle-danger cursor-pointer text-xs font-medium"
                    @click="toggleResultDetail(result.id)"
                  >
                    {{ t('admin.scheduledTests.errorMessage') }}
                    <Icon
                      name="chevronDown"
                      size="sm"
                      :class="[
                        'inline transition-transform duration-200',
                        expandedResultIds.has(result.id) ? 'rotate-180' : ''
                      ]"
                    />
                  </div>
                  <pre
                    v-if="expandedResultIds.has(result.id)"
                    class="scheduled-log scheduled-log-danger mt-1 max-h-32 overflow-auto whitespace-pre-wrap p-2 text-xs"
                  >{{ result.error_message }}</pre>
                </div>
                <div v-else-if="result.response_text" class="mt-2">
                  <div
                    class="scheduled-detail-toggle cursor-pointer text-xs font-medium"
                    @click="toggleResultDetail(result.id)"
                  >
                    {{ t('admin.scheduledTests.responseText') }}
                    <Icon
                      name="chevronDown"
                      size="sm"
                      :class="[
                        'inline transition-transform duration-200',
                        expandedResultIds.has(result.id) ? 'rotate-180' : ''
                      ]"
                    />
                  </div>
                  <pre
                    v-if="expandedResultIds.has(result.id)"
                    class="scheduled-log mt-1 max-h-32 overflow-auto whitespace-pre-wrap p-2 text-xs"
                  >{{ result.response_text }}</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      :show="showDeleteConfirm"
      :title="t('admin.scheduledTests.deletePlan')"
      :message="t('admin.scheduledTests.confirmDelete')"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteConfirm = false"
    />
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Input from '@/components/common/Input.vue'
import Toggle from '@/components/common/Toggle.vue'
import { Icon } from '@/components/icons'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import type { ScheduledTestPlan, ScheduledTestResult } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

const props = defineProps<{
  show: boolean
  accountId: number | null
  modelOptions: SelectOption[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

// State
const loading = ref(false)
const creating = ref(false)
const loadingResults = ref(false)
const plans = ref<ScheduledTestPlan[]>([])
const results = ref<ScheduledTestResult[]>([])
const expandedPlanId = ref<number | null>(null)
const expandedResultIds = reactive(new Set<number>())
const showAddForm = ref(false)
const showDeleteConfirm = ref(false)
const deletingPlan = ref<ScheduledTestPlan | null>(null)
const editingPlanId = ref<number | null>(null)
const updating = ref(false)
const editForm = reactive({
  model_id: '' as string,
  cron_expression: '' as string,
  max_results: '100' as string,
  enabled: true,
  auto_recover: false
})

const newPlan = reactive({
  model_id: '' as string,
  cron_expression: '' as string,
  max_results: '100' as string,
  enabled: true,
  auto_recover: false
})

const resetNewPlan = () => {
  newPlan.model_id = ''
  newPlan.cron_expression = ''
  newPlan.max_results = '100'
  newPlan.enabled = true
  newPlan.auto_recover = false
}

// Load plans when dialog opens
watch(
  () => props.show,
  async (visible) => {
    if (visible && props.accountId) {
      await loadPlans()
    } else {
      plans.value = []
      results.value = []
      expandedPlanId.value = null
      expandedResultIds.clear()
      showAddForm.value = false
      showDeleteConfirm.value = false
    }
  }
)

const loadPlans = async () => {
  if (!props.accountId) return
  loading.value = true
  try {
    plans.value = await adminAPI.scheduledTests.listByAccount(props.accountId)
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to load plans')
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  if (!props.accountId || !newPlan.model_id || !newPlan.cron_expression) return
  creating.value = true
  try {
    const maxResults = Number(newPlan.max_results) || 100
    await adminAPI.scheduledTests.create({
      account_id: props.accountId,
      model_id: newPlan.model_id,
      cron_expression: newPlan.cron_expression,
      enabled: newPlan.enabled,
      max_results: maxResults,
      auto_recover: newPlan.auto_recover
    })
    appStore.showSuccess(t('admin.scheduledTests.createSuccess'))
    showAddForm.value = false
    resetNewPlan()
    await loadPlans()
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to create plan')
  } finally {
    creating.value = false
  }
}

const handleToggleEnabled = async (plan: ScheduledTestPlan, enabled: boolean) => {
  try {
    const updated = await adminAPI.scheduledTests.update(plan.id, { enabled })
    const index = plans.value.findIndex((p) => p.id === plan.id)
    if (index !== -1) {
      plans.value[index] = updated
    }
    appStore.showSuccess(t('admin.scheduledTests.updateSuccess'))
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to update plan')
  }
}

const startEdit = (plan: ScheduledTestPlan) => {
  editingPlanId.value = plan.id
  editForm.model_id = plan.model_id
  editForm.cron_expression = plan.cron_expression
  editForm.max_results = String(plan.max_results)
  editForm.enabled = plan.enabled
  editForm.auto_recover = plan.auto_recover
}

const cancelEdit = () => {
  editingPlanId.value = null
}

const handleEdit = async () => {
  if (!editingPlanId.value || !editForm.model_id || !editForm.cron_expression) return
  updating.value = true
  try {
    const updated = await adminAPI.scheduledTests.update(editingPlanId.value, {
      model_id: editForm.model_id,
      cron_expression: editForm.cron_expression,
      max_results: Number(editForm.max_results) || 100,
      enabled: editForm.enabled,
      auto_recover: editForm.auto_recover
    })
    const index = plans.value.findIndex((p) => p.id === editingPlanId.value)
    if (index !== -1) {
      plans.value[index] = updated
    }
    appStore.showSuccess(t('admin.scheduledTests.updateSuccess'))
    editingPlanId.value = null
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to update plan')
  } finally {
    updating.value = false
  }
}

const confirmDeletePlan = (plan: ScheduledTestPlan) => {
  deletingPlan.value = plan
  showDeleteConfirm.value = true
}

const handleDelete = async () => {
  if (!deletingPlan.value) return
  try {
    await adminAPI.scheduledTests.delete(deletingPlan.value.id)
    appStore.showSuccess(t('admin.scheduledTests.deleteSuccess'))
    plans.value = plans.value.filter((p) => p.id !== deletingPlan.value!.id)
    if (expandedPlanId.value === deletingPlan.value.id) {
      expandedPlanId.value = null
      results.value = []
    }
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to delete plan')
  } finally {
    showDeleteConfirm.value = false
    deletingPlan.value = null
  }
}

const toggleExpand = async (planId: number) => {
  if (expandedPlanId.value === planId) {
    expandedPlanId.value = null
    results.value = []
    expandedResultIds.clear()
    return
  }

  expandedPlanId.value = planId
  expandedResultIds.clear()
  loadingResults.value = true
  try {
    results.value = await adminAPI.scheduledTests.listResults(planId, 20)
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to load results')
    results.value = []
  } finally {
    loadingResults.value = false
  }
}

const toggleResultDetail = (resultId: number) => {
  if (expandedResultIds.has(resultId)) {
    expandedResultIds.delete(resultId)
  } else {
    expandedResultIds.add(resultId)
  }
}

const resultStatusClass = (status: ScheduledTestResult['status']) => {
  if (status === 'success') return 'scheduled-badge-success'
  if (status === 'running') return 'scheduled-badge-info'
  return 'scheduled-badge-danger'
}
</script>

<style scoped>
.scheduled-muted,
.scheduled-help-text {
  color: var(--nm-ink-faint);
}

.scheduled-section-title,
.scheduled-plan-title {
  color: var(--nm-ink);
}

.scheduled-field-label,
.scheduled-toggle-label,
.scheduled-detail-toggle {
  color: var(--nm-ink-muted);
}

.scheduled-form-panel,
.scheduled-plan,
.scheduled-result-item {
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius);
  background: var(--nm-surface);
}

.scheduled-form-panel,
.scheduled-edit-panel {
  background: var(--nm-surface-soft);
}

.scheduled-empty {
  border: 1px dashed var(--nm-border);
  border-radius: var(--nm-radius);
  background: var(--nm-surface);
}

.scheduled-plan:hover {
  border-color: var(--nm-ink-muted);
}

.scheduled-edit-panel,
.scheduled-results-panel {
  border-top: 1px solid var(--nm-border-light);
}

.scheduled-help-dot,
.scheduled-icon-button,
.scheduled-button,
.scheduled-badge {
  border-radius: var(--nm-radius-sm);
}

.scheduled-help-dot {
  border: 1px solid var(--nm-border);
  background: var(--nm-surface);
  color: var(--nm-ink-faint);
}

.scheduled-help-dot:hover,
.scheduled-icon-button:hover {
  background: var(--nm-accent-soft);
  color: var(--nm-accent-text);
}

.scheduled-icon-button {
  display: inline-flex;
  min-height: 2rem;
  min-width: 2rem;
  align-items: center;
  justify-content: center;
  color: var(--nm-ink-faint);
  cursor: pointer;
}

.scheduled-button {
  cursor: pointer;
  line-height: 1.25rem;
}

.scheduled-icon-button-danger:hover {
  background: var(--nm-danger-soft);
  color: var(--nm-danger-text);
}

.scheduled-button-primary {
  border: 1px solid var(--nm-accent);
  background: var(--nm-accent);
  color: var(--nm-on-accent);
}

.scheduled-button-primary:hover:not(:disabled) {
  border-color: var(--nm-accent-strong);
  background: var(--nm-accent-strong);
}

.scheduled-button:disabled {
  cursor: not-allowed;
}

.scheduled-button-secondary {
  border: 1px solid var(--nm-border);
  background: var(--nm-surface);
  color: var(--nm-ink-muted);
}

.scheduled-button-secondary:hover {
  border-color: var(--nm-ink-muted);
  color: var(--nm-ink);
}

.scheduled-badge-success {
  background: var(--nm-success-soft);
  color: var(--nm-success-text);
}

.scheduled-badge-info {
  background: var(--nm-info-soft);
  color: var(--nm-info-text);
}

.scheduled-badge-danger {
  background: var(--nm-danger-soft);
  color: var(--nm-danger-text);
}

.scheduled-detail-toggle:hover {
  color: var(--nm-accent-text);
}

.scheduled-detail-toggle-danger {
  color: var(--nm-danger-text);
}

.scheduled-detail-toggle-danger:hover {
  color: var(--nm-danger);
}

.scheduled-log {
  border: 1px solid var(--nm-border-light);
  border-radius: var(--nm-radius);
  background: var(--nm-surface-soft);
  color: var(--nm-ink-muted);
}

.scheduled-log-danger {
  border-color: var(--nm-danger);
  background: var(--nm-danger-soft);
  color: var(--nm-danger-text);
}
</style>
