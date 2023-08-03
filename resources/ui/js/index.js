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

    const accordionHeaders = document.querySelectorAll('.accordion-header');
    accordionHeaders.forEach(accordionHeader => {
        accordionHeader.addEventListener('click', () => {
            const accordion = accordionHeader.closest('.accordion');
            const accordionContent = accordion.querySelector('.accordion-content');
            accordionContent.classList.toggle('active');
        });
    });
}

window.addEventListener('hashchange', _ => {
    commons.hideMessage();
    navi.loadPage(pageMap);
})
window.addEventListener("DOMContentLoaded", _ => onLoad())
