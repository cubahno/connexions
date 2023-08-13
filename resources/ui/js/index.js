import * as config from './config.js';
import * as settings from './settings.js';
import * as commons from './commons.js';
import * as navi from './navi.js';

const showServices = () => {
    config.serviceTable.innerHTML = '';
    console.log("loading service list");

    fetch(`${config.url}/services`)
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
                    nameLink = `.root`
                }
                const svcNameCell = document.createElement('td');
                svcNameCell.innerHTML = `<a href="#/services/${nameLink}">${name}</a>`;
                row.appendChild(svcNameCell);

                const swaggerCell = document.createElement('td');
                let swaggerLink = '&nbsp;';
                if (isOpenApi && name !==`/`) {
                    swaggerLink = `<a href="#/services/${name}/ui"><img class="swagger-icon" src="/ui/icons/swagger.svg"></a>`;
                }
                swaggerCell.innerHTML = swaggerLink;
                row.appendChild(swaggerCell);

                const rmCell = document.createElement('td');
                rmCell.innerHTML = `✖`;
                rmCell.className = 'remove-service';
                rmCell.title = `Remove service ${name}`;
                rmCell.onclick = () => {
                    if (confirm(`Are you sure you want to remove service ${name}?\nAll files will be deleted!`)) {
                        fetch(`${config.url}/services/${nameLink}`, {
                            method: 'DELETE'
                        })
                            .then(res => res.json())
                            .then(res => {
                                commons.showSuccessOrError(res.message, res.success)
                                showServices();
                            });
                    }
                }
                row.appendChild(rmCell);

                config.serviceTable.appendChild(row);
                i += 1;
            }

            config.serviceTable.style.display = 'block';
        });
}

const serviceHome = match => {
    const {name, ix, action} = match.params;
    const service = name;

    navi.resetContents();
    const editor = commons.getCodeEditor(`replacements2`, `json`);
    editor.setValue(`{\n\t\n}`);
    editor.clearSelection();

    console.log(`service home ${service} ix=${ix} action=${action}`);

    fetch(`${config.url}/services/${service}`)
        .then(res => res.json())
        .then(data => {
            navi.applySelection(`service-${service}`, 'selected-service');

            const endpoints = data['endpoints'];
            let name = service;
            if (name === `.root`) {
                name = `Root level`
            } else {
                name = `/${name}`
            }
            config.contentTitleEl.innerHTML = `${name} resources`;

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
                pathCell.innerHTML = `<a href="#/services/${service}/${num}/result">${path}</a>`;
                pathCell.className = `fixed-resource-path`;
                row.appendChild(pathCell);

                const editCell = document.createElement('td');
                if (type === `overwrite`) {
                    editCell.innerHTML =`<a href="#/services/${service}/${num}/edit">✎</a>`;
                    editCell.className = 'edit-resource';
                    editCell.title = `Edit resource ${method} ${path}`;
                } else {
                    editCell.innerHTML = `&nbsp`;
                }

                row.appendChild(editCell);

                const rmCell = document.createElement('td');
                if (type === `overwrite`) {
                    //rmCell.innerHTML = `🔁`;
                    rmCell.innerHTML = `✖`;
                    rmCell.className = 'remove-resource';
                    rmCell.title = `Remove resource ${method} ${path}`;
                    rmCell.onclick = () => {
                        if (confirm(`Are you sure you want to remove resource ${method} ${path}?\nAll files will be deleted!`)) {
                            fetch(`${config.url}/services/${service}/resources/${method.toLowerCase()}?path=${path}`, {
                                method: 'DELETE'
                            })
                                .then(res => res.json())
                                .then(res => {
                                    commons.showSuccessOrError(res.message, res.success)
                                    serviceHome(match);
                                });
                        }
                    }
                } else {
                    rmCell.innerHTML = `&nbsp`;
                }

                row.appendChild(rmCell);

                table.appendChild(row);
                i += 1;
            }
            config.fixedServiceContainer.style.display = 'block';

            // onLoad

            if (ix !== undefined) {
                navi.applySelection(`resource-${ix}`, 'selected-resource');
                if (action === `edit`) {
                    editResourceLoad(service, endpoints[ix - 1].method, endpoints[ix - 1].path);
                } else if (action === `result`) {
                    loadResource(service, endpoints[ix - 1].path, endpoints[ix - 1].method, endpoints[ix - 1].type === `openapi`);
                }
            }
        });
}

