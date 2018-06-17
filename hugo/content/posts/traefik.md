+++ 
draft = false
date = 2018-06-17T00:30:40+02:00
title = "Setting up a reverse proxy, traefik"
slug = "traefik" 
tags = ["treafik","reverse proxy","web","go"]
categories = ["go","web","server"]
+++

So I run a few services at home, primarily for testing and development only. I simply do not have the infrastructure or network to do so for anything I deem important.
That said, being able to test some things quickly on a system where I have to care little about introducing breaking changes is very handy (not that I often do, just when it happens I don't do it on production :p ).

That said, it was time to set up a reverse proxy for some services that I have running because they are recurrent things: Grafana, Prometheus and Alertmanager are things that I almost continiously have running (or try to at least).
Aside from those I have some TICK stuff running that I like to play with and use for certain side projects. So to make all of these a little bit more accesable, why no setup a reverse proxy?

# Reverse proxy?

According to [wikipedia](https://en.wikipedia.org/wiki/Reverse_proxy)

> In computer networks, a reverse proxy is a type of proxy server that retrieves resources on behalf of a client from one or more servers. These resources are then returned to the client as if they originated from the Web server itself.[1] Unlike a forward proxy, which is an intermediary for its associated clients to contact any server, a reverse proxy is an intermediary for its associated servers to be contacted by any client.

Simply said, a reverse proxy is an intermediate server that handles requests from clients and makes sure the requests get to the correct server, without exposing that server directly to the client.

Case: I have a prometheus server running.

* Setup and run a prometheus server

Then I can kinda pick, I could simply pick a port on my router and forward that straight to the prometheus instance running. But then I'd have to work with domain.name:port and I find that annoying. It might be fine for computers, but I'm human, I find it easier to remember that my prometheus runs at https://prometheus.foo.bar instead of A.B.C.D:9090 (and that's not even speaking about IPv6).
In which case I could setup dns for https://prometheus.foo.bar to point to the correct ip address and forward https traffic to the proemtheus port. But that would make life difficult if I wanna do that for more then only prometheus, as I intent too.

* Forward all http(s) traffic to the server that will host the reverse proxy
* Configure a reverse proxy with a rule that forwards/redirect to the server running prometheus.

## Setup and run prometheus (or any other service)

So quickly gonna go over this step, Prometheus is quite straightforward to get up and running, there's always the [getting started](https://prometheus.io/docs/prometheus/latest/getting_started/) on their website.
The things we need from this step is the ip and port on which Prometheus runs. The ip is the ip of the server (find this though `ip a` or the DHCP/static asignment) and the port is by default 9090 unless otherwise specified.

## Forward http(s) traffic

Pretty simple too, go to your router (or firewall) of choice and make sure to forward the http(s) traffic to the server on which you will run the reverse proxy.
Here you can make the decision to either forward http/https each on their own, or redirect all http and https to the https port on the server (it's 2018, you should be using https by now, there's no excuse anymore).

## Configure the reverse proxy

Now this differs from which service you pick as a reverse proxy Apache, Nginx, Traefik, ... . I'm gonna continue for Traefik from now on. Nginx has an [admin guide](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/) on how to setup a reverse proxy on Nginx.

Treafik is a simple to configure, feature rich reverse proxy written in Go. It gets configured through a [TOML](https://en.wikipedia.org/wiki/TOML) config file. But back to getting our Promtheus behind the reverse proxy.

So traefik works by defining frontends and backends, and then linking them to eachother. So lets define our prometheus backend.

```
[backends]
  [backends.prometheus]
    [backends.prometheus.servers]
      [backends.prometheus.servers.server0]
        url = "http://prometheus.server.ip:port"
```

Then we define the frontend and couple them together.

```
[fronteds]
  [frontends.prometheus]
    entryPoints = ["http"]
    backend = "prometheus"
    [frontends.prometheus.routes]
      [frontends.prometheus.routes.route0]
        rule = "Host:prometheus.foo.bar"
```

Setting this up, will make the prometheus server available on http://prometheus.foo.bar . We can repeat this way of working for whatever service we want too.
More config options can be found [here](https://docs.traefik.io/configuration/backends/file/). To secure the website with https, when need to change `entryPoints = ["http"]` to `entryPoints = ["http","https"]` and readjust the config according to the options mentioned here [here](https://docs.traefik.io/configuration/acme/#configuration).
