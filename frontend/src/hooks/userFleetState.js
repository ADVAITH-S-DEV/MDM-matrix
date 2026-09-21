import { useState, useEffect } from 'react';
import { fetchInitialDevices } from '../services/api';

export const useFleetState = (token, onLogout) => {
  const [devices, setDevices] = useState([]);

  useEffect(() => {
    if (!token) return;

    // Load initial state
    fetchInitialDevices(token)
      .then(data => setDevices(data || []))
      .catch(() => onLogout());

    // Derive WebSocket URL from environment variable, fallback to localhost
    const apiUrl = import.meta.env.VITE_API_URL || "http://localhost:8080";
    const wsUrl = apiUrl.replace(/^http/, 'ws');
    const ws = new WebSocket(`${wsUrl}/admin/ws?token=${token}`);
    
    ws.onopen = () => console.log("🟢 Connected to Live Admin Hub");
    
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      
      if (data.event === "device_update") {
        setDevices(prev => prev.map(dev => 
          dev.id === data.device_id ? { 
            ...dev, 
            status: data.status || dev.status, 
            battery: data.battery !== undefined ? data.battery : dev.battery,
            last_seen: new Date().toISOString() 
          } : dev
        ));
      }
    };

    return () => {
      ws.close();
      console.log("🔴 Disconnected from Admin Hub");
    };
  }, [token, onLogout]);

  return { devices };
};