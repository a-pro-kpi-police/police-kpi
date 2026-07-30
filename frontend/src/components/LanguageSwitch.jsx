import { useI18n } from '../i18n.jsx'

export default function LanguageSwitch({ className = '' }) {
  const { t, lang, setLang } = useI18n()

  return (
    <div
      className={`language-switch ${className}`.trim()}
      data-lang={lang}
      role="group"
      aria-label={t('languageSwitch')}
    >
      <button
        type="button"
        className={lang === 'ru' ? 'active' : ''}
        aria-pressed={lang === 'ru'}
        onClick={() => setLang('ru')}
      >
        RU
      </button>
      <button
        type="button"
        className={lang === 'kk' ? 'active' : ''}
        aria-pressed={lang === 'kk'}
        onClick={() => setLang('kk')}
      >
        KZ
      </button>
    </div>
  )
}
