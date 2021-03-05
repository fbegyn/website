---
date: 2020-03-22
title: "Setting up a Wireguard VPN"
tags: ['networking','vpn']
draft: false
---

# Why do you need this?

The use cases for a VPN are extremely varied. Maybe you just want access to some systems that run
behind a firewall (without opening them to the internet). Or you just want to use the Pihole server
that you are running at home while on the road. Maybe your entire company suddenly has to work from
home and you need a quick and easy fix to offer access to the remote workers? Or just want the peace
of mind that, when you're sitting in your local bar, your internet traffic is securely sent to an
endpoint that you know.

# Why Wireguard?

I've played around a bit with some VPN services, but I personally find that Wireguard offers the most
straightforward way to set up a VPN. The tooling is easy and simple.

# Installing Wireguard

Both steps below will install `wireguard` and its dependencies onto your system. Wireguard comes
with a kernel module.

## Arch linux

On Arch linux, the installation is simple. Wireguard is available through the AUR. See the list of
packages [here](https://wiki.archlinux.org/index.php/WireGuard). For a normal Arch install, you can
just do `yay -S wireguard-arch wireguard-tools` (or use whatever AUR helper you have).

## Debian based 

For Debian-based systems, the installation is equally as simple. The `wireguard` package is available
in the unstable repos of Debian. So it can easily be enabled with the following commands.

```shell
$ echo "deb http://deb.debian.org/debian/ unstable main" | sudo tee /etc/apt/sources.list.d/unstable-wireguard.list
$ printf 'Package: *\nPin: release a=unstable\nPin-Priority: 90\n' | sudo tee /etc/apt/preferences.d/limit-unstable
$ apt update
$ apt install wireguard
```

These commands will enable the unstable repo, assign a priority to the unstable packages (determines
which package shows up if it is in both stable and unstable) and then install Wireguard onto your
system.

# Setup

The setup for Wireguard is one of the strong suits for me personally. The helper scripts that
Wireguard offers make it incredibly easy to setup a VPN server and offer secure configuration for your
clients.

## Key generation

First, we need to generate some keys for our server and the clients:

```shell
$ cd /etc/wireguard
$ mkdir keys
$ cd keys
$ umask 077; wg genkey | tee server-priv-key | wg pubkey > server-pub-key
$ umask 077; wg genkey | tee client-foo-priv-key | wg pubkey > client-foo-pub-key
$ umask 077; wg genkey | tee client-bar-priv-key | wg pubkey > client-bar-pub-key
```

If you want to ensure an additional layer of security of top of these keys, it is possible to also
add a pre-shared secret. Here I'll demonstrate it for the  `foo` client `wg genpsk > client-foo-preshared`.

The good thing about this, is that you can also request that your client generates his own key and
just sends the public key to you. This way the client can be sure that the server admin can't
'impersonate' him by using the private key of the client.

## Configs

### Server

So on the server side:

* the VPN interface will get the name of the config, in this case `wg0`
* `interface` block: defines the properties for the system itself
  * `Address`: specifies the address and netmask of the VPN interface. Here `wg0` would get `172.12.1.1/24`
  * `ListenPort`: specifies to port on which the VPN server will listen. This port needs to be
  port-forwarded from your router to the VPN server (UDP only).
  * `PrivateKey`: the content of `server-priv-key`
  * `PostUp`: this action will be executed after the `wg0` interface goes up, here we do a simple NAT
  MASQUERADE. But these can be replaced with a script that does more complex firewalling.
  * `PostDown`: reverse of `PostUp`
* `peer` block: defines the properties of the clients/peers connecting to the server. I'll expand on
  the `foo` peer block.
  * `PublicKey`: the content of `client-foo-pub-key`
  * OPTIONAL - `PresharedKey`: the content of `client-foo-preshared`
  * `AllowedIPs`: this will ensure that the peer is only allowed traffic from this IP. If it connects
  with another IP, it will not work.

An example configuration for the server side would be:

```shell
/etc/wireguard/wg0.conf
-----------------------
[Interface]
Address = 172.12.1.1/24
ListenPort = 51820
PrivateKey = SERVER_PRIVATE_KEY

# note - substitute eth0 in the following lines to match the Internet-facing interface
# if the server is behind a router and receive traffic via NAT, this iptables rules are not needed
PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

[Peer]
# foo
PublicKey = PEER_FOO_PUBLIC_KEY
PresharedKey = PRE-SHARED_KEY
AllowedIPs = 172.12.1.2/32

[Peer]
# bar
PublicKey = PEER_BAR_PUBLIC_KEY
AllowedIPs = 172.12.1.3/32
```

After we've configured the server side, we can enable the interface by running `sudo wg-quick up wg0`.
If you're running a systemd based system, you can also enable and start the service by running `sudo
systemctl start wg-quick@<config name>`. So for the example above it would be `sudo systemctl start
wg-quick@wg0`

### Client foo

Now let's look over the config for the `foo` client.

* `interface` block: describes the behaviour of the system the interface is created on.
  * `Address`: specifies the address and netmask of the VPN interface. Here `wg0` would get `172.12.1.1/24`
  * `PrivateKey`: the content of `client-foo-priv-key`
  * OPTIONAL - `DNS`: sets the DNS server of the system to this when the tunnel is up. Also gets
  removed when stopping the VPN connection with `wg-quick down <config name>`
* `peer` block: defines the properties of the clients/peers connecting to the server. I'll expand on
  the `foo` peer block.
  * `PublicKey`: the content of `server-pub-key`
  * OPTIONAL - `PresharedKey`: the content of `client-foo-preshared`
  * `AllowedIPs`: this determines which traffic get send over the VPN. `0.0.0.0/0` and `::/0` are
  wild cards that will route all traffic over the VPN. If you only want to route traffic to your
  local network over the VPN, you can do that by specifying the CIDR of the network here.
  * `Endpoint`: the public access for the VPN server from previous section. This would be the WAN IP
  of your router or the DNS associated with it.

Below is an example configuration for a client/peer of the VPN server above:

```shell
foo.conf
----------------
[Interface]
Address = 172.12.1.2/24
PrivateKey = PEER_FOO_PRIVATE_KEY

[Peer]
PublicKey = SERVER_PUBLICKEY
PresharedKey = PRE-SHARED_KEY
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = my.ddns.example.com:51820
```

### Client bar

The client `bar` has almost identical setup as the client `foo`. It does not have the pre-shared
secret and also only routes the `172.13.0.0/16` network over the VPN.

```shell
bar.conf
----------------
[Interface]
Address = 172.12.1.3/24
PrivateKey = PEER_BAR_PRIVATE_KEY
DNS = 172.13.1.2  # DNS server on local network 172.13.0.0/16

[Peer]
PublicKey = SERVER_PUBLICKEY
PresharedKey = PRE-SHARED KEY
AllowedIPs = 172.13.0.0/16
Endpoint = my.ddns.example.com:51820
```

# Some remarks

* Pick a high random number as for the port for your VPN server, preferably not something you can find
  in any of the examples
* Personally I prefer that my clients generate their own key pairs and just let me know the public key.
* Secure firewalling on the server side can prevent your VPN clients from accessing parts of the
  network that they shouldn't.
* Revoking a clients access is as easy as removing the peer from the server config.

# Sources:

* [Arch linux wiki Wireguard](https://wiki.archlinux.org/index.php/WireGuard)
* [Debian wiki wireguard](https://wiki.debian.org/Wireguard)
