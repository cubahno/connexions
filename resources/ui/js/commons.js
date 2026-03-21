import * as config from './config.js';

export const getEditorTheme = () => {
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
    return isDark ? config.editor.darkTheme : config.editor.theme;
}

export const updateAllEditorThemes = () => {
    const theme = `ace/theme/${getEditorTheme()}`;
    for (const editor of activeEditors) {
        editor.setTheme(theme);
    }
}

const activeEditors = new Set();

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

export const getCodeEditor = (htmlID, mode, opts) => {
    const codeEditorContainer = document.getElementById(htmlID);
    const editor = ace.edit(codeEditorContainer);

    const options = {
        showLineNumbers: true,
        mode: `ace/mode/${mode}`,
        showPrintMargin: false,
        ...opts,
    };
    editor.setOptions(options);

    editor.setTheme(`ace/theme/${getEditorTheme()}`);
    editor.setFontSize(`${config.editor.fontSize}px`);
    editor.resize();

    activeEditors.add(editor);
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
            const mapped = getCodeEditorMode(value)
            editor.setOptions({
                mode: `ace/mode/${mapped}`,
            })
        })
    }

    return editor;
}

export const isResourceEditable = contentType => {
    const editableContentTypes = {
        'application/json': true,
        'application/x-yaml': true,
        'text/plain; charset=utf-8': true,
        'text/markdown; charset=utf-8' : true,
        'text/html; charset=utf-8': true,
        'text/xml; charset=utf-8': true,
    }
    return editableContentTypes.hasOwnProperty(contentType);
}
