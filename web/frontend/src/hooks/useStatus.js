import { useState, useEffect, useCallback, useRef } from 'react';
import { fetchStatus } from '../api/status';

/**
 * Status data fetch qilish va 30s interval bilan yangilash
 */
export function useStatus() {
    const [data, setData] = useState(null);
    const [error, setError] = useState(null);
    const [loading, setLoading] = useState(true);
    const dataRef = useRef(null);

    const load = useCallback(async () => {
        try {
            const result = await fetchStatus();
            setData(result);
            dataRef.current = result;
            setError(null);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }, []);

    // Tashqaridan chaqirish uchun
    const refresh = useCallback(() => {
        load();
    }, [load]);

    useEffect(() => {
        load();
        const interval = setInterval(load, 30000);
        return () => clearInterval(interval);
    }, [load]);

    return { data, error, loading, refresh, dataRef };
}
