import * as config from "./config.js";
import * as navi from "./navi.js";
import * as services from "./services.js";

export const home = () => {
    navi.resetContents();
    services.show();

    config.homeContents.style.display = 'block';
}

export const showVersion = () => {
    const el = document.getElementById('app-version');
    if (config.version !== "") {
        el.textContent = config.version;
    }
}
