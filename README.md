![Build status](https://github.com/function61/dockersockproxy/workflows/Build/badge.svg)
[![Docker pulls](https://img.shields.io/docker/pulls/fn61/dockersockproxy.svg?style=for-the-badge)](https://hub.docker.com/r/fn61/dockersockproxy/)


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


Binding only to VPN interface (e.g. Tailscale)
----------------------------------------------

[Tailscale uses CGNAT IP prefix](https://tailscale.com/kb/1015/100.x-addresses/), i.e. `100.64.0.0/10`.

If you want to only listen on that interface, you can run the container with `--addr=100.64.0.0/10:4431`
(of course you can change port if you want). We'll pick the first matching interface with matching
IP assigned from the prefix you specified.

NOTE: In this case you're likely needing to use host network namespace (and remove port mapping) with `$ docker run ...`.
