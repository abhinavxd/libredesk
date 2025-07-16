import { format, isToday, isTomorrow, addDays, setHours, setMinutes } from 'date-fns'

/**
 * Business hours composable providing generic business hours utilities.
 */
export function useBusinessHours() {

  /**
   * Get the business hours by ID from a list
   * @param {number} businessHoursId - Business hours ID
   * @param {Array} businessHoursList - List of business hours objects
   * @returns {Object|null} Business hours object or null
   */
  function getBusinessHoursById(businessHoursId, businessHoursList) {
    if (!businessHoursId || !businessHoursList) {
      return null
    }

    return businessHoursList.find(bh => bh.id === businessHoursId)
  }

  /**
   * Determine which business hours to use based on configuration
   * @param {Object} options - Configuration options
   * @param {boolean} options.showOfficeHours - Whether to show office hours
   * @param {boolean} options.showAfterAssignment - Whether to show team hours after assignment
   * @param {number|null} options.assignedBusinessHoursId - Business hours ID from assignment
   * @param {number|null} options.defaultBusinessHoursId - Default business hours ID
   * @param {Array} options.businessHoursList - List of available business hours
   * @returns {Object|null} Business hours object or null
   */
  function resolveBusinessHours(options) {
    const {
      showOfficeHours,
      showAfterAssignment,
      assignedBusinessHoursId,
      defaultBusinessHoursId,
      businessHoursList
    } = options

    if (!showOfficeHours) {
      return null
    }

    let businessHoursId = null

    // Check if we should use assigned business hours
    if (showAfterAssignment && assignedBusinessHoursId) {
      businessHoursId = assignedBusinessHoursId
    } else if (defaultBusinessHoursId) {
      // Fallback to default business hours
      businessHoursId = parseInt(defaultBusinessHoursId)
    }

    return getBusinessHoursById(businessHoursId, businessHoursList)
  }

  /**
   * Check if a given time is within business hours
   * @param {Object} businessHours - Business hours object
   * @param {Date} date - Date to check
   * @param {number} utcOffset - UTC offset in minutes
   * @returns {boolean} True if within business hours
   */
  function isWithinBusinessHours(businessHours, date, utcOffset = 0) {
    if (!businessHours || businessHours.is_always_open) {
      return true
    }

    // Convert to business timezone
    const localDate = new Date(date.getTime() + (utcOffset * 60000))
    
    // Check if it's a holiday
    if (isHoliday(businessHours, localDate)) {
      return false
    }

    const dayName = getDayName(localDate.getDay())
    const schedule = businessHours.hours[dayName]

    if (!schedule || !schedule.open || !schedule.close) {
      return false
    }

    // Check if open and close times are the same (closed day)
    if (schedule.open === schedule.close) {
      return false
    }

    const currentTime = format(localDate, 'HH:mm')
    return currentTime >= schedule.open && currentTime <= schedule.close
  }

  /**
   * Check if a date is a holiday
   * @param {Object} businessHours - Business hours object
   * @param {Date} date - Date to check
   * @returns {boolean} True if it's a holiday
   */
  function isHoliday(businessHours, date) {
    if (!businessHours.holidays || businessHours.holidays.length === 0) {
      return false
    }
    const dateStr = format(date, 'yyyy-MM-dd')
    return businessHours.holidays.some(holiday => holiday.date === dateStr)
  }

  /**
   * Get the next working time
   * @param {Object} businessHours - Business hours object
   * @param {Date} fromDate - Date to start from
   * @param {number} utcOffset - UTC offset in minutes
   * @returns {Date|null} Next working time or null
   */
  function getNextWorkingTime(businessHours, fromDate, utcOffset = 0) {
    if (!businessHours || businessHours.is_always_open) {
      return fromDate
    }

    // Check up to 14 days ahead
    for (let i = 0; i < 14; i++) {
      const checkDate = addDays(fromDate, i)
      const localDate = new Date(checkDate.getTime() + (utcOffset * 60000))

      // Skip holidays
      if (isHoliday(businessHours, localDate)) {
        continue
      }

      const dayName = getDayName(localDate.getDay())
      const schedule = businessHours.hours[dayName]

      if (!schedule || !schedule.open || !schedule.close || schedule.open === schedule.close) {
        continue
      }

      // Parse opening time
      const [openHour, openMinute] = schedule.open.split(':').map(Number)
      let nextWorking = setMinutes(setHours(localDate, openHour), openMinute)

      // If it's the same day and current time is before opening time
      if (i === 0) {
        const currentTime = format(localDate, 'HH:mm')
        if (currentTime < schedule.open) {
          // Convert back from business timezone to user timezone
          return new Date(nextWorking.getTime() - (utcOffset * 60000))
        }
        // If it's the same day but past opening time, continue to next day
        continue
      }

      // For future days, return the opening time
      // Convert back from business timezone to user timezone
      return new Date(nextWorking.getTime() - (utcOffset * 60000))
    }

    return null
  }

  /**
   * Get day name from day number
   * @param {number} dayNum - Day number (0 = Sunday, 1 = Monday, etc.)
   * @returns {string} Day name
   */
  function getDayName(dayNum) {
    const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
    return days[dayNum]
  }

  /**
   * Format the next working time for display
   * @param {Date} nextWorkingTime - Next working time
   * @returns {string} Formatted string
   */
  function formatNextWorkingTime(nextWorkingTime) {
    if (!nextWorkingTime) {
      return ''
    }

    if (isToday(nextWorkingTime)) {
      return `today at ${format(nextWorkingTime, 'h:mm a')}`
    } else if (isTomorrow(nextWorkingTime)) {
      return `tomorrow at ${format(nextWorkingTime, 'h:mm a')}`
    } else {
      return `on ${format(nextWorkingTime, 'EEEE')} at ${format(nextWorkingTime, 'h:mm a')}`
    }
  }
  /**
   * Get business hours status message and whether it's within business hours
   * @param {Object} businessHours - Business hours object
   * @param {number} utcOffset - UTC offset in minutes
   * @param {string} withinHoursMessage - Message to show when within hours
   * @returns {Object|null} { status: string|null, isWithin: boolean } or null
   */
  function getBusinessHoursStatus(businessHours, utcOffset = 0, withinHoursMessage = '') {
    if (!businessHours) {
      return null
    }

    const now = new Date()
    const within = isWithinBusinessHours(businessHours, now, utcOffset)

    let status = null
    if (within) {
      status = withinHoursMessage
    } else {
      const nextWorkingTime = getNextWorkingTime(businessHours, now, utcOffset)
      if (nextWorkingTime) {
        status = `We'll be back ${formatNextWorkingTime(nextWorkingTime)}`
      } else {
        status = 'We are currently offline'
      }
    }

    return { status, isWithin: within }
  }

  return {
    getBusinessHoursById,
    resolveBusinessHours,
    isWithinBusinessHours,
    getNextWorkingTime,
    formatNextWorkingTime,
    getBusinessHoursStatus,
    isHoliday,
    getDayName
  }
}
