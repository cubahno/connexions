import * as config from './config.js';
import * as navi from "./navi.js";

const STORAGE_KEY_STARS = 'cxs-starred-services';

let cachedServices = null;

const getStarred = () => {
    try {
        const raw = localStorage.getItem(STORAGE_KEY_STARS);
        return raw ? JSON.parse(raw) : [];
    } catch {
        return [];
    }
};

const setStarred = (list) => {
    localStorage.setItem(STORAGE_KEY_STARS, JSON.stringify(list));
};

const toggleStar = (name) => {
    const starred = getStarred();
    const idx = starred.indexOf(name);
    if (idx === -1) {
        starred.push(name);
    } else {
        starred.splice(idx, 1);
    }
    setStarred(starred);
};

const renderServices = (services, selected) => {
    const tbody = document.getElementById('table-body');
    tbody.innerHTML = '';

    const starred = getStarred();

    const sorted = [...services].sort((a, b) => {
        const aStarred = starred.includes(a.name);
        const bStarred = starred.includes(b.name);
        if (aStarred !== bStarred) return aStarred ? -1 : 1;
        return a.name.localeCompare(b.name);
    });

    let i = 0;
    for (let { name, type, resourceNumber } of sorted) {
        const num = i + 1;
        const row = document.createElement('tr');
        row.id = `service-${name}`;

        const originalName = name;
        const isStarred = starred.includes(originalName);

        const starCell = document.createElement('td');
        const star = document.createElement('span');
        star.className = 'star-toggle' + (isStarred ? ' starred' : '');
        star.innerHTML = isStarred
            ? '<i class="fa-solid fa-star"></i>'
            : '<i class="fa-regular fa-star"></i>';
        star.title = 'Toggle favorite';
        star.addEventListener('click', (e) => {
            e.stopPropagation();
            e.preventDefault();
            toggleStar(originalName);
            const current = document.querySelector('#service-table tr.selected-service');
            const currentId = current ? current.id.replace('service-', '') : '';
            renderServices(services, currentId);
        });
        starCell.appendChild(star);
        row.appendChild(starCell);

        let nameLink = name;
        if (name === ``) {
            name = "/"
            nameLink = `.root`
        }

        const svcNameCell = document.createElement('td');
        svcNameCell.innerHTML = `<a href="#/services/${nameLink}">${name}</a>`;
        row.appendChild(svcNameCell);

        const numResCell = document.createElement('td');
        numResCell.innerHTML = resourceNumber;
        numResCell.title = `Number of resources`;

        row.appendChild(numResCell);

        tbody.appendChild(row);
        i += 1;
    }

    config.serviceTable.style.display = 'block';
    if (selected !== ``) {
        navi.applySelection(`service-${selected}`, 'selected-service');
    }
};

export const show = (selected = '') => {
    config.servicesLink.className = `menu-link active`;

    const tbody = document.getElementById('table-body');
    tbody.innerHTML = '';

    console.log("loading service list");

    fetch(config.serviceUrl)
        .then(res => res.json())
        .then(data => {
            cachedServices = data['items'];
            renderServices(cachedServices, selected);
        });
}
