import * as commons from './commons.js';
import * as config from './config.js';
import * as navi from './navi.js';
import * as settings from './settings.js';
import * as services from './services.js';
import * as home from './home.js';
import * as contexts from './contexts.js';
import * as resources from './resources.js';

const pageMap = new Map([
    ['', home.home],
    ['#/import', home.importForm],
    ["#/settings", settings.editForm],
    ['#/services/add', services.newForm],
    ['#/services/:name/ui', services.showSwagger],
    ['#/services/:name', resources.show],
    ['#/services/:name/:ix/:action', resources.show],
    ['#/services', services.show],
    ['#/contexts', contexts.show],
    ['#/contexts/:name', contexts.editForm],
    ['#/contexts/add', contexts.editForm],
]);

async function onLoad() {
    navi.resetContents();
    navi.loadPage(pageMap);

    const accordionHeaders = document.querySelectorAll('.accordion-header');
    accordionHeaders.forEach(accordionHeader => {
        accordionHeader.addEventListener('click', () => {
            const accordion = accordionHeader.closest('.accordion');
            const accordionContent = accordion.querySelector('.accordion-content');
            accordionContent.classList.toggle('active');
        });
    });

    document.getElementById('settings-save-button').addEventListener('click', settings.save);
    document.getElementById('settings-default-save-button').addEventListener('click', settings.restore);

    document.getElementById('fileupload').addEventListener('change', () => {
        const file = document.getElementById('fileupload').files[0];
        const selectedFilenameElement = document.getElementById('selected-filename');
        selectedFilenameElement.textContent = '';
        if (file) {
            // Display the filename in the element
            selectedFilenameElement.textContent = file.name;
            commons.getCodeEditor(`selected-text-response`, `yaml`).setValue(``);
        }
    });
    document.getElementById('upload-button').addEventListener('click', services.saveWithFile);
    document.getElementById('res-upload-button').addEventListener('click',services.saveWithoutFile);
    document.getElementById('export-link').href = `${config.homeUrl}/export`;
}

window.addEventListener('hashchange', _ => {
    commons.hideMessage();
    navi.loadPage(pageMap);
})
window.addEventListener("DOMContentLoaded", _ => onLoad())
