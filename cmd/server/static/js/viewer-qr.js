url = document.getElementById("qrcode-link").href;
console.log(url);

new QRCode(document.getElementById("qrcode"),
  url
);
