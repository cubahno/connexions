import * as config from './config.js';
import * as commons from './commons.js';
import * as validators from './validators.js';
import * as navi from "./navi.js";
import * as services from "./services.js";

export const show = match => {
    services.show();

    const {name, ix, action} = match.params;
    const service = name;

    navi.resetContents();
    const editor = commons.getCodeEditor(`context-replacements`, `yaml`);

    console.log(`service home ${service} ix=${ix} action=${action}`);

    fetch(`${config.serviceUrl}/${service}`)
        .then(res => res.json())
        .then(data => {
            if (data.success === false) {
                commons.showSuccessOrError(data.message, data.success);
                return;
            }
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

            for (const { method, path, type, overwrites, contentType } of endpoints) {
                const num = i + 1;
                let icon = ``;
                if (overwrites) {
                    icon = ` <span title="overwrites" class="overwrites-icon">🔁</span>`;
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
                if (type === `fixed` && commons.isResourceEditable(contentType)) {
                    editCell.innerHTML =`<a href="#/services/${service}/${num}/edit">✎</a>`;
                    editCell.className = 'edit-resource';
                    editCell.title = `Edit resource ${method} ${path}`;
                } else {
                    editCell.innerHTML = `&nbsp`;
                }

                row.appendChild(editCell);

                const rmCell = document.createElement('td');
                if (type === `fixed`) {
                    rmCell.innerHTML = `✖`;
                    rmCell.className = 'remove-resource';
                    rmCell.title = `Remove resource ${method} ${path}`;
                    rmCell.onclick = () => {
                        if (confirm(`Are you sure you want to remove resource ${method} ${path}?\nAll files will be deleted!`)) {
                            fetch(`${config.serviceUrl}/${service}/${num-1}`, {
                                method: 'DELETE'
                            })
                                .then(res => res.json())
                                .then(res => {
                                    const hashParams = location.hash.split(`/`);
                                    const service = hashParams[2];
                                    if (res.success) {
                                        location.hash = `#/services/${service}`;
                                        location.reload(true);
                                    }
                                    window.setTimeout(_ => {
                                        commons.showSuccessOrError(res.message, res.success);
                                    }, 100);
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
                    edit(service, ix - 1);
                } else if (action === `result`) {
                    generateResult(service, ix - 1, endpoints[ix - 1].path, endpoints[ix - 1].method);
                }
            }
        });
}

export const generateResult = (service, ix, path, method) => {
    console.log(`loadResource: ${method} /${service}${path}`);

    const onDone = () => {
        config.generatorCont.style.display = 'block';
        config.resourceRefreshBtn.onclick = () => generateResult(service, ix, path, method);
        config.resourceRefreshBtn.style.display = 'block';
    }
    commons.hideMessage();
    let replacements = null;
    const replacementsEditor = commons.getCodeEditor(`context-replacements`, `yaml`);
    const yamlContent = replacementsEditor.getValue();
    if (yamlContent) {
        const yamlObject = jsyaml.load(yamlContent);
        const jsonContent = JSON.stringify(yamlObject, null, 2);
        replacements = validators.fixAndValidateJSON(jsonContent);
    }
    document.getElementById(`resource-edit-container`).style.display = 'none';

    fetch(`${config.serviceUrl}/${service}/${ix}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            replacements: replacements,
        }),
    })
        .then(async res => {
            if (res.status === 500) {
                commons.showError(await res.text() || `Internal server error`);
                return;
            }
            return res;
        })
        .then(res => res && res.json())
        .then(res => {
            if (!res) {
                return;
            }
            const reqPath = res["request"]["path"];
            if (reqPath) {
                document.getElementById('request-path').innerHTML = reqPath;
                document.getElementById('request-path-container').style.display = 'block';
            }
            const reqContentType = res["request"]["contentType"];

            if (method.toLowerCase() === 'get') {
                // document.getElementById('request-body-container').style.display = 'none';
            } else {
                let formattedBody = ``;
                const reqBody = res["request"]["body"];
                if (reqBody !== undefined && reqBody.length > 0 && reqContentType === `application/json`) {
                    const jsonObject = JSON.parse(reqBody);
                    formattedBody = JSON.stringify(jsonObject, null, 2);
                }

                if (formattedBody.length) {
                    const reqView = commons.getCodeEditor(`request-body`, `json`);
                    reqView.setValue(formattedBody);
                    reqView.clearSelection();
                    reqView.setReadOnly(true);
                    // document.getElementById('request-body-container').style.display = 'block';
                }
            }

            const curlBlock = document.getElementById('example-curl');
            const baseUrl = `${window.location.protocol}//${window.location.host}`;
            curlBlock.textContent = `curl --request ${method} \\\n'${baseUrl}${reqPath}'`;
            if (reqContentType) {
                curlBlock.textContent += ` \\\n--header 'Content-Type: ${reqContentType}'`
            }
            const exampleCurl = res.request?.examples?.curl;
            if (exampleCurl) {
                curlBlock.textContent += ` \\\n${exampleCurl}`;
            }

            const resContent = res.response.content;
            let decodedBytes = ``;
            if (resContent) {
                decodedBytes = atob(resContent);
            }

            let resView = ``;
            if (decodedBytes && commons.isResourceEditable(res["response"]["contentType"])) {
                try {
                    const jsonObject = JSON.parse(decodedBytes);
                    resView = JSON.stringify(jsonObject, null, 2);
                } catch (error) {
                    resView = decodedBytes;
                }
            } else if (decodedBytes) {
                // resView = `<a href="${baseUrl}${res.request.path}" target="_blank"><i class="fa-solid fa-up-right-from-square"></i> View</a>`;
            }

            const respView = commons.getCodeEditor(`response-body`, `json`);
            respView.setValue(resView);
            respView.clearSelection();
            respView.setReadOnly(true);
            // document.getElementById('response-body').innerHTML = resView;
            document.getElementById('response-body-container').style.display = 'block';

            const copyCodeElement = document.querySelector(".copy-code");
            const originalCopyIcon = `<i class="fa-solid fa-copy"></i> Copy`;
            copyCodeElement.addEventListener("click", () => {
                const codeText = curlBlock.textContent;
                navigator.clipboard.writeText(codeText).then(() => {
                    console.log("Code copied to clipboard!");

                    copyCodeElement.innerHTML = `<i class="fas fa-check"></i> Copied!`;
                    setTimeout(() => {
                        copyCodeElement.innerHTML = originalCopyIcon;
                    }, 2000);
                }).catch((error) => {
                    console.error("Failed to copy code:", error);
                });
            });

        }).then(onDone);
}

const edit = (service, ix) => {
    console.log(`editResource: #${ix+1}`);
    const cont = document.getElementById('resource-edit-container');
    navi.applySelection(`service-${service}`, 'selected-service');
    document.getElementById(`generator-container`).style.display = 'none';
    const editor = commons.getEditorForm(`res-selected-text-response`, `res-response-content-type`);

    cont.style.display = 'block';
    fetch(`${config.serviceUrl}/${service}/${ix}`)
        .then(res => res.json())
        .then(res => {
            document.getElementById(`res-endpoint-path`).value = res.path;
            document.getElementById(`res-endpoint-method`).value = res.method;
            document.getElementById(`res-response-content-type`).value = res.contentType;

            if (!commons.isResourceEditable(res.contentType)) {
                console.log(`resource ${res.contentType} is not editable`);
                return;
            }

            const mode = commons.getCodeEditorMode(res.extension);
            console.log(`editor mode: ${mode}`);
            editor.setValue(res.content);
            editor.setOptions({
                mode: `ace/mode/${mode}`,
            })
            editor.clearSelection();

            document.getElementById(`res-response-content-type`).value = mode;
            document.getElementById('res-save-button').addEventListener('click', async event => {
                event.preventDefault();
                await services.saveWithoutFile(cont)
            });
        });
}
