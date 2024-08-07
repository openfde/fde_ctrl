package gpu

import (
	"errors"
	"os"
)

type x100 struct {
}

func (gpu x100) IsReady() (bool, error) {
	_, err := os.Stat("/etc/powervr.ini") //must has  cur_gl with powervr.ini
	if err == nil {
		_, err = os.Stat("/dev/cur_gl")
		if err != nil {
			return false, errors.New("cur_gl not found")
		}
	}
	return true, nil
}
