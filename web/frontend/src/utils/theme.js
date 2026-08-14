/**
 * Theme management — localStorage asosida
 */

const STORAGE_KEY = 'theme';

export function getInitialTheme() {
    return localStorage.getItem(STORAGE_KEY) || 'light';
}

export function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem(STORAGE_KEY, theme);
}

export function toggleTheme() {
    const current = document.documentElement.getAttribute('data-theme');
    const next = current === 'dark' ? 'light' : 'dark';
    applyTheme(next);
    return next;
}
