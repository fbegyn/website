---
date: 2018-08-18
title: "DSCP, what is it and what does it do?"
tags: [dscp,qos,tos,networking]
---
# DSCP?

First of all, what is DSCP?

DSCP stand for Differentiated Services Code Point and  is used to identify "services" straight from the IP header. DSCP is carried in the Differentiated Services (DS) field and together with the ECN field (which occupies the lower 2 bits) is carried in the legacy ToS field of the IP header.

The DS field occupies the upper 6 bits of a byte and thus has 64 possible Code Points (DSCP) it can carry. Most commonly used DSCP values are standardised into 3 pools:

* xxxxx0  Standards Action
* xxxx11  Experimental or Local Use	Reserved for experimental or Local Use
* xxxx01  Standards Action

I'll list the standards actions below so we have an overview of them.

|Name       |Binary  |Decimal |
|-----------|:-------|:-------|
|CS0        |000000  |0       |
|CS1        |001000  |8       |
|CS2        |010000  |16      |
|CS3        |011000  |24      |
|CS4        |100000  |32      |
|CS5        |101000  |40      |
|CS6        |110000  |48      |
|CS7        |111000  |56      |
|AF11       |001010  |10      |
|AF12       |001100  |12      |
|AF13       |001110  |14      |
|AF21       |010010  |18      |
|AF22       |010100  |20      |
|AF23       |010110  |22      |
|AF31       |011010  |26      |
|AF32       |011100  |28      |
|AF33       |011110  |30      |
|AF41       |100010  |34      |
|AF42       |100100  |36      |
|AF43       |100110  |38      |
|EF         |101110  |46      |
|VOICE-ADMIT|101100  |44      |

Same translations from the table above:

* CS: Class Selector - marks traffic as a certain class (Network control, realtime, ...)
* AF: Assured Forwarding - when an AF class is overloaded, preferentially discard packets. These classes differentiate in the drop probability, but the [wikipedia](https://en.wikipedia.org/wiki/Differentiated_services#Assured_Forwarding) sums it up much better then I ever could.
* EF: Expedited forwarding - priority access to a link

Everything aside from these standard can be used to your own whims, even the standards code points can be used to your own whims (but that's bad practice, so lets not do that).

# How is DSCP used?

When a packet enters a router, the router reads the DSCP value. Based on the DSCP value the packet gets categorised into one 64 behaviours (these are called Per Hop Behaviour or PHB). A PHB group is not unique to a DSCP, as multiple DSCPs can share the same PHB group. Each PHB group has a queue in which packets are stored prior to forwarding.

DSCP is also designed to be backwards compatible with ToS. So non-DSCP, ToS compliant HW can use DSCP by making smart use of the ToS field.

# Conversions

So how does one go from ToS to DSCP and back again?

As mentioned before, DSCP and ECN together form a byte, in which DSCP uses the upper 6 and ECN the lower 2. ToS on it's own is a complete byte.
```
|xxxxxx |  xx |
| DSCP  | ECN |
|     ToS     |
```

So the conversions are straightforward knowing this:

* ToS -> DSCP: `ToS value >> 2`
* DSCP -> ToS: `DSCP value << 2` (that is assuming the ECN is set to `00`, otherwise see the above schematic and common sense to determine the ToS value).

# Usefull things when working with DSCP

Ping has to option to set a ToS field (and after this blog, you know how to use this to set a DSCP value). `ping -Q <ToS value> <target>`. You can use both decimal/hexadecimal notation for ipv4, but ipv6 only accepts ipv6.

Fping is also useful, here you can use `fping -O <ToS value> <target>` to set the ToS.

Capturing packets with specific DSCP value can be done with `tcpdump`. `tcpdump -v -n -i ppp0 'ip and (ip[1] & 0xfc) >> 2 == 0x12'` for example captures DSCP 0x12 or 18 packets. You can also use it to capture a specific ToS packet with `tcpdump -v -n -i ppp0 'ip and ip[1] & 0xfc == 72'`. (Credits for this paragraph to [this page](https://www.tucny.com/home/dscp-tos))
