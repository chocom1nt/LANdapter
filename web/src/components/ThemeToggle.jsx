import React from 'react';
import { useTheme } from '../contexts/ThemeContext';

const ThemeToggle = ({ compact = false }) => {
  const { theme, toggleTheme } = useTheme();

  return (
    <button
      onClick={toggleTheme}
      className={`${
        compact ? 'px-3 py-1 text-sm' : 'w-full flex items-center justify-center px-4 py-2'
      } rounded-lg bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors`}
    >
      {theme === 'light' ? '🌙 Тёмная' : '☀️ Светлая'}
    </button>
  );
};

export default ThemeToggle;