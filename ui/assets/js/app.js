import * as mycro from './mycro.js';
import * as tmpl from './templating.js';
import * as router from './router.js';

export function App(config) {
    if (config.projectsElement) {
        this.projectsElement = document.querySelector(config.projectsElement);
    } else {
        this.projectsElement = document.querySelector("#projects");
    }

    if (config.filesElement) {
        this.filesElement = document.querySelector(config.filesElement);
    } else {
        this.filesElement = document.querySelector("#files");
    }

    if (config.workareaElement) {
        this.workareaElement = document.querySelector(config.workareaElement);
    } else {
        this.workareaElement = document.querySelector("#workarea");
    }

    this.router = new router.Router();
}

App.prototype.run = function() {
    this.projectsElement.addEventListener('change', this.projectSelected.bind(this), false);

    var filesElementObserver = new MutationObserver(this.updateFileClickWatcher.bind(this));
    filesElementObserver.observe(this.filesElement, { attributes: true, childList: true, subtree: true });

    mycro.listProjects().then(projects => {
        for (let k in projects) {
            var option = document.createElement("option");
            option.text = k;
            this.projectsElement.add(option);
        }
        this.projectSelected();
    });
}

App.prototype.updateFileClickWatcher = function() {
    var fileElements = document.querySelectorAll(".file");
    for (var i = 0, len = fileElements.length; i < len; i++) {
        fileElements[i].addEventListener('click', this.fileSelected.bind(this), false);
    }
}

App.prototype.projectSelected = function() {
    var projectName = this.projectsElement.value
    this.router.setProject(projectName);
    mycro.listFiles(projectName).then(files => {
        this.filesElement.innerHTML = "";
        this.workareaElement.innerHTML = "";
        for (var i = 0, len = files.children.length; i < len; i++) {
            this.walkFileTree(files.children[i], this.filesElement);
        }
    });
}

App.prototype.walkFileTree = function(branch, parent) {
    var id = window.btoa("checkbox_" + branch.full_path);
    var data = { id: id, name: branch.name, path: branch.full_path };
    if (branch.is_dir == true) {
        var frag = tmpl.render("folder_tmpl", data);
        // Todo: find better solution since this need knowlodge of the dom
        var content = frag.querySelector(".foldercontent");
        parent.appendChild(frag);
        for (var i = 0, len = branch.children.length; i < len; i++) {
            this.walkFileTree(branch.children[i], content);
        }
    } else {
        var frag = tmpl.render("file_tmpl", data);
        parent.appendChild(frag);
    }
}

App.prototype.fileSelected = function(event) {
    var filename = event.target.getAttribute("data-path");
    this.router.setFile(filename);
    var project = this.router.project()
    mycro.getFile(project, filename).then(content => {
        this.workareaElement.innerHTML = "";

        var data = {
            id: window.btoa("content_" + project + filename),
            project: project,
            path: filename,
            content: content,
        };

        var editor = tmpl.render("editor_tmpl", data);
        this.workareaElement.appendChild(editor);

        // Todo: there is no listener on button to trigger saveFile
        var controls = tmpl.render("controls_tmpl", data);
        this.workareaElement.appendChild(controls);
    });
}

App.prototype.saveFile = function(event) {
    console.log(event);
    var data = document.getElementById(this.getAttribute("data-target"));
    var project = data.getAttribute("data-project");
    var file = data.getAttribute("data-file");
    mycro.writeFile(project, file, data.value).then(content => {
        console.log("saved");
        /*
        mycro.publish(project).then(content => {
            console.log("published");
        });
        */
    });
}

