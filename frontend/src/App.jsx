import { useCallback, useEffect, useMemo, useState } from 'react';
import { loginAdmin, sendDeviceCommand } from './services/api';
import { useFleetState } from './hooks/fleet.js';
import LoginForm from './components/LoginForm';
import DeviceTable from './components/DeviceTable';
import './App.css';

function App() {
  const [token, setToken] = useState(localStorage.getItem('mdm_token') || '');
  const [error, setError] = useState('');
  const [loggingIn, setLoggingIn] = useState(false);
  const [commands, setCommands] = useState({});
  const logout = useCallback(() => { setToken(''); setCommands({}); localStorage.removeItem('mdm_token'); }, []);
  const { devices, connectionStatus, commandCompletions } = useFleetState(token, logout);

  useEffect(() => setCommands(previous => {
    const next = { ...previous };
    let changed = false;
    Object.entries(commandCompletions).forEach(([deviceId, commandId]) => {
      if (next[deviceId]?.commandId === commandId && next[deviceId].status !== 'completed') {
        next[deviceId] = { ...next[deviceId], status: 'completed', message: `${next[deviceId].label} completed` };
        changed = true;
      }
    });
    return changed ? next : previous;
  }), [commandCompletions]);

  const summary = useMemo(() => ({
    total: devices.length,
    online: devices.filter(device => device.status === 'online').length,
    offline: devices.filter(device => device.status !== 'online').length,
    active: Object.values(commands).filter(command => ['dispatching', 'pending', 'queued'].includes(command.status)).length
  }), [devices, commands]);

  const login = async (username, password) => {
    setError(''); setLoggingIn(true);
    try { const data = await loginAdmin(username, password); setToken(data.token); localStorage.setItem('mdm_token', data.token); }
    catch (loginError) { setError(loginError.message); }
    finally { setLoggingIn(false); }
  };

  const sendCommand = async (deviceId, type) => {
    const label = { lock: 'Lock', wipe: 'Wipe', update_policy: 'Policy update' }[type];
    const online = devices.find(device => device.id === deviceId)?.status === 'online';
    setCommands(previous => ({ ...previous, [deviceId]: { type, label, status: 'dispatching', message: `Sending ${label.toLowerCase()}...` } }));
    try {
      const result = await sendDeviceCommand(deviceId, type, token);
      setCommands(previous => ({ ...previous, [deviceId]: { type, label, commandId: result.command_id, status: online ? 'pending' : 'queued', message: online ? `${label} sent — awaiting device` : `${label} queued until reconnect` } }));
    } catch (commandError) {
      setCommands(previous => ({ ...previous, [deviceId]: { type, label, status: 'failed', message: commandError.message } }));
    }
  };

  if (!token) return <LoginForm onLogin={login} error={error} loading={loggingIn} />;
  return <main className="dashboard">
    <header className="topbar"><div className="brand"><b>M</b><div><small>MDM MATRIX</small><h1>Fleet command center</h1></div></div><div className="top-actions"><span className={`live ${connectionStatus}`}><i />{connectionStatus === 'connected' ? 'Live' : connectionStatus}</span><button onClick={logout}>Sign out</button></div></header>
    <section className="hero"><div><small>REAL-TIME DEVICE OPERATIONS</small><h2>Your fleet, visible and actionable.</h2><p>Monitor simulated endpoints, dispatch commands, and follow acknowledgements as they happen.</p></div><b>M</b></section>
    <section className="stats"><article><span>Total devices</span><strong>{summary.total}</strong><small>Enrolled endpoints</small></article><article><span>Online now</span><strong className="green">{summary.online}</strong><small>Sending heartbeats</small></article><article><span>Needs attention</span><strong className="orange">{summary.offline}</strong><small>Currently offline</small></article><article><span>Active commands</span><strong>{summary.active}</strong><small>Pending or queued</small></article></section>
    <section className="fleet"><div className="section-title"><div><small>DEVICE INVENTORY</small><h2>Managed fleet</h2></div><span>{devices.length} devices</span></div><DeviceTable devices={devices} onCommand={sendCommand} commandStates={commands} /></section>
  </main>;
}

export default App;
