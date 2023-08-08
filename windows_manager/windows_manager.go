package windows_manager

import (
	"fde_ctrl/conf"
	"fde_ctrl/logger"
	"fde_ctrl/tools"

	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const fdeWindowsManager = "fde_wm" //actually fde_wm is renamed from mutter

func Start(mainCtx context.Context, windowsConfig conf.WindowsManager, mainCtxCancelFunc context.CancelFunc) (cmdWinMan *exec.Cmd, err error) {
	var name = windowsConfig.Name
	switch name {
	case fdeWindowsManager: //actually fde_wm is renamed from mutter
		{
			cmdWinMan = exec.CommandContext(mainCtx, name)
		}
	default:
		{
			cmdWinMan = exec.CommandContext(mainCtx, name)
		}
	}
	_, exist := tools.ProcessExists(name)
	if exist {
		return nil, nil
	}
	if windowsConfig.IsWayland() {
		cmdWinMan.Env = append(os.Environ(), "XDG_SESSION_TYPE=wayland")
	}
	err = cmdWinMan.Start()
	if err != nil {
		logger.Error("start_wm", nil, err)
		mainCtxCancelFunc()
		return
	}
	go func() {
		err := cmdWinMan.Wait()
		if err != nil {
			logger.Error("wait_wm", nil, err)
			return
		}
		mainCtxCancelFunc()
	}()

	waitCnt := 0

	//wait for the wayland-0
	if windowsConfig.IsWayland() {
		for {
			userID := os.Getuid()
			path := "/run/user/" + fmt.Sprint(userID) + "/wayland-0"
			if _, err := os.Stat(path); os.IsNotExist(err) {
				logger.Info("wayland-0", "not exist")
				time.Sleep(time.Second)
				waitCnt++
			} else {
				break
			}
			if waitCnt > 60 {
				logger.Error("wait_for_wayland-0", "timeout 60s", nil)
				break
			}
		}
	}

	return cmdWinMan, nil

}
