export default function FleetStats({ summary }) {
  return <section className="stats" aria-label="Fleet summary">
    <article><span>Total devices</span><strong>{summary.total}</strong><small>Enrolled endpoints</small></article>
    <article><span>Online now</span><strong className="green">{summary.online}</strong><small>Sending heartbeats</small></article>
    <article><span>Needs attention</span><strong className="orange">{summary.offline}</strong><small>Currently offline</small></article>
    <article><span>Active commands</span><strong>{summary.active}</strong><small>Pending or queued</small></article>
  </section>;
}
