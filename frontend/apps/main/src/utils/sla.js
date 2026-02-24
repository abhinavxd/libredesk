import { differenceInMinutes } from 'date-fns'

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
                value: `${Math.floor(diffInMinutes / 1440)} days`
            }
        }
        return {
            status: 'remaining',
            value: diffInMinutes < 60 ? `${diffInMinutes} mins` : `${Math.floor(diffInMinutes / 60)} hrs`
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
            value: `${Math.floor(overdueMins / 1440)} days`
        }
    }
    return {
        status,
        value: overdueMins < 60 ? `${overdueMins} mins` : `${Math.floor(overdueMins / 60)} hrs`
    }
}