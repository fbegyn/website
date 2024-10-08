{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    devshell = {
      url = "github:numtide/devshell";
      inputs = {
        nixpkgs.follows = "nixpkgs";
      };
    };
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    devshell,
    ...
  } @ inputs:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
        config = import ./go.nix;
        overlays = [ devshell.overlays.default ];
      };
    in rec {
      defaultPackage = pkgs.buildGoModule {
        name = "website";
        pname = "server";
        src = ./.;
        CGO_ENABLED = 0;
        vendorHash = "sha256-8w2zS+3YyBD+mTBXc3l574ru0gPWTmZtjwDm8cRqjYg";
        ldFlages = [
          "-S" "-W"
        ];
        subPackages = [];
      };
      devShells = rec {
        default = website;
        website = pkgs.devshell.mkShell {
          name = "website";
          packages = with pkgs; [
            go_1_23
            nix
            git
            gotools
            go-tools
            gotestsum
            gofumpt
            golangci-lint
            nfpm
            goreleaser
            nodejs
            deno
          ];
          commands = [
            {
              name = "tailwind:watch";
              command = ''
                npx tailwindcss -i ./static/css/begyn.css -o static/css/output.css --watch
              '';
            }
            {
              name = "website:build";
              command = ''
                make clean
                make build
              '';
            }
            {
              name = "website:package";
              command = ''
                make package
              '';
            }
          ];
        };
      };
      nixosModules.website = { config, lib, pkgs, ...}:
        with lib;
        let
          cfg = config.services.fbegyn.website;
        in {
          options.services.fbegyn.website = {
            enable = mkEnableOption "enables fbegyn's personal website server";
            domain = mkOption {
              type = types.str;
              default = "francis.begyn.be";
              example = "francis.begyn.be";
              description = "The domain NGINX should use.";
            };
            useACMEHost = mkOption {
              type = types.str;
              default = "francis.begyn.be";
              example = "francis.begyn.be";
              description = "The ACME host that should be used by NGINX";
            };
            home = mkOption {
              type = types.str;
              default = "/srv/fbegyn/website";
              example = "/var/lib/website";
              description = "Working directory of the website";
            };
            aliases = mkOption {
              type = types.listOf types.str;
              default = [];
              example = [ "francis.begyn.eu" ];
              description = "The aliases NGINX should use.";
            };
            port = mkOption {
              type = types.int;
              default = 3114;
              example = 3114;
              description = "The port number for the website server";
            };
          };

          config = mkIf cfg.enable {
            users.users.fbegyn = {
              createHome = true;
              isSystemUser = true;
              group = "fbegyn";
              home = "${cfg.home}";
              description = "francis.begyn.be";
            };
            users.groups.fbegyn.members = [ "francis" ];
            systemd.services.website = {
              enable = true;
              serviceConfig = {
                Environment = "SERVER_PORT=${toString cfg.port}";
                EnvironmentFile = "/srv/fbegyn/website/.env";
                User = "fbegyn";
                Group = "fbegyn";
                WorkingDirectory = "${cfg.home}";
                ExecStart = "${defaultPackage}/bin/server";
              };
              wantedBy = [ "multi-user.target" ];
              after = [ "network.target" ];
            };
            # francis.begyn.be website/blog
            services.nginx.virtualHosts.francis = {
              forceSSL = true;
              serverName = "${cfg.domain}";
              serverAliases = cfg.aliases;
              useACMEHost = "${cfg.useACMEHost}";
              root = "/var/www/${cfg.domain}";
              locations."/" = {
                proxyPass = "http://127.0.0.1:${toString cfg.port}";
                extraConfig = ''
                  add_header Permissions-Policy interest-cohort=();
                '';
              };
            };
          };
        };
    });
}
