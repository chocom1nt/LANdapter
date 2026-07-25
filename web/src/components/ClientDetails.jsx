import React from 'react';

const ClientDetails = ({ devices, stats, loading }) => {
  if (loading) return <div className="text-gray-500 animate-pulse">Загрузка...</div>;

  // Парсинг устройств
  let parsedDevices = null;
  if (typeof devices === 'string') {
    try { parsedDevices = JSON.parse(devices); } catch {}
  } else if (Array.isArray(devices)) {
    parsedDevices = devices;
  }

  // Парсинг статистики
  let parsedStats = null;
  if (typeof stats === 'string') {
    try { parsedStats = JSON.parse(stats); } catch {}
  } else if (stats && typeof stats === 'object') {
    parsedStats = stats;
  }

  const hasDevices = parsedDevices && Array.isArray(parsedDevices) && parsedDevices.length > 0;
  const hasStats = parsedStats && typeof parsedStats === 'object';

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      {/* Блок устройств */}
      <div>
        <h4 className="font-semibold text-lg mb-3 text-gray-700 dark:text-gray-300">Устройства</h4>
        {hasDevices ? (
          <div className="overflow-x-auto bg-white dark:bg-gray-800 rounded-lg shadow-sm border dark:border-gray-700">
            <table className="min-w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-700">
                <tr>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Название</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Класс</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Статус</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-600">
                {parsedDevices.map((d, idx) => (
                  <tr key={idx} className="hover:bg-gray-50 dark:hover:bg-gray-700 transition">
                    <td className="px-4 py-2">{d.FriendlyName || d.friendlyName || '-'}</td>
                    <td className="px-4 py-2">{d.Class || d.class || '-'}</td>
                    <td className="px-4 py-2">
                      <span className={`px-2 py-0.5 rounded-full text-xs ${
                        (d.Status || d.status) === 'OK' || (d.Status || d.status) === 'Ok'
                          ? 'bg-green-100 text-green-800'
                          : 'bg-yellow-100 text-yellow-800'
                      }`}>
                        {d.Status || d.status || '-'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-gray-500">Нет данных об устройствах</div>
        )}
      </div>

      {/* Блок статистики */}
      <div>
        <h4 className="font-semibold text-lg mb-3 text-gray-700 dark:text-gray-300">Системная статистика</h4>
        {hasStats ? (
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border dark:border-gray-700 p-4 space-y-3">
            <div className="flex justify-between items-center border-b dark:border-gray-600 pb-2">
              <span className="text-gray-600 dark:text-gray-400">Загрузка CPU</span>
              <span className="font-mono font-medium">
                {parsedStats.cpu_percent !== undefined ? parsedStats.cpu_percent + '%' : '-'}
              </span>
            </div>
            <div className="flex justify-between items-center border-b dark:border-gray-600 pb-2">
              <span className="text-gray-600 dark:text-gray-400">Оперативная память</span>
              <span className="font-mono font-medium">
                {parsedStats.mem_available_mb !== undefined && parsedStats.mem_total_mb !== undefined
                  ? `${Math.round(parsedStats.mem_available_mb)} / ${Math.round(parsedStats.mem_total_mb)} MB`
                  : parsedStats.mem_used_mb && parsedStats.mem_total_mb
                  ? `${Math.round(parsedStats.mem_used_mb)} / ${Math.round(parsedStats.mem_total_mb)} MB`
                  : '-'}
              </span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-600 dark:text-gray-400">Время работы</span>
              <span className="font-mono font-medium">
                {parsedStats.uptime_seconds ? formatUptime(parsedStats.uptime_seconds) : parsedStats.uptime_human || '-'}
              </span>
            </div>
          </div>
        ) : (
          <div className="text-gray-500">Нет данных статистики</div>
        )}
      </div>
    </div>
  );
};

function formatUptime(seconds) {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const parts = [];
  if (days) parts.push(`${days}д`);
  if (hours) parts.push(`${hours}ч`);
  if (minutes) parts.push(`${minutes}м`);
  return parts.length ? parts.join(' ') : '< 1 мин';
}

export default ClientDetails;