#!/bin/bash

set -o errexit

SCRIPT_DIR="$(cd $(dirname $0)/; pwd)"
TEST_BASEDIR="${SCRIPT_DIR}/tmp"
ROOTFS_DIR="$(cd ${SCRIPT_DIR}/../plugin/rootfs; pwd)"
IP_ADDR=$(ip route show default | grep -oE 'src [0-9\.]+' | awk '{print $2}')
echo "SCRIPT_DIR: ${SCRIPT_DIR}"
echo "TEST_BASEDIR: ${TEST_BASEDIR}"
echo "ROOTFS_DIR: ${ROOTFS_DIR}"
echo "IP_ADDR: ${IP_ADDR}"

MINIO_ROOT_USER=minio-root-user
MINIO_ROOT_PASSWORD=minio-root-password

prepare() {
	mkdir -p "$TEST_BASEDIR"
	cat > "${TEST_BASEDIR}/docker-compose.yml" <<__EOF__
version: '3'
services:
  redis:
    image: redis:7.4.0-bookworm
    command: redis-server
    volumes:
      - "${TEST_BASEDIR}/redis-data:/data"
    ports:
      - "16777:6379"
  minio:
    image: bitnami/minio:2024.8.29-debian-12-r1
    volumes:
      - "${TEST_BASEDIR}/minio-data:/bitnami/minio/data"
    ports:
      - "19000:9000"
      - "19001:9001"
    environment:
      - MINIO_ROOT_USER=${MINIO_ROOT_USER}
      - MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}
__EOF__
	[ -x "${TEST_BASEDIR}/juicefs" ] || cp -af "${ROOTFS_DIR}/bin/juicefs" "${TEST_BASEDIR}/juicefs"
	cd "${TEST_BASEDIR}"
	mkdir -p minio-data && chown -R 1001:1001 minio-data
	docker-compose up -d
	./juicefs format \
		--storage=s3 \
		--bucket=http://myjfs.${IP_ADDR}:19000 \
		--access-key=${MINIO_ROOT_USER} \
		--secret-key=${MINIO_ROOT_PASSWORD} \
		redis://${IP_ADDR}:16777 myjfs
	./juicefs mount -d redis://${IP_ADDR}:16777 "${TEST_BASEDIR}/myjfs"
	echo "$(date -Iseconds) Hello world!" >> "${TEST_BASEDIR}/myjfs/hello.txt"
	cd -
}

clean() {
	cd "${TEST_BASEDIR}"
	if [ -d "./myjfs" ]; then
		local inode=$(stat -c %i ./myjfs)
		if [ "x$inode" == "x1" ]; then
			echo "Umount myjfs ..."
			umount ./myjfs
		fi
	fi
	docker-compose down
	cd -
	rm -rf "${TEST_BASEDIR}"
}

test_plugin() {
	local volname=myjfsvolume
	local nvol=$(docker volume ls | grep $volname | wc -l)
	if [ "x$nvol" == x1 ]; then
		echo "Remove docker volume: $volname"
		docker volume rm $volname
	fi
	echo "Create docker volume: $volname"
	docker volume create -d juicedata/juicefs:${PLUGIN_TAG:latest} -o name=myjfs -o metaurl=redis://${IP_ADDR}:16777 $volname
	docker run --rm -it -v myjfsvolume:/opt busybox cat /opt/hello.txt

	nvol=$(docker volume ls | grep $volname | wc -l)
	if [ "x$nvol" == x1 ]; then
		echo "Remove docker volume: $volname"
		docker volume rm $volname
	fi
}

usage() {
	echo "Usage: $1 {prepare|test|clean}"
	exit 1
}


if [ $# -lt 1 ]; then
	usage "$0"
fi

SUBCMD=$1

case $SUBCMD in
	prepare)
		prepare
		;;
	clean)
		clean
		;;
	test)
		test_plugin
		;;
	*)
		usage
		;;
esac
