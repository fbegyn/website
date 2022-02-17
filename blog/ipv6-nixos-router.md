---
title: IPv6 on a NixOS router
date: 2022-02-18
tags: [ linux, networking, nixos, router, ipv6 ]
draft: true
---

**DISCLAIMER: This is a WIP and is not finished yet**

# IPv6 on a NixOS router

I've written before on how to use [a NixOS
router](https://francis.begyn.be/blog/nixos-home-router) for my home network.
Since that has been successful so far, I wanted to look into enabling IPv6 on my
home network and since my ISP offers it, I might as well use it.

First off all, I needed to click on a checkbox on my ISPs admin page to enable
IPv6 on my connection. After a few seconds my internet rebooted and it showed in
the admin page that IPv6 was now enabled on my connection.

# Convincing NixOS to use the IPv6

Great, the ISP claims that I have IPv6, now to change some settings on the NixOS
router and see if I can actually connect to the IPv6 internet.

As shown in the previous blog, my ISP uses PPPoE for my internet connection. So
lets first tell `pppd` to support IPv6.

```nix
services.pppd = {
  enable = true;
  peers = {
    isp1-pppoe = {
      autostart = true;
      enable = true;
      config = ''
        plugin rp-pppoe.so wan0

        name "<username>"
        password "<password>"

        +ipv6 ipv6cp-use-ipaddr

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

Well, that seems simple. The `+ipv6` should be pretty self explanatory. The
`ipv6cp-use-paddr`[1](https://linux.die.net/man/8/pppd) makes it so the local
identifier for IPv6 is the local IPv4 address. When we now restart the PPPoE
connection and run `ip -6 a`, we see some IPv6 addresses appearing on the `ppp0`
interface. A quick check proves that we can connect to the IPv6 internet:

```plain
$ ping6 google.com
PING google.com (2a00:1450:4001:808::200e): 56 data bytes
64 bytes from fra02s19-in-x0e.1e100.net: icmp_seq=0 ttl=115 time=51.320 ms
64 bytes from fra02s19-in-x0e.1e100.net: icmp_seq=1 ttl=115 time=43.248 ms
64 bytes from fra02s19-in-x0e.1e100.net: icmp_seq=2 ttl=115 time=33.240 ms
64 bytes from fra02s19-in-x0e.1e100.net: icmp_seq=3 ttl=115 time=43.208 ms
^C--- google.com ping statistics ---
4 packets transmitted, 4 packets received, 0% packet loss
round-trip min/avg/max/stddev = 33.240/42.754/51.320/6.410 ms
```

Nice, our router is connected to the IPv6 internet now. But it gets this IP
assigned through the `pppd` daemon. Since I want some more control over this,
lets assign one through `dhcpcd`. This requires some tweaking to our `networking`
settings of the NixOS router.

```nix
networking = {
   ...
   interfaces = {
     ...
     ppp0 = {useDHCP = true; } # enable dhcpcd on this interface
   };
   dhcpcd = {
     enable = true;
     # Do not remove interface configuration on shutdown.
     persistent = true;
     allowInterfaces = [ "ppp0" ];
     extraConfig = ''
       # don't touch our DNS settings
       nohook resolv.conf

       # generate a RFC 4361 complient DHCP ID
       duid

       # We don't want to expose our hw addr from the router to the internet,
       # so we generate a RFC7217 address.
       slaac private

       # we only want to handle IPv6 with dhcpcd, the IPv4 is still done
       # through pppd daemon
       noipv6rs
       ipv6only

       # settings for the interface
       interface ppp0
         ipv6rs              # router advertisement solicitaion
         iaid 1              # interface association ID
         ia_pd 1 lan0        # request a PD and assign to interface
     '';
  };
};
```

After applying these settings we should end up in the same state as before.
Except I didn't, I lost the IPv6 settings and connectivity. After banging against
several walls for way too much time, I realized my firewall settings. I block
access to the router by default, but since I'm now requesting things from my
ISP, I need to allow certain services to my router.

I made the following modifications to the `input` chain of the router:

```nix
networking.nftables.ruleset = ''
  table inet filter {
    ...
    chain input {
        ...
        # Always allow router solicitation from any LAN.
        ip6 nexthdr icmpv6 icmpv6 type nd-router-solicit counter accept

        # Default route via NDP.
        ip6 nexthdr icmpv6 icmpv6 type nd-router-advert counter accept

        # DHCPv6
        udp dport dhcpv6-client udp sport dhcpv6-server counter accept comment "IPv6 DHCP"
    }
    ...
  }
'';
```

After applying these settings, I did end up in the same state as before: the
router had IPv6 connectivity.

```plain
ping6 google.com
PING google.com (2a00:1450:4001:808::200e): 56 data bytes
64 bytes from fra02s19-in-x0e.1e100.net: icmp_seq=0 ttl=115 time=32.619 ms
64 bytes from fra02s19-in-x0e.1e100.net: icmp_seq=1 ttl=115 time=33.596 ms
64 bytes from fra02s19-in-x0e.1e100.net: icmp_seq=2 ttl=115 time=43.601 ms
64 bytes from fra02s19-in-x0e.1e100.net: icmp_seq=3 ttl=115 time=33.402 ms
^C--- google.com ping statistics ---
4 packets transmitted, 4 packets received, 0% packet loss
round-trip min/avg/max/stddev = 32.619/35.805/43.601/4.516 ms
```

Even more, where before I only had IPv6 on the `ppp0` interface, I know also had
it on the internal interface `lan0`! So all that is left now is to start up a
router advertisement daemon and all my devices should immediately have IPv6
connectivity. I picked [corerad](https://github.com/mdlayher/corerad) for this
purpose, it has some nice Prometheus metrics built in and a [simple config
file](https://corerad.net/intro/).

```nix
services.corerad = {
  enable = true;
  package = unstable.corerad;         # unstable refers to the unstable branch of nixpkgs
  settings = {
    debug = {
      address = "localhost:9430";
      prometheus = true;              # enable prometheus metrics
    };
    interfaces = [
      {
        name = "ppp0";
        monitor = false;              # see the remark below
      }
      {
        name = "lan0";
        advertise = true;
        prefix = [
          { prefix = "::/64"; }
        ];
      }
    ];
  };
};
```

These settings is all that stand between my devices and IPv6 connectivity. So
after applying these I tested out the IPv6 connectivity on my laptop and it was successful!

In a good evenings work I set up IPv6 for my home network.
