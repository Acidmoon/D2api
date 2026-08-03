/**
 * 模型指纹检测视图的共享格式化助手：
 * 判定徽章/状态样式、分数与耗时的显示、探测项与语言的文案映射。
 * 判定口径（分档阈值、解释文案）遵循 docs/MODEL_FINGERPRINT_AUDIT_CN.md §7.4。
 */

import { useI18n } from 'vue-i18n'
import type {
  FingerprintBand,
  FingerprintReferenceSource,
  FingerprintStatus,
  FingerprintVerdict,
} from '@/api/admin/fingerprint'

/** 参考基准超过 60 天未重注册即提示（设计文档 §10：建议 1–2 个月重注册） */
export const REFERENCE_STALE_MS = 60 * 24 * 60 * 60 * 1000

/** 细分档（band）映射到 verdict 文案 key；consistent/mostly_consistent 同属「一致」档 */
export type VerdictTextKey = 'consistent' | 'mostlyConsistent' | 'warning' | 'anomalous' | 'insufficient'

export function verdictTextKey(verdict: FingerprintVerdict, band?: FingerprintBand): VerdictTextKey | null {
  if (verdict === 'insufficient') return 'insufficient'
  if (verdict === 'warning') return 'warning'
  if (verdict === 'anomalous') return 'anomalous'
  if (verdict === 'consistent') {
    return band === 'mostly_consistent' ? 'mostlyConsistent' : 'consistent'
  }
  return null
}

export function useFingerprintFormat() {
  const { t, te } = useI18n()

  function verdictLabel(verdict: FingerprintVerdict, band?: FingerprintBand): string {
    const key = verdictTextKey(verdict, band)
    return key ? t(`admin.fingerprint.verdict.${key}`) : t('admin.fingerprint.detail.noData')
  }

  function verdictBadgeText(verdict: FingerprintVerdict, band?: FingerprintBand): string {
    const key = verdictTextKey(verdict, band)
    return key ? t(`admin.fingerprint.verdict.badge.${key}`) : ''
  }

  function verdictExplain(verdict: FingerprintVerdict, band?: FingerprintBand, k?: number, total?: number): string {
    const key = verdictTextKey(verdict, band)
    if (!key) return ''
    if (key === 'insufficient') {
      return t('admin.fingerprint.verdict.explain.insufficient', { k: k ?? 0, total: total ?? 16 })
    }
    return t(`admin.fingerprint.verdict.explain.${key}`)
  }

  function verdictBadgeClass(verdict: FingerprintVerdict): string {
    switch (verdict) {
      case 'consistent':
        return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
      case 'warning':
        return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
      case 'anomalous':
        return 'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300'
      default:
        // insufficient 与未知判定都用灰色，绝不与「一致」混淆（§7.4 禁止表述）
        return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
    }
  }

  function statusBadgeClass(status: FingerprintStatus): string {
    switch (status) {
      case 'running':
        return 'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300'
      case 'failed':
        return 'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300'
      default:
        return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
    }
  }

  function formatScore(score: number | null | undefined): string {
    if (score === null || score === undefined) return t('admin.fingerprint.detail.noData')
    return score.toFixed(3)
  }

  function formatDuration(ms: number | null | undefined): string {
    if (!ms || ms <= 0) return t('admin.fingerprint.detail.noData')
    const seconds = Math.round(ms / 1000)
    if (seconds < 60) return `${seconds}s`
    const minutes = Math.floor(seconds / 60)
    return `${minutes}m ${seconds % 60}s`
  }

  function taskLabel(task: string): string {
    const key = `admin.fingerprint.detail.tasks.${task}`
    return te(key) ? t(key) : task
  }

  function languageLabel(language: string): string {
    const key = `admin.fingerprint.detail.language.${language}`
    return te(key) ? t(key) : language
  }

  function sourceLabel(source: FingerprintReferenceSource | string): string {
    return source === 'zenodo_import'
      ? t('admin.fingerprint.references.sourceZenodo')
      : t('admin.fingerprint.references.sourceAccountSampled')
  }

  function isReferenceStale(enrolledAt: string): boolean {
    const time = new Date(enrolledAt).getTime()
    if (!Number.isFinite(time)) return false
    return Date.now() - time > REFERENCE_STALE_MS
  }

  /** 分布 map → "答案×次数" 的紧凑文本，按次数降序取前 5 个 */
  function formatDistribution(dist: Record<string, number> | null | undefined, limit = 5): string {
    if (!dist) return t('admin.fingerprint.detail.noData')
    const entries = Object.entries(dist).sort((a, b) => b[1] - a[1])
    if (entries.length === 0) return t('admin.fingerprint.detail.noData')
    return entries
      .slice(0, limit)
      .map(([answer, count]) => `${answer}×${count}`)
      .join('，')
  }

  return {
    verdictLabel,
    verdictBadgeText,
    verdictExplain,
    verdictBadgeClass,
    statusBadgeClass,
    formatScore,
    formatDuration,
    taskLabel,
    languageLabel,
    sourceLabel,
    isReferenceStale,
    formatDistribution,
  }
}
