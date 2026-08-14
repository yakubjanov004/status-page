/**
 * API layer — status va project endpointlari
 */

const API_BASE = '/api/public/status';

export async function fetchStatus() {
    const res = await fetch(`${API_BASE}?t=${Date.now()}`);
    if (!res.ok) throw new Error('HTTP ' + res.status);
    return res.json();
}

export async function fetchProjectStatus(slug, range = '7d') {
    const res = await fetch(`${API_BASE}/project/${slug}?range=${range}`);
    if (!res.ok) throw new Error('HTTP ' + res.status);
    return res.json();
}
