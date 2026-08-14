/**
 * Vaqtni formatlash yordamchi funksiyalari
 */

export function fmtDuration(secs) {
    if (!secs || secs <= 0) return '< 1 daq';
    const h = Math.floor(secs / 3600);
    const m = Math.floor((secs % 3600) / 60);
    if (h === 0 && m === 0) return '< 1 daq';
    let s = '';
    if (h > 0) s += `${h}s `;
    if (m > 0 || h === 0) s += `${m}d`;
    return s.trim();
}

export function fmtTime(isoStr) {
    if (!isoStr) return '—';
    try {
        return new Date(isoStr).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });
    } catch { return isoStr; }
}

export function fmtDateTime(isoStr) {
    if (!isoStr) return '—';
    try {
        const d = new Date(isoStr);
        return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch { return isoStr; }
}

export function fmtDate(dateStr) {
    try {
        return new Date(dateStr + 'T00:00:00').toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
    } catch { return dateStr; }
}

export function timeAgo(isoStr) {
    if (!isoStr) return 'hozirgina';
    try {
        const diff = Math.floor((Date.now() - new Date(isoStr)) / 1000);
        if (diff < 60) return 'hozirgina';
        if (diff < 3600) return `${Math.floor(diff / 60)} daqiqa oldin`;
        if (diff < 86400) return `${Math.floor(diff / 3600)} soat oldin`;
        return fmtDateTime(isoStr);
    } catch { return 'hozirgina'; }
}

export function escHtml(str) {
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}
