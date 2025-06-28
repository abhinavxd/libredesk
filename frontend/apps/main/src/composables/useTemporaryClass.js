import { ref, onUnmounted, nextTick } from 'vue'

export function useTemporaryClass(containerID, className, timeMs = 300) {
  const container = ref(null)
  const applyClass = async () => {
    await nextTick()
    container.value = document.getElementById(containerID)
    if (container.value) {
      container.value.classList.add(className)
      setTimeout(() => {
        container.value?.classList.remove(className)
      }, timeMs)
    }
  }
  applyClass()
  onUnmounted(() => {
    if (container.value) {
      container.value.classList.remove(className)
      container.value = null
    }
  })
  return {
    container
  }
}
