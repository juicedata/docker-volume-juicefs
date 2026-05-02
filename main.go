package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sirupsen/logrus"
)

const (
	socketAddress = "/run/docker/plugins/jfs.sock"
	cliPath       = "/usr/bin/juicefs"
	ceCliPath     = "/bin/juicefs"
)

type jfsVolume struct {
	Name           string
	Options        map[string]string
	Source         string
	Mountpoint     string
	connections    int
	mountHealthCheck chan struct{}
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

func safeClose(ch chan struct{}) {
	defer func() { recover() }()
	if ch != nil {
		close(ch)
	}
}

func isActiveMount(mountpoint string) (bool, error) {
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		sepIdx := -1
		for i, f := range fields {
			if f == "-" {
				sepIdx = i
				break
			}
		}
		if sepIdx == -1 || len(fields) < sepIdx+3 {
			continue
		}
		mountPath := fields[4]
		unescaped := strings.ReplaceAll(mountPath, "\\040", " ")
		if unescaped == mountpoint {
			return true, nil
		}
	}
	return false, nil
}

func isFuseMount(mountpoint string) bool {
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		sepIdx := -1
		for i, f := range fields {
			if f == "-" {
				sepIdx = i
				break
			}
		}
		if sepIdx == -1 || len(fields) < sepIdx+3 {
			continue
		}
		mountPath := fields[4]
		unescaped := strings.ReplaceAll(mountPath, "\\040", " ")
		if unescaped == mountpoint {
			fsType := fields[sepIdx+1]
			if strings.Contains(fsType, "fuse") {
				return true
			}
		}
	}
	return false
}

func ceMount(v *jfsVolume) error {
	options := map[string]string{}
	format := exec.Command(ceCliPath, "format", "--no-update")
	for k, v := range v.Options {
		if k == "env" {
			format.Env = append(os.Environ(), strings.Split(v, ",")...)
			logrus.Debugf("modified env: %s", format.Env)
			continue
		}
		options[k] = v
	}
	// Raw formatFlags passthrough - passed directly to juicefs format
	// Example: -o formatFlags="--compress lz4 --trash-days 0"
	if rawFlags, ok := options["formatFlags"]; ok {
		format.Args = append(format.Args, strings.Fields(rawFlags)...)
		delete(options, "formatFlags")
	}

	formatOptions := []string{
		"block-size",
		"compress",
		"shards",
		"storage",
		"bucket",
		"access-key",
		"secret-key",
		"encrypt-rsa-key",
		"trash-days",
	}
	for _, formatOption := range formatOptions {
		val, ok := options[formatOption]
		if !ok {
			continue
		}
		format.Args = append(format.Args, fmt.Sprintf("--%s=%s", formatOption, val))
		delete(options, formatOption)
	}
	format.Args = append(format.Args, v.Source, v.Name)
	logrus.Debug(format)
	if out, err := format.CombinedOutput(); err != nil {
		logrus.Errorf("juicefs format error: %s", out)
		return logError(err.Error())
	}

	// options left for `juicefs mount`
	mount := exec.Command(ceCliPath, "mount")

	// Raw mountFlags passthrough - passed directly to juicefs mount
	// Example: -o mountFlags="--all-squash 1000:1000 --cache-size 10G"
	if rawFlags, ok := options["mountFlags"]; ok {
		mount.Args = append(mount.Args, strings.Fields(rawFlags)...)
		delete(options, "mountFlags")
	}

	// Boolean flags that need --flag format (not --flag=value)
	mountFlags := []string{
		"cache-partial-only",
		"enable-xattr",
		"no-syslog",
		"no-usage-report",
		"writeback",
	}
	for _, mountFlag := range mountFlags {
		_, ok := options[mountFlag]
		if !ok {
			continue
		}
		mount.Args = append(mount.Args, fmt.Sprintf("--%s", mountFlag))
		delete(options, mountFlag)
	}
	for mountOption, val := range options {
		mount.Args = append(mount.Args, fmt.Sprintf("--%s=%s", mountOption, val))
	}
	mount.Args = append(mount.Args, v.Source, v.Mountpoint)
	logrus.Debug(mount)
	go func() {
		output, _ := mount.CombinedOutput()
		logrus.Debug(string(output))
	}()

	touch := exec.Command("touch", v.Mountpoint+"/.juicefs")
	var fileinfo os.FileInfo
	var err error
	mounted := false
	for attempt := 0; attempt < 10; attempt++ {
		if fileinfo, err = os.Lstat(v.Mountpoint); err == nil {
			stat, ok := fileinfo.Sys().(*syscall.Stat_t)
			if !ok {
				return logError("Not a syscall.Stat_t")
			}
			if stat.Ino == 1 {
				if err = touch.Run(); err == nil {
					mounted = true
					break
				}
			}
		}
		logrus.Debugf("Error in attempt %d: %#v", attempt+1, err)
		time.Sleep(time.Second)
	}
	if err != nil || !mounted {
		if err == nil {
			err = fmt.Errorf("failed to verify mount after 10 attempts")
		}
		return logError(err.Error())
	}

	safeClose(v.mountHealthCheck)
	v.mountHealthCheck = make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if !isFuseMount(v.Mountpoint) {
					logrus.Errorf("FUSE mount at %s is no longer active", v.Mountpoint)
				}
			case <-v.mountHealthCheck:
				return
			}
		}
	}()
	return nil
}

