package main

import (
	"bytes"
	"errors"
	"fde_ctrl/logger"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
)

type VncAppInterface interface {
	Setup(r *gin.RouterGroup)
}

type VncAppImpl struct {
}

func (impl VncAppImpl) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.POST("/vnc", impl.startVncAppHandle)
}

func constructXstartup(name, path string) error {
	data := []byte("#!/bin/bash\n" + path + "\n")

	file, err := os.OpenFile("/tmp/"+name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		logger.Error("Error creating file:", name, err)
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		logger.Error("Error writing to file:", name, err)
		return err
	}
	return nil
}

type startAppRequest struct {
	App     string
	Path    string
	SysOnly bool
}

func (impl VncAppImpl) startVncAppHandle(c *gin.Context) {
	var request startAppRequest
	err := c.ShouldBind(&request)
	if err != nil {
		ResponseParamterError(c, err)
		return
	}
	port, err := impl.startVncApp(request.App, request.Path, request.SysOnly)
	if err != nil {
		ResponseError(c, http.StatusInternalServerError, err)
		return
	}
	Response(c, startAppResponse{
		Port: port,
	})
}

// start a app ,return the port or error
func (impl VncAppImpl) startVncApp(app, path string, sysOnly bool) (port string, err error) {
	if sysOnly {
		app = "sysonly"
	}
	logger.Info("start_app", app)
	app = strings.ToLower(app)
	app = strings.ReplaceAll(app, " ", "_")
	err, exist, port := grepApp(app)
	if err != nil {
		return
	}
	if exist {
		return
	}
	var arg []string
	arg = append(arg, "--SecurityTypes=None", "-name="+app, "--I-KNOW-THIS-IS-INSECURE")
	logger.Info("app_not_start", app)
	if !sysOnly {
		err = constructXstartup(app, path)
		if err != nil {
			return
		}
		arg = append(arg, "-xstartup=/tmp/"+app)
	}

	// arg = append(arg, "-localhost=yes")

	logger.Info("debug_arg", arg)
	cmdVnc := exec.Command("vncserver", arg...)
	cmdVnc.Env = append(os.Environ())
	cmdVnc.Env = append(cmdVnc.Env, "LD_PRELOAD=/lib/aarch64-linux-gnu/libgcc_s.so.1")

	cmdVnc.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	debugMode := os.Getenv("fde_debug")
	var stdout, stderr io.ReadCloser
	if debugMode == "debug" {
		stdout, err = cmdVnc.StdoutPipe()
		if err != nil {
			logger.Error("stdout pipe for vnc server", nil, err)
			return
		}
		stderr, err = cmdVnc.StderrPipe()
		if err != nil {
			logger.Error("stdout pipe for vnc server", nil, err)
			return
		}
	}

	err = cmdVnc.Start()
	if err != nil {
		logger.Error("start vnc server failed", nil, err)
		err = errors.New("start vnc server failed")
		return
	}
	// var wstatus syscall.WaitStatus
	// _, err = syscall.Wait4(cmdVnc.Process.Pid, &wstatus, 0, nil)
	// if err != nil {
	// 	logger.Error("wait vnc server failed", nil, err)
	// }
	if debugMode == "debug" {
		output, err := ioutil.ReadAll(io.MultiReader(stdout, stderr))
		if err != nil {
			logger.Error("read start vnc server failed", nil, err)
		}
		logger.Info("debug_vnc", output)
	}
	cmdVnc.Wait()
	//to grep the port
	err, _, port = grepApp(app)
	return
}

type startAppResponse struct {
	Port string
}

func grepApp(name string) (err error, exist bool, port string) {
	psCmd := exec.Command("ps", "-ef")
	grepCmd := exec.Command("grep", "vnc")
	xgrepCmd := exec.Command("grep", "-v", "grep")

	// 将 ps 命令的输出传递给 grep 命令进行过滤
	var output bytes.Buffer
	grepCmd.Stdin, _ = psCmd.StdoutPipe()
	xgrepCmd.Stdin, _ = grepCmd.StdoutPipe()
	xgrepCmd.Stdout = &output
	err = psCmd.Start()
	if err != nil {
		return
	}
	err = grepCmd.Start()
	if err != nil {
		return
	}
	err = xgrepCmd.Start()
	if err != nil {
		return
	}
	err = psCmd.Wait()
	if err != nil {
		return
	}
	grepCmd.Wait()
	xgrepCmd.Wait()
	// 解析 grep 命令的输出

	lines := bytes.Split(output.Bytes(), []byte("\n"))
	for _, line := range lines {
		if strings.Contains(string(line), name) {
			var appName string
			appName, port = parseApp(string(line))
			if name == appName {
				exist = true
				return
			} else {

				port = ""
			}
		}
	}
	return
}

func parseApp(args string) (appName, port string) {
	// 将args按空格分割成多个参数
	argList := strings.Split(args, "tigervnc")
	if len(argList) < 2 {
		return
	}
	argList = strings.Split(argList[1], " ")
	if len(argList) < 3 {
		return
	}
	argList = argList[2:]
	// 创建一个FlagSet对象
	fs := flag.NewFlagSet("temporaryFlagSet", flag.ContinueOnError)
	fs.Usage = func() {}

	// 定义一个名为desktop的string类型flag
	fs.StringVar(&appName, "desktop", "", "desktop default")
	fs.StringVar(&port, "rfbport", "5901", "5901")

	var ignore bytes.Buffer
	fs.SetOutput(&ignore)

	// 解析参数
	fs.Parse(argList)
	return

}
