package gpu

import (
	"strings"
	"os"
	"errors"
	"bufio"
)


func  IsReady() (bool,error){

	node := "/sys/class/drm/card0/device/uevent"

	_, err := os.Stat(node)
	if err != nil {
		if os.IsNotExist(err) {
			return  false,nil
		}
	}
	file, err := os.Open(node)
	if err != nil {
		return false,err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.Contains(line,"DRIVER") {
			lists := strings.Split(line,"=")
			if len(lists) == 2 {
				if strings.Contains(lists[1],"phytium_display_pci") {
					_,err = os.Stat("/etc/powervr.ini") //only kylin os has powervr.ini
					if err == nil {
						_,err = os.Stat("/dev/cur_gl")
						if err != nil {
							return false,errors.New("cur_gl not found")
						}
					}
				}
			}
			continue
		}
	}
	return true,nil
}

