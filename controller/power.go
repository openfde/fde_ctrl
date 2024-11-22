package controller

import (
	"fde_ctrl/process_chan"
	"fde_ctrl/response"
	"os/exec"

	"github.com/gin-gonic/gin"
)

type PowerManager struct {
}

func (impl PowerManager) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.POST("/power/off", impl.poweroffHandler)
	v1.POST("/power/logout", impl.logoutHandler)
	v1.POST("/power/restart", impl.restartHandler)
	v1.POST("/power/lock", impl.lockHandler)
}

func (impl PowerManager) lockHandler(c *gin.Context) {
	exec.Command("dm-tool", "lock").Start()
	response.Response(c, nil)
}

func (impl PowerManager) logoutHandler(c *gin.Context) {
	process_chan.SendLogout()
	response.Response(c, nil)
}

func (impl PowerManager) poweroffHandler(c *gin.Context) {
	process_chan.SendPoweroff()
	response.Response(c, nil)
}

func (impl PowerManager) restartHandler(c *gin.Context) {
	process_chan.SendRestart()
	response.Response(c, nil)
}
