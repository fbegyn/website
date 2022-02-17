---
title: Using Nixos as a router
date: 2021-02-07
tags: [ linux, networking, nixos, router]
draft: false
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
APU2](https://pcengines.ch/apu2.htm) boards (and it's relatives) and it being
x86, offering most Linux flavors as OS and having quite the community behind
them, I ended up picking the [APU2E4](https://pcengines.ch/apu2e4.htm). This
mainly because the ,theoretically more performant Intel i210AT NIC vs Intel
i211AT NIC in the other models and with 3 ethernet ports I have enough.

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

## Configuring as a router

So, now we have an embedded device with Nixos, so lets turn this into a router.
First thing we'll need to do is enable IP forwarding on this machine, since we'll
definitely forward packets.

```nix
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

```nix
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

```nix
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

```nix
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

```nix
networking = {
  ...
  nftables = {
    ...
    ruleset = ''
      table ip filter {
        # enable flow offloading for better throughput
        flowtable f {
          hook ingress priority 0;
          devices = { ppp0, lan };
        }

        chain output {
          type filter hook output priority 100; policy accept;
        }

        chain input {
          type filter hook input priority filter; policy drop;

          # Allow trusted networks to access the router
          iifname {
            "lan",
          } counter accept

          # Allow returning traffic from ppp0 and drop everthing else
          iifname "ppp0" ct state { established, related } counter accept
          iifname "ppp0" drop
        }
        
        chain forward {
          type filter hook forward priority filter; policy drop;

          # enable flow offloading for better throughput
          ip protocol { tcp, udp } flow offload @f

          # Allow trusted network WAN access
          iifname {
                  "lan",
          } oifname {
                  "ppp0",
          } counter accept comment "Allow trusted LAN to WAN"

          # Allow established WAN to return
          iifname {
                  "ppp0",
          } oifname {
                  "lan",
          } ct state established,related counter accept comment "Allow established back to LANs"
        }
      }

      table ip nat {
        chain prerouting {
          type nat hook output priority filter; policy accept;
        }

        # Setup NAT masquerading on the ppp0 interface
        chain postrouting {
          type nat hook postrouting priority filter; policy accept;
          oifname "ppp0" masquerade
        } 
      }
    '';
  };
};
```

This should cover a fairly basic `nftables` ruleset that offers internet
connectivity to `lan` and locks `iot` completely to local connections only. At
the end, lets install some handy packages for a router.

```nix
environment.systemPackages = with pkgs; [
  vim                 # my preferred editor
  htop                # to see the system load
  ppp                 # for some manual debugging of pppd
  ethtool             # manage NIC settings (offload, NIC feeatures, ...)
  tcpdump             # view network traffic
  conntrack-tools     # view network connection states
];
```

## DHCP server

So, now that we got the routing part set up, we need to make sure that devices
that plug into these networks can get some IP addresses. For this, we spin up a
quick DHCP server on the router (or any other compute connected to the networks).

```nix
services.dhcpd4 = {
    enable = true;
    interfaces = [ "lan" "iot" ];
    extraConfig = ''
      option domain-name-servers 10.5.1.10, 1.1.1.1;
      option subnet-mask 255.255.255.0;

      subnet 10.1.1.0 netmask 255.255.255.0 {
        option broadcast-address 10.1.1.255;
        option routers 10.1.1.1;
        interface lan;
        range 10.1.1.128 10.1.1.254;
      }

      subnet 10.1.90.0 netmask 255.255.255.0 {
        option broadcast-address 10.1.90.255;
        option routers 10.1.90.1;
        option domain-name-servers 10.1.1.10;
        interface iot;
        range 10.1.90.128 10.1.90.254;
      }
    '';
  };
```

## Performance tuning

The squeeze the most out of this timy box, I played around a bit with some
interrupt steering and packet handling settings. After a few minutes I found what
seemed to work optimally for me and wrote up this little script that is called at
startup. It sets the [smp
affinity](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/6/html/performance_tuning_guide/s-cpu-irq)
and
[RPS](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/6/html/performance_tuning_guide/network-rps).
Generally, you want to avoid CPU core handling the HW interrupts from SMP
affinity for RPS.

```shell
#! /usr/bin/env sh

smp1=8
rps1=7
smp2=8
rps2=7

# set balancer for enp1s0
echo ${smp1} > /proc/irq/36/smp_affinity
echo ${smp1} > /proc/irq/37/smp_affinity
echo ${smp1} > /proc/irq/38/smp_affinity
echo ${smp1} > /proc/irq/39/smp_affinity
echo ${smp1} > /proc/irq/40/smp_affinity

# set rps for enp1s0
echo ${rps1} > /sys/class/net/enp1s0/queues/rx-0/rps_cpus
echo ${rps1} > /sys/class/net/enp1s0/queues/rx-1/rps_cpus
echo ${rps1} > /sys/class/net/enp1s0/queues/rx-2/rps_cpus
echo ${rps1} > /sys/class/net/enp1s0/queues/rx-3/rps_cpus

# set balancer for enp2s0
echo ${smp2} > /proc/irq/42/smp_affinity
echo ${smp2} > /proc/irq/43/smp_affinity
echo ${smp2} > /proc/irq/44/smp_affinity
echo ${smp2} > /proc/irq/45/smp_affinity
echo ${smp2} > /proc/irq/46/smp_affinity

# set rps for enp2s0
echo ${rps2} > /sys/class/net/enp2s0/queues/rx-0/rps_cpus
echo ${rps2} > /sys/class/net/enp2s0/queues/rx-1/rps_cpus
echo ${rps2} > /sys/class/net/enp2s0/queues/rx-2/rps_cpus
echo ${rps2} > /sys/class/net/enp2s0/queues/rx-3/rps_cpus
```

## IoT vlan: Chromecast

So, first device for the `iot` vlan is a chromecast. I want to be able to use
this chromecast as you normally can, this means that when I am connected to the
network I can cast content to the chromecast. After reading through
[some](https://baihuqian.github.io/2020-12-13-secure-home-network-using-chromecast-across-vlans/)
blog post and asking around a bit, I determined the following:

* TCP ports `8008-8009` to the chromecast
* high UDP ports `32768-61000` to and from the chromecast
* mDNS is being used coming from the chromecast

To handle the mDNS, I set up a mDNS reflector using Avahi. Again, Nixos makes
this almost too easy:

```nix
services.avahi = {
  enable = true;
  reflector = true;
  interfaces = [
    "lan"
    "iot"
  ];
};
```

And then I modified the `input` chain on the router where the Avahi service is
running:

```
chain input {
  ...
  # Accept mDNS for avahi reflection
  iifname "iot" ip saddr <chromecast IP> tcp dport { llmnr } counter accept
  iifname "iot" ip saddr <chromecast IP> udp dport { mdns, llmnr } counter accept
  ...
}
```

This will make the chromecast show up on all devices, but the casting will fail
since the `iot` vlan is still locked down. So, we need to allow the chromecast to
access the internet and communicate with the devices on the `lan` network
(uncertain about this, I need to play around a bit more the implement the first 2
rules I mentioned at the beginning of this chapter). For this, we modify the
`forward` chain of the firewall:

```
chain forward {
  # Chromecast
  iifname "iot" oifname "ppp0" ip saddr <chromecast IP> tcp dport { 80, 443 } counter accept
  iifname "iot" ip saddr <chromecast IP> oifname { "enp2s0", "lan0", "wifi" } counter accept
}
```

# Update 2022-02-18

I've updated the blog a couple of times in the past few months and hopefully
filtered out some issues and errors. I also had a specific question about some
`nftables` error:

```
Error: Could not process rule: No such file or directory
ip protocol { tcp, udp } flow offload @f
```

I had to dive into some notes to find what this was about since I did remember
encoutering it myself. As it turns out, this can occur when some error exists in
the `nftables` configuration. In my case this was because `nftables ` was applied
way before `ppp0` interface was online. Since this interface was mentioned in the
flowtable (and didn't exist yet), the flowtable configuration was rejected by
`nftables` and that causes the error.

So if you encounter this error, double check the config and order in which the
configurations are applied.

## Sources

* [PCEngine APU site](https://pcengines.ch/apu2e4.htm)
* [Teklager APU products](https://teklager.se/en/products/routers/apu2e4-open-source-router)
* [EDPnet site](https://www.edpnet.be/en/support/installation-and-usage/internet/learn-about-fiber-installation/i-have-a-fiber-connection-what-should-i-know-about-the-internal-cabling.html)
* [Nixos APU install notes](https://gist.github.com/tomfitzhenry/35389b0907d9c9172e5d790ca9e0d0dc)
* [nftables wiki](https://wiki.nftables.org/wiki-nftables/index.php/Main_Page)
* [nftables flow offloading](https://wiki.nftables.org/wiki-nftables/index.php/Flowtable)
