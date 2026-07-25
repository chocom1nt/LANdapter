import React, { useState, useEffect } from 'react';
import { getClients, getClientDevices, getClientStats } from '../api';
import LoadingSpinner from '../components/common/LoadingSpinner';
import ErrorMessage from '../components/common/ErrorMessage';
import ClientDetails from '../components/ClientDetails';

const Clients = () => {
  const [clients, setClients] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [expandedClient, setExpandedClient] = useState(null);
  const [devicesData, setDevicesData] = useState({});
  const [statsData, setStatsData] = useState({});
  const [detailsLoading, setDetailsLoading] = useState({});

  const fetchClients = async () => {
    try {
      const data = await getClients();
      setClients(data);
    } catch (err) {
      setError('Не удалось загрузить клиентов');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchClients();
    const interval = setInterval(fetchClients, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleExpand = async (clientId) => {
    if (expandedClient === clientId) {
      setExpandedClient(null);
      return;
    }
    setExpandedClient(clientId);
    setDetailsLoading(prev => ({ ...prev, [clientId]: true }));
    try {
      const [devices, stats] = await Promise.all([
        getClientDevices(clientId),
        getClientStats(clientId),
      ]);
      setDevicesData(prev => ({ ...prev, [clientId]: devices }));
      setStatsData(prev => ({ ...prev, [clientId]: stats }));
    } catch (err) {
      console.error('Failed to fetch details', err);
    } finally {
      setDetailsLoading(prev => ({ ...prev, [clientId]: false }));
    }
  };

  const getOsLabel = (os) => {
    if (os.includes('windows')) return 'Windows';
    if (os.includes('linux')) return 'Linux';
    if (os.includes('darwin') || os.includes('mac')) return 'macOS';
    return 'Unknown';
  };

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message={error} />;

  return (
    <div>
      <h2 className="text-3xl font-bold mb-6">Клиенты</h2>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-700">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Хост</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">ОС</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Статус</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">MAC</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Действия</th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {clients.map((client, index) => (
                <React.Fragment key={client.id}>
                  <tr className="hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors" style={{ animationDelay: `${index * 30}ms` }}>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <button
                        onClick={() => handleExpand(client.id)}
                        className="text-blue-600 dark:text-blue-400 hover:underline font-medium"
                      >
                        {client.hostname}
                      </button>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">{getOsLabel(client.os)}</td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium flex items-center gap-1 ${
                        client.online ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                      }`}>
                        <span className={`inline-block w-2 h-2 rounded-full ${client.online ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`}></span>
                        {client.online ? 'Онлайн' : 'Офлайн'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap font-mono text-sm">{client.mac || '—'}</td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <button
                        onClick={() => handleExpand(client.id)}
                        className="text-sm bg-blue-500 text-white px-3 py-1 rounded hover:bg-blue-600 transition"
                      >
                        {expandedClient === client.id ? 'Скрыть' : 'Подробности'}
                      </button>
                    </td>
                  </tr>
                  {expandedClient === client.id && (
                    <tr>
                      <td colSpan="5" className="px-6 py-4 bg-gray-50 dark:bg-gray-700">
                        <ClientDetails
                          devices={devicesData[client.id]}
                          stats={statsData[client.id]}
                          loading={detailsLoading[client.id]}
                        />
                      </td>
                    </tr>
                  )}
                </React.Fragment>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default Clients;