import timezones from '@shared-ui/constants/timezones.json'

const now = new Date()

const formatLabel = (timeZone) => {
  const label = timeZone.replaceAll('_', ' ')
  try {
    const offset = new Intl.DateTimeFormat('en', {
      timeZone,
      timeZoneName: 'longOffset'
    })
      .formatToParts(now)
      .find((part) => part.type === 'timeZoneName').value
    return `${label} (${offset === 'GMT' ? 'UTC+00:00' : offset.replace('GMT', 'UTC')})`
  } catch {
    return label
  }
}

export const timeZoneOptions = [
  { label: formatLabel('UTC'), value: 'UTC' },
  ...timezones
    .filter((zone) => zone !== 'UTC')
    .map((zone) => ({ label: formatLabel(zone), value: zone }))
]
