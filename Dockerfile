FROM scratch

ENTRYPOINT ["/usr/bin/dockersockproxy"]

ADD rel/dockersockproxy_linux-amd64 /usr/bin/dockersockproxy
