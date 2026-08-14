import { useState } from 'react';
import { fmtDuration, fmtDateTime, timeAgo } from '../utils/format';
import './OutagesSection.css';
import './EventsTabs.css';

export default function EventsTabs({ outages, maintenanceLogs }) {
    const [activeTab, setActiveTab] = useState('outages');

    const outageCount = outages?.length || 0;
    const logCount = maintenanceLogs?.length || 0;

    return (
        <div className="incidents-card fade-in">
            <div className="events-tabs-header">
                <button
                    className={`events-tab ${activeTab === 'outages' ? 'active' : ''}`}
                    onClick={() => setActiveTab('outages')}
                >
                    ⚠️ So'nggi muammolar
                    {outageCount > 0 && <span className="tab-count down">{outageCount}</span>}
                </button>
                <button
                    className={`events-tab ${activeTab === 'maintenance' ? 'active' : ''}`}
                    onClick={() => setActiveTab('maintenance')}
                >
                    🔄 Tizim yangilanishlari
                    {logCount > 0 && <span className="tab-count">{logCount}</span>}
                </button>
            </div>

            {activeTab === 'outages' && <OutagesContent outages={outages} />}
            {activeTab === 'maintenance' && <MaintenanceContent logs={maintenanceLogs} />}
        </div>
    );
}

function OutagesContent({ outages }) {
    if (!outages || outages.length === 0) {
        return (
            <div className="incidents-empty">
                <div className="ei-icon">🎉</div>
                <div>So'nggi 7 kunda muammo kuzatilmadi</div>
                <div style={{ marginTop: 6, fontSize: 12, color: 'var(--text-faint)' }}>
                    Barcha tizimlar barqaror ishlamoqda
                </div>
            </div>
        );
    }

    return outages.map((o, i) => {
        const dur = fmtDuration(o.duration_seconds);
        const startFmt = fmtDateTime(o.start);
        const endFmt = o.is_ongoing ? 'Davom etmoqda' : fmtDateTime(o.end);

        return (
            <div className="incident-row" key={i}>
                <div className="incident-icon">{o.is_ongoing ? '🔴' : '⚠️'}</div>
                <div className="incident-body">
                    <div className="incident-service">
                        {o.service_name}
                        {o.is_ongoing && <span className="ongoing-badge">Davom etmoqda</span>}
                    </div>
                    <div className="incident-time">{startFmt} → {endFmt}</div>
                </div>
                <div className="incident-duration">{dur}</div>
            </div>
        );
    });
}

function MaintenanceContent({ logs }) {
    if (!logs || logs.length === 0) {
        return (
            <div className="incidents-empty" style={{ padding: 24 }}>
                <div>Hozircha tizim yangilanishlari kuzatilmadi</div>
            </div>
        );
    }

    return logs.map((log, i) => {
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
                        <span style={{ fontSize: 11, color: 'var(--text-muted)', marginLeft: 6, fontWeight: 'normal' }}>
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
    });
}
