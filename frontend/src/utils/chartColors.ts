import { ref } from 'vue'
import type { Ref } from 'vue'

/**
 * Single source of truth for chart colors (Chart.js / ECharts canvases).
 *
 * Hex values mirror the design tokens in `src/style.css`:
 * - `--chart-1..5` — the indigo-led semantic chart slots. `--chart-1` is the
 *   indigo accent and uses the canonical `--nm-accent` hex verbatim
 *   (`5b58ff` light / `7b78ff` dark; the HSL triplet is the same color with
 *   rounding slack). `--chart-2..5` are computed from their HSL triplets.
 * - `--nm-*` — QW-console tokens for grid lines, axis text and tooltip
 *   surfaces, so canvases stay coherent with the surrounding cards in both
 *   themes.
 */

export interface ChartTheme {
  /** Indigo accent — `--chart-1` / `--nm-accent`. */
  primary: string
  /** Sky blue — `--chart-2`. */
  blue: string
  /** Green — `--chart-3`. */
  green: string
  /** Amber — `--chart-4`. */
  amber: string
  /** Pink — `--chart-5`. */
  pink: string
  /** Semantic danger red for error series — `--nm-danger`. */
  danger: string
  /** Neutral gray for "other"/residual series — `--nm-ink-faint`. */
  muted: string
  /** Grid / axis lines — `--nm-border`. */
  grid: string
  /** Axis tick & legend text — `--nm-ink-muted` (equals `--muted-foreground`). */
  text: string
  /** Chart surface — `--nm-surface`; used as tooltip background. */
  surface: string
  /** Tooltip title text — `--nm-ink`. */
  tooltipTitle: string
  /** Tooltip body text — `--nm-ink-muted`. */
  tooltipBody: string
  /** Rotating series palette for multi-series charts (indigo-led family). */
  series: string[]
}

const LIGHT_THEME: ChartTheme = {
  primary: '#5b58ff', // --chart-1: 241 100% 67% / --nm-accent
  blue: '#0da2e7', // --chart-2: 199 89% 48%
  green: '#2aac7d', // --chart-3: 158 61% 42%
  amber: '#fbbd23', // --chart-4: 43 96% 56%
  pink: '#ed457d', // --chart-5: 340 82% 60%
  danger: '#d93025', // --nm-danger
  muted: '#86909c', // --nm-ink-faint
  grid: '#e6e9ef', // --nm-border
  text: '#4e5969', // --nm-ink-muted / --muted-foreground
  surface: '#ffffff', // --nm-surface
  tooltipTitle: '#0b0c0f', // --nm-ink
  tooltipBody: '#4e5969', // --nm-ink-muted
  series: [
    '#5b58ff', // chart-1 indigo
    '#0da2e7', // chart-2 blue
    '#2aac7d', // chart-3 green
    '#fbbd23', // chart-4 amber
    '#ed457d', // chart-5 pink
    '#4643e8', // --nm-accent-strong
    '#0b6bcb', // --nm-info
    '#15803d', // --nm-success
    '#b45309', // --nm-warning
    '#86909c' // --nm-ink-faint
  ]
}

const DARK_THEME: ChartTheme = {
  primary: '#7b78ff', // --chart-1: 241 100% 74% / --nm-accent (dark)
  blue: '#3ebaf4', // --chart-2: 199 89% 60%
  green: '#46d29f', // --chart-3: 158 61% 55%
  amber: '#fbc94b', // --chart-4: 43 96% 64%
  pink: '#f1749e', // --chart-5: 340 82% 70%
  danger: '#f0655f', // --nm-danger (dark)
  muted: '#7a8290', // --nm-ink-faint (dark)
  grid: '#2a2e37', // --nm-border (dark)
  text: '#b4bcc6', // --nm-ink-muted / --muted-foreground (dark)
  surface: '#15171c', // --nm-surface (dark)
  tooltipTitle: '#f2f4f8', // --nm-ink (dark)
  tooltipBody: '#b4bcc6', // --nm-ink-muted (dark)
  series: [
    '#7b78ff', // chart-1 indigo
    '#3ebaf4', // chart-2 blue
    '#46d29f', // chart-3 green
    '#fbc94b', // chart-4 amber
    '#f1749e', // chart-5 pink
    '#918fff', // --nm-accent-strong (dark)
    '#6fb3f0', // --nm-info (dark)
    '#46c26a', // --nm-success (dark)
    '#e8a33d', // --nm-warning (dark)
    '#7a8290' // --nm-ink-faint (dark)
  ]
}

export const CHART_COLORS: Record<'light' | 'dark', ChartTheme> = {
  light: LIGHT_THEME,
  dark: DARK_THEME
}

/** Theme-aware palette lookup for chart rendering code. */
export function getChartTheme(isDark: boolean): ChartTheme {
  return isDark ? CHART_COLORS.dark : CHART_COLORS.light
}

/** Append a hex alpha channel to a 6-digit hex color (e.g. 0.125 -> "20"). */
export function withAlpha(hex: string, alpha: number): string {
  const clamped = Math.max(0, Math.min(1, alpha))
  const channel = Math.round(clamped * 255)
    .toString(16)
    .padStart(2, '0')
  return `${hex}${channel}`
}

const isDarkMode = ref(false)
let themeWatcherStarted = false

function startThemeWatcher(): void {
  if (themeWatcherStarted || typeof document === 'undefined') return
  themeWatcherStarted = true
  isDarkMode.value = document.documentElement.classList.contains('dark')
  if (typeof MutationObserver === 'undefined') return
  // The `dark` class on <html> is toggled by the theme switch (AppSidebar /
  // main.ts). Watching it keeps every chart palette in sync with the theme.
  new MutationObserver(() => {
    isDarkMode.value = document.documentElement.classList.contains('dark')
  }).observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
}

/**
 * Shared reactive dark-mode flag for chart palettes. Returns a module-level
 * ref that tracks the `dark` class on `<html>`, so palettes computed from it
 * switch live when the theme changes.
 */
export function useChartDarkMode(): Ref<boolean> {
  startThemeWatcher()
  return isDarkMode
}
