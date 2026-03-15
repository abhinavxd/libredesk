import { format, isToday, isTomorrow, addDays, setHours, setMinutes } from 'date-fns'
import { useI18n } from 'vue-i18n'

/**
 * Business hours composable providing generic business hours utilities.
 */
export function useBusinessHours() {
  const { t } = useI18n()

  function getBusinessHoursById(businessHoursId, businessHoursList) {
    if (!businessHoursId || !businessHoursList) {
      return null
    }

    return businessHoursList.find(bh => bh.id === businessHoursId)
  }

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

  function isHoliday(businessHours, date) {
    if (!businessHours.holidays || businessHours.holidays.length === 0) {
      return false
    }
    const dateStr = format(date, 'yyyy-MM-dd')
    return businessHours.holidays.some(holiday => holiday.date === dateStr)
  }

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

      // Handle same-day logic
      if (i === 0) {
        const currentTime = format(localDate, 'HH:mm')
        // Currently within business hours
        if (currentTime >= schedule.open && currentTime < schedule.close) {
          return new Date(localDate.getTime() - (utcOffset * 60000))
        }
        // Before opening time today
        if (currentTime < schedule.open) {
          return new Date(nextWorking.getTime() - (utcOffset * 60000))
        }
        // Past closing time, check next day
        continue
      }

      // For future days, return the opening time
      // Convert back from business timezone to user timezone
      return new Date(nextWorking.getTime() - (utcOffset * 60000))
    }

    return null
  }

  // Returns English day name to match backend hours keys
  function getDayName(dayNum) {
    return ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'][dayNum]
  }

  function formatNextWorkingTime(nextWorkingTime) {
    if (!nextWorkingTime) {
      return ''
    }

    if (isToday(nextWorkingTime)) {
      return t('globals.messages.backTodayAt', { time: format(nextWorkingTime, 'h:mm a') })
    } else if (isTomorrow(nextWorkingTime)) {
      return t('globals.messages.backTomorrowAt', { time: format(nextWorkingTime, 'h:mm a') })
    } else {
      return t('globals.messages.backOnDayAt', { 
        day: format(nextWorkingTime, 'EEEE'), 
        time: format(nextWorkingTime, 'h:mm a') 
      })
    }
  }
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
        status = t('globals.messages.wellBeBack', { when: formatNextWorkingTime(nextWorkingTime) })
      } else {
        status = t('globals.messages.currentlyOffline')
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
