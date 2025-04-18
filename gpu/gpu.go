package gpu

import (
	"bufio"
	"fde_ctrl/windows_manager"
	"os"
	"strings"
)

type Gpu interface {
	IsReady(windows_manager.FDEMode) (bool, error)
}

func IsReady(mode windows_manager.FDEMode) (bool, error) {
	node := "/sys/class/drm/card0/device/uevent"

	_, err := os.Stat(node)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
	}
	file, err := os.Open(node)
	if err != nil {
		return false, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.Contains(line, "DRIVER") {
			lists := strings.Split(line, "=")
			if len(lists) == 2 {
				var gpu Gpu
				gpu = nil
				if strings.Contains(lists[1], "phytium_display_pci") {
					gpu = new(x100)
				} else if strings.Contains(lists[1], "jmgpu") {
					gpu = new(JM9100)
				}
				if gpu != nil {
					return gpu.IsReady(mode)
				}
			}
		}
	}
	return true, nil
}
