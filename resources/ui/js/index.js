import * as commons from './commons.js';
import * as navi from './navi.js';
import * as services from './services.js';
import * as home from './home.js';
import * as resources from './resources.js';

const pageMap = new Map([
    ['', home.home],
    ['#/services/:name*', resources.show],
    ['#/services', services.show],
]);

async function onLoad() {
    navi.resetContents();
    navi.loadPage(pageMap);
    home.showVersion();

    // Theme toggle
    const themeToggle = document.getElementById('theme-toggle');
    const savedTheme = localStorage.getItem('theme') || 'light';
    if (savedTheme === 'dark') {
        document.documentElement.setAttribute('data-theme', 'dark');
        themeToggle.textContent = '☀️';
    }
    themeToggle.addEventListener('click', () => {
        const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
        if (isDark) {
            document.documentElement.removeAttribute('data-theme');
            localStorage.setItem('theme', 'light');
            themeToggle.textContent = '🌙';
        } else {
            document.documentElement.setAttribute('data-theme', 'dark');
            localStorage.setItem('theme', 'dark');
            themeToggle.textContent = '☀️';
        }
        commons.initAceThemeSelect();
        commons.updateAllEditorThemes();
    });

    commons.initAceThemeSelect();

    // Panel resizer
    const RESIZER_STORAGE_KEY = 'panel-split';
    const contentPanels = document.querySelector('.content-panels');
    const resizer = document.querySelector('.panel-resizer');

    const savedSplit = localStorage.getItem(RESIZER_STORAGE_KEY);
    if (savedSplit) {
        contentPanels.style.setProperty('--resources-width', savedSplit + '%');
    }

    resizer.addEventListener('mousedown', (e) => {
        e.preventDefault();
        resizer.classList.add('dragging');
        contentPanels.classList.add('resizing');

        const onMouseMove = (e) => {
            const rect = contentPanels.getBoundingClientRect();
            const pct = ((e.clientX - rect.left) / rect.width) * 100;
            const clamped = Math.min(Math.max(pct, 20), 80);
            contentPanels.style.setProperty('--resources-width', clamped + '%');
        };

        const onMouseUp = () => {
            document.removeEventListener('mousemove', onMouseMove);
            document.removeEventListener('mouseup', onMouseUp);
            resizer.classList.remove('dragging');
            contentPanels.classList.remove('resizing');
            const leftPanel = document.querySelector('.panel-resources');
            const pct = (leftPanel.offsetWidth / contentPanels.offsetWidth) * 100;
            localStorage.setItem(RESIZER_STORAGE_KEY, pct.toFixed(1));
        };

        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('mouseup', onMouseUp);
    });

    // Copy buttons
    document.addEventListener('click', (e) => {
        const btn = e.target.closest('.copy-btn');
        if (!btn) return;
        e.stopPropagation();

        const target = btn.dataset.copyTarget;
        let text = '';
        if (target === 'curl') {
            text = document.getElementById('example-curl').textContent;
        } else {
            const el = document.getElementById(target);
            if (el && el.env) {
                text = el.env.editor.getValue();
            }
        }

        if (!text) return;
        navigator.clipboard.writeText(text).then(() => {
            btn.textContent = 'Copied!';
            setTimeout(() => { btn.textContent = 'Copy'; }, 2000);
        });
    });

    const ACCORDION_STORAGE_KEY = 'accordion-states';
    const getAccordionStates = () => {
        try { return JSON.parse(localStorage.getItem(ACCORDION_STORAGE_KEY)) || {}; }
        catch { return {}; }
    };

    const accordionHeaders = document.querySelectorAll('.accordion-header');
    const savedStates = getAccordionStates();

    accordionHeaders.forEach(accordionHeader => {
        const key = accordionHeader.textContent.trim();
        const accordionContent = accordionHeader.closest('.accordion').querySelector('.accordion-content');

        if (key in savedStates) {
            accordionContent.classList.toggle('active', savedStates[key]);
        }

        accordionHeader.addEventListener('click', () => {
            accordionContent.classList.toggle('active');
            const states = getAccordionStates();
            states[key] = accordionContent.classList.contains('active');
            localStorage.setItem(ACCORDION_STORAGE_KEY, JSON.stringify(states));
        });
    });
}

window.addEventListener('hashchange', _ => {
    commons.hideMessage();
    navi.loadPage(pageMap);
})
window.addEventListener("DOMContentLoaded", _ => onLoad())
