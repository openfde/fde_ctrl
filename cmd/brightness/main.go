/*
* Copyright （C)  2023 OpenFDE , All rights reserved.
 */

package main

import (
	"bufio"
	"context"
	"fde_ctrl/logger"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var _version_ = "v0.1"
var _tag_ = "v0.1"
var _date_ = "20230101"

func main() {

	var version, help bool
	var mode, bus, brightness string
	flag.BoolVar(&version, "version", false, "print version")
	flag.BoolVar(&help, "help", false, "print help")
	flag.StringVar(&mode, "mode", "detect", "decide to detect|set|get")
	flag.StringVar(&bus, "bus", "11", "i2c bus")
	flag.StringVar(&brightness, "brightness", "80", "adjust breightness")
	flag.Parse()
	if help {
		fmt.Println("fde_brightness:")
		fmt.Println("\t-v: print versions and tags")
		fmt.Println("\t-h: print help")
		fmt.Println("\t-m: input the running mode[detect|get|set]")
		return
	}

	if version {
		fmt.Printf("Version: %s, tag: %s , date: %s \n", _version_, _tag_, _date_)
		return
	}
	var err error
	mainCtx, _ := context.WithCancel(context.Background())
	switch {
	case mode == "detect":
		{
			err = detect(mainCtx)
		}
	case mode == "get":
		{
			err = get(mainCtx, bus)
		}
	case mode == "set":
		{
			err = set(mainCtx, bus, brightness)
		}
	}
	if err != nil {
		os.Exit(1)
	}
}

func detect(mainCtx context.Context) error {
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
	fmt.Println(se)
	return nil
}

func set(mainCtx context.Context, bus, brightness string) error {
	var cmd *exec.Cmd
	cmd = exec.CommandContext(mainCtx, "ddcutil", "--bus", bus, "setvcp", "10", brightness)

	if err := cmd.Start(); err != nil {
		logger.Error("start_ddcutil_set", nil, err)
		return err
	}

	if err := cmd.Wait(); err != nil {
		logger.Error("wait_ddcutil_set", nil, err)
		return err
	}
	return nil
}

func get(mainCtx context.Context, bus string) error {
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
