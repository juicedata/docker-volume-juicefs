# Docker Volume Plugin for Juicefs

[![Build Status](https://travis-ci.com/juicedata/docker-volume-juicefs.svg?token=ACsZ5AkewTgk5D5wzzds&branch=master)](https://travis-ci.com/juicedata/docker-volume-juicefs)

Modified from https://github.com/vieux/docker-volume-sshfs

## Usage

``` shell
docker plugin install juicedata/juicefs

# JuiceFS Community Edition
docker volume create -d juicedata/juicefs:latest -o name=$JFS_VOL -o metaurl=$JFS_META_URL jfsvolume
docker run -it -v jfsvolume:/opt busybox ls /opt

# JuiceFS Enterprise Edition
docker volume create -d juicedata/juicefs:latest -o name=$JFS_VOL -o token=$JFS_TOKEN -o access-key=$JFS_ACCESSKEY -o secret-key=$JFS_SECRETKEY jfsvolume
docker run -it -v jfsvolume:/opt busybox ls /opt
```

## Development

Boot up vagrant environment

``` shell
vagrant up
vagrant ssh
```

Inside vagrant

``` shell
export WORKDIR=~/go/src/docker-volume-juicefs
mkdir -p $WORKDIR
rsync -avz --exclude plugin --exclude .git --exclude .vagrant /vagrant/ $WORKDIR/
cd $WORKDIR
make
make enable
docker volume create -d juicedata/juicefs:next -o name=$JFS_VOL -o token=$JFS_TOKEN -o access-key=$JFS_ACCESSKEY -o secret-key=$JFS_SECRETKEY jfsvolume
docker run -it -v jfsvolume:/opt busybox ls /opt
```

### Docker swarm

Install juicedata/juicefs plugin on **every** worker node, otherwise service mounting JuiceFS volume will not be scheduled.

Use `docker service` to deploy to Docker swarm

``` shell
docker service create --name nginx --mount \
type=volume,volume-driver=juicedata/juicefs,source=jfsvolume,destination=/jfs,\
volume-opt=name=$JFS_VOL,volume-opt=token=$JFS_TOKEN,volume-opt=access-key=$JFS_ACCESSKEY,volume-opt=secret-key=$JFS_SECRETKEY nginx:alpine
```

Scale up

``` shell
docker service scale nginx=3
```

Deployment from docker compose file is not supported because there is no way to pass volume options.

## Debug

Enable debug information

``` shell
docker plugin disable juicedata/juicefs:latest
docker plugin set juicedata/juicefs:latest DEBUG=1
docker plugin enable juicedata/juicefs:latest
```

To quickly test out HEAD version:

``` shell
docker plugin disable juicedata/juicefs:latest
CC=/usr/bin/musl-gcc go build -o bin/docker-volume-juicefs --ldflags '-linkmode external -extldflags "-static"' .
mv bin/docker-volume-juicefs /var/lib/docker/plugins/3dea603741f58726d65b273d095f2bc01d1a1c8954a5498f5592041df8cdcd6c/rootfs
docker plugin enable juicedata/juicefs:latest
```

The stdout of the plugin is redirected to dockerd log. The entries have a `plugin=<ID>` suffix.

`runc`, the default docker container runtime can be used to collect juicefs log

``` shell
# runc --root /run/docker/plugins/runtime-root/plugins.moby list
ID                                                                 PID         STATUS      BUNDLE
452d2c0cf3fd45e73a93a2f2b00d03ed28dd2bc0c58669cca9d4039e8866f99f   3672        running     /run/docker/containerd/...

# runc --root /run/docker/plugins/runtime-root/plugins.moby exec 452d2c0cf3fd45e73a93a2f2b00d03ed28dd2bc0c58669cca9d4039e8866f99f cat /var/log/juicefs.log
umount: can't unmount /jfs/volumes/ci-aliyun: Invalid argument
Unable to connect to local syslog daemon
2018/05/07 13:56:19.752864 <INFO>: Cache dir: /var/jfsCache/ci-aliyun limit: 1024 MB
2018/05/07 13:56:19.756331 <INFO>: Found 0 cached blocks (0 bytes)
2018/05/07 13:56:20.913240 <INFO>: mount successfully, st_dev: 48
```

NOTE: the directory for plugin runtime could be `moby-plugins` in some version of Docker.
