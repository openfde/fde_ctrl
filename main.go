package main

import (
	"context"
	"fde_ctrl/conf"
	"fde_ctrl/controller"
	"fde_ctrl/gpu"
	"fde_ctrl/controller/middleware"
	"fde_ctrl/fdedroid"
	"fde_ctrl/logger"
	"fde_ctrl/process_chan"
	"fde_ctrl/websocket"
	"fde_ctrl/windows_manager"
	"flag"
	"fmt"
	"os/exec"


	"github.com/gin-gonic/gin"
)


func setup(r *gin.Engine, configure conf.Configure) error {

	var vnc controller.VncAppImpl
	var apps controller.Apps
	var pm controller.PowerManager
	var  xserver controller.XserverAppImpl
	group := r.Group("/api")
	err := apps.Scan(configure)
	if err != nil {
		return err
	}
	var controllers []controller.Controller
	controllers = append(controllers,  pm, &apps, vnc, xserver)
	for _, value := range controllers {
		value.Setup(group)
	}

	return nil
}

var _version_ = "v0.1"
var _tag_ = "v0.1"
var _date_ = "20230101"

func main() {
	var version, help bool
	flag.BoolVar(&version, "v", false, "-v")
	flag.BoolVar(&help, "h", false, "-h")
	flag.Parse()
	if help {
		fmt.Println("fde_ctrl:")
		fmt.Println("\t-v: print versions and tags")
		fmt.Println("\t-h: print vhelp")
		return
	}

	if version {
		fmt.Printf("Version: %s, tag: %s , date: %s \n", _version_, _tag_, _date_)
		return
	}
	configure, err := conf.Read()
	if err != nil {
		logger.Error("read_conf", nil, err)
		return
	}
	ready,err :=  gpu.IsReady()
	if err != nil {
		logger.Error("gpu_is_ready",nil,err)
		return
	}
	if !ready {
		logger.Warn("gpu_is_not_ready",nil)
		return
	}
	mainCtx, mainCtxCancelFunc := context.WithCancel(context.Background())

	var cmds []*exec.Cmd
	cmdFs := exec.CommandContext(mainCtx, "fde_fs","-m")
	err = cmdFs.Start()
	if err != nil {
		logger.Error("start_mount", nil, err)
		return
	}
	go func() {
		err := cmdFs.Wait()
		if err != nil {
			logger.Error("wait_fs", nil, err)
			mainCtxCancelFunc()
			return
		}
	}()

	//step 1 start windowsmanager
	var cmdWinMan *exec.Cmd
	cmdWinMan, err = windows_manager.Start(mainCtx, configure.WindowsManager, mainCtxCancelFunc)
	if err != nil {
		logger.Error("start_windows_manager", configure.WindowsManager.Name, err)
		return
	}
	logger.Info("start_windows_manager", configure.WindowsManager)
	if cmdWinMan != nil {
		cmds = append(cmds, cmdWinMan)
	}
	var droid fdedroid.Fdedroid
	if configure.WindowsManager.IsWayland() {
		droid = new(fdedroid.Waydroid)
	} else {
		droid = new(fdedroid.Anbox)
	}
	cmdSession, err := droid.Start(mainCtx, mainCtxCancelFunc, configure)
	if err != nil {
		logger.Error("fdedroid_start", configure.WindowsManager.IsWayland(), err)
		killSonProcess(cmds)
		return
	}
	cmds = append(cmds, cmdSession)

	go websocket.SetupWebSocket()
	//scan app from linux
	engine := gin.New()
	engine.Use(middleware.LogHandler(), gin.Recovery())
	engine.Use(middleware.ErrHandler())
	if err := setup(engine, configure); err != nil {
		logger.Error("setup", nil, err)
		return
	}
	go engine.Run("localhost:18080")

	if mainCtx.Err() == nil {
		select {
		case <-mainCtx.Done():
			{
				logger.Info("context_done", "exit due to unexpected canceled context")
				exitFde(configure,cmds)
				return
			}
		case action := <-process_chan.ProcessChan:
			{
				exitFde(configure,cmds)
				switch action {
				case process_chan.Restart:
					{
						logger.Info("restart", "exit due to some one send restart signal")
						cmd := exec.Command("reboot")
						err = cmd.Run()
						if err != nil {
							logger.Error("restart_failed", nil, err)
						}
						return
					}
				case process_chan.Logout:
					{
						// logout
						var vnc controller.VncAppImpl
						vnc.StopAll()
						logger.Info("logout", "exit due to some one send logout signal")
						return
					}
				case process_chan.Poweroff:
					{
						// poweroff
						logger.Info("power_off", "exit due to some one send poweroff signal")
						cmd := exec.Command("shutdown", "-h", "now")
						err = cmd.Run()
						if err != nil {
							logger.Error("shutdown_failed", nil, err)
						}
						return
					}
				}
			}
		}
	} else {
		logger.Error("main_ctx_error", nil, mainCtx.Err())
	}
}
func  exitFde(configure conf.Configure,cmds []*exec.Cmd) {
	err := exec.Command("fde_fs", "-u").Run()
	if err != nil {
		logger.Error("umount_in_main", nil, err)
	}
	killSonProcess(cmds)
	if configure.WindowsManager.IsWayland() {
		fdedroid.StopWaydroidContainer(context.Background())
	} else {
		fdedroid.StopAndroidContainer(context.Background(), fdedroid.FDEContainerName)
	}
}

func killSonProcess(cmds []*exec.Cmd) {
	for index := range cmds {
		logger.Info("kill_son_process", index)
		cmds[index].Process.Kill()
		// cmds[index].Process.Wait()
	}
}

