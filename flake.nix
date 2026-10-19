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
    nixpkgs,
    flake-utils,
    devshell,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
        # config = import ./go.nix;
        overlays = [ devshell.overlays.default ];
      };
    in rec {
      defaultPackage = pkgs.buildGoModule {
        name = "website";
        pname = "server";
        src = ./.;
        vendorHash = "sha256-wGCPHY6eNztX+iVIVTeanuYrud2uCIicUluA+Nz2/JU=";
        ldflags = [
          "-s" "-w"
        ];
        # internal/multiplex tests hang in the nix build sandbox
        doCheck = false;
        subPackages = [];
      };
      devShells = rec {
        default = website;
        website = pkgs.devshell.mkShell {
          name = "website";
          packages = with pkgs; [
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
            nodejs
            deno
            rsync
            openssh
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
            {
              name = "website:deploy";
              help = "website:deploy <user>@<host> <root directory>";
              command = ''
                ./scripts/website-deploy "$@"
              '';
            }
          ];
        };
      };
      packages.website-deploy = pkgs.writeShellApplication {
        name = "website-deploy";
        runtimeInputs = with pkgs; [ git rsync openssh ];
        text = builtins.readFile ./scripts/website-deploy;
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
            workingDirectory = mkOption {
              type = types.str;
              default = "/srv/fbegyn/website/production";
              example = "/var/lib/website/staging";
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
            drafts = mkOption {
              type = types.bool;
              default = false;
              example = true;
              description = "Render drafts on the port+1 port and expose under drafts.domain";
            };
          };

          config = mkIf cfg.enable {
            users.users.fbegyn = {
              createHome = true;
              isSystemUser = true;
              group = "fbegyn";
              home = "${cfg.workingDirectory}";
              description = "francis.begyn.be";
            };
            users.groups.fbegyn.members = [ "francis" ];
            systemd.services.website = {
              serviceConfig = {
                Environment = "SERVER_PORT=${toString cfg.port}";
                EnvironmentFile = "${cfg.workingDirectory}/.env";
                User = "fbegyn";
                Group = "fbegyn";
                WorkingDirectory = "${cfg.workingDirectory}";
                ExecStart = "${defaultPackage}/bin/server serve";
                ExecReload = "${pkgs.coreutils}/bin/kill -HUP $MAINPID";
              };
              wantedBy = [ "multi-user.target" ];
              after = [ "network.target" ];
            };
            systemd.services.website-draft = mkIf cfg.drafts {
              serviceConfig = {
                Environment = "SERVER_PORT=${toString cfg.port}";
                EnvironmentFile = "${cfg.workingDirectory}/.env";
                User = "fbegyn";
                Group = "fbegyn";
                WorkingDirectory = "${cfg.workingDirectory}";
                ExecStart = "${defaultPackage}/bin/server serve --drafts --port ${toString (cfg.port+1)}";
                ExecReload = "${pkgs.coreutils}/bin/kill -HUP $MAINPID";
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
            services.nginx.virtualHosts.francis-drafts = mkIf cfg.drafts {
              forceSSL = true;
              serverName = "drafts.${cfg.domain}";
              useACMEHost = "${cfg.useACMEHost}";
              root = "/var/www/${cfg.domain}";
              locations."/" = {
                proxyPass = "http://127.0.0.1:${toString (cfg.port+1)}";
                extraConfig = ''
                  add_header Permissions-Policy interest-cohort=();
                '';
              };
            };
          };
        };
    });
}
