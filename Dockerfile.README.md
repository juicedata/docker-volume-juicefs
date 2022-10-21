# Quick reference
- **Maintained by**:
    [the JuiceFS Community](https://github.com/juicedata/juicefs)
- **Repository**:
    [juicedata/docker-volume-juicefs](https://github.com/juicedata/docker-volume-juicefs)
- **Where to get help**:
   [Documents](https://www.juicefs.com/docs/community/juicefs_on_docker#docker-volume-plugin), [the JuiceFS Community Slack](https://join.slack.com/t/juicefs/shared_invite/zt-n9h5qdxh-YD7e0JxWdesSEa9vY_f_DA), [Server Fault](https://serverfault.com/help/on-topic), [Unix & Linux](https://unix.stackexchange.com/help/on-topic), or [Stack Overflow](https://stackoverflow.com/help/on-topic)

# JuiceFS Volume Plugin for Docker
This is the JuiceFS docker volume plugin image for both the open source edition and the cloud service, which makes it easy to use JuiceFS as a high performance, massive space persistent storage for Docker containers.

## Notice
This is a Docker Volume Plugin image, if you need the standard image with JuiceFS clients packaged, please use the [juicedata/mount](https://hub.docker.com/r/juicedata/mount).

# How to use
## Denpendency
Since JuiceFS mount depends on FUSE, please make sure that the FUSE driver is already installed on the host, in the case of Debian/Ubuntu.
```shell
sudo apt-get -y install fuse
```

## Installation
```shell
$ sudo docker plugin install juicedata/juicefs --alias juicefs
```

## Create a volume
```shell
sudo docker volume create -d juicefs \  
-o name=<VOLUME_NAME> \  
-o metaurl=<META_URL> \  
-o storage=<STORAGE_TYPE> \  
-o bucket=<BUCKET_NAME> \  
-o access-key=<ACCESS_KEY> \  
-o secret-key=<SECRET_KEY> \  
jfsvolume
```

Please refer to the following instructions to modify the command:
- `<VOLUME_NAME>` - Name of the JuiceFS filesystem
- `<META_URL>` - Database address for metadata storage, please refer to [How to Set Up Metadata Engine](https://www.juicefs.com/docs/community/databases_for_metadata/)
- `<STORAGE_TYPE>` - Storage type, please refer to [How to Set Up Object Storage](https://www.juicefs.com/docs/community/how_to_setup_object_storage)
- `<BUCKET_NAME>` - Data storage address, it usually the bucket endpoint address of the object storage.
- `<ACCESS_KEY>`  and `<SECRET_KEY>`  - Keys used to access the object storage.

## Bind the volume to a container
```shell
sudo docker run -it -v jfsvolume:/mnt busybox ls /mnt
```

## Upgrade
```shell
sudo docker plugin upgrade juicefs  
sudo docker plugin enable juicefs
```

## Uninstall
```shell
sudo docker plugin disable juicefs
sudo docker plugin rm juicefs
```

# Troubleshooting
## Storage volumes are not used but cannot be deleted
This may occur because the parameters set when creating the storage volume are incorrect. It is recommended to check the type of object storage, bucket name, Access Key, Secret Key, database address and other information. You can try disabling and re-enabling the juicefs volume plugin to release the failed volume, and then recreate the storage volume with the correct parameter information.

## Log of the collection volume plugin
To troubleshoot, you can open a new terminal window and execute the following command while performing the operation to view the live log information.

```
journalctl -f -u docker | grep "plugin="
```

To learn more about the JuiceFS volume plugin, you can visit the [juicedata/docker-volume-juicefs](https://github.com/juicedata/docker-volume-juicefs) code repository.