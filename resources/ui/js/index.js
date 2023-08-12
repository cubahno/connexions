const url = window.location.origin;

const serviceTable = document.getElementById('service-table');
const generatorCont = document.getElementById('generator-container');
const serviceResourcesEl = document.getElementById('service-resources');
const contentTitleEl = document.getElementById('container-title');
const iframeContents = document.getElementById('iframe-contents');
const servicesUploadForm = document.getElementById('services-upload');
const messageCont = document.getElementById('message');
const fileUploadBtn = document.getElementById('fileupload');
const settingsEditor = document.getElementById('settings-editor');
const fixedServiceContainer = document.getElementById('fixed-service-container');
const resourceRefreshBtn = document.getElementById('refresh');
const responseEditContainer =  document.getElementById(`selected-text-response`);

const resetContents = () => {
    console.log(`reset contents`);
    iframeContents.src = '';
    iframeContents.style.display = 'none';

    serviceResourcesEl.innerHTML = '';
    serviceResourcesEl.style.display = 'none';

    generatorCont.style.display = 'none';

    servicesUploadForm.style.display = 'none';
    settingsEditor.style.display = 'none';
    fixedServiceContainer.style.display = 'none';

    document.getElementById('fixed-service-table-body').innerHTML = '';
    document.getElementById('resource-result').innerHTML = '';

    resourceRefreshBtn.style.display = 'none';
}

const showServices = () => {
    serviceTable.innerHTML = '';
    console.log("loading service list");

    fetch(`${url}/services`)
        .then(getResponseJson)
        .then(data => {
            const services = data['items'];

            let i = 0;
            for (let { name, isOpenApi } of services) {
                const num = i + 1;
                const row = document.createElement('tr');
                row.id = `service-${name}`;

                const cell1 = document.createElement('td');
                cell1.textContent = `${num}`;
                row.appendChild(cell1);

                let nameLink = name;
                if (name === ``) {
                    name = "/"
                    nameLink = `--`
                }
                const svcNameCell = document.createElement('td');
                svcNameCell.innerHTML = `<a href="#/services/${nameLink}">${name}</a>`;
                row.appendChild(svcNameCell);

                const swaggerCell = document.createElement('td');
                let swaggerLink = '&nbsp;';
                if (isOpenApi) {
                    swaggerLink = `<a href="#/services/${name}/ui"><img class="swagger-icon" src="/ui/icons/swagger.svg"></a>`;
                }
                swaggerCell.innerHTML = swaggerLink;

                row.appendChild(swaggerCell);

                serviceTable.appendChild(row);
                i += 1;
            }

            serviceTable.style.display = 'block';
        });
}

const applySelection = (targetEl, selectionClassName) => {
    console.log(`applying selection for ${targetEl}`);
    if (!targetEl) {
        return;
    }

    if (targetEl === `service---`) {
        targetEl = `service-`;
    }

    const collection = document.getElementsByClassName(selectionClassName);
    for (let i = 0; i < collection.length; i++) {
        collection[i].classList.remove(selectionClassName);
    }
    const row = document.getElementById(targetEl);
    if (!row) {
        console.log(`no row found for ${targetEl}`);
        return;
    }
    row.classList.add(selectionClassName);
}

const serviceHome = match => {
    const service = match.params.name;
    resetContents();
    const editor = getCodeEditor(`replacements2`, `json`);
    editor.setValue(`{\n\t\n}`);
    editor.clearSelection();

    fetch(`${url}/services/${service}`)
        .then(getResponseJson)
        .then(data => {
            applySelection(`service-${service}`, 'selected-service');

            const endpoints = data['endpoints'];
            let name = service;
            if (name === `--`) {
                name = `Root level`
            }
            contentTitleEl.innerHTML = `${name} resources`;

            const table = document.getElementById('fixed-service-table-body');
            let i = 0;

            for (const { method, path, type } of endpoints) {
                const num = i + 1;
                let icon = ``;
                if (type === `overwrite`) {
                    // icon = ` <span title="overwrites" style="text-decoration: none;">🔁</span>`;
                }

                const row = document.createElement('tr');
                row.id = `resource-${num}`;

                const cell1 = document.createElement('td');
                cell1.textContent = `${num}`;
                cell1.className = 'fixed-resource-num';
                row.appendChild(cell1);

                const methodCell = document.createElement('td');
                methodCell.innerHTML = `${method.toUpperCase()} ${icon}`;
                methodCell.className = `fixed-resource-method ${method}`;
                row.appendChild(methodCell);

                const pathCell = document.createElement('td');
                pathCell.innerHTML = `${path}`;
                pathCell.className = `fixed-resource-path`;
                row.appendChild(pathCell);

                pathCell.onclick = () => {
                    applySelection(`resource-${num}`, 'selected-resource');
                    loadResource(service, path, method, type === `openapi`);
                }

                table.appendChild(row);
                i += 1;
            }
            fixedServiceContainer.style.display = 'block';
        });
}

