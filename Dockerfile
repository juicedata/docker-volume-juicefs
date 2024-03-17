FROM golang:1.18 as builder

ARG GOPROXY
ENV GOPROXY=${GOPROXY:-"https://proxy.golang.org,direct"}
ARG JUICEFS_CE_VERSION
ENV JUICEFS_CE_VERSION=${JUICEFS_CE_VERSION:-"1.0.4"}
ARG ARCH
ENV ARCH=${ARCH:-"amd64"}

WORKDIR /docker-volume-juicefs
COPY . .
RUN apt-get update && apt-get install -y curl musl-tools tar gzip upx-ucl && \
    CC=/usr/bin/musl-gcc go build -o bin/docker-volume-juicefs --ldflags '-linkmode external -extldflags "-static"' .

WORKDIR /workspace
RUN curl -fsSL -o juicefs-ce.tar.gz https://github.com/juicedata/juicefs/releases/download/v${JUICEFS_CE_VERSION}/juicefs-${JUICEFS_CE_VERSION}-linux-${ARCH}.tar.gz && \
    tar -zxf juicefs-ce.tar.gz -C /tmp && \
    curl -fsSL -o /juicefs https://s.juicefs.com/static/juicefs && \
    chmod +x /juicefs

FROM python:2.7
RUN mkdir -p /run/docker/plugins /jfs/state /jfs/volumes
COPY --from=builder /docker-volume-juicefs/bin/docker-volume-juicefs /
COPY --from=builder /tmp/juicefs /bin/
COPY --from=builder /juicefs /usr/bin/
RUN /usr/bin/juicefs version && /bin/juicefs --version
CMD ["docker-volume-juicefs"]
