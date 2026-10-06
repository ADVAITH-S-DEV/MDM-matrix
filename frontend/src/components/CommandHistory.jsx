const labels = { update_policy: 'Policy update', lock: 'Lock', unlock: 'Unlock', wipe: 'Wipe' };

export default function CommandHistory({ commands, metrics, loading, error, onRefresh }) {
  const completionRate = metrics?.total ? Math.round((metrics.completed / metrics.total) * 100) : 0;
  const averageSeconds = metrics ? (metrics.average_completion_ms / 1000).toFixed(1) : '0.0';

  return <section className="command-panel">
    <div className="section-title">
      <div><small>PROCESS VISIBILITY</small><h2>Command activity</h2></div>
      <button className="refresh-button" onClick={onRefresh} disabled={loading}>{loading ? 'Refreshing...' : 'Refresh'}</button>
    </div>
    <div className="process-metrics">
      <span><b>{metrics?.completed ?? 0}</b> completed</span>
      <span><b>{metrics?.active ?? 0}</b> active</span>
      <span><b>{completionRate}%</b> completion rate</span>
      <span><b>{averageSeconds}s</b> average cycle time</span>
    </div>
    {error && <p className="insights-error">{error}</p>}
    <div className="history-list">
      {commands.map(command => <article key={command.id}>
        <div><strong>{labels[command.type] || command.type}</strong><code>{command.device_id}</code></div>
        <span className={`command-status ${command.status}`}>{command.status}</span>
        <time>{new Date(command.created_at).toLocaleString()}</time>
      </article>)}
      {!commands.length && <p className="empty">No commands have been dispatched yet.</p>}
    </div>
  </section>;
}
