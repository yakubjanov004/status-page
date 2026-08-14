import './FilterBar.css';

export default function FilterBar({ filterQuery, onFilterChange, showOnlyIssues, onToggleIssues }) {
    return (
        <div className="filter-bar">
            <input
                type="text"
                className="search-input"
                placeholder="Komponentlarni izlash..."
                value={filterQuery}
                onChange={(e) => onFilterChange(e.target.value)}
            />
            <button
                className={`toggle-btn ${showOnlyIssues ? 'active' : ''}`}
                onClick={onToggleIssues}
            >
                Faqat muammolarni ko'rsatish
            </button>
        </div>
    );
}
