
import { defineMessage as msg } from '@lingui/core/macro'
const RATING_EMOJI = { 1: '😢', 2: '😕', 3: '😊', 4: '😃', 5: '🤩' }
const RATING_MESSAGES = {
  1: msg({ message: 'Poor' }),
  2: msg({ message: 'Fair' }),
  3: msg({ message: 'Good' }),
  4: msg({ message: 'Great' }),
  5: msg({ message: 'Excellent' })
}

export function csatRatingEmoji (rating) {
  return RATING_EMOJI[rating] || ''
}

export function csatRatingMessage (rating) {
  return RATING_MESSAGES[rating] || ''
}
