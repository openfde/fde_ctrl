/*
* Copyright （C)  2023 OpenFDE , All rights reserved.
 */

package main

import (
	"bufio"
	"context"
	"fde_ctrl/logger"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	mainCtx, _ := context.WithCancel(context.Background())
	cmd := exec.CommandContext(mainCtx, "ddcutil", "detect")

	output, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("brightness_stdout", nil, err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		logger.Error("start_ddcutil", nil, err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(output)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "I2C bus") {
			lines := strings.Split(line, "-")
			fmt.Println(lines[len(lines)-1])
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("scan_ddcutil", nil, err)
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		logger.Error("wait_ddcutil", nil, err)
		os.Exit(1)
	}
}
