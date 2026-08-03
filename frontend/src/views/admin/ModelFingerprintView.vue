<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- 页头：标题 + §7.4 判定原理一句话版 -->
      <div class="card">
        <h1 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('admin.fingerprint.title') }}
        </h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.fingerprint.description') }}
        </p>
        <p class="mt-3 rounded-lg bg-gray-50 p-3 text-xs leading-5 text-gray-500 dark:bg-dark-800 dark:text-gray-400">
          {{ t('admin.fingerprint.help') }}
        </p>
      </div>

      <!-- 区块一：发起检测（测账号 / 测外部端点） -->
      <div class="card">
        <h2 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">
          {{ t('admin.fingerprint.create.sectionTitle') }}
        </h2>
        <FingerprintAuditForm
          :accounts="accounts"
          :references="references"
          @created="handleAuditCreated"
        />
      </div>

      <!-- 区块二：参考基准管理 -->
      <div class="card">
        <h2 class="mb-1 text-base font-semibold text-gray-900 dark:text-white">
          {{ t('admin.fingerprint.references.sectionTitle') }}
        </h2>
        <p class="mb-4 text-xs text-gray-400">
          {{ t('admin.fingerprint.references.sectionDesc') }}
        </p>
        <FingerprintReferencePanel
          :references="references"
          :accounts="accounts"
          :loading="referencesLoading"
          @changed="loadReferences"
        />
      </div>

      <!-- 区块三：检测记录 -->
      <div class="card">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">
            {{ t('admin.fingerprint.records.sectionTitle') }}
          </h2>
          <button type="button" class="btn btn-secondary px-2.5 py-1 text-xs" @click="loadAudits">
            {{ t('common.refresh') }}
          </button>
        </div>
        <FingerprintAuditTable
          :audits="audits"
          :accounts="accounts"
          :loading="auditsLoading"
          @select="openReport"
          @remove="askDeleteAudit"
        />
      </div>
    </div>

    <FingerprintReportDialog
      :show="showReport"
      :audit-id="selectedAuditId"
      :accounts="accounts"
      @close="showReport = false"
    />

    <ConfirmDialog
      :show="showDeleteConfirm"
      :title="t('admin.fingerprint.records.deleteConfirmTitle')"
      :message="t('admin.fingerprint.records.deleteConfirmMessage', { id: deleteAuditId ?? '' })"
      danger
      @confirm="confirmDeleteAudit"
      @cancel="showDeleteConfirm = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import type { FingerprintAuditSummary, FingerprintReference } from '@/api/admin/fingerprint'
import type { Account } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import FingerprintAuditForm from '@/components/admin/fingerprint/FingerprintAuditForm.vue'
import FingerprintReferencePanel from '@/components/admin/fingerprint/FingerprintReferencePanel.vue'
import FingerprintAuditTable from '@/components/admin/fingerprint/FingerprintAuditTable.vue'
import FingerprintReportDialog from '@/components/admin/fingerprint/FingerprintReportDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

const { t } = useI18n()
const appStore = useAppStore()

const accounts = ref<Account[]>([])
const references = ref<FingerprintReference[]>([])
const audits = ref<FingerprintAuditSummary[]>([])
const referencesLoading = ref(false)
const auditsLoading = ref(false)

const showReport = ref(false)
const selectedAuditId = ref<string | null>(null)

let pollTimer: ReturnType<typeof setTimeout> | null = null

function stopPolling() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

/** 轮询进行中的任务：2.5s 一次，没有 running 任务即停止 */
function schedulePoll() {
  stopPolling()
  if (audits.value.some((audit) => audit.status === 'running')) {
    pollTimer = setTimeout(loadAudits, 2500)
  }
}

async function loadAccounts() {
  try {
    const res = await adminAPI.accounts.list(1, 200, { lite: '1' })
    accounts.value = res.items || []
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.fingerprint.create.accountsLoadFailed')))
  }
}

async function loadReferences() {
  referencesLoading.value = true
  try {
    references.value = await adminAPI.fingerprint.listReferences()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.fingerprint.references.loadFailed')))
  } finally {
    referencesLoading.value = false
  }
}

async function loadAudits() {
  auditsLoading.value = true
  try {
    audits.value = await adminAPI.fingerprint.listAudits()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.fingerprint.records.loadFailed')))
  } finally {
    auditsLoading.value = false
    schedulePoll()
  }
}

function handleAuditCreated() {
  void loadAudits()
  // 「现场注册参考」发起的检测会顺带产生新参考，完成后一并刷新
  void loadReferences()
}

function openReport(id: string) {
  selectedAuditId.value = id
  showReport.value = true
}

// 删除检测记录：ConfirmDialog 确认后调用 DELETE /audits/:id 并刷新列表
const showDeleteConfirm = ref(false)
const deleteAuditId = ref<string | null>(null)

function askDeleteAudit(id: string) {
  deleteAuditId.value = id
  showDeleteConfirm.value = true
}

async function confirmDeleteAudit() {
  showDeleteConfirm.value = false
  const id = deleteAuditId.value
  if (!id) return
  try {
    await adminAPI.fingerprint.deleteAudit(id)
    appStore.showSuccess(t('admin.fingerprint.records.deleted'))
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.fingerprint.records.deleteFailed')))
  } finally {
    deleteAuditId.value = null
    void loadAudits()
  }
}

onMounted(() => {
  void loadAccounts()
  void loadReferences()
  void loadAudits()
})

onUnmounted(stopPolling)
</script>
