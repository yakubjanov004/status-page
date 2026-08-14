import { fmtDate } from '../utils/format';
import FilterBar from './FilterBar';
import ComponentRow from './ComponentRow';
import './StatusCard.css';

export default function StatusCard({
    projects,
    filterQuery,
    onFilterChange,
    showOnlyIssues,
    onToggleIssues,
    onOpenProject,
}) {
    // Date range
    let dateRangeStr = "So'nggi 7 kun";
    if (projects && projects.length > 0) {
        const comps = projects[0].components;
        if (comps && comps.length > 0 && comps[0].history && comps[0].history.length > 0) {
            const hist = comps[0].history;
            const d0 = fmtDate(hist[0].date);
            const d1 = fmtDate(hist[hist.length - 1].date);
            dateRangeStr = `${d0} – ${d1}`;
        }
    }

    // Filter components
    let visibleCount = 0;
    const projectGroups = (projects || []).map((proj) => {
        let isProjUp = true;
        let isProjDown = false;

        const filteredComps = (proj.components || []).filter((comp) => {
            const q = filterQuery.toLowerCase();
            const matchSearch =
                comp.name.toLowerCase().includes(q) ||
                proj.name.toLowerCase().includes(q);

            const pct = typeof comp.uptime_pct === 'number' ? comp.uptime_pct : 100;
            const hasIssue = !comp.is_up || pct < 99;
            const matchFilter = !showOnlyIssues || hasIssue;

            if (!comp.is_up) isProjDown = true;
            if (hasIssue) isProjUp = false;

            return matchSearch && matchFilter;
        });

        visibleCount += filteredComps.length;

        return {
            ...proj,
            filteredComps,
            isProjUp,
            isProjDown,
        };
    });

    return (
        <div className="status-section fade-in">
            <FilterBar
                filterQuery={filterQuery}
                onFilterChange={onFilterChange}
                showOnlyIssues={showOnlyIssues}
                onToggleIssues={onToggleIssues}
            />
            <div className="section-header">
                <div className="section-title">Tizim holati</div>
                <div className="date-range">‹ {dateRangeStr} ›</div>
            </div>

            {!projects || projects.length === 0 ? (
                <div className="no-results">Hali xizmatlar sozlanmagan.</div>
            ) : visibleCount === 0 ? (
                <div className="no-results">Qidiruvingizga mos komponent topilmadi.</div>
            ) : (
                <div className="projects-grid">
                    {projectGroups.map((proj) => {
                        if (proj.filteredComps.length === 0) return null;

                        let badgeClass = 'up';
                        let badgeText = 'Ishlamoqda';
                        let statusIcon = '✅';
                        if (proj.isProjDown) {
                            badgeClass = 'down';
                            badgeText = 'Uzilish';
                            statusIcon = '🔴';
                        } else if (!proj.isProjUp) {
                            badgeClass = 'warn';
                            badgeText = 'Qisman uzilish';
                            statusIcon = '🟡';
                        }

                        return (
                            <div className={`project-card card-${badgeClass}`} key={proj.id || proj.slug}>
                                <div
                                    className="project-card-header"
                                    onClick={() => onOpenProject(proj.slug)}
                                >
                                    <div className="project-card-left">
                                        <span className="project-card-icon">{statusIcon}</span>
                                        <span className="project-card-name">{proj.name}</span>
                                        <span className={`proj-badge ${badgeClass}`}>
                                            {badgeText}
                                        </span>
                                    </div>
                                    <span className="project-card-hint">
                                        Batafsil ko'rish uchun bosing
                                    </span>
                                </div>
                                <div className="project-card-body">
                                    {proj.filteredComps.map((comp, i) => (
                                        <ComponentRow key={i} comp={comp} />
                                    ))}
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
