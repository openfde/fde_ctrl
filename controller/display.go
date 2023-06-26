package controller

import (
	"bufio"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"os"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

type DisplayManager struct {
}

func (impl DisplayManager) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.GET("/display/mirror", impl.mirrorHandler)
}

func (impl DisplayManager) isConnected() bool {
	cmd := exec.Command("xrandr")
	cmd.Env = os.Environ()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("stdoutpipe_xrandr", nil, err)
		return false
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		logger.Error("start_xrandr", nil, err)
		return false
	}
	key := "DP-1 disconnected"

	// 逐行读取标准输出
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, key) && !strings.Contains(line, "eDP-1") {
			return false
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Error("xrandr_scanner", nil, err)
		return false
	}

	// 等待命令完成
	if err := cmd.Wait(); err != nil {
		logger.Error("xrandr_wait", nil, err)
		return false
	}
	return true
}

func (impl DisplayManager) mirrorHandler(c *gin.Context) {
	if !impl.isConnected() {
		response.Response(c, "display disconnected")
		return
	}
	cmd := exec.Command("xrandr", "--output", "DP-1", "--auto")
	cmd.Env = os.Environ()
	cmd.Run()
	cmd = exec.Command("xrandr", "--output", "DP-1", "--same-as", "eDP-1")
	cmd.Env = os.Environ()
	cmd.Run()
	response.Response(c, "display connected")
}
