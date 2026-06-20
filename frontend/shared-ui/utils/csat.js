const RATING_EMOJI = { 1: '😢', 2: '😕', 3: '😊', 4: '😃', 5: '🤩' }
const RATING_TEXT_KEY = {
  1: 'globals.terms.poor',
  2: 'globals.terms.fair',
  3: 'globals.terms.good',
  4: 'globals.terms.great',
  5: 'globals.terms.excellent'
}

export function csatRatingEmoji (rating) {
  return RATING_EMOJI[rating] || ''
}

export function csatRatingTextKey (rating) {
  return RATING_TEXT_KEY[rating] || ''
}
