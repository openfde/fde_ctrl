package controller

import (
	"errors"
	"fde_ctrl/conf"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type XserverAppImpl struct {
	Conf conf.Configure
}

func (impl XserverAppImpl) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.POST("/xserver", impl.startAppHandle)
}

func (impl XserverAppImpl) isClientServer(app string) string {
	for _, v := range impl.Conf.FusionApp.CServerList {
		if v.ClientName == app {
			return v.ServerName
		}
	}
	return ""
}

const LOCAL_SERVER = ":0"

func constructXServerstartup(name, path, display, serverName string) (bashFile string, err error) {
	path = removeDesktopArgs(path)
	data := []byte("#!/bin/bash\n" +
		"export GDK_BACKEND=x11\n" +
		"export QT_QPA_PLATFORM=xcb\n" +
		"export DISPLAY=" + display + "\n")
	if serverName != "" {
		data = append(data, []byte(serverName+" & \n")...)
	}
	if display == LOCAL_SERVER { //local server means
		data = append(data, []byte("fde_switch_next_desktop \n")...)
	}
	data = append(data, []byte(path+"\n")...)

	bashFile = "/tmp/xserver_" + name
	file, err := os.OpenFile(bashFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		logger.Error("Error creating file:", name, err)
		return
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		logger.Error("Error writing to file:", name, err)
		return
	}
	return
}

func (impl XserverAppImpl) startAppHandle(c *gin.Context) {
	var request startAppRequest
	err := c.ShouldBind(&request)
	if err != nil {
		response.ResponseParamterError(c, err)
		return
	}
	// Check if xserver process is already running
	cmd := exec.Command("pgrep", "-f", "com.fde.x11.xserver")
	if err := cmd.Run(); err == nil {
		// Process exists
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			response.ResponseError(c, http.StatusPreconditionRequired, errors.New("xserver process already running"))
			return
		}
	}
	err = impl.startApp(request.App, request.Path, request.Display, request.WithOutTheme)
	if err != nil {
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}
	response.Response(c, startAppResponse{
		Port: request.Display,
	})
}

func (impl XserverAppImpl) isWitoutTheme(app string) bool {
	for _, v := range impl.Conf.Xserver.WithOutThemeList {
		if strings.Contains(app, v) {
			return true
		}
	}
	return false
}

// start a app ,return the port or error
func (impl XserverAppImpl) startApp(app, path, display string, withoutTheme bool) (err error) {
	logger.Info("start_app", app+" "+display)
	serverName := impl.isClientServer(path)

	filePath, err := constructXServerstartup(app, path, display, serverName)
	if err != nil {
		return
	}
	withoutThemeConfig := impl.isWitoutTheme(app)
	cmdApp := exec.Command(filePath)
	if checkDistribID(Kylin) {
		if !withoutTheme && !withoutThemeConfig {
			cmdApp.Env = append(cmdApp.Env, "QT_QPA_PLATFORMTHEME=ukui")
		}
	}
	// cmdVnc.Env = append(os.Environ())
	for _, v := range os.Environ() {
		if withoutTheme || withoutThemeConfig {
			if strings.Contains(v, "QT_QPA_PLATFORMTHEME") || strings.Contains(v, "XDG_CURRENT_DESKTOP") {
				continue
			}
		}
		if strings.Contains(v, "DISPLAY") {
			continue
		}
		cmdApp.Env = append(cmdApp.Env, v)
	}
	logger.Info("env_input", cmdApp.Env)

	cmdApp.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	debugMode := os.Getenv("fde_debug")
	var stdout, stderr io.ReadCloser
	if debugMode == "debug" {
		stdout, err = cmdApp.StdoutPipe()
		if err != nil {
			logger.Error("stdout pipe for xserver", nil, err)
			return
		}
		stderr, err = cmdApp.StderrPipe()
		if err != nil {
			logger.Error("stdout pipe for xserver", nil, err)
			return
		}
	}

	err = cmdApp.Start()
	if err != nil {
		logger.Error("start xserver failed", app, err)
		err = errors.New("start xserver " + app + " failed")
		return
	}
	// var wstatus syscall.WaitStatus
	// _, err = syscall.Wait4(cmdVnc.Process.Pid, &wstatus, 0, nil)
	// if err != nil {
	// 	logger.Error("wait vnc server failed", nil, err)
	// }
	if debugMode == "debug" {
		output, err := io.ReadAll(io.MultiReader(stdout, stderr))
		if err != nil {
			logger.Error("read start xserver server failed", nil, err)
		}
		logger.Info("debug_xserver", output)
	}
	timer := time.NewTimer(500 * time.Millisecond)
	var chWait chan struct{}
	go func() {
		err := cmdApp.Wait()
		if err != nil {
			logger.Error("wait_app", app, err)
			chWait <- struct{}{}
		}
	}()
	select {
	case <-chWait:
		{
			//i3 failed
			return errors.New("wait xserver " + app + " failed")
		}
	case <-timer.C:
		{
			//after 500ms waitting
		}
	}
	return
}
