import { createI18n } from 'vue-i18n'
import enUS from './locales/en-US.ts'
import koKR from './locales/ko-KR.ts'

const messages = {
  'en-US': enUS,
  'ko-KR': koKR
}

// localStorage에서 저장된 언어를 가져오거나 한국어를 기본값으로 사용
const savedLocale = localStorage.getItem('locale') || 'ko-KR'
console.log('i18n 초기화 언어:', savedLocale)

const i18n = createI18n({
  legacy: false,
  locale: savedLocale,
  fallbackLocale: 'ko-KR',
  globalInjection: true,
  messages
})

export default i18n
