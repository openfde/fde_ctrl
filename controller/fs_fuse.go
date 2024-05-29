package controller

import (
	"fde_ctrl/conf"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"io/ioutil"
	"os/exec"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type FsFuseManager struct {
	Config conf.CustomerConfigure
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

func mountFdePtfs() error {
	cmd := exec.Command("fde_fs", "-pm")
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
func umountFdePtfs() error {
	cmd := exec.Command("fde_fs", "-pu")
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

func (impl FsFuseManager) setHandler(c *gin.Context) {
	if !impl.Config.PersonalDirFusing.Fusing {
		response.Response(c, nil)
		return
	}
	if get() {
		response.Response(c, nil)
		return
	}
	go func() {
		mountFdePtfs()
	}()

	response.Response(c, nil)
}

func FsFusingExit() {
	if get() {
		umountFdePtfs()
		return
	}
}

func (impl FsFuseManager) exitHandler(c *gin.Context) {
	FsFusingExit()
	response.Response(c, nil)
}
