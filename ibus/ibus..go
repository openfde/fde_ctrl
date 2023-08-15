package ibus

import (
	"context"
	"fde_ctrl/logger"
	"fde_ctrl/tools"
	"os/exec"
)

const ibusName = "ibus-daemon"

func Start(mainCtx context.Context) error {
	cmdIbus := exec.CommandContext(mainCtx, ibusName, "-d")
	_, exist := tools.ProcessExists(ibusName)
	if exist {
		return nil
	}
	err := cmdIbus.Run()
	if err != nil {
		logger.Error("start_ibus", nil, err)
		return err
	}
	return nil
}
