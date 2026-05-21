import notificationSound from '../assets/notification.mp3'

const PLAY_THROTTLE_MS = 1500
let audio = null
let lastPlayedAt = 0

export function initAudioContext() {
  if (audio) return
  audio = new Audio(notificationSound)
  audio.volume = 0.5
  audio.load()
}

export function playNotificationSound() {
  if (!audio) return
  const now = Date.now()
  if (now - lastPlayedAt < PLAY_THROTTLE_MS) return
  lastPlayedAt = now
  audio.currentTime = 0
  audio.play().catch(() => {})
}
