import * as config from './config.js';
import * as commons from './commons.js';
import * as validators from './validators.js';
import * as navi from "./navi.js";


export const show = match => {
    const {name, ix, action} = match.params;
    const service = name;

    navi.resetContents();
    const editor = commons.getCodeEditor(`replacements2`, `json`);
    editor.setValue(`{\n\t\n}`);
    editor.clearSelection();

    console.log(`service home ${service} ix=${ix} action=${action}`);

    fetch(`${config.serviceUrl}/${service}`)
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
                            fetch(`${config.serviceUrl}/${service}/resources/${method.toLowerCase()}?path=${path}`, {
                                method: 'DELETE'
                            })
                                .then(res => res.json())
                                .then(res => {
                                    commons.showSuccessOrError(res.message, res.success)
                                    show(match);
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
                    edit(service, endpoints[ix - 1].method, endpoints[ix - 1].path);
                } else if (action === `result`) {
                    generateResult(service, endpoints[ix - 1].path, endpoints[ix - 1].method, endpoints[ix - 1].type === `openapi`);
                }
            }
        });
}

export const generateResult = (service, path, method, isOpenApi) => {
    console.log(`loadResource: ${method} /${service}${path}`);

    const onDone = () => {
        config.generatorCont.style.display = 'block';
        config.resourceRefreshBtn.onclick = () => generateResult(service, path, method, isOpenApi);
        config.resourceRefreshBtn.style.display = 'block';
    }
    commons.hideMessage();
    document.getElementById(`resource-edit-container`).style.display = 'none';

    let replacements = validators.fixAndValidateJSON(document.getElementById('replacements').value.trim());
    fetch(`${config.serviceUrl}/${service}`, {
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

const edit = (service, method, path) => {
    console.log(`editResource: ${method} /${service}${path}`);
    const cont = document.getElementById('resource-edit-container');
    navi.applySelection(`service-${service}`, 'selected-service');
    document.getElementById(`generator-container`).style.display = 'none';
    const editor = commons.getEditorForm(`res-selected-text-response`, `res-response-content-type`);

    cont.style.display = 'block';
    fetch(`${config.serviceUrl}/${service}/resources/${method.toLowerCase()}?path=${path}`)
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
