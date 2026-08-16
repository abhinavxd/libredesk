export function applyTemporaryClass (containerID, className, timeMs = 300) {
  const container = document.getElementById(containerID)
  if (!container) return
  container.classList.add(className)
  setTimeout(() => {
    container.classList.remove(className)
  }, timeMs)
}
