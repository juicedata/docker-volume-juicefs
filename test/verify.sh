#!/bin/bash

docker volume create -d juicedata/juicefs -o name=$JFS_VOL -o token=$JFS_TOKEN -o accesskey=$ACCESS_KEY -o secretkey=$SECRET_KEY jfsvolume

docker run --rm -v jfsvolume:/write busybox sh -c "echo hello > /write/world"
docker run --rm -v jfsvolume:/read busybox sh -c "grep -Fxq hello /read/world"
docker run --rm -v jfsvolume:/list busybox sh -c "ls /list"

docker volume rm jfsvolume

docker-compose -f test/docker-compose.yml up
docker-compose -f test/docker-compose.yml down --volume

