import * as commons from './commons.js';
import * as navi from "./navi.js";
import * as config from "./config.js";
import * as services from "./services.js";


export const editForm = () => {
    console.log(`settings edit`);
    navi.applySelection(`n/a`, 'selected-service');
    navi.resetContents();
    config.contentTitleEl.innerHTML = `Edit Settings`;

    const editor = commons.getCodeEditor(`code-editor`, `yaml`);

    fetch(config.settingsUrl)
        .then(res => res.text())
        .then(res => {
            editor.setValue(res);
            editor.clearSelection();
        })

    config.settingsEditor.style.display = 'block';
}

export const save = () => {
    const editor = commons.getCodeEditor(`code-editor`, `yaml`);
    const yaml = editor.getValue();
    commons.showWarning("Reloading settings...")

    fetch(config.settingsUrl, {
        method: "PUT",
        headers: {
            "Content-Type": "application/json"
        },
        body: yaml,
    }).then(res => res.json()).then(res => {
        commons.showSuccessOrError(res.message, res.success);
        services.show();
    });
}

export const restore = () => {
    fetch(config.settingsUrl, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
    }).then(res => res.json()).then(res => {
        commons.showSuccessOrError(res.message, res.success);
        services.show();
        editForm();
    });
}
