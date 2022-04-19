FROM golang:1.17 as builder

ARG GOPROXY

WORKDIR /docker-volume-juicefs
COPY . .
ENV GOPROXY=${GOPROXY:-"https://proxy.golang.org,direct"}
RUN apt-get update && apt-get install -y curl musl-tools upx-ucl && \
    CC=/usr/bin/musl-gcc go build -o bin/docker-volume-juicefs --ldflags '-linkmode external -extldflags "-static"' .

WORKDIR /workspace
RUN git clone --depth=1 https://github.com/juicedata/juicefs && \
    cd juicefs && STATIC=1 make && upx juicefs

RUN curl -fsSL -o /juicefs https://s.juicefs.com/static/juicefs \
    && chmod +x /juicefs

FROM jfloff/alpine-python:2.7-slim
RUN mkdir -p /run/docker/plugins /jfs/state /jfs/volumes
COPY --from=builder /docker-volume-juicefs/bin/docker-volume-juicefs /
COPY --from=builder /workspace/juicefs/juicefs /bin/
COPY --from=builder /juicefs /usr/bin/
RUN /usr/bin/juicefs version && /bin/juicefs --version
CMD ["docker-volume-juicefs"]
