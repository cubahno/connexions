import * as config from './config.js';
import * as commons from './commons.js';
import * as navi from "./navi.js";
import * as services from './services.js';

// add new service
export const newForm = () => {
    console.log(`add new service`);
    navi.applySelection(`n/a`, 'selected-service');
    navi.resetContents();
    services.show();

    config.contentTitleEl.innerHTML = `Add new service to the list`;

    navi.setupTabbedContent(`new-service-tab-container`);

    config.serviceCreateContainer.style.display = 'block';

    const onSubmit = formId => {
        const formCont = document.getElementById(formId);
        const submitBtn = formCont.querySelector('.button');
        submitBtn.addEventListener('click', async event => {
            event.preventDefault();
            await saveFormWithFile(formCont);
        });

        const textResp = formCont.querySelector('.selected-text-response');
        const ctEl = formCont.querySelector('.response-content-type');
        const editor = commons.getEditorForm(textResp.id, ctEl.id);

        const fileEl = formCont.querySelector('[type="file"]');
        fileEl.addEventListener('change', event => {
            const file = event.target.files[0];
            const selectedFilenameElement = formCont.querySelector('.selected-filename');
            selectedFilenameElement.textContent = '';
            if (file) {
                selectedFilenameElement.textContent = file.name;
                editor.setValue(``);
            }
        });
    }

    onSubmit(`fixed-service-form`);
    onSubmit(`openapi-service-form`);
}

export const show = (selected = '') => {
    config.servicesLink.className = `menu-link active`;
    config.contextsLink.className = `menu-link inactive`;

    const addNewCont = document.getElementById('add-new-service-cont').innerHTML;
    config.serviceTable.innerHTML = '';
    config.contextTable.style.display = 'none';
    config.contextEditContainer.style.display = 'none';

    console.log("loading service list");

    const t = document.createElement(`table`);
    const newRow = t.insertRow();
    newRow.innerHTML = addNewCont;

    fetch(config.serviceUrl)
        .then(res => res.json())
        .then(data => {
            const services = data['items'];

            let i = 0;
            for (let { name, openApiResources } of services) {
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
                const hasOpenApi = openApiResources && openApiResources.length > 0;
                if (hasOpenApi) {
                    swaggerLink = `<a href="#/services/${nameLink}/ui"><img class="swagger-icon" src="icons/swagger.svg"></a>`;
                }
                swaggerCell.innerHTML = swaggerLink;
                row.appendChild(swaggerCell);

                const rmCell = document.createElement('td');
                rmCell.innerHTML = `✖`;
                rmCell.id = `remove-service-${name}`;
                rmCell.className = `remove-service ${name}`;
                rmCell.title = `Remove service ${name}`;

                row.appendChild(rmCell);

                t.appendChild(row);
                i += 1;
            }

            config.serviceTable.innerHTML = t.innerHTML;
            config.serviceTable.style.display = 'block';
            if (selected !== ``) {
                navi.applySelection(`service-${selected}`, 'selected-service');
            }

            const elements = document.querySelectorAll(`.remove-service`);
            elements.forEach(element => {
                element.addEventListener(`click`, event => {
                    const serviceName = event.target.id.replace(`remove-service-`, ``);
                    let nameLink = serviceName;
                    if (serviceName === ``) {
                        name = "/"
                        nameLink = `.root`
                    }

                    if (confirm(`Are you sure you want to remove service ${serviceName}?\nAll files will be deleted!`)) {
                        fetch(`${config.serviceUrl}/${nameLink}`, {
                            method: 'DELETE'
                        })
                            .then(res => res.json())
                            .then(res => {
                                if (res.success) {
                                    navi.resetContents()
                                }
                                window.setTimeout(_ => {
                                    commons.showSuccessOrError(res.message, res.success);
                                }, 300)
                                location.hash = `#/services`;
                            });
                    }
                });
            });
        });
}

export const showSwagger = match => {
    services.show();

    const service = match.params.name;
    show(service);
    navi.resetContents();

    console.log(`Show swagger for ${service}`);

    config.contentTitleEl.innerHTML = `${service} Swagger / OpenAPI`;

    config.iframeContents.src = `${config.homeUrl}/swaggerui?specUrl=${appConfig.serviceUrl}/${service}/spec`;
    config.iframeContents.style.display = 'block';
}

export async function saveFormWithFile(container) {
    let formData = new FormData();

    const isOpenApi = container.querySelector('input[name="is_openapi"]').value === '1';
    const path = container.querySelector('input[name="path"]').value.trim();
    const url = container.querySelector('input[name="url"]').value.trim();
    const response = commons.getCodeEditor(`selected-text-response`, `json`).getValue();

    let method = `GET`;
    const methodEl = container.querySelector('select[name="method"]');
    if (methodEl) {
        method = methodEl.value;
    }

    const contentMap = {
        yml: `yaml`,
        markdown: `md`,
        text: `txt`,
    }
    const contentTypeVal = container.querySelector('select[name="content_type"]').value;
    const contentType = contentMap.hasOwnProperty(contentTypeVal) ? contentMap[contentTypeVal] : contentTypeVal;

    const fileInput = container.querySelector('[type="file"]');
    if (fileInput && fileInput.files.length > 0) {
        formData.append("file", fileInput.files[0]);
    }

    formData.append("response", response);
    formData.append("contentType", contentType);
    formData.append("method", method);
    formData.append("isOpenApi", isOpenApi.toString());
    formData.append("path", path);
    formData.append("url", url);

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
        if (res.success) {
            const ix = res.id + 1;
            const hashParams = location.hash.split(`/`);
            const service = hashParams[2];

            console.log(`reloading service ${service} resources`);
            location.hash = `#/services/${service}/${ix}/edit`;
            location.reload(true);
        } else {
            commons.showError(res.message)
        }
    });
}

async function save(formData) {
    config.messageCont.textContent = '';
    return fetch(config.serviceUrl, {
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
