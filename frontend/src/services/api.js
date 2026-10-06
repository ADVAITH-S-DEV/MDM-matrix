const BASE_URL = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/$/, '');

const errorFromResponse = async (res, fallback) => {
  const message = (await res.text()).trim();
  return new Error(message || fallback);
};

export const loginAdmin = async (username, password) => {
  const res = await fetch(`${BASE_URL}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password })
  });
  if (!res.ok) throw await errorFromResponse(res, 'Login failed');
  return res.json();
};

export const fetchInitialDevices = async (token) => {
  const res = await fetch(`${BASE_URL}/devices`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  if (!res.ok) throw await errorFromResponse(res, 'Unable to load devices');
  return res.json();
};

export const sendDeviceCommand = async (deviceId, type, token) => {
  const res = await fetch(`${BASE_URL}/devices/${deviceId}/command`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({ type })
  });
  if (!res.ok) throw await errorFromResponse(res, 'Failed to dispatch');
  return res.json();
};

export const fetchCommands = async (token, limit = 20) => {
  const res = await fetch(`${BASE_URL}/commands?limit=${limit}`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  if (!res.ok) throw await errorFromResponse(res, 'Unable to load command history');
  return res.json();
};

export const fetchCommandMetrics = async (token) => {
  const res = await fetch(`${BASE_URL}/metrics/commands`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  if (!res.ok) throw await errorFromResponse(res, 'Unable to load command metrics');
  return res.json();
};
