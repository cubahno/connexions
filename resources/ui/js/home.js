import * as config from "./config.js";
import * as commons from "./commons.js";
import * as services from "./services.js";
import * as navi from "./navi.js";

export const importForm = () => {
    config.servicesLink.className = `menu-link inactive`;
    config.contextsLink.className = `menu-link active`;

    navi.resetContents();

    config.contentTitleEl.innerHTML = `Import all resources from a file`;
    config.servicesUploadForm.style.display = 'block';

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
}
