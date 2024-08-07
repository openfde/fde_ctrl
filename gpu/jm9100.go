package gpu

import (
	"os/exec"
	"time"
)

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
	// Run fde_fs -s command
	if err := exec.Command("fde_fs", "-s").Run(); err != nil {
		return false, err
	}

	// Start fde-renderer process
	if err := exec.Command("fde-renderer").Start(); err != nil {
		// Check again if fde-renderer process is already running using a loop
		for i := 0; i < 10; i++ {
			cmd := exec.Command("pgrep", "fde-renderer")
			if err := cmd.Run(); err == nil {
				// fde-renderer process is already running
				return true, nil
			}
			time.Sleep(time.Second) // Wait for 1 second before checking again
		}
		return false, nil
	}

	return true, nil
}
