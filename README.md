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

Deploy this container as a Swarm service (a bare container will also suffice), with ENV var: `SERVERCERT_KEY`.
The content is base64 encoded version of `server.key` file.

You probably need this as a Swarm service, if you have a multi-node cluster, because some
apps need to connect to manager nodes (see placement contstraint).

```
$ SERVERCERT_KEY="..."
$ DOCKERSOCKPROXY_VERSION="..."
$ docker service create \
	--name dockersockproxy \
	--constraint node.role==manager \
	--publish 4431:4431 \
	--env "SERVERCERT_KEY=$SERVERCERT_KEY" \
	--mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
	--network fn61 \
	"fn61/dockersockproxy:$DOCKERSOCKPROXY_VERSION"
```
