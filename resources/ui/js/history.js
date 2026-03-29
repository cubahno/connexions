import * as config from './config.js';
import * as commons from './commons.js';
import * as navi from './navi.js';
import * as services from './services.js';

// Go's []byte JSON-marshals as a base64 string. Decode it back to text,
// then detect whether the result is valid JSON for syntax highlighting.
const decodeBody = (raw) => {
    let text;
    try {
        text = atob(raw);
    } catch {
        text = String(raw);
    }
    let mode = 'text';
    try {
        text = JSON.stringify(JSON.parse(text), null, 2);
        mode = 'json';
    } catch {}
    return {text, mode};
};

const formatTime = (dateStr) => {
    const now = new Date();
    const date = new Date(dateStr);
    const diffSec = Math.floor((now - date) / 1000);
    if (diffSec < 60) return `${diffSec}s ago`;
    if (diffSec < 300) return `${Math.floor(diffSec / 60)}m ago`;
    return date.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit', second: '2-digit'});
};

const statusClass = (code) => {
    if (code >= 200 && code < 300) return 'status-2xx';
    if (code >= 400 && code < 500) return 'status-4xx';
    if (code >= 500) return 'status-5xx';
    return '';
};

const showTabs = (service) => {
    config.serviceTabs.style.display = 'flex';
    config.tabResources.href = `#/services/${service}`;
    config.tabHistory.href = `#/history/${service}`;
    config.tabResources.classList.remove('active');
    config.tabHistory.classList.add('active');
};

const showDetail = (entry) => {
    const detail = document.getElementById('history-detail');
    detail.style.display = 'block';

    const title = document.getElementById('history-detail-title');
    const req = entry.request;
    const resp = entry.response;
    title.textContent = req ? `${req.method} ${req.url}` : 'Detail';

    // Request headers
    const reqHeadersBody = document.getElementById('history-req-headers-body');
    reqHeadersBody.innerHTML = '';
    if (req && req.headers) {
        for (const h of req.headers) {
            const row = document.createElement('tr');
            const colonIdx = h.indexOf(': ');
            const name = colonIdx >= 0 ? h.substring(0, colonIdx) : h;
            const value = colonIdx >= 0 ? h.substring(colonIdx + 2) : '';
            const nameCell = document.createElement('td');
            nameCell.textContent = name;
            const valueCell = document.createElement('td');
            valueCell.textContent = value;
            row.append(nameCell, valueCell);
            reqHeadersBody.appendChild(row);
        }
    }

    // Request body
    const reqBodyContainer = document.getElementById('history-req-body-container');
    if (req && req.body && req.body.length > 0) {
        reqBodyContainer.style.display = 'block';
        const {text, mode} = decodeBody(req.body);
        const editor = commons.getCodeEditor('history-req-body', mode, {maxLines: Infinity});
        editor.setValue(text);
        editor.clearSelection();
        editor.setReadOnly(true);
    } else {
        reqBodyContainer.style.display = 'none';
    }

    // Response headers
    const respHeadersBody = document.getElementById('history-resp-headers-body');
    respHeadersBody.innerHTML = '';
    if (resp && resp.headers) {
        for (const h of resp.headers) {
            const row = document.createElement('tr');
            const colonIdx = h.indexOf(': ');
            const name = colonIdx >= 0 ? h.substring(0, colonIdx) : h;
            const value = colonIdx >= 0 ? h.substring(colonIdx + 2) : '';
            const nameCell = document.createElement('td');
            nameCell.textContent = name;
            const valueCell = document.createElement('td');
            valueCell.textContent = value;
            row.append(nameCell, valueCell);
            respHeadersBody.appendChild(row);
        }
    }

    // Response body
    const respBodyContainer = document.getElementById('history-resp-body-container');
    if (resp && resp.body && resp.body.length > 0) {
        respBodyContainer.style.display = 'block';
        const {text, mode} = decodeBody(resp.body);
        const editor = commons.getCodeEditor('history-resp-body', mode, {maxLines: Infinity});
        editor.setValue(text);
        editor.clearSelection();
        editor.setReadOnly(true);
    } else {
        respBodyContainer.style.display = 'none';
    }
};

const renderEntries = (items, service) => {
    const tbody = document.getElementById('history-table-body');
    tbody.innerHTML = '';

    if (!items || items.length === 0) {
        const row = document.createElement('tr');
        const cell = document.createElement('td');
        cell.colSpan = 5;
        cell.textContent = 'No history entries';
        cell.style.textAlign = 'center';
        cell.style.color = 'var(--text-muted)';
        row.appendChild(cell);
        tbody.appendChild(row);
        return;
    }

    // Show newest first
    const sorted = [...items].reverse();

    sorted.forEach((entry, i) => {
        const row = document.createElement('tr');
        row.id = `history-${entry.id}`;
        row.style.cursor = 'pointer';
        row.onclick = () => {
            navi.applySelection(`history-${entry.id}`, 'selected-resource');
            showDetail(entry);
        };

        const numCell = document.createElement('td');
        numCell.textContent = `${i + 1}`;
        row.appendChild(numCell);

        const methodCell = document.createElement('td');
        const method = entry.request ? entry.request.method : '';
        methodCell.textContent = method;
        methodCell.className = `fixed-resource-method ${method.toLowerCase()}`;
        row.appendChild(methodCell);

        const pathCell = document.createElement('td');
        pathCell.className = 'fixed-resource-path';
        const pathSpan = document.createElement('span');
        pathSpan.textContent = entry.resource || (entry.request ? entry.request.url : '');
        pathCell.appendChild(pathSpan);
        pathCell.title = pathSpan.textContent;
        row.appendChild(pathCell);

        const statusCell = document.createElement('td');
        if (entry.response) {
            statusCell.textContent = entry.response.statusCode;
            statusCell.className = `history-status ${statusClass(entry.response.statusCode)}`;
        }
        row.appendChild(statusCell);

        const timeCell = document.createElement('td');
        timeCell.textContent = formatTime(entry.createdAt);
        timeCell.className = 'history-time';
        row.appendChild(timeCell);

        tbody.appendChild(row);
    });
};

const fetchAndRender = (service) => {
    const historyApiUrl = `${config.historyUrl}/${service}`;
    return fetch(historyApiUrl)
        .then(res => res.json())
        .then(data => {
            renderEntries(data.items, service);
        })
        .catch(err => {
            console.error('Failed to fetch history:', err);
        });
};

export const show = (match) => {
    const {name} = match.params;
    const service = name;

    navi.resetContents();
    services.show();

    navi.applySelection(`service-${service}`, 'selected-service');

    let displayName = service;
    if (displayName === '.root') {
        displayName = 'Root level';
    } else {
        displayName = `/${displayName}`;
    }
    config.contentTitleEl.innerHTML = `${displayName} history`;

    showTabs(service);
    config.historyContainer.style.display = 'block';

    fetchAndRender(service);

    document.getElementById('history-refresh').onclick = () => fetchAndRender(service);
    document.getElementById('history-clear').onclick = () => {
        if (!confirm(`Clear history for ${displayName}?`)) return;
        const historyApiUrl = `${config.historyUrl}/${service}`;
        fetch(historyApiUrl, {method: 'DELETE'})
            .then(() => fetchAndRender(service))
            .catch(err => console.error('Failed to clear history:', err));
    };
};
