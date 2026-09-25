FROM alpine:3.23

RUN apk add --no-cache \
    bash \
    curl \
    coreutils

COPY ./scripts/keepalive-request.sh /usr/local/bin/keepalive

RUN chmod +x /usr/local/bin/keepalive

ENTRYPOINT ["/usr/local/bin/keepalive"]
