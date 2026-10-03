export default function DashboardHeader({ connectionStatus, onLogout }) {
  return <header className="topbar">
    <div className="brand"><img src="/mdm-matrix-logo.png" alt="MDM Matrix" /><div><small>MDM MATRIX</small><h1>Fleet command center</h1></div></div>
    <div className="top-actions">
      <span className={`live ${connectionStatus}`}><i />{connectionStatus === 'connected' ? 'Live' : connectionStatus}</span>
      <button onClick={onLogout}>Sign out</button>
    </div>
  </header>;
}
