// master.js
//
// Reveal.js multiplex master. Hooks into Reveal lifecycle events and
// pushes serialised state to the in-process Go multiplex server over a
// plain WebSocket. Replaces the socket.io-based original.
//
// Expects window.RevealMultiplex = { secret, id, url } before the
// reveal.js script tag runs (set inline in the template).

(function () {
    var cfg = window.RevealMultiplex || {};
    if (!cfg.secret || !cfg.id) {
        console.warn("[multiplex] missing secret/id, master disabled");
        return;
    }

    var origin = (cfg.url || window.location.origin).replace(/\/+$/, "");
    var wsURL = origin.replace(/^http/, "ws") + "/multiplex/presenter";

    var ws;
    var queue = [];
    var ready = false;

    function connect() {
        ws = new WebSocket(wsURL);
        ws.addEventListener("open", function () {
            ready = true;
            while (queue.length) ws.send(queue.shift());
        });
        ws.addEventListener("close", function () {
            ready = false;
            // Brief backoff and reconnect — the presenter is supposed
            // to keep the page open for the duration of a talk, so we
            // shouldn't hide transient hiccups.
            setTimeout(connect, 1000);
        });
        ws.addEventListener("error", function (e) {
            console.warn("[multiplex] ws error", e);
        });
    }
    connect();

    function send(state) {
        var frame = JSON.stringify({
            secret: cfg.secret,
            socketId: cfg.id,
            state: state,
        });
        if (ready) ws.send(frame);
        else queue.push(frame);
    }

    // Push state on every interesting Reveal event. Reveal.getState()
    // returns the {indexh, indexv, indexf, paused, overview} bag that
    // viewers feed into Reveal.setState().
    var events = ["slidechanged", "fragmentshown", "fragmenthidden",
                  "overviewshown", "overviewhidden", "paused", "resumed"];
    events.forEach(function (ev) {
        Reveal.on(ev, function () { send(Reveal.getState()); });
    });

    // Send an initial state so a viewer who joins after the talk has
    // already started lands on the right slide.
    Reveal.on("ready", function () { send(Reveal.getState()); });
})();
