import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { getClients, getFiles } from '../api';
import LoadingSpinner from '../components/common/LoadingSpinner';
import { FaDesktop, FaFileAlt, FaCheckCircle, FaTimesCircle } from 'react-icons/fa';

const Dashboard = () => {
  const [clients, setClients] = useState([]);
  const [files, setFiles] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const clientsData = await getClients();
        setClients(Array.isArray(clientsData) ? clientsData : []);
      } catch (err) {
        console.error('Failed to fetch clients:', err);
      }
      try {
        const filesData = await getFiles();
        setFiles(Array.isArray(filesData) ? filesData : []);
      } catch (err) {
        console.error('Failed to fetch files:', err);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
    const interval = setInterval(fetchData, 15000);
    return () => clearInterval(interval);
  }, []);

  const online = clients.filter(c => c.online).length;
  const offline = clients.length - online;

  if (loading) return <LoadingSpinner />;

  return (
    <div>
      <h2 className="text-3xl font-bold mb-6">Панель управления</h2>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow transition hover:scale-105">
          <div className="flex items-center gap-4">
            <FaDesktop className="text-blue-500 w-8 h-8" />
            <div>
              <h3 className="text-lg font-semibold text-gray-500 dark:text-gray-400">Клиенты</h3>
              <p className="text-3xl font-bold">{clients.length}</p>
            </div>
          </div>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow transition hover:scale-105">
          <div className="flex items-center gap-4">
            <FaCheckCircle className="text-green-500 w-8 h-8" />
            <div>
              <h3 className="text-lg font-semibold text-gray-500 dark:text-gray-400">Онлайн</h3>
              <p className="text-3xl font-bold text-green-600">{online}</p>
            </div>
          </div>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow transition hover:scale-105">
          <div className="flex items-center gap-4">
            <FaTimesCircle className="text-red-500 w-8 h-8" />
            <div>
              <h3 className="text-lg font-semibold text-gray-500 dark:text-gray-400">Офлайн</h3>
              <p className="text-3xl font-bold text-red-600">{offline}</p>
            </div>
          </div>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow transition hover:scale-105">
          <div className="flex items-center gap-4">
            <FaFileAlt className="text-purple-500 w-8 h-8" />
            <div>
              <h3 className="text-lg font-semibold text-gray-500 dark:text-gray-400">Файлы</h3>
              <p className="text-3xl font-bold">{files.length}</p>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow">
          <div className="flex justify-between items-center mb-3">
            <h3 className="text-xl font-semibold">Клиенты</h3>
            <Link to="/clients" className="text-blue-600 hover:underline text-sm">Все клиенты</Link>
          </div>
          {clients.slice(0, 5).map(client => (
            <div key={client.id} className="flex items-center justify-between py-2 border-b dark:border-gray-700 last:border-0">
              <span className="flex items-center gap-2">
                <span className={`inline-block w-2 h-2 rounded-full ${client.online ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`}></span>
                {client.hostname}
              </span>
              <span className="text-sm text-gray-500">{client.os}</span>
            </div>
          ))}
          {clients.length === 0 && <p className="text-gray-500">Нет клиентов</p>}
        </div>

        <div className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow">
          <div className="flex justify-between items-center mb-3">
            <h3 className="text-xl font-semibold">Файлы</h3>
            <Link to="/install" className="text-blue-600 hover:underline text-sm">Загрузить</Link>
          </div>
          {files.slice(0, 5).map(file => (
            <div key={file.id} className="flex items-center justify-between py-2 border-b dark:border-gray-700 last:border-0">
              <span className="flex items-center gap-2">
                <FaFileAlt className="text-blue-400" />
                {file.name}
              </span>
              <span className="text-sm text-gray-500">{(file.size / 1024).toFixed(1)} KB</span>
            </div>
          ))}
          {files.length === 0 && <p className="text-gray-500">Файлы не загружены</p>}
        </div>
      </div>
    </div>
  );
};

export default Dashboard;