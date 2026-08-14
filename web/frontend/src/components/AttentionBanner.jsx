import './AttentionBanner.css';

export default function AttentionBanner({ projects, onOpenProject }) {
    const issues = [];

    if (projects) {
        projects.forEach((p) => {
            (p.components || []).forEach((c) => {
                const pct = typeof c.uptime_pct === 'number' ? c.uptime_pct : 100;
                if (!c.is_up) {
                    issues.push({ name: c.name, slug: p.slug });
                }
            });
        });
    }

    if (issues.length === 0) return null;

    return (
        <div className="attention-banner fade-in">
            <div className="attention-title">⚠ Diqqat talab qiladi</div>
            <div className="attention-links">
                Muammoli xizmatlar:{' '}
                {issues.map((issue, i) => (
                    <button key={i} onClick={() => onOpenProject(issue.slug)}>
                        {issue.name}
                    </button>
                ))}
            </div>
        </div>
    );
}
