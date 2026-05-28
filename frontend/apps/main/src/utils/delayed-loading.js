const pendingHides = new WeakMap()

export function delayedLoading (target, key, delayMs = 300, minVisibleMs = 500) {
  const prevHide = pendingHides.get(target)?.get(key)
  if (prevHide !== undefined) {
    clearTimeout(prevHide)
    pendingHides.get(target).delete(key)
  }

  let triggeredAt = 0
  let showTimer = null

  if (target[key] === true) {
    triggeredAt = Date.now()
  } else {
    showTimer = setTimeout(() => {
      triggeredAt = Date.now()
      target[key] = true
    }, delayMs)
  }

  return {
    release () {
      if (showTimer) clearTimeout(showTimer)
      if (!triggeredAt) return
      const remaining = minVisibleMs - (Date.now() - triggeredAt)
      if (remaining <= 0) {
        target[key] = false
        return
      }
      const hideTimer = setTimeout(() => {
        target[key] = false
        pendingHides.get(target)?.delete(key)
      }, remaining)
      let map = pendingHides.get(target)
      if (!map) {
        map = new Map()
        pendingHides.set(target, map)
      }
      map.set(key, hideTimer)
    },
    cancel () {
      if (showTimer) clearTimeout(showTimer)
      const pendingHide = pendingHides.get(target)?.get(key)
      if (pendingHide !== undefined) {
        clearTimeout(pendingHide)
        pendingHides.get(target).delete(key)
      }
      target[key] = false
    }
  }
}
