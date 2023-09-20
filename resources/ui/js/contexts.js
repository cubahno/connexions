import * as config from "./config.js";
import * as navi from "./navi.js";
import * as commons from "./commons.js";

export const show = (selected=``) => {
    config.servicesLink.className = `menu-link inactive`;
    config.contextsLink.className = `menu-link active`;

    config.contentTitleEl.innerHTML = `Select context from the menu`;

    const addNewCont = document.getElementById('add-new-context-cont').innerHTML;
    config.serviceTable.innerHTML = '';
    config.fixedServiceContainer.style.display = 'none';

    console.log("loading context list");

    const t = document.createElement(`table`);
    const newRow = t.insertRow();
    newRow.innerHTML = addNewCont;

    fetch(config.contextUrl)
        .then(res => res.json())
        .then(data => {
            const items = data['items'];
            console.log(items);
            let i = 0;
            for (let name of items) {
                const num = i + 1;
                const row = document.createElement('tr');
                row.id = `context-${name}`;

                const cell1 = document.createElement('td');
                cell1.textContent = `${num}`;
                row.appendChild(cell1);

                const mameCell = document.createElement('td');
                mameCell.innerHTML = `<a href="#/contexts/${name}">${name}</a>`;
                row.appendChild(mameCell);

                const rmCell = document.createElement('td');
                rmCell.innerHTML = `✖`;
                rmCell.id = `remove-context-${name}`;
                rmCell.className = `remove-context ${name}`;
                rmCell.title = `Remove context ${name}`;

                row.appendChild(rmCell);

                t.appendChild(row);
                i += 1;
            }

            config.contextTable.innerHTML = t.innerHTML;
            config.contextTable.style.display = 'block';
            if (selected !== ``) {
                navi.applySelection(`context-${selected}`, 'selected-context');
            }

            const elements = document.querySelectorAll(`.remove-context`);
            elements.forEach(element => {
                element.addEventListener(`click`, event => {
                    const name = event.target.id.replace(`remove-context-`, ``);

                    if (confirm(`Are you sure you want to remove context ${name}?\n`)) {
                        fetch(`${config.contextUrl}/${name}`, {
                            method: 'DELETE'
                        })
                            .then(res => res.json())
                            .then(res => {
                                if (res.success) {
                                    location.hash = `#/contexts`;
                                    location.reload(true);
                                }
                                window.setTimeout(_ => {
                                    commons.showSuccessOrError(res.message, res.success);
                                }, 100);
                            });
                    }
                });
            });
        });
}

export const editForm = (match) => {
    let name = ``;
    let title = `Add new context`
    if (match !== undefined) {
        name = match.params.name;
        title = `Edit context "${name}"`;
    }

    console.log(`edit context`);

    navi.resetContents();
    show(name);

    const editor = commons.getCodeEditor('context-code-editor');
    editor.setOptions({
        mode: `ace/mode/yaml`,
    });
    config.contentTitleEl.innerHTML = title;

    if (name !== ``) {
        fetch(`${config.contextUrl}/${name}`)
            .then(res => res.text())
            .then(res => {
                document.getElementById(`context-name`).value = name;
                editor.setValue(res);
                editor.clearSelection();
            })
    }

    config.contextEditContainer.style.display = 'block';

    document.getElementById(`context-save-button`).addEventListener(`click`, save);
}

export const save = () => {
    const editor = commons.getCodeEditor(`context-code-editor`, `yaml`);
    const yaml = editor.getValue();
    commons.showWarning("Saving context...")
    const name = document.getElementById(`context-name`).value;

    const formData = new FormData();
    formData.append("content", yaml);
    formData.append("name", name);

    fetch(config.contextUrl, {
        method: "PUT",
        body: formData,
    }).then(res => res.json()).then(res => {
        commons.showSuccessOrError(res.message, res.success);
        show(name);
    });
}
