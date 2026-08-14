import { fmtDuration, fmtDateTime, timeAgo } from '../utils/format';
import './OutagesSection.css';

export default function MaintenanceLogs({ logs }) {
    return (
        <div className="incidents-card fade-in">
            <div className="card-header">
                <div className="card-title">Tizim yangilanishlari (Restart)</div>
                <div className="date-range">Oxirgi hodisalar</div>
            </div>

            {!logs || logs.length === 0 ? (
                <div className="incidents-empty" style={{ padding: 24 }}>
                    <div>Hozircha tizim yangilanishlari kuzatilmadi</div>
                </div>
            ) : (
                logs.map((log, i) => {
                    const dur = log.duration_seconds != null ? fmtDuration(log.duration_seconds) : '';
                    const startFmt = fmtDateTime(log.started_at);
                    const timeAgoStr = timeAgo(log.started_at);

                    let icon = '🔄';
                    if (log.event_type === 'start') icon = '▶️';
                    else if (log.event_type === 'stop') icon = '⏹️';

                    let title = log.service_name || 'Tizim';
                    if (title.endsWith('.service')) title = title.replace('.service', '');

                    let badgeClass = 'up';
                    if (log.event_type === 'stop') badgeClass = 'down';

                    return (
                        <div className="incident-row" key={i}>
                            <div className="incident-icon">{icon}</div>
                            <div className="incident-body">
                                <div className="incident-service">
                                    {title}
                                    <span
                                        className={`proj-badge ${badgeClass}`}
                                        style={{ fontSize: 9, padding: '1px 5px' }}
                                    >
                                        {log.event_type || 'restart'}
                                    </span>
                                    <span
                                        style={{
                                            fontSize: 11,
                                            color: 'var(--text-muted)',
                                            marginLeft: 6,
                                            fontWeight: 'normal',
                                        }}
                                    >
                                        {timeAgoStr}
                                    </span>
                                </div>
                                <div className="incident-time">
                                    {startFmt} - {log.description || 'Qayta ishga tushirildi'}
                                </div>
                            </div>
                            <div className="incident-duration">{dur}</div>
                        </div>
                    );
                })
            )}
        </div>
    );
}
