package gpu

import "os/exec"

type JM9100 struct {
}

func (gpu JM9100) IsReady() (bool, error) {
	//start fde-renderer
	exec.Command("fde-renderer").Run()
	return true, nil
}
