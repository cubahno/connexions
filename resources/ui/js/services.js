import * as config from './config.js';
import * as commons from './commons.js';
import * as navi from "./navi.js";
import * as resources from "./resources.js";

export const newForm = () => {
    console.log(`add new service`);
    navi.applySelection(`n/a`, 'selected-service');
    navi.resetContents();
    commons.getEditorForm('selected-text-response', 'response-content-type');
    config.contentTitleEl.innerHTML = `Add new service to the list`;

    config.servicesUploadForm.style.display = 'block';
}

export const show = () => {
    config.serviceTable.innerHTML = '';
    console.log("loading service list");

    fetch(`${config.url}/services`)
        .then(res => res.json())
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
                                show();
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

export const showSwagger = match => {
    const service = match.params.name;
    navi.applySelection(`service-${service}`, 'selected-service');
    navi.resetContents();

    console.log(`Show swagger for ${service}`);

    config.contentTitleEl.innerHTML = `${service} Swagger / OpenAPI`;

    config.iframeContents.src = `${config.url}/ui/swaggerui?specUrl=/services/${service}/spec`;
    config.iframeContents.style.display = 'block';
}

export async function saveWithFile(event) {
    event.preventDefault();
    let formData = new FormData();

    const isOpenApi = document.querySelector('input[name="is_openapi"]:checked').value === '1';
    const method = document.getElementById('endpoint-method').value.trim();
    const path = document.getElementById('endpoint-path').value.trim();
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

    await save(formData);
}

export async function saveWithoutFile(event) {
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

    await save(formData).then(res => {
        console.log(res);
        if (res.success) {
            const hashParams = location.hash.split(`/`);
            const service = hashParams[2];
            const ix = hashParams[3];

            console.log(`reloading service ${service} resources`);
            resources.show({params: {name: service, ix: ix, action: `edit`}});
        }
    });
}

async function save(formData) {
    config.messageCont.textContent = '';
    return fetch('/services', {
        method: "POST",
        body: formData,
    }).then(res => res.json()).then(res => {
        commons.showSuccessOrError(res.message, res.success);

        if (res.success) {
            show();
        }
        return res;
    });
}
