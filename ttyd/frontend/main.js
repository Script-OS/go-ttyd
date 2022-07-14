// import { default as xterm } from 'https://cdn.jsdelivr.net/npm/xterm@4.19.0/lib/xterm.js'

// debugger

const wsProtocol = document.location.protocol == 'https:' ? 'wss:' : 'ws:';
const socket = new WebSocket(`${wsProtocol}//${document.location.host}/ws`);

let term = new Terminal({
    fontFamily: 'GoMono Nerd Font',
});
const fitAddon = new FitAddon.FitAddon();
const uincode11Addon = new Unicode11Addon.Unicode11Addon();
// const webglAddon = new WebglAddon.WebglAddon();

let LinkSet = [];

const MarkerContent = document.getElementById("media");//document.createElement("div");

term.parser.registerOscHandler(9999, function (payload) {
    if (payload == "") {
        LinkSet = [];
        for (let it of MarkerContent.children) {
            it.style.display = 'none';
        }
        return true;
    }
    let parts = payload.split(';');
    if (parts.length >= 1) {
        let type = parts[0];
        switch (type) {
            case 'link':
                {
                    let len = parseInt(parts[1]);
                    let link = parts.slice(2).join(';');
                    let x = term.buffer.active.cursorX;
                    let y = term.buffer.active.baseY + term.buffer.active.cursorY;
                    LinkSet.push({
                        x, y, len, link,
                    });
                }
                break;
            case 'media':
                {
                    let id = parts[1];
                    let text = atob(parts[2]);
                    let lines = parseInt(parts[3]);
                    let url = parts.slice(4).join(';');
                    let y = term.buffer.active.baseY + term.buffer.active.cursorY;
                    let CellHeight = term._core._renderService.dimensions.actualCellHeight;

                    id = 'media-content-' + id;

                    let media = document.getElementById(id);

                    if (media == null) {
                        media = document.createElement("div");
                        media.style.height = `${CellHeight * lines}px`;
                        media.style.position = 'absolute';
                        media.style.left = `32px`;
                        media.id = id;
                        MarkerContent.appendChild(media);

                        fetch(url, { method: 'HEAD' }).then((res) => {
                            let contentType = res.headers.get('content-type');
                            let mediaType = contentType.split('/')[0];

                            let media = document.getElementById(id);
                            if (media == null) {
                                return;
                            }
                            if (mediaType == 'image') {
                                media.innerHTML = `<image src="${url}" alt="${text}" height="${CellHeight * lines}">`;
                                media.onclick = function () {
                                    window.open(url, "_blank");
                                }
                            } else if (mediaType == 'video') {
                                media.innerHTML = `<video controls src="${url}" alt="${text}" height="${CellHeight * lines}">`;
                            }
                        });
                    }
                    media.style.top = `${CellHeight * (y + 1)}px`;
                    media.style.display = 'block';
                }
                break;
            case 'cleanMedia':
                MarkerContent.innerHTML = "";
                break;
        }
    }
    return true;
});

term.registerLinkProvider({
    provideLinks(y, callback) {
        let links = [];
        for (let link of LinkSet) {
            if (link.y == y - 1) {
                links.push({
                    range: {
                        start: { x: link.x + 1, y },
                        end: { x: link.x + link.len, y },
                    },
                    text: link.link,
                    activate() {
                        window.open(link.link, "_blank");
                    }
                });
            }
        }
        if (links.length > 0) {
            callback(links);
        } else {
            callback(undefined);
        }
    }
});

function RefreshScroll() {
    let viewport = document.querySelector('.xterm-scroll-area');
    const CellHeight = term._core._renderService.dimensions.actualCellHeight;
    MarkerContent.parentElement.scrollTop = Math.round(viewport.parentElement.scrollTop / CellHeight) * CellHeight;
    MarkerContent.style.height = viewport.style.height;
}

// Connection opened
socket.addEventListener('open', function (event) {
    term.open(document.getElementById('terminal'));
    document.querySelector(".xterm-screen").insertBefore(MarkerContent.parentElement, document.querySelector(".xterm-decoration-container"));
    let viewport = document.querySelector('.xterm-scroll-area');
    viewport.parentElement.addEventListener("scroll", RefreshScroll);
    term.loadAddon(fitAddon);
    term.loadAddon(uincode11Addon);
    // term.loadAddon(webglAddon);
    fitAddon.fit();
});

// Listen for messages
socket.addEventListener('message', async function (event) {
    term.write(new Uint8Array(await event.data.arrayBuffer()));
    // let arr = new Uint8Array(await event.data.arrayBuffer());
    // console.log('Message from server:', new TextDecoder().decode(arr));
});

term.onRender(RefreshScroll);

const encoder = new TextEncoder();

term.onData((chunk) => {
    let encoded = encoder.encode(chunk);
    let buffer = new Uint8Array(4 + encoded.length);
    buffer.set(new Uint32Array([1]), 0);
    buffer.set(encoded, 4);
    socket.send(buffer);
});

term.onBinary((chunk) => {
    let buffer = new Uint8Array(4 + chunk.length);
    buffer.set(new Uint32Array([1]), 0);
    for (let i = 0; i < chunk.length; i++) {
        buffer[i + 4] = chunk.charCodeAt(i);
    }
    socket.send(buffer);
});

term.onResize((data) => {
    let encoded = encoder.encode(JSON.stringify(data));
    let buffer = new Uint8Array(4 + encoded.length);
    buffer.set(new Uint32Array([2]), 0);
    buffer.set(encoded, 4);
    socket.send(buffer);
});

socket.addEventListener('close', function () {
    console.log("connection closed");
});

window.addEventListener('resize', function () {
    fitAddon.fit();
});
