import React, { useState, useEffect } from 'react';
import { getClients, install, uploadFile, getFiles, deleteFile } from '../api';
import LoadingSpinner from '../components/common/LoadingSpinner';
import { FaUpload, FaTrash, FaFile } from 'react-icons/fa';

const Install = () => {
  const [clients, setClients] = useState([]);
  const [files, setFiles] = useState([]);
  const [selectedClients, setSelectedClients] = useState([]);
  const [selectedFiles, setSelectedFiles] = useState([]);
  const [mode, setMode] = useState('quiet');
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);

  // Загрузка клиентов и файлов
  const loadData = async () => {
    try {
      const clientsData = await getClients();
      setClients(Array.isArray(clientsData) ? clientsData : []);
    } catch (err) {
      console.error('Failed to load clients', err);
    }
    try {
      const filesData = await getFiles();
      setFiles(Array.isArray(filesData) ? filesData : []);
    } catch (err) {
      console.error('Failed to load files', err);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleFileUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    setUploading(true);
    try {
      await uploadFile(file);
      await loadData(); // обновить список
    } catch (err) {
      alert('Ошибка загрузки: ' + err.message);
    } finally {
      setUploading(false);
      e.target.value = ''; // сброс input
    }
  };

  const handleDeleteFile = async (fileId) => {
    if (!confirm('Удалить файл?')) return;
    try {
      await deleteFile(fileId);
      await loadData();
    } catch (err) {
      alert('Ошибка удаления: ' + err.message);
    }
  };

  const toggleClient = (id) => {
    setSelectedClients(prev =>
      prev.includes(id) ? prev.filter(c => c !== id) : [...prev, id]
    );
  };

  const toggleFile = (id) => {
    setSelectedFiles(prev =>
      prev.includes(id) ? prev.filter(f => f !== id) : [...prev, id]
    );
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (selectedClients.length === 0 || selectedFiles.length === 0) {
      alert('Выберите клиентов и файлы');
      return;
    }
    setLoading(true);
    try {
      const result = await install(selectedFiles, selectedClients, mode);
      alert(`Задание создано (ID: ${result.job_id})`);
      setSelectedClients([]);
      setSelectedFiles([]);
    } catch (err) {
      alert('Ошибка: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <LoadingSpinner />;

  return (
    <div className="max-w-6xl mx-auto">
      <h2 className="text-3xl font-bold mb-6">Установка ПО</h2>
      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Блок: Файлы */}
        <div className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow">
          <div className="flex justify-between items-center mb-3">
            <h3 className="text-xl font-semibold">1. Выберите файлы</h3>
            <label className="cursor-pointer bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded flex items-center gap-2">
              <FaUpload />
              {uploading ? 'Загрузка...' : 'Загрузить'}
              <input
                type="file"
                className="hidden"
                onChange={handleFileUpload}
                disabled={uploading}
              />
            </label>
          </div>
          <div className="max-h-60 overflow-y-auto grid grid-cols-1 md:grid-cols-2 gap-2">
            {files.map(f => (
              <div key={f.id} className="flex items-center justify-between p-2 border rounded hover:bg-gray-50 dark:hover:bg-gray-700">
                <label className="flex items-center space-x-2 cursor-pointer flex-1">
                  <input
                    type="checkbox"
                    checked={selectedFiles.includes(f.id)}
                    onChange={() => toggleFile(f.id)}
                    className="form-checkbox h-5 w-5 text-blue-600"
                  />
                  <FaFile className="text-blue-500" />
                  <span className="text-sm truncate">{f.name}</span>
                  <span className="text-xs text-gray-400 ml-auto">{(f.size / 1024).toFixed(1)} KB</span>
                </label>
                <button
                  type="button"
                  onClick={() => handleDeleteFile(f.id)}
                  className="text-red-500 hover:text-red-700 ml-2"
                >
                  <FaTrash />
                </button>
              </div>
            ))}
            {files.length === 0 && <p className="text-gray-500 col-span-2">Нет загруженных файлов</p>}
          </div>
        </div>

        {/* Блок: Клиенты */}
        <div className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow">
          <h3 className="text-xl font-semibold mb-3">2. Выберите клиентов</h3>
          <div className="max-h-64 overflow-y-auto grid grid-cols-2 md:grid-cols-3 gap-2">
            {clients.map(c => (
              <label key={c.id} className={`flex items-center space-x-2 cursor-pointer p-2 rounded border ${selectedClients.includes(c.id) ? 'border-blue-500 bg-blue-50 dark:bg-blue-900' : 'border-transparent'}`}>
                <input
                  type="checkbox"
                  checked={selectedClients.includes(c.id)}
                  onChange={() => toggleClient(c.id)}
                  className="form-checkbox h-5 w-5 text-blue-600"
                />
                <span className="text-sm">
                  {c.hostname}
                  <span className={`ml-2 text-xs ${c.online ? 'text-green-600' : 'text-red-600'}`}>
                    {c.online ? '●' : '○'}
                  </span>
                </span>
              </label>
            ))}
          </div>
        </div>

        {/* Блок: Режим */}
        <div className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow flex items-center gap-6">
          <span className="text-xl font-semibold">3. Режим установки</span>
          <label className="flex items-center space-x-2">
            <input
              type="radio"
              value="quiet"
              checked={mode === 'quiet'}
              onChange={() => setMode('quiet')}
              className="form-radio h-4 w-4 text-blue-600"
            />
            <span>Тихая</span>
          </label>
          <label className="flex items-center space-x-2">
            <input
              type="radio"
              value="interactive"
              checked={mode === 'interactive'}
              onChange={() => setMode('interactive')}
              className="form-radio h-4 w-4 text-blue-600"
            />
            <span>Интерактивная</span>
          </label>
        </div>

        <button
          type="submit"
          disabled={loading || selectedClients.length === 0 || selectedFiles.length === 0}
          className="w-full bg-green-600 text-white py-3 rounded-lg hover:bg-green-700 disabled:opacity-50 text-xl"
        >
          {loading ? 'Отправка...' : '🚀 Установить'}
        </button>
      </form>
    </div>
  );
};

export default Install;