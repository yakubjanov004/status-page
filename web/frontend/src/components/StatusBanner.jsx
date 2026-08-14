import './StatusBanner.css';

export default function StatusBanner({ status }) {
    let bannerClass = 'operational';
    let bannerIcon = '✅';
    let bannerTitle = 'Barcha tizimlar ishlamoqda';
    let bannerSub = "Tizimlarimizda hech qanday muammo yo'q.";

    const st = status || '';
    if (st.includes('Partially') || st.includes('Degraded')) {
        bannerClass = 'degraded';
        bannerIcon = '⚠️';
        bannerTitle = 'Qisman tizim uzilishi';
        bannerSub = "Ba'zi xizmatlarda muammolar mavjud.";
    } else if (st.includes('Outage') || st.includes('Major')) {
        bannerClass = 'outage';
        bannerIcon = '🔴';
        bannerTitle = 'Jiddiy tizim uzilishi';
        bannerSub = "Tizimlarimizda keng ko'lamli muammolar mavjud.";
    }

    return (
        <div className={`status-banner ${bannerClass} fade-in`}>
            <div className="banner-body">
                <div className="banner-icon">{bannerIcon}</div>
                <div>
                    <div className="banner-title">{bannerTitle}</div>
                    <div className="banner-sub">{bannerSub}</div>
                </div>
            </div>
        </div>
    );
}
