<template>
  <template v-for="item in items" :key="item.path">
    <!-- Hairline divider above groups that opt in (expanded sidebar only).
         Rendered for the first group too (QW: 首页/API Keys | 模型…). -->
    <div
      v-if="item.dividerBefore && !collapsed"
      class="mx-2 my-2 h-px flex-shrink-0 bg-border/70"
    ></div>
    <!-- Collapsible group (has children) -->
    <template v-if="item.children?.length">
      <button
        type="button"
        class="sidebar-link mb-1 w-full"
        :class="{
          'sidebar-link-active': isGroupActive(item),
          'sidebar-link-collapsed': collapsed
        }"
        :title="collapsed ? item.label : undefined"
        :aria-expanded="isGroupExpanded(item)"
        @click="handleGroupClick(item)"
      >
        <component :is="item.icon" class="h-5 w-5 flex-shrink-0" />
        <span
          class="sidebar-label sidebar-label-flex"
          :class="{ 'sidebar-label-collapsed': collapsed }"
          :aria-hidden="collapsed ? 'true' : 'false'"
        >
          <span class="min-w-0 truncate">{{ item.label }}</span>
          <ChevronDownIcon
            class="h-4 w-4 flex-shrink-0 transition-transform duration-200"
            :class="isGroupExpanded(item) ? 'rotate-180' : ''"
          />
        </span>
      </button>
      <!-- Children -->
      <div v-if="!collapsed && isGroupExpanded(item)" class="mb-1 ml-4 border-l border-border pl-2">
        <router-link
          v-for="child in item.children"
          :key="child.path"
          :to="child.path"
          class="sidebar-link mb-0.5 py-1.5 text-sm"
          :class="{ 'sidebar-link-active': route.path === child.path }"
          @click="emit('menu-click', child.path)"
        >
          <component :is="child.icon" class="h-4 w-4 flex-shrink-0" />
          <span>{{ child.label }}</span>
        </router-link>
      </div>
    </template>
    <!-- Normal item (no children) -->
    <router-link
      v-else
      :to="item.path"
      class="sidebar-link mb-1"
      :class="{ 'sidebar-link-active': isActive(item.path), 'sidebar-link-collapsed': collapsed }"
      :title="collapsed ? item.label : undefined"
      :id="itemIdMap?.[item.path]"
      :data-tour="itemTourMap?.[item.path]"
      @click="emit('menu-click', item.path)"
    >
      <span v-if="item.iconSvg" class="h-5 w-5 flex-shrink-0 sidebar-svg-icon" v-html="sanitizeSvg(item.iconSvg)"></span>
      <component v-else :is="item.icon" class="h-5 w-5 flex-shrink-0" />
      <span class="sidebar-label" :class="{ 'sidebar-label-collapsed': collapsed }" :aria-hidden="collapsed ? 'true' : 'false'">{{ item.label }}</span>
    </router-link>
  </template>
</template>

<script lang="ts">
/**
 * Shared nav-item contract for the sidebar navigation. Declared in a plain
 * <script> block so AppSidebar can `import type { NavItem } from './SidebarNavList.vue'`.
 */
export interface NavItem {
  path: string
  label: string
  icon: unknown
  iconSvg?: string
  hideInSimpleMode?: boolean
  children?: NavItem[]
  /** Render a hairline group divider above this top-level entry. */
  dividerBefore?: boolean
  /**
   * When true, the parent item only toggles the expand/collapse state and
   * does NOT navigate to its `path`. The `path` is purely a stable key.
   */
  expandOnly?: boolean
  /**
   * 可选的功能开关 getter。返回 false 时菜单项被隐藏；返回 undefined/true 时显示。
   * 宽容策略（undefined → 显示）避免 public settings 未加载完成时菜单闪烁消失。
   * Getter 里访问的 reactive 来源（store / composable）会被 computed 自动追踪，
   * 开关切换时菜单自动更新。（过滤发生在 AppSidebar，本组件只做纯渲染。）
   */
  featureFlag?: () => boolean | undefined
}
</script>

