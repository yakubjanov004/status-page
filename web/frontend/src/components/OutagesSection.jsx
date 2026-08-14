import { fmtDuration, fmtDateTime } from '../utils/format';
import './OutagesSection.css';

export default function OutagesSection({ outages }) {
    return (
        <div className="incidents-card fade-in">
            <div className="card-header">
                <div className="card-title">So'nggi muammolar</div>
                <div className="date-range">So'nggi 7 kun</div>
            </div>

            {!outages || outages.length === 0 ? (
                <div className="incidents-empty">
                    <div className="ei-icon">🎉</div>
                    <div>So'nggi 7 kunda muammo kuzatilmadi</div>
                    <div style={{ marginTop: 6, fontSize: 12, color: 'var(--text-faint)' }}>
                        Barcha tizimlar barqaror ishlamoqda
                    </div>
                </div>
            ) : (
                outages.map((o, i) => {
                    const dur = fmtDuration(o.duration_seconds);
                    const startFmt = fmtDateTime(o.start);
                    const endFmt = o.is_ongoing ? 'Davom etmoqda' : fmtDateTime(o.end);

                    return (
                        <div className="incident-row" key={i}>
                            <div className="incident-icon">
                                {o.is_ongoing ? '🔴' : '⚠️'}
                            </div>
                            <div className="incident-body">
                                <div className="incident-service">
                                    {o.service_name}
                                    {o.is_ongoing && (
                                        <span className="ongoing-badge">Davom etmoqda</span>
                                    )}
                                </div>
                                <div className="incident-time">
                                    {startFmt} → {endFmt}
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
