// import { default as xterm } from 'https://cdn.jsdelivr.net/npm/xterm@4.19.0/lib/xterm.js'

// debugger

const wsProtocol = document.location.protocol == 'https:' ? 'wss:' : 'ws:';
const socket = new WebSocket(`${wsProtocol}//${document.location.host}/ws`);

let term = new Terminal();
const fitAddon = new FitAddon.FitAddon();
term.loadAddon(fitAddon);

// Connection opened
socket.addEventListener('open', function (event) {
    term.open(document.getElementById('terminal'));
    fitAddon.fit();
    // socket.send('Hello Server!');
});

// Listen for messages
socket.addEventListener('message', async function (event) {
    term.write(new Uint8Array(await event.data.arrayBuffer()));
    // let arr = new Uint8Array(await event.data.arrayBuffer());
    // console.log('Message from server:', new TextDecoder().decode(arr));
});

const encoder = new TextEncoder();

term.onData((chunk) => {
    let encoded = encoder.encode(chunk);
    let buffer = new Uint8Array(4 + encoded.length);
    buffer.set(new Uint32Array([1]), 0);
    buffer.set(encoded, 4);
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
