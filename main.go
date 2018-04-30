package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sirupsen/logrus"
)

const socketAddress = "/run/docker/plugins/jfs.sock"

type jfsVolume struct {
	Name  string
	Token string

	AccessKey string
	SecretKey string

	Options []string

	Mountpoint  string
	connections int
}

type jfsDriver struct {
	sync.RWMutex

	root    string
	volumes map[string]*jfsVolume
}

func newJfsDriver(root string) (*jfsDriver, error) {
	return &jfsDriver{root: root}, nil
}

func (d *jfsDriver) Create(r *volume.CreateRequest) error {
	logrus.WithField("method", "create").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	v := &jfsVolume{}

	for key, val := range r.Options {
		switch key {
		case "name":
			v.Name = val
		case "token":
			v.Token = val
		case "accesskey":
			v.AccessKey = val
		case "secretkey":
			v.SecretKey = val
		default:
			if val != "" {
				v.Options = append(v.Options, key+"="+val)
			} else {
				v.Options = append(v.Options, key)
			}
		}
	}

	if v.Name == "" {
		return logError("'name' option required")
	}

	v.Mountpoint = filepath.Join(d.root, v.Name)
	d.volumes[r.Name] = v

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

func logError(format string, args ...interface{}) error {
	logrus.Errorf(format, args...)
	return fmt.Errorf(format, args...)
}

func main() {
	debug := os.Getenv("DEBUG")
	if ok, _ := strconv.ParseBool(debug); ok {
		logrus.SetLevel(logrus.DebugLevel)
	}

	d, err := newJfsDriver("/jfs")
	if err != nil {
		logrus.Fatal(err)
	}
	h := volume.NewHandler(d)
	logrus.Infof("listening on %s", socketAddress)
	logrus.Error(h.ServeUnix(socketAddress, 0))
}
