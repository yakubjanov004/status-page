import { fmtDate, timeAgo } from '../utils/format';
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
    primaryProblem,
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
        let upCount = 0;
        let downCount = 0;

        const filteredComps = (proj.components || []).filter((comp) => {
            const q = filterQuery.toLowerCase();
            const matchSearch =
                comp.name.toLowerCase().includes(q) ||
                proj.name.toLowerCase().includes(q);

            const hasIssue = !comp.is_up;
            const matchFilter = !showOnlyIssues || hasIssue;

            return matchSearch && matchFilter;
        });
        
        (proj.components || []).forEach(comp => {
            if (comp.is_up) upCount++;
            else downCount++;
        });
        
        let isProjDown = downCount > 0 && upCount === 0; // All down
        let isProjUp = downCount === 0; // All up
        // If downCount > 0 && upCount > 0, both are false, which triggers 'warn' (Qisman uzilish) in rendering

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
                <div className="section-title">Loyihalar</div>
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
                                    <div className="project-card-top">
                                        <div className="project-card-left">
                                            <span className="project-card-icon">{statusIcon}</span>
                                            <span className="project-card-name">{proj.name}</span>
                                        </div>
                                        <span className={`proj-badge ${badgeClass}`}>
                                            {badgeText}
                                        </span>
                                    </div>
                                    <div className="project-card-meta-row">
                                        {proj.restart_count > 0 && (
                                            <span className="restart-badge" title="Qayta ishga tushirishlar soni">
                                                🔄 {proj.restart_count} marta restart
                                                {proj.last_restart_at && ` · ${timeAgo(proj.last_restart_at)}`}
                                            </span>
                                        )}
                                        <span className="project-card-hint">
                                            Batafsil ko'rish uchun bosing →
                                        </span>
                                    </div>
                                </div>
                                <div className="project-card-body">
                                    {proj.filteredComps.map((comp, i) => (
                                        <ComponentRow
                                            key={i}
                                            comp={comp}
                                            isPrimaryProblem={
                                                !!primaryProblem &&
                                                primaryProblem.projectSlug === proj.slug &&
                                                primaryProblem.componentName === comp.name
                                            }
                                        />
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
