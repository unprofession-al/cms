async function listProjects() {
    try {
        let p = await axios.get('/sites/');
        let result = p.data;
        return result
    } catch (e) {
        console.error(e);
    }
    return {};
}

async function listFiles(project) {
    try {
        let p = await axios.get('/sites/' + project + '/files/');
        let result = p.data;
        return result
    } catch (e) {
        console.error(e);
    }
    return {};
}

async function getFile(project, filename) {
    try {
        let p = await axios.get('/sites/' + project + '/files' + filename);
        let result = p.data;
        return result
    } catch (e) {
        console.error(e);
    }
    return {};
}

async function writeFile(project, filename, content) {
    try {
        let p = await axios.post('/sites/' + project + '/files' + filename, content, {params: { o: "all" }});
        let result = p.data;
        return result
    } catch (e) {
        console.error(e);
    }
    return {};
}

async function publish(project) {
    try {
        let p = await axios.put('/sites/' + project + '/publish/');
        let result = p.data;
        return result
    } catch (e) {
        console.error(e);
    }
    return {};
}

export { listProjects, listFiles, getFile, writeFile, publish };
