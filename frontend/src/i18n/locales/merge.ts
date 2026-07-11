export type LocaleMessages = Record<string, unknown>

function isMessageObject(value: unknown): value is LocaleMessages {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/**
 * Merge project-specific locale overrides without replacing complete upstream
 * sections such as `admin`, `common`, or `dashboard`.
 */
export function mergeLocaleMessages(
  base: LocaleMessages,
  overrides: LocaleMessages
): LocaleMessages {
  const merged: LocaleMessages = { ...base }

  for (const [key, overrideValue] of Object.entries(overrides)) {
    const baseValue = merged[key]
    merged[key] =
      isMessageObject(baseValue) && isMessageObject(overrideValue)
        ? mergeLocaleMessages(baseValue, overrideValue)
        : overrideValue
  }

  return merged
}
