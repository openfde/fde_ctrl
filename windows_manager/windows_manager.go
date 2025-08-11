package windows_manager

import (
	"context"
	"errors"
	"fde_ctrl/logger"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// mode including desktop shell and desktop environment
type FDEMode string

const DESKTOP_MODE_SHELL FDEMode = "shell"             //start by manual with X11 server
const DESKTOP_MODE_ENVIRONMENT FDEMode = "environment" // start by lightdm with wayland server
const DESKTOP_MODE_SHARED FDEMode = "shared"           // start by manual on ubuntu shared with the wayland server

const SocketCustomName = "fde-wayland-0"
const SocketDefaultName = "wayland-0"

var fdeMode FDEMode = DESKTOP_MODE_ENVIRONMENT

type WindowsManager interface {
	Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc, socket string) (*exec.Cmd, error)
}

func GetFDEMode() FDEMode {
	return fdeMode
}

func Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc, mode FDEMode) (cmdWinMan *exec.Cmd, socket string, err error) {
	var wm WindowsManager
	fdeMode = mode
	userID := os.Getuid()
	//todo the wayland display could be wayland-1 or n not only just wayland-0
	path := "/run/user/" + fmt.Sprint(userID)
	socket = SocketDefaultName

	path = filepath.Join(path, socket)

	if mode != DESKTOP_MODE_SHELL {
		if mode == DESKTOP_MODE_ENVIRONMENT {
			wm = new(Mutter)
			os.Remove(path) //do not rm socket when in shared mode
			os.Remove(path + ".lock")
			cmdWinMan, err = wm.Start(mainCtx, mainCtxCancelFunc, socket)
			if err != nil {
				return
			}
		}

		waitCnt := 0
		//wait for the wayland-0
		for {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				logger.Info("wayland-display", "not exist")
				time.Sleep(time.Second)
				waitCnt++
			} else {
				break
			}
			if waitCnt > 60 {
				logger.Error("wait_for_wayland-display", "timeout 60s", nil)
				return nil, socket, errors.New("time out for waiting wayland display")
			}
		}
		err = os.Chmod(path, 0777)
		if err != nil {
			logger.Error("chmod_wayland_display", "failed to set permissions", err)
			return nil, socket, err
		}
	}
	//enable tap to click
	settingCmd := exec.CommandContext(mainCtx, "gsettings", "set", "org.gnome.desktop.peripherals.touchpad", "tap-to-click", "true")
	err = settingCmd.Run()
	if err != nil {
		logger.Error("wayland_set_tap_to_click", nil, err)
	}
	return cmdWinMan, socket, nil
}
