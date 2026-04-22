import { computed, watch } from 'vue'
import { useStorage } from '@vueuse/core'

// Auto-discover themes from `frontend/apps/main/src/themes/*/theme.js`.
// Each theme directory should export a `{ id, label }` object as default and
// (optionally) ship a `theme.scss` next to it. Both files are picked up by
// Vite glob so a drop-in theme requires zero edits to this file or any other
// upstream code.
//
//   themes/
//     <name>/
//       theme.js      // export default { id: '<name>', label: '<Label>' }
//       theme.scss    // [data-theme="<name>"] { --primary: ...; ... }
//
// CSS targets `[data-theme="<id>"]` so themes are scoped to opt-in. The
// default theme has no data-theme attribute, so stock upstream is unchanged.
const themeModules = import.meta.glob('../themes/*/theme.js', { eager: true })
// Side-effect import so each theme's stylesheet is included in the bundle.
import.meta.glob('../themes/*/theme.scss', { eager: true })

const DEFAULT_THEME = { id: 'default', label: 'Default' }

const discovered = Object.values(themeModules)
  .map((mod) => mod.default || mod)
  .filter((t) => t && typeof t.id === 'string' && t.id !== 'default')

export const THEMES = [DEFAULT_THEME, ...discovered]

const STORAGE_KEY = 'libredesk-theme'
const DEFAULT_ID = DEFAULT_THEME.id

const isValid = (id) => THEMES.some((t) => t.id === id)

export function useTheme () {
  const stored = useStorage(STORAGE_KEY, DEFAULT_ID)
  const activeTheme = computed(() => (isValid(stored.value) ? stored.value : DEFAULT_ID))

  // Reflect the active theme on <html data-theme="..."> so themes can opt in
  // via attribute selectors. Default is left as-is so it costs nothing.
  watch(
    activeTheme,
    (id) => {
      if (typeof document === 'undefined') return
      if (id === DEFAULT_ID) {
        document.documentElement.removeAttribute('data-theme')
      } else {
        document.documentElement.setAttribute('data-theme', id)
      }
    },
    { immediate: true }
  )

  function setTheme (id) {
    if (isValid(id)) stored.value = id
  }

  return {
    THEMES,
    currentTheme: activeTheme,
    setTheme,
    hasMultipleThemes: computed(() => THEMES.length > 1)
  }
}
