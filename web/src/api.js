const getApiBase = () => {
  return localStorage.getItem('api_url') || '/api/v1';
};

export async function getClients() {
  try {
    const res = await fetch(`${getApiBase()}/clients`);
    if (!res.ok) throw new Error('Failed to fetch clients');
    const data = await res.json();
    return Array.isArray(data) ? data : [];
  } catch (err) {
    console.error('getClients error:', err);
    return [];
  }
}

export async function install(fileIds, clientIds, mode = 'quiet') {
  const res = await fetch(`${getApiBase()}/install`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ file_ids: fileIds, client_ids: clientIds, mode }),
  });
  if (!res.ok) throw new Error('Install failed');
  return res.json();
}

export async function uploadFile(file) {
  const formData = new FormData();
  formData.append('file', file);
  const res = await fetch(`/api/v1/upload`, { // пока оставляем фиксированным, но можно тоже сделать из localStorage
    method: 'POST',
    body: formData,
  });
  if (!res.ok) throw new Error('Upload failed');
  return res.json();
}

export async function getClientDevices(clientId) {
  const res = await fetch(`${getApiBase()}/clients/${clientId}/devices`);
  if (!res.ok) throw new Error('Failed to fetch devices');
  return res.json();
}

export async function getClientStats(clientId) {
  const res = await fetch(`${getApiBase()}/clients/${clientId}/stats`);
  if (!res.ok) throw new Error('Failed to fetch stats');
  return res.json();
}

export async function wakeOnLan(clientIds) {
  const res = await fetch(`${getApiBase()}/wol`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ client_ids: clientIds }),
  });
  if (!res.ok) throw new Error('WOL failed');
  return res.json();
}

export async function parseDriver(url) {
  const res = await fetch(`${getApiBase()}/parse-driver`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url }),
  });
  if (!res.ok) throw new Error('Parse failed');
  return res.json();
}

export async function getFiles() {
  const res = await fetch(`${getApiBase()}/files`);
  if (!res.ok) throw new Error('Failed to fetch files');
  return res.json();
}

// Удалить файл по ID
export async function deleteFile(fileId) {
  const res = await fetch(`${getApiBase()}/files/${fileId}`, {
    method: 'DELETE',
  });
  if (!res.ok) throw new Error('Failed to delete file');
  return res.json();
}