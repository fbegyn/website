{ pkgs ? import <nixpkgs> {} }:

let
  srcNoTarget = dir:
    builtins.filterSource
    (path: type: type != "directory" || builtins.baseNameOf path != "target")
    dir;
  src = ./.;
in pkgs.stdenv.mkDerivation {
  name = "website";
  version = "HEAD";
  inherit src;

  buildInputs = [
    pkgs.go
  ];

  buildPhase = ''
    ${pkgs.go}/bin/go get -d ./
    ls -al ./
  '';

  installPhase = ''
    mkdir -p $out/bin
    cp -rf ./bin/website $out/bin/website
  '';
}
