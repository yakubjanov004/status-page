import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { applyTheme, getInitialTheme } from './utils/theme';
import App from './App';
import './index.css';

// Boshlang'ich tema qo'llash
applyTheme(getInitialTheme());

createRoot(document.getElementById('root')).render(
    <StrictMode>
        <App />
    </StrictMode>
);
