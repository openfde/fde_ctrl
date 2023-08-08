package fdedroid

import (
	"context"
	"fde_ctrl/conf"
	"os/exec"
)

type Fdedroid interface {
	Start(mainCtx context.Context, mainCtxCancelFunc context.CancelFunc, configure conf.Configure) (cmdWaydroid *exec.Cmd, err error)
}
