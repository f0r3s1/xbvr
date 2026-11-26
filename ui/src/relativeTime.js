import i18n from './i18n'

const UNITS = [['year', 31536000], ['month', 2592000], ['week', 604800], ['day', 86400], ['hour', 3600], ['minute', 60]]

// "in 12 hours" / "vor 3 Tagen" in the UI language, from the browser's own formatter
export function relativeTime (date) {
  const seconds = (new Date(date).getTime() - Date.now()) / 1000
  const rtf = new Intl.RelativeTimeFormat(i18n.global.locale.value, { numeric: 'auto' })
  for (const [unit, size] of UNITS) {
    if (Math.abs(seconds) >= size) return rtf.format(Math.round(seconds / size), unit)
  }
  return rtf.format(Math.round(seconds), 'second')
}
