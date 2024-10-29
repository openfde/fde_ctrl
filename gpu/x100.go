package gpu

import (
	"errors"
	"fde_ctrl/windows_manager"
	"os"
)

type x100 struct {
}

func (gpu x100) IsReady(windows_manager.FDEMode) (bool, error) {
	_, err := os.Stat("/etc/powervr.ini") //must has  cur_gl with powervr.ini
	if err == nil {
		_, err = os.Stat("/dev/cur_gl")
		if err != nil {
			return false, errors.New("cur_gl not found")
		}
	}
	return true, nil
}
