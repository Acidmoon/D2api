import { describe, expect, it } from 'vitest'
import type { PromptAuditConfig } from '../types'
import {
  buildUpdateRequest,
  configToDraft,
  draftFingerprint,
  emptyEventFilters,
  eventFilterPayload,
  formatUserGuardWhitelist,
  hasExplicitDeleteRange,
  parseUserGuardWhitelist,
  SCANNER_CATALOG,
} from '../viewModel'

const config = (): PromptAuditConfig => ({
  enabled: true,
  blocking_enabled: false,
  store_pass_events: false,
  effective_mode: 'async_audit',
  strategy: 'priority',
  worker_count: 4,
  queue_capacity: 100,
  scanners: SCANNER_CATALOG.map((item) => item.id),
  all_groups: true,
  group_ids: [],
  endpoints: [{
    id: 'guard-1', name: 'Guard One', protocol: 'openai_compatible', base_url: 'http://127.0.0.1:8000',
    model: 'sileader/qwen3guard:0.6b', timeout_ms: 3000, input_limit: 4000, enabled: true,
    has_token: true, token_status: 'configured',
  }],
  user_guard: { enabled: false, threshold: 3, window_minutes: 10, ban_duration_minutes: 60, whitelist_user_ids: [] },
  config_version: 7,
  updated_at: '2026-07-16T00:00:00Z',
  updated_by: 1,
  change_summary: '{}',
})

describe('Prompt Audit view model', () => {
  it('normalizes legacy null collections from the public config', () => {
    const legacy = { ...config(), group_ids: null, scanners: null, endpoints: null } as unknown as PromptAuditConfig
    expect(configToDraft(legacy)).toMatchObject({ group_ids: [], scanners: [], endpoints: [] })
  })

  it('models all nine official input scanners', () => {
    expect(SCANNER_CATALOG).toHaveLength(9)
    expect(SCANNER_CATALOG.map((item) => item.id)).toContain('suicide_and_self_harm')
  })

  it('keeps, replaces, or explicitly clears a saved token without copying plaintext from the server', () => {
    const draft = configToDraft(config())
    expect(draft.endpoints[0].token).toBe('')
    expect(buildUpdateRequest(draft).endpoints[0]).toMatchObject({ token: undefined, clear_token: false })

    draft.endpoints[0].token = 'temporary-canary-token'
    expect(buildUpdateRequest(draft).endpoints[0]).toMatchObject({ token: 'temporary-canary-token', clear_token: false })

    draft.endpoints[0].token = ''
    draft.endpoints[0].clear_token = true
    expect(buildUpdateRequest(draft).endpoints[0]).toMatchObject({ token: undefined, clear_token: true })
  })

  it('sends trimmed endpoint system prompt and omits it when blank', () => {
    const draft = configToDraft(config())
    expect(buildUpdateRequest(draft).endpoints[0].system_prompt).toBeUndefined()

    draft.endpoints[0].system_prompt = '  你是内容安全审核员  '
    expect(buildUpdateRequest(draft).endpoints[0].system_prompt).toBe('你是内容安全审核员')

    draft.endpoints[0].system_prompt = '   '
    expect(buildUpdateRequest(draft).endpoints[0].system_prompt).toBeUndefined()
  })

  it('tracks dirty state from the full normalized save payload', () => {
    const original = configToDraft(config())
    const changed = configToDraft(config())
    expect(draftFingerprint(changed)).toBe(draftFingerprint(original))
    changed.queue_capacity += 1
    expect(draftFingerprint(changed)).not.toBe(draftFingerprint(original))
  })

  it('fills account guard defaults for legacy configs and round-trips the section', () => {
    const legacy = { ...config() } as unknown as PromptAuditConfig
    delete (legacy as { user_guard?: unknown }).user_guard
    const draft = configToDraft(legacy)
    expect(draft.user_guard).toEqual({ enabled: false, threshold: 3, window_minutes: 10, ban_duration_minutes: 60, whitelist_user_ids: [] })

    draft.user_guard = { enabled: true, threshold: 5, window_minutes: 30, ban_duration_minutes: 120, whitelist_user_ids: [9, 3] }
    expect(buildUpdateRequest(draft).user_guard).toEqual({ enabled: true, threshold: 5, window_minutes: 30, ban_duration_minutes: 120, whitelist_user_ids: [3, 9] })
  })

  it('parses whitelist text with dedupe, sorting, and invalid token reporting', () => {
    expect(parseUserGuardWhitelist('3\n9, 5\n3\nabc\n-1,0，7；')).toEqual({ ids: [3, 5, 7, 9], invalid: ['abc', '-1', '0'] })
    expect(parseUserGuardWhitelist('')).toEqual({ ids: [], invalid: [] })
    expect(formatUserGuardWhitelist([3, 5])).toBe('3\n5')
  })

  it('requires a valid explicit range and sends canonical ISO timestamps for filter deletion', () => {
    const filters = emptyEventFilters()
    expect(hasExplicitDeleteRange(filters)).toBe(false)
    filters.start_at = '2026-07-15T10:00'
    filters.end_at = '2026-07-16T10:00'
    filters.group_id = '9'
    expect(hasExplicitDeleteRange(filters)).toBe(true)
    expect(eventFilterPayload(filters)).toMatchObject({
      group_id: 9,
      start_at: new Date(filters.start_at).toISOString(),
      end_at: new Date(filters.end_at).toISOString(),
    })
  })
})
