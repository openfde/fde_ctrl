package emugl

import (
	"bufio"
	"errors"
	"fde_ctrl/logger"
	"os"
	"os/exec"
	"strings"
	"time"
)

func IsEmugl() bool {
	file, err := os.Open("/var/lib/waydroid/waydroid_base.prop")
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ro.hardware.gralloc=") {
			value := strings.TrimPrefix(line, "ro.hardware.gralloc=")
			return value == "ranchu"
		}
	}
	return false
}

func StartFDERender() error {
	//start fde-renderer
	// Check if fde-renderer process is already running
	cmd := exec.Command("pgrep", "fde-renderer")
	if err := cmd.Run(); err == nil {
		// fde-renderer process is already running
		// Kill fde-renderer process
		if err := exec.Command("pkill", "fde-renderer").Run(); err != nil {
			logger.Error("kill_fde_renderer_exist", nil, err)
			return err
		}
	}
	// Run fde_fs -s command to set softmode of the secure mode on kylin os
	if err := exec.Command("fde_fs", "-s").Run(); err != nil {
		logger.Error("set_secure_softmode", nil, err)
		return err
	}

	// Start fde-renderer process
	if err := exec.Command("fde-renderer").Start(); err != nil {
		// Check again if fde-renderer process is already running using a loop
		for i := 0; i < 5; i++ {
			cmd := exec.Command("pgrep", "fde-renderer")
			if err := cmd.Run(); err == nil {
				// fde-renderer process is already running
				return nil
			}
			time.Sleep(time.Second) // Wait for 1 second before checking again
		}
		err = errors.New("fde-renderer process is not running")
		logger.Error("check_fde_renderer_exists", nil, err)
		return err
	}
	if _, err := os.Stat("/var/lib/fde/sockets/qemu_pipe"); os.IsNotExist(err) {
		logger.Error("check_fde_renderer_qemupipe_exists", nil, errors.New("qemu_pipe not exist"))
		return err
	}
	return nil
}
