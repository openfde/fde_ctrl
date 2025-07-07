/*
* Copyright （C)  2023 OpenFDE , All rights reserved.
 */

package main

import (
	"context"
	"fde_ctrl/conf"
	"fde_ctrl/controller"
	"fde_ctrl/controller/middleware"
	navi "fde_ctrl/desktop_navi"
	"fde_ctrl/fdedroid"
	"fde_ctrl/gpu"
	"fde_ctrl/logger"
	"fde_ctrl/process_chan"
	"fde_ctrl/tools"
	"fde_ctrl/windows_manager"
	"flag"
	"fmt"
	"os"

	"os/exec"

	"github.com/gin-gonic/gin"
)

var _version_ = "v0.1"
var _tag_ = "v0.1"
var _date_ = "20230101"

func parseArgs() (mode, app, msg string, snavi, return_directly bool) {
	var version, help bool
	flag.BoolVar(&version, "v", false, "-v")
	flag.BoolVar(&help, "h", false, "-h")
	flag.BoolVar(&snavi, "n", false, "-n")
	flag.StringVar(&mode, "m", string(windows_manager.DESKTOP_MODE_ENVIRONMENT), "-m")
	flag.StringVar(&app, "a", string("openfde"), "-a")
	flag.StringVar(&msg, "msg", "", "-msg {json string}")
	flag.Parse()
	if help {
		fmt.Println("fde_ctrl:")
		fmt.Println("\t-v: print versions and tags")
		fmt.Println("\t-h: print help")
		fmt.Println("\t-n: start navi")
		fmt.Println("\t-m: input the running mode[shell|environment|shared]")
		fmt.Println("\t -msg: input a json string to notify by use dbus ")
		return_directly = true
		return
	}
	if version {
		fmt.Printf("Version: %s, tag: %s , date: %s \n", _version_, _tag_, _date_)
		return_directly = true
		return
	}
	return
}

const errnoPidMaxOutOfLimit = 10

func main() {
	var mode, app, msg string
	var snavi bool
	var return_directly bool
	if mode, app, msg, snavi, return_directly = parseArgs(); return_directly {
		return
	}

	if len(msg) != 0 {
		logger.Info("send_dbus_message", msg)
		err := tools.SendDbusMessage(msg)
		if err != nil {
			os.Exit(1)
			fmt.Println(err)
		}
		os.Exit(0)
	}

	if CheckPidMax() {
		os.Exit(errnoPidMaxOutOfLimit)
	}

	configure, err := conf.Read()
	if err != nil {
		logger.Error("read_conf", nil, err)
		return
	}
	ready, err := gpu.IsReady(windows_manager.FDEMode(mode))
	if err != nil {
		logger.Error("gpu_is_ready", nil, err)
		return
	}
	if !ready {
		logger.Warn("gpu_is_not_ready", nil)
		return
	}

	if snavi {
		logger.Info("start_navi", nil)
		navi.StartFdeNavi() // start desktop navi
	}
	mainCtx, mainCtxCancelFunc := context.WithCancel(context.Background())

	var cmds []*exec.Cmd
	cmdFs := exec.CommandContext(mainCtx, "fde_fs", "-m")
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
	var socketName string
	cmdWinMan, socketName, err = windows_manager.Start(mainCtx, mainCtxCancelFunc, windows_manager.FDEMode(mode))
	if err != nil {
		logger.Error("start_windows_manager", mode, err)
		return
	}
	logger.Info("start_windows_manager_mode", mode)
	if cmdWinMan != nil {
		cmds = append(cmds, cmdWinMan)
	}
	var droid fdedroid.Fdedroid
	droid = new(fdedroid.Waydroid)
	cmdSession, err := droid.Start(mainCtx, mainCtxCancelFunc, configure, socketName)
	if err != nil {
		logger.Error("fdedroid_start", mode, err)
		killSonProcess(cmds)
		return
	}
	cmds = append(cmds, cmdSession)

	//scan app from linux
	engine := gin.New()
	engine.Use(middleware.LogHandler(), gin.Recovery())
	engine.Use(middleware.ErrHandler())
	controller.Setup(engine, app, configure)
	go engine.Run("localhost:18080")

	if mainCtx.Err() == nil {
		select {
		case <-mainCtx.Done():
			{
				logger.Info("context_done", "exit due to unexpected canceled context")
				exitFde(configure, cmds)
				return
			}
		case action := <-process_chan.ProcessChan:
			{
				exitFde(configure, cmds)
				switch action {
				case process_chan.Restart:
					{
						logger.Info("restart", "exit due to some one send restart signal")
						var cmd *exec.Cmd
						if mode != string(windows_manager.DESKTOP_MODE_ENVIRONMENT) {
							cmd = exec.Command("fde_utils restart &")
						} else {
							cmd = exec.Command("reboot")
						}
						err = cmd.Run()
						if err != nil {
							logger.Error("restart_failed", nil, err)
						}
						return
					}
				case process_chan.Logout, process_chan.Unexpected:
					{
						// logout
						var vnc controller.VncAppImpl
						vnc.StopAll()
						if action == process_chan.Unexpected {
							logger.Info("unexpected_exit", "pid max is out of limit 65535")
							os.Exit(10)
						} else {
							logger.Info("logout", "exit due to some one send logout signal")
						}
						return
					}
				case process_chan.Poweroff:
					{
						// poweroff
						logger.Info("power_off", "exit due to some one send poweroff signal")
						var cmd *exec.Cmd
						if mode != string(windows_manager.DESKTOP_MODE_ENVIRONMENT) {
							cmd = exec.Command("fde_utils", "stop")
						} else {
							cmd = exec.Command("shutdown", "-h", "now")
						}
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
func exitFde(configure conf.Configure, cmds []*exec.Cmd) {
	controller.FsFusingExit()
	err := exec.Command("fde_fs", "-u").Run()
	if err != nil {
		logger.Error("umount_in_main", nil, err)
	}
	killSonProcess(cmds)
	fdedroid.StopWaydroidContainer(context.Background())
}

func killSonProcess(cmds []*exec.Cmd) {
	for index := range cmds {
		logger.Info("kill_son_process", index)
		cmds[index].Process.Kill()
		// cmds[index].Process.Wait()
	}
}