func eeMount(v *jfsVolume) error {
	auth := exec.Command(cliPath, "auth", v.Name)
	options := map[string]string{}
	for k, v := range v.Options {
		if k == "env" {
			auth.Env = append(os.Environ(), strings.Split(v, ",")...)
			logrus.Debugf("modified env: %s", auth.Env)
			continue
		}
		options[k] = v
	}
	// Raw authFlags passthrough - passed directly to juicefs auth
	// Example: -o authFlags="--token xxx --access-key yyy"
	if rawFlags, ok := options["authFlags"]; ok {
		auth.Args = append(auth.Args, strings.Fields(rawFlags)...)
		delete(options, "authFlags")
	}

	commonOptions := []string{"subdir"}
	authOptions := slices.Concat([]string{
		"token",
		"accesskey",
		"accesskey2",
		"access-key",
		"access-key2",
		"bucket",
		"bucket2",
		"secretkey",
		"secretkey2",
		"secret-key",
		"secret-key2",
		"passphrase",
	}, commonOptions)
	for _, authOption := range authOptions {
		val, ok := options[authOption]
		if !ok {
			continue
		}
		// auth 的参数确实可以是空, 没有flag
		auth.Args = append(auth.Args, fmt.Sprintf("--%s=%s", authOption, val))
		if !slices.Contains(commonOptions, authOption) {
			delete(options, authOption)
		}
	}
	logrus.Debug(auth)
	if out, err := auth.CombinedOutput(); err != nil {
		logrus.Errorf("juicefs auth error: %s", out)
		return logError(err.Error())
	}

	// options left for `juicefs mount`
	mount := exec.Command(cliPath, "mount", v.Name, v.Mountpoint)

	// Raw mountFlags passthrough - passed directly to juicefs mount
	// Example: -o mountFlags="--all-squash 1000:1000 --cache-size 10G"
	if rawFlags, ok := options["mountFlags"]; ok {
		mount.Args = append(mount.Args, strings.Fields(rawFlags)...)
		delete(options, "mountFlags")
	}

	// Boolean flags that need --flag format (not --flag=value)
	mountFlags := []string{
		"external",
		"internal",
		"gc",
		"dry",
		"flip",
		"no-sync",
		"allow-other",
		"allow-root",
		"enable-xattr",
	}
	for _, mountFlag := range mountFlags {
		_, ok := options[mountFlag]
		if !ok {
			continue
		}
		mount.Args = append(mount.Args, fmt.Sprintf("--%s", mountFlag))
		delete(options, mountFlag)
	}
	for mountOption, val := range options {
		mount.Args = append(mount.Args, fmt.Sprintf("--%s=%s", mountOption, val))
	}
	logrus.Debug(mount)
	if out, err := mount.CombinedOutput(); err != nil {
		logrus.Errorf("juicefs mount error: %s", out)
		return logError(err.Error())
	}

	touch := exec.Command("touch", v.Mountpoint+"/.juicefs")
	var fileinfo os.FileInfo
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if fileinfo, err = os.Lstat(v.Mountpoint); err == nil {
			stat, ok := fileinfo.Sys().(*syscall.Stat_t)
			if !ok {
				return logError("Not a syscall.Stat_t")
			}
			if stat.Ino == 1 {
				if err = touch.Run(); err == nil {
					return nil
				}
			}
		}
		logrus.Debugf("Error in attempt %d: %#v", attempt+1, err)
		time.Sleep(time.Second)
	}
	return logError(err.Error())
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

	if !strings.Contains(v.Source, "://") {
		return eeMount(v)
	}
	return ceMount(v)
}

func umountVolume(v *jfsVolume) error {
	cmd := exec.Command("umount", v.Mountpoint)
	logrus.Debug(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		logrus.Errorf("juicefs umount error: %s", out)
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
		case "metaurl":
			v.Source = val
			if !strings.Contains(v.Source, "://") {
				// Default scheme of meta URL is redis://
				v.Source = "redis://" + v.Source
			}
		default:
			v.Options[key] = val
		}
	}

	if v.Name == "" {
		return logError("'name' option required")
	}
	if v.Source == "" {
		v.Source = v.Name
	}

	v.Mountpoint = filepath.Join(d.root, r.Name)
	d.volumes[r.Name] = v

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

	safeClose(v.mountHealthCheck)
	v.mountHealthCheck = nil

	if isMounted, _ := isActiveMount(v.Mountpoint); isMounted {
		if err := umountVolume(v); err != nil {
			return logError("failed to unmount %s: %s", r.Name, err)
		}
	} else {
		logrus.Warnf("mountpoint %s is not active, cleaning up leaked data", v.Mountpoint)
		if err := os.RemoveAll(v.Mountpoint); err != nil {
			return logError(err.Error())
		}
		delete(d.volumes, r.Name)
		d.saveState()
		return nil
	}

	if err := os.Remove(v.Mountpoint); err != nil && !os.IsNotExist(err) {
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

	err := mountVolume(v)
	if err != nil {
		return &volume.MountResponse{}, logError("failed to mount %s: %s", r.Name, err)
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

	if err := umountVolume(v); err != nil {
		return logError("failed to umount %s: %s", r.Name, err)
	}

	safeClose(v.mountHealthCheck)
	v.mountHealthCheck = nil

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
