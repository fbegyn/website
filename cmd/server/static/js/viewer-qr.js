// viewer-qr.js
//
// Render a QR code into #qrcode whose payload is the absolute URL of
// the #qrcode-link anchor. Uses qrcodejs (loaded just before this).
//
// We resolve to an absolute URL via the URL constructor so the QR is
// scannable regardless of where the page is mounted.
(function () {
    var anchor = document.getElementById("qrcode-link");
    var target = document.getElementById("qrcode");
    if (!anchor || !target) return;
    var absolute = new URL(anchor.getAttribute("href"), window.location.origin).toString();
    new QRCode(target, {
        text: absolute,
        width: 320,
        height: 320,
        correctLevel: QRCode.CorrectLevel.M,
    });
})();
