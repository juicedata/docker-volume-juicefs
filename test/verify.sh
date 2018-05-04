#!/bin/bash

docker volume create -d juicedata/juicefs -o name=$JFS_VOL -o token=$JFS_TOKEN -o accesskey=$ACCESS_KEY -o secretkey=$SECRET_KEY jfsvolume

docker run --rm -v jfsvolume:/write busybox sh -c "sleep 3 && echo hello > /write/world"
docker run --rm -v jfsvolume:/read busybox sh -c "sleep 3 && grep -Fxq hello /read/world"
docker run --rm -v jfsvolume:/list busybox sh -c "sleep 3 && ls /list"

sleep 3

docker volume rm jfsvolume
