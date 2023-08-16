import * as config from "./config.js";

export const show = () => {
    config.servicesLink.className = `menu-link inactive`;
    config.contextsLink.className = `menu-link active`;

    const addNewCont = document.getElementById('add-new-context-cont').innerHTML;
    config.serviceTable.innerHTML = '';
    const newRow = config.serviceTable.insertRow();
    newRow.innerHTML = addNewCont;
    console.log("loading context list");

    config.serviceTable.style.display = 'block';
}
