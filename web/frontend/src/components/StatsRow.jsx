import './StatsRow.css';

export default function StatsRow({ overallUptime, totalServices, activeIncidents }) {
    const upPct = typeof overallUptime === 'number' ? overallUptime.toFixed(3) : '—';

    return (
        <div className="stats-row fade-in">
            <div className="stat-card">
                <div className="stat-label">7-Kunlik Uptime</div>
                <div className="stat-value">{upPct}<span>%</span></div>
                <div className="stat-sub">Barcha xizmatlar bo'yicha umumiy</div>
            </div>
            <div className="stat-card">
                <div className="stat-label">Faol xizmatlar</div>
                <div className="stat-value">{totalServices || 0}</div>
                <div className="stat-sub">Kuzatilayotgan komponentlar</div>
            </div>
            <div className="stat-card">
                <div className="stat-label">Faol muammolar</div>
                <div
                    className="stat-value"
                    style={{ color: activeIncidents > 0 ? 'var(--down)' : 'var(--up)' }}
                >
                    {activeIncidents || 0}
                </div>
                <div className="stat-sub">
                    {activeIncidents === 0
                        ? 'Barcha tizimlar joyida'
                        : 'Hozirda ishlamayotgan xizmatlar'}
                </div>
            </div>
        </div>
    );
}
