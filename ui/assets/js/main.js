import * as mycro from './mycro.js';
import * as tmpl from './templating.js';

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
        workareaElement.innerHTML = "";
        files.children.forEach(function (child) {
            walkFileTree(child, filesElement);
        });
    });

};

function walkFileTree(branch, parent) {
    var id = window.btoa("checkbox_" + branch.full_path);
    var data = { id: id, name: branch.name, path: branch.full_path };
    if (branch.is_dir == true) {
        var frag = tmpl.render("folder_tmpl", data);
        // Todo: find better solution since this need knowlodge of the dom
        var content = frag.querySelector(".foldercontent");
        parent.appendChild(frag);

        branch.children.forEach(function (child) {
            walkFileTree(child, content);
        });
    } else {
        var frag = tmpl.render("file_tmpl", data);
        parent.appendChild(frag);
    }
};

function fileSelected() {
    var filename = this.getAttribute("data-path");
    router.setFile(filename);
    var project = router.project()
    mycro.getFile(project, filename).then(content => {
        workareaElement.innerHTML = "";

        var data = {
            id: window.btoa("content_" + project + filename),
            project: project,
            path: filename,
            content: content,
        };

        var editor = tmpl.render("editor_tmpl", data);
        workareaElement.appendChild(editor);

        var controls = tmpl.render("controls_tmpl", data);
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
