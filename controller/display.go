package controller

import (
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"io"
	"io/ioutil"
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
	cmd.Env = os.Environ()
	var stdout, stderr io.ReadCloser
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("stdout pipe for vnc server", nil, err)
		return
	}
	stderr, err = cmd.StderrPipe()
	if err != nil {
		logger.Error("stdout pipe for vnc server", nil, err)
		return
	}
	cmd.Start()
	output, err := ioutil.ReadAll(io.MultiReader(stdout, stderr))
	if err != nil {
		logger.Error("read start vnc server failed", nil, err)
	}
	cmd.Wait()

	logger.Info("debug_xrandr_auto", string(output))
	cmd = exec.Command("xrandr", "--output DP-1", "--same-as eDP-1")
	cmd.Env = os.Environ()
	cmd.Start()

	output, err = ioutil.ReadAll(io.MultiReader(stdout, stderr))
	if err != nil {
		logger.Error("read start vnc server failed", nil, err)
	}
	logger.Info("debug_xrandr_mirror", string(output))
	cmd.Wait()
	response.Response(c, nil)
}
