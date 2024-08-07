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
				switch lists[1] {
				case "phytium_display_pci":
					{ //x100
						gpu = new(x100)
					}
				case "jmgpu":
					{ //jm9100
						gpu = new(JM9100)
					}
				}
				return gpu.IsReady()
			}
		}
	}
	return true, nil
}
