import { fmtDuration } from '../utils/format';
import UptimeBars from './UptimeBars';
import './ComponentRow.css';

export default function ComponentRow({ comp }) {
    const isUp = comp.is_up;
    const pct = typeof comp.uptime_pct === 'number' ? comp.uptime_pct : 100;
    const dotClass = isUp ? (pct < 100 ? 'warn' : 'up') : 'down';
    const statusText = isUp ? (pct < 99 ? 'Qisman uzilish' : 'Ishlamoqda') : 'Uzilish';
    const badgeClass = isUp ? '' : 'down';

    const downSecs = comp.total_downtime_secs || 0;
    const outCount = comp.total_outages || 0;
    const latency = comp.latency || 0;

    let metaStr = '';
    if (outCount > 0) {
        metaStr = `${outCount} uzilish, ${fmtDuration(downSecs)} ishlamadi`;
    } else {
        metaStr = "Belgilangan vaqt oralig'ida uzilishlar yo'q";
    }

    return (
        <div className="comp-row">
            <div className="comp-info">
                <div className="comp-left">
                    <div className={`status-dot ${dotClass}`}></div>
                    <div className="comp-name">{comp.name}</div>
                    <div className={`comp-status-badge ${badgeClass}`}>{statusText}</div>
                </div>
                <div className="comp-right">
                    {latency > 0 && <div className="latency-badge">{latency}ms</div>}
                    <div className="comp-meta">
                        <strong>{pct.toFixed(3)}%</strong> ishlash vaqti<br />
                        <span style={{ fontSize: '11px' }}>{metaStr}</span>
                    </div>
                </div>
            </div>
            <UptimeBars history={comp.history} />
        </div>
    );
}
