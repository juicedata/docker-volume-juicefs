package main

import (
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sirupsen/logrus"
)

const socketAddress = "/run/docker/plugins/jfs.sock"

type jfsDriver struct {
}

func (d *jfsDriver) Create(r *volume.CreateRequest) error {
	return nil
}

func (d *jfsDriver) Remove(r *volume.RemoveRequest) error {
	return nil
}

func (d *jfsDriver) List() (*volume.ListResponse, error) {
	return nil, nil
}

func (d *jfsDriver) Get(r *volume.GetRequest) (*volume.GetResponse, error) {
	return nil, nil
}

func (d *jfsDriver) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	return nil, nil
}

func (d *jfsDriver) Mount(r *volume.MountRequest) (*volume.MountResponse, error) {
	return nil, nil
}

func (d *jfsDriver) Unmount(r *volume.UnmountRequest) error {
	return nil
}

func (d *jfsDriver) Capabilities() *volume.CapabilitiesResponse {
	return nil
}

func newjfsDriver() (*jfsDriver, error) {
	return &jfsDriver{}, nil
}

func main() {
	d, err := newjfsDriver()
	if err != nil {
		logrus.Fatal(err)
	}
	h := volume.NewHandler(d)
	logrus.Error(h.ServeUnix(socketAddress, 0))
}
