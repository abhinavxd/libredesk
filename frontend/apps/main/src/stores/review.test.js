import { describe, expect, it } from 'vitest'
import { reviewEventToast } from './review'

const t = (key, values = {}) => `${key}${Object.keys(values).length ? ' ' + JSON.stringify(values) : ''}`
const submission = { uuid: 'r1', author_id: 7, author_name: 'Casey', subject: 'Invoice', status: 'pending' }

describe('review event toasts', () => {
  it('tells reviewers, but not the author, about a new submission', () => {
    expect(reviewEventToast('review_created', submission, { userID: 3, isReviewer: true }, t)).toEqual({
      variant: 'info',
      description: t('review.toast.newSubmission', { name: 'Casey', subject: 'Invoice' })
    })
    expect(reviewEventToast('review_created', submission, { userID: 7, isReviewer: false }, t)).toBeNull()
    expect(reviewEventToast('review_created', submission, { userID: 3, isReviewer: false }, t)).toBeNull()
  })

  it('tells the author when their email is approved or returned, with the reason', () => {
    const approved = { ...submission, status: 'approved', reviewer_name: 'Robin' }
    expect(reviewEventToast('review_updated', approved, { userID: 7 }, t).variant).toBe('success')
    const denied = { ...submission, status: 'denied', reviewer_name: 'Robin', decision_note: 'Too formal' }
    expect(reviewEventToast('review_updated', denied, { userID: 7 }, t)).toEqual({
      variant: 'warning',
      description: t('review.toast.yourDeniedWithReason', { name: 'Robin', subject: 'Invoice', reason: 'Too formal' })
    })
    expect(reviewEventToast('review_updated', { ...denied, decision_note: '' }, { userID: 7 }, t).description).toContain(
      'review.toast.yourDenied '
    )
  })

  it('stays quiet about other people’s decisions and the author’s own withdrawals', () => {
    const approved = { ...submission, status: 'approved', reviewer_name: 'Robin' }
    expect(reviewEventToast('review_updated', approved, { userID: 3, isReviewer: true }, t)).toBeNull()
    expect(reviewEventToast('review_updated', { ...submission, status: 'withdrawn' }, { userID: 7 }, t)).toBeNull()
  })

  it('falls back to a placeholder subject', () => {
    const toast = reviewEventToast('review_created', { ...submission, subject: '' }, { userID: 3, isReviewer: true }, t)
    expect(toast.description).toContain('review.noSubject')
  })
})