<script setup lang="ts">
import { h, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { sanitizeSvg } from '@/utils/sanitize'

/**
 * Renders one sidebar section's nav list: optional divider, collapsible groups
 * (with nested children) and plain router-link items. Purely presentational —
 * the `items` array arrives pre-filtered from AppSidebar (feature flags /
 * simple mode), and item clicks are re-emitted so the parent keeps its
 * onboarding-tour logic (`handleMenuItemClick`).
 *
 * Group expand/collapse state is intentionally per-instance: the sections
 * rendered simultaneously (admin + admin-personal for admins) have disjoint
 * group paths, and the admin-personal / user sections — which DO share group
 * paths via `buildSelfNavItems` — never render together (mutually exclusive
 * `v-if`/`v-else-if` in AppSidebar). The route-based fallback in
 * `isGroupExpanded` also does not depend on this set, so no cross-section
 * synchronization is needed.
 */
const props = defineProps<{
  /** Pre-filtered top-level nav entries for one sidebar section. */
  items: NavItem[]
  /** True when the sidebar is collapsed to the icon rail. */
  collapsed: boolean
  /** Optional path → element id map (onboarding-tour anchors, e.g. `#sidebar-channel-manage`). */
  itemIdMap?: Record<string, string>
  /** Optional path → `data-tour` value map (onboarding-tour anchors, e.g. `sidebar-my-keys`). */
  itemTourMap?: Record<string, string>
}>()

const emit = defineEmits<{
  (e: 'menu-click', path: string): void
}>()

const route = useRoute()
const router = useRouter()

// Track which parent nav groups are expanded (per instance, see above).
const expandedGroups = ref<Set<string>>(new Set())

const ChevronDownIcon = {
  render: () =>
    h(
      'svg',
      { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' },
      [
        h('path', {
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round',
          d: 'm19.5 8.25-7.5 7.5-7.5-7.5'
        })
      ]
    )
}

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(path + '/')
}

function isGroupActive(item: NavItem): boolean {
  if (!item.children) return false
  return item.children.some(child => route.path === child.path)
}

function isGroupExpanded(item: NavItem): boolean {
  return expandedGroups.value.has(item.path) || isGroupActive(item)
}

function toggleGroup(item: NavItem) {
  if (expandedGroups.value.has(item.path)) {
    expandedGroups.value.delete(item.path)
  } else {
    expandedGroups.value.add(item.path)
  }
}

/**
 * Click handler for collapsible parent items.
 * - When sidebar is collapsed: do nothing (children are not visible).
 * - When `expandOnly` is true: only toggle expand state.
 * - Otherwise (default, e.g. /admin/orders): navigate to the parent path
 *   (router-link semantics) and ensure the group is expanded.
 */
function handleGroupClick(item: NavItem) {
  // 折叠态没有展开面板：点击组图标直接跳转到组的首个页面。
  if (props.collapsed) {
    router.push(item.path)
    return
  }
  if (item.expandOnly) {
    toggleGroup(item)
    return
  }
  // Push to path and ensure expanded
  if (route.path !== item.path) {
    router.push(item.path)
  }
  if (!expandedGroups.value.has(item.path)) {
    expandedGroups.value.add(item.path)
  }
}
</script>

<style scoped>
.sidebar-link-collapsed {
  gap: 0;
  padding-left: 0.875rem;
  padding-right: 0.875rem;
}

.sidebar-label {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    max-width 0.2s ease,
    opacity 0.12s ease,
    transform 0.12s ease;
  max-width: 12rem;
}

.sidebar-label-flex {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.sidebar-label-collapsed {
  max-width: 0;
  opacity: 0;
  transform: translateX(-4px);
  pointer-events: none;
}

/* Custom SVG icon in sidebar: constrain size without overriding uploaded SVG colors */
.sidebar-svg-icon {
  color: currentColor;
}

.sidebar-svg-icon :deep(svg) {
  display: block;
  width: 1.25rem;
  height: 1.25rem;
}
</style>
