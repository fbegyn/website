// client.js
//
// Reveal.js multiplex viewer. Connects to the Go multiplex server's
// SSE endpoint for the configured socketId and applies state changes
// to Reveal as they arrive.
//
// Expects window.RevealMultiplex = { id, url } before this script
// runs (set inline in the template).

(function () {
    var cfg = window.RevealMultiplex || {};
    if (!cfg.id) {
        console.warn("[multiplex] missing socket id, viewer disabled");
        return;
    }

    var origin = (cfg.url || window.location.origin).replace(/\/+$/, "");
    var sseURL = origin + "/multiplex/" + encodeURIComponent(cfg.id) + "/events";

    function apply(payload) {
        try {
            var msg = JSON.parse(payload);
            if (msg && msg.state) Reveal.setState(msg.state);
        } catch (e) {
            console.warn("[multiplex] bad payload", e, payload);
        }
    }

    function connect() {
        var es = new EventSource(sseURL);
        es.onmessage = function (ev) { apply(ev.data); };
        es.onerror = function () {
            // EventSource auto-reconnects; just log so a presenter
            // looking at the audience screen can spot trouble.
            console.warn("[multiplex] sse error, will retry");
        };
    }

    if (typeof Reveal !== "undefined" && Reveal.isReady && Reveal.isReady()) {
        connect();
    } else {
        Reveal.on("ready", connect);
    }
})();
