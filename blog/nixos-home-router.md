---
title: Using Nixos as a router
date: 2021-01-28
tags: [ linux, networking, nixos, router]
draft: true
---

# Using Nixos as a router (NaaR)

So, I moved places this year and for the first couple of weeks I used the
fritzbox 7350 from my ISP. While it's good, it lacks the flexibility that I want
in my home network. I like some VLANs, some options to play with and a solid
control on my firewall.

## The hunt for a replacement

My ISP has 3 requirements for a router:

* be able to build up a PPP connection,
* support VLAN 10 tagging,
* have Gigabit Ethernet ports (otherwise, the speed will be limited to 100 Mbps).

I've used an Ubiquiti Edgerouter lite 3 before of which I'm quite happy. It
offers a lot of solid features at a good pricepoint (and with some hacking, [can
be flashed with other
OSes](https://openwrt.org/toh/ubiquiti/edgerouter.lite#flash_layout)). Some
router with good support for OpenWRT would also be a good fit.

But hearing some good things about the [PCEngine
APU](https://pcengines.ch/apu2.htm) boards and it being x86, offering most Linux
flavors as OS and having quite the community behind them, I ended up picking the
[APU2E4](https://pcengines.ch/apu2e4.htm). This mainly because the ,theoretically
more performant Intel i210AT NIC vs Intel i211AT NIC in the other models and with
3 ethernet ports I have enough.

## Installing Nixos

After waiting a couple of days, all equipment has arrived and I can start
installing Nixos. This is easy enough:

1. Construct the APU in it's enclosure and mount the mSata drive
2. Flash Nixos image on USB drive
3. Insert USB drive in APU2
4. Connect console cable to APU
5. Boot (read the notes below)
6. Select the USB drive
7. Install Nixos according to the [installation guide](https://nixos.org/manual/nixos/stable/index.html#ch-installation).

Some remarks on the installation process:
* When booting, hit tab to edit the boot entry. Normally Nixos does not output to
  serial in the boot process, so we need to enable is by appending
  `console=ttyS0,115200` to the boot entry. All characters appear twice, so just
  make sure you type it correctyl ;) . `ctrl+l` can be used to refresh the screen.
* After installing, you want to make sure that the [PCEngine
  APU](https://github.com/NixOS/nixos-hardware/blob/master/pcengines/apu/default.nix)
  entry from the Nixos hardware repo is present, as it enables the console port.
* I'm not sure if the APU would support EFI, so mine is just classic MBR. Maybe
  I'll try this in the future. It is Coreboot, so it *should*.

## Configuring as a router

So, now we have an embedded device with Nixos, so lets turn this into a router.
First thing we'll need to do is enable IP forwarding on this machine, since we'll
definitely forward packets.

```
boot.kernel.sysctl = {
  # if you use ipv4, this is all you need
  "net.ipv4.conf.all.forwarding" = true;

  # If you want to use it for ipv6
  "net.ipv6.conf.all.forwarding" = true;

  # source: https://github.com/mdlayher/homelab/blob/master/nixos/routnerr-2/configuration.nix#L52
  # By default, not automatically configure any IPv6 addresses.
  "net.ipv6.conf.all.accept_ra" = 0;
  "net.ipv6.conf.all.autoconf" = 0;
  "net.ipv6.conf.all.use_tempaddr" = 0;

  # On WAN, allow IPv6 autoconfiguration and tempory address use.
  "net.ipv6.conf.${name}.accept_ra" = 2;
  "net.ipv6.conf.${name}.autoconf" = 1;
};
```

Next up, lets configure some interfaces. The physical interfaces won't be used by
me. I need a `wan` interface to handle the WAN side of the router. It's couple to
the `wan` VLAN and doesn't use DHCP since I'll be setting up PPPoE on top of it.
And then we also have a `lan` and `iot` interface and VLAN, each we assign a
static IP to.

```
networking = {
  useDHCP = false;
  hostName = "router";
  nameserver = [ "<DNS IP>" ];
  # Define VLANS
  vlans = {
    wan = {
      id = 10;
      interface = "enp1s0";
    };
    lan = {
      id = 20;
      interface = "enp2s0";
    };
    iot = {
      id = 90;
      interface = "enp2s0";
    };
  };

  interfaces = {
    # Don't request DHCP on the physical interfaces
    enp1s0.useDHCP = false;
    enp2s0.useDHCP = false;
    enp2s0.useDHCP = false;
    
    # Handle the VLANs
    wan.useDHCP = false;
    lan = {
      ipv4.addresses = [{
        address = "10.1.1.1";
        prefixLength = 24;
      }];
    };
    iot = {
      ipv4.addresses = [{
        address = "10.1.90.1";
        prefixLength = 24;
      }];
    };
  }:
};
```

So after a `nixos-rebuild switch`, we should see all the interfaces and vlans
appear with the settings we specified. As mentioned before, I need a PPPoE
session. Luckily, Nixos makes this incredibly easy:

```
# setup pppoe session
services.pppd = {
  enable = true;
  peers = {
    edpnet = {
      # Autostart the PPPoE session on boot
      autostart = true;
      enable = true;
      config = ''
        plugin rp-pppoe.so wan
        
        # pppd supports multiple ways of entering credentials,
        # this is just 1 way
        name "${secrets.pppoe.username}"
        password "${secrets.pppoe.pass}"

        persist
        maxfail 0
        holdoff 5

        noipdefault
        defaultroute
      '';
    };
  };
};
```

When the system starts up `pppd-edpnet.service`, you should now see a `ppp0`
interface with an IP address (note: if you already have a default gateway set on
the system when the PPPoE seesion comes online, it **will not** replace the
degault gateway).  
Now onto the firewall configuration. I prefer to use
[nftables](https://wiki.nftables.org/wiki-nftables/index.php/Main_Page). So let's
disable the Nixos firewall and enable nftables.

```
networking = {
  ...
  nat.enable = false;
  firewall.enable = false;
  nftables = {
    enable = true;
  };
};
```

And then we can setup a straight forward `nftables` ruleset for a basic firewall.
The `iot` vlan will get locked down, when we add devices to this later, we'll
open op the vlan as is needed.

```
networking = {
  ...
  nftables = {
    ...
    ruleset = ''
    '';
  };
};
```

## Performance tuning

## IoT vlan: Chromecast

## Sources

* [PCEngine APU site](https://pcengines.ch/apu2e4.htm)
* [Teklager APU products](https://teklager.se/en/products/routers/apu2e4-open-source-router)
* [EDPnet site](https://www.edpnet.be/en/support/installation-and-usage/internet/learn-about-fiber-installation/i-have-a-fiber-connection-what-should-i-know-about-the-internal-cabling.html)
* [Nixos APU install notes](https://gist.github.com/tomfitzhenry/35389b0907d9c9172e5d790ca9e0d0dc)
* [nftables wiki](https://wiki.nftables.org/wiki-nftables/index.php/Main_Page)
