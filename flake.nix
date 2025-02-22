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
        env.CGO_ENABLED = 0;
        vendorHash = "sha256-Em+JgHXYgcy8GLNCVDEqNPuJA9BAqbDE22bcfsbAcJE=";
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
              name = "website:build:bin";
              command = ''
                make clean
                make build
              '';
            }
            {
              name = "website:build:nix";
              command = ''
                nix build
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
            multiplex = {
              enable = mkEnableOption "enables the multiplex server on this host under multiplex-server.service";
              location = mkOption {
                type = types.str;
                default = "/socket.io";
                example = "/multiplex/socket";
                description = "nginx location to run the multiplex server on";
              };
              command = mkOption {
                type = types.str;
                description = "command to execute under de multiplex server systemd unit";
              };
              port = mkOption {
                type = types.int;
                default = 8000;
                example = 8000;
                description = "The port number for the multiplex server";
              };
              home = mkOption {
                type = types.str;
                default = "/srv/fbegyn/multiplex-ts-server";
                example = "/var/lib/website";
                description = "Working directory of the website";
              };
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
              serviceConfig = {
                Environment = "SERVER_PORT=${toString cfg.port}";
                EnvironmentFile = "${cfg.home}/.env";
                User = "fbegyn";
                Group = "fbegyn";
                WorkingDirectory = "${cfg.home}";
                ExecStart = "${defaultPackage}/bin/server serve";
              };
              wantedBy = [ "multi-user.target" ];
              after = [ "network.target" ];
            };
            systemd.services.multiplex-server = mkIf cfg.multiplex.enable {
              serviceConfig = {
                EnvironmentFile = "${cfg.multiplex.home}/.env";
                User = "fbegyn";
                Group = "fbegyn";
                WorkingDirectory = "${cfg.multiplex.home}";
                ExecStart = "${cfg.multiplex.command}";
              };
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
              locations."${cfg.multiplex.location}" = mkIf cfg.multiplex.enable {
                proxyPass = "http://127.0.0.1:${toString cfg.multiplex.port}";
                extraConfig = ''
                  add_header Permissions-Policy interest-cohort=();
                '';
              };
            };
          };
        };
    });
}
