package controller

import (
	"bytes"
	"errors"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
)

type VncAppImpl struct {
}

func (impl VncAppImpl) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.POST("/vnc", impl.startVncAppHandle)
	v1.POST("/stop_vnc", impl.stopVncAppHandle)
	v1.POST("/vnc/stops", impl.StopAllHandle)
}

func removeDesktopArgs(path string) (filteredPath string) {
	if len(path) > 2 {
		if string(path[len(path)-2]) == "%" {

		}
	}
	fields := strings.Fields(path)
	var validLength = len(fields)
	if len(fields) > 1 {
		//linux do not support a path which contains white space
		newfield := fields[len(fields)-1]
		if string(newfield[0]) == "%" || (len(newfield) > 2 && string(newfield[len(newfield)-2]) == "%") {
			validLength = len(fields) - 1
		}
	}
	for i := 0; i < validLength; i++ {
		filteredPath += fields[i]
		if i < validLength-1 {
			filteredPath += " "
		}
	}
	return
}

func constructXstartup(name, path string) error {
	path = removeDesktopArgs(path)
	script := "#!/bin/bash\n" +
		"ibus-daemon -d  -n " + name + " \n" +
		"sleep 1 \n" +
		"ibus engine lotime \n" +
		"export GDK_BACKEND=x11\n" +
		"export QT_QPA_PLATFORM=xcb\n" +
		"export QT_IM_MODULE=ibus\n" +
		"export QT4_IM_MODULE=ibus\n" +
		"export GTK_IM_MODULE=ibus\n"
	script += "i3 &\n" +
		"export im=ibus\n" + path + "\n"
	data := []byte(script)

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
	Display string
}

func (impl VncAppImpl) stopVncAppHandle(c *gin.Context) {

	var request startAppRequest
	err := c.ShouldBind(&request)
	if err != nil {
		response.ResponseParamterError(c, err)
		return
	}
	if len(request.App) == 0 && !request.SysOnly {
		response.ResponseParamterError(c, errors.New("invalid parameters"))
		return
	}

	err = impl.stopVncApp(request.App, request.SysOnly)
	if err != nil {
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}
	response.Response(c, nil)
}

const VncServer = "Xtigervnc"

func (impl VncAppImpl) StopAllHandle(c *gin.Context) {
	err := impl.StopAll()
	if err != nil {
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}
	response.Response(c, nil)
}

func (impl VncAppImpl) StopAll() error {
	for {
		err, exist, port, app := grepApp(VncServer)
		if err != nil {
			return err
		}
		if exist && len(port) != 0 {
			err := impl.doStop(app, port)
			if err != nil {
				return err
			}
		} else {
			return nil
		}
	}
}

func (impl VncAppImpl) startVncAppHandle(c *gin.Context) {
	var request startAppRequest
	err := c.ShouldBind(&request)
	if err != nil {
		response.ResponseParamterError(c, err)
		return
	}
	port, err := impl.startVncApp(request.App, request.Path, request.SysOnly)
	if err != nil {
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}
	response.Response(c, startAppResponse{
		Port: port,
	})
}

func simplifyPort(port string) (string, error) {
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return "", err
	}
	if portInt >= 6000 {
		return strconv.Itoa(portInt%1000 + 100), nil
	} else {
		return strconv.Itoa(portInt % 100), nil
	}
}

func (impl VncAppImpl) doStop(app, port string) (err error) {
	logger.Info("debug_arg", app+"@"+port)
	port, err = simplifyPort(port)
	if err != nil {
		return
	}
	cmdVnc := exec.Command("vncserver", "--kill", ":"+port)
	cmdVnc.Env = append(os.Environ())
	cmdVnc.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	var stdout, stderr io.ReadCloser
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
	err = cmdVnc.Start()
	if err != nil {
		logger.Error("stop vnc  app failed", app+"@"+port, err)
		err = errors.New("stop vnc server failed")
		return
	}
	output, err := ioutil.ReadAll(io.MultiReader(stdout, stderr))
	if err != nil {
		logger.Error("read start vnc server failed", nil, err)
	}
	cmdVnc.Wait()
	logger.Info("debug_vnc", string(output))
	_, pid, err := grepIbusApp(app)
	if err != nil {
		logger.Error("grep_ibus_daemon_app", nil, err)
	} else {
		if pid != "" {
			iPid, err := strconv.Atoi(pid)
			if err != nil {
				return err
			}
			logger.Info("kill_ibus_daemon", fmt.Sprint(iPid, " "+app))
			syscall.Kill(iPid, syscall.SIGTERM)
		}
	}
	return nil
}

// start a app ,return the port or error
func (impl VncAppImpl) stopVncApp(app string, sysOnly bool) (err error) {
	if sysOnly {
		app = "sysonly"
	}
	logger.Info("stop_app", app)
	app = strings.ToLower(app)
	app = strings.ReplaceAll(app, " ", "_")
	err, exist, port, _ := grepApp(app)
	if err != nil {
		return
	}
	if exist {
		return impl.doStop(app, port)
	}
	return nil
}

// start a app ,return the port or error
func (impl VncAppImpl) startVncApp(app, path string, sysOnly bool) (port string, err error) {
	if sysOnly {
		app = "sysonly"
	}
	logger.Info("start_app", app)
	app = strings.ToLower(app)
	app = strings.ReplaceAll(app, " ", "_")
	err, exist, port, _ := grepApp(app)
	if err != nil {
		return
	}
	if exist {
		return
	}
	var arg []string
	arg = append(arg, "--SecurityTypes=None", "-name="+app, "--I-KNOW-THIS-IS-INSECURE", "-localhost=yes")
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
		logger.Error("start vnc server failed", app, err)
		err = errors.New("start vnc " + app + " failed")
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
	err, _, port, _ = grepApp(app)
	return
}

type startAppResponse struct {
	Port string
}

func grepApp(name string) (err error, exist bool, port, appName string) {
	psCmd := exec.Command("ps", "-ef")
	grepCmd := exec.Command("grep", "Xtigervnc")
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
			appName, port = parseApp(string(line))
			if (name == appName) || (name == VncServer) {
				exist = true
				return
			} else {
				port = ""
			}
		}
	}
	return
}

func parseApp(line string) (appName, port string) {
	fields := strings.Fields(line)
	for index, value := range fields {
		if strings.Contains(value, "rfbport") {
			port = fields[index+1]
		}
		if strings.Contains(value, "desktop") {
			appName = fields[index+1]
		}
	}
	return
}

func grepIbusApp(name string) (exist bool, pid string, err error) {
	psCmd := exec.Command("ps", "-ef")
	grepCmd := exec.Command("grep", "ibus-daemon")
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
	logger.Info("stop_ibus_daemon", name)
	lines := bytes.Split(output.Bytes(), []byte("\n"))
	for _, line := range lines {
		if strings.Contains(string(line), name) {
			fields := strings.Fields(string(line))
			if len(fields) < 3 {
				err = errors.New("not match ibus-daemon")
				return
			}
			pid = fields[1]
			exist = true
			return
		} else {
			pid = ""
			exist = false
		}
	}
	return
}
