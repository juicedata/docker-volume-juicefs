#!/bin/bash

docker plugin install juicedata/juicefs --grant-all-permissions

docker volume create -d juicedata/juicefs -o name=$JFS_VOL -o token=$JFS_TOKEN -o accesskey=$ACCESS_KEY -o secretkey=$SECRET_KEY jfsvolume
docker run --rm -v jfsvolume:/write busybox sh -c "echo hello > /write/world"
docker run --rm -v jfsvolume:/read busybox grep -Fxq hello /read/world
docker run --rm -v jfsvolume:/list busybox ls /list
docker volume rm jfsvolume
