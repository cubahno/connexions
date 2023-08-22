import * as config from "./config.js";
import * as commons from "./commons.js";
import * as navi from "./navi.js";
import * as services from "./services.js";

export const home = () => {
    navi.resetContents();
    services.show();

    config.homeContents.style.display = 'block';
}

export const importForm = () => {
    config.servicesLink.className = `menu-link inactive`;
    config.contextsLink.className = `menu-link active`;

    navi.resetContents();
    services.show();

    config.contentTitleEl.innerHTML = `Import resources from a file`;
    config.resourcesImportForm.style.display = 'block';

    document.getElementById('zip-fileupload').addEventListener('change', () => {
        const file = document.getElementById('zip-fileupload').files[0];
        const selectedFilenameElement = document.getElementById('zip-selected-filename');
        selectedFilenameElement.textContent = '';
        if (file) {
            // Display the filename in the element
            selectedFilenameElement.textContent = file.name;
        }
    });
    document.getElementById('zip-upload-button').addEventListener('click', save);
}

export async function save(event) {
    event.preventDefault();
    let formData = new FormData();

    formData.append("file", document.getElementById('zip-fileupload').files[0]);

    config.messageCont.textContent = '';
    return await fetch(`${config.homeUrl}/import`, {
        method: "POST",
        body: formData,
    }).then(res => res.json()).then(res => {
        commons.showSuccessOrError(res.message, res.success);

        if (res.success) {
            services.show();
        }
        return res;
    });
}
