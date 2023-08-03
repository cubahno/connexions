import * as config from './config.js';
import * as navi from "./navi.js";

export const show = (selected = '') => {
    config.servicesLink.className = `menu-link active`;

    config.serviceTable.innerHTML = '';

    console.log("loading service list");

    const t = document.createElement(`table`);

    fetch(config.serviceUrl)
        .then(res => res.json())
        .then(data => {
            const services = data['items'];

            let i = 0;
            for (let { name, type, resourceNumber } of services) {
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

                const numResCell = document.createElement('td');
                numResCell.innerHTML = resourceNumber;
                numResCell.title = `Number of resources`;

                row.appendChild(numResCell);

                t.appendChild(row);
                i += 1;
            }

            config.serviceTable.innerHTML = t.innerHTML;
            config.serviceTable.style.display = 'block';
            if (selected !== ``) {
                navi.applySelection(`service-${selected}`, 'selected-service');
            }
        });
}
