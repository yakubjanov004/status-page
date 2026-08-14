import { useState, useEffect, useCallback } from 'react';
import { fetchProjectStatus } from '../api/status';
import ComponentRow from './ComponentRow';
import OutagesSection from './OutagesSection';
import './ProjectModal.css';

const RANGES = ['7d', '30d', '90d'];
const RANGE_LABELS = { '7d': '7 kun', '30d': '30 kun', '90d': '90 kun' };

export default function ProjectModal({ slug, isOpen, onClose }) {
    const [range, setRange] = useState('7d');
    const [data, setData] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState(null);

    const loadData = useCallback(async (r) => {
        if (!slug) return;
        setLoading(true);
        setError(null);
        try {
            const result = await fetchProjectStatus(slug, r);
            setData(result);
            window.location.hash = '/project/' + slug;
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }, [slug]);

    useEffect(() => {
        if (isOpen && slug) {
            setRange('7d');
            loadData('7d');
        }
    }, [isOpen, slug, loadData]);

    const handleRangeChange = (r) => {
        setRange(r);
        loadData(r);
    };

    const handleClose = () => {
        onClose();
        // URL hash tozalash
        if (window.location.hash) {
            window.history.pushState('', document.title, window.location.pathname);
        }
    };

    // Overlay bosilsa yopish
    const handleOverlayClick = (e) => {
        if (e.target === e.currentTarget) {
            handleClose();
        }
    };

    // Body scroll qulflash
    useEffect(() => {
        if (isOpen) {
            document.body.style.overflow = 'hidden';
        } else {
            document.body.style.overflow = '';
        }
        return () => {
            document.body.style.overflow = '';
        };
    }, [isOpen]);

    return (
        <div
            className={`modal-overlay ${isOpen ? 'open' : ''}`}
            onClick={handleOverlayClick}
        >
            <div className="modal-content">
                <div className="modal-header">
                    <h2>{data?.name || 'Yuklanmoqda...'}</h2>
                    <button className="modal-close" onClick={handleClose}>
                        &times;
                    </button>
                </div>
                <div className="modal-tabs">
                    {RANGES.map((r) => (
                        <button
                            key={r}
                            className={`tab-btn ${range === r ? 'active' : ''}`}
                            onClick={() => handleRangeChange(r)}
                        >
                            {RANGE_LABELS[r]}
                        </button>
                    ))}
                </div>
                <div className="modal-body">
                    {loading ? (
                        <div className="modal-loading">Yuklanmoqda...</div>
                    ) : error ? (
                        <div className="modal-error">
                            Loyiha ma'lumotlarini yuklashda xatolik
                        </div>
                    ) : data ? (
                        <>
                            {(data.components || []).length === 0 ? (
                                <div className="modal-loading">
                                    Komponentlar topilmadi.
                                </div>
                            ) : (
                                (data.components || []).map((comp, i) => (
                                    <ComponentRow key={i} comp={comp} />
                                ))
                            )}
                            {data.incidents && data.incidents.length > 0 && (
                                <OutagesSection outages={data.incidents} />
                            )}
                        </>
                    ) : null}
                </div>
            </div>
        </div>
    );
}
