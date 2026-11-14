import { differenceInMinutes } from 'date-fns'
import { getRelativeTime } from '@shared-ui/utils/datetime'

/**
 * Calculates the SLA (Service Level Agreement) status based on the due date.
 *
 * @param {string} dueAt - The due date and time in ISO format.
 * @param {string} actualAt - The actual date and time in ISO format.
 * @returns {Object} An object containing the SLA status and the remaining or overdue time.
 * @returns {string} return.status - The SLA status, either 'remaining' or 'overdue'.
 * @returns {string} return.value - The remaining or overdue time in minutes, hours, or days.
 */
export function calculateSla (dueAt, actualAt) {
    const compareTime = actualAt ? new Date(actualAt) : new Date()
    const dueTime = new Date(dueAt)
    // Difference in minutes will be negative if overdue, positive if remaining.
    const diffInMinutes = differenceInMinutes(dueTime, compareTime)

    // No actual at and diffInMinutes is positive; there is still time remaining.
    if (!actualAt && diffInMinutes >= 0) {
        if (diffInMinutes >= 2880) {
            return {
                status: 'remaining',
                value: getRelativeTime(compareTime, dueTime)
            }
        }
        return {
            status: 'remaining',
            value: diffInMinutes < 60 ? `${diffInMinutes}m` : `${Math.floor(diffInMinutes / 60)}h`
        }
    }

    let status = 'hit'
    if (diffInMinutes < 0) {
        status = 'overdue'
    }

    const overdueMins = Math.abs(diffInMinutes)
    if (overdueMins >= 2880) {
        return {
            status,
            value: diffInMinutes < 0 ? getRelativeTime(dueTime, compareTime) : getRelativeTime(compareTime, dueTime)
        }
    }
    return {
        status,
        value: overdueMins < 60 ? `${overdueMins}m` : `${Math.floor(overdueMins / 60)}h`
    }
}
