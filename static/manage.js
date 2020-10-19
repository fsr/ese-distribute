const key = window.location.search;

function renderAll(state) {
    links.innerHTML = "";
    const urls = Object.keys(state);
    urls.sort();
    for (var idx = 0; idx < urls.length; ++idx) {
        const url = urls[idx];
        var li = document.createElement("li");
        li.style = "list-style: none; display: flex; margin-bottom: 1rem; align-items: center;";

        var a = document.createElement("a");
        a.innerText = url + " (" + state[url] + " free)";
        a.href = "#" + url;
        a.addEventListener("click", () => {
            document.location.hash = url;
            update();
        });
        li.appendChild(a);

        links.appendChild(li);
    }
}

function render(state) {
    if (document.location.hash.trim() == "") {
        renderAll(state);
        addButton.classList.add("hidden");
        singleRoomDiv.classList.add("hidden");
        selectDiv.classList.remove("hidden");
        links.classList.remove("hidden");
        registerDiv.classList.remove("hidden");
    } else {
        var url = document.location.hash.trim().substr(1)
        roomURL.href = url;
        roomURL.innerText = url;
        freeCount.innerText = state[url] || "0";
        addButton.classList.remove("hidden");
        singleRoomDiv.classList.remove("hidden");
        selectDiv.classList.add("hidden");
        links.classList.add("hidden");
        registerDiv.classList.add("hidden");
    }
    document.getElementById("page-content").classList.remove("hidden");
}

function select() {
    if (urlInput.value.trim() == "") return;
    document.location.hash = "#" + urlInput.value.trim();

    update();
    return false;
}

function add(count) {
    var req = new XMLHttpRequest();
    req.open("POST", "/api/free" + window.location.search, true);
    req.setRequestHeader( 'Content-Type', 'application/x-www-form-urlencoded' );
    req.onreadystatechange = function () {
        if (this.readyState == 4 && this.status == 200) {
            render(JSON.parse(req.responseText));
        }
    }
    url = document.location.hash.substr(1);
    req.send("count=" + count + "&url=" + encodeURIComponent(url));
    return false;
}

function register(url) {
    var req = new XMLHttpRequest();
    req.open("POST", "/api/register" + window.location.search, true);
    req.setRequestHeader( 'Content-Type', 'application/x-www-form-urlencoded' );
    req.onreadystatechange = () => {
        if (req.readyState == 4 && req.status == 200) {
            document.location.hash = "#" + url;
            render(JSON.parse(req.responseText));
        }

        if (req.readyState == 4 && req.status == 409) {
            alert("room already registered!");
        }
    }
    req.send("count=" + 100000 + "&url=" + encodeURIComponent(url));
    return false;
}

function deleteRoom() {
    if (!confirm("really delete room?")) {
        return;
    }

    var req = new XMLHttpRequest();
    req.open("POST", "/api/delete" + window.location.search, true);
    req.setRequestHeader( 'Content-Type', 'application/x-www-form-urlencoded' );
    req.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            document.location.hash = "";
            render(JSON.parse(req.responseText));
        }
    }
    url = document.location.hash.substr(1);
    req.send("url=" + encodeURIComponent(url));
    return false;
}

function update() {
    var req = new XMLHttpRequest();
    req.open("GET", "/api/state?" + window.location.search, true);
    req.onreadystatechange = function () {
        if (this.readyState == 4 && this.status == 200) {
            render(JSON.parse(req.responseText));
        }
    }

    req.send();
}

setInterval(update, 2000);
update();
