export const url = window.location.origin;
export const homeUrl = `${url}${appConfig.homeUrl}`;
export const serviceUrl = `${url}${appConfig.serviceUrl}`;
export const contextUrl = `${url}${appConfig.contextUrl}`;
export const settingsUrl = `${url}${appConfig.settingsUrl}`;
export const serviceTable = document.getElementById('service-table');
export const contextTable = document.getElementById('context-table');
export const servicesLink = document.getElementById('services-link');
export const contextsLink = document.getElementById('contexts-link');
export const generatorCont = document.getElementById('generator-container');
export const contentTitleEl = document.getElementById('container-title');
export const iframeContents = document.getElementById('iframe-contents');
export const servicesUploadForm = document.getElementById('services-upload');
export const resourcesImportForm = document.getElementById('resources-import');
export const messageCont = document.getElementById('message');
export const fileUploadBtn = document.getElementById('fileupload');
export const settingsEditor = document.getElementById('settings-editor');
export const fixedServiceContainer = document.getElementById('fixed-service-container');
export const resourceRefreshBtn = document.getElementById('refresh');
export const responseContentTypeEl = document.getElementById(`response-content-type`);
export const contextEditContainer = document.getElementById('context-edit-container');
