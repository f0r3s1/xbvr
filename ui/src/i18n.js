import { createI18n } from 'vue-i18n'

const localeModules = import.meta.glob('./locales/*.json', { eager: true })

function loadLocaleMessages () {
  const messages = {}
  for (const path in localeModules) {
    const matched = path.match(/([A-Za-z0-9-_]+)\.json$/i)
    if (matched && matched.length > 1) {
      messages[matched[1]] = localeModules[path].default || localeModules[path]
    }
  }
  return messages
}

const messages = loadLocaleMessages()

// the browser's preferred languages in order; "fr-CA" also matches a plain "fr" file
function browserLocale () {
  const available = Object.keys(messages)
  for (const wanted of navigator.languages || [navigator.language]) {
    const lang = wanted.split('-')[0]
    const match = available.find(l => l === wanted) || available.find(l => l === lang || l.split('-')[0] === lang)
    if (match) return match
  }
  return 'en-GB'
}

const LANGUAGE_KEY = 'language'

function savedLanguage () {
  try { return localStorage.getItem(LANGUAGE_KEY) } catch { return null }
}

export const availableLocales = Object.keys(messages)

const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: (availableLocales.includes(savedLanguage()) && savedLanguage()) || browserLocale(),
  fallbackLocale: 'en-GB',
  messages,
  missingWarn: false,
  fallbackWarn: false
})

// null follows the browser again
export function setLanguage (locale) {
  try {
    if (locale) localStorage.setItem(LANGUAGE_KEY, locale)
    else localStorage.removeItem(LANGUAGE_KEY)
  } catch { /* storage blocked: the choice lasts for this visit */ }
  i18n.global.locale.value = locale || browserLocale()
}

export function chosenLanguage () {
  return savedLanguage()
}

export default i18n
