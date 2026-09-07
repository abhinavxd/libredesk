import { ref } from 'vue'

const open = ref(false)

export function useKeyboardShortcutsDialog() {
  const show = () => {
    open.value = true
  }
  return { open, show }
}
