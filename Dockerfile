FROM golang:1.14-alpine as builder
COPY . /go/src/github.com/juicedata/docker-volume-juicefs
WORKDIR /go/src/github.com/juicedata/docker-volume-juicefs
RUN set -ex \
    && apk add --no-cache --virtual .build-deps \
    gcc libc-dev wget\
    && go install --ldflags '-extldflags "-static"' \
    && apk del .build-deps
WORKDIR /
RUN wget -q juicefs.io/static/juicefs \
    && chmod +x juicefs
CMD ["/go/bin/docker-volume-juicefs"]

FROM jfloff/alpine-python:2.7-slim
RUN mkdir -p /run/docker/plugins /jfs/state /jfs/volumes
COPY --from=builder /go/bin/docker-volume-juicefs .
COPY --from=builder /juicefs /usr/local/bin/
RUN juicefs version
# Workaround for case insensitive file system on macOS
RUN rm -rf /usr/share/terminfo
CMD ["docker-volume-juicefs"]
