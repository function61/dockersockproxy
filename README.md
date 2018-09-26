[![Build Status](https://travis-ci.org/function61/dockersockproxy.svg?branch=master)](https://travis-ci.org/function61/dockersockproxy)
[![Download](https://img.shields.io/docker/pulls/fn61/dockersockproxy.svg)](https://hub.docker.com/r/fn61/dockersockproxy/)

What?
-----

Proxies Docker's socket over TLS.

Docker daemon has essentially this same already implemented,
but I cannot be bothered to configure that for CoreOS and dev VMs.

With this I can publish this proxy as a container and have total control over the TLS parameters.


Usage
-----

Deploy this container as a Swarm service (a bare container also will suffice), with ENV var: `SERVERCERT_KEY`. The content is base64 encoded version of `server.key` file.

```
$ docker run -d --name dockersockproxy -v /var/run/docker.sock:/var/run/docker.sock -p 4431:4431 -e SERVERCERT_KEY=... fn61/dockersockproxy:20180926_1544_97ccec80
```
