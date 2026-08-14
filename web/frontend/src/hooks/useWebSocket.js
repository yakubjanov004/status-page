import { useEffect, useRef, useCallback } from 'react';

/**
 * WebSocket orqali real-time yangilanishlarni olish.
 * WS message kelganda onUpdate callback chaqiriladi (debounce bilan).
 */
export function useWebSocket(onUpdate) {
    const wsRef = useRef(null);
    const debounceRef = useRef(null);
    const onUpdateRef = useRef(onUpdate);

    // Callback ref — stale closure oldini olish
    useEffect(() => {
        onUpdateRef.current = onUpdate;
    }, [onUpdate]);

    const connect = useCallback(() => {
        const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const ws = new WebSocket(`${proto}//${window.location.host}/ws`);

        ws.onopen = () => {
            console.log('[WS] Connected');
        };

        ws.onclose = () => {
            console.log('[WS] Disconnected, retrying in 5s…');
            setTimeout(connect, 5000);
        };

        ws.onmessage = () => {
            // Debounce — 2s interval'dan tez qayta fetch qilmaymiz
            if (debounceRef.current) return;
            debounceRef.current = setTimeout(() => {
                debounceRef.current = null;
                if (onUpdateRef.current) {
                    onUpdateRef.current();
                }
            }, 2000);
        };

        wsRef.current = ws;
    }, []);

    useEffect(() => {
        connect();
        return () => {
            if (wsRef.current) {
                wsRef.current.onclose = null; // Reconnect oldini olish
                wsRef.current.close();
            }
            if (debounceRef.current) {
                clearTimeout(debounceRef.current);
            }
        };
    }, [connect]);
}
