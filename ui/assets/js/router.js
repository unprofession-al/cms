export function Router() {}

Router.prototype.setProject = function(name) {
    window.location.hash = '#/' + name;
}

Router.prototype.setFile = function(name) {
    var proj = this.project();
    window.location.hash = '#/' + proj + name;
}

Router.prototype.project = function() {
    return window.location.hash.split("/")[1]
}

Router.prototype.file = function() {
    return window.location.hash.split(this.project())[1]
}
