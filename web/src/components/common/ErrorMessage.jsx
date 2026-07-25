import React from 'react';

const ErrorMessage = ({ message }) => (
  <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative" role="alert">
    <strong className="font-bold">Ошибка!</strong>
    <span className="block sm:inline"> {message}</span>
  </div>
);

export default ErrorMessage;