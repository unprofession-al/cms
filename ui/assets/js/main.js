import * as mycro from './mycro.js';

var projectsElement = document.querySelector("#projects");
projectsElement.addEventListener('change', projectSelected, false);

var filesElement = document.querySelector("#files");
var filesElementObserver = new MutationObserver(updateFileClickWatcher);
function updateFileClickWatcher() {
    var fileElements = document.querySelectorAll(".file");
    fileElements.forEach(function (fileElement) {
        fileElement.addEventListener('click', fileSelected, false);
    });
};
filesElementObserver.observe(filesElement, { attributes: true, childList: true, subtree: true });

var workareaElement = document.querySelector("#workarea");

mycro.listProjects().then(projects => {
    for (let k in projects) {
        var option = document.createElement("option");
        option.text = k;
        projectsElement.add(option);
    }
    projectSelected();
});

function projectSelected() {
    var projectName = projectsElement.value
    router.setProject(projectName);
    mycro.listFiles(projectName).then(files => {
        filesElement.innerHTML = "";
        files.children.forEach(function (child) {
            walkFileTree(child, filesElement);
        });
    });

};

function walkFileTree(branch, parent) {
    if (branch.is_dir == true) {
        var checkbox = document.createElement("input");
        checkbox.className = "hidden checkbox";
        checkbox.type = "checkbox";
        checkbox.id = "cb_" + branch.full_path;

        var folder = document.createElement("label");
        folder.className = "folder";
        folder.setAttribute("data-path", branch.full_path);
        folder.setAttribute("for", "cb_" + branch.full_path);
        folder.innerHTML = branch.name;

        var content = document.createElement("div")
        content.className = "foldercontent";

        parent.appendChild(checkbox);
        parent.appendChild(folder);
        parent.appendChild(content);

        branch.children.forEach(function (child) {
            walkFileTree(child, content);
        });
    } else {
        var file = document.createElement("div");
        file.className = "file"
        file.setAttribute("data-path", branch.full_path);
        file.innerHTML = branch.name;
        parent.appendChild(file);
    }
};

function fileSelected() {
    var filename = this.getAttribute("data-path");
    router.setFile(filename);
    var project = router.project()
    mycro.getFile(project, filename).then(content => {
        workareaElement.innerHTML = "";
        var id = window.btoa("content_"+ project + filename);

        var textarea = document.createElement("textarea");
        textarea.className = "raweditor";
        textarea.id = id;
        textarea.setAttribute("data-file", filename);
        textarea.setAttribute("data-project", project);
        textarea.value = content;

        var controls = document.createElement("div");
        controls.className = "controls";

        var button = document.createElement("button");
        button.setAttribute("data-file", filename);
        button.setAttribute("data-project", project);
        button.setAttribute("data-target", id);
        controls.className = "save";
        button.innerHTML = "save";
        button.addEventListener('click', saveFile, false);
        controls.appendChild(button);

        workareaElement.appendChild(textarea);
        workareaElement.appendChild(controls);
    });
};

function saveFile() {
    var data = document.getElementById(this.getAttribute("data-target"));
    var project = data.getAttribute("data-project");
    var file = data.getAttribute("data-file");
    mycro.writeFile(project, file, data.value).then(content => {
        console.log("saved");
        mycro.publish(project).then(content => {
            console.log("published");
        });
    });
};

var router = {
    setProject : function(name) {
        window.location.hash = '#/' + name;
    },

    setFile : function(name) {
        var proj = this.project();
        window.location.hash = '#/' + proj + name;
    },

    project : function() {
        return window.location.hash.split("/")[1]
    },

    file : function() {
        return window.location.hash.split(this.project())[1]
    },
};

