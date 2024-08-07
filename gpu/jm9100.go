package gpu

import "os/exec"

type JM9100 struct {
}

func (gpu JM9100) IsReady() (bool, error) {
	//start fde-renderer
	// Check if fde-renderer process is already running
	cmd := exec.Command("pgrep", "fde-renderer")
	if err := cmd.Run(); err == nil {
		// fde-renderer process is already running
		return true, nil
	}

	// Start fde-renderer process
	if err := exec.Command("fde-renderer").Start(); err != nil {
		return false, err
	}

	return true, nil
}
