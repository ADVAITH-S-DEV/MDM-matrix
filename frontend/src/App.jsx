import { useState } from 'react';
import { loginAdmin, sendDeviceCommand } from './services/api';
import { useFleetState } from './hooks/useFleetState';
import LoginForm from './components/LoginForm';
import DeviceTable from './components/DeviceTable';
import './App.css';

function App() {
  const [token, setToken] = useState(localStorage.getItem('mdm_token') || '');
  const [error, setError] = useState('');
  
  const handleLogout = () => {
    setToken('');
    localStorage.removeItem('mdm_token');
  };

  const { devices } = useFleetState(token, handleLogout);

  const handleLogin = async (username, password) => {
    setError('');
    try {
      const data = await loginAdmin(username, password);
      setToken(data.token);
      localStorage.setItem('mdm_token', data.token);
    } catch (err) {
      setError(err.message);
    }
  };

  const handleCommand = async (deviceId, type) => {
    try {
      await sendDeviceCommand(deviceId, type, token);
    } catch (err) {
      console.error("Command failed", err);
    }
  };

  if (!token) {
    return <LoginForm onLogin={handleLogin} error={error} />;
  }

  return (
    <div className="dashboard">
      <header>
        <h1>MDM Admin Dashboard</h1>
        <button className="logout-btn" onClick={handleLogout}>Logout</button>
      </header>
      <DeviceTable devices={devices} onCommand={handleCommand} />
    </div>
  );
}

export default App;