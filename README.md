# Docker Volume Plugin for Juicefs

Modified from https://github.com/vieux/docker-volume-sshfs

## Usage

``` shell
$ docker plugin install juicedata/juicefs
Plugin "juicedata/juicefs" is requesting the following privileges:
 - network: [host]
 - device: [/dev/fuse]
 - capabilities: [CAP_SYS_ADMIN]
Do you grant the above permissions? [y/N]

$ docker volume create -d juicedata/juicefs:next -o name=$JFS_VOL -o token=$JFS_TOKEN -o accesskey=$ACCESS_KEY -o secretkey=$SECRET_KEY jfsvolume
$ docker run -it -v jfsvolume:/opt busybox ls /opt
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
docker volume create -d juicedata/juicefs:next -o name=$JFS_VOL -o token=$JFS_TOKEN -o accesskey=$ACCESS_KEY -o secretkey=$SECRET_KEY jfsvolume
docker run -it -v jfsvolume:/opt busybox ls /opt
```

## Debug

Enable debug information

``` shell
docker plugin set juicedata/juicefs:next DEBUG=1
```

The stdout of the plugin is redirected to dockerd log. The entries have a `plugin=<ID>` suffix.

`docker-runc`, the default docker container runtime can be used to collect juicefs log

``` shell
# docker-runc --root /var/run/docker/plugins/runtime-root/plugins.moby list
ID                                                                 PID         STATUS      BUNDLE
452d2c0cf3fd45e73a93a2f2b00d03ed28dd2bc0c58669cca9d4039e8866f99f   3672        running     /run/docker/containerd/...

# docker-runc --root /var/run/docker/plugins/runtime-root/plugins.moby exec 452d2c0cf3fd45e73a93a2f2b00d03ed28dd2bc0c58669cca9d4039e8866f99f cat /var/log/juicefs.log
umount: can't unmount /jfs/volumes/ci-aliyun: Invalid argument
Unable to connect to local syslog daemon
2018/05/07 13:56:19.752864 <INFO>: Cache dir: /var/jfsCache/ci-aliyun limit: 1024 MB
2018/05/07 13:56:19.756331 <INFO>: Found 0 cached blocks (0 bytes)
2018/05/07 13:56:20.913240 <INFO>: mount successfully, st_dev: 48
```

NOTE: the directory for plugin runtime could be `moby-plugins` in some version of Docker.
