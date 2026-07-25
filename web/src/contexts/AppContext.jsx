import React, { createContext, useContext, useState, useEffect } from 'react';

const AppContext = createContext();

export const AppProvider = ({ children }) => {
  const [files, setFiles] = useState(() => {
    try {
      const saved = localStorage.getItem('uploadedFiles');
      return saved ? JSON.parse(saved) : [];
    } catch {
      return [];
    }
  });

  useEffect(() => {
    localStorage.setItem('uploadedFiles', JSON.stringify(files));
  }, [files]);

  const [language, setLanguage] = useState(() => {
    return localStorage.getItem('language') || 'ru';
  });

  const addFile = (fileId, fileName) => {
    setFiles(prev => [{ id: fileId, name: fileName, uploadedAt: new Date().toLocaleString() }, ...prev]);
  };

  const removeFile = (fileId) => {
    setFiles(prev => prev.filter(f => f.id !== fileId));
  };

  // Простые переводы
  const translations = {
    ru: {
      dashboard: 'Панель управления',
      clients: 'Клиенты',
      files: 'Файлы',
      install: 'Установка',
      settings: 'Настройки',
      total_clients: 'Всего клиентов',
      online: 'Онлайн',
      offline: 'Офлайн',
    },
    en: {
      dashboard: 'Dashboard',
      clients: 'Clients',
      files: 'Files',
      install: 'Install',
      settings: 'Settings',
      total_clients: 'Total clients',
      online: 'Online',
      offline: 'Offline',
    },
  };

  const t = (key) => translations[language]?.[key] || key;

  return (
    <AppContext.Provider value={{ files, addFile, removeFile, language, setLanguage, t }}>
      {children}
    </AppContext.Provider>
  );
};

export const useApp = () => {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('useApp must be used within AppProvider');
  }
  return context;
};