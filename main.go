package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sirupsen/logrus"
)

const socketAddress = "/run/docker/plugins/jfs.sock"

type jfsVolume struct {
	Name  string
	Token string

	AccessKey string
	SecretKey string

	Options map[string]string

	Mountpoint  string
	connections int
}

type jfsDriver struct {
	sync.RWMutex

	root      string
	statePath string
	volumes   map[string]*jfsVolume
}

func newJfsDriver(root string) (*jfsDriver, error) {
	logrus.WithField("method", "newJfsDriver").Debug(root)

	d := &jfsDriver{
		root:      filepath.Join(root, "volumes"),
		statePath: filepath.Join(root, "state", "jfs-state.json"),
		volumes:   map[string]*jfsVolume{},
	}

	if data, err := ioutil.ReadFile(d.statePath); err != nil {
		if os.IsNotExist(err) {
			logrus.WithField("statePath", d.statePath).Debug("no state found")
		} else {
			return nil, err
		}
	} else {
		if err := json.Unmarshal(data, &d.volumes); err != nil {
			return nil, err
		}
	}

	return d, nil
}

func (d *jfsDriver) saveState() {
	data, err := json.Marshal(d.volumes)
	if err != nil {
		logrus.WithField("statePath", d.statePath).Error(err)
	}

	if err := ioutil.WriteFile(d.statePath, data, 0600); err != nil {
		logrus.WithField("saveState", d.statePath).Error(err)
	}
}

func mountVolume(v *jfsVolume) error {
	fi, err := os.Lstat(v.Mountpoint)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(v.Mountpoint, 0755); err != nil {
			return logError(err.Error())
		}
	} else if err != nil {
		return logError(err.Error())
	}

	if fi != nil && !fi.IsDir() {
		return logError("%v already exist and it's not a directory", v.Mountpoint)
	}

	auth := exec.Command("juicefs", "auth", v.Name, "--token="+v.Token)
	if v.AccessKey != "" {
		auth.Args = append(auth.Args, "--accesskey="+v.AccessKey)
	}
	if v.SecretKey != "" {
		auth.Args = append(auth.Args, "--secretkey="+v.SecretKey)
	}
	// possible options for `juicefs auth`
	candidates := []string{"bucket", "bucket2", "accesskey2", "secretkey2", "passphrase"}
	for _, candidate := range candidates {
		val, ok := v.Options[candidate]
		if ok && val != "" {
			auth.Args = append(auth.Args, "--"+candidate+"="+val)
		}
	}
	logrus.Debug(auth)
	if err := auth.Run(); err != nil {
		return logError(err.Error())
	}

	mount := exec.Command("juicefs", "mount", v.Name, v.Mountpoint)
	logrus.Debug(mount)
	if err := mount.Run(); err != nil {
		return logError(err.Error())
	}

	touch := exec.Command("touch", v.Mountpoint+"/.juicefs")
	for attemp := 0; attemp < 3; attemp++ {
		if fileinfo, err := os.Lstat(v.Mountpoint); err == nil {
			stat, ok := fileinfo.Sys().(*syscall.Stat_t)
			if !ok {
				return logError("Not a syscall.Stat_t")
			}
			if stat.Ino == 1 {
				if err := touch.Run(); err == nil {
					return nil
				}
			}
		}
		logrus.Debugf("Error in attemp %d: %#v", attemp+1, err)
		time.Sleep(time.Second)
	}
	return logError(err.Error())
}

func umountVolume(v *jfsVolume) error {
	cmd := exec.Command("umount", v.Mountpoint)
	logrus.Debug(cmd)
	if err := cmd.Run(); err != nil {
		return logError(err.Error())
	}
	return nil
}

func (d *jfsDriver) Create(r *volume.CreateRequest) error {
	logrus.WithField("method", "create").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	v := &jfsVolume{
		Options: map[string]string{},
	}

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
			v.Options[key] = val
		}
	}

	if v.Name == "" {
		return logError("'name' option required")
	}

	v.Mountpoint = filepath.Join(d.root, v.Name)
	d.volumes[r.Name] = v

	err := mountVolume(v)

	if err != nil {
		return err
	}

	d.saveState()
	return nil
}

func (d *jfsDriver) Remove(r *volume.RemoveRequest) error {
	logrus.WithField("method", "remove").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	v, ok := d.volumes[r.Name]

	if !ok {
		return logError("volume %s not found", r.Name)
	}

	if v.connections != 0 {
		return logError("volume %s is in use", r.Name)
	}

	if err := umountVolume(v); err != nil {
		return err
	}

	if err := os.Remove(v.Mountpoint); err != nil {
		return logError(err.Error())
	}

	delete(d.volumes, r.Name)
	d.saveState()

	return nil
}

func (d *jfsDriver) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	logrus.WithField("method", "path").Debugf("%#v", r)

	d.RLock()
	defer d.RUnlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return &volume.PathResponse{}, logError("volume %s not found", r.Name)
	}

	return &volume.PathResponse{Mountpoint: v.Mountpoint}, nil
}

func (d *jfsDriver) Mount(r *volume.MountRequest) (*volume.MountResponse, error) {
	logrus.WithField("method", "mount").Debugf("%#v", r)

	v, ok := d.volumes[r.Name]
	if !ok {
		return &volume.MountResponse{}, logError("volume %s not found", r.Name)
	}
	v.connections++
	return &volume.MountResponse{Mountpoint: v.Mountpoint}, nil
}

func (d *jfsDriver) Unmount(r *volume.UnmountRequest) error {
	logrus.WithField("method", "umount").Debugf("%#v", r)

	v, ok := d.volumes[r.Name]
	if !ok {
		return logError("volume %s not found", r.Name)
	}

	v.connections--
	return nil
}

func (d *jfsDriver) Get(r *volume.GetRequest) (*volume.GetResponse, error) {
	logrus.WithField("method", "get").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return &volume.GetResponse{}, logError("volume %s not found", r.Name)
	}

	return &volume.GetResponse{Volume: &volume.Volume{Name: r.Name, Mountpoint: v.Mountpoint}}, nil
}

func (d *jfsDriver) List() (*volume.ListResponse, error) {
	logrus.WithField("method", "list").Debugf("")

	d.Lock()
	defer d.Unlock()

	var vols []*volume.Volume
	for name, v := range d.volumes {
		vols = append(vols, &volume.Volume{Name: name, Mountpoint: v.Mountpoint})
	}
	return &volume.ListResponse{Volumes: vols}, nil
}

func (d *jfsDriver) Capabilities() *volume.CapabilitiesResponse {
	logrus.WithField("method", "capabilities").Debugf("")

	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "local"}}
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
