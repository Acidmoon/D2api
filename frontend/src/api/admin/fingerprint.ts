/**
 * Admin Model Fingerprint API endpoints
 * 模型指纹检测（arXiv:2607.10252 的工程化，见 docs/MODEL_FINGERPRINT_AUDIT_CN.md）：
 * 发起检测、轮询任务状态、管理参考基准。全部接口仅 admin。
 * 外部端点的 api_key 只在任务运行期由后端持有，不落盘、不回显。
 */

import { apiClient } from '../client'

export type FingerprintProvider = 'openai' | 'anthropic' | 'gemini' | 'grok'
export type FingerprintAPIMode = 'chat_completions' | 'responses'
export type FingerprintTargetType = 'account' | 'external'
export type FingerprintTaskKind = 'audit' | 'register_reference'
export type FingerprintStatus = 'running' | 'done' | 'failed'
export type FingerprintVerdict = 'consistent' | 'warning' | 'anomalous' | 'insufficient' | ''
export type FingerprintBand = 'consistent' | 'mostly_consistent' | 'warning' | 'anomalous' | ''
export type FingerprintFlag = 'response_caching' | 'hidden_reasoning' | 'not_applicable' | string
export type FingerprintReferenceSource = 'account_sampled' | 'zenodo_import'

export interface FingerprintProgress {
  done: number
  total: number
}

export interface FingerprintTarget {
  type: FingerprintTargetType
  account_id?: number
  base_url?: string
  provider?: string
  model: string
}

/** 异步任务状态（进行中轮询返回；POST 发起时同样返回此结构） */
export interface FingerprintTaskStatus {
  task_id: string
  kind: FingerprintTaskKind
  status: FingerprintStatus
  progress: FingerprintProgress
  model: string
  reference_model?: string
  error?: string
  created_at: string
  finished_at?: string
}

/** 检测记录列表的摘要项（进行中的排在最前） */
export interface FingerprintAuditSummary {
  id: string
  target: FingerprintTarget
  reference_model: string
  status: FingerprintStatus
  progress: FingerprintProgress
  score: number | null
  verdict: FingerprintVerdict
  flags: FingerprintFlag[]
  error?: string
  created_at: string
  duration_ms: number
}

export interface FingerprintReportReference {
  model: string
  source: FingerprintReferenceSource
  enrolled_at: string
}

export interface FingerprintReportCell {
  task: string
  language: string
  /** 未进入 JSD 计算的 cell 为 null */
  jsd: number | null
  valid: number
  invalid: number
  refusal: number
  empty: number
  /** hidden_reasoning / response_caching / not_applicable / insufficient_samples */
  excluded?: string
  top_answers: Record<string, number>
  reference_top_answers?: Record<string, number>
  t0_answers?: string[]
  /** 仅 keep_raw=true 时存在 */
  samples?: string[]
}

/** 完整检测报告（任务完成/失败后 GET /audits/:id 返回） */
export interface FingerprintReport {
  id: string
  target: FingerprintTarget
  reference: FingerprintReportReference
  status: FingerprintStatus
  progress: FingerprintProgress
  /** 证据不足时为 null */
  score: number | null
  verdict: FingerprintVerdict
  band?: FingerprintBand
  /** 进入 JSD 的有效 cell 数 k */
  cell_count: number
  /** 有效 cell 的平均有效样本数 n */
  avg_samples: number
  split_half_jsd: number | null
  /** T=0 答案与参考不一致的 cell 数 */
  t0_mismatch_cells: number
  flags: FingerprintFlag[]
  error?: string
  /** 电池执行中最近一次探测失败的摘要（部分失败时帮助定位原因） */
  last_error?: string
  created_at: string
  duration_ms: number
  cells: FingerprintReportCell[]
}

export interface FingerprintReferenceCell {
  samples: number
  valid: number
  distribution: Record<string, number>
  t0_answers?: string[]
}

export interface FingerprintReference {
  model: string
  source: FingerprintReferenceSource
  source_account_id?: number
  enrolled_at: string
  /** 键为 "<task>|<language>" */
  cells: Record<string, FingerprintReferenceCell>
}

export interface CreateAuditParams {
  target_type: FingerprintTargetType
  account_id?: number
  base_url?: string
  api_key?: string
  provider?: FingerprintProvider
  api_mode?: FingerprintAPIMode
  model: string
  reference_model: string
  /** 提供时先对该账号现场采样注册参考，再测目标 */
  reference_account_id?: number
  keep_raw?: boolean
}

export interface RegisterReferenceParams {
  account_id: number
  model: string
}

/**
 * 发起一次指纹检测（异步），返回任务状态，前端轮询 getAudit。
 */
export async function createAudit(params: CreateAuditParams): Promise<FingerprintTaskStatus> {
  const { data } = await apiClient.post<FingerprintTaskStatus>('/admin/fingerprint/audits', params)
  return data
}

/**
 * 检测记录列表（按时间倒序，进行中的在最前）。
 */
export async function listAudits(options?: {
  signal?: AbortSignal
}): Promise<FingerprintAuditSummary[]> {
  const { data } = await apiClient.get<FingerprintAuditSummary[]>('/admin/fingerprint/audits', {
    signal: options?.signal,
  })
  return data
}

/**
 * 查询检测任务：进行中返回任务状态，完成/失败返回完整报告。
 */
export async function getAudit(
  id: string,
  options?: { signal?: AbortSignal }
): Promise<FingerprintTaskStatus | FingerprintReport> {
  const { data } = await apiClient.get<FingerprintTaskStatus | FingerprintReport>(
    `/admin/fingerprint/audits/${encodeURIComponent(id)}`,
    { signal: options?.signal }
  )
  return data
}

/** getAudit 返回值的类型守卫：报告有 cells/verdict，任务状态只有 task_id/kind。 */
export function isFingerprintReport(
  detail: FingerprintTaskStatus | FingerprintReport
): detail is FingerprintReport {
  return (detail as FingerprintReport).cells !== undefined
}

/**
 * 对可信账号现场采样注册参考指纹（异步任务，同样用 getAudit 轮询）。
 */
export async function registerReference(
  params: RegisterReferenceParams
): Promise<FingerprintTaskStatus> {
  const { data } = await apiClient.post<FingerprintTaskStatus>(
    '/admin/fingerprint/references',
    params
  )
  return data
}

/**
 * 参考指纹列表（按注册时间倒序）。
 */
export async function listReferences(options?: {
  signal?: AbortSignal
}): Promise<FingerprintReference[]> {
  const { data } = await apiClient.get<FingerprintReference[]>('/admin/fingerprint/references', {
    signal: options?.signal,
  })
  return data
}

export const fingerprintAPI = {
  createAudit,
  listAudits,
  getAudit,
  isFingerprintReport,
  registerReference,
  listReferences,
}

export default fingerprintAPI
