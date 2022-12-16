---
title: Using Tailscale
date: 2022-12-16
tags: [vpn, tailscale, wireguard, nixos]
draft: true
---

# Tailscale: the VPN almost indistinguishable from magic

I've been using [Tailscale](https://tailscale.com/) as my VPN for a LONG time. So
it's about time that I wrote something about it. My title references one of
[Arthur C. Clarke's 3 laws](https://en.wikipedia.org/wiki/Clarke%27s_three_laws):

> Any sufficiently advanced technology is indistinguishable from magic.

I used this because Tailscale is the first VPN provider that I used that really
embodies that statement, mainly because it just works.

## It just works

Before Tailscale, I ran my own stitched together
[Wireguard](https://www.wireguard.com/) network thak mostly served my purpose.
And before that, I ran iterations of
[IPsec](https://nl.wikipedia.org/wiki/IPsec), [OpenVPN](https://openvpn.net/) and
a variety of [ssh](https://goteleport.com/blog/ssh-tunneling-explained/) based
[tunnel](https://github.com/sshuttle/sshuttle) software. While I had fun and
learned several things playing around with all this technology, it always had
it's problems. The central server got borked, the network didn't allow certain
ports or methods, NAT failues, ... the batlle was endless. Even with the
Wireguard setup, which massively simplified exchanging keys and where I used
multiple lightweight tunnels to connect things, these issues still occured.

Even other VPN providers that were based on these technologies had issues.
[Mullvad](https://mullvad.net/en/) worked relatively well, but then at the time
lacked some features I needed.

Meanwhile, from day 1, Tailscale just worked. Go the site, create an account,
install the programs and follow the quick start. That's all, I could immediately
connect to all my devices that I connect with Tailscale.

## It's easy

And connecting those devices was very easy. If it could be done interactively,
just type `tailscale up` on the device, it prompts a login URL. You then simply
go the the URL, sign in with your account and seconds later the device is online
and shows up in the web interface.

If you then type `tailscale status`, you get an overview of the devices.

## It has features ... and it's growing

Aside from connecting devices together in your own little private network (or
[tailnet](https://tailscale.com/kb/1136/tailnet/)), Tailscale can DO a lot more
these days.

Connecting these devices together is backed by an incredibly [powerful ACL
structure](https://tailscale.com/kb/1018/acls/) where you have full control of
who can access what. Not only can you create [hosts]() and [groups](), but you
can also write [tests]() for this logic. I can definitely see a use case for this
in businesses where multiple people might modify the ACL or on my own, to ensure
that I don't accidently share my highly sensitive server with my friend!

"Share server with your friend?" you ask. YES, with Tailscale you can [share
servers accross account/Tailnets](https://tailscale.com/kb/1084/sharing/)! This
makes it easy to share your FTP or
[minecraft](https://tailscale.com/kb/1137/minecraft/) server with friends and
family without ever having to expose those servers to the Internet.

Aside from that, [Taildrop](https://tailscale.com/kb/1106/taildrop/) makes
sharing file incredibly easy and the [Tailscale
SSH](https://tailscale.com/kb/1193/tailscale-ssh/) functionality can easily
secure your server by allowing SSH from authenticated Tailscale accounts only.

These are only a short summary of the features that I've used, but there are
[plenty](https://tailscale.com/kb/1153/enabling-https/)
[more](https://tailscale.com/kb/1223/tailscale-funnel/) to discover and play
with!

## How I use it

Personally I use Nixos, so I took some time recently to update and create a small
[Nixos
module](https://github.com/fbegyn/nixos-configuration/blob/main/services/tailscale.nix)
that can be used to setup and run Tailscale. It's largely inspired by [this
blogpost](https://tailscale.com/blog/nixos-minecraft/) and offers me some
flexibility to what I can configure on hosts. So I can easily expose a certain
subnet over Tailscale or enable the Tailscale SSH functionality.

## Sources

* [Tailscale](https://tailscale.com/)
* [Tailscale docs](https://tailscale.com/kb/)
* [Minecraft on Nixos](https://tailscale.com/blog/nixos-minecraft/)
