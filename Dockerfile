FROM golang:1.17 as builder

ARG GOPROXY
ARG JUICEFS_CE_VERSION

WORKDIR /docker-volume-juicefs
COPY . .
ENV GOPROXY=${GOPROXY:-"https://proxy.golang.org,direct"}
RUN apt-get update && apt-get install -y curl musl-tools tar gzip upx-ucl \
    && CC=/usr/bin/musl-gcc go build -o bin/docker-volume-juicefs \
       --ldflags '-linkmode external -extldflags "-static"' .

WORKDIR /workspace
ENV JUICEFS_CE_VERSION=${JUICEFS_CE_VERSION:-"main"}
RUN curl -fsSL -o juicefs-${JUICEFS_CE_VERSION}.tar.gz \
       https://github.com/juicedata/juicefs/archive/${JUICEFS_CE_VERSION}.tar.gz \
    && mkdir juicefs \
    && tar -xf juicefs-${JUICEFS_CE_VERSION}.tar.gz --strip-components=1 -C juicefs \
    && cd juicefs && STATIC=1 make && upx juicefs

RUN curl -fsSL -o /juicefs https://s.juicefs.com/static/juicefs \
    && chmod +x /juicefs

FROM jfloff/alpine-python:2.7-slim
RUN mkdir -p /run/docker/plugins /jfs/state /jfs/volumes
COPY --from=builder /docker-volume-juicefs/bin/docker-volume-juicefs /
COPY --from=builder /workspace/juicefs/juicefs /bin/
COPY --from=builder /juicefs /usr/bin/
RUN /usr/bin/juicefs version && /bin/juicefs --version
CMD ["docker-volume-juicefs"]
