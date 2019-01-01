/*

## CREDITS

This thing is almont completely stolen from:

    Simple JavaScript Templating
    John Resig - https://johnresig.com/ - MIT Licensed

and

    https://davidwalsh.name/convert-html-stings-dom-nodes

## USAGE

In the HTML file, define the templates:

    <html lang="en">
        <head>...</head>
        <body>
            ...
            <script type="module" src="main.js"></script>
        </body>
    </html>

    <script type="text/html" id="test_tmpl">
        <div>Data is "<%=data%>"</div>
    </script>

Define the JavaScript

    import * as tmpl from './templating.js';

    var x = { data: "text" };
    var test = tmpl.render("test_tmpl", x);
    var b = document.querySelector("body");
    b.appendChild(test);

*/

var cache = {};

export function render(str, data){
    // Figure out if we're getting a template, or if we need to
    // load the template - and be sure to cache the result.
    var fn = !/\W/.test(str) ?
        cache[str] = cache[str] ||
        render(document.getElementById(str).innerHTML) :

        // Generate a reusable function that will serve as a template
        // generator (and which will be cached).
        new Function("obj",
            "var p=[],print=function(){p.push.apply(p,arguments);};" +

            // Introduce the data as local variables using with(){}
            "with(obj){p.push('" +

            // Convert the template into pure JavaScript
            str
            .replace(/[\r\t\n]/g, " ")
            .split("<%").join("\t")
            .replace(/((^|%>)[^\t]*)'/g, "$1\r")
            .replace(/\t=(.*?)%>/g, "',$1,'")
            .split("\t").join("');")
            .split("%>").join("p.push('")
            .split("\r").join("\\'")
            + "');}return document.createRange().createContextualFragment(p.join(''));");

    // Provide some basic currying to the user
    return data ? fn( data ) : fn;
};
