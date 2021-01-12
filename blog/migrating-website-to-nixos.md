---
title: Nixos website migration
date: 2021-01-13
tags: [ linux, nixos, website]
draft: true
---

# Nixos website migration

As of a couple of days ago, the site you're reading hsa been migrated to a Nixos
host. For those of you that don't know Nixos:

> NixOS is a Linux distribution built on top of the Nix package manager. It uses
> declarative configuration and allows reliable system upgrades.

And it broke other Linux distributions for me. It really delivers on the
declerative and reliable system upgrades. With some smart structuring and a
[github repo](https://github.com/fbegyn/nixos-configuration) it become incredibly
easy to setup new systems with configuration files like you want.
[Nixpkgs](https://github.com/NixOS/nixpkgs) also comes with a lot of software packaged.

## Migrating it: what needs to migrate?

This site used to run on a DigitalOcean droplet with Debian 9. It was a fine
little host that hosted this website and some other applications I used and
played around with. So I first decided that the only thing I want migrated in the
first place is, was that website. It had a complex setup with Traefik and the
binary that hosts this site, consequences of me using it as a test lab.
So I decided on a simpler model: NGINX proxy for the TLS termination and then the
webserver running this website.

After looking around a bit, I decided on Hetzner Cloud for the hosting. The
cheapest model they offer is cheaper then the DO droplet and they over the option
to mount the Nixos iso onto the VM. This makes installing Nixos incredibly easy
on these boxes: create the VM, mount the ISO onto it, restart and attach to the
console. You should see Nixos booting up.

Next up: the reverse proxy. Nixos makes this really easy with the
`services.nginx` and `security.acme` options, these 2 combined make setting up a
HTTPS host really straight forward. We first set `security.acme.accpetTerms =
true;` to state that we agree with the terms and then we can request certificates
by using the following scheme:

```
ecurity.acme.certs."<fqdn>".email = "<email>";
```

Then we can setup an NGINX proxy with the following scheme:

```
services.nginx.enable = true;
  services.nginx.virtualHosts."<fqdn>" = {
    forceSSL = true; # force HTTPS
    enableACME = true; # use acme for certs
    root = "/var/www/<fqdn>";
    locations."/" = {
      proxyPass = "http://localhost:3114"; # pass to the webserver running the site
    };
  };
```

Currently the site runs through a systemd services, that's also created through
Nixos:

```
systemd.services.website = {
    enable = true;
    unitConfig = {
      description = "This hosts francis.begyn.be";
    };
    serviceConfig = {
      User = "francis";
      Group = "francis";
      WorkingDirectory = "/home/francis/francis.begyn.be";
      ExecStart = "/home/francis/Go/bin/websiteserver";
    };
    wantedBy = [ "default.target" ];
    after = [ "network.target" ];
  };
```

## Future improvements

Currently a cronjob fetches the website source code and rebuilds it on the server
the old fashioned way, I want to dive a bit into it so that this can be more Nixified.
