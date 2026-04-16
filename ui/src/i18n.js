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

export default createI18n({
  legacy: false,
  globalInjection: true,
  locale: navigator.language,
  fallbackLocale: 'en-GB',
  messages: loadLocaleMessages(),
  missingWarn: false,
  fallbackWarn: false
})
