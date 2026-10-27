import { useI18n } from './LanguageProvider.jsx';

export default function LanguageSelector({ className = '' }) {
  const { language, setLanguage, supportedLanguages, t } = useI18n();

  return (
    <label className={`language-selector ${className}`.trim()}>
      <span className="language-selector__label">{t('common.language')}</span>
      <select
        aria-label={t('common.language')}
        value={language}
        onChange={(event) => setLanguage(event.target.value)}
      >
        {supportedLanguages.map(({ code, label }) => (
          <option key={code} value={code}>{label}</option>
        ))}
      </select>
    </label>
  );
}
