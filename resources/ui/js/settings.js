import * as commons from './commons.js';

export const save = () => {
    const editor = commons.getCodeEditor(`code-editor`, `yaml`);
    const yaml = editor.getValue();
    commons.showWarning("Reloading settings...")

    fetch('/settings', {
        method: "PUT",
        headers: {
            "Content-Type": "application/json"
        },
        body: yaml,
    }).then(res => res.json()).then(res => {
        commons.showSuccessOrError(res.message, res.success);
        showServices();
    });
}
