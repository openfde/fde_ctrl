package fdedroid

import (
	"context"
	"fde_ctrl/conf"
	"fde_ctrl/windows_manager"
	"os/exec"
)

type Fdedroid interface {
	Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc, configure conf.Configure, socket string, windows_manager.FDEMode mode) (cmdWaydroid *exec.Cmd, err error)
}
