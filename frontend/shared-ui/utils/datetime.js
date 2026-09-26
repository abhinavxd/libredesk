import { format, differenceInMinutes, differenceInHours, differenceInDays, differenceInMonths, differenceInYears } from 'date-fns'

export function getRelativeTime (timestamp, now = new Date()) {
  try {
    const mins = differenceInMinutes(now, timestamp)
    const hours = differenceInHours(now, timestamp)
    const days = differenceInDays(now, timestamp)
    const months = differenceInMonths(now, timestamp)
    const years = differenceInYears(now, timestamp)

    if (mins === 0) return 'now'
    if (mins < 60) return `${mins}m`
    if (hours < 24) return `${hours}h`
    if (days < 31) return `${days}d`
    if (months < 12) return `${months}mo`
    return `${years}y`
  } catch (error) {
    console.error('Error parsing time', error, 'timestamp', timestamp)
    return ''
  }
}

export const formatDuration = (seconds, showSeconds = true) => {
  const totalSeconds = Math.floor(seconds)
  if (totalSeconds < 60) return `${totalSeconds}s`
  if (totalSeconds < 3600) return `${Math.floor(totalSeconds / 60)}m ${totalSeconds % 60}s`
  const hours = Math.floor(totalSeconds / 3600)
  const mins = Math.floor((totalSeconds % 3600) / 60)
  const secs = totalSeconds % 60
  return `${hours}h ${mins}m ${showSeconds ? `${secs}s` : ''}`
}

// Desk-wide time format, '12h' or '24h'. The agent app sets it once settings load, the widget keeps the default.
let timeFormat = '12h'

export const setTimeFormat = (value) => {
  timeFormat = value === '24h' ? '24h' : '12h'
}

// timePattern returns the date-fns time pattern for the desk-wide time format.
export const timePattern = (withSeconds = false) => {
  if (timeFormat === '24h') return withSeconds ? 'HH:mm:ss' : 'HH:mm'
  return withSeconds ? 'pp' : 'p'
}

export const formatMessageTimestamp = (time) => {
  return format(time, timeFormat === '24h' ? 'd MMM, HH:mm' : 'd MMM, hh:mm a')
}

export const formatFullTimestamp = (time) => {
  return format(time, timeFormat === '24h' ? 'd MMM yyyy, HH:mm' : 'd MMM yyyy, hh:mm a')
}

export const formatDateTime = (time) => {
  return format(time, timeFormat === '24h' ? 'PP, HH:mm:ss' : 'PPpp')
}
