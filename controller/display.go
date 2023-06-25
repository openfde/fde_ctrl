package controller

import (
	"bufio"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"io"
	"io/ioutil"
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

func (impl DisplayManager) mirrorHandler(c *gin.Context) {

	// 将 ps 命令的输出传递给 grep 命令进行过滤
	grepCmd := exec.Command("xrandr")
	grepCmd.Env = os.Environ()
	infoOutput, _ := grepCmd.StdoutPipe()
	err := grepCmd.Start()
	if err != nil {
		return
	}
	err = grepCmd.Start()
	if err != nil {
		return
	}

	// 读取子进程的输出
	scanner := bufio.NewScanner(infoOutput)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "DP-1 disconnected") {
			response.Response(c, nil)
			return
		}
		// 在这里对子进程的输出进行分析处理
		logger.Info("debug_line", line)
	}
	cmd := exec.Command("xrandr", "--output", "DP-1", "--auto")
	cmd.Env = os.Environ()
	var stdout, stderr io.ReadCloser
	stdout, err = cmd.StdoutPipe()
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
	cmd = exec.Command("xrandr", "--output", "DP-1", "--same-as", "eDP-1")
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