const loadResource = (service, path, method, isOpenApi) => {
    console.log(`loadResource: ${method} /${service}${path}`);

    const onDone = () => {
        generatorCont.style.display = 'block';
        resourceRefreshBtn.onclick = () => loadResource(service, path, method, isOpenApi);
        resourceRefreshBtn.style.display = 'block';
    }
    hideMessage();

    let replacements = fixAndValidateJSON(document.getElementById('replacements').value.trim());
    fetch(`${url}/services/${service}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            resource: path,
            method: method,
            replacements: replacements,
            isOpenApi: isOpenApi,
        }),
    })
        .then(getResponseJson)
        .then(payload => {
            const reqPath = payload["request"]["path"];
            if (reqPath) {
                document.getElementById('request-path').innerHTML = reqPath;
                document.getElementById('request-path-container').style.display = 'block';
            }

            if (method.toLowerCase() === 'get') {
                document.getElementById('request-body-container').style.display = 'none';
            } else {
                document.getElementById('request-body').textContent = JSON.stringify(payload["request"]["body"], null, 2);
                document.getElementById('request-body-container').style.display = 'block';
            }

            document.getElementById('response-body').textContent = JSON.stringify(payload["response"]["content"], null, 2);
            document.getElementById('response-body-container').style.display = 'block';
        }).then(onDone);
}

const serviceSwagger = match => {
    const service = match.params.name;
    applySelection(`service-${service}`, 'selected-service');
    resetContents();

    console.log(`Show swagger for ${service}`);

    contentTitleEl.innerHTML = `${service} Swagger / OpenAPI`;

    iframeContents.src = `${url}/ui/swaggerui?specUrl=/services/${service}/spec`;
    iframeContents.style.display = 'block';
}

const uploadNewServices = () => {
    console.log(`add new service`);
    applySelection(`n/a`, 'selected-service');
    resetContents();
    showResponseEditForm();
    contentTitleEl.innerHTML = `Add new service to the list`;

    servicesUploadForm.style.display = 'block';
}

async function uploadServiceFile() {
    let formData = new FormData();

    //const name = document.getElementById('endpoint-service-name').value.trim();
    const isOpenApi = document.querySelector('input[name="is_openapi"]:checked').value === '1';
    const method = document.getElementById('endpoint-method').value.trim();
    let path = '';
    if (!isOpenApi) {
        path = document.getElementById('endpoint-path').value.trim();
    }
    const response = getCodeEditor(`selected-text-response`, `json`).getValue();
    console.log(`response: ${response}`);

    formData.append("file", fileUploadBtn.files[0]);
    formData.append("response", response);
    //formData.append("name", name);
    formData.append("method", method);
    formData.append("isOpenApi", isOpenApi.toString());
    formData.append("path", path);

    messageCont.textContent = '';
    await fetch('/services', {
        method: "POST",
        body: formData,
    }).then(res => res.json()).then(res => {
        showSuccessOrError(res.message, res.success);

        if (res.success) {
            showServices();
        }
    });
}

const showSuccess = text => {
    showMessage(text, 'success')
}

const showWarning = text => {
    showMessage(text, 'warning')
}

const showError = text => {
    showMessage(text, 'error')
}

const showSuccessOrError = (text, success) => {
    console.log(text);
    showMessage(text, success ? 'success' : 'error')
}


const showMessage = (text, alertType) => {
    messageCont.textContent = text;
    messageCont.className = `alert-${alertType}`
    messageCont.style.display = 'block';
    messageCont.style.opacity = '1';
}

const hideMessage = () => {
    messageCont.style.display = 'none';
}

const settingsEdit = () => {
    console.log(`settings edit`);
    applySelection(`n/a`, 'selected-service');
    resetContents();
    contentTitleEl.innerHTML = `Edit Settings`;

    const editor = getCodeEditor(`code-editor`, `yaml`);

    fetch(`${url}/settings`)
        .then(getResponseText)
        .then(res => {
            editor.setValue(res);
            editor.clearSelection();
        })

    settingsEditor.style.display = 'block';
}

const settingsSave = () => {
    const editor = getCodeEditor(`code-editor`, `yaml`);
    const yaml = editor.getValue();
    showWarning("Reloading settings...")

    fetch('/settings', {
        method: "PUT",
        headers: {
            "Content-Type": "application/json"
        },
        body: yaml,
    }).then(getResponseJson).then(res => {
        showSuccessOrError(res.message, res.success);
        showServices();
    });
}

const settingsRestore = () => {
    fetch('/settings', {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
    }).then(res => res.json()).then(res => {
        showSuccessOrError(res.message, res.success);
        showServices();
        settingsEdit();
    });
}

const showResponseEditForm = () => {
    console.log(`response edit`);
    applySelection(`n/a`, 'selected-service');

    const editor = getCodeEditor(`selected-text-response`, `json`);
    editor.setValue(``);
    editor.clearSelection();

    responseEditContainer.style.display = 'block';
}

const getCodeEditor = (htmlID, mode) => {
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

// Helper function to check if a placeholder pattern matches the current hash
const isPlaceholderMatch = (placeholderPattern, currentHash) => {
    const patternParts = placeholderPattern.split('/');
    const hashParts = currentHash.split('/');

    if (patternParts.length !== hashParts.length) {
        return false;
    }

    for (let i = 0; i < patternParts.length; i++) {
        const patternPart = patternParts[i];
        if (patternPart.startsWith(':')) {
            continue; // Skip placeholder parts
        }

        if (patternPart !== hashParts[i]) {
            return false;
        }
    }

    return true;
}

// Helper function to extract the placeholder values from the current hash
const getPlaceholderMatch = (placeholderPattern, currentHash) => {
    const patternParts = placeholderPattern.split('/');
    const hashParts = currentHash.split('/');

    const match = {
        params: {}
    };

    for (let i = 0; i < patternParts.length; i++) {
        const patternPart = patternParts[i];
        if (patternPart.startsWith(':')) {
            const paramName = patternPart.substring(1);
            match.params[paramName] = decodeURI(hashParts[i]);
        }
    }

    return match;
}

const loadPage = pageMap => {
    const currentHash = window.location.hash;
    console.log(`currentHash ${currentHash}`);

    // Check if the current hash matches any exact matches in the map
    if (pageMap.has(currentHash)) {
        console.log("page exact match");
        const pageFunction = pageMap.get(currentHash);
        pageFunction();
        return;
    }

    // Check if the current hash matches any placeholders in the map
    for (const [key, pageFunction] of pageMap) {
        if (isPlaceholderMatch(key, currentHash)) {
            console.log("page matched by pattern");
            const match = getPlaceholderMatch(key, currentHash);
            pageFunction(match);
            return;
        }
    }
    console.log("no page matched");
}

const fixAndValidateJSON = str => {
    if (!str) {
        return null;
    }

    try {
        let trimmedStr = str.trim();
        let fixedStr = trimmedStr.replace(/\n/g, '');
        fixedStr = fixedStr.replace(/"\s*:\s*"/g, '":"');
        return JSON.parse(fixedStr);
    } catch (error) {
        console.log("error", error);
        return null;
    }
}

const getResponseJson = res => {
    if (!res.ok) {
        showError(res.statusText || 'Network response was not ok');
        throw new Error('Network response was not ok');
    }
    return res.json()
}

const getResponseText = res => {
    if (!res.ok) {
        showError(res.text() || res.statusText || 'Network response was not ok');
        throw new Error('Network response was not ok');
    }
    return res.text()
}

const getResponseAuto = (async res => {
    if (!res.ok) {
        showError(res.statusText || 'Network response was not ok');
        throw new Error('Network response was not ok');
    }
    const contentType = res.headers.get('Content-Type');
    if (contentType && contentType.includes('application/json')) {
        return [JSON.stringify(await res.json(), null, 2), `json`];
    }
    return [await res.text(), `text`];
})

const pageMap = new Map([
    ["#/settings", settingsEdit],
    ['#/services/upload', uploadNewServices],
    ['#/services/:name/ui', serviceSwagger],
    ['#/services/:name', serviceHome],
    ['#/services', () => showServices],

]);

const onLoad = () => {
    resetContents();
    showServices();
    loadPage(pageMap);

    // Get the accordion header and content elements
    const accordionHeader = document.querySelector('.accordion-header');
    const accordionContent = document.querySelector('.accordion-content');
    accordionHeader.addEventListener('click', () => {
        accordionContent.classList.toggle('active');
    });

    document.getElementById('settings-save-button').addEventListener('click', settingsSave);
    document.getElementById('settings-default-save-button').addEventListener('click', settingsRestore);

    document.getElementById('fileupload').addEventListener('change', () => {
        const file = document.getElementById('fileupload').files[0];
        const selectedFilenameElement = document.getElementById('selected-filename');
        selectedFilenameElement.textContent = '';
        if (file) {
            // Display the filename in the element
            selectedFilenameElement.textContent = file.name;
        }
    });
}

window.addEventListener('hashchange', _ => {
    hideMessage();
    loadPage(pageMap);
})
window.addEventListener("DOMContentLoaded", _ => onLoad())
