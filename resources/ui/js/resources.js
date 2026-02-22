import * as config from './config.js';
import * as commons from './commons.js';
import * as validators from './validators.js';
import * as navi from "./navi.js";
import * as services from "./services.js";

export const show = match => {
    services.show();

    const {name} = match.params;
    const service = name;

    // Get ix from query parameter
    // Try both window.location.search (for ?ix=1 after hash) and hash query string (for #/path?ix=1)
    let ix = null;

    // First try from actual URL query string (http://...?ix=1)
    const urlParams = new URLSearchParams(window.location.search);
    ix = urlParams.get('ix');

    // If not found, try from hash query string (#/path?ix=1)
    if (!ix) {
        const hashParts = window.location.hash.split('?');
        if (hashParts.length > 1) {
            const hashParams = new URLSearchParams(hashParts[1]);
            ix = hashParams.get('ix');
        }
    }

    const serviceResourcesUrl = `${config.serviceUrl}/${service}`;

    // If ix is provided, skip fetching routes and go directly to generate
    if (ix !== null && ix !== undefined) {
        // Get endpoint info from the table (which should already be populated)
        const row = document.getElementById(`resource-${ix}`);
        if (row) {
            // Extract path and method from the table row
            const pathCell = row.querySelector('.fixed-resource-path a');
            const methodCell = row.querySelector('.fixed-resource-method');

            if (pathCell && methodCell) {
                const path = pathCell.textContent;
                const method = methodCell.textContent.toLowerCase();

                navi.applySelection(`resource-${ix}`, 'selected-resource');
                generateResult(service, ix - 1, path, method);
                return;
            }
        }
        // If row not found, fall through to fetch routes
    }

    navi.resetContents();

    fetch(serviceResourcesUrl)
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
            const mapped = {};

            for (const { method, path, contentType } of endpoints) {
                const num = i + 1;
                const row = document.createElement('tr');
                row.id = `resource-${num}`;
                row.style.cursor = 'pointer';
                row.onclick = () => { window.location.hash = `#/services/${service}?ix=${num}`; };

                const cell1 = document.createElement('td');
                cell1.textContent = `${num}`;
                cell1.className = 'fixed-resource-num';
                row.appendChild(cell1);

                const methodCell = document.createElement('td');
                methodCell.innerHTML = `${method.toUpperCase()}`;
                methodCell.className = `fixed-resource-method ${method.toLowerCase()}`;
                row.appendChild(methodCell);

                const pathCell = document.createElement('td');
                pathCell.innerHTML = `<span>${path}</span>`;
                pathCell.className = `fixed-resource-path`;
                pathCell.title = path;
                row.appendChild(pathCell);

                table.appendChild(row);
                i += 1;
            }
            config.fixedServiceContainer.style.display = 'block';

            // If ix is present, generate the resource
            if (ix !== null && ix !== undefined) {
                navi.applySelection(`resource-${ix}`, 'selected-resource');
                generateResult(service, ix - 1, endpoints[ix - 1].path, endpoints[ix - 1].method);
            }
        });
}

