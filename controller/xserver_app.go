package controller

import (
	"errors"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"io"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type XserverAppImpl struct {
}

func (impl XserverAppImpl) Setup(r *gin.RouterGroup) {
	home, err := os.UserHomeDir()
	if err == nil {
		_, err := os.Stat(home + "/.config")
		if err != nil {
			if os.IsNotExist(err) {
				os.Mkdir(home+"/.config", os.ModeDir|0700)
			}
		}
		_, err = os.Stat(home + "/.config/i3")
		if err != nil {
			if os.IsNotExist(err) {
				os.Mkdir(home+"/.config/i3", os.ModeDir|0700)
			}
		}
		os.Remove(home + "/.config/i3/config")
		impl.copyFile(home+"/.config/i3/config", "/etc/i3/config")
		os.Chown(home+"/.config/i3/config", os.Getuid(), os.Getegid())
	}
	v1 := r.Group("/v1")
	v1.POST("/xserver", impl.startAppHandle)
}

func (impl XserverAppImpl) copyFile(dst, src string) (err error) {
	srcFile, _ := os.Open(src)
	defer srcFile.Close()

	dstFile, _ := os.Create(dst)
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		logger.Error("copy_i3_config", nil, err)
	}
	return
}

func constructXServerstartup(name, path, display string) (bashFile string, err error) {
	path = removeDesktopArgs(path)
	data := []byte("#!/bin/bash\n" +
		"export GDK_BACKEND=x11\n" +
		"export QT_QPA_PLATFORM=xcb\n" +
		"export DISPLAY=" + display + "\n")
	if checkDistribID() {
		data = append(data, []byte(
			"export QT_QPA_PLATFORMTHEME=ukui \n ")...)
	}
	data = append(data, []byte(path+"\n")...)

	bashFile = "/tmp/" + name
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
	err = impl.startApp(request.App, request.Path, request.Display)
	if err != nil {
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}
	response.Response(c, startAppResponse{
		Port: request.Display,
	})
}

// start a app ,return the port or error
func (impl XserverAppImpl) startApp(app, path, display string) (err error) {
	logger.Info("start_app", app+" "+display)
	filePath, err := constructXServerstartup(app, path, display)
	if err != nil {
		return
	}
	cmdi3 := exec.Command("i3")
	cmdi3.Env = append(cmdi3.Env, "DISPLAY="+display)
	err = cmdi3.Start()
	if err != nil {
		logger.Error("start_xserver_i3", app, err)
		return
	}
	timer := time.NewTimer(500 * time.Millisecond)
	var ch chan struct{}
	go func() {
		err := cmdi3.Wait()
		if err != nil {
			logger.Error("wait_i3", nil, err)
			ch <- struct{}{}
		}
	}()
	select {
	case <-ch:
		{
			//i3 failed
			return errors.New("wait i3 failed for staring " + app)
		}
	case <-timer.C:
		{
			//after 500ms waitting
		}
	}

	cmdApp := exec.Command(filePath)
	// cmdVnc.Env = append(os.Environ())
	// cmdVnc.Env = append(cmdVnc.Env, "LD_PRELOAD=/lib/aarch64-linux-gnu/libgcc_s.so.1")

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
	timer = time.NewTimer(500 * time.Millisecond)
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
