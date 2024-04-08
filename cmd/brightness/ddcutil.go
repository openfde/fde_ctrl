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

type DDcutil struct {
}

func (impl DDcutil) Detect(mainCtx context.Context) error {
	var cmd *exec.Cmd
	cmd = exec.CommandContext(mainCtx, "ddcutil", "detect")
	output, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("brightness_stdout", nil, err)
		return err
	}

	if err := cmd.Start(); err != nil {
		logger.Error("start_ddcutil", nil, err)
		return err
	}

	scanner := bufio.NewScanner(output)

	var se []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "I2C bus") {
			lines := strings.Split(line, "-")
			se = append(se, lines[len(lines)-1])
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("scan_ddcutil", nil, err)
		return err
	}

	if err := cmd.Wait(); err != nil {
		logger.Error("wait_ddcutil", nil, err)
		return err
	}

	str := strings.Join(se, ",")
	fmt.Println(str)
	return nil
}

func (impl DDcutil) Set(mainCtx context.Context, bus, brightness string) error {
	var cmd *exec.Cmd
	cmd = exec.CommandContext(mainCtx, "ddcutil", "--bus", bus, "setvcp", "10", brightness)
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(string(output), "No monitor detected") {
			os.Exit(BrightnessErrorBusInvalid)
		}
	}
	return nil
}

func (impl *DDcutil) Get(mainCtx context.Context, bus string) error {
	var cmd *exec.Cmd
	cmd = exec.CommandContext(mainCtx, "ddcutil", "--bus", bus, "getvcp", "10")
	output, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		logger.Error("start_ddcutil_set", nil, err)
		return err
	}

	scanner := bufio.NewScanner(output)
	var brightness, maxBrightness string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "No monitor detected") {
			os.Exit(BrightnessErrorBusInvalid)
		}
		lines := strings.Fields(line)
		if len(lines) > 10 {
			brightness = lines[8]
			brightness = strings.TrimSuffix(brightness, ",")
			maxBrightness = lines[len(lines)-1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("scanner_ddcutil_get", nil, err)
		return err
	}

	if err := cmd.Wait(); err != nil {
		logger.Error("wait_ddcutil_get", nil, err)
		return err
	}
	fmt.Println(brightness, maxBrightness)
	return nil
}
