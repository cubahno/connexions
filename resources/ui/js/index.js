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
        commons.updateAllEditorThemes();
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
