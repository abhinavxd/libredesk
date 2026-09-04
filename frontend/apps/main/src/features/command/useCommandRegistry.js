import { computed } from 'vue'
import { useUserStore } from '@main/stores/user'

export const normalizeSearchText = (text) =>
  String(text || '')
    .toLowerCase()
    .replace(/\s+/g, ' ')
    .trim()

export const commandMatches = (command, term) => {
  const needle = normalizeSearchText(term)
  if (!needle) return true
  const haystack = normalizeSearchText([command.label, ...(command.keywords || [])].join(' '))
  return needle.split(' ').every((word) => haystack.includes(word))
}

export function useCommandRegistry(providers) {
  const userStore = useUserStore()

  const allowed = (command) => {
    if (!command.permission) return true
    const required = Array.isArray(command.permission) ? command.permission : [command.permission]
    return required.some((permission) => userStore.can(permission))
  }

  // Children inherit their parent's visibility.
  const commands = computed(() => {
    const all = providers.flatMap((provider) => provider.value)
    const byId = new Map(all.map((command) => [command.id, command]))
    const visible = new Map()
    const isVisible = (command) => {
      if (visible.has(command.id)) return visible.get(command.id)
      let result = allowed(command)
      if (result && command.parent) {
        const parent = byId.get(command.parent)
        result = Boolean(parent) && isVisible(parent)
      }
      visible.set(command.id, result)
      return result
    }
    return all.filter(isVisible)
  })

  const byId = computed(() => new Map(commands.value.map((command) => [command.id, command])))

  const get = (id) => byId.value.get(id) || null

  const childrenOf = (parentId) =>
    commands.value.filter((command) => (command.parent || null) === (parentId || null))

  return { commands, get, childrenOf }
}
