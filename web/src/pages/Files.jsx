import React, { useState, useEffect } from 'react';
import FileUploader from '../components/FileUploader';
import { FaTrash } from 'react-icons/fa';
import { getFiles, deleteFile } from '../api';

const Files = () => {
  const [files, setFiles] = useState([]);
  const [loading, setLoading] = useState(true);

  const loadFiles = async () => {
    try {
      const data = await getFiles();
      setFiles(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to load files', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadFiles();
  }, []);

  const handleUpload = () => {
    loadFiles();
  };

  const handleDelete = async (fileId) => {
    if (!confirm('Удалить файл?')) return;
    try {
      await deleteFile(fileId);
      await loadFiles();
    } catch (err) {
      alert('Ошибка удаления: ' + err.message);
    }
  };

  if (loading) return <div className="text-center py-8">Загрузка...</div>;

  return (
    <div>
      <h2 className="text-3xl font-bold mb-6">Библиотека файлов</h2>
      <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow mb-6">
        <FileUploader onFileUploaded={handleUpload} />
      </div>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-700">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Название</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Тип</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Размер</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Дата загрузки</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Версия</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Описание</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Действия</th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {files.map((file) => (
                <tr key={file.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                  <td className="px-4 py-3">{file.name}</td>
                  <td className="px-4 py-3">{file.type || '-'}</td>
                  <td className="px-4 py-3">{(file.size / 1024).toFixed(1)} KB</td>
                  <td className="px-4 py-3">{file.uploadedAt ? new Date(file.uploadedAt).toLocaleString() : '-'}</td>
                  <td className="px-4 py-3">{file.version || '-'}</td>
                  <td className="px-4 py-3">{file.description || '-'}</td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => handleDelete(file.id)}
                      className="text-red-500 hover:text-red-700 transition"
                    >
                      <FaTrash />
                    </button>
                  </td>
                </tr>
              ))}
              {files.length === 0 && (
                <tr>
                  <td colSpan="7" className="text-center py-6 text-gray-500">Нет загруженных файлов</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default Files;