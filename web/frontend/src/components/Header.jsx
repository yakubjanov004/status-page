import { useState } from 'react';
import { toggleTheme, getInitialTheme } from '../utils/theme';
import { timeAgo } from '../utils/format';
import './Header.css';

export default function Header({ siteName, lastUpdated }) {
    const [theme, setTheme] = useState(getInitialTheme());

    const handleToggle = () => {
        const next = toggleTheme();
        setTheme(next);
    };

    return (
        <header className="site-header">
            <div className="wrap">
                <div className="header-inner">
                    <a href="/" className="logo" id="site-logo">
                        <div className="logo-mark">D</div>
                        <span className="logo-text">{siteName || 'Server Status'}</span>
                    </a>
                    <div className="header-right">
                        <div className="last-updated">
                            <span className="live-dot"></span>
                            <span>Yangilandi: {timeAgo(lastUpdated)}</span>
                        </div>
                        <button
                            className="theme-btn"
                            onClick={handleToggle}
                            title="Toggle theme"
                            aria-label="Toggle dark/light theme"
                        >
                            {theme === 'dark' ? '☀️' : '🌙'}
                        </button>
                    </div>
                </div>
            </div>
        </header>
    );
}
