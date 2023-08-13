import * as commons from './commons.js';
import * as navi from './navi.js';
import * as settings from './settings.js';
import * as services from './services.js';
import * as resources from './resources.js';

const pageMap = new Map([
    ["#/settings", settings.editForm],
    ['#/services/upload', services.newForm],
    ['#/services/:name/ui', services.showSwagger],
    ['#/services/:name', resources.show],
    ['#/services/:name/:ix/:action', resources.show],
    ['#/services', () => services.show()],

]);

const onLoad = () => {
    navi.resetContents();
    services.show();
    navi.loadPage(pageMap);

    // Get the accordion header and content elements
    const accordionHeader = document.querySelector('.accordion-header');
    const accordionContent = document.querySelector('.accordion-content');
    accordionHeader.addEventListener('click', () => {
        accordionContent.classList.toggle('active');
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
}

window.addEventListener('hashchange', _ => {
    commons.hideMessage();
    navi.loadPage(pageMap);
})
window.addEventListener("DOMContentLoaded", _ => onLoad())
