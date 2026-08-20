const BASE_URL = 'http://localhost:8080';

export const loginAdmin = async (username, password) => {
  const res = await fetch(`${BASE_URL}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password })
  });
  if (!res.ok) throw new Error('Invalid credentials');
  return res.json();
};

export const fetchInitialDevices = async (token) => {
  const res = await fetch(`${BASE_URL}/devices`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  if (!res.ok) throw new Error('Unauthorized');
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
  if (!res.ok) throw new Error('Failed to dispatch');
  return res.json();
};