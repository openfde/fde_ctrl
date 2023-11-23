package windows_manager

import (
	"errors"
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
	cmdWinMan = exec.CommandContext(mainCtx, name)
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
			//todo the wayland display could be wayland-1 or n not only just wayland-0
			path := "/run/user/" + fmt.Sprint(userID) + "/wayland-0"
			if _, err := os.Stat(path); os.IsNotExist(err) {
				logger.Info("wayland-disopay", "not exist")
				time.Sleep(time.Second)
				waitCnt++
			} else {
				break
			}
			if waitCnt > 60 {
				logger.Error("wait_for_wayland-display", "timeout 60s", nil)
				return nil, errors.New("time out for waiting wayland display")
			}
		}
		//enable tap to click
		settingCmd := exec.CommandContext(mainCtx, "gsettings", "set", "org.gnome.desktop.peripherals.touchpad", "tap-to-click", "true")
		err = settingCmd.Run()
		if err != nil {
			logger.Error("wayland_set_tap_to_click", nil, err)
		}
	}

	return cmdWinMan, nil

}
