package controller

import (
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

type FsFuseManager struct {
}

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
	cmd := exec.Command("fde_fs", "-pq")
	output, err := cmd.Output()
	if err != nil {
		logger.Error("execute_command", nil, err)
		return false
	}
	result := strings.TrimSpace(string(output))
	if result == "true" {
		return true
	} else {
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

func (impl FsFuseManager) notify() {
	if get() {
		return
	}
	go func() {
		mountFdePtfs()
	}()
}

func (impl *FsFuseManager) Init() {
	userEventNotifier.Register(impl.notify)
	return
}

func (impl FsFuseManager) setHandler(c *gin.Context) {
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
