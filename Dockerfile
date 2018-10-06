FROM scratch

CMD ["dockersockproxy"]

ADD rel/dockersockproxy_linux-amd64 /usr/local/bin/dockersockproxy
