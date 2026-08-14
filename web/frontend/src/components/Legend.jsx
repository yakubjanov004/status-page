import './Legend.css';

export default function Legend() {
    return (
        <div className="legend-row fade-in" style={{ display: 'flex' }}>
            <div className="legend-item">
                <div className="legend-color" style={{ background: 'var(--up)' }}></div>
                100% ishlash vaqti
            </div>
            <div className="legend-item">
                <div className="legend-color" style={{ background: '#34d399' }}></div>
                ≥ 90%
            </div>
            <div className="legend-item">
                <div className="legend-color" style={{ background: 'var(--warn)' }}></div>
                ≥ 50%
            </div>
            <div className="legend-item">
                <div className="legend-color" style={{ background: 'var(--down)' }}></div>
                &lt; 50%
            </div>
            <div className="legend-item">
                <div className="legend-color c-nodata"></div>
                Ma'lumot yo'q
            </div>
        </div>
    );
}
