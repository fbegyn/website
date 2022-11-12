{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    flake-compat = {
      url = "github:edolstra/flake-compat";
      flake = false;
    };
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    ...
  } @ inputs:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
        config = import ./go.nix;
      };
    in rec {
      defaultPackage = pkgs.buildGoModule {
        name = "website";
        src = pkgs.stdenv.mkDerivation {
          name = "gosrc";
          srcs = [ ./go.mod ./go.sum ./cmd ];
          phases = "installPhase";
          installPhase = ''
            mkdir $out
            for src in $srcs; do
              for srcFile in $src; do
                cp -r $srcFile $out/$(stripHash $srcFile)
              done
            done
          '';
        };
        CGO_ENABLED = 0;
        vendorSha256 = null;
        ldFlages = [
          "-S" "-W"
        ];
      };
      devShell = pkgs.mkShell rec {
        buildInputs = with pkgs; [
          go
          nix
          git
          gotools
          go-tools
          gotestsum
          gofumpt
          golangci-lint
          nfpm
          goreleaser
        ];
      };
    });
}
