export default function DeviceTable({ devices, onCommand }) {
  return (
    <div className="table-container">
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Name</th>
            <th>Status</th>
            <th>Battery</th>
            <th>Last Seen</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {devices.map(dev => (
            <tr key={dev.id}>
              <td>{dev.id}</td>
              <td>{dev.name}</td>
              <td><span className={`status-badge ${dev.status}`}>{dev.status}</span></td>
              <td>{dev.battery}%</td>
              <td>{new Date(dev.last_seen).toLocaleTimeString()}</td>
              <td className="actions">
                <button onClick={() => onCommand(dev.id, 'lock')}>Lock</button>
                <button onClick={() => onCommand(dev.id, 'wipe')} className="danger">Wipe</button>
                <button onClick={() => onCommand(dev.id, 'update_policy')}>Policy Update</button>
              </td>
            </tr>
          ))}
          {devices.length === 0 && (
            <tr><td colSpan="6" style={{ textAlign: 'center' }}>No devices enrolled.</td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}