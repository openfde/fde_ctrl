package gpu

import (
	"errors"
	"fde_ctrl/windows_manager"
)

type JM9100 struct {
}

func (gpu JM9100) IsReady(mode windows_manager.FDEMode) (bool, error) {
	if mode == windows_manager.DESKTOP_MODE_ENVIRONMENT {
		return false, errors.New("JM9100 is not supported in environment mode")
	}
	return true, nil
}
