var polling = false;

function start() {
    if (polling == true) {
        return;
    }
    polling = true;
    document.getElementById("start").setAttribute("style", "display: none");
    document.getElementById("pairing").setAttribute("style", "display: inline");
}

function poll() {
    if (!polling) {
        return;
    }

    var req = new XMLHttpRequest();
    req.open("POST", "/api/poll", true);
    req.setRequestHeader( 'Content-Type', 'application/x-www-form-urlencoded' );
    req.onreadystatechange = function () {
        if (this.readyState == 4 && this.status == 200) {
            if (polling == false) {
                return;
            }
            if (req.responseText == "wait") {
                return;
            }
            if (req.responseText == "nouuid") {
                polling = false;
                start();
                return;
            }
            window.location = req.responseText;
            polling = false;
            document.getElementById("start").setAttribute("style", "display: inline");
            document.getElementById("pairing").setAttribute("style", "display: none");
        }
    }

    req.send();
}

setInterval(poll, 2000);
