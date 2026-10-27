import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { defaultLanguage, supportedLanguages, translations } from './translations.js';

const LanguageContext = createContext(null);
const STORAGE_KEY = 'landapter.language';
const supportedCodes = new Set(supportedLanguages.map(({ code }) => code));

function detectLanguage() {
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored && supportedCodes.has(stored)) return stored;
  } catch {
    // Storage may be disabled by browser policy; continue with browser locale.
  }

  const browserLanguage = (typeof navigator !== 'undefined' && navigator.language) || '';
  if (browserLanguage.toLowerCase().startsWith('ru')) return 'ru';
  if (browserLanguage.toLowerCase().startsWith('zh')) return 'zh-CN';
  return defaultLanguage;
}

export function LanguageProvider({ children }) {
  const [language, setLanguageState] = useState(detectLanguage);

  const setLanguage = useCallback((nextLanguage) => {
    if (!supportedCodes.has(nextLanguage)) return;
    setLanguageState(nextLanguage);
    try {
      window.localStorage.setItem(STORAGE_KEY, nextLanguage);
    } catch {
      // The selection still applies for the current session without storage.
    }
  }, []);

  const t = useCallback((key, params = {}) => {
    const template = translations[language]?.[key] ?? translations[defaultLanguage]?.[key] ?? key;
    return String(template).replace(/\{([a-zA-Z0-9_]+)\}/g, (match, name) =>
      Object.prototype.hasOwnProperty.call(params, name) ? String(params[name]) : match
    );
  }, [language]);

  useEffect(() => {
    if (typeof document !== 'undefined') document.documentElement.lang = language;
  }, [language]);

  const value = useMemo(() => ({ language, setLanguage, supportedLanguages, t }), [language, setLanguage, t]);
  return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>;
}

export function useI18n() {
  const value = useContext(LanguageContext);
  if (!value) throw new Error('useI18n must be used within LanguageProvider');
  return value;
}
