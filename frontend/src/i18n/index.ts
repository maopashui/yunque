import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from './locales/en'
import zh from './locales/zh'

export type AppLocale = 'zh' | 'en'

const resources = {
  zh: { translation: zh },
  en: { translation: en },
} as const

/** English when the OS/browser language is English; otherwise Chinese. */
export function localeFromSystem(): AppLocale {
  const tags =
    typeof navigator === 'undefined'
      ? []
      : navigator.languages?.length
        ? navigator.languages
        : [navigator.language]
  const primary = (tags[0] ?? 'zh').toLowerCase()
  return primary.startsWith('en') ? 'en' : 'zh'
}

export function htmlLang(locale: string): string {
  return locale === 'en' ? 'en' : 'zh-CN'
}

export function applyDocumentLang(locale: string) {
  if (typeof document === 'undefined') return
  document.documentElement.lang = htmlLang(locale)
}

void i18n.use(initReactI18next).init({
  resources,
  lng: localeFromSystem(),
  fallbackLng: 'zh',
  interpolation: { escapeValue: false },
})

applyDocumentLang(i18n.language)

i18n.on('languageChanged', (lng) => {
  applyDocumentLang(lng)
})

if (typeof window !== 'undefined') {
  window.addEventListener('languagechange', () => {
    void i18n.changeLanguage(localeFromSystem())
  })
}

export default i18n
