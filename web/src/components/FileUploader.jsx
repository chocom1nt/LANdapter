import React, { useState, useRef } from 'react';
import { uploadFile } from '../api';

const FileUploader = ({ onFileUploaded }) => {
  const [dragging, setDragging] = useState(false);
  const [loading, setLoading] = useState(false);
  const fileInputRef = useRef(null);

  const handleFile = async (file) => {
    setLoading(true);
    try {
      const result = await uploadFile(file);
      onFileUploaded(result.file_id, result.name);
    } catch (err) {
      alert('Ошибка загрузки: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  const onDrop = (e) => {
    e.preventDefault();
    setDragging(false);
    if (e.dataTransfer.files.length) {
      handleFile(e.dataTransfer.files[0]);
    }
  };

  return (
    <div
      className={`border-2 border-dashed rounded p-4 text-center transition-colors ${
        dragging ? 'border-blue-500 bg-blue-50 dark:bg-blue-900' : 'border-gray-300 dark:border-gray-600'
      }`}
      onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
      onDragLeave={() => setDragging(false)}
      onDrop={onDrop}
      onClick={() => fileInputRef.current?.click()}
    >
      {loading ? (
        <span>Загрузка...</span>
      ) : (
        <>
          <p className="text-gray-600 dark:text-gray-300">
            Перетащите файл сюда или кликните для выбора
          </p>
          <input
            type="file"
            ref={fileInputRef}
            className="hidden"
            onChange={(e) => e.target.files && handleFile(e.target.files[0])}
          />
        </>
      )}
    </div>
  );
};

export default FileUploader;