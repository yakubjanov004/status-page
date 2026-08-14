import { fmtDate, fmtTime, fmtDuration } from '../utils/format';
import './UptimeBars.css';

export default function UptimeBars({ history }) {
    if (!history || history.length === 0) return null;

    return (
        <>
            <div className="bars-wrap">
                {history.map((day, i) => {
                    const p = typeof day.pct === 'number' ? day.pct : (day.is_up ? 100 : 0);
                    let barClass = 'c-none';
                    if (day.has_data === false) barClass = 'c-nodata';
                    else if (p >= 100)  barClass = 'c-100';
                    else if (p >= 90)   barClass = 'c-good';
                    else if (p >= 50)   barClass = 'c-warn';
                    else if (p > 0)     barClass = 'c-bad';
                    else if (!day.is_up) barClass = 'c-bad';

                    return (
                        <div className={`bar ${barClass}`} key={i}>
                            <Tooltip day={day} pct={p} />
                        </div>
                    );
                })}
            </div>
            <div className="bars-label">
                <span>{history.length} kun oldin</span>
                <span>Bugun</span>
            </div>
        </>
    );
}

function Tooltip({ day, pct }) {
    const dateFmt = fmtDate(day.date);

    if (day.has_data === false) {
        return (
            <div className="bar-tip">
                <div className="tip-date">{dateFmt}</div>
                <div className="tip-uptime">Monitoring boshlanmagan</div>
            </div>
        );
    }

    if (!day.outages || day.outages.length === 0) {
        return (
            <div className="bar-tip">
                <div className="tip-date">{dateFmt}</div>
                <div className="tip-uptime">
                    {pct >= 100 ? '100.000' : pct.toFixed(3)}% ishlash vaqti
                </div>
                <div className="tip-no-incidents">Bu kunda muammo bo'lmagan</div>
            </div>
        );
    }

    return (
        <div className="bar-tip">
            <div className="tip-date">{dateFmt}</div>
            <div className="tip-uptime">{pct.toFixed(3)}% ishlash vaqti</div>
            {day.outages.map((o, i) => (
                <div className="tip-outage" key={i}>
                    <div>
                        <div className="tip-outage-label">
                            ⚠ Uzilish &nbsp; {fmtDuration(o.duration_seconds)}
                        </div>
                        <div className="tip-outage-time">
                            {fmtTime(o.start)} – {fmtTime(o.end)}
                        </div>
                    </div>
                </div>
            ))}
        </div>
    );
}
