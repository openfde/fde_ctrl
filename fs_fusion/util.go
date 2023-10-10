package fs_fusion

import (
	"fde_ctrl/logger"
	"fmt"
	"io/fs"
	"context"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/winfsp/cgofuse/fuse"
)

func validPermR(uid, duid, gid, dgid uint32, perm uint32) bool {
	var own uint32
	if uid == duid {
		own = (perm & uint32(0b111000000)) >> 6
		if own >= 4 {
			return true
		}
	} else if gid == dgid {
		own = (perm & uint32(0b000111000)) >> 3
	} else {
		own = perm & uint32(0b000000111)
	}

	if own >= 4 {
		return true
	}
	return false
}

func validPermW(uid, duid, gid, dgid int32, perm uint32) bool {
	var own uint32
	if uid == duid {
		own = (perm & uint32(0b111000000)) >> 6
		if own >= 4 {
			return true
		}
	} else if gid == dgid {
		own = (perm & uint32(0b000111000)) >> 3
	} else {
		own = perm & uint32(0b000000111)
	}

	if (own & 1 << 1) == 2 {
		return true
	}
	return false
}

const FSPrefix = "volumes"
const PathPrefix = "/volumes/"

func readProcess(pid uint32) {
	ioutil.ReadFile("/proc/" + fmt.Sprint(pid) + "/environ")
}

func Mount(cancelFunc context.CancelFunc, mountedChan chan string) (err error) {
	syscall.Umask(0)
	mounts, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		logger.Error("mount_read_mountinfo", mounts, err)
		return
	}
	mountInfoByDevice := readDevicesAndMountPoint(mounts)
	files, err := ioutil.ReadDir("/dev/disk/by-uuid")
	if err != nil {
		logger.Error("mount_read_disk", mounts, err)
		return
	}
	logger.Info("mount_info_by_device", mountInfoByDevice)
	volumes, err := supplementVolume(files, mountInfoByDevice)
	if err != nil {
		logger.Error("mount_supplement_volume", mounts, err)
		return
	}
	logger.Info("in_mount", volumes)
	for _, mountInfo := range volumes {
		_, err := os.Stat(PathPrefix + mountInfo.Volume)
		if err != nil {
			if os.IsNotExist(err) {
				err = os.Mkdir(PathPrefix+mountInfo.Volume, os.ModeDir+0750)
				if err != nil {
					logger.Error("mount_mkdir_for_volumes", mountInfo, err)
					return err
				}
			} else {
				logger.Error("mount_stat_volume", mountInfo.Volume, err)
				return err
			}
		}
		args := []string{"-o", "allow_other", PathPrefix + mountInfo.Volume}
		ptfs := Ptfs{}
		ptfs.root = mountInfo.MountPoint
		logger.Info("for_mount", args)
		logger.Info("for_mount_root", ptfs.root)
		var host *fuse.FileSystemHost
		host = fuse.NewFileSystemHost(&ptfs)
		logger.Info("mount_debug",os.Getuid())
		go func(a []string,f context.CancelFunc){
			tr := host.Mount("", args)
			if !tr {
				cancelFunc()
				logger.Error("mount_fuse_error", tr, nil)
			}
		}(args,cancelFunc)
	}
	mountedChan <- "success"
	return nil
}

type volumeAndMountPoint struct {
	Volume     string
	MountPoint string
	MountID    string
	// MountType  string
}

const LenFieldOfSelfMountInfo = 9

func readDevicesAndMountPoint(mounts []byte) map[string]volumeAndMountPoint {
	var mountInfoByDevice map[string]volumeAndMountPoint
	mountInfoByDevice = make(map[string]volumeAndMountPoint)
	lines := strings.Split(string(mounts), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		//below is a line example of the mountinfo
		//35 29 8:5 / /data rw,relatime shared:7 - ext4 /dev/sda5 rw
		//807 790 7:1 / /var/lib/waydroid/rootfs/vendor ro,relatime shared:446 - ext4 /dev/loop1 ro
		if len(fields) < LenFieldOfSelfMountInfo {
			continue
		}
		//只有第3个元素，仅仅包含/才是原始挂载
		if len(fields[3]) > 1 {
			continue
		}
		//文件系统不是ext4的过滤掉
		if fields[8] != "ext4" {
			continue
		}
		//不需要loop设备
		if strings.Contains(fields[9], "loop") {
			continue
		}
		mountPoint := fields[4]
		if mountPoint == "/boot" {
			continue
		}
		mountID := fields[0]
		if value, exist := mountInfoByDevice[fields[9]]; exist {
			srcMountID, err := strconv.Atoi(value.MountID)
			if err != nil {
				continue
			}
			currentMountID, err := strconv.Atoi(fields[0])
			if err != nil {
				continue
			}
			if currentMountID > srcMountID {
				mountPoint = value.MountPoint
				mountID = value.MountID
			}
		}
		mountInfoByDevice[fields[9]] = volumeAndMountPoint{
			MountPoint: mountPoint,
			MountID:    mountID,
		}
	}
	return mountInfoByDevice

}

func supplementVolume(files []fs.FileInfo, mountInfoByDevice map[string]volumeAndMountPoint) (map[string]volumeAndMountPoint, error) {
	var volumesByDevice map[string]volumeAndMountPoint
	volumesByDevice = make(map[string]volumeAndMountPoint)
	for _, v := range files {
		name, err := os.Readlink("/dev/disk/by-uuid/" + v.Name())
		if err != nil {
			logger.Error("read_volumes", name, err)
			return nil, err
		}
		name = strings.Replace(name, "../..", "/dev", 1)
		if value, exist := mountInfoByDevice[name]; exist {
			volumesByDevice[name] = volumeAndMountPoint{
				Volume:     v.Name(),
				MountPoint: value.MountPoint,
				MountID:    value.MountID,
			}
		}
	}
	return volumesByDevice, nil
}

func UmountAllVolumes() error {
	entries, err := os.ReadDir(PathPrefix)
	if err != nil {
		return err
	}
	syscall.Setreuid(-1,0)
	for _, volume := range entries {
		path := PathPrefix + volume.Name()
		err = syscall.Unmount(path, 0)
		if err != nil {
			logger.Error("umount_volumes", path, err)
		}
	}
	return nil
}
