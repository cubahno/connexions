import * as config from './config.js';
import {contentTitleEl} from "./config.js";

export const loadPage = pageMap => {
    const currentHash = window.location.hash;
    console.log(`currentHash ${currentHash}`);

    // Check if the current hash matches any exact matches in the map
    if (pageMap.has(currentHash)) {
        console.log("page exact match");
        const pageFunction = pageMap.get(currentHash);
        pageFunction();
        return;
    }

    // Check if the current hash matches any placeholders in the map
    for (const [key, pageFunction] of pageMap) {
        if (isPlaceholderMatch(key, currentHash)) {
            console.log("page matched by pattern");
            const match = getPlaceholderMatch(key, currentHash);
            pageFunction(match);
            return;
        }
    }
    console.log("no page matched");
}

// Helper function to check if a placeholder pattern matches the current hash
export const isPlaceholderMatch = (placeholderPattern, currentHash) => {
    const patternParts = placeholderPattern.split('/');
    const hashParts = currentHash.split('/');

    if (patternParts.length !== hashParts.length) {
        return false;
    }

    for (let i = 0; i < patternParts.length; i++) {
        const patternPart = patternParts[i];
        if (patternPart.startsWith(':')) {
            continue; // Skip placeholder parts
        }

        if (patternPart !== hashParts[i]) {
            return false;
        }
    }

    return true;
}

// Helper function to extract the placeholder values from the current hash
export const getPlaceholderMatch = (placeholderPattern, currentHash) => {
    const patternParts = placeholderPattern.split('/');
    const hashParts = currentHash.split('/');

    const match = {
        params: {}
    };

    for (let i = 0; i < patternParts.length; i++) {
        const patternPart = patternParts[i];
        if (patternPart.startsWith(':')) {
            const paramName = patternPart.substring(1);
            match.params[paramName] = decodeURI(hashParts[i]);
        }
    }

    return match;
}

export const applySelection = (targetEl, selectionClassName) => {
    console.log(`applying selection for ${targetEl}`);
    if (!targetEl) {
        return;
    }

    if (targetEl === `service-.root`) {
        targetEl = `service-`;
    }

    const collection = document.getElementsByClassName(selectionClassName);
    for (let i = 0; i < collection.length; i++) {
        collection[i].classList.remove(selectionClassName);
    }
    const row = document.getElementById(targetEl);
    if (!row) {
        console.log(`no row found for ${targetEl}`);
        return;
    }
    row.classList.add(selectionClassName);
}

export const resetContents = () => {
    console.log(`reset contents`);
    config.homeContents.style.display = 'none';
    config.contentTitleEl.innerHTML = '';
    config.iframeContents.src = '';
    config.iframeContents.style.display = 'none';

    config.generatorCont.style.display = 'none';

    config.serviceTable.style.display = 'none';
    config.contextTable.style.display = 'none';
    config.contextTable.innerHTML = '';

    config.servicesUploadForm.style.display = 'none';
    config.resourcesImportForm.style.display = 'none';
    config.settingsEditor.style.display = 'none';
    config.fixedServiceContainer.style.display = 'none';

    document.getElementById('fixed-service-table-body').innerHTML = '';
    document.getElementById('resource-result').innerHTML = '';
    document.getElementById('resource-edit-container').style.display = 'none';
    config.resourceRefreshBtn.style.display = 'none';
}
