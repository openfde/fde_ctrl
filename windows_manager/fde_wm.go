package windows_manager

import (
	"fde_ctrl/logger"
	"fde_ctrl/tools"

	"context"
	"os"
	"os/exec"
)

type Mutter struct {
}

const fdeWindowsManager = "fde_wm" //actually fde_wm is renamed from mutter

func (wm Mutter) Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc, socket string) (cmdWinMan *exec.Cmd, err error) {
	cmdWinMan = exec.CommandContext(mainCtx, fdeWindowsManager,"--mutter-plugin=/usr/local/lib/aarch64-linux-gnu/mutter-6/plugins/libdefault.so")
	_, exist := tools.ProcessExists(fdeWindowsManager)
	if exist {
		return nil, nil
	}
	cmdWinMan.Env = append(os.Environ(), "XDG_SESSION_TYPE=wayland")
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

	return cmdWinMan, nil

}
