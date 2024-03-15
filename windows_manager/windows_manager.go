package windows_manager

import (
	"context"
	"errors"
	"fde_ctrl/logger"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// mode including desktop shell and desktop environment
type FDEMode string

const DESKTOP_MODE_SHELL FDEMode = "shell"             //start by manual
const DESKTOP_MODE_ENVIRONMENT FDEMode = "environment" // start by lightdm

type WindowsManager interface {
	Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc) (*exec.Cmd, error)
}

func Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc, mode FDEMode) (cmdWinMan *exec.Cmd, err error) {
	var wm WindowsManager
	if mode == DESKTOP_MODE_SHELL {
		wm = new(WestonWM)
	} else {
		wm = new(Mutter)
	}
	cmdWinMan, err = wm.Start(mainCtx, mainCtxCancelFunc)
	if err != nil {
		return
	}
	waitCnt := 0
	//wait for the wayland-0
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
	return cmdWinMan, nil
}
