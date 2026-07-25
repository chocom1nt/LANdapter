import React, { useState } from 'react';
import ThemeToggle from '../components/ThemeToggle';
import { useApp } from '../contexts/AppContext';

const Settings = () => {
  const { language, setLanguage } = useApp();
  const [apiUrl, setApiUrl] = useState(() => {
    return localStorage.getItem('api_url') || '/api/v1';
  });

  const handleApiUrlChange = (e) => {
    const val = e.target.value;
    setApiUrl(val);
    localStorage.setItem('api_url', val);
    window.location.reload();
  };

  const handleLanguageChange = (e) => {
    const val = e.target.value;
    setLanguage(val);
    localStorage.setItem('language', val);
  };

  return (
    <div>
      <h2 className="text-3xl font-bold mb-6">Настройки</h2>
      <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow space-y-6">
        <div className="flex items-center justify-between">
          <span className="text-lg font-medium">Тема</span>
          <ThemeToggle compact />
        </div>
        <div className="border-t dark:border-gray-700 pt-4">
          <label className="block text-sm font-medium mb-1">Язык интерфейса</label>
          <select
            value={language}
            onChange={handleLanguageChange}
            className="w-full border rounded px-3 py-2 dark:bg-gray-700 dark:border-gray-600"
          >
            <option value="ru">Русский</option>
            <option value="en">English</option>
          </select>
        </div>
        <div className="border-t dark:border-gray-700 pt-4">
          <label className="block text-sm font-medium mb-1">API сервер</label>
          <input
            type="text"
            value={apiUrl}
            onChange={handleApiUrlChange}
            className="w-full border rounded px-3 py-2 dark:bg-gray-700 dark:border-gray-600"
            placeholder="/api/v1"
          />
          <p className="text-xs text-gray-500 mt-1">Перезагрузит страницу для применения</p>
        </div>
        <div className="border-t dark:border-gray-700 pt-4">
          <p className="text-sm text-gray-500">Версия приложения: 0.1.0</p>
        </div>
      </div>
    </div>
  );
};

export default Settings;