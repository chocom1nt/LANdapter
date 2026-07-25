import React from 'react';
import { NavLink } from 'react-router-dom';
import ThemeToggle from './ThemeToggle';
import { FaTachometerAlt, FaDesktop, FaFileAlt, FaBox, FaCog } from 'react-icons/fa';

const Sidebar = () => {
  const navItems = [
    { path: '/', label: 'Дашборд', icon: FaTachometerAlt },
    { path: '/clients', label: 'Клиенты', icon: FaDesktop },
    { path: '/files', label: 'Файлы', icon: FaFileAlt },
    { path: '/install', label: 'Установка', icon: FaBox },
    { path: '/settings', label: 'Настройки', icon: FaCog },
  ];

  return (
    <aside className="w-64 bg-gray-100 dark:bg-gray-800 h-screen p-4 flex flex-col border-r border-gray-200 dark:border-gray-700 transition-colors">
      <h1 className="text-2xl font-bold text-blue-600 dark:text-blue-400 mb-8">LANdapter</h1>
      <nav className="flex-1 space-y-2">
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) =>
              `flex items-center space-x-3 px-4 py-2 rounded-lg transition-colors ${
                isActive
                  ? 'bg-blue-500 text-white dark:bg-blue-600'
                  : 'text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700'
              }`
            }
          >
            <item.icon className="w-5 h-5" />
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>
      <div className="mt-auto pt-4 border-t border-gray-200 dark:border-gray-700">
        <ThemeToggle />
      </div>
    </aside>
  );
};

export default Sidebar;