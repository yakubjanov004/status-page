import { useState, useCallback } from 'react';
import { useStatus } from './hooks/useStatus';
import { useWebSocket } from './hooks/useWebSocket';
import Header from './components/Header';
import StatusBanner from './components/StatusBanner';
import StatsRow from './components/StatsRow';
import AttentionBanner from './components/AttentionBanner';
import StatusCard from './components/StatusCard';
import OutagesSection from './components/OutagesSection';
import MaintenanceLogs from './components/MaintenanceLogs';
import ProjectModal from './components/ProjectModal';
import Legend from './components/Legend';
import Skeleton from './components/Skeleton';
import Footer from './components/Footer';
import './components/Skeleton.css';

export default function App() {
    const { data, error, loading, refresh } = useStatus();
    const [filterQuery, setFilterQuery] = useState('');
    const [showOnlyIssues, setShowOnlyIssues] = useState(false);
    const [modalSlug, setModalSlug] = useState('');
    const [modalOpen, setModalOpen] = useState(false);

    // WebSocket — yangilanish kelganda refresh chaqiramiz
    useWebSocket(refresh);

    const handleOpenProject = useCallback((slug) => {
        setModalSlug(slug);
        setModalOpen(true);
    }, []);

    const handleCloseModal = useCallback(() => {
        setModalOpen(false);
        setModalSlug('');
    }, []);

    const handleToggleIssues = useCallback(() => {
        setShowOnlyIssues((prev) => !prev);
    }, []);

    // Title yangilash
    if (data) {
        const siteName = data.site_name || 'Server Status';
        document.title = `${siteName} — ${data.status || 'Status'}`;
    }

    return (
        <>
            <Header
                siteName={data?.site_name}
                lastUpdated={data?.last_updated}
            />

            <main>
                <div className="wrap">
                    <div className="page-body">
                        {loading && !data ? (
                            <Skeleton />
                        ) : error && !data ? (
                            <div className="status-banner outage fade-in">
                                <div className="banner-body">
                                    <div className="banner-icon">🔴</div>
                                    <div>
                                        <div className="banner-title">Ulanish xatosi</div>
                                        <div className="banner-sub">
                                            Tizim holatini yuklab bo'lmadi. 30 soniyadan so'ng qayta uriniladi…
                                        </div>
                                    </div>
                                </div>
                            </div>
                        ) : data ? (
                            <>
                                <AttentionBanner
                                    projects={data.projects}
                                    onOpenProject={handleOpenProject}
                                />

                                <StatusBanner status={data.status} />

                                <StatsRow
                                    overallUptime={data.overall_uptime_pct}
                                    totalServices={data.total_services}
                                    activeIncidents={data.active_incidents}
                                />

                                <StatusCard
                                    projects={data.projects}
                                    filterQuery={filterQuery}
                                    onFilterChange={setFilterQuery}
                                    showOnlyIssues={showOnlyIssues}
                                    onToggleIssues={handleToggleIssues}
                                    onOpenProject={handleOpenProject}
                                />

                                <MaintenanceLogs logs={data.maintenance_logs} />

                                <OutagesSection outages={data.recent_outages} />

                                <Legend />
                            </>
                        ) : null}
                    </div>
                </div>
            </main>

            <ProjectModal
                slug={modalSlug}
                isOpen={modalOpen}
                onClose={handleCloseModal}
            />

            <Footer
                siteName={data?.site_name}
                lastUpdated={data?.last_updated}
            />
        </>
    );
}
