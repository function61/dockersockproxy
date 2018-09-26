What?
-----

Proxies Docker's socket over TLS.

Docker daemon has essentially this same already implemented,
but I cannot be bothered to configure that for CoreOS and dev VMs.

With this I can publish this proxy as a container and have total control over the TLS parameters.


Usage
-----

Deploy this container as a Swarm service (a bare container also will suffice), with one ENV var: `SERVERCERT_KEY`.

The content is base64 encoded version of `server.key` file.
