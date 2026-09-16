import { ref } from 'vue'
import { MACRO_CONTEXT } from '@main/constants/conversation'

export const MACROS_COMMAND = 'macros'

const open = ref(false)
const parent = ref(null)
const searchTerm = ref('')
const macroContext = ref(MACRO_CONTEXT.REPLY)

export function useCommandPalette() {
  const openPalette = ({ parent: target = null } = {}) => {
    parent.value = target
    searchTerm.value = ''
    open.value = true
  }

  const closePalette = () => {
    open.value = false
  }

  const togglePalette = () => {
    if (open.value) closePalette()
    else openPalette({ parent: defaultParent() })
  }

  const setParent = (target) => {
    parent.value = target
    searchTerm.value = ''
  }

  // Ctrl+K inside the new-conversation dialog goes straight to macros.
  const defaultParent = () =>
    macroContext.value === MACRO_CONTEXT.NEW_CONVERSATION ? MACROS_COMMAND : null

  const setMacroContext = (context) => {
    macroContext.value = context
  }

  return {
    open,
    parent,
    searchTerm,
    macroContext,
    openPalette,
    closePalette,
    togglePalette,
    setParent,
    setMacroContext
  }
}
