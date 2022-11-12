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
      nixpkgs = import nixpkgs {
        inherit system;
        config = import go.nix;
      };
    in rec {
      packages = {
        website = buildGoModule {
          name = "website";
          src = ./;
          CGO_ENABLED = 0;
          vendorSha256 = "";
          subPackages = [];
          ldFlages = [
            "-S" "-W"
          ];
        };
      };
      defaultPackage = packages.website;
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
