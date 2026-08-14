import { timeAgo } from '../utils/format';
import './Footer.css';

export default function Footer({ siteName, lastUpdated }) {
    return (
        <footer className="site-footer">
            <div className="wrap">
                Powered by{' '}
                <a href="https://status.darrov.uz">
                    {siteName || 'Server Status'} Monitor
                </a>
                &nbsp;·&nbsp;
                <span>Yangilandi: {timeAgo(lastUpdated)}</span>
            </div>
        </footer>
    );
}
