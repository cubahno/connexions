export const fixAndValidateJSON = str => {
    if (!str) {
        return null;
    }

    try {
        let trimmedStr = str.trim();
        let fixedStr = trimmedStr.replace(/\n/g, '');
        fixedStr = fixedStr.replace(/"\s*:\s*"/g, '":"');
        return JSON.parse(fixedStr);
    } catch (error) {
        console.log("error", error);
        return null;
    }
}
