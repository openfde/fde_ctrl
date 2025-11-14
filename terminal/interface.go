package terminal

import (
	"bufio"
	"fde_ctrl/logger"
	"os"
	"strings"
)

type Terminal interface {
	GetTerminal() (string, string)
}

func parseOSRelese() string {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		logger.Error("open_os_release", nil, err)
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			return strings.TrimPrefix(line, "ID=")
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("scan_os_release", nil, err)
		return ""
	}
	return ""
}

const (
	IDKylin  = "kylin"
	IDUos    = "uos"
	IDUbuntu = "ubuntu"
)

func GetTerminalProgram() (string, string) {
	var ter Terminal
	osName := parseOSRelese()
	switch osName {
	case IDKylin:
		{
			ter = KylinTerminalImpl{}
		}
	case IDUos:
		{
			ter = UosTerminalImpl{}
		}
	case IDUbuntu:
		{
			ter = UbuntuTerminalImpl{}
		}
	default:
		{
			return "", ""
		}
	}
	return ter.GetTerminal()
}
