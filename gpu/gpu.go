package gpu

import (
	"bufio"
	"os"
	"strings"
)

type Gpu interface {
	IsReady() (bool, error)
}

func IsReady() (bool, error) {
	node := "/sys/class/drm/card0/device/uevent"

	_, err := os.Stat(node)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
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
					return gpu.IsReady()
				}
			}
		}
	}
	return true, nil
}
