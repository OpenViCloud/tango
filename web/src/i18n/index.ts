import i18n from "i18next"
import { initReactI18next } from "react-i18next"

import enCommon from "./locales/en/common.json"
import viCommon from "./locales/vi/common.json"

const LANGUAGE_STORAGE_KEY = "tango.locale"
const fallbackLanguage = "en"
const supportedLanguages = ["en", "vi"] as const

const getInitialLanguage = () => {
  const storedLanguage = window.localStorage.getItem(LANGUAGE_STORAGE_KEY)
  if (storedLanguage && supportedLanguages.includes(storedLanguage as (typeof supportedLanguages)[number])) {
    return storedLanguage
  }

  const browserLanguage = window.navigator.language.split("-")[0]
  if (supportedLanguages.includes(browserLanguage as (typeof supportedLanguages)[number])) {
    return browserLanguage
  }

  return fallbackLanguage
}

void i18n.use(initReactI18next).init({
  resources: {
    en: {
      common: enCommon,
    },
    vi: {
      common: viCommon,
    },
  },
  lng: getInitialLanguage(),
  fallbackLng: fallbackLanguage,
  defaultNS: "common",
  interpolation: {
    escapeValue: false,
  },
})

i18n.on("languageChanged", (language) => {
  window.localStorage.setItem(LANGUAGE_STORAGE_KEY, language)
})

export { LANGUAGE_STORAGE_KEY, supportedLanguages }
export default i18n