const loadResource = (service, path, method, isOpenApi) => {
    console.log(`loadResource: ${method} /${service}${path}`);

    const onDone = () => {
        config.generatorCont.style.display = 'block';
        config.resourceRefreshBtn.onclick = () => loadResource(service, path, method, isOpenApi);
        config.resourceRefreshBtn.style.display = 'block';
    }
    commons.hideMessage();
    document.getElementById(`resource-edit-container`).style.display = 'none';

    let replacements = fixAndValidateJSON(document.getElementById('replacements').value.trim());
    fetch(`${config.url}/services/${service}`, {
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
        .then(res => res.json())
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

const editResourceLoad = (service, method, path) => {
    console.log(`editResource: ${method} /${service}${path}`);
    const cont = document.getElementById('resource-edit-container');
    navi.applySelection(`service-${service}`, 'selected-service');
    document.getElementById(`generator-container`).style.display = 'none';
    const editor = showResponseEditForm(`res-selected-text-response`, `res-response-content-type`);

    cont.style.display = 'block';
    fetch(`${config.url}/services/${service}/resources/${method.toLowerCase()}?path=${path}`)
        .then(res => res.json())
        .then(res => {
            document.getElementById(`res-endpoint-path`).value = res.path;
            document.getElementById(`res-endpoint-method`).value = res.method;
            document.getElementById(`res-response-content-type`).value = res.contentType;

            const mode = commons.getCodeEditorMode(res.contentType);
            editor.setValue(res.content);
            editor.setOptions({
                mode: `ace/mode/${mode}`,
            })
            editor.clearSelection();
        });
}

const serviceSwagger = match => {
    const service = match.params.name;
    navi.applySelection(`service-${service}`, 'selected-service');
    navi.resetContents();

    console.log(`Show swagger for ${service}`);

    config.contentTitleEl.innerHTML = `${service} Swagger / OpenAPI`;

    config.iframeContents.src = `${config.url}/ui/swaggerui?specUrl=/services/${service}/spec`;
    config.iframeContents.style.display = 'block';
}

const uploadNewServices = () => {
    console.log(`add new service`);
    navi.applySelection(`n/a`, 'selected-service');
    navi.resetContents();
    showResponseEditForm('selected-text-response', 'response-content-type');
    config.contentTitleEl.innerHTML = `Add new service to the list`;

    config.servicesUploadForm.style.display = 'block';
}

export async function saveResource(event) {
    event.preventDefault();
    let formData = new FormData();

    const isOpenApi = document.querySelector('input[name="is_openapi"]:checked').value === '1';
    const method = document.getElementById('endpoint-method').value.trim();
    let path = '';
    if (!isOpenApi) {
        path = document.getElementById('endpoint-path').value.trim();
    }
    const response = commons.getCodeEditor(`selected-text-response`, `json`).getValue();

    const contentMap = {
        yml: `yaml`,
        markdown: `md`,
        text: `txt`,
    }
    const ctValue = config.responseContentTypeEl.value;
    const contentType = contentMap.hasOwnProperty(ctValue) ? contentMap[ctValue] : ctValue;

    formData.append("file", config.fileUploadBtn.files[0]);
    formData.append("response", response);
    formData.append("contentType", contentType);
    formData.append("method", method);
    formData.append("isOpenApi", isOpenApi.toString());
    formData.append("path", path);

    await submitResourceSave(formData);
}

export async function updateResource(event) {
    event.preventDefault();
    let formData = new FormData();

    const method = document.getElementById('res-endpoint-method').value.trim();
    const path = document.getElementById('res-endpoint-path').value.trim();
    const response = commons.getCodeEditor(`res-selected-text-response`, `json`).getValue();

    const contentMap = {
        yml: `yaml`,
        markdown: `md`,
        text: `txt`,
    }
    const ctValue = config.responseContentTypeEl.value;
    const contentType = contentMap.hasOwnProperty(ctValue) ? contentMap[ctValue] : ctValue;

    formData.append("response", response);
    formData.append("contentType", contentType);
    formData.append("method", method);
    formData.append("path", path);

    await submitResourceSave(formData).then(res => {
        console.log(res);
        if (res.success) {
            const hashParams = location.hash.split(`/`);
            const service = hashParams[2];
            const ix = hashParams[3];
            alert(service);
            console.log(`reloading service ${service} resources`);
            serviceHome({params: {name: service, ix: ix, action: `edit`}});
        }
    });
}

async function submitResourceSave(formData) {
    config.messageCont.textContent = '';
    return fetch('/services', {
        method: "POST",
        body: formData,
    }).then(res => res.json()).then(res => {
        commons.showSuccessOrError(res.message, res.success);

        if (res.success) {
            showServices();
        }
        return res;
    });
}

const settingsEdit = () => {
    console.log(`settings edit`);
    navi.applySelection(`n/a`, 'selected-service');
    navi.resetContents();
    config.contentTitleEl.innerHTML = `Edit Settings`;

    const editor = commons.getCodeEditor(`code-editor`, `yaml`);

    fetch(`${config.url}/settings`)
        .then(getResponseText)
        .then(res => {
            editor.setValue(res);
            editor.clearSelection();
        })

    config.settingsEditor.style.display = 'block';
}

const settingsSave = () => {
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

const settingsRestore = () => {
    fetch('/settings', {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
    }).then(res => res.json()).then(res => {
        commons.showSuccessOrError(res.message, res.success);
        showServices();
        settingsEdit();
    });
}

const showResponseEditForm = (editorId, typeId) => {
    console.log(`response edit in ${editorId}`);

    const editor = commons.getCodeEditor(editorId, `json`);
    editor.setValue(``);
    editor.clearSelection();

    document.getElementById(typeId).addEventListener(`change`, el => {
        const value = el.target.value;
        editor.setOptions({
            mode: `ace/mode/${value}`,
        })
    })
    return editor;
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
        commons.showError(res.statusText || 'Network response was not ok');
        throw new Error('Network response was not ok');
    }
    return res.json()
}

const getResponseText = res => {
    if (!res.ok) {
        commons.showError(res.text() || res.statusText || 'Network response was not ok');
        throw new Error('Network response was not ok');
    }
    return res.text()
}

const pageMap = new Map([
    ["#/settings", settingsEdit],
    ['#/services/upload', uploadNewServices],
    ['#/services/:name/ui', serviceSwagger],
    ['#/services/:name', serviceHome],
    ['#/services/:name/:ix/:action', serviceHome],
    ['#/services', () => showServices],

]);

const onLoad = () => {
    navi.resetContents();
    showServices();
    navi.loadPage(pageMap);

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
            commons.getCodeEditor(`selected-text-response`, `yaml`).setValue(``);
        }
    });
    document.getElementById('upload-button').addEventListener('click', saveResource);
    document.getElementById('res-upload-button').addEventListener('click', updateResource);
}

window.addEventListener('hashchange', _ => {
    commons.hideMessage();
    navi.loadPage(pageMap);
})
window.addEventListener("DOMContentLoaded", _ => onLoad())
