PLUGIN_NAME = juicedata/juicefs
PLUGIN_TAG ?= latest

all: clean rootfs create

clean:
	@echo "### rm ./plugin"
	@rm -rf ./plugin

rootfs:
	@echo "### docker build: rootfs image with docker-volume-juicefs"
	@docker build -t ${PLUGIN_NAME}:rootfs .
	@echo "### create rootfs directory in ./plugin/rootfs"
	@mkdir -p ./plugin/rootfs
	@docker create --name tmp ${PLUGIN_NAME}:rootfs
	@docker export tmp | tar -x -C ./plugin/rootfs
	@echo "### copy config.json to ./plugin/"
	@cp config.json ./plugin/
	@docker rm -vf tmp

create:
	@echo "### remove existing plugin ${PLUGIN_NAME}:${PLUGIN_TAG} if exists"
	@docker plugin rm -f ${PLUGIN_NAME}:${PLUGIN_TAG} || true
	@echo "### create new plugin ${PLUGIN_NAME}:${PLUGIN_TAG} from ./plugin"
	@docker plugin create ${PLUGIN_NAME}:${PLUGIN_TAG} ./plugin

enable:		
	@echo "### enable plugin ${PLUGIN_NAME}:${PLUGIN_TAG}"		
	docker plugin enable ${PLUGIN_NAME}:${PLUGIN_TAG}

test: enable volume compose

volume:
	@echo "### test volume create and mount"
	docker volume create -d ${PLUGIN_NAME}:${PLUGIN_TAG} -o name=${JFS_VOL} -o token=${JFS_TOKEN} -o accesskey=${JFS_ACCESSKEY} -o secretkey=${JFS_SECRETKEY} jfsvolume

	docker run --rm -v jfsvolume:/write busybox sh -c "echo hello > /write/world"
	docker run --rm -v jfsvolume:/read busybox sh -c "grep -Fxq hello /read/world"
	docker run --rm -v jfsvolume:/list busybox sh -c "ls /list"

	docker volume rm jfsvolume

compose:
	@echo "### test compose"
	docker-compose -f docker-compose.yml up
	docker-compose -f docker-compose.yml down --volume

push:
	@echo "### push plugin ${PLUGIN_NAME}:${PLUGIN_TAG}"
	docker login --username ${DOCKER_USERNAME} --password ${DOCKER_PASSWORD}
	docker plugin push ${PLUGIN_NAME}:${PLUGIN_TAG}
