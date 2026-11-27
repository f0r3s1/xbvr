// dates such as releases and birthdays are stored as UTC days; read them as that day whatever the browser's time zone
export function utcDay (iso) {
  const d = new Date(iso)
  return new Date(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate())
}

// an unset date is year 1, which a server outside UTC sends as e.g. "0000-12-31T19:03:58-04:56"
export function hasDate (iso) {
  return !!iso && new Date(iso).getUTCFullYear() > 1900
}

export function isoDay (iso) {
  return new Date(iso).toISOString().slice(0, 10)
}
