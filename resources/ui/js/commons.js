import * as config from './config.js';

export const showSuccess = text => {
    showMessage(text, 'success')
}

export const showWarning = text => {
    showMessage(text, 'warning')
}

export const showError = text => {
    showMessage(text, 'error')
}

export const showSuccessOrError = (text, success) => {
    showMessage(text, success ? 'success' : 'error')
}

export const showMessage = (text, alertType) => {
    config.messageCont.textContent = text;
    config.messageCont.className = `alert-${alertType}`
    config.messageCont.style.display = 'block';
    config.messageCont.style.opacity = '1';
}

export const hideMessage = () => {
    config.messageCont.style.display = 'none';
}

export const getCodeEditor = (htmlID, mode) => {
    // Get the code editor container element
    const codeEditorContainer = document.getElementById(htmlID);

    // Create the Ace Editor instance
    const editor = ace.edit(codeEditorContainer);

    // Set the editor options
    editor.setOptions({
        // Enable line numbers
        showLineNumbers: true,
        mode: `ace/mode/${mode}`,
        showPrintMargin: false,
    });

    editor.setTheme("ace/theme/xcode");
    editor.setFontSize("14px");
    editor.resize();

    return editor;
}

export const getCodeEditorMode = value => {
    const contentMap = {
        yml: `yaml`,
        md: `markdown`,
        txt: `text`,
    }
    return contentMap.hasOwnProperty(value) ? contentMap[value] : value;
}

export const getEditorForm = (editorId, typeId) => {
    console.log(`response edit in ${editorId}`);

    const editor = getCodeEditor(editorId, `json`);
    editor.setValue(``);
    editor.clearSelection();
    console.log(`typeID:`, typeId);
    if (typeId !== undefined) {
        document.getElementById(typeId).addEventListener(`change`, el => {
            const value = el.target.value;
            editor.setOptions({
                mode: `ace/mode/${value}`,
            })
        })
    }

    return editor;
}
