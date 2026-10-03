import { useEffect, useState } from 'react';
import { fetchInitialDevices } from '../services/api';

export const useFleetState = (token, onLogout) => {
  const [devices, setDevices] = useState([]);
  const [connectionStatus, setConnectionStatus] = useState('disconnected');
  const [commandCompletions, setCommandCompletions] = useState({});

  useEffect(() => {
    if (!token) return;
    setConnectionStatus('connecting');
    fetchInitialDevices(token).then(data => setDevices(data || [])).catch(onLogout);
    const apiUrl = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/$/, '');
    const ws = new WebSocket(`${apiUrl.replace(/^http/, 'ws')}/admin/ws?token=${token}`);
    ws.onopen = () => setConnectionStatus('connected');
    ws.onerror = () => setConnectionStatus('disconnected');
    ws.onclose = () => setConnectionStatus('disconnected');
    ws.onmessage = event => {
      const data = JSON.parse(event.data);
      if (data.event === 'device_update') {
        setDevices(previous => previous.map(device => device.id === data.device_id ? { ...device, status: data.status || device.status, battery: data.battery ?? device.battery, last_seen: new Date().toISOString() } : device));
      } else if (data.event === 'command_completed') {
        setCommandCompletions(previous => ({ ...previous, [data.device_id]: data.command_id }));
      }
    };
    return () => ws.close();
  }, [token, onLogout]);

  return { devices, connectionStatus, commandCompletions };
};
