package controller

import (
	"fde_ctrl/response"
	"os"
	"os/exec"

	"github.com/gin-gonic/gin"
)

type DisplayManager struct {
}

func (impl DisplayManager) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.GET("/display/mirror", impl.mirrorHandler)
}

func (impl DisplayManager) mirrorHandler(c *gin.Context) {
	cmd := exec.Command("xrandr", "--output DP-1", "--auto")
	cmd.Env = append(os.Environ())
	cmd.Run()
	cmd = exec.Command("xrandr", "--output DP-1", "--same-as eDP-1")
	cmd.Env = append(os.Environ())
	cmd.Run()
	response.Response(c, nil)
}
