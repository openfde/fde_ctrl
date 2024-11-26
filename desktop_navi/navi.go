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
	fdeDesktopInt := currentDesktopInt
	if currentDesktopInt == 0 {
		fdeDesktopInt++
	}
	haveAIdleDesktop := false
	var hasWinOnDesktopFlg bool
	if numOfDesktop == 1 { //only 1 desktop
		hasWinOnDesktopFlg = true
	} else { //condition: two or more desktops
		hasWinOnDesktopFlg = hasWinOnDesktop(strconv.Itoa(fdeDesktopInt))
	}
	if hasWinOnDesktopFlg && numOfDesktop >= 4 { //find a idle desktop from 1 to the last one
		for i := 1; i < numOfDesktop-1; i++ {
			if !hasWinOnDesktop(strconv.Itoa(i)) {
				haveAIdleDesktop = true
				fdeDesktopInt = i
				break
			}
		}
		if haveAIdleDesktop {
			if currentDesktopInt != fdeDesktopInt {
				exec.Command("wmctrl", "-s", strconv.Itoa(fdeDesktopInt)).Run()
				return
			}
		}
	}

	if hasWinOnDesktopFlg {
		logger.Info(" win on desktop", nil)
		newNumOfDesktop := numOfDesktop + 1
		exec.Command("wmctrl", "-n", strconv.Itoa(newNumOfDesktop)).Run()
		for i := newNumOfDesktop - 1; i > fdeDesktopInt; i-- {
			moveWindowsBetweenDesktops(strconv.Itoa(i-1), strconv.Itoa(i))
		}
	}
	if currentDesktopInt != fdeDesktopInt {
		exec.Command("wmctrl", "-s", strconv.Itoa(fdeDesktopInt)).Run()
	}
}
