package controller

import (
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type FsFuseManager struct {
	Fusing string
}

var fslock sync.Mutex

func (impl FsFuseManager) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.GET("/fs_fusing", impl.getHandler)
	v1.POST("/fs_fusing", impl.setHandler)
	v1.POST("/fs_fusing/exit", impl.exitHandler)
}

type fdefsResponse struct {
	Mounted bool
}

func get() bool {
	fslock.Lock()
	// Check if /proc/self/mounts contains "fde_ptfs" keyword
	mounts, err := ioutil.ReadFile("/proc/self/mounts")
	defer fslock.Unlock()
	if err != nil {
		logger.Error("read_mounts_file", nil, err)
		return false
	}
	if strings.Contains(string(mounts), "fde_ptfs") {
		logger.Info("fde_ptfs_found", nil)
		return true
	} else {
		logger.Info("fde_ptfs_not_found", nil)
		return false
	}

}

func (impl FsFuseManager) getHandler(c *gin.Context) {
	if get() {
		response.Response(c, fdefsResponse{Mounted: true})
	} else {
		response.Response(c, fdefsResponse{Mounted: false})
	}
	return
}

func mountFdePtfs(sourcePath, targetPath string) error {
	cmd := exec.Command("fde_ptfs", "-o", "nonempty", "-o", "allow_other", sourcePath, targetPath)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

type mountInfo struct {
	Root   string
	Target string
}

var homeDirNameMap map[string]string
var androidDirList, linuxDirList []string

func init() {
	homeDirNameMap = make(map[string]string)
	// Initialize the map with key-value pairs
	homeDirNameMap["Documents"] = "文档"
	homeDirNameMap["Download"] = "下载"
	homeDirNameMap["Music"] = "音乐"
	homeDirNameMap["Videos"] = "视频"
	homeDirNameMap["Pictures"] = "图片"
	linuxDirList = append(linuxDirList, "Documents")
	linuxDirList = append(linuxDirList, "Download")
	linuxDirList = append(linuxDirList, "Music")
	linuxDirList = append(linuxDirList, "Videos")
	linuxDirList = append(linuxDirList, "Pictures")

	androidDirList = append(androidDirList, "Documents")
	androidDirList = append(androidDirList, "Download")
	androidDirList = append(androidDirList, "Music")
	androidDirList = append(androidDirList, "Movies")
	androidDirList = append(androidDirList, "Pictures")
}

func getUserFolders() ([]string, []string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	var realLinuxDirList = make([]string, len(linuxDirList))
	var realAndroidList = make([]string, len(androidDirList))
	_, err = os.Stat(filepath.Join(homeDir, linuxDirList[0]))
	if err == nil { //en
		for i, v := range linuxDirList {
			realLinuxDirList[i] = filepath.Join(homeDir, v)
			realAndroidList[i] = filepath.Join(homeDir, ".local/share/openfde/media/0", androidDirList[i])
		}
	} else { //zh
		for i, v := range linuxDirList {
			realLinuxDirList[i] = filepath.Join(homeDir, homeDirNameMap[v])
			realAndroidList[i] = filepath.Join(homeDir, ".local/share/openfde/media/0", androidDirList[i])
		}
	}
	return realLinuxDirList, realAndroidList, nil

}

var fsExit chan string

func (impl FsFuseManager) setHandler(c *gin.Context) {
	if get() {
		response.Response(c, nil)
		return
	}
	list, androidList, err := getUserFolders()
	if err != nil {
		logger.Error("get_user_folders", nil, err)
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}
	go func() {
		for i, v := range list {
			logger.Info("mount_fde_ptfs", list[i]+" "+androidList[i])
			go mountFdePtfs(v, androidList[i])
		}
		select {
		case <-fsExit:
			return
		}
	}()

	response.Response(c, nil)
}

func (impl FsFuseManager) exitHandler(c *gin.Context) {
	if get() {
		fsExit <- "exit"
		return
	}

	response.Response(c, nil)
}
