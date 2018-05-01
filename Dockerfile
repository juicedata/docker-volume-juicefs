FROM golang:1.9-alpine as builder
COPY . /go/src/github.com/juicedata/docker-volume-juicefs
WORKDIR /go/src/github.com/juicedata/docker-volume-juicefs
RUN set -ex \
    && apk add --no-cache --virtual .build-deps \
    gcc libc-dev \
    && go install --ldflags '-extldflags "-static"' \
    && apk del .build-deps
CMD ["/go/bin/docker-volume-juicefs"]

FROM alpine
RUN mkdir -p /run/docker/plugins /jfs/state /jfs/volumes
COPY --from=builder /go/bin/docker-volume-juicefs .
CMD ["docker-volume-juicefs"]
