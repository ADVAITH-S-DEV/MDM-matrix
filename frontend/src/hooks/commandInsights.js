import { useCallback, useEffect, useState } from 'react';
import { fetchCommandMetrics, fetchCommands } from '../services/api';

export function useCommandInsights(token) {
  const [history, setHistory] = useState([]);
  const [metrics, setMetrics] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const refresh = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [commands, commandMetrics] = await Promise.all([
        fetchCommands(token),
        fetchCommandMetrics(token)
      ]);
      setHistory(commands || []);
      setMetrics(commandMetrics);
    } catch (requestError) {
      setError(requestError.message || 'Unable to load command insights');
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    const initialRefresh = window.setTimeout(refresh, 0);
    const interval = window.setInterval(refresh, 15000);
    return () => {
      window.clearTimeout(initialRefresh);
      window.clearInterval(interval);
    };
  }, [refresh]);

  return { history, metrics, loading, error, refresh };
}
