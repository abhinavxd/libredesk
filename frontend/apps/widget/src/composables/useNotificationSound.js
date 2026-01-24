import notificationSound from '../assets/notification.mp3'

let audio = null

// Call on first user interaction to unlock audio
export function initAudioContext() {
  if (audio) return
  audio = new Audio(notificationSound)
  audio.volume = 0.5
  audio.load()
}

export function playNotificationSound() {
  if (!audio) return
  audio.currentTime = 0
  audio.play().catch(() => {})
}
