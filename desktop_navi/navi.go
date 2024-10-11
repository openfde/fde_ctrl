package navi

import (
	"bytes"
	"fde_ctrl/logger"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func hasWinOnDesktop(originalDesktop string) bool {
	cmd := exec.Command("wmctrl", "-l")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false
	}

	windows := strings.Split(out.String(), "\n")
	for _, window := range windows {
		fields := strings.Fields(window)
		if len(fields) > 2 && fields[1] == originalDesktop {
			return true
		}
	}
	return false
}

func moveWindowsBetweenDesktops(originalDesktop, targetDesktop string) {
	cmd := exec.Command("wmctrl", "-l")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return
	}

	windows := strings.Split(out.String(), "\n")
	for _, window := range windows {
		fields := strings.Fields(window)
		if len(fields) > 2 && fields[1] == originalDesktop {
			windowID := fields[0]
			exec.Command("wmctrl", "-ir", windowID, "-t", targetDesktop).Run()
		}
	}
}

func StartFdeNavi() {
	_, err := os.Stat("/usr/bin/fde_navi")
	if err != nil {
		return
	}

	cmd := exec.Command("pgrep", "fde_navi")
	err = cmd.Run()
	if err != nil {
		exec.Command("fde_navi").Start()
	}

	cmd = exec.Command("wmctrl", "-d")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return
	}

	numOfDesktop := len(strings.Split(out.String(), "\n")) - 1
	cmd = exec.Command("wmctrl", "-d")
	out.Reset()
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return
	}

	var currentDesktop string
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.Contains(line, "*") {
			fields := strings.Fields(line)
			currentDesktop = fields[0]
			break
		}
	}
	logger.Info("current_dekstop", currentDesktop)

	currentDesktopInt, _ := strconv.Atoi(currentDesktop)
	next := currentDesktopInt + 1
	logger.Info("next_dekstop", next)

	var res int
	if next == numOfDesktop {
		res = 1
	} else {
		if hasWinOnDesktop(strconv.Itoa(next)) {
			res = 1
		} else {
			res = 0
		}
	}

	if res == 1 {
		newNumOfDesktop := numOfDesktop + 1
		exec.Command("wmctrl", "-n", strconv.Itoa(newNumOfDesktop)).Run()
		for i := numOfDesktop; i > currentDesktopInt+1; i-- {
			moveWindowsBetweenDesktops(strconv.Itoa(i-1), strconv.Itoa(i))
		}
	}

	exec.Command("wmctrl", "-s", strconv.Itoa(next)).Run()
}
