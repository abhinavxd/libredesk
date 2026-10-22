import { ref, isRef } from 'vue'

export function useSendDelay(delayMs = 5000) {
  const pendingSendTimer = ref(null)
  const sendCountdown = ref(0)
  const sendCancelResolve = ref(null)
  const sendProgressStyle = ref({ width: '100%', transition: 'none' })

  const cancelPendingSend = () => {
    if (pendingSendTimer.value) {
      clearInterval(pendingSendTimer.value)
      pendingSendTimer.value = null
      sendCountdown.value = 0
      sendProgressStyle.value = { width: '100%', transition: 'none' }
      if (sendCancelResolve.value) {
        sendCancelResolve.value(true)
        sendCancelResolve.value = null
      }
    }
  }

  // Returns true if the send was cancelled, false if the delay elapsed normally.
  const executeSendWithDelay = () => {
    const ms = isRef(delayMs) ? delayMs.value : delayMs
    return new Promise((resolve) => {
      sendCountdown.value = Math.round(ms / 1000)
      sendCancelResolve.value = resolve
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          sendProgressStyle.value = { width: '0%', transition: `width ${ms}ms linear` }
        })
      })
      const tick = setInterval(() => {
        sendCountdown.value -= 1
        if (sendCountdown.value <= 0) {
          clearInterval(tick)
          pendingSendTimer.value = null
          sendCancelResolve.value = null
          sendProgressStyle.value = { width: '100%', transition: 'none' }
          resolve(false)
        }
      }, 1000)
      pendingSendTimer.value = tick
    })
  }

  return {
    pendingSendTimer,
    sendCountdown,
    sendProgressStyle,
    cancelPendingSend,
    executeSendWithDelay
  }
}
