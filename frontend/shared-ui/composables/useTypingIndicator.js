import { ref, onUnmounted } from 'vue'

export function useTypingIndicator (sendTypingCallback, otherAttributes = {}) {
  const typingTimer = ref(null)
  const isCurrentlyTyping = ref(false)

  const startTyping = () => {
    if (!isCurrentlyTyping.value) {
      isCurrentlyTyping.value = true
      sendTypingCallback?.(true, otherAttributes)
    }

    // Clear existing timer
    if (typingTimer.value) {
      clearTimeout(typingTimer.value)
    }

    // Set timer to stop typing after 2 seconds of inactivity
    typingTimer.value = setTimeout(() => {
      stopTyping()
    }, 2000)
  }

  const stopTyping = () => {
    setTimeout(() => {
      if (isCurrentlyTyping.value) {
        isCurrentlyTyping.value = false
        sendTypingCallback?.(false, otherAttributes)
      }
    }, 500)

    if (typingTimer.value) {
      clearTimeout(typingTimer.value)
      typingTimer.value = null
    }
  }

  // Clean up on unmount
  onUnmounted(() => {
    stopTyping()
  })

  return {
    startTyping,
    stopTyping,
    isCurrentlyTyping
  }
}