export const generateResult = (service, ix, path, method) => {
    const onDone = () => {
        config.generatorCont.style.display = 'block';
        config.resourceRefreshBtn.onclick = () => generateResult(service, ix, path, method);
        config.resourceRefreshBtn.style.display = 'inline';
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

    // Use .root as-is for the generate endpoint (backend will convert it)
    const generateUrl = `${config.serviceUrl}/${service}`;
    const payload = {
        path: path,
        method: method,
        context: replacements,
    };

    fetch(generateUrl, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
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
                console.log(`No response`);
                return;
            }

            // Clear previous data first
            document.getElementById('request-path').innerHTML = '';
            document.getElementById('request-path-container').style.display = 'none';
            document.getElementById('request-body-container').style.display = 'none';
            document.getElementById('response-body-container').style.display = 'none';

            let reqPath = res["path"];
            if (!reqPath) {
                commons.showError('No path returned from server. The request may have failed.');
                return;
            }

            // Decode URL-encoded path for better readability
            document.getElementById('request-path').innerHTML = decodeURIComponent(reqPath);
            document.getElementById('request-path-container').style.display = 'block';
            const reqContentType = res["contentType"];
            const reqBody = res["body"];
            const reqHeaders = res["headers"] || {};

            let formattedBody = ``;
            let reqBodyString = null;
            if (reqBody !== undefined && reqBody !== null) {
                if (reqContentType === `application/json`) {
                    formattedBody = JSON.stringify(reqBody, null, 2);
                    reqBodyString = JSON.stringify(reqBody);
                    console.log(`formattedBody:`, formattedBody);
                }
            }

            if (formattedBody.length) {
                document.getElementById('request-body-container').style.display = 'block';
                const reqView = commons.getCodeEditor(`request-body`, `json`);
                reqView.setValue(formattedBody);
                reqView.clearSelection();
                reqView.setReadOnly(true);
            }

            const curlBlock = document.getElementById('example-curl');
            const baseUrl = `${window.location.protocol}//${window.location.host}`;

            // Use service name for the URL prefix, converting .root back to empty string
            const servicePrefix = service === '.root' ? '' : `/${service}`;
            curlBlock.textContent = `curl --request ${method} \\\n'${baseUrl}${servicePrefix}${reqPath}'`;
            if (reqContentType) {
                curlBlock.textContent += ` \\\n--header 'Content-Type: ${reqContentType}'`
            }
            // Add generated headers to cURL
            for (const [headerName, headerValue] of Object.entries(reqHeaders)) {
                curlBlock.textContent += ` \\\n--header '${headerName}: ${headerValue}'`;
            }
            // Add request body to cURL
            if (reqBodyString && method.toLowerCase() !== 'get') {
                curlBlock.textContent += ` \\\n--data '${reqBodyString.replace(/'/g, "\\'")}'`;
            }
            const exampleCurl = res.request?.examples?.curl;
            if (exampleCurl) {
                curlBlock.textContent += ` \\\n${exampleCurl}`;
            }

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

            // Make actual API call to get response
            if (reqPath) {
                // Convert .root back to empty string for actual API requests
                const apiService = service === '.root' ? '' : service;
                const apiUrl = apiService ? `${baseUrl}/${apiService}${reqPath}` : `${baseUrl}${reqPath}`;
                const fetchOptions = {
                    method: method.toUpperCase(),
                    headers: { ...reqHeaders }
                };

                if (reqContentType) {
                    fetchOptions.headers['Content-Type'] = reqContentType;
                }

                if (reqBodyString && method.toLowerCase() !== 'get') {
                    fetchOptions.body = reqBodyString;
                }

                fetch(apiUrl, fetchOptions)
                .then(response => {
                    const responseContentType = response.headers.get('Content-Type');
                    return response.text().then(text => ({
                        status: response.status,
                        contentType: responseContentType,
                        body: text
                    }));
                })
                .then(responseData => {
                    console.log('API Response:', responseData);

                    // Display response in code editor
                    let formattedResponse = responseData.body;
                    if (responseData.contentType && responseData.contentType.includes('application/json')) {
                        try {
                            // Check if body is already an object or a string
                            const jsonObject = typeof responseData.body === 'string'
                                ? JSON.parse(responseData.body)
                                : responseData.body;
                            formattedResponse = JSON.stringify(jsonObject, null, 2);
                        } catch (e) {
                            console.error('Failed to parse JSON response:', e);
                        }
                    }

                    document.getElementById('response-body-container').style.display = 'block';
                    const responseView = commons.getCodeEditor(`response-body`, `json`);
                    responseView.setValue(formattedResponse);
                    responseView.clearSelection();
                    responseView.setReadOnly(true);
                })
                .catch(error => {
                    console.error('API call failed:', error);
                    const responseView = commons.getCodeEditor(`response-body`, `json`);
                    responseView.setValue(`Error: ${error.message}`);
                    responseView.clearSelection();
                    responseView.setReadOnly(true);
                });
            }

        }).then(onDone);
}
